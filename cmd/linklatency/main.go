package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ngonzalez/web-tools/internal/latency"
)

const (
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorBold  = "\033[1m"
)

func main() {
	concurrency := flag.Int("c", 10, "Number of concurrent requests")
	flag.IntVar(concurrency, "concurrency", 10, "Number of concurrent requests")

	timeout := flag.Int("t", 30, "Request timeout in seconds")
	flag.IntVar(timeout, "timeout", 30, "Request timeout in seconds")

	maxDepth := flag.Int("d", 0, "Maximum crawl depth (0 = unlimited)")
	flag.IntVar(maxDepth, "depth", 0, "Maximum crawl depth (0 = unlimited)")

	verbose := flag.Bool("v", false, "Show progress while crawling")
	flag.BoolVar(verbose, "verbose", false, "Show progress while crawling")

	barWidth := flag.Int("w", 30, "Width of the bar graph")
	flag.IntVar(barWidth, "width", 30, "Width of the bar graph")

	showSize := flag.Bool("s", false, "Show page sizes")
	flag.BoolVar(showSize, "size", false, "Show page sizes")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s%sLinkLatency%s - Measure page load times\n\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: linklatency [options] <url>\n\n")
		fmt.Fprintf(os.Stderr, "Crawls a website and measures the latency of each page,\n")
		fmt.Fprintf(os.Stderr, "displaying results as a bar graph sorted by load time.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --concurrency int   Number of concurrent requests (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -t, --timeout int       Request timeout in seconds (default 30)\n")
		fmt.Fprintf(os.Stderr, "  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose           Show progress while crawling\n")
		fmt.Fprintf(os.Stderr, "  -w, --width int         Width of the bar graph (default 30)\n")
		fmt.Fprintf(os.Stderr, "  -s, --size              Show page sizes\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  linklatency https://example.com\n")
		fmt.Fprintf(os.Stderr, "  linklatency -c 5 -d 2 -s https://example.com\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	startURL := args[0]

	config := latency.Config{
		Concurrency: *concurrency,
		Timeout:     time.Duration(*timeout) * time.Second,
		MaxDepth:    *maxDepth,
		Verbose:     *verbose,
	}

	fmt.Printf("%s%sLinkLatency%s starting...\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Target: %s\n", startURL)
	fmt.Printf("Concurrency: %d, Timeout: %ds, Max Depth: %d\n\n", config.Concurrency, *timeout, config.MaxDepth)

	m := latency.New(config)
	result, err := m.Measure(startURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result.PrintSummary(*barWidth, *showSize)
}
