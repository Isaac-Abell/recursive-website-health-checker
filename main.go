// main.go - Entry Point & Crawl Coordinator
//
// Implements a concurrent web crawler using a worker pool pattern.
// Workers pull tasks from a channel and submit results back.
// The main loop manages visited URLs and dispatches new tasks.

package main

import (
	"flag"
	"log/slog"
)

func main() {
	tasks := make(chan Task, 100)
	results := make(chan Result, 100)
	visited := make(map[string]bool)
	registry := make(map[string]Result)
	pending := 0

	initialURL := flag.String("url", "https://isaacabell.com", "URL to crawl")
	workers := flag.Int("workers", 5, "Number of workers")
	flag.Parse()

	// Spawn worker pool
	for i := 0; i < *workers; i++ {
		go func() {
			for t := range tasks {
				results <- Scout(t.URL, t.Depth)
			}
		}()
	}

	// Seed the crawl
	visited[*initialURL] = true
	pending++
	tasks <- Task{URL: *initialURL, Depth: 0}

	totalLinksFound := 0

	// Main dispatch loop
	for pending > 0 {
		res := <-results
		pending--
		registry[res.URL] = res

		totalLinksFound += len(res.FoundLinks)
		slog.Info("crawl progress", "pending", pending, "found", totalLinksFound)

		for _, link := range res.FoundLinks {
			if !visited[link] {
				visited[link] = true
				pending++
				go func(l string, d int) {
					tasks <- Task{URL: l, Depth: d}
				}(link, res.Depth+1)
			}
		}
	}

	close(tasks)
	close(results)

	slog.Info("Crawl complete.")
	Visualize(registry)
}
