package rtsp

import (
	"net/url"
	"strings"
)

type Auth interface {
	// Header updates token and returns value for Authorization header.
	Header(method, uri string) string
}

func NewAuth(u *url.URL, header string) Auth {
	login := u.User.Username()
	password, ok := u.User.Password()
	if !ok {
		return nil
	}

	header = strings.TrimSpace(header)

	if method, params, ok := strings.Cut(header, " "); ok {
		switch strings.ToLower(method) {
		case "digest":
			return NewAuthDigest(login, password, params)
		case "basic":
			return NewAuthBasic(login, password, params)
		}
	}

	return nil
}
