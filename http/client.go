package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

func NewHTTPClient(timeout *time.Duration) *http.Client {
	if timeout != nil {
		t := &http.Transport{
			Dial: (&net.Dialer{
				Timeout: *timeout,
			}).Dial,
			TLSHandshakeTimeout: *timeout,
			MaxIdleConns:        5,
			IdleConnTimeout:     *timeout,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
		}
		return &http.Client{
			Transport: t,
			Timeout:   *timeout,
		}
	}
	return &http.Client{}
}
