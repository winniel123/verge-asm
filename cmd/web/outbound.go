package main

import (
	"net/http"
	"time"
)

func newOutboundClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		// A followed 3xx lets a third party pick our next destination, e.g. IMDS (ADR-0196 §1).
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}
