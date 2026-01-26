// scrapePage.go - Link Extraction Engine
//
// Provides two strategies for extracting links from web content:
// 1. Browser-based: Uses headless Chrome for JavaScript-rendered pages
// 2. Regex-based: Fast pattern matching for static content (JS/JSON files)

package main

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/chromedp/chromedp"
)

// Regex to find http/https URLs in any text
var urlRegex = regexp.MustCompile(`https?://[^ "'<>\n\t\r();]+`)

// scrapeWithRegex is the "brute force" scanner for JS files or HTML source
func scrapeWithRegex(content string) []string {
	return urlRegex.FindAllString(content, -1)
}

// scrapeWithBrowser handles complex HTML pages
func ScrapeWithBrowser(urlStr string) []string {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-sandbox", true),
        chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.WindowSize(1920, 1080),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var pageSource string
	var domLinks []string

	err := chromedp.Run(ctx,
		chromedp.Navigate(urlStr),
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(jsScrollTrigger, nil),
		chromedp.Poll("window.isScrollDone === true", nil),
		chromedp.Evaluate(jsExtractLinks, &domLinks),
		chromedp.OuterHTML("html", &pageSource),
	)

	if err != nil {
		fmt.Printf("Browser Error on %s: %v\n", urlStr, err)
		return nil
	}

	regexLinks := scrapeWithRegex(pageSource)
	return append(domLinks, regexLinks...)
}

// --- JAVASCRIPT PAYLOADS ---
const jsScrollTrigger = `
(async () => {
    window.isScrollDone = false;
    await new Promise((resolve) => {
        let totalHeight = 0;
        let distance = 100;
        let timer = setInterval(() => {
            window.scrollBy(0, distance);
            totalHeight += distance;
            let currentPos = window.scrollY + window.innerHeight;
            // Stop if at bottom or safety limit (20k pixels)
            if(currentPos >= document.body.scrollHeight || totalHeight > 20000){
                clearInterval(timer);
                resolve();
            }
        }, 100);
    });
    window.isScrollDone = true;
})()
`

const jsExtractLinks = `
(() => {
    const urls = new Set();
    document.querySelectorAll('a[href], [src]').forEach(el => {
        const val = el.getAttribute('href') || el.getAttribute('src');
        if (val) {
            try {
                const abs = new URL(val, window.location.href).href; 
                if (abs.startsWith('http')) urls.add(abs);
            } catch(e) {}
        }
    });
    return Array.from(urls);
})()
`
