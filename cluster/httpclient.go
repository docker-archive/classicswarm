package cluster

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// NewHTTPClientTimeout is used to create the HTTP Client and URL
func NewHTTPClientTimeout(daemonURL string, tlsConfig *tls.Config, timeout time.Duration) (*http.Client, *url.URL, error) {
	u, err := url.Parse(daemonURL)
	if err != nil {
		return nil, nil, err
	}
	if u.Scheme == "" || u.Scheme == "tcp" {
		if tlsConfig == nil {
			u.Scheme = "http"
		} else {
			u.Scheme = "https"
		}
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: timeout,
	}
	return httpClient, u, nil
}
