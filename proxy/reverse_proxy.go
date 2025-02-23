package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// ReverseProxy forwards requests to backend servers
func ReverseProxy(target string) http.Handler {
	url, _ := url.Parse(target)

	proxy := httputil.NewSingleHostReverseProxy(url)

	// Modify request before forwarding
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Set("X-Reverse-Proxy", "Go-Nginx-Clone")
		return nil
	}

	return proxy
}
