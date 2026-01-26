// policy.go - URL Validation Policy
//
// Centralizes the business logic for determining which URLs are
// eligible for crawling. This makes it easy to expand or modify
// the crawl scope in the future.

package main

import (
	"net/url"
	"strings"
)

// validateURL checks if a URL should be crawled based on domain rules.
func validateURL(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	host := strings.ToLower(u.Hostname())

	return host == "isaacabell.com" || host == "www.isaacabell.com" || host == "isaac-abell.github.io" || host == "www.isaac-abell.github.io"
}
