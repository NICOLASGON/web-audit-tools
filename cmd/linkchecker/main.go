package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ngonzalez/web-tools/internal/crawler"
)

const (
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorBold  = "\033[1m"
)

func main() {
	// Define flags
	concurrency := flag.Int("c", 10, "Number of concurrent requests")
	flag.IntVar(concurrency, "concurrency", 10, "Number of concurrent requests")

	timeout := flag.Int("t", 10, "Request timeout in seconds")
	flag.IntVar(timeout, "timeout", 10, "Request timeout in seconds")

	maxDepth := flag.Int("d", 0, "Maximum crawl depth (0 = unlimited)")
	flag.IntVar(maxDepth, "depth", 0, "Maximum crawl depth (0 = unlimited)")

	verbose := flag.Bool("v", false, "Show all visited URLs")
	flag.BoolVar(verbose, "verbose", false, "Show all visited URLs")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s%sLinkChecker%s - A broken link detector\n\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: linkchecker [options] <url>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --concurrency int   Number of concurrent requests (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -t, --timeout int       Request timeout in seconds (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose           Show all visited URLs\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  linkchecker https://example.com\n")
		fmt.Fprintf(os.Stderr, "  linkchecker -c 20 -t 5 -d 3 -v https://example.com\n")
	}

	flag.Parse()

	// Check for URL argument
	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	startURL := args[0]

	// Configure crawler
	config := crawler.Config{
		Concurrency: *concurrency,
		Timeout:     time.Duration(*timeout) * time.Second,
		MaxDepth:    *maxDepth,
		Verbose:     *verbose,
	}

	fmt.Printf("%s%sLinkChecker%s starting...\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Target: %s\n", startURL)
	fmt.Printf("Concurrency: %d, Timeout: %ds, Max Depth: %d\n\n", config.Concurrency, *timeout, config.MaxDepth)

	// Create and run crawler
	c := crawler.New(config)
	result, err := c.Crawl(startURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print results
	result.PrintSummary()

	// Exit with error code if broken links found
	if len(result.BrokenLinks) > 0 {
		os.Exit(1)
	}
}
