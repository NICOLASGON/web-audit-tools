package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ngonzalez/web-tools/internal/analyzer"
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

	details := flag.Bool("details", true, "Show detailed breakdown of non-analyzable links")
	flag.BoolVar(details, "D", true, "Show detailed breakdown of non-analyzable links")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s%sLinkAnalyzer%s - Detect non-analyzable links\n\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: linkanalyzer [options] <url>\n\n")
		fmt.Fprintf(os.Stderr, "Detects and categorizes all links that cannot be crawled:\n")
		fmt.Fprintf(os.Stderr, "  - External links (different domains)\n")
		fmt.Fprintf(os.Stderr, "  - Non-HTTP links (mailto, tel, javascript, etc.)\n")
		fmt.Fprintf(os.Stderr, "  - File links (PDF, images, documents, etc.)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --concurrency int   Number of concurrent requests (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -t, --timeout int       Request timeout in seconds (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose           Show all visited URLs\n")
		fmt.Fprintf(os.Stderr, "  -D, --details           Show detailed breakdown (default true)\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  linkanalyzer https://example.com\n")
		fmt.Fprintf(os.Stderr, "  linkanalyzer -c 20 -d 3 -v https://example.com\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	startURL := args[0]

	config := analyzer.Config{
		Concurrency: *concurrency,
		Timeout:     time.Duration(*timeout) * time.Second,
		MaxDepth:    *maxDepth,
		Verbose:     *verbose,
	}

	fmt.Printf("%s%sLinkAnalyzer%s starting...\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Target: %s\n", startURL)
	fmt.Printf("Concurrency: %d, Timeout: %ds, Max Depth: %d\n\n", config.Concurrency, *timeout, config.MaxDepth)

	a := analyzer.New(config)
	result, err := a.Analyze(startURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result.PrintSummary(*details)
}
