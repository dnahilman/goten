package internal

import (
	"net"
	"net/http"
	"strings"
)

// GetClientIP resolves the client IP for a request. The result is recorded as
// the session's ip_address (audit/display only) — it is never used as a security
// control, so a spoofed value cannot bypass authentication.
//
// Resolution order:
//  1. ipHeader, if non-empty (a header you fully control at the edge).
//  2. X-Forwarded-For, X-Real-IP, CF-Connecting-IP — the left-most value.
//  3. r.RemoteAddr (the TCP peer).
//
// SPOOFING NOTE: the left-most X-Forwarded-For entry is the value the *client*
// sent and is only trustworthy if your reverse proxy OVERWRITES the header with
// the real peer address rather than appending to it. With nginx, use:
//
//	proxy_set_header X-Forwarded-For $remote_addr;   # overwrite (safe)
//
// and NOT $proxy_add_x_forwarded_for (which appends, letting a client prepend a
// forged IP that ends up left-most). If you cannot guarantee an overwriting
// proxy, pass a header you set yourself via the ipHeader argument.
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
