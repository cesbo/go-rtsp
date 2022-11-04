package rtsp

import (
	"bufio"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequest_parseRequestLine(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		method, requestURI, proto, ok := parseRequestLine("OPTIONS rtsp://example.com RTSP/1.0")
		assert.Equal(t, "OPTIONS", method)
		assert.Equal(t, "rtsp://example.com", requestURI)
		assert.Equal(t, "RTSP/1.0", proto)
		assert.True(t, ok)
	})

	t.Run("valid without proto", func(t *testing.T) {
		method, requestURI, proto, ok := parseRequestLine("OPTIONS rtsp://example.com")
		assert.Equal(t, "OPTIONS", method)
		assert.Equal(t, "rtsp://example.com", requestURI)
		assert.Equal(t, "", proto)
		assert.True(t, ok)
	})
}

func TestRequest_Options(t *testing.T) {
	assert := assert.New(t)

	u, _ := url.Parse("rtsp://user:secret@localhost:8554/test")
	c := &Client{
		URL:       u,
		UserAgent: "test/0.0",
	}

	var sb strings.Builder
	c.bw = bufio.NewWriter(&sb)

	req := &Request{
		Method: MethodOptions,
		URL:    u,
	}
	if err := c.SendRequest(req); !assert.NoError(err) {
		return
	}

	s := sb.String()

	// Should be empty line at the end
	if !assert.True(strings.HasSuffix(s, "\r\n\r\n")) {
		return
	}

	result := strings.Split(s, "\r\n")
	assert.Equal("OPTIONS rtsp://localhost:8554/test RTSP/1.0", result[0])

	expectedHeaders := map[string]string{
		"CSeq":       "1",
		"User-Agent": "test/0.0",
	}

	resultHeaders := map[string]string{}
	for _, line := range result[1:] {
		if line == "" {
			break
		}
		if h, v, ok := strings.Cut(line, ": "); ok {
			resultHeaders[h] = v
		}
	}

	assert.Equal(expectedHeaders, resultHeaders)
}

func TestRequest_Describe(t *testing.T) {
	assert := assert.New(t)

	u, _ := url.Parse("rtsp://user:secret@localhost:8554/test")
	c := &Client{URL: u}

	// receive 401 Unauthorized
	responseData := bufio.NewReader(strings.NewReader(strings.Join(
		[]string{
			`RTSP/1.0 401 Unauthorized`,
			`CSeq: 1`,
			`WWW-Authenticate: Digest realm="test", nonce="1234"`,
			``,
			``,
		},
		"\r\n",
	)))
	response, err := ReadResponse(responseData)
	if !assert.NoError(err) {
		return
	}
	c.auth = NewAuth(c.URL, response.Header.Get("WWW-Authenticate"))
	if !assert.NotNil(c.auth) {
		return
	}

	var sb strings.Builder
	c.bw = bufio.NewWriter(&sb)
	req := &Request{
		Method: MethodDescribe,
		URL:    u,
	}
	if err := c.SendRequest(req); !assert.NoError(err) {
		return
	}

	result := strings.Split(sb.String(), "\r\n")

	expectedHeaders := map[string]string{
		"CSeq": "1",
		"Authorization": strings.Join(
			[]string{
				`Digest username="user"`,
				`uri="rtsp://localhost:8554/test"`,
				`realm="test"`,
				`nonce="1234"`,
				`response="73cef206011b7b200d349acffd2d695b"`,
			},
			", ",
		),
	}

	resultHeaders := map[string]string{}
	for _, line := range result[1:] {
		if line == "" {
			break
		}
		if h, v, ok := strings.Cut(line, ": "); ok {
			resultHeaders[h] = v
		}
	}

	assert.Equal(expectedHeaders, resultHeaders)
}
