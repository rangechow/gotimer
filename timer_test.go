package gotimer

import (
	"container/heap"
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func GenTimer(index int) *event {

	e := &event{
		name:   "e" + strconv.Itoa(index),
		expire: time.Now(),
	}

	return e
}

func GenTimerQueue(size int) *eventQueue {

	tq := make(eventQueue, 0)
	heap.Init(&tq)

	for i := 1; i <= size; i++ {
		e := GenTimer(i)
		heap.Push(&tq, e)
		//time.Sleep(5)
	}

	return &tq
}

func TestTimerQueueAdd(t *testing.T) {

	var size int = 10000

	tq := GenTimerQueue(size)

	if len(*tq) != size {
		t.Errorf("优先队列的大小%d不等于%d", len(*tq), size)
		return
	}

	e := GenTimer(size + 1)
	heap.Push(tq, e)

	if len(*tq) != size+1 {
		t.Errorf("优先队列的大小%d不等于%d", len(*tq), size+1)
		return
	}

	for i, e := range *tq {
		if i == 0 {
			if e.name != "e1" {
				t.Error("e1不是第一个超时的定时器")
				return
			}
		}
		//t.Logf("event %s expire %v index %d", e.name, e.expire, e.index)
	}
	t.Log("pass")
}

func TestTimerQueueDel(t *testing.T) {

	var size int = 10000

	tq := GenTimerQueue(size)

	if len(*tq) != size {
		t.Errorf("优先队列的大小%d不等于%d", len(*tq), size)
		return
	}

	heap.Remove(tq, 0)

	for i, e := range *tq {
		if i == 0 {
			if e.name == "e1" {
				t.Error("第一个超时的定时器e1没有删除成功")
				return
			}
		}
		//t.Logf("event %s expire %v index %d", e.name, e.expire, e.index)
	}

	if len(*tq) != size-1 {
		t.Errorf("优先队列的大小%d不等于%d", len(*tq), size-1)
		return
	}

	rand.Seed(42)
	for len(*tq) > 0 {
		heap.Remove(tq, rand.Intn(len(*tq)))
	}

	if len(*tq) != 0 {
		t.Errorf("优先队列的大小%d不等于%d", len(*tq), 0)
		return
	}
	t.Log("pass")
}

func TestTimerQueuePop(t *testing.T) {

	var size int = 10000

	tq := GenTimerQueue(size)

	if len(*tq) != size {
		t.Errorf("优先队列的大小%d不等于%d", len(*tq), size)
		return
	}

	e := heap.Pop(tq).(*event)

	if e.name != "e1" {
		t.Error("第一个超时的定时器e1没有弹出成功")
		return
	}

	if len(*tq) != size-1 {
		t.Errorf("优先队列的大小%d不等于%d", len(*tq), size-1)
		return
	}

	for len(*tq) > 0 {
		heap.Pop(tq)
	}

	if len(*tq) != 0 {
		t.Errorf("优先队列的大小%d不等于%d", len(*tq), 0)
		return
	}
	t.Log("pass")
}

type exampleSvr int

func (s *exampleSvr) CheckTimer1(ctx context.Context, uid uint64) error {
	return nil
}

func (s *exampleSvr) CheckTimer2(ctx context.Context, uid uint32) error {
	return nil
}

func TestTimerRegister(t *testing.T) {

	s := new(exampleSvr)

	timer := new(timerService)

	timer.Register(s)

	timer.AsyncRun()

	_, err := timer.AddTimer(time.Millisecond*1000, true, "CheckTimer1", 10001)

	if err != nil {
		t.Errorf("添加定时器失败 %v", err)
		return
	}

	_, err = timer.AddTimer(time.Millisecond*1000, true, "CheckTimer3", 10001)

	if err == nil {
		t.Errorf("添加定时器错误 %v", err)
		return
	}

	timer.StopTimerService()

	t.Log("pass")

	return
}

func TestTimerReAdd(t *testing.T) {

	s := new(exampleSvr)

	var timer timerService

	timer.Register(s)

	timer.AsyncRun()

	_, err := timer.AddTimer(time.Millisecond*1000, true, "CheckTimer1", 10001)

	if err != nil {
		t.Errorf("添加定时器失败 %v", err)
		return
	}

	_, err = timer.AddTimer(time.Millisecond*1000, true, "CheckTimer1", 10001)

	if err != nil {
		t.Errorf("添加定时器错误 %v", err)
		return
	}

	timer.StopTimerService()
	t.Log("pass")
	return
}
