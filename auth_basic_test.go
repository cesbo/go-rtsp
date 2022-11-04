package rtsp

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthBasic_Header(t *testing.T) {
	assert := assert.New(t)

	u, err := url.Parse("rtsp://test:pass@127.0.0.1:34001/digest/")
	if !assert.NoError(err) {
		return
	}

	// uri is the request URL without credentials
	// in real code will be droppend on SendRequest()
	uri := "rtsp://127.0.0.1:34001/digest/"

	login := u.User.Username()
	password, _ := u.User.Password()
	auth := NewAuthBasic(login, password, `realm="test"`)

	expectedHeader := `Basic dGVzdDpwYXNz`

	assert.Equal(
		expectedHeader,
		auth.Header("DESCRIBE", uri),
	)
}
