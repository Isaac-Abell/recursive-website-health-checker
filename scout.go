/*
Scout: Distributed Network Observability & Discovery Agent

Architecture:
1. Coordinator: Implements a Worker Pool pattern with buffered channels for highly concurrent link processing.
2. Scout (Prober): A hybrid engine using Headless Chrome (for DOM/Hydration) and raw Regex (for static assets/JS bundles).

Current Capabilities:
- Recursively crawls target sites with depth control.
- Handles Client-Side Rendering (CSR) and Infinite Scroll via JS injection.
- "Asset Drilling": Regex-scans non-HTML payloads (JS/JSON) for hidden endpoints.
*/

/*
TODO Roadmap (L3 -> L4 Engineering Goals):

[] Visualization Dashboard:
	- Real-time web dashboard to visualize crawl status (or just a markdown), link graphs, and stats.
	- Objective: Enhance user experience and provide actionable insights.
*/

package main

import (
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"
)

// Helper to determine if we need the heavy browser
func isHTML(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/html")
}

func getWithTrace(urlStr string, timeout time.Duration) (*http.Response, Latencies, error) {
	var l Latencies
	var dnsStart, tcpStart, tlsStart time.Time

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, l, err
	}

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:  func(_ httptrace.DNSDoneInfo) { l.DNS = time.Since(dnsStart) },

		ConnectStart: func(_, _ string) { tcpStart = time.Now() },
		ConnectDone: func(_, _ string, err error) {
			if err == nil {
				l.TCP = time.Since(tcpStart)
			}
		},

		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			if err == nil {
				l.TLS = time.Since(tlsStart)
			}
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	transport := &http.Transport{
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	resp, err := client.Do(req)

	return resp, l, err
}

func Scout(urlStr string, depth int) Result {
	// 1. INITIAL FETCH (We need to check Content-Type before launching Chrome)
	var resp *http.Response
	var timings Latencies

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return Result{URL: urlStr, Err: err, Depth: depth}
	}

	// 2. EXECUTE PROBE
	resp, timings, err = Probe(req, DefaultProbeConfig())

	if err != nil {
		return Result{URL: urlStr, Err: err, Depth: depth}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Returns Result on bad status, including the timings captured so far
		return Result{
			URL:        urlStr,
			StatusCode: resp.StatusCode,
			Depth:      depth,
			DNS:        timings.DNS,
			TCP:        timings.TCP,
			TLS:        timings.TLS,
			Total:      timings.DNS + timings.TCP + timings.TLS,
		}
	}

	contentType := resp.Header.Get("Content-Type")
	var foundLinks []string

	// If it's an HTML page, we use the browser to handle find links.
	// If it's JS/JSON/CSS, just regex the raw text.
	if isHTML(contentType) {
		foundLinks = ScrapeWithBrowser(urlStr)
	} else {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("failed to read body", "url", urlStr, "error", err)
		} else {
			foundLinks = scrapeWithRegex(string(bodyBytes))
		}
	}

	// Deduplicate and validate everything before returning.
	uniqueLinks := make(map[string]bool)
	var validLinks []string

	for _, link := range foundLinks {
		clean := strings.TrimRight(link, `.,;:)}]'"`)

		if !uniqueLinks[clean] && validateURL(clean) {
			uniqueLinks[clean] = true
			validLinks = append(validLinks, clean)
		}
	}

	return Result{
		URL:        urlStr,
		StatusCode: resp.StatusCode,
		FoundLinks: validLinks,
		Depth:      depth,
		DNS:        timings.DNS,
		TCP:        timings.TCP,
		TLS:        timings.TLS,
		Total:      timings.DNS + timings.TCP + timings.TLS,
	}
}
