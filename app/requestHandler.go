package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type HttpRequestHandler struct {
	conn      net.Conn
	directory string
}

func (h *HttpRequestHandler) Handle() {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(h.conn)
	reader := bufio.NewReader(h.conn)

	// Read the first line (request line)
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading request line:", err)
		return
	}

	h.processRequest(line, reader)
}

func (h *HttpRequestHandler) processRequest(line string, reader *bufio.Reader) {
	tokens := strings.Fields(line)
	if len(tokens) != 3 {
		h.sendResponse("HTTP/1.1", 400, "Bad Request", "")
		return
	}

	httpMethod := tokens[0]
	requestTarget := tokens[1]
	httpVersion := tokens[2]

	// Read headers
	headers := make(map[string]string)
	for {
		headerLine, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading header line:", err)
			return
		}

		headerLine = strings.TrimSpace(headerLine)
		if headerLine == "" {
			break
		}

		headerTokens := strings.SplitN(headerLine, ": ", 2)
		if len(headerTokens) == 2 {
			headers[headerTokens[0]] = headerTokens[1]
		}
	}

	switch httpMethod {
	case "GET":
		h.handleGet(requestTarget, httpVersion, headers)
	case "POST":
		h.handlePost(requestTarget, httpVersion, reader, headers)
	default:
		h.sendResponse(httpVersion, 405, "Method Not Allowed", "")
	}
}

func (h *HttpRequestHandler) handleGet(target, httpVersion string, headers map[string]string) {
	switch {
	case target == "/":
		h.sendResponse(httpVersion, 200, "OK", "OK")
	case strings.HasPrefix(target, "/files"):
		h.handleFileRequest(target, httpVersion)
	case target == "/user-agent":
		h.handleUserAgentRequest(httpVersion, headers)
	case strings.HasPrefix(target, "/echo"):
		h.handleEchoRequest(target, httpVersion, headers)
	default:
		h.sendResponse(httpVersion, 404, "Not Found", "")
	}
}

func (h *HttpRequestHandler) handlePost(target, httpVersion string, reader *bufio.Reader, headers map[string]string) {
	if strings.HasPrefix(target, "/files") {
		h.handleFileCreation(target, httpVersion, reader, headers)
	}
}

func (h *HttpRequestHandler) handleFileCreation(target, httpVersion string, reader *bufio.Reader, headers map[string]string) {
	fileName := strings.TrimPrefix(target, "/files/")
	filePath := filepath.Join(h.directory, fileName)
	contentLength, err := strconv.Atoi(headers["Content-Length"])
	if err != nil {
		h.sendResponse(httpVersion, 400, "Bad Request", "")
		return
	}

	body := make([]byte, contentLength)
	_, err = io.ReadFull(reader, body)
	if err != nil {
		h.sendResponse(httpVersion, 500, "Internal Server Error", "")
		return
	}

	err = os.WriteFile(filePath, body, 0644)
	if err != nil {
		h.sendResponse(httpVersion, 500, "Internal Server Error", "")
		return
	}

	h.sendResponse(httpVersion, 201, "Created", "")
}

func (h *HttpRequestHandler) handleFileRequest(target, httpVersion string) {
	fileName := strings.TrimPrefix(target, "/files/")
	filePath := filepath.Join(h.directory, fileName)

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		h.sendResponse(httpVersion, 404, "Not Found", "")
		return
	}

	headers := fmt.Sprintf("Content-Type: application/octet-stream\r\nContent-Length: %d\r\n", len(fileContent))
	h.sendResponseWithHeaders(httpVersion, 200, "OK", headers, fileContent)
}

func (h *HttpRequestHandler) handleUserAgentRequest(httpVersion string, headers map[string]string) {
	userAgent := headers["User-Agent"]
	if userAgent == "" {
		h.sendResponse(httpVersion, 400, "Bad Request", "")
		return
	}

	h.sendResponse(httpVersion, 200, "OK", userAgent)
}

func (h *HttpRequestHandler) handleEchoRequest(target, httpVersion string, headers map[string]string) {
	msg := strings.TrimPrefix(target, "/echo/")
	acceptEncoding := headers["Accept-Encoding"]
	isGzip := strings.Contains(acceptEncoding, "gzip")

	h.sendResponse(httpVersion, 200, "OK", msg, isGzip)
}

func (h *HttpRequestHandler) sendResponse(httpVersion string, statusCode int, statusMessage, content string, isGzip ...bool) {
	var responseBody []byte
	if len(isGzip) > 0 && isGzip[0] {
		var byteStream strings.Builder
		gzipWriter := gzip.NewWriter(&byteStream)
		_, err := gzipWriter.Write([]byte(content))
		if err != nil {
			fmt.Println("Error writing is_gzip content:", err)
			return
		}
		err = gzipWriter.Close()
		if err != nil {
			return
		}
		responseBody = []byte(byteStream.String())
	} else {
		responseBody = []byte(content)
	}

	headers := fmt.Sprintf("Content-Type: text/plain\r\nContent-Length: %d\r\n", len(responseBody))
	if len(isGzip) > 0 && isGzip[0] {
		headers += "Content-Encoding: gzip\r\n"
	}

	h.sendResponseWithHeaders(httpVersion, statusCode, statusMessage, headers, responseBody)
}

func (h *HttpRequestHandler) sendResponseWithHeaders(httpVersion string, statusCode int, statusMessage, headers string, body []byte) {
	response := fmt.Sprintf("%s %d %s\r\n%s\r\n", httpVersion, statusCode, statusMessage, headers)
	_, err := h.conn.Write([]byte(response))
	if err != nil {
		return
	}
	_, err = h.conn.Write(body)
	if err != nil {
		return
	}
	err = h.conn.Close()
	if err != nil {
		return
	}
}
