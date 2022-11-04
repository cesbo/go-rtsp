package rtsp

import (
	"encoding/base64"
	"fmt"
)

// The 'Basic' HTTP Authentication Scheme
// https://datatracker.ietf.org/doc/html/rfc7617
type AuthBasic struct {
	header string
}

func NewAuthBasic(user, pass, _ string) *AuthBasic {
	token := []byte(user + ":" + pass)
	header := fmt.Sprintf(
		"Basic %s",
		base64.StdEncoding.EncodeToString(token),
	)

	return &AuthBasic{header}
}

func (a *AuthBasic) Header(method, uri string) string {
	return a.header
}
