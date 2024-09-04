package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type HttpRequestHandler struct {
	conn      net.Conn
	directory string
}

func (h *HttpRequestHandler) HandleRequest() {
	defer h.conn.Close()

	reader := bufio.NewReader(h.conn)
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading request:", err)
		return
	}

	method, target, _, err := parseRequestLine(requestLine)
	if err != nil {
		h.sendResponse(http.StatusBadRequest, "Bad Request")
		return
	}

	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			headers[parts[0]] = parts[1]
		}
	}

	switch method {
	case "GET":
		h.handleGet(target)
	default:
		h.sendResponse(http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func parseRequestLine(line string) (string, string, string, error) {
	parts := strings.Fields(line)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid request line")
	}
	return parts[0], parts[1], parts[2], nil
}

func (h *HttpRequestHandler) handleGet(target string) {
	if target == "/" {
		h.sendResponse(http.StatusOK, "OK")
		return
	}

	if strings.HasPrefix(target, "/echo/") {
		msg := strings.TrimPrefix(target, "/echo/")
		h.sendResponseWithBody(http.StatusOK, "OK", msg, false)
		return
	}

	filePath := filepath.Join(h.directory, target)
	if fileInfo, err := os.Stat(filePath); err == nil && !fileInfo.IsDir() {
		file, err := os.Open(filePath)
		if err != nil {
			h.sendResponse(http.StatusInternalServerError, "Internal Server Error")
			return
		}
		defer file.Close()

		h.conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
		h.conn.Write([]byte("Content-Type: text/plain\r\n"))
		h.conn.Write([]byte("\r\n"))
		io.Copy(h.conn, file)
	} else {
		h.sendResponse(http.StatusNotFound, "Not Found")
	}
}

func (h *HttpRequestHandler) sendResponseWithBody(statusCode int, statusText, body string, gzip bool) {
	headers := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n", statusCode, statusText, len(body))
	h.conn.Write([]byte(headers))
	h.conn.Write([]byte(body))
}

func (h *HttpRequestHandler) sendResponse(statusCode int, statusText string) {
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n\r\n", statusCode, statusText)
	h.conn.Write([]byte(response))
}
