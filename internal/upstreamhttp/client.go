package upstreamhttp

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func New(responseHeaderTimeout, requestTimeout time.Duration, maxConnsPerHost int) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = responseHeaderTimeout
	if maxConnsPerHost > 0 {
		transport.MaxConnsPerHost = maxConnsPerHost
		transport.MaxIdleConnsPerHost = maxConnsPerHost
		if transport.MaxIdleConns < maxConnsPerHost*4 {
			transport.MaxIdleConns = maxConnsPerHost * 4
		}
	}
	return &http.Client{
		Transport:     transport,
		Timeout:       requestTimeout,
		CheckRedirect: sameOriginRedirect,
	}
}

func sameOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return nil
	}
	initial := via[0].URL
	if !sameOrigin(initial, req.URL) {
		return fmt.Errorf("refusing cross-origin upstream redirect from %s to %s", safeOrigin(initial), safeOrigin(req.URL))
	}
	return nil
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) &&
		strings.EqualFold(a.Hostname(), b.Hostname()) &&
		effectivePort(a) == effectivePort(b)
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	if strings.EqualFold(u.Scheme, "http") {
		return "80"
	}
	return ""
}

func safeOrigin(u *url.URL) string {
	if u == nil {
		return "<invalid>"
	}
	return u.Scheme + "://" + u.Host
}
