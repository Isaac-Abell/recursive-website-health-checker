// types.go - Core Data Structures
//
// Defines the data types used throughout the crawler:
// - Task: represents a URL to crawl with its depth
// - Result: captures probe results including latency breakdown
// - Latencies: timing data for DNS, TCP, and TLS phases
// - ProbeConfig: configuration for HTTP probing with retries

package main

import (
	"time"
)

// Task represents a single crawl job.
type Task struct {
	URL   string
	Depth int
}

// Result captures the outcome of probing a URL.
type Result struct {
	URL        string
	StatusCode int
	FoundLinks []string
	Err        error
	Depth      int
	DNS        time.Duration
	TCP        time.Duration
	TLS        time.Duration
	Total      time.Duration
}

// Latencies holds timing breakdown for connection phases.
type Latencies struct {
	DNS time.Duration
	TCP time.Duration
	TLS time.Duration
}

// ProbeConfig controls retry and timeout behavior.
type ProbeConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	Timeout    time.Duration
}
