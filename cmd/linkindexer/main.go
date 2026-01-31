package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ngonzalez/web-tools/internal/indexer"
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

	noRobots := flag.Bool("no-robots", false, "Skip robots.txt checking")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s%sLinkIndexer%s - Detect non-indexable links\n\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: linkindexer [options] <url>\n\n")
		fmt.Fprintf(os.Stderr, "Detects links that won't be indexed by search engines:\n")
		fmt.Fprintf(os.Stderr, "  - Links with rel=\"nofollow\"\n")
		fmt.Fprintf(os.Stderr, "  - Links with rel=\"sponsored\" or rel=\"ugc\"\n")
		fmt.Fprintf(os.Stderr, "  - Pages with <meta name=\"robots\" content=\"noindex\">\n")
		fmt.Fprintf(os.Stderr, "  - Pages with X-Robots-Tag: noindex header\n")
		fmt.Fprintf(os.Stderr, "  - URLs blocked by robots.txt\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --concurrency int   Number of concurrent requests (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -t, --timeout int       Request timeout in seconds (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose           Show all visited URLs\n")
		fmt.Fprintf(os.Stderr, "  -D, --details           Show detailed breakdown (default true)\n")
		fmt.Fprintf(os.Stderr, "      --no-robots         Skip robots.txt checking\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  linkindexer https://example.com\n")
		fmt.Fprintf(os.Stderr, "  linkindexer -c 20 -d 3 -v https://example.com\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	startURL := args[0]

	config := indexer.Config{
		Concurrency:    *concurrency,
		Timeout:        time.Duration(*timeout) * time.Second,
		MaxDepth:       *maxDepth,
		Verbose:        *verbose,
		CheckRobotsTxt: !*noRobots,
	}

	fmt.Printf("%s%sLinkIndexer%s starting...\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Target: %s\n", startURL)
	fmt.Printf("Concurrency: %d, Timeout: %ds, Max Depth: %d\n", config.Concurrency, *timeout, config.MaxDepth)
	if config.CheckRobotsTxt {
		fmt.Printf("Checking robots.txt: yes\n")
	}
	fmt.Println()

	idx := indexer.New(config)
	result, err := idx.Analyze(startURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result.PrintSummary(*details)
}
