package rtsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponse_parseResponseLine(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert := assert.New(t)

		proto, status, code, ok := parseResponseLine("RTSP/1.0 200 OK")
		assert.Equal("RTSP/1.0", proto)
		assert.Equal("200 OK", status)
		assert.Equal(200, code)
		assert.True(ok)
	})

	t.Run("valid without reason", func(t *testing.T) {
		assert := assert.New(t)

		proto, status, code, ok := parseResponseLine("RTSP/1.0 200")
		assert.Equal("RTSP/1.0", proto)
		assert.Equal("200", status)
		assert.Equal(200, code)
		assert.True(ok)
	})
}
