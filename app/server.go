package main

import (
	"fmt"
	"net"
	"sync"
)

const (
	port = 4221
)

type HttpServer struct {
	directory string
	running   bool
	wg        sync.WaitGroup
}

func NewHttpServer(directory string) *HttpServer {
	return &HttpServer{
		directory: directory,
		running:   true,
	}
}

func (s *HttpServer) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {

		}
	}(listener)

	fmt.Printf("Server is listening on port %d\n", port)

	for s.running {
		conn, err := listener.Accept()
		if err != nil {
			if s.running {
				fmt.Println("Error accepting connection:", err)
			}
			continue
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			handler := HttpRequestHandler{conn: conn, directory: s.directory}
			handler.Handle()
		}()
	}
	s.wg.Wait()
}

func (s *HttpServer) Stop() {
	s.running = false
	fmt.Println("Server stopping...")
}
