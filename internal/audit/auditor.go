package audit

import (
	"fmt"
	"net/url"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/ngonzalez/web-tools/internal/analyzer"
	"github.com/ngonzalez/web-tools/internal/canonical"
	"github.com/ngonzalez/web-tools/internal/crawler"
	"github.com/ngonzalez/web-tools/internal/indexer"
	"github.com/ngonzalez/web-tools/internal/latency"
	"github.com/ngonzalez/web-tools/internal/pagerank"
	"github.com/ngonzalez/web-tools/internal/serp"
)

// Config holds auditor configuration
type Config struct {
	Concurrency int
	Timeout     time.Duration
	MaxDepth    int
	Verbose     bool
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Concurrency: 10,
		Timeout:     15 * time.Second,
		MaxDepth:    0,
		Verbose:     false,
	}
}

// Auditor runs a complete site audit
type Auditor struct {
	config Config
	result *AuditResult
}

// New creates a new Auditor
func New(config Config) *Auditor {
	return &Auditor{
		config: config,
	}
}

// Run executes the full audit
func (a *Auditor) Run(targetURL string) (*AuditResult, error) {
	// Validate URL
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	a.result = &AuditResult{
		URL:       targetURL,
		StartTime: time.Now(),
	}

	// Run all checks
	fmt.Printf("\n%s%s[1/6]%s Analyzing broken links...\n", colorBold, colorCyan, colorReset)
	a.runBrokenLinksCheck(targetURL)

	fmt.Printf("%s%s[2/6]%s Analyzing non-analyzable links...\n", colorBold, colorCyan, colorReset)
	a.runAnalyzerCheck(targetURL)

	fmt.Printf("%s%s[3/6]%s Analyzing indexability...\n", colorBold, colorCyan, colorReset)
	a.runIndexerCheck(targetURL)

	fmt.Printf("%s%s[4/6]%s Checking canonicals...\n", colorBold, colorCyan, colorReset)
	a.runCanonicalCheck(targetURL)

	fmt.Printf("%s%s[5/6]%s Measuring performance...\n", colorBold, colorCyan, colorReset)
	a.runLatencyCheck(targetURL)

	fmt.Printf("%s%s[6/6]%s Analyzing SEO and PageRank...\n", colorBold, colorCyan, colorReset)
	a.runSEOCheck(targetURL)
	a.runPageRankCheck(targetURL)

	a.result.EndTime = time.Now()
	a.result.Duration = a.result.EndTime.Sub(a.result.StartTime)

	// Calculate scores and build issues
	a.result.CalculateScores()
	a.result.BuildIssues()

	return a.result, nil
}

func (a *Auditor) runBrokenLinksCheck(targetURL string) {
	config := crawler.Config{
		Concurrency: a.config.Concurrency,
		Timeout:     a.config.Timeout,
		MaxDepth:    a.config.MaxDepth,
		Verbose:     false,
	}

	c := crawler.New(config)
	result, err := c.Crawl(targetURL)
	if err != nil {
		if a.config.Verbose {
			fmt.Printf("  %sError: %v%s\n", colorRed, err, colorReset)
		}
		return
	}

	a.result.BrokenLinks = len(result.BrokenLinks)
	for _, bl := range result.BrokenLinks {
		a.result.BrokenURLs = append(a.result.BrokenURLs, bl.BrokenURL)
	}
	a.result.TotalVisited(result.TotalVisited)

	if a.config.Verbose {
		fmt.Printf("  %s✓ %d broken links found%s\n", colorGray, a.result.BrokenLinks, colorReset)
	}
}

func (r *AuditResult) TotalVisited(count int) {
	if count > r.TotalPages {
		r.TotalPages = count
	}
}

func (a *Auditor) runAnalyzerCheck(targetURL string) {
	config := analyzer.Config{
		Concurrency: a.config.Concurrency,
		Timeout:     a.config.Timeout,
		MaxDepth:    a.config.MaxDepth,
		Verbose:     false,
	}

	az := analyzer.New(config)
	result, err := az.Analyze(targetURL)
	if err != nil {
		if a.config.Verbose {
			fmt.Printf("  %sError: %v%s\n", colorRed, err, colorReset)
		}
		return
	}

	a.result.TotalVisited(result.TotalPages)
	a.result.TotalLinks = result.TotalLinks

	// Count by type
	for linkType, links := range result.LinksByType {
		switch linkType {
		case analyzer.LinkTypeExternal:
			a.result.ExternalLinks = len(links)
		case analyzer.LinkTypeFile:
			a.result.FileLinks = len(links)
		case analyzer.LinkTypeMailto:
			a.result.MailtoLinks = len(links)
		case analyzer.LinkTypeJavaScript:
			a.result.JSLinks = len(links)
		}
	}

	if a.config.Verbose {
		fmt.Printf("  %s✓ %d external links, %d files%s\n", colorGray, a.result.ExternalLinks, a.result.FileLinks, colorReset)
	}
}

func (a *Auditor) runIndexerCheck(targetURL string) {
	config := indexer.Config{
		Concurrency:    a.config.Concurrency,
		Timeout:        a.config.Timeout,
		MaxDepth:       a.config.MaxDepth,
		Verbose:        false,
		CheckRobotsTxt: true,
	}

	idx := indexer.New(config)
	result, err := idx.Analyze(targetURL)
	if err != nil {
		if a.config.Verbose {
			fmt.Printf("  %sError: %v%s\n", colorRed, err, colorReset)
		}
		return
	}

	a.result.TotalVisited(result.TotalPages)
	a.result.NoIndexPages = len(result.PagesWithNoIndex)

	// Count nofollow links
	for reason, issues := range result.ByReason {
		switch reason {
		case indexer.ReasonNoFollow:
			a.result.NoFollowLinks = len(issues)
		case indexer.ReasonRobotsTxt:
			a.result.RobotBlocked = len(issues)
		}
	}

	if a.config.Verbose {
		fmt.Printf("  %s✓ %d noindex pages, %d nofollow links%s\n", colorGray, a.result.NoIndexPages, a.result.NoFollowLinks, colorReset)
	}
}

func (a *Auditor) runCanonicalCheck(targetURL string) {
	config := canonical.Config{
		Concurrency: a.config.Concurrency,
		Timeout:     a.config.Timeout,
		MaxDepth:    a.config.MaxDepth,
		Verbose:     false,
	}

	checker := canonical.New(config)
	result, err := checker.Check(targetURL)
	if err != nil {
		if a.config.Verbose {
			fmt.Printf("  %sError: %v%s\n", colorRed, err, colorReset)
		}
		return
	}

	a.result.TotalVisited(result.TotalPages)
	a.result.MissingCanonical = len(result.ByType[canonical.IssueMissingCanonical])
	a.result.MismatchCanonical = len(result.ByType[canonical.IssueCanonicalMismatch]) + len(result.ByType[canonical.IssueNonCanonicalLink])
	a.result.RedirectToCanonical = len(result.ByType[canonical.IssueRedirectToCanonical])

	if a.config.Verbose {
		fmt.Printf("  %s✓ %d missing canonical, %d incorrect%s\n", colorGray, a.result.MissingCanonical, a.result.MismatchCanonical, colorReset)
	}
}

func (a *Auditor) runLatencyCheck(targetURL string) {
	config := latency.Config{
		Concurrency: a.config.Concurrency,
		Timeout:     a.config.Timeout,
		MaxDepth:    a.config.MaxDepth,
		Verbose:     false,
	}

	m := latency.New(config)
	result, err := m.Measure(targetURL)
	if err != nil {
		if a.config.Verbose {
			fmt.Printf("  %sError: %v%s\n", colorRed, err, colorReset)
		}
		return
	}

	a.result.TotalVisited(len(result.Pages))

	// Calculate stats
	var totalDuration time.Duration
	for _, page := range result.Pages {
		if page.Error != "" {
			continue
		}

		totalDuration += page.Duration

		if page.Duration > a.result.MaxLatency {
			a.result.MaxLatency = page.Duration
		}

		if page.Duration > 1*time.Second {
			a.result.SlowPages++
		}
		if page.Duration > 3*time.Second {
			a.result.VerySlowPages++
		}
	}

	if len(result.Pages) > 0 {
		a.result.AvgLatency = totalDuration / time.Duration(len(result.Pages))
	}

	if a.config.Verbose {
		fmt.Printf("  %s✓ Average latency: %v, %d slow pages%s\n", colorGray, a.result.AvgLatency.Round(time.Millisecond), a.result.SlowPages, colorReset)
	}
}

func (a *Auditor) runSEOCheck(targetURL string) {
	config := serp.Config{
		Timeout: a.config.Timeout,
		Verbose: false,
	}

	fetcher := serp.New(config)
	meta, err := fetcher.Analyze(targetURL)
	if err != nil {
		if a.config.Verbose {
			fmt.Printf("  %sError: %v%s\n", colorRed, err, colorReset)
		}
		return
	}

	a.result.HasTitle = meta.Title != ""
	a.result.TitleLength = utf8.RuneCountInString(meta.Title)
	a.result.HasMetaDescription = meta.MetaDescription != ""
	a.result.DescriptionLength = utf8.RuneCountInString(meta.MetaDescription)
	a.result.HasOGTags = meta.OGTitle != "" || meta.OGDescription != ""
	a.result.HasTwitterCards = meta.TwitterCard != ""
	a.result.HasCanonical = meta.Canonical != ""
	a.result.HasH1 = meta.H1 != ""
	a.result.SchemaTypes = meta.SchemaTypes

	if a.config.Verbose {
		fmt.Printf("  %s✓ Title: %v, Description: %v, OG: %v%s\n",
			colorGray,
			a.result.HasTitle,
			a.result.HasMetaDescription,
			a.result.HasOGTags,
			colorReset)
	}
}

func (a *Auditor) runPageRankCheck(targetURL string) {
	config := pagerank.Config{
		Concurrency:   a.config.Concurrency,
		Timeout:       a.config.Timeout,
		MaxDepth:      a.config.MaxDepth,
		Verbose:       false,
		DampingFactor: 0.85,
		MaxIterations: 50,
	}

	pr := pagerank.New(config)
	result, err := pr.Crawl(targetURL)
	if err != nil {
		if a.config.Verbose {
			fmt.Printf("  %sError: %v%s\n", colorRed, err, colorReset)
		}
		return
	}

	a.result.TotalVisited(result.TotalPages)
	a.result.TotalLinks = result.TotalLinks

	// Count orphan and dead-end pages
	for _, page := range result.Scores {
		if page.InLinks == 0 {
			a.result.OrphanPages++
		}
		if page.OutLinks == 0 {
			a.result.DeadEndPages++
		}
	}

	// Get top pages
	sorted := make([]pagerank.PageScore, len(result.Scores))
	copy(sorted, result.Scores)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	topCount := 5
	if topCount > len(sorted) {
		topCount = len(sorted)
	}

	for i := 0; i < topCount; i++ {
		a.result.TopPages = append(a.result.TopPages, PageRankInfo{
			URL:     sorted[i].URL,
			Score:   sorted[i].Score,
			InLinks: sorted[i].InLinks,
		})
	}

	if a.config.Verbose {
		fmt.Printf("  %s✓ %d orphan pages, %d dead-ends%s\n", colorGray, a.result.OrphanPages, a.result.DeadEndPages, colorReset)
	}
}
