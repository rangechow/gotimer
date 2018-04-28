
# gotimer [![GoDoc](https://godoc.org/github.com/rangechow/gotimer?status.svg)](https://godoc.org/github.com/rangechow/gotimer)

gotimer is a Go library for timer as a service or a module of service.

## Features

* easy to use
* auto register the event handler
* use handler name to add timer
* can add many timer at the same time and supports various timers mode
* pass the parameters to timer when add
* sync run or async run

## Installation

Standard `go get`:

```
$ go get github.com/rangechow/gotimer
```

## Usage at a glance

1. design the common form timer handler as the method of type.

```go
package main

import (
    "context"
    "fmt"
)

type exampleSvr int

func (s *exampleSvr) CheckTimer(ctx context.Context, id uint64) error {
    fmt.Println("check timer uid = " + id)
    return nil
}

```

2. init timer service and sync run

```go
package main

import (
    "fmt"
    "time"

    "github/rangechow/gotimer"
)


func main() {
    
    s := new(exampleSvr)

    var timer gotimer.timerService
    timer.Register(s)

    _, err := timer.AddTimer(time.Millisecond*1000, true, "CheckTimer", 10001)

    if err != nil {
        fmt.Printf("add timer failed %v", err)
        return
    }

    timer.Run()
}


```

2. or async run as a module of service

```go
package main

import (
    "fmt"
    "time"

    "github/rangechow/gotimer"
)

func main() {
    
    s := new(exampleSvr)

    var timer gotimer.timerService
    timer.Register(s)
    timer.AsyncRun()

    _, err := timer.AddTimer(time.Millisecond*1000, true, "CheckTimer", 10001)

    if err != nil {
        fmt.Printf("add timer failed %v", err)
        return
    }

    // you service run forever
    // in this exampe, it will end
}


```

## Some details

when designing a stateful service, we often need a timer to trigger some active actions or events.
Timer process flow: register handler --> run timer service --> add timer --> trigger 

### register handler

Timer handler is part of service, so we used a registration model similar to GRPC.
The difference is that we do not have a pre-generated interface.
We used the mode of handler.
```go 
func (receiver) (context, interface{}) (error)
```
At the beginning, we call the registration api, passing our service.
Timer will auto register handler. 
Sometimes some extra functions are registered, but it doesn't matter, we won't call it.
When we register handler, we also need to specify the timer's accuracy and maximum timeout.

### run timer service

As part of the service, the timer will execute asynchronously on one goroutine.
Therefore, we do not need to worry about the race condition of the timer data.
In addition, you can use the timer service synchronously.
Timer process flow change to : register handler --> add timer --> run timer service --> trigger

### add timer

There are general two types of timers:
  1.  one shot at specified time
  2.  repeat interval time

We have api support for the above operation.
And Trigger a handler by specifying handler name.
When we add the timer, we need to specify the name of the handler, and we can pass the corresponding parameter to the handler.
The Add APIs will return sequence number, it would be used to delete the unexpired timer.

### trigger

We use priority queue management timer.
Each tick, we will delete the timer in the trash, and then checking the timer expired in the queue.
If the timer expires, the handler will execute in a goroutine.

## API Documentation

see the [Godoc](http://godoc.org/github.com/rangechow/gotimer).


