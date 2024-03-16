package test

import (
	"context"
	"github.com/jqqjj/go-utils"
	"log"
	"testing"
	"time"
)

func TestExchanger(t *testing.T) {
	e := utils.NewExchanger[any]()

	ctx, cancel := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	ch := make(chan any)
	ch2 := make(chan any)

	e.Publish("test", []byte("1"))
	e.Publish("test", 1)

	e.Subscribe(ctx, "test", ch)
	e.Subscribe(ctx2, "test2", ch2)

	e.Publish("test2", []byte("1"))
	e.Publish("test", []byte("3"))
	e.Publish("test", "4")

	go func() {
		time.Sleep(time.Second * 2)
		cancel()
	}()

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case v := <-ch:
			switch v.(type) {
			case string:
				log.Println("receive string:", v.(string))
			case []byte:
				log.Println("receive []byte:", string(v.([]byte)))
			}
		}
	}

	if e.TopicCount() != 1 {
		t.Error("not match topic count")
		return
	}
	cancel2()
	time.Sleep(time.Second)
	if e.TopicCount() != 0 {
		t.Error("not match topic count 2")
		return
	}

	if e.SubscriberCountOfTopic("test") != 0 {
		t.Error("not match subscribers")
		return
	}

	if e.SubscriberCountOfTopic("test2") != 0 {
		t.Error("not match subscribers 2")
		return
	}
}
