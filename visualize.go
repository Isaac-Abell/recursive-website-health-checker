// visualize.go - Crawl Report Generator
//
// Generates a Markdown report with:
// 1. A Mermaid network graph showing page dependencies
// 2. A detailed table of all requests with latency breakdown

package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

func visualizeNetwork(w io.Writer, registry map[string]Result, keys []string, urlToID map[string]int) {
	fmt.Fprintln(w, "## 1. Network Graph (Coalesced)")
	fmt.Fprintln(w, "Dependencies found inside `.js` files are shown as direct links from the parent page.")
	fmt.Fprintln(w, "\n```mermaid")
	fmt.Fprintln(w, "graph LR")

	// Styles
	fmt.Fprintln(w, "    classDef success fill:#aaffaa,stroke:#006600,stroke-width:2px,color:black;")
	fmt.Fprintln(w, "    classDef error fill:#ffaaaa,stroke:#990000,stroke-width:2px,stroke-dasharray: 5 5,color:black;")
	fmt.Fprintln(w, "    classDef warn fill:#ffeba1,stroke:#9e7d02,stroke-width:2px,color:black;")

	// A. Nodes (Filter out JS files)
	for _, url := range keys {
		if isJS(url) {
			continue
		}

		res := registry[url]
		id := getID(url, urlToID)

		styleClass := "success"
		label := fmt.Sprintf("%s<br/>(%dms)", url, res.Total.Milliseconds())

		if res.Err != nil || res.StatusCode >= 400 {
			styleClass = "error"
			label = fmt.Sprintf("%s<br/>FAIL", url)
		} else if res.StatusCode >= 300 {
			styleClass = "warn"
		}

		fmt.Fprintf(w, "    %s[\"%s\"]:::%s\n", id, label, styleClass)
	}

	fmt.Fprintln(w, "")
	graphEdges := buildCoalescedGraph(registry, keys)

	parents := make([]string, 0, len(graphEdges))
	for p := range graphEdges {
		parents = append(parents, p)
	}
	sort.Strings(parents)

	for _, parent := range parents {
		parentID := getID(parent, urlToID)
		children := graphEdges[parent]

		for _, child := range children {
			if childID := getID(child, urlToID); childID != "unknown" {
				fmt.Fprintf(w, "    %s --> %s\n", parentID, childID)
			}
		}
	}
	fmt.Fprintln(w, "```")
}

func visualizeTable(w io.Writer, registry map[string]Result, keys []string) {
	fmt.Fprintln(w, "\n## 2. Request Details")
	fmt.Fprintln(w, "| Ind | URL | Status | Depth | DNS | TCP | TLS | Total | Error |")
	fmt.Fprintln(w, "|:---:|---|---|:---:|---|---|---|---|---|")

	for _, url := range keys {
		r := registry[url]

		icon := "🟢"
		statusDisplay := fmt.Sprintf("%d", r.StatusCode)

		if r.Err != nil || r.StatusCode >= 400 {
			icon = "🔴"
			statusDisplay = fmt.Sprintf("**%d**", r.StatusCode)
		} else if r.StatusCode >= 300 {
			icon = "🟡"
		}

		urlText := fmt.Sprintf("`%s`", url)
		if isJS(url) {
			urlText = fmt.Sprintf("*%s*", url)
		}

		errStr := ""
		if r.Err != nil {
			errStr = fmt.Sprintf("`%s`", r.Err.Error())
		}

		fmt.Fprintf(w, "| %s | %s | %s | %d | %v | %v | %v | **%v** | %s |\n",
			icon, urlText, statusDisplay, r.Depth, r.DNS, r.TCP, r.TLS, r.Total, errStr)
	}
}

func Visualize(registry map[string]Result) error {
	f, err := os.Create("crawl_report.md")
	if err != nil {
		return err
	}
	defer f.Close()

	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		rI, rJ := registry[keys[i]], registry[keys[j]]
		if rI.Depth != rJ.Depth {
			return rI.Depth < rJ.Depth
		}
		return keys[i] < keys[j]
	})

	urlToID := make(map[string]int)
	for i, k := range keys {
		urlToID[k] = i
	}

	fmt.Fprintln(f, "# Crawl Visualization Summary")
	fmt.Fprintf(f, "**Date:** %s\n\n", time.Now().Format(time.RFC1123))

	visualizeNetwork(f, registry, keys, urlToID)
	visualizeTable(f, registry, keys)

	return nil
}
