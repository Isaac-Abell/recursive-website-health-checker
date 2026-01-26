// prober.go - HTTP Probing with Latency Tracing
//
// Implements HTTP requests with detailed latency measurement using
// httptrace. Includes exponential backoff retry logic for resilience.

package main

import (
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
	"time"
)

// DefaultProbeConfig returns standard defaults for probing.
func DefaultProbeConfig() ProbeConfig {
	return ProbeConfig{
		MaxRetries: 3,
		BaseDelay:  500 * time.Millisecond,
		Timeout:    10 * time.Second,
	}
}

func Probe(req *http.Request, config ProbeConfig) (*http.Response, Latencies, error) {
	var finalErr error

	transport := &http.Transport{
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	for i := 0; i <= config.MaxRetries; i++ {
		var timings Latencies
		var dnsStart, tcpStart, tlsStart time.Time

		trace := &httptrace.ClientTrace{
			DNSStart: func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
			DNSDone:  func(_ httptrace.DNSDoneInfo) { timings.DNS = time.Since(dnsStart) },

			ConnectStart: func(_, _ string) { tcpStart = time.Now() },
			ConnectDone: func(_, _ string, err error) {
				if err == nil {
					timings.TCP = time.Since(tcpStart)
				}
			},

			TLSHandshakeStart: func() { tlsStart = time.Now() },
			TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
				if err == nil {
					timings.TLS = time.Since(tlsStart)
				}
			},
		}

		ctx := httptrace.WithClientTrace(req.Context(), trace)
		reqWithTrace := req.WithContext(ctx)

		resp, err := client.Do(reqWithTrace)

		if err == nil {
			return resp, timings, nil
		}

		finalErr = err

		if i == config.MaxRetries {
			break
		}

		sleepDuration := config.BaseDelay * time.Duration(1<<i)

		select {
		case <-time.After(sleepDuration):
			continue
		case <-req.Context().Done():
			return nil, Latencies{}, req.Context().Err()
		}
	}

	return nil, Latencies{}, finalErr
}
