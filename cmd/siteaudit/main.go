package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ngonzalez/web-tools/internal/audit"
)

const (
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorBold  = "\033[1m"
)

func main() {
	concurrency := flag.Int("c", 10, "Number of concurrent requests")
	flag.IntVar(concurrency, "concurrency", 10, "Number of concurrent requests")

	timeout := flag.Int("t", 15, "Request timeout in seconds")
	flag.IntVar(timeout, "timeout", 15, "Request timeout in seconds")

	maxDepth := flag.Int("d", 0, "Maximum crawl depth (0 = unlimited)")
	flag.IntVar(maxDepth, "depth", 0, "Maximum crawl depth (0 = unlimited)")

	verbose := flag.Bool("v", false, "Show detailed progress")
	flag.BoolVar(verbose, "verbose", false, "Show detailed progress")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s%sSiteAudit%s - Complete SEO audit tool\n\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: siteaudit [options] <url>\n\n")
		fmt.Fprintf(os.Stderr, "Performs a comprehensive audit of your website including:\n")
		fmt.Fprintf(os.Stderr, "  • Broken links detection (404 errors)\n")
		fmt.Fprintf(os.Stderr, "  • Non-analyzable links (external, files, mailto, etc.)\n")
		fmt.Fprintf(os.Stderr, "  • Indexability issues (nofollow, noindex, robots.txt)\n")
		fmt.Fprintf(os.Stderr, "  • Canonical URL verification\n")
		fmt.Fprintf(os.Stderr, "  • Performance measurement (page latency)\n")
		fmt.Fprintf(os.Stderr, "  • SEO analysis (title, description, OG tags, schema)\n")
		fmt.Fprintf(os.Stderr, "  • PageRank calculation (internal link structure)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --concurrency int   Number of concurrent requests (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -t, --timeout int       Request timeout in seconds (default 15)\n")
		fmt.Fprintf(os.Stderr, "  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose           Show detailed progress\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  siteaudit https://example.com\n")
		fmt.Fprintf(os.Stderr, "  siteaudit -d 3 -v https://example.com\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	targetURL := args[0]

	config := audit.Config{
		Concurrency: *concurrency,
		Timeout:     time.Duration(*timeout) * time.Second,
		MaxDepth:    *maxDepth,
		Verbose:     *verbose,
	}

	fmt.Printf("\n%s%s╔══════════════════════════════════════════════════════════════════════════════╗%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s║                              SITE AUDIT                                       ║%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════════════════════════╝%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("\nTarget: %s\n", targetURL)
	fmt.Printf("Config: concurrency=%d, timeout=%ds, depth=%d\n", config.Concurrency, *timeout, config.MaxDepth)

	auditor := audit.New(config)
	result, err := auditor.Run(targetURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	result.PrintReport()

	// Exit code based on score
	if result.OverallScore < 50 {
		os.Exit(2)
	}
	if result.OverallScore < 70 {
		os.Exit(1)
	}
}
