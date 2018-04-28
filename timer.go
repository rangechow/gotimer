package gotimer

import (
	"container/heap"
	"context"
	"fmt"
	"reflect"
	"time"
)

// timerHander is called when the event expire
//
// common form:
// func (receiver) (context, interface{}) (error)
type timerHandler struct {
	receiver reflect.Value
	method   reflect.Method
	arg      reflect.Type
}

// timerService is timer manager, It can be used independently or as a module of the service
type timerService struct {
	isInit   bool
	accuracy time.Duration //tick interval accuracy

	handlers map[string]*timerHandler // name to handler
	events   eventQueue
	trash    []int // timer events waiting to be deleted
	sequence int   // timer sequence

	add    chan event    // add timer chan
	addret chan int      // after add timer return seq chan
	del    chan int      // delete timer chan
	done   chan struct{} // stop timer chan
}

// Precompute the reflect type for context.
var typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()

const defaultAccuracy = 100 * time.Millisecond

// SetAccuracy can change the accuracy of the timer.
// default accuracy is 100 Millisecond.
// It is only valid before run timer.
func (t *timerService) SetAccuracy(accuracy time.Duration) error {

	if accuracy <= 0 {
		return fmt.Errorf("accuracy must bigger than zero")
	}

	t.accuracy = accuracy

	return nil
}

// Register add handler from type method to timer.
// The name of the handler must be unique
func (t *timerService) Register(svr interface{}) error {

	if !t.isInit {

		if t.accuracy == 0 {
			t.accuracy = defaultAccuracy
		}

		t.events = make(eventQueue, 0)
		heap.Init(&t.events)

		t.handlers = make(map[string]*timerHandler)
		t.trash = make([]int, 0)
		t.add = make(chan event)
		t.addret = make(chan int)
		t.del = make(chan int, 100)
		t.done = make(chan struct{})

		t.isInit = true
	}

	receiver := reflect.ValueOf(svr)
	methods := reflect.TypeOf(svr)

	for i := 0; i < methods.NumMethod(); i++ {

		method := methods.Method(i)
		mType := method.Type
		mName := method.Name

		if _, ok := t.handlers[mName]; ok {
			//fmt.Printf("unfound timer handler %s", handlerName)
			continue
		}

		// Method needs three ins: receiver, context.Context, args, .
		if mType.NumIn() != 3 {
			//fmt.Printf("method %v has wrong number of ins:%v\n", mName, mType.NumIn())
			continue
		}
		// Method needs one out.
		if mType.NumOut() != 1 {
			//fmt.Printf("method %s has wrong number of outs:%v\n", mName, mType.NumOut())
			continue
		}
		// First arg must be context.Context
		ctxType := mType.In(1)
		if !ctxType.Implements(typeOfContext) {
			//fmt.Printf("method %s, must use context.Context as the first parameter\n", mName)
			continue
		}
		// one args need not be a pointer.
		argsType := mType.In(2)

		t.handlers[mName] = &timerHandler{receiver, method, argsType}
		fmt.Printf("Register method %v|type %v\n", mName, mType)
	}
	return nil
}

// AddOneshotTimer will add a timer that trigger once at specified time.
// It wrap AddTimer.
func (t *timerService) AddOneshotTimer(datatime time.Time, handlerName string, args interface{}) (int, error) {

	now := time.Now()

	if now.After(datatime) {
		return 0, fmt.Errorf("timer expire time %d before now %d", datatime.Unix(), now.Unix())
	}

	interval := datatime.Sub(now)

	return t.AddTimer(interval, false, handlerName, args)
}

// AddTimer can add a timer that trigger from now until interval.
// If you want the timer trigger repeate, you could set isRepeate true.
// It is a blocking function that waiting timer's sequence
func (t *timerService) AddTimer(interval time.Duration, isRepeate bool, handlerName string, args interface{}) (int, error) {

	if interval < 0 {
		return 0, fmt.Errorf("Add timer failed, interval %d not allow", interval)
	}

	if _, ok := t.handlers[handlerName]; !ok {
		return 0, fmt.Errorf("unfound timer handler %s", handlerName)
	}

	now := time.Now()

	e := event{
		name:     handlerName,
		arg:      args,
		interval: interval,
		isRepeat: isRepeate,
		expire:   now.Add(interval),
	}

	// send event to timer goroutine and wait return
	for {
		select {
		case t.add <- e:
			return <-t.addret, nil
		default:
			continue
		}
	}

	return 0, fmt.Errorf("add time failed")
}

// doAddTimer receive event into the priority queue, and return the sequence number.
// The add chan and addret chan have only one buffer
func (t *timerService) doAddTimer(e *event) {
	t.sequence++
	e.seq = t.sequence
	heap.Push(&t.events, e)
	t.addret <- e.seq
}

// DelTimer send a sequence of deleting timer to timer goroutine.
// The seq will push to a slice waiting for delete.
func (t *timerService) DelTimer(seq int) {
	t.del <- seq
}

// doDelTimer is the actual performer of the delete operation.
// It will call everytime before checking timeout.
func (t *timerService) doDelTimer() {
	for _, seq := range t.trash {
		for _, e := range t.events {
			if e.seq == seq {
				heap.Remove(&t.events, e.index)
				break
			}

		}
	}
	t.trash = t.trash[:0]
}

// runTimeout triggers a timeout timer and calls the corresponding hangler.
func (t *timerService) runTimeout() {

	now := time.Now()

	t.doDelTimer()

	for {

		if len(t.events) == 0 || t.events[0].expire.After(now) {
			break
		}

		e := heap.Pop(&t.events).(*event)

		handler := t.handlers[e.name]

		if handler == nil {
			continue
		}

		go func() {

			fc := handler.method.Func
			// Invoke the method, providing a new value for the reply.
			ret := fc.Call([]reflect.Value{handler.receiver, reflect.ValueOf(context.Background()), reflect.ValueOf(e.arg)})
			// The return value for the method is an error.
			errInter := ret[0].Interface()
			if errInter != nil {
				fmt.Errorf("run timer %s err %v", e.name, errInter.(error))
			}

		}()

		if e.isRepeat {
			e.expire = now.Add(e.interval)
			heap.Push(&t.events, e)
		}
	}

}

// Run function synchronous execution timer service.
func (t *timerService) Run() {
	tick := time.Tick(t.accuracy)

	for {
		select {
		case <-tick:
			t.runTimeout()
		case e := <-t.add:
			t.doAddTimer(&e)
		case seq := <-t.del:
			t.trash = append(t.trash, seq)
		case <-t.done:
			return
		}
	}
}

// AsyncRun function asynchronous execution timer service.
func (t *timerService) AsyncRun() {
	go func() {
		t.Run()
	}()
}

// You can stop the timer service.
func (t *timerService) StopTimerService() {
	close(t.done)
}
