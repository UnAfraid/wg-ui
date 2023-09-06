package api

import (
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
)

func checkOrigin(r *http.Request, allowedSubscriptionOrigins []string) bool {
	origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}

	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}

	for _, allowedHost := range allowedSubscriptionOrigins {
		allowedHost = strings.TrimSpace(allowedHost)
		if allowedHost == "*" {
			return true
		}
		if equalASCIIFold(u.Host, allowedHost) {
			return true
		}
	}

	return equalASCIIFold(u.Host, r.Host)
}

func equalASCIIFold(s, t string) bool {
	for s != "" && t != "" {
		sr, size := utf8.DecodeRuneInString(s)
		s = s[size:]
		tr, size := utf8.DecodeRuneInString(t)
		t = t[size:]
		if sr == tr {
			continue
		}
		if 'A' <= sr && sr <= 'Z' {
			sr = sr + 'a' - 'A'
		}
		if 'A' <= tr && tr <= 'Z' {
			tr = tr + 'a' - 'A'
		}
		if sr != tr {
			return false
		}
	}
	return s == t
}
