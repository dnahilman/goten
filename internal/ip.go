package internal

import (
	"net"
	"net/http"
	"strings"
)

func GetClientIP(r *http.Request, ipHeader string) string {
	if ipHeader != "" {
		if v := r.Header.Get(ipHeader); v != "" {
			return strings.TrimSpace(strings.Split(v, ",")[0])
		}
	}
	for _, h := range []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} {
		if v := r.Header.Get(h); v != "" {
			return strings.TrimSpace(strings.Split(v, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
