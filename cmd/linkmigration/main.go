package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ngonzalez/web-tools/internal/migration"
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

	verbose := flag.Bool("v", false, "Show progress for each URL checked")
	flag.BoolVar(verbose, "verbose", false, "Show progress for each URL checked")

	useGET := flag.Bool("g", false, "Use GET requests instead of HEAD for checking")
	flag.BoolVar(useGET, "get", false, "Use GET requests instead of HEAD for checking")

	csvOutput := flag.Bool("csv", false, "Output lost links as CSV")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s%sLinkMigration%s - Detect lost links after site migration\n\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: linkmigration [options] <old-site-url> <new-site-url>\n\n")
		fmt.Fprintf(os.Stderr, "This tool crawls the old site to collect all URLs, then checks if each\n")
		fmt.Fprintf(os.Stderr, "URL is available on the new site (by mapping the domain).\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --concurrency int   Number of concurrent requests (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -t, --timeout int       Request timeout in seconds (default 10)\n")
		fmt.Fprintf(os.Stderr, "  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose           Show progress for each URL checked\n")
		fmt.Fprintf(os.Stderr, "  -g, --get               Use GET requests instead of HEAD for checking\n")
		fmt.Fprintf(os.Stderr, "      --csv               Output lost links as CSV format\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  linkmigration https://old-site.com https://new-site.com\n")
		fmt.Fprintf(os.Stderr, "  linkmigration -c 20 -d 3 -v https://old.example.com https://new.example.com\n")
		fmt.Fprintf(os.Stderr, "  linkmigration --csv https://old-site.com https://new-site.com > lost-links.csv\n")
	}

	flag.Parse()

	// Check for URL arguments
	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	oldSiteURL := args[0]
	newSiteURL := args[1]

	// Configure migrator
	config := migration.Config{
		Concurrency: *concurrency,
		Timeout:     time.Duration(*timeout) * time.Second,
		MaxDepth:    *maxDepth,
		Verbose:     *verbose,
		UseHEAD:     !*useGET,
	}

	if !*csvOutput {
		fmt.Printf("%s%sLinkMigration%s starting...\n", colorBold, colorCyan, colorReset)
		fmt.Printf("Old site: %s\n", oldSiteURL)
		fmt.Printf("New site: %s\n", newSiteURL)
		fmt.Printf("Concurrency: %d, Timeout: %ds, Max Depth: %d\n", config.Concurrency, *timeout, config.MaxDepth)
	}

	// Create and run migrator
	m := migration.New(config)
	result, err := m.Check(oldSiteURL, newSiteURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print results
	if *csvOutput {
		fmt.Print(result.ExportCSV())
	} else {
		result.PrintSummary()
	}

	// Exit with error code if lost links found
	if len(result.LostLinks) > 0 {
		os.Exit(1)
	}
}
