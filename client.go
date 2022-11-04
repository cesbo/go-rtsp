package rtsp

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Method Definitions
const (
	MethodDescribe     = "DESCRIBE"
	MethodGetParameter = "GET_PARAMETER"
	MethodOptions      = "OPTIONS"
	MethodPlay         = "PLAY"
	MethodSetup        = "SETUP"
	MethodTeardown     = "TEARDOWN"
)

var (
	ErrClientClosed    = fmt.Errorf("client closed")
	ErrResponseTimeout = fmt.Errorf("response timeout")
)

// Real Time Streaming Protocol (RTSP)
// https://datatracker.ietf.org/doc/html/rfc2326
type Client struct {
	UseTCP         bool
	URL            *url.URL
	UserAgent      string
	ConnectTimeout time.Duration
	RequestTimeout time.Duration

	conn net.Conn
	br   *bufio.Reader
	bw   *bufio.Writer

	cseq    int
	session string
	auth    Auth
	sdp     []*SdpItem

	transport Transport
}

const defaultTimeout = 5 * time.Second

func getSession(response *Response) string {
	if session := response.Header.Get("Session"); session != "" {
		session, _, _ = strings.Cut(session, ";")
		return strings.TrimSpace(session)
	}

	return ""
}

func (c *Client) do(ctx context.Context, request *Request) (response *Response, err error) {
	cseq := c.cseq

	done := make(chan error, 1)
	defer func() {
		select {
		case err = <-done:
			// context is canceled. socket closed in goroutine.
		default:
			// func is finished. do nothing.
		}

		close(done)
	}()

	go func() {
		// wait for context cancel or timeout
		timeout := time.NewTimer(c.RequestTimeout)
		defer timeout.Stop()

		select {
		case <-done:
			// func is finished. do nothing
		case <-ctx.Done():
			// context is canceled. close socket to break IO
			done <- ctx.Err()
			c.conn.Close()
		case <-timeout.C:
			// timeout. close socket to break IO
			done <- ErrResponseTimeout
			c.conn.Close()
		}
	}()

	for {
		if err = c.SendRequest(request); err != nil {
			return nil, err
		}

		response, err = ReadResponse(c.br)
		if err != nil {
			return nil, err
		}

		err = response.ReadBody(c.br)
		if err != nil {
			return nil, err
		}

		if response.StatusCode == http.StatusUnauthorized {
			// Limit authentication attempts
			if (c.cseq - cseq) > 1 {
				break
			}

			if h := response.Header.Get("www-authenticate"); h != "" {
				c.auth = NewAuth(request.URL, h)
				continue
			}
		}

		break
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", response.Status)
	}

	return response, nil
}

// Start connectes to the RTSP server and get SDP
func (c *Client) Start(ctx context.Context) error {
	c.session = ""

	var (
		response *Response
		err      error
	)

	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = defaultTimeout
	}

	if c.RequestTimeout == 0 {
		c.RequestTimeout = defaultTimeout
	}

	dialer := &net.Dialer{
		Timeout: c.ConnectTimeout,
	}

	c.conn, err = dialer.DialContext(ctx, "tcp", c.URL.Host)
	if err != nil {
		return err
	}

	c.br = bufio.NewReader(c.conn)
	c.bw = bufio.NewWriter(c.conn)

	if c.UseTCP {
		c.transport = NewTransportTCP(c.br)
	} else {
		c.transport = NewTransportUDP()
	}

	request := &Request{
		Method: MethodOptions,
		URL:    c.URL,
	}

	// OPTIONS
	if _, err = c.do(ctx, request); err != nil {
		return err
	}

	// DESCRIBE
	request.Method = MethodDescribe
	request.Header = http.Header{
		"Accept": []string{SdpMimeType},
	}

	if response, err = c.do(ctx, request); err != nil {
		return err
	}

	// Content-Base
	controlURL := c.URL
	if v := response.Header.Get("content-base"); v != "" {
		controlURL, err = url.Parse(v)
		if err != nil {
			return fmt.Errorf("invalid content-base: %w", err)
		}
	}

	c.sdp, err = ParseSDP(controlURL, response.Body)
	if err != nil {
		return fmt.Errorf("invalid sdp: %w", err)
	}

	return nil
}

// GetSDP returns SDP items
func (c *Client) GetSDP() []*SdpItem {
	return c.sdp
}

// Setup sends request to setup the stream delivery
func (c *Client) Setup(ctx context.Context, mediaID int, control *url.URL) error {
	if c.conn == nil {
		return ErrClientClosed
	}

	transport, err := c.transport.Setup(mediaID)
	if err != nil {
		return err
	}

	request := &Request{
		Method: MethodSetup,
		URL:    control,
		Header: http.Header{
			"Transport": []string{transport},
		},
	}

	response, err := c.do(ctx, request)
	if err != nil {
		return err
	}

	if c.session == "" {
		c.session = getSession(response)
	}

	return nil
}

// Play sends request to start the stream delivery.
// Waits for ctx.Done or any error on transport.
// For UDP transport it sends keep-alive requests each 30 second
func (c *Client) Play(ctx context.Context, handler MediaHandler) error {
	if c.conn == nil {
		return ErrClientClosed
	}

	request := &Request{
		Method: MethodPlay,
		URL:    c.URL,
		Header: http.Header{
			"Range": []string{"npt=0.000-"},
		},
	}

	if _, err := c.do(ctx, request); err != nil {
		return err
	}

	c.transport.Play(handler)

	var tickerC <-chan time.Time
	if !c.UseTCP {
		// For UDP transport send keep-alive requests every 30 seconds and Teardown before exit
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		tickerC = ticker.C
	}

	for {
		select {
		case <-tickerC:
			// only for UDP
			if err := c.Ping(ctx); err != nil {
				c.Close()
				return err
			}
			continue

		case err := <-c.transport.Err():
			c.Close()
			return err

		case <-ctx.Done():
			c.Close()
			return nil
		}
	}
}

// Ping sends request to keep the connection alive
func (c *Client) Ping(ctx context.Context) error {
	if c.conn == nil {
		return ErrClientClosed
	}

	request := &Request{
		Method: MethodGetParameter,
		URL:    c.URL,
	}

	_, err := c.do(ctx, request)
	return err
}

// Teardown sends request to stop the stream delivery
func (c *Client) Teardown(ctx context.Context) error {
	if c.conn == nil {
		return ErrClientClosed
	}

	request := &Request{
		Method: MethodTeardown,
		URL:    c.URL,
	}

	_, err := c.do(ctx, request)
	return err
}

// Close closes the connection and closes all transports.
// Should be called after Play() finished
func (c *Client) Close() {
	if c.transport != nil {
		c.transport.Close()
		c.transport = nil
	}

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}
