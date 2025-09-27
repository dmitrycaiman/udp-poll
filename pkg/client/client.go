package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
)

const bufferSize = 1 << 16

type Client struct {
	conn    net.Conn
	readCh  chan []byte
	address string
}

func New(address string) *Client { return &Client{address: address, readCh: make(chan []byte)} }

func (slf *Client) Run(ctx context.Context) error {
	conn, err := net.Dial("udp", slf.address)
	if err != nil {
		return fmt.Errorf("failed to dial to <%v>: %w", slf.address, err)
	}
	slf.conn = conn

	go func() {
		<-ctx.Done()
		slf.close()
	}()

	log.Printf("client started: local address <%v>, remote address <%v>\n", slf.conn.LocalAddr(), slf.conn.RemoteAddr())
	return slf.read(ctx)
}

func (slf *Client) Send(message []byte) error {
	_, err := slf.conn.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (slf *Client) Read() <-chan []byte { return slf.readCh }

func (slf *Client) read(ctx context.Context) error {
	buf := make([]byte, bufferSize)
	defer close(slf.readCh)

	for {
		n, err := slf.conn.Read(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("stop reading: %w", err)
			}
			log.Printf("failed to read: %v\n", err)
			continue
		}

		message := make([]byte, n)
		copy(message, buf[:n])

		select {
		case slf.readCh <- message:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (slf *Client) close() {
	if err := slf.conn.Close(); err != nil {
		log.Printf("failed to close client: %v\n", err)
		return
	}
	log.Printf("shutdown...\n")
}
