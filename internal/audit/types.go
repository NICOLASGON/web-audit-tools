package audit

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Severity levels for issues
type Severity int

const (
	SeverityCritical Severity = iota
	SeverityHigh
	SeverityMedium
	SeverityLow
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityCritical:
		return "CRITICAL"
	case SeverityHigh:
		return "HIGH"
	case SeverityMedium:
		return "MEDIUM"
	case SeverityLow:
		return "LOW"
	case SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

func (s Severity) Color() string {
	switch s {
	case SeverityCritical:
		return colorRed + colorBold
	case SeverityHigh:
		return colorRed
	case SeverityMedium:
		return colorYellow
	case SeverityLow:
		return colorBlue
	case SeverityInfo:
		return colorGray
	default:
		return colorReset
	}
}

// Category of audit checks
type Category string

const (
	CategoryBrokenLinks   Category = "Broken Links"
	CategoryIndexability  Category = "Indexability"
	CategoryCanonical     Category = "Canonicals"
	CategoryPerformance   Category = "Performance"
	CategorySEO           Category = "SEO"
	CategoryArchitecture  Category = "Architecture"
)

// Issue represents a single audit issue
type Issue struct {
	Category    Category
	Severity    Severity
	Title       string
	Description string
	Count       int
	Examples    []string
	Suggestion  string
}

// AuditResult holds the complete audit results
type AuditResult struct {
	URL           string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration

	// Summary stats
	TotalPages    int
	TotalLinks    int

	// Broken links
	BrokenLinks   int
	BrokenURLs    []string

	// Non-analyzable links
	ExternalLinks int
	FileLinks     int
	MailtoLinks   int
	JSLinks       int

	// Indexability
	NoFollowLinks int
	NoIndexPages  int
	RobotBlocked  int

	// Canonicals
	MissingCanonical   int
	MismatchCanonical  int
	RedirectToCanonical int

	// Performance
	SlowPages      int   // > 1s
	VerySlowPages  int   // > 3s
	AvgLatency     time.Duration
	MaxLatency     time.Duration

	// SEO (from start page)
	HasTitle           bool
	TitleLength        int
	HasMetaDescription bool
	DescriptionLength  int
	HasOGTags          bool
	HasTwitterCards    bool
	HasCanonical       bool
	HasH1              bool
	SchemaTypes        []string

	// PageRank
	OrphanPages    int
	DeadEndPages   int
	TopPages       []PageRankInfo

	// All issues
	Issues []Issue

	// Scores
	OverallScore     int
	BrokenLinksScore int
	SEOScore         int
	PerformanceScore int
	ArchitectureScore int
}

// PageRankInfo holds basic PageRank info
type PageRankInfo struct {
	URL     string
	Score   float64
	InLinks int
}

// ANSI colors
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
	colorUnder  = "\033[4m"
)

// CalculateScores calculates audit scores
func (r *AuditResult) CalculateScores() {
	// Broken Links Score (0-100)
	if r.TotalLinks > 0 {
		brokenRatio := float64(r.BrokenLinks) / float64(r.TotalLinks)
		r.BrokenLinksScore = 100 - int(brokenRatio*500) // -5 points per 1% broken
		if r.BrokenLinksScore < 0 {
			r.BrokenLinksScore = 0
		}
	} else {
		r.BrokenLinksScore = 100
	}

	// SEO Score (0-100)
	seoPoints := 0
	if r.HasTitle {
		seoPoints += 15
		if r.TitleLength >= 30 && r.TitleLength <= 60 {
			seoPoints += 10
		}
	}
	if r.HasMetaDescription {
		seoPoints += 15
		if r.DescriptionLength >= 70 && r.DescriptionLength <= 155 {
			seoPoints += 10
		}
	}
	if r.HasCanonical {
		seoPoints += 15
	}
	if r.HasH1 {
		seoPoints += 10
	}
	if r.HasOGTags {
		seoPoints += 10
	}
	if r.HasTwitterCards {
		seoPoints += 5
	}
	if len(r.SchemaTypes) > 0 {
		seoPoints += 10
	}
	r.SEOScore = seoPoints

	// Performance Score (0-100)
	if r.TotalPages > 0 {
		slowRatio := float64(r.SlowPages) / float64(r.TotalPages)
		verySlowRatio := float64(r.VerySlowPages) / float64(r.TotalPages)
		r.PerformanceScore = 100 - int(slowRatio*50) - int(verySlowRatio*100)
		if r.PerformanceScore < 0 {
			r.PerformanceScore = 0
		}
	} else {
		r.PerformanceScore = 100
	}

	// Architecture Score (0-100)
	archPoints := 100
	if r.TotalPages > 0 {
		orphanRatio := float64(r.OrphanPages) / float64(r.TotalPages)
		deadEndRatio := float64(r.DeadEndPages) / float64(r.TotalPages)
		canonicalIssues := float64(r.MissingCanonical+r.MismatchCanonical) / float64(r.TotalPages)

		archPoints -= int(orphanRatio * 200)
		archPoints -= int(deadEndRatio * 100)
		archPoints -= int(canonicalIssues * 100)
	}
	if archPoints < 0 {
		archPoints = 0
	}
	r.ArchitectureScore = archPoints

	// Overall Score (weighted average)
	r.OverallScore = (r.BrokenLinksScore*25 + r.SEOScore*25 + r.PerformanceScore*25 + r.ArchitectureScore*25) / 100
}

// BuildIssues generates the issues list from results
func (r *AuditResult) BuildIssues() {
	r.Issues = nil

	// Broken links
	if r.BrokenLinks > 0 {
		severity := SeverityMedium
		if r.BrokenLinks > 10 {
			severity = SeverityHigh
		}
		if r.BrokenLinks > 50 {
			severity = SeverityCritical
		}
		r.Issues = append(r.Issues, Issue{
			Category:    CategoryBrokenLinks,
			Severity:    severity,
			Title:       "Broken links detected",
			Description: fmt.Sprintf("%d link(s) return a 404 error or are unreachable", r.BrokenLinks),
			Count:       r.BrokenLinks,
			Examples:    r.BrokenURLs,
			Suggestion:  "Fix or remove broken links. 404 errors hurt user experience and SEO.",
		})
	}

	// Missing title
	if !r.HasTitle {
		r.Issues = append(r.Issues, Issue{
			Category:    CategorySEO,
			Severity:    SeverityCritical,
			Title:       "Missing title tag",
			Description: "The homepage has no <title> tag",
			Suggestion:  "Add a unique and descriptive <title> tag (30-60 characters).",
		})
	} else if r.TitleLength < 30 || r.TitleLength > 60 {
		r.Issues = append(r.Issues, Issue{
			Category:    CategorySEO,
			Severity:    SeverityMedium,
			Title:       "Suboptimal title length",
			Description: fmt.Sprintf("Title is %d characters (recommended: 30-60)", r.TitleLength),
			Suggestion:  "Adjust title length for optimal SERP display.",
		})
	}

	// Missing meta description
	if !r.HasMetaDescription {
		r.Issues = append(r.Issues, Issue{
			Category:    CategorySEO,
			Severity:    SeverityHigh,
			Title:       "Missing meta description",
			Description: "The homepage has no meta description",
			Suggestion:  "Add a unique and engaging meta description (70-155 characters).",
		})
	} else if r.DescriptionLength < 70 || r.DescriptionLength > 155 {
		r.Issues = append(r.Issues, Issue{
			Category:    CategorySEO,
			Severity:    SeverityLow,
			Title:       "Suboptimal meta description length",
			Description: fmt.Sprintf("Description is %d characters (recommended: 70-155)", r.DescriptionLength),
			Suggestion:  "Adjust length to avoid truncation in Google results.",
		})
	}

	// Missing canonical
	if r.MissingCanonical > 0 {
		severity := SeverityMedium
		if r.MissingCanonical > r.TotalPages/2 {
			severity = SeverityHigh
		}
		r.Issues = append(r.Issues, Issue{
			Category:    CategoryCanonical,
			Severity:    severity,
			Title:       "Missing canonicals",
			Description: fmt.Sprintf("%d page(s) have no canonical tag", r.MissingCanonical),
			Count:       r.MissingCanonical,
			Suggestion:  "Add <link rel=\"canonical\"> on each page to avoid duplicate content.",
		})
	}

	// Canonical mismatches
	if r.MismatchCanonical > 0 {
		r.Issues = append(r.Issues, Issue{
			Category:    CategoryCanonical,
			Severity:    SeverityHigh,
			Title:       "Incorrect canonicals",
			Description: fmt.Sprintf("%d link(s) point to non-canonical URLs", r.MismatchCanonical),
			Count:       r.MismatchCanonical,
			Suggestion:  "Update links to point to canonical URLs.",
		})
	}

	// Noindex pages
	if r.NoIndexPages > 0 {
		r.Issues = append(r.Issues, Issue{
			Category:    CategoryIndexability,
			Severity:    SeverityInfo,
			Title:       "Noindex pages",
			Description: fmt.Sprintf("%d page(s) have a noindex directive", r.NoIndexPages),
			Count:       r.NoIndexPages,
			Suggestion:  "Verify these pages should be excluded from indexing.",
		})
	}

	// NoFollow links
	if r.NoFollowLinks > 0 {
		r.Issues = append(r.Issues, Issue{
			Category:    CategoryIndexability,
			Severity:    SeverityInfo,
			Title:       "Nofollow links",
			Description: fmt.Sprintf("%d link(s) have the rel=\"nofollow\" attribute", r.NoFollowLinks),
			Count:       r.NoFollowLinks,
			Suggestion:  "Nofollow links don't pass PageRank. Use them wisely.",
		})
	}

	// Slow pages
	if r.SlowPages > 0 {
		severity := SeverityLow
		if r.SlowPages > r.TotalPages/4 {
			severity = SeverityMedium
		}
		if r.VerySlowPages > 0 {
			severity = SeverityHigh
		}
		r.Issues = append(r.Issues, Issue{
			Category:    CategoryPerformance,
			Severity:    severity,
			Title:       "Slow pages detected",
			Description: fmt.Sprintf("%d page(s) >1s, including %d >3s. Average latency: %v", r.SlowPages, r.VerySlowPages, r.AvgLatency.Round(time.Millisecond)),
			Count:       r.SlowPages,
			Suggestion:  "Optimize performance: compression, caching, images, minified CSS/JS.",
		})
	}

	// Orphan pages
	if r.OrphanPages > 1 { // Start page is always orphan
		r.Issues = append(r.Issues, Issue{
			Category:    CategoryArchitecture,
			Severity:    SeverityMedium,
			Title:       "Orphan pages",
			Description: fmt.Sprintf("%d page(s) have no internal incoming links", r.OrphanPages),
			Count:       r.OrphanPages,
			Suggestion:  "Add internal links to these pages to improve discoverability.",
		})
	}

	// Dead end pages
	if r.DeadEndPages > 0 {
		severity := SeverityLow
		if r.DeadEndPages > r.TotalPages/4 {
			severity = SeverityMedium
		}
		r.Issues = append(r.Issues, Issue{
			Category:    CategoryArchitecture,
			Severity:    severity,
			Title:       "Dead-end pages",
			Description: fmt.Sprintf("%d page(s) have no outgoing links", r.DeadEndPages),
			Count:       r.DeadEndPages,
			Suggestion:  "Add outgoing links to improve navigation and distribute PageRank.",
		})
	}

	// Missing Open Graph
	if !r.HasOGTags {
		r.Issues = append(r.Issues, Issue{
			Category:    CategorySEO,
			Severity:    SeverityLow,
			Title:       "Missing Open Graph tags",
			Description: "The homepage has no Open Graph tags",
			Suggestion:  "Add og:title, og:description, og:image for better social sharing.",
		})
	}

	// Missing Twitter Cards
	if !r.HasTwitterCards {
		r.Issues = append(r.Issues, Issue{
			Category:    CategorySEO,
			Severity:    SeverityInfo,
			Title:       "Missing Twitter Cards",
			Description: "The homepage has no Twitter Card tags",
			Suggestion:  "Add twitter:card, twitter:title, twitter:description for Twitter.",
		})
	}

	// Missing structured data
	if len(r.SchemaTypes) == 0 {
		r.Issues = append(r.Issues, Issue{
			Category:    CategorySEO,
			Severity:    SeverityLow,
			Title:       "Missing structured data",
			Description: "No Schema.org structured data detected",
			Suggestion:  "Add JSON-LD data for rich snippets (Organization, WebSite, etc.).",
		})
	}

	// Sort issues by severity
	sort.Slice(r.Issues, func(i, j int) bool {
		return r.Issues[i].Severity < r.Issues[j].Severity
	})
}

// PrintReport displays the full audit report
func (r *AuditResult) PrintReport() {
	r.printHeader()
	r.printScores()
	r.printSummary()
	r.printIssues()
	r.printRecommendations()
	r.printFooter()
}

func (r *AuditResult) printHeader() {
	fmt.Println()
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("%s%s                           SEO AUDIT REPORT                              %s\n", colorBold, colorCyan, colorReset)
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println()
	fmt.Printf("  URL: %s%s%s\n", colorBlue, r.URL, colorReset)
	fmt.Printf("  Date: %s%s%s\n", colorGray, r.StartTime.Format("2006-01-02 15:04:05"), colorReset)
	fmt.Printf("  Audit duration: %s%v%s\n", colorYellow, r.Duration.Round(time.Second), colorReset)
	fmt.Println()
}

func (r *AuditResult) printScores() {
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("%s%s  SCORES%s\n", colorBold, colorCyan, colorReset)
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	// Overall score with big display
	scoreColor := colorRed
	grade := "F"
	if r.OverallScore >= 90 {
		scoreColor = colorGreen
		grade = "A"
	} else if r.OverallScore >= 80 {
		scoreColor = colorGreen
		grade = "B"
	} else if r.OverallScore >= 70 {
		scoreColor = colorYellow
		grade = "C"
	} else if r.OverallScore >= 50 {
		scoreColor = colorYellow
		grade = "D"
	}

	fmt.Printf("  %s%sOverall Score: %d/100 (%s)%s\n\n", colorBold, scoreColor, r.OverallScore, grade, colorReset)

	// Individual scores with bars
	printScoreBar("Broken Links", r.BrokenLinksScore, 20)
	printScoreBar("SEO", r.SEOScore, 20)
	printScoreBar("Performance", r.PerformanceScore, 20)
	printScoreBar("Architecture", r.ArchitectureScore, 20)

	fmt.Println()
}

func printScoreBar(label string, score int, width int) {
	filled := score * width / 100
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}

	color := colorRed
	if score >= 80 {
		color = colorGreen
	} else if score >= 60 {
		color = colorYellow
	}

	bar := strings.Repeat("█", filled)
	empty := strings.Repeat("░", width-filled)

	fmt.Printf("  %-15s %s%s%s%s %3d%%\n", label, color, bar, colorGray, empty, score)
}

func (r *AuditResult) printSummary() {
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("%s%s  SUMMARY%s\n", colorBold, colorCyan, colorReset)
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	fmt.Printf("  %sPages analyzed:%s        %d\n", colorGray, colorReset, r.TotalPages)
	fmt.Printf("  %sInternal links:%s        %d\n", colorGray, colorReset, r.TotalLinks)
	fmt.Printf("  %sExternal links:%s        %d\n", colorGray, colorReset, r.ExternalLinks)
	fmt.Printf("  %sBroken links:%s          %s%d%s\n", colorGray, colorReset, getCountColor(r.BrokenLinks, 0, 5), r.BrokenLinks, colorReset)
	fmt.Printf("  %sAverage latency:%s       %v\n", colorGray, colorReset, r.AvgLatency.Round(time.Millisecond))
	fmt.Printf("  %sMax latency:%s           %v\n", colorGray, colorReset, r.MaxLatency.Round(time.Millisecond))
	fmt.Println()
}

func getCountColor(count, goodMax, warnMax int) string {
	if count <= goodMax {
		return colorGreen
	}
	if count <= warnMax {
		return colorYellow
	}
	return colorRed
}

func (r *AuditResult) printIssues() {
	if len(r.Issues) == 0 {
		fmt.Printf("%s%s  ✓ No issues detected!%s\n\n", colorBold, colorGreen, colorReset)
		return
	}

	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("%s%s  ISSUES DETECTED (%d)%s\n", colorBold, colorCyan, len(r.Issues), colorReset)
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	// Group by severity
	bySeverity := make(map[Severity][]Issue)
	for _, issue := range r.Issues {
		bySeverity[issue.Severity] = append(bySeverity[issue.Severity], issue)
	}

	severities := []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo}

	for _, sev := range severities {
		issues := bySeverity[sev]
		if len(issues) == 0 {
			continue
		}

		fmt.Printf("  %s[%s]%s\n\n", sev.Color(), sev.String(), colorReset)

		for _, issue := range issues {
			fmt.Printf("    %s• %s%s", colorYellow, issue.Title, colorReset)
			if issue.Count > 0 {
				fmt.Printf(" (%d)", issue.Count)
			}
			fmt.Println()
			fmt.Printf("      %s%s%s\n", colorGray, issue.Description, colorReset)

			if len(issue.Examples) > 0 {
				for i, ex := range issue.Examples {
					if i >= 3 {
						fmt.Printf("        %s... and %d more%s\n", colorGray, len(issue.Examples)-3, colorReset)
						break
					}
					if len(ex) > 60 {
						ex = ex[:57] + "..."
					}
					fmt.Printf("        %s→ %s%s\n", colorGray, ex, colorReset)
				}
			}
			fmt.Println()
		}
	}
}

func (r *AuditResult) printRecommendations() {
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("%s%s  PRIORITY RECOMMENDATIONS%s\n", colorBold, colorCyan, colorReset)
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	// Get high priority issues
	var priorities []Issue
	for _, issue := range r.Issues {
		if issue.Severity <= SeverityMedium {
			priorities = append(priorities, issue)
		}
	}

	if len(priorities) == 0 {
		fmt.Printf("  %s✓ Your site is well optimized!%s\n\n", colorGreen, colorReset)
		fmt.Printf("  Suggestions for further improvement:\n")
		fmt.Printf("  • Continue monitoring for broken links\n")
		fmt.Printf("  • Regularly analyze performance\n")
		fmt.Printf("  • Enrich content with structured data\n")
	} else {
		for i, issue := range priorities {
			if i >= 5 {
				break
			}
			fmt.Printf("  %s%d.%s %s%s%s\n", colorYellow, i+1, colorReset, colorBold, issue.Title, colorReset)
			fmt.Printf("     %s%s%s\n", colorGray, issue.Suggestion, colorReset)
			fmt.Println()
		}
	}
}

func (r *AuditResult) printFooter() {
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("%s  Audit generated by web-tools/siteaudit%s\n", colorGray, colorReset)
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println()
}
