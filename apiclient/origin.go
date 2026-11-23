package apiclient

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ValidateNodeURL restricts credential-bearing requests to HTTPS or loopback HTTP.
func ValidateNodeURL(endpoint string) (*url.URL, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("invalid node URL")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	local := strings.EqualFold(host, "localhost") || ip != nil && ip.IsLoopback()
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return nil, errors.New("remote node connections require HTTPS")
	}
	return u, nil
}

// NewNodeClient keeps configured node requests direct and rejects redirects.
func NewNodeClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &http.Client{
		Transport: transport, Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}
