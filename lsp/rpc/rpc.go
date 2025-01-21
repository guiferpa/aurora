package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Example message:
//
//	Content-Length: ...\r\n
//	\r\n
//	{
//		"jsonrpc": "2.0",
//		"id": 1,
//		"method": "textDocument/completion",
//		"params": {
//			...
//		}
//	}
func EncodeMessage(msg any) string {
	content, err := json.Marshal(msg)
	if err != nil {
		// if can't encode our message we are in trouble, so we should just stop
		panic(err)
	}

	// now time to build message as per lsp specs:
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification

	return fmt.Sprintf(
		"Content-Length: %d\r\n\r\n%s",
		len(content),
		content,
	)
}

type BaseMessage struct {
	Method string `json:"method"`
}

// decode header and context
func DecodeMessage(msg []byte) (string, []byte, error) {
	header, content, found := bytes.Cut(msg, []byte{'\r', '\n', '\r', '\n'})
	if !found {
		return "", nil, errors.New("did not find separator")
	}

	// Content-Length: <number>
	contentLengthBytes := header[len("Content-Length: "):]
	contentLength, err := strconv.Atoi(string(contentLengthBytes))
	if err != nil {
		return "", nil, err
	}

	var baseMessage BaseMessage
	if err := json.Unmarshal(content[:contentLength], &baseMessage); err != nil {
		return "", nil, err
	}

	return baseMessage.Method, content[:contentLength], nil
}

// https://pkg.go.dev/bufio#SplitFunc
// not using atEOF because we want to read forever as it is a long running process
func Split(data []byte, _ bool) (advance int, token []byte, err error) {
	// goal is to read the header first which gives content length
	//  if we haven't read all bytes to read a full content i.e. the json. we do nothing and check again in the next cycle when we have collected enough bytes
	//  if we have all the bytes to form a full content, then we send advance token to let os. what we have read the byes and this must buffer can be cleared

	header, content, found := bytes.Cut(data, []byte{'\r', '\n', '\r', '\n'})
	if !found {
		// means we are waiting for more information and not ready now
		return 0, nil, nil
	}

	// Content-Length: <number>
	contentLengthBytes := header[len("Content-Length: "):]
	contentLength, err := strconv.Atoi(string(contentLengthBytes))
	if err != nil {
		// this means we did not get a number in the content length, so we are throwing error
		return 0, nil, err
	}

	if len(content) < contentLength {
		// not ready
		return 0, nil, nil
	}

	// 4 because we have 4 separators
	totalLength := len(header) + 4 + contentLength

	// return data up until total length
	return totalLength, data[:totalLength], nil
}
