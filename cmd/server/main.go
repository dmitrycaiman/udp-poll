package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"udp-poll/entity"
	"udp-poll/pkg/server"
)

const (
	defaultAddress = ":12345"
)

func main() {
	address := flag.String("a", defaultAddress, "destination address")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	newServer := server.New(*address, handler)

	wg := &sync.WaitGroup{}
	listenSignals(safeExec(ctx, wg, newServer))
	cancel()
	wg.Wait()
}

func listenSignals(serverSignal <-chan error) {
	signalCh := make(chan os.Signal, 1)
	defer close(signalCh)

	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-signalCh:
		log.Printf("received OS signal: %v", s.String())
	case err := <-serverSignal:
		log.Printf("termination signal from server: %v", err)
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

// handler валидирует получаемые пакеты и формирует ответ при необходимости.
// В лог будут выведены параметры успешно обработанного запроса.
func handler(data []byte) ([]byte, error) {
	receivedAt := time.Now().UTC()
	b, err := entity.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack: %w", err)
	}
	req := &entity.Request{}
	if err := json.Unmarshal(b, req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	b, err = json.Marshal(
		&entity.Response{
			Number:     req.Number,
			CreatedAt:  req.CreatedAt,
			ReceivedAt: receivedAt.Unix(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}
	pack, err := entity.NewPack(b)
	if err != nil {
		return nil, fmt.Errorf("failed to create pack: %w", err)
	}

	log.Printf(
		"successfully handled message №%v: created at %v, received at %v",
		req.Number,
		time.Unix(req.CreatedAt, 0).Format(time.DateTime),
		receivedAt.Format(time.DateTime),
	)
	return pack, nil
}
