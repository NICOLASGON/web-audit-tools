package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ngonzalez/web-tools/internal/pagerank"
)

const (
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorBold  = "\033[1m"
)

func main() {
	concurrency := flag.Int("c", 10, "Number of concurrent requests")
	flag.IntVar(concurrency, "concurrency", 10, "Number of concurrent requests")

	timeout := flag.Int("t", 10, "Request timeout in seconds")
	flag.IntVar(timeout, "timeout", 10, "Request timeout in seconds")

	maxDepth := flag.Int("d", 0, "Maximum crawl depth (0 = unlimited)")
	flag.IntVar(maxDepth, "depth", 0, "Maximum crawl depth (0 = unlimited)")

	verbose := flag.Bool("v", false, "Show crawl progress")
	flag.BoolVar(verbose, "verbose", false, "Show crawl progress")

	topN := flag.Int("n", 20, "Number of top pages to display")
	flag.IntVar(topN, "top", 20, "Number of top pages to display")

	damping := flag.Float64("damping", 0.85, "Damping factor (0-1)")

	maxIter := flag.Int("iter", 100, "Maximum PageRank iterations")

	barWidth := flag.Int("w", 20, "Width of bar graph")
	flag.IntVar(barWidth, "width", 20, "Width of bar graph")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s%sPageRank%s - Calculate page importance\n\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: pagerank [options] <url>\n\n")
		fmt.Fprintf(os.Stderr, "Crawls a website and calculates the PageRank score\n")
		fmt.Fprintf(os.Stderr, "for each page based on internal link structure.\n\n")
		fmt.Fprintf(os.Stderr, "The PageRank algorithm measures page importance based on:\n")
		fmt.Fprintf(os.Stderr, "  - Number of incoming links\n")
		fmt.Fprintf(os.Stderr, "  - Quality of linking pages (their PageRank)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --concurrency int   Number of concurrent requests (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -t, --timeout int       Request timeout in seconds (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose           Show crawl progress\n")
		fmt.Fprintf(os.Stderr, "  -n, --top int           Number of top pages to display (default 20)\n")
		fmt.Fprintf(os.Stderr, "      --damping float     Damping factor 0-1 (default 0.85)\n")
		fmt.Fprintf(os.Stderr, "      --iter int          Maximum iterations (default 100)\n")
		fmt.Fprintf(os.Stderr, "  -w, --width int         Bar graph width (default 20)\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  pagerank https://example.com\n")
		fmt.Fprintf(os.Stderr, "  pagerank -n 50 -d 3 https://example.com\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	startURL := args[0]

	config := pagerank.Config{
		Concurrency:   *concurrency,
		Timeout:       time.Duration(*timeout) * time.Second,
		MaxDepth:      *maxDepth,
		Verbose:       *verbose,
		DampingFactor: *damping,
		MaxIterations: *maxIter,
	}

	fmt.Printf("%s%sPageRank%s starting...\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Target: %s\n", startURL)
	fmt.Printf("Concurrency: %d, Timeout: %ds, Max Depth: %d\n", config.Concurrency, *timeout, config.MaxDepth)
	fmt.Printf("Damping: %.2f, Max Iterations: %d\n\n", config.DampingFactor, config.MaxIterations)

	crawler := pagerank.New(config)
	result, err := crawler.Crawl(startURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result.PrintSummary(*topN, *barWidth)
}
