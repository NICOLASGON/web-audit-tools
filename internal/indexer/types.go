package indexer

import (
	"fmt"
	"sort"
	"strings"
)

// NoIndexReason indicates why a link is not indexable
type NoIndexReason int

const (
	ReasonNoFollow NoIndexReason = iota
	ReasonNoIndex
	ReasonNoIndexHeader
	ReasonSponsored
	ReasonUGC
	ReasonCanonicalMismatch
	ReasonRobotsTxt
)

func (r NoIndexReason) String() string {
	switch r {
	case ReasonNoFollow:
		return "rel=\"nofollow\""
	case ReasonNoIndex:
		return "meta noindex"
	case ReasonNoIndexHeader:
		return "X-Robots-Tag: noindex"
	case ReasonSponsored:
		return "rel=\"sponsored\""
	case ReasonUGC:
		return "rel=\"ugc\""
	case ReasonCanonicalMismatch:
		return "canonical mismatch"
	case ReasonRobotsTxt:
		return "blocked by robots.txt"
	default:
		return "unknown"
	}
}

func (r NoIndexReason) Description() string {
	switch r {
	case ReasonNoFollow:
		return "Link has rel=\"nofollow\" - search engines won't follow this link"
	case ReasonNoIndex:
		return "Page has <meta name=\"robots\" content=\"noindex\"> - page won't be indexed"
	case ReasonNoIndexHeader:
		return "Page has X-Robots-Tag: noindex header - page won't be indexed"
	case ReasonSponsored:
		return "Link has rel=\"sponsored\" - marked as paid/advertisement"
	case ReasonUGC:
		return "Link has rel=\"ugc\" - marked as user-generated content"
	case ReasonCanonicalMismatch:
		return "Page canonical URL differs from actual URL"
	case ReasonRobotsTxt:
		return "URL is blocked by robots.txt"
	default:
		return "Unknown reason"
	}
}

// NonIndexableLink represents a link that won't be indexed
type NonIndexableLink struct {
	URL       string
	SourceURL string
	Reasons   []NoIndexReason
	Details   string // Additional info like canonical URL
}

// IndexerResult holds the analysis results
type IndexerResult struct {
	StartURL           string
	TotalPages         int
	TotalLinks         int
	IndexableLinks     int
	NonIndexableLinks  []NonIndexableLink
	ByReason           map[NoIndexReason][]NonIndexableLink
	RobotsTxtRules     []string
	PagesWithNoIndex   []string
}

// NewIndexerResult creates a new result
func NewIndexerResult(startURL string) *IndexerResult {
	return &IndexerResult{
		StartURL: startURL,
		ByReason: make(map[NoIndexReason][]NonIndexableLink),
	}
}

// AddNonIndexable adds a non-indexable link
func (r *IndexerResult) AddNonIndexable(link NonIndexableLink) {
	r.NonIndexableLinks = append(r.NonIndexableLinks, link)
	for _, reason := range link.Reasons {
		r.ByReason[reason] = append(r.ByReason[reason], link)
	}
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// PrintSummary displays the results
func (r *IndexerResult) PrintSummary(showDetails bool) {
	fmt.Println()
	fmt.Printf("%s%s=== Indexability Analysis ===%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Start URL: %s%s%s\n", colorBlue, r.StartURL, colorReset)
	fmt.Printf("Pages analyzed: %s%d%s\n", colorGreen, r.TotalPages, colorReset)
	fmt.Printf("Total links found: %s%d%s\n", colorGreen, r.TotalLinks, colorReset)
	fmt.Println()

	indexable := r.TotalLinks - len(r.NonIndexableLinks)
	nonIndexable := len(r.NonIndexableLinks)

	fmt.Printf("%s%sIndexability Status:%s\n", colorBold, colorYellow, colorReset)
	fmt.Printf("  %sIndexable links:     %s%d%s\n", colorGreen, colorBold, indexable, colorReset)
	fmt.Printf("  %sNon-indexable links: %s%d%s\n", colorRed, colorBold, nonIndexable, colorReset)

	if nonIndexable > 0 {
		pct := float64(nonIndexable) / float64(r.TotalLinks) * 100
		fmt.Printf("  %sNon-indexable rate:  %.1f%%%s\n", colorYellow, pct, colorReset)
	}

	// Pages with noindex
	if len(r.PagesWithNoIndex) > 0 {
		fmt.Println()
		fmt.Printf("%s%sPages with noindex (%d):%s\n", colorBold, colorRed, len(r.PagesWithNoIndex), colorReset)
		for i, page := range r.PagesWithNoIndex {
			if i >= 10 {
				fmt.Printf("  %s... and %d more%s\n", colorGray, len(r.PagesWithNoIndex)-10, colorReset)
				break
			}
			fmt.Printf("  %s\n", page)
		}
	}

	// Breakdown by reason
	if len(r.ByReason) > 0 {
		fmt.Println()
		fmt.Printf("%s%sBreakdown by Reason:%s\n", colorBold, colorYellow, colorReset)

		reasons := []NoIndexReason{
			ReasonNoFollow,
			ReasonNoIndex,
			ReasonNoIndexHeader,
			ReasonSponsored,
			ReasonUGC,
			ReasonCanonicalMismatch,
			ReasonRobotsTxt,
		}

		for _, reason := range reasons {
			links := r.ByReason[reason]
			if len(links) == 0 {
				continue
			}
			fmt.Printf("\n  %s%s%s (%d links)\n", colorCyan, reason.String(), colorReset, len(links))
			fmt.Printf("  %s%s%s\n", colorGray, reason.Description(), colorReset)
		}
	}

	if showDetails && len(r.NonIndexableLinks) > 0 {
		r.printDetails()
	}

	fmt.Println()
}

func (r *IndexerResult) printDetails() {
	fmt.Println()
	fmt.Printf("%s%s=== Non-Indexable Links Details ===%s\n", colorBold, colorPurple, colorReset)

	// Group by source page
	bySource := make(map[string][]NonIndexableLink)
	for _, link := range r.NonIndexableLinks {
		bySource[link.SourceURL] = append(bySource[link.SourceURL], link)
	}

	// Sort sources
	var sources []string
	for src := range bySource {
		sources = append(sources, src)
	}
	sort.Strings(sources)

	for _, source := range sources {
		links := bySource[source]
		fmt.Printf("\n%s%s%s\n", colorCyan, source, colorReset)

		for _, link := range links {
			reasons := make([]string, len(link.Reasons))
			for i, r := range link.Reasons {
				reasons[i] = r.String()
			}
			fmt.Printf("  %s→%s %s\n", colorYellow, colorReset, link.URL)
			fmt.Printf("    %s[%s]%s\n", colorRed, strings.Join(reasons, ", "), colorReset)
			if link.Details != "" {
				fmt.Printf("    %s%s%s\n", colorGray, link.Details, colorReset)
			}
		}
	}
}
