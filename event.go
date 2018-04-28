package gotimer

import "time"

// user add a timer event to the timer service
// the event will run ontime
type event struct {
	// use for find
	seq int
	// index for PriorityQueue
	index int
	// handler name , to find handler
	name string
	// timer args, local copy
	arg interface{}
	// interval
	interval time.Duration
	// isRepeat
	isRepeat bool
	// expire time
	expire time.Time
}

type eventQueue []*event

func (q eventQueue) Len() int { return len(q) }

// if i < j return true , priority is lowest
func (q eventQueue) Less(i, j int) bool {
	return q[i].expire.Before(q[j].expire)
}

func (q eventQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *eventQueue) Push(x interface{}) {
	n := len(*q)
	e := x.(*event)
	e.index = n
	*q = append(*q, e)
}

func (q *eventQueue) Pop() interface{} {
	old := *q
	n := len(old)
	e := old[n-1]
	e.index = -1
	*q = old[0 : n-1]
	return e
}
