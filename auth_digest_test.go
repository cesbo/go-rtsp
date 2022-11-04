package rtsp

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthDigest_Header(t *testing.T) {
	assert := assert.New(t)

	u, err := url.Parse("rtsp://test:pass@127.0.0.1:34001/digest/")
	if !assert.NoError(err) {
		return
	}

	// uri is the request URL without credentials
	// in real code will be droppend on SendRequest()
	uri := "rtsp://127.0.0.1:34001/digest/"

	authItems := []string{
		`realm="test"`,
		`domain="digest"`,
		`nonce="9a52e5d50ca0f63e5b0b9188b1e32a15"`,
	}
	authHeader := strings.Join(authItems, ", ")

	login := u.User.Username()
	password, _ := u.User.Password()
	auth := NewAuthDigest(login, password, authHeader)

	expectedItems := []string{
		`Digest username="test"`,
		`uri="rtsp://127.0.0.1:34001/digest/"`,
		`realm="test"`,
		`nonce="9a52e5d50ca0f63e5b0b9188b1e32a15"`,
		`response="7e50c7dadf6909bc91e389b258788707"`,
	}
	expectedHeader := strings.Join(expectedItems, ", ")

	assert.Equal(
		expectedHeader,
		auth.Header("DESCRIBE", uri),
	)
}
