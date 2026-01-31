package canonical

import (
	"fmt"
	"sort"
)

// IssueType categorizes canonical issues
type IssueType int

const (
	IssueNonCanonicalLink IssueType = iota // Link points to non-canonical URL
	IssueMissingCanonical                  // Page has no canonical tag
	IssueSelfCanonical                     // OK: page canonical points to itself
	IssueRedirectToCanonical               // Link causes redirect to canonical
	IssueCanonicalMismatch                 // Canonical differs from accessed URL
	IssueCanonicalChain                    // Canonical points to another page with different canonical
)

func (t IssueType) String() string {
	switch t {
	case IssueNonCanonicalLink:
		return "Non-canonical link"
	case IssueMissingCanonical:
		return "Missing canonical"
	case IssueRedirectToCanonical:
		return "Redirect to canonical"
	case IssueCanonicalMismatch:
		return "Canonical mismatch"
	case IssueCanonicalChain:
		return "Canonical chain"
	default:
		return "Unknown"
	}
}

func (t IssueType) Description() string {
	switch t {
	case IssueNonCanonicalLink:
		return "Link points to a URL that is not the canonical version"
	case IssueMissingCanonical:
		return "Page has no canonical tag defined"
	case IssueRedirectToCanonical:
		return "Link causes a redirect to the canonical URL"
	case IssueCanonicalMismatch:
		return "Canonical URL differs from the accessed URL"
	case IssueCanonicalChain:
		return "Canonical points to a page that has a different canonical"
	default:
		return ""
	}
}

// CanonicalIssue represents a canonical URL issue
type CanonicalIssue struct {
	Type         IssueType
	SourceURL    string // Page where the link was found
	LinkedURL    string // URL that was linked
	CanonicalURL string // The canonical URL (if different)
	FinalURL     string // URL after redirects (if applicable)
}

// PageCanonical stores canonical info for a page
type PageCanonical struct {
	URL          string
	CanonicalURL string
	HasCanonical bool
	IsSelfRef    bool
}

// CanonicalResult holds analysis results
type CanonicalResult struct {
	StartURL       string
	TotalPages     int
	TotalLinks     int
	Issues         []CanonicalIssue
	ByType         map[IssueType][]CanonicalIssue
	PagesWithout   []string // Pages without canonical
	NonCanonicals  map[string]string // URL -> canonical mapping
}

// NewCanonicalResult creates a new result
func NewCanonicalResult(startURL string) *CanonicalResult {
	return &CanonicalResult{
		StartURL:      startURL,
		ByType:        make(map[IssueType][]CanonicalIssue),
		NonCanonicals: make(map[string]string),
	}
}

// AddIssue adds an issue
func (r *CanonicalResult) AddIssue(issue CanonicalIssue) {
	r.Issues = append(r.Issues, issue)
	r.ByType[issue.Type] = append(r.ByType[issue.Type], issue)
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
)

// PrintSummary displays the results
func (r *CanonicalResult) PrintSummary(showDetails bool) {
	fmt.Println()
	fmt.Printf("%s%s=== Canonical Analysis ===%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Start URL: %s%s%s\n", colorBlue, r.StartURL, colorReset)
	fmt.Printf("Pages analyzed: %s%d%s\n", colorGreen, r.TotalPages, colorReset)
	fmt.Printf("Links checked: %s%d%s\n", colorGreen, r.TotalLinks, colorReset)
	fmt.Println()

	// Count issues
	totalIssues := len(r.Issues)
	if totalIssues == 0 {
		fmt.Printf("%s%s✓ No canonical issues detected!%s\n", colorBold, colorGreen, colorReset)
		fmt.Println()
		return
	}

	fmt.Printf("%s%s✗ %d issue(s) detected:%s\n", colorBold, colorRed, totalIssues, colorReset)
	fmt.Println()

	// Summary by type
	fmt.Printf("%s%sSummary by type:%s\n", colorBold, colorYellow, colorReset)

	issueTypes := []IssueType{
		IssueNonCanonicalLink,
		IssueRedirectToCanonical,
		IssueCanonicalMismatch,
		IssueMissingCanonical,
		IssueCanonicalChain,
	}

	for _, t := range issueTypes {
		issues := r.ByType[t]
		if len(issues) == 0 {
			continue
		}

		color := colorYellow
		if t == IssueNonCanonicalLink || t == IssueCanonicalChain {
			color = colorRed
		}

		fmt.Printf("  %s%-25s%s %d\n", color, t.String()+":", colorReset, len(issues))
	}

	if showDetails {
		r.printDetails()
	}

	// Recommendations
	r.printRecommendations()

	fmt.Println()
}

func (r *CanonicalResult) printDetails() {
	fmt.Println()
	fmt.Printf("%s%s=== Issue Details ===%s\n", colorBold, colorPurple, colorReset)

	// Group by issue type
	issueTypes := []IssueType{
		IssueNonCanonicalLink,
		IssueRedirectToCanonical,
		IssueCanonicalMismatch,
		IssueMissingCanonical,
		IssueCanonicalChain,
	}

	for _, t := range issueTypes {
		issues := r.ByType[t]
		if len(issues) == 0 {
			continue
		}

		fmt.Println()
		color := colorYellow
		if t == IssueNonCanonicalLink || t == IssueCanonicalChain {
			color = colorRed
		}

		fmt.Printf("%s%s%s (%d)%s\n", colorBold, color, t.String(), len(issues), colorReset)
		fmt.Printf("%s%s%s\n", colorGray, t.Description(), colorReset)

		// Group by source URL for cleaner output
		bySource := make(map[string][]CanonicalIssue)
		for _, issue := range issues {
			bySource[issue.SourceURL] = append(bySource[issue.SourceURL], issue)
		}

		// Sort sources
		var sources []string
		for src := range bySource {
			sources = append(sources, src)
		}
		sort.Strings(sources)

		displayed := 0
		for _, source := range sources {
			if displayed >= 10 {
				remaining := len(sources) - 10
				if remaining > 0 {
					fmt.Printf("\n  %s... and %d more pages%s\n", colorGray, remaining, colorReset)
				}
				break
			}

			srcIssues := bySource[source]
			fmt.Printf("\n  %sOn:%s %s\n", colorCyan, colorReset, truncateURL(source, 70))

			for i, issue := range srcIssues {
				if i >= 5 {
					fmt.Printf("    %s... and %d more links%s\n", colorGray, len(srcIssues)-5, colorReset)
					break
				}

				fmt.Printf("    %s→%s %s\n", colorYellow, colorReset, truncateURL(issue.LinkedURL, 65))
				if issue.CanonicalURL != "" && issue.CanonicalURL != issue.LinkedURL {
					fmt.Printf("      %sCanonical:%s %s\n", colorGreen, colorReset, truncateURL(issue.CanonicalURL, 60))
				}
				if issue.FinalURL != "" && issue.FinalURL != issue.LinkedURL {
					fmt.Printf("      %sRedirects to:%s %s\n", colorGray, colorReset, truncateURL(issue.FinalURL, 55))
				}
			}

			displayed++
		}
	}
}

func (r *CanonicalResult) printRecommendations() {
	fmt.Println()
	fmt.Printf("%s%s=== Recommendations ===%s\n", colorBold, colorCyan, colorReset)

	if len(r.ByType[IssueNonCanonicalLink]) > 0 {
		fmt.Printf("\n%s1. Non-canonical links:%s\n", colorYellow, colorReset)
		fmt.Printf("   Update links to point directly to canonical URLs.\n")
		fmt.Printf("   This avoids redirects and improves crawl budget.\n")
	}

	if len(r.ByType[IssueRedirectToCanonical]) > 0 {
		fmt.Printf("\n%s2. Redirects to canonical:%s\n", colorYellow, colorReset)
		fmt.Printf("   Replace links with final URLs to avoid redirects.\n")
	}

	if len(r.ByType[IssueMissingCanonical]) > 0 {
		fmt.Printf("\n%s3. Missing canonicals:%s\n", colorYellow, colorReset)
		fmt.Printf("   Add a <link rel=\"canonical\"> tag on each page.\n")
	}

	if len(r.ByType[IssueCanonicalChain]) > 0 {
		fmt.Printf("\n%s4. Canonical chains:%s\n", colorRed, colorReset)
		fmt.Printf("   Canonicals should point to the final version, not an\n")
		fmt.Printf("   intermediate page. Fix chains A→B→C to A→C.\n")
	}
}

func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}
