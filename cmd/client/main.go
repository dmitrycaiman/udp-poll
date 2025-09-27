package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"udp-poll/pkg/client"
	"udp-poll/pkg/poll"
)

const (
	defaultTaskCount   = 10_000
	defaultWorkerCount = 10
	defaultAddress     = ":12345"
)

func main() {
	taskCount := flag.Int("n", defaultTaskCount, "number of requests to send")
	workerCount := flag.Int("w", defaultWorkerCount, "number of workers")
	address := flag.String("a", defaultAddress, "destination address")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	newClient := client.New(*address)
	newPoll := poll.New(newClient, *taskCount, *workerCount)

	wg := &sync.WaitGroup{}
	listenSignals(safeExec(ctx, wg, newClient), safeExec(ctx, wg, newPoll))
	cancel()
	wg.Wait()
}

func listenSignals(clientSignal, pollSignal <-chan error) {
	signalCh := make(chan os.Signal, 1)
	defer close(signalCh)

	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-signalCh:
		log.Printf("received OS signal: %v", s.String())
	case err := <-clientSignal:
		log.Printf("termination signal from client: %v", err)
	case err := <-pollSignal:
		log.Printf("termination signal from poll: %v", err)
	}
}

func safeExec(ctx context.Context, wg *sync.WaitGroup, f interface{ Run(context.Context) error }) <-chan error {
	wg.Add(1)
	errSignal := make(chan error, 1)
	go func() {
		defer wg.Done()
		defer close(errSignal)
		defer func() {
			if r := recover(); r != nil {
				log.Printf("recovered from panic: %v", r)
			}
		}()
		errSignal <- f.Run(ctx)
	}()
	return errSignal
}
