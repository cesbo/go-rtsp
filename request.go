package rtsp

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
)

type Request struct {
	Method string
	URL    *url.URL
	Header http.Header
}

func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	method, line, ok = strings.Cut(line, " ")
	if !ok {
		return
	}

	requestURI, proto, _ = strings.Cut(line, " ")

	return
}

// ReadRequest reads request from the client.
func ReadRequest(r *bufio.Reader) (request *Request, err error) {
	tp := textproto.NewReader(r)

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

	method, requestURI, _, ok := parseRequestLine(line)
	if !ok {
		return nil, fmt.Errorf("invalid request line %q", line)
	}

	requestURL, err := url.Parse(requestURI)
	if err != nil {
		return nil, fmt.Errorf("invalid request url %s", err)
	}

	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}

	request = &Request{
		Method: method,
		URL:    requestURL,
		Header: http.Header(mimeHeader),
	}

	return request, nil
}

// SendRequest sends RTSP/1.0 request to the server.
func (c *Client) SendRequest(request *Request) error {
	var err error

	// Ommit credentials from the request line
	user := request.URL.User
	request.URL.User = nil
	uri := request.URL.String()
	request.URL.User = user

	// Send the request
	_, err = fmt.Fprintf(c.bw, "%s %s RTSP/1.0\r\n", request.Method, uri)
	if err != nil {
		return err
	}

	// User-Agent
	if c.UserAgent != "" {
		_, err = fmt.Fprintf(c.bw, "User-Agent: %s\r\n", c.UserAgent)
		if err != nil {
			return err
		}
	}

	// CSeq
	c.cseq += 1
	_, err = fmt.Fprintf(c.bw, "CSeq: %d\r\n", c.cseq)
	if err != nil {
		return err
	}

	// Session
	if c.session != "" {
		_, err = fmt.Fprintf(c.bw, "Session: %s\r\n", c.session)
		if err != nil {
			return err
		}
	}

	// Authorization
	if c.auth != nil {
		_, err = fmt.Fprintf(
			c.bw,
			"Authorization: %s\r\n",
			c.auth.Header(request.Method, uri),
		)
		if err != nil {
			return err
		}
	}

	// Header
	if request.Header != nil {
		err = request.Header.Write(c.bw)
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprint(c.bw, "\r\n")
	if err != nil {
		return err
	}

	return c.bw.Flush()
}
