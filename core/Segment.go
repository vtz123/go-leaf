package core

import (
	"errors"
	"log"
	"sync"
	"time"
)

const (
	SegMent_Duration = 15 * 60 * 1000 // 15min
	MAX_STEP         = 1000000        // 最大步长不超过100,0000
	MIN_STEP         = 1000
)

// 号段：[left,right)
type Segment struct {
	offset int64 // 消费偏移
	left   int64 // 左区间
	right  int64 // 右区间
}

// 关联到bizTag的号码池
type BizAlloc struct {
	mutex        sync.Mutex
	bizTag       string      // 业务标识
	segments     []*Segment  // 双Buffer, 最少0个, 最多2个号段在内存
	isAllocating bool        // 是否正在分配中(远程获取)
	waiting      []chan byte // 与wakeup()配合使用 因号码池空而挂起等待的客户端，
	// 当DB跌宕，双buffer用完， 分配器无法向DB中获取号段，则提醒客户端，拒绝客户端申请id
	// 当双buffer已经申请到了，唤醒客户端，向其提供分配id服务
	lastSegmentTime int64 //更新双buffer访问时间 根据号段消耗周期，自动调整step长度
	step            int64 //记录上次step长度
}

// Segment号段分配器
type AllocSegment struct {
	mutex  sync.Mutex
	bizMap map[string]*BizAlloc
}

// 全局分配器
var GAllocSegment *AllocSegment

// 分配器初始化
func InitAllocSegment() (err error) {
	GAllocSegment = &AllocSegment{
		bizMap: map[string]*BizAlloc{},
	}
	return
}

// 双Buffer中剩余号段数量
func (bizAlloc *BizAlloc) leftCount() (count int64) {
	for i := 0; i < len(bizAlloc.segments); i++ {
		count += bizAlloc.segments[i].right - bizAlloc.segments[i].left - bizAlloc.segments[i].offset
	}
	return count
}

func (bizAlloc *BizAlloc) leftCountWithMutex() (count int64) {
	bizAlloc.mutex.Lock()
	defer bizAlloc.mutex.Unlock()
	return bizAlloc.leftCount()
}

//动态调整step长度(step初始值为1000)，并更新buffer   bizAlloc在调用函数中已被上锁
func (bizAlloc *BizAlloc) newSegment() (seg *Segment, err error) {
	var (
		maxId    int64
		dataStep int64
	)
	timeNow := getTimeNow()
	//先调整step长度
	if bizAlloc.lastSegmentTime == 0 || bizAlloc.step == 0 {
		//直接从数据库取数
		//do nothing    step不调整
	} else { //根据消耗周期，动态调整step

		wasteTime := timeNow - bizAlloc.lastSegmentTime
		if wasteTime < SegMent_Duration {
			if bizAlloc.step*2 > MAX_STEP {
				//do nothing
			} else {
				bizAlloc.step = bizAlloc.step * 2
			}
		} else if wasteTime < SegMent_Duration*2 {
			//do nothing
		} else {
			if (bizAlloc.step / 2) > MIN_STEP {
				bizAlloc.step = bizAlloc.step / 2
			} else {
				//do nothing
			}

		}
	}

	bizAlloc.lastSegmentTime = timeNow //更新访问时间
	//log.Print("bizAlloc.step:")

	// 再更新maxId,step   并取数
	if maxId, dataStep, err = GMysql.NextId(bizAlloc.bizTag, bizAlloc.step); err != nil {
		//log.Print("111111")
		return
	}

	bizAlloc.step = dataStep //记录step
	log.Printf("bizAlloc.step已更新: %v", bizAlloc.step)
	if dataStep != bizAlloc.step {
		log.Println(dataStep)
		log.Println(bizAlloc.step)

		log.Panic("step调整错误")
	}
	seg = &Segment{}
	seg.left = maxId - dataStep
	seg.right = maxId
	return
}

func (bizAlloc *BizAlloc) wakeup() {
	var (
		waitChan chan byte
	)
	for _, waitChan = range bizAlloc.waiting {
		close(waitChan)
	}
	bizAlloc.waiting = bizAlloc.waiting[:0]
}

// 分配号码段, 直到足够2个segment, 否则始终不会退出
func (bizAlloc *BizAlloc) fillSegments() {
	var (
		failTimes int64 // 连续分配失败次数
		seg       *Segment
		err       error
	)

	for { //考虑到DB跌宕时，请求不成功
		bizAlloc.mutex.Lock()
		if len(bizAlloc.segments) <= 1 { // 只剩余<=1段, 那么继续获取新号段
			bizAlloc.mutex.Unlock()
			// 请求mysql获取号段
			if seg, err = bizAlloc.newSegment(); err != nil {
				failTimes++
				if failTimes > 3 { // 连续失败超过3次则停止分配
					bizAlloc.mutex.Lock()
					bizAlloc.wakeup() // 唤醒等待者, 让它们立马失败
					goto LEAVE
				}
			} else {
				failTimes = 0 // 分配成功则失败次数重置为0
				// 新号段补充进去
				bizAlloc.mutex.Lock()
				bizAlloc.segments = append(bizAlloc.segments, seg)
				bizAlloc.wakeup()               // 尝试唤醒等待资源的调用
				if len(bizAlloc.segments) > 1 { // 已生成2个号段, 停止继续分配
					log.Printf("已分配 now len segments %v", len(bizAlloc.segments))
					goto LEAVE
				} else {
					bizAlloc.mutex.Unlock()
				}
			}
		} else {
			// never reach
			break
		}
	}

LEAVE:
	bizAlloc.isAllocating = false
	bizAlloc.mutex.Unlock()
}

func (bizAlloc *BizAlloc) popNextId() (nextId int64) {
	nextId = bizAlloc.segments[0].left + bizAlloc.segments[0].offset
	bizAlloc.segments[0].offset++
	if nextId+1 >= bizAlloc.segments[0].right { //使用另一个buffer
		bizAlloc.segments = append(bizAlloc.segments[:0], bizAlloc.segments[1:]...) // 弹出第一个seg, 后续seg向前移动
	}
	return
}

func (bizAlloc *BizAlloc) nextId() (nextId int64, err error) {
	var (
		waitChan  chan byte
		waitTimer *time.Timer
		hasId     = false
	)

	bizAlloc.mutex.Lock()

	defer bizAlloc.mutex.Unlock()

	// 1, 有剩余号码, 立即分配返回
	if bizAlloc.leftCount() != 0 {
		nextId = bizAlloc.popNextId() //会根据buffer号段数量，随时启用第二个buffer
		hasId = true
	}

	// 2, 段<=1个并且bizAlloc没有在分配 , 启动补偿线程
	log.Printf("now len segments(buffer数量):  %v", len(bizAlloc.segments))
	if len(bizAlloc.segments) <= 1 && !bizAlloc.isAllocating {
		bizAlloc.isAllocating = true
		go bizAlloc.fillSegments()
	}
	//log.Printf("next len segments:   %v",len(bizAlloc.segments))
	// 分配到号码, 立即退出
	if hasId {
		return
	}

	// 3, 没有剩余号码, 此时补偿线程一定正在运行, 等待其至多一段时间
	waitChan = make(chan byte, 1)
	bizAlloc.waiting = append(bizAlloc.waiting, waitChan) // 排队等待唤醒

	// 释放锁, 等待补偿线程唤醒
	bizAlloc.mutex.Unlock()

	waitTimer = time.NewTimer(2 * time.Second) // 最多等待2秒
	select {
	case <-waitChan:
	case <-waitTimer.C:
	}

	// 4, 再次上锁尝试获取号码
	bizAlloc.mutex.Lock()
	if bizAlloc.leftCount() != 0 {
		nextId = bizAlloc.popNextId()
	} else {
		err = errors.New("no available id")
	}
	return
}

func (alloc *AllocSegment) NextId(bizTag string) (nextId int64, err error) {
	var (
		bizAlloc *BizAlloc
		exist    bool
	)

	alloc.mutex.Lock()
	if bizAlloc, exist = alloc.bizMap[bizTag]; !exist {
		bizAlloc = &BizAlloc{
			bizTag:          bizTag,
			segments:        make([]*Segment, 0),
			isAllocating:    false,
			waiting:         make([]chan byte, 0),
			step:            0,
			lastSegmentTime: 0,
		}
		alloc.bizMap[bizTag] = bizAlloc
	}
	alloc.mutex.Unlock()

	nextId, err = bizAlloc.nextId()
	return
}

func (alloc *AllocSegment) LeftCount(bizTag string) (leftCount int64) {
	var (
		bizAlloc *BizAlloc
	)

	alloc.mutex.Lock()
	bizAlloc, _ = alloc.bizMap[bizTag]
	alloc.mutex.Unlock()

	if bizAlloc != nil {
		leftCount = bizAlloc.leftCountWithMutex()
	}
	return
}
