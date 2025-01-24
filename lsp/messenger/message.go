package messenger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
func Encode(msg any) string {
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
func Decode(msg []byte) (string, []byte, error) {
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

func Split(data []byte, _ bool) (advance int, token []byte, err error) {
	separators := []byte{'\r', '\n', '\r', '\n'}
	header, content, found := bytes.Cut(data, separators)
	if !found {
		return 0, nil, nil
	}

	// Content-Length: <number>
	contentLengthBytes := header[len("Content-Length: "):]
	contentLength, err := strconv.Atoi(string(contentLengthBytes))
	if err != nil {
		return 0, nil, err
	}

	if len(content) < contentLength {
		return 0, nil, nil
	}

	totalLength := len(header) + len(separators) + contentLength

	return totalLength, data[:totalLength], nil
}

func Write(writer io.Writer, msg any) (int, error) {
	if msg == nil {
		return 0, nil
	}
	reply := Encode(msg)
	return writer.Write([]byte(reply))
}
