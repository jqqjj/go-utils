package main

import (
	"context"
	"fmt"
	"github.com/jqqjj/go-utils"
	"math/rand"
	"sync"
	"time"
)

func main() {
	w := utils.NewWorkerPool(1)

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1000)
	go func() {
		for i := 0; i < 1000; i++ {
			index := i
			ch := w.Submit(ctx, func(ctx context.Context) {
				time.Sleep(time.Second * 3)
				fmt.Println(time.Now().Format("2006-01-02 15:04:05"), index)
			})
			go func() {
				<-ch
				wg.Done()
			}()
		}
	}()

	go func() {
		subCtx, subCancel := context.WithTimeout(ctx, time.Second*130)
		defer subCancel()

		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	LOOP:
		for {
			n := rd.Intn(7)
			w.SetWorkerNum(n)

			select {
			case <-subCtx.Done():
				break LOOP
			default:
				time.Sleep(time.Second * 3)
			}
		}

		fmt.Println("修改为1进程")
		w.SetWorkerNum(1)
	}()

	wg.Wait()
	fmt.Println("退出")
	cancel()
}
