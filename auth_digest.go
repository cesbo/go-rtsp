package rtsp

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// HTTP Digest Access Authentication
// https://datatracker.ietf.org/doc/html/rfc7616
type AuthDigest struct {
	login    string
	password string
	nonce    string
	opaque   string
	realm    string
}

func NewAuthDigest(login, password, header string) *AuthDigest {
	a := &AuthDigest{
		login:    login,
		password: password,
	}

	h := strings.Split(header, ",")

	for _, v := range h {
		v = strings.TrimSpace(v)

		if name, value, ok := strings.Cut(v, "="); ok {
			switch name {
			case "nonce":
				a.nonce = strings.Trim(value, "\"")
			case "opaque":
				a.opaque = strings.Trim(value, "\"")
			case "realm":
				a.realm = strings.Trim(value, "\"")
			}
		}
	}

	return a
}

func (a *AuthDigest) Header(method, uri string) string {
	if a.nonce == "" {
		return ""
	}

	ha1 := md5.New()
	fmt.Fprintf(ha1, "%s:%s:%s", a.login, a.realm, a.password)
	sum1 := ha1.Sum(nil)

	ha2 := md5.New()
	fmt.Fprintf(ha2, "%s:%s", method, uri)
	sum2 := ha2.Sum(nil)

	ha3 := md5.New()
	fmt.Fprintf(ha3, "%x:%s:%x", sum1, a.nonce, sum2)
	resp := ha3.Sum(nil)

	var sb strings.Builder

	_, _ = fmt.Fprintf(
		&sb,
		`Digest username="%s", uri="%s", realm="%s", nonce="%s", response="%x"`,
		a.login,
		uri,
		a.realm,
		a.nonce,
		resp,
	)

	if a.opaque != "" {
		_, _ = fmt.Fprintf(&sb, `, opaque="%s"`, a.opaque)
	}

	return sb.String()
}
