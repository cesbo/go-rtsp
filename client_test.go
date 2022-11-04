package rtsp

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testServer(
	fn func(ctx context.Context, r *Request, w *bufio.Writer),
) (addr string, close context.CancelFunc) {
	srv, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil
	}

	addr = srv.Addr().String()

	ctx := context.Background()
	ctx, close = context.WithCancel(ctx)

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	go func() {
		conn, _ := srv.Accept()
		defer conn.Close()

		r := bufio.NewReader(conn)
		w := bufio.NewWriter(conn)

		for ctx.Err() == nil {
			request, _ := ReadRequest(r)
			fn(ctx, request, w)
		}
	}()

	return addr, close
}

func TestClient_getSession(t *testing.T) {
	type fields struct {
		name     string
		header   http.Header
		expected string
	}

	tests := []fields{
		{
			name: "OK",
			header: http.Header{
				"Session": []string{"12345678"},
			},
			expected: "12345678",
		},
		{
			name:     "No session",
			header:   http.Header{},
			expected: "",
		},
		{
			name: "OK with spaces",
			header: http.Header{
				"Session": []string{" 12345678 "},
			},
			expected: "12345678",
		},
		{
			name: "with timeout",
			header: http.Header{
				"Session": []string{" 12345678 ; timeout=999"},
			},
			expected: "12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			r := &Response{
				StatusCode: 200,
				Header:     tt.header,
			}

			assert.Equal(tt.expected, getSession(r))
		})
	}
}

func TestClient_Setup(t *testing.T) {
	t.Run("skip unknown transport", func(t *testing.T) {
		assert := assert.New(t)

		sdp := strings.Join(
			[]string{
				`v=0`,
				`o=- 0 0 IN IP4 192.168.10.93`,
				`s=Unnamed`,
				`i=N/A`,
				`c=IN IP4 192.168.10.95`,
				`t=0 0`,
				`a=recvonly`,
				`m=video 5006 RTP/AVP 97`,
				`a=rtpmap:97 H264/90000`,
				`m=audio 8004 RTP/AVP 111`,
				`a=rtpmap:111 UNK/8000`,
				`m=audio 5004 RTP/AVP 96`,
				`a=rtpmap:96 mpeg4-generic/8000/2`,
			},
			"\r\n",
		)

		addr, closeServer := testServer(
			func(ctx context.Context, r *Request, w *bufio.Writer) {
				switch r.Method {
				case MethodOptions:
					w.WriteString("RTSP/1.0 200 OK\r\n")
					w.WriteString("CSeq: 1\r\n")
					w.WriteString("Public: OPTIONS, DESCRIBE, SETUP, PLAY, TEARDOWN\r\n")
					w.WriteString("\r\n")
					w.Flush()
					return
				case MethodDescribe:
					w.WriteString("RTSP/1.0 200 OK\r\n")
					w.WriteString("CSeq: 2\r\n")
					w.WriteString("Content-Type: application/sdp\r\n")
					w.WriteString("Content-Length: " + strconv.Itoa(len(sdp)) + "\r\n")
					w.WriteString("\r\n")
					w.WriteString(sdp)
					w.Flush()
					return
				case MethodSetup:
					cseq, _ := strconv.Atoi(r.Header.Get("CSeq"))
					if cseq >= 3 && cseq <= 4 {
						w.WriteString("RTSP/1.0 200 OK\r\n")
						w.WriteString("CSeq: " + strconv.Itoa(cseq) + "\r\n")
						w.WriteString("Session: 12345678\r\n")
						w.WriteString("\r\n")
						w.Flush()
						return
					}
				}

				w.WriteString("RTSP/1.0 500 Internal Server Error\r\n")
				w.WriteString("\r\n")
				w.Flush()
			},
		)
		defer closeServer()

		u, err := url.Parse("rtsp://" + addr)
		if !assert.NoError(err) {
			return
		}

		c := &Client{
			URL: u,
		}

		ctx := context.Background()

		if err := c.Start(ctx); !assert.NoError(err) {
			return
		}

		count := 0

		for mediaID, sdpItem := range c.GetSDP() {
			if sdpItem.Media == nil {
				continue
			}

			switch mediaID {
			case 0:
				if _, ok := sdpItem.Media.(*MediaH264); assert.True(ok) {
					count += 1
				}
			case 1:
				assert.Fail("mediaID=1 should be skipped")
			case 2:
				if _, ok := sdpItem.Media.(*MediaMPEG4); assert.True(ok) {
					count += 1
				}
			}

			if err := c.Setup(ctx, mediaID, sdpItem.URL); !assert.NoError(err) {
				return
			}
		}

		assert.Equal(2, count)
	})
}

func TestClient_do(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		assert := assert.New(t)

		requestReceived := false

		addr, closeServer := testServer(
			func(ctx context.Context, r *Request, w *bufio.Writer) {
				requestReceived = (r.Method == MethodOptions)

				w.WriteString("RTSP/1.0 200 OK\r\n")
				<-ctx.Done()
			},
		)
		defer closeServer()

		clientErrorCh := make(chan error, 1)
		ctxClient, closeClient := context.WithCancel(context.Background())

		go func() {
			u, _ := url.Parse("rtsp://" + addr)
			rtspClient := &Client{
				URL: u,
			}
			defer rtspClient.Close()

			err := rtspClient.Start(ctxClient)
			clientErrorCh <- err
		}()

		time.Sleep(50 * time.Millisecond)
		closeClient()

		assert.True(requestReceived)

		timer := time.NewTimer(100 * time.Millisecond)
		defer timer.Stop()

		select {
		case err := <-clientErrorCh:
			assert.ErrorIs(err, context.Canceled)
		case <-timer.C:
			assert.Fail("timeout")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		assert := assert.New(t)

		requestReceived := false

		addr, closeServer := testServer(
			func(ctx context.Context, r *Request, w *bufio.Writer) {
				requestReceived = (r.Method == MethodOptions)
				w.WriteString("RTSP/1.0 200 OK\r\n")
				<-ctx.Done()
			},
		)
		defer closeServer()

		clientErrorCh := make(chan error, 1)

		go func() {
			u, _ := url.Parse("rtsp://" + addr)
			rtspClient := &Client{
				URL:            u,
				RequestTimeout: 50 * time.Millisecond,
			}
			defer rtspClient.Close()

			err := rtspClient.Start(context.Background())
			clientErrorCh <- err
		}()

		time.Sleep(50 * time.Millisecond)
		assert.True(requestReceived)

		timer := time.NewTimer(100 * time.Millisecond)
		defer timer.Stop()

		select {
		case err := <-clientErrorCh:
			assert.ErrorIs(err, ErrResponseTimeout)
		case <-timer.C:
			assert.Fail("timeout")
		}
	})
}

func TestClient_SDP_stuck(t *testing.T) {
	assert := assert.New(t)

	addr, closeServer := testServer(
		func(ctx context.Context, r *Request, w *bufio.Writer) {
			switch r.Method {
			case MethodOptions:
				w.WriteString("RTSP/1.0 200 OK\r\n")
				w.WriteString("CSeq: 1\r\n")
				w.WriteString("Public: OPTIONS, DESCRIBE, SETUP, PLAY, TEARDOWN\r\n")
				w.WriteString("\r\n")
				w.Flush()
			case MethodDescribe:
				w.WriteString("RTSP/1.0 200 OK\r\n")
				w.WriteString("CSeq: 2\r\n")
				w.WriteString("Content-Type: application/sdp\r\n")
				w.WriteString("Content-Length: 100\r\n")
				w.WriteString("\r\n")
				w.Flush()
				<-ctx.Done() // stuck until context is canceled
			default:
				assert.Fail("unexpected method")
			}
		},
	)
	defer closeServer()

	clientErrorCh := make(chan error, 1)
	ctxClient, closeClient := context.WithCancel(context.Background())

	go func() {
		u, _ := url.Parse("rtsp://" + addr)
		rtspClient := &Client{
			URL: u,
		}
		defer rtspClient.Close()

		err := rtspClient.Start(ctxClient)
		clientErrorCh <- err
	}()

	time.Sleep(50 * time.Millisecond)
	closeClient()

	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()

	select {
	case err := <-clientErrorCh:
		assert.ErrorIs(err, context.Canceled)
	case <-timer.C:
		assert.Fail("timeout")
	}
}
