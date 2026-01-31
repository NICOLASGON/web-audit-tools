package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ngonzalez/web-tools/internal/metacheck"
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

	showAll := flag.Bool("a", false, "Show all issues (including short and duplicates)")
	flag.BoolVar(showAll, "all", false, "Show all issues (including short and duplicates)")

	limit := flag.Int("n", 20, "Maximum number of pages to display per category")
	flag.IntVar(limit, "limit", 20, "Maximum number of pages to display per category")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s%sMetaCheck%s - Meta description length checker\n\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: metacheck [options] <url>\n\n")
		fmt.Fprintf(os.Stderr, "Crawls a website and checks meta description lengths.\n")
		fmt.Fprintf(os.Stderr, "Lists pages with descriptions that are too long (>155 chars),\n")
		fmt.Fprintf(os.Stderr, "too short (<70 chars), missing, or duplicated.\n\n")
		fmt.Fprintf(os.Stderr, "Recommended meta description length: 70-155 characters\n")
		fmt.Fprintf(os.Stderr, "Ideal length: 120-155 characters\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --concurrency int   Number of concurrent requests (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -t, --timeout int       Request timeout in seconds (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose           Show crawl progress\n")
		fmt.Fprintf(os.Stderr, "  -a, --all               Show all issues (short, duplicates)\n")
		fmt.Fprintf(os.Stderr, "  -n, --limit int         Max pages per category (default 20)\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  metacheck https://example.com\n")
		fmt.Fprintf(os.Stderr, "  metacheck -a -d 3 https://example.com\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	startURL := args[0]

	config := metacheck.Config{
		Concurrency: *concurrency,
		Timeout:     time.Duration(*timeout) * time.Second,
		MaxDepth:    *maxDepth,
		Verbose:     *verbose,
	}

	fmt.Printf("%s%sMetaCheck%s starting...\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Target: %s\n", startURL)
	fmt.Printf("Concurrency: %d, Timeout: %ds, Max Depth: %d\n\n", config.Concurrency, *timeout, config.MaxDepth)

	checker := metacheck.New(config)
	result, err := checker.Check(startURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result.PrintSummary(*showAll, *limit)

	// Exit code based on issues
	if result.TooLongCount > 0 || result.MissingCount > 0 {
		os.Exit(1)
	}
}
