// generateGraph.go - Graph Coalescing Algorithm
//
// This file implements the logic to build a "coalesced" dependency graph.
// JavaScript files are treated as intermediate nodes - their discovered links
// are attributed directly to the parent HTML page that referenced them.
// This produces a cleaner visualization where JS internals are hidden.

package main

import (
	"fmt"
	"sort"
	"strings"
)

// getID returns a stable node identifier for Mermaid graph rendering.
func getID(url string, lookup map[string]int) string {
	if id, ok := lookup[url]; ok {
		return fmt.Sprintf("node%d", id)
	}
	return "unknown"
}

func isJS(u string) bool {
	clean := u
	if idx := strings.Index(clean, "?"); idx != -1 {
		clean = clean[:idx]
	}
	return strings.HasSuffix(clean, ".js")
}

// Helper: The Optimized Graph Builder
func buildCoalescedGraph(registry map[string]Result, keys []string) map[string][]string {
	adjList := make(map[string][]string)

	// Memoization Table: map[JS_URL] -> Set_of_Real_Children
	// We cache the *Set* to make merging fast.
	memo := make(map[string]map[string]bool)

	// Cycle Prevention: map[JS_URL] -> currently_visiting
	visiting := make(map[string]bool)

	// Forward declaration for recursion
	var resolve func(u string) map[string]bool

	resolve = func(u string) map[string]bool {
		// Base Case: If already computed, return cached result
		if cached, ok := memo[u]; ok {
			return cached
		}

		// Cycle Detection: If we are already visiting this node in the current stack,
		// return empty to break the cycle.
		if visiting[u] {
			return nil
		}

		// Mark as visiting
		visiting[u] = true

		// Recursive Step
		results := make(map[string]bool)

		if res, exists := registry[u]; exists {
			for _, child := range res.FoundLinks {
				if isJS(child) {
					childResults := resolve(child)
					for realLink := range childResults {
						results[realLink] = true
					}
				} else {
					results[child] = true
				}
			}
		}

		// Backtrack & Memoize
		visiting[u] = false
		memo[u] = results
		return results
	}

	for _, parentURL := range keys {
		if isJS(parentURL) {
			continue
		}

		finalDestinations := make(map[string]bool)

		if res, exists := registry[parentURL]; exists {
			for _, link := range res.FoundLinks {
				if isJS(link) {
					resolved := resolve(link)
					for r := range resolved {
						finalDestinations[r] = true
					}
				} else {
					finalDestinations[link] = true
				}
			}
		}

		if len(finalDestinations) > 0 {
			dests := make([]string, 0, len(finalDestinations))
			for d := range finalDestinations {
				dests = append(dests, d)
			}
			sort.Strings(dests)
			adjList[parentURL] = dests
		}
	}

	return adjList
}
