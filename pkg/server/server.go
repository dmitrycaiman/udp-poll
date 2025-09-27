package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
)

const bufferSize = 1 << 15

type Handler func(data []byte) ([]byte, error)

type Server struct {
	listener net.PacketConn
	handler  Handler
	address  string
}

func New(address string, handler Handler) *Server {
	return &Server{handler: handler, address: address}
}

func (slf *Server) Run(ctx context.Context) error {
	listener, err := net.ListenPacket("udp", slf.address)
	if err != nil {
		return fmt.Errorf("failed to start listening on address <%v>: %w", slf.address, err)
	}
	slf.listener = listener
	go func() {
		<-ctx.Done()
		slf.close()
	}()

	log.Printf("waiting for data on <%v>...\n", listener.LocalAddr())
	buf := make([]byte, bufferSize)
	for {
		n, addr, err := listener.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("listener was closed: %w", err)
			}
			log.Printf("failed to read from <%v>: %v\n", addr.String(), err)
			continue
		}

		message := make([]byte, n)
		copy(message, buf[:n])
		go slf.serve(message, addr)
	}
}

func (slf *Server) serve(message []byte, addr net.Addr) {
	answer, err := slf.handler(message)
	if err != nil {
		log.Printf("failed to handle message: %v\n", err)
		return
	}

	_, err = slf.listener.WriteTo(answer, addr)
	if err != nil {
		log.Printf("failed to answer: %v\n", err)
		return
	}
}

func (slf *Server) close() {
	if err := slf.listener.Close(); err != nil {
		log.Printf("failed to close listener: %v\n", err)
		return
	}
	log.Printf("shutdown...\n")
}
