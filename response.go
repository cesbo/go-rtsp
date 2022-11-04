package rtsp

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
)

type Response struct {
	Proto      string
	StatusCode int
	Status     string
	Header     http.Header
	Body       []byte
}

func parseResponseLine(line string) (proto, status string, code int, ok bool) {
	proto, status, ok = strings.Cut(line, " ")
	if !ok {
		return
	}

	status = strings.TrimSpace(status)
	statusCode, _, _ := strings.Cut(status, " ")

	var err error
	code, err = strconv.Atoi(statusCode)
	ok = (err == nil) && (code >= 100) && (code <= 999)

	return
}

// ReadResponse reads a response from the server.
func ReadResponse(reader *bufio.Reader) (response *Response, err error) {
	tp := textproto.NewReader(reader)

	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	var line string
	line, err = tp.ReadLine()
	if err != nil {
		return nil, err
	}

	proto, status, code, ok := parseResponseLine(line)
	if !ok {
		return nil, fmt.Errorf("invalid response line %q", line)
	}

	// Parse the response headers.
	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}

	response = &Response{
		Proto:      proto,
		StatusCode: code,
		Status:     status,
		Header:     http.Header(mimeHeader),
	}

	return response, nil
}

// ReadBody reads the body after response.
func (r *Response) ReadBody(reader *bufio.Reader) error {
	v := r.Header.Get("content-length")
	if v == "" {
		return nil
	}

	contentLength, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("invalid content-length %q", v)
	}

	if contentLength == 0 {
		return nil
	}

	if contentLength > (64 * 1024) {
		return fmt.Errorf("content-length too large %d", contentLength)
	}

	r.Body = make([]byte, contentLength)
	if _, err = io.ReadFull(reader, r.Body); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return err
	}

	return nil
}
