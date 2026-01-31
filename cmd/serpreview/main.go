package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ngonzalez/web-tools/internal/serp"
)

const (
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorBold  = "\033[1m"
)

func main() {
	timeout := flag.Int("t", 30, "Request timeout in seconds")
	flag.IntVar(timeout, "timeout", 30, "Request timeout in seconds")

	verbose := flag.Bool("v", false, "Verbose output")
	flag.BoolVar(verbose, "verbose", false, "Verbose output")

	analysisOnly := flag.Bool("a", false, "Show analysis only (no preview)")
	flag.BoolVar(analysisOnly, "analysis", false, "Show analysis only (no preview)")

	previewOnly := flag.Bool("p", false, "Show preview only (no analysis)")
	flag.BoolVar(previewOnly, "preview", false, "Show preview only (no analysis)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s%sSERPreview%s - See how your page appears on Google\n\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: serpreview [options] <url>\n\n")
		fmt.Fprintf(os.Stderr, "Analyzes a page's SEO metadata and shows:\n")
		fmt.Fprintf(os.Stderr, "  - Google search result preview (SERP snippet)\n")
		fmt.Fprintf(os.Stderr, "  - Title and meta description analysis\n")
		fmt.Fprintf(os.Stderr, "  - Open Graph and Twitter Card tags\n")
		fmt.Fprintf(os.Stderr, "  - Canonical URL and robots directives\n")
		fmt.Fprintf(os.Stderr, "  - Schema.org structured data\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -t, --timeout int   Request timeout in seconds (default 30)\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose       Verbose output\n")
		fmt.Fprintf(os.Stderr, "  -a, --analysis      Show analysis only (no preview)\n")
		fmt.Fprintf(os.Stderr, "  -p, --preview       Show preview only (no analysis)\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  serpreview https://example.com\n")
		fmt.Fprintf(os.Stderr, "  serpreview -a example.com\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	targetURL := args[0]

	config := serp.Config{
		Timeout: time.Duration(*timeout) * time.Second,
		Verbose: *verbose,
	}

	fetcher := serp.New(config)
	meta, err := fetcher.Analyze(targetURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Show preview unless analysis-only mode
	if !*analysisOnly {
		preview := meta.GeneratePreview()
		preview.PrintGooglePreview()
	}

	// Show analysis unless preview-only mode
	if !*previewOnly {
		meta.PrintMetaAnalysis()
	}
}
