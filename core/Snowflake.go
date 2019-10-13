package core

import (
	"errors"
	_ "net"
	"sync"
	"time"
)

const (
	TimeStart       = 1567296000000                    //2019-10-01 00:00:00 +0000 UTC
	BitLenTime      = 41                               // time最长位数
	BitLenSequence  = 12                               // sequence number最长位数
	BitLenMachineID = 63 - BitLenTime - BitLenSequence // 机器id最长位数 10位
)

// Snowflake号段分配器
type AllocSnowflake struct {
	mutex      sync.Mutex
	MachineMap map[int32]*Snowflake
}

// 全局分配器
var GAllocSnowflake *AllocSnowflake

type Snowflake struct {
	mutex         *sync.Mutex
	lastTimeStamp int64 //上次访问的时间
	sequence      int32
	machineID     int32
}

//分配器初始化
func InitAllocSnowflake() error {
	GAllocSnowflake = &AllocSnowflake{
		MachineMap: map[int32]*Snowflake{},
	}
	return nil
}

func (alloc *AllocSnowflake) NewSnowflake(machineid int32) error {
	//检查机器id合理性
	alloc.mutex.Lock()
	defer alloc.mutex.Unlock()

	if machineid >= getMachineID() {
		err := errors.New("machineid too big")
		return err
	}
	if _, exist := alloc.MachineMap[machineid]; exist {
		err := errors.New("machineid repeated")
		return err
	}

	snowflake := new(Snowflake)
	snowflake.lastTimeStamp = getTimeNow()
	snowflake.mutex = new(sync.Mutex)
	snowflake.machineID = machineid
	snowflake.sequence = 0

	alloc.MachineMap[machineid] = snowflake

	return nil

}

func (alloc *AllocSnowflake) BoolmachineMap(machineid int32) bool {
	alloc.mutex.Lock()
	defer alloc.mutex.Unlock()
	if _, exist := alloc.MachineMap[machineid]; exist {
		//err := errors.New("machineid repeated")
		return true
	}
	return false
}

func (s *Snowflake) NextId() (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	//比较当前时间 与 起始时间、上次访问时间
	currentTime := getTimeNow()
	if currentTime <= TimeStart {
		err := errors.New("machine 时间故障")
		return 0, err
	}
	if currentTime < s.lastTimeStamp {
		err := errors.New("Clock moved backwards, Refuse gen id")
		return 0, err
	}
	if currentTime == s.lastTimeStamp {
		s.sequence++
		if s.sequence > getMaxSequenceID() {
			time.Sleep(time.Nanosecond) //停1ns
			currentTime = getTimeNow()
		}
	} else {
		s.sequence++
	}

	s.lastTimeStamp = currentTime

	elapsedTime := currentTime - TimeStart

	ts := int64(elapsedTime<<22) | int64(s.machineID<<12) | int64(s.sequence)
	return ts, nil
}

//获取当前时间UTC
func getTimeNow() int64 {
	trow := time.Now().UnixNano() / int64(time.Millisecond)
	return trow
}

//将日期转化为UTX时间（ms）
func dataToUTC(y, m, d, h, min int) int64 {
	timeStart := time.Date(y, time.Month(m), d, h, min, 0, 0, time.UTC).UnixNano() / int64(time.Millisecond)
	return timeStart
}

//Machine最大数
func getMachineID() int32 {
	return (1<<BitLenMachineID - 1)
}

//Sequence最大数
func getMaxSequenceID() int32 {
	return (1<<BitLenSequence - 1)
}

//纳秒级别转换为毫秒级别
func tomsTime(t time.Time) int64 {
	return t.UTC().UnixNano() / int64(time.Millisecond)
}
