package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ngonzalez/web-tools/internal/canonical"
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

	verbose := flag.Bool("v", false, "Show all visited URLs")
	flag.BoolVar(verbose, "verbose", false, "Show all visited URLs")

	details := flag.Bool("details", true, "Show detailed breakdown")
	flag.BoolVar(details, "D", true, "Show detailed breakdown")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s%sLinkCanonical%s - Verify canonical URLs\n\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: linkcanonical [options] <url>\n\n")
		fmt.Fprintf(os.Stderr, "Crawls a website and verifies that all internal links\n")
		fmt.Fprintf(os.Stderr, "point to canonical URLs.\n\n")
		fmt.Fprintf(os.Stderr, "Detects:\n")
		fmt.Fprintf(os.Stderr, "  - Links pointing to non-canonical URLs\n")
		fmt.Fprintf(os.Stderr, "  - Links causing redirects to canonical\n")
		fmt.Fprintf(os.Stderr, "  - Pages with missing canonical tags\n")
		fmt.Fprintf(os.Stderr, "  - Canonical URL mismatches\n")
		fmt.Fprintf(os.Stderr, "  - Canonical chains (A→B→C)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --concurrency int   Number of concurrent requests (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -t, --timeout int       Request timeout in seconds (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose           Show all visited URLs\n")
		fmt.Fprintf(os.Stderr, "  -D, --details           Show detailed breakdown (default true)\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  linkcanonical https://example.com\n")
		fmt.Fprintf(os.Stderr, "  linkcanonical -c 20 -d 3 -v https://example.com\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	startURL := args[0]

	config := canonical.Config{
		Concurrency: *concurrency,
		Timeout:     time.Duration(*timeout) * time.Second,
		MaxDepth:    *maxDepth,
		Verbose:     *verbose,
	}

	fmt.Printf("%s%sLinkCanonical%s starting...\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Target: %s\n", startURL)
	fmt.Printf("Concurrency: %d, Timeout: %ds, Max Depth: %d\n\n", config.Concurrency, *timeout, config.MaxDepth)

	checker := canonical.New(config)
	result, err := checker.Check(startURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result.PrintSummary(*details)

	// Exit with error code if issues found
	if len(result.Issues) > 0 {
		os.Exit(1)
	}
}
