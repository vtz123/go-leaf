package cache

type SegmentViewBuffer struct {
	maxId int64
	nowId int64
}

type Buffer struct {
	right int64
	step  int64
}
