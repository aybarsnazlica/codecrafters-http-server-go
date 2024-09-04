package main

import (
	"flag"
)

func main() {
	directory := flag.String("directory", "", "Directory to serve files from")
	flag.Parse()

	server := NewHttpServer(*directory)
	server.Start()
}
