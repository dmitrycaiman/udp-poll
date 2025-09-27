package poll

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"udp-poll/entity"
	"udp-poll/pkg/client"
)

const responseTimeout = 100 * time.Millisecond

type Poll struct {
	c                        *client.Client
	processingRequests       []chan *entity.Response
	counter, success, failed atomic.Int64
	taskCount, workerCount   int
}

func New(c *client.Client, taskCount, workerCount int) *Poll {
	return &Poll{
		c:                  c,
		processingRequests: make([]chan *entity.Response, taskCount),
		taskCount:          taskCount,
		workerCount:        workerCount,
	}
}

func (slf *Poll) Run(ctx context.Context) error {
	slf.counter.Add(1)

	var wg = &sync.WaitGroup{}
	defer wg.Wait()
	sendCh := make(chan int64)
	defer close(sendCh)

	wg.Add(slf.workerCount + 1)
	for range slf.workerCount {
		go func() {
			defer wg.Done()

			for {
				select {
				case num, ok := <-sendCh:
					if !ok {
						return
					}

					message := fmt.Sprintf("successfully received message №%v", num)
					if err := slf.send(ctx, num); err != nil {
						slf.failed.Add(1)
						message = fmt.Sprintf("failed to send message №%v: %v", num, err.Error())
					} else {
						slf.success.Add(1)
					}
					slf.print(ctx, num, message)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go slf.waitForRead(ctx)
	go slf.waitForFinish(ctx, wg)

	for i := range slf.taskCount {
		select {
		case sendCh <- int64(i + 1):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("poll is stopped")
}

func (slf *Poll) waitForRead(ctx context.Context) {
	for {
		select {
		case data, ok := <-slf.c.Read():
			if !ok {
				return
			}
			go slf.receive(data)
		case <-ctx.Done():
			return
		}
	}
}

func (slf *Poll) waitForFinish(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if slf.counter.Load() == int64(slf.taskCount+1) {
			succ, fail := slf.success.Load(), slf.failed.Load()
			perc := float64(100*fail) / float64(slf.taskCount)
			log.Printf("all of %v requests are handled: success %v, failed %v (%3.1f%%)\n", slf.taskCount, succ, fail, perc)
			return
		}
		runtime.Gosched()
	}
}

func (slf *Poll) send(ctx context.Context, num int64) error {
	req, err := json.Marshal(
		&entity.Request{
			Number:    num,
			CreatedAt: time.Now().Unix(),
			Payload:   randomPayload(int(num)),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to marshal message №%v: %v", num, err)
	}
	pack, err := entity.NewPack(req)
	if err != nil {
		return fmt.Errorf("failed to create pack №%v: %v", num, err)
	}

	receiveCh := make(chan *entity.Response)
	slf.processingRequests[num-1] = receiveCh
	if err := slf.c.Send(pack); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(responseTimeout):
		return fmt.Errorf("timeout for waiting of response №%v", num)
	case res := <-receiveCh:
		if res.Number != num {
			return fmt.Errorf("unexpected response num: wanted №%v, got №%v", num, res.Number)
		}
		return nil
	}
}

func (slf *Poll) receive(data []byte) error {
	data, err := entity.Unpack(data)
	if err != nil {
		log.Printf("failed to unpack message: %v\n", err)
		return err
	}
	res := &entity.Response{}
	if err := json.Unmarshal(data, res); err != nil {
		log.Printf("failed to unmarshal message: %v\n", err)
		return err
	}
	if res.Number <= 0 || res.Number > int64(slf.taskCount) {
		return fmt.Errorf("wrong response number: wanted [1, %v], got %v", slf.taskCount, res.Number)
	}

	select {
	case slf.processingRequests[res.Number-1] <- res:
		return nil
	default:
		return fmt.Errorf("abort unexpected message №%v", res.Number)
	}
}

func (slf *Poll) print(ctx context.Context, num int64, message string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if slf.counter.CompareAndSwap(num, -num) {
			log.Println(message)
			slf.counter.CompareAndSwap(-num, num+1)
			return
		}
		runtime.Gosched()
	}
}
