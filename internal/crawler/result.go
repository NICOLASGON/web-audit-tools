package crawler

import (
	"fmt"
	"strings"
)

// BrokenLink represents a broken link found during crawling
type BrokenLink struct {
	SourceURL  string
	BrokenURL  string
	StatusCode int
	Error      string
}

// CrawlResult holds the complete results of a crawl session
type CrawlResult struct {
	StartURL     string
	TotalVisited int
	BrokenLinks  []BrokenLink
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// PrintSummary displays the crawl results in a formatted way
func (r *CrawlResult) PrintSummary() {
	fmt.Println()
	fmt.Printf("%s%s=== Crawl Summary ===%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Start URL: %s%s%s\n", colorBlue, r.StartURL, colorReset)
	fmt.Printf("Total pages visited: %s%d%s\n", colorGreen, r.TotalVisited, colorReset)
	fmt.Println()

	if len(r.BrokenLinks) == 0 {
		fmt.Printf("%s%s✓ No broken links found!%s\n", colorBold, colorGreen, colorReset)
		return
	}

	fmt.Printf("%s%s✗ Found %d broken link(s):%s\n\n", colorBold, colorRed, len(r.BrokenLinks), colorReset)

	for i, link := range r.BrokenLinks {
		fmt.Printf("%s[%d]%s %s%s%s\n", colorYellow, i+1, colorReset, colorRed, link.BrokenURL, colorReset)
		fmt.Printf("    Found on: %s\n", link.SourceURL)
		if link.StatusCode > 0 {
			fmt.Printf("    Status: %s%d%s\n", colorRed, link.StatusCode, colorReset)
		}
		if link.Error != "" {
			fmt.Printf("    Error: %s\n", link.Error)
		}
		fmt.Println()
	}
}

// PrintProgress displays progress information for a visited URL
func PrintProgress(url string, statusCode int, depth int) {
	status := fmt.Sprintf("%d", statusCode)
	var statusColor string

	switch {
	case statusCode >= 200 && statusCode < 300:
		statusColor = colorGreen
	case statusCode >= 300 && statusCode < 400:
		statusColor = colorYellow
	case statusCode >= 400:
		statusColor = colorRed
	default:
		statusColor = colorReset
	}

	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s%s[%s]%s %s\n", indent, statusColor, status, colorReset, url)
}

// PrintError displays an error for a URL
func PrintError(url string, err string, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s%s[ERR]%s %s - %s\n", indent, colorRed, colorReset, url, err)
}
