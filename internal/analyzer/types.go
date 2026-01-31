package analyzer

import (
	"fmt"
	"sort"
)

// LinkType categorizes the type of link
type LinkType int

const (
	LinkTypeInternal LinkType = iota
	LinkTypeExternal
	LinkTypeMailto
	LinkTypeTel
	LinkTypeJavaScript
	LinkTypeAnchor
	LinkTypeData
	LinkTypeFile
	LinkTypeOther
)

func (t LinkType) String() string {
	switch t {
	case LinkTypeInternal:
		return "Internal"
	case LinkTypeExternal:
		return "External"
	case LinkTypeMailto:
		return "Email (mailto)"
	case LinkTypeTel:
		return "Phone (tel)"
	case LinkTypeJavaScript:
		return "JavaScript"
	case LinkTypeAnchor:
		return "Anchor (#)"
	case LinkTypeData:
		return "Data URI"
	case LinkTypeFile:
		return "File/Document"
	case LinkTypeOther:
		return "Other"
	default:
		return "Unknown"
	}
}

// Link represents a link found on a page
type Link struct {
	URL       string
	SourceURL string
	Type      LinkType
	FileType  string // For LinkTypeFile: pdf, jpg, etc.
}

// AnalysisResult holds the complete analysis results
type AnalysisResult struct {
	StartURL       string
	TotalPages     int
	TotalLinks     int
	LinksByType    map[LinkType][]Link
	ExternalByHost map[string][]Link
}

// NewAnalysisResult creates a new AnalysisResult
func NewAnalysisResult(startURL string) *AnalysisResult {
	return &AnalysisResult{
		StartURL:       startURL,
		LinksByType:    make(map[LinkType][]Link),
		ExternalByHost: make(map[string][]Link),
	}
}

// AddLink adds a link to the results
func (r *AnalysisResult) AddLink(link Link) {
	r.TotalLinks++
	r.LinksByType[link.Type] = append(r.LinksByType[link.Type], link)
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

// PrintSummary displays the analysis results
func (r *AnalysisResult) PrintSummary(showDetails bool) {
	fmt.Println()
	fmt.Printf("%s%s=== Link Analysis Summary ===%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Start URL: %s%s%s\n", colorBlue, r.StartURL, colorReset)
	fmt.Printf("Pages analyzed: %s%d%s\n", colorGreen, r.TotalPages, colorReset)
	fmt.Printf("Total links found: %s%d%s\n", colorGreen, r.TotalLinks, colorReset)
	fmt.Println()

	// Count by type
	fmt.Printf("%s%sLinks by Category:%s\n", colorBold, colorYellow, colorReset)
	fmt.Println()

	typeOrder := []LinkType{
		LinkTypeInternal,
		LinkTypeExternal,
		LinkTypeFile,
		LinkTypeMailto,
		LinkTypeTel,
		LinkTypeJavaScript,
		LinkTypeAnchor,
		LinkTypeData,
		LinkTypeOther,
	}

	for _, t := range typeOrder {
		links := r.LinksByType[t]
		if len(links) == 0 {
			continue
		}

		color := colorGreen
		if t != LinkTypeInternal {
			color = colorYellow
		}

		fmt.Printf("  %s%-20s%s %d\n", color, t.String()+":", colorReset, len(links))
	}

	// Non-analyzable links details
	if showDetails {
		r.printNonAnalyzableDetails()
	}
}

func (r *AnalysisResult) printNonAnalyzableDetails() {
	fmt.Println()
	fmt.Printf("%s%s=== Non-Analyzable Links Details ===%s\n", colorBold, colorPurple, colorReset)

	// External links grouped by host
	if links := r.LinksByType[LinkTypeExternal]; len(links) > 0 {
		fmt.Println()
		fmt.Printf("%s%sExternal Links (%d):%s\n", colorBold, colorYellow, len(links), colorReset)

		// Group by host
		byHost := make(map[string][]Link)
		for _, link := range links {
			host := extractHost(link.URL)
			byHost[host] = append(byHost[host], link)
		}

		// Sort hosts
		var hosts []string
		for host := range byHost {
			hosts = append(hosts, host)
		}
		sort.Strings(hosts)

		for _, host := range hosts {
			hostLinks := byHost[host]
			fmt.Printf("\n  %s%s%s (%d links)\n", colorCyan, host, colorReset, len(hostLinks))
			for _, link := range hostLinks {
				if len(hostLinks) <= 5 {
					fmt.Printf("    %s%s%s\n", colorGray, link.URL, colorReset)
					fmt.Printf("      from: %s\n", link.SourceURL)
				}
			}
			if len(hostLinks) > 5 {
				fmt.Printf("    %s... and %d more%s\n", colorGray, len(hostLinks), colorReset)
			}
		}
	}

	// File links grouped by type
	if links := r.LinksByType[LinkTypeFile]; len(links) > 0 {
		fmt.Println()
		fmt.Printf("%s%sFile/Document Links (%d):%s\n", colorBold, colorYellow, len(links), colorReset)

		// Group by file type
		byType := make(map[string][]Link)
		for _, link := range links {
			byType[link.FileType] = append(byType[link.FileType], link)
		}

		// Sort types
		var types []string
		for t := range byType {
			types = append(types, t)
		}
		sort.Strings(types)

		for _, ft := range types {
			typeLinks := byType[ft]
			fmt.Printf("\n  %s.%s%s (%d files)\n", colorCyan, ft, colorReset, len(typeLinks))
			for i, link := range typeLinks {
				if i >= 5 {
					fmt.Printf("    %s... and %d more%s\n", colorGray, len(typeLinks)-5, colorReset)
					break
				}
				fmt.Printf("    %s\n", link.URL)
			}
		}
	}

	// Email links
	if links := r.LinksByType[LinkTypeMailto]; len(links) > 0 {
		fmt.Println()
		fmt.Printf("%s%sEmail Links (%d):%s\n", colorBold, colorYellow, len(links), colorReset)
		seen := make(map[string]bool)
		for _, link := range links {
			email := extractEmail(link.URL)
			if !seen[email] {
				seen[email] = true
				fmt.Printf("  %s\n", email)
			}
		}
	}

	// Phone links
	if links := r.LinksByType[LinkTypeTel]; len(links) > 0 {
		fmt.Println()
		fmt.Printf("%s%sPhone Links (%d):%s\n", colorBold, colorYellow, len(links), colorReset)
		seen := make(map[string]bool)
		for _, link := range links {
			phone := extractPhone(link.URL)
			if !seen[phone] {
				seen[phone] = true
				fmt.Printf("  %s\n", phone)
			}
		}
	}

	// JavaScript links
	if links := r.LinksByType[LinkTypeJavaScript]; len(links) > 0 {
		fmt.Println()
		fmt.Printf("%s%sJavaScript Links (%d):%s\n", colorBold, colorYellow, len(links), colorReset)
		fmt.Printf("  %sThese links use JavaScript and cannot be statically analyzed%s\n", colorGray, colorReset)
	}

	// Anchor links
	if links := r.LinksByType[LinkTypeAnchor]; len(links) > 0 {
		fmt.Println()
		fmt.Printf("%s%sAnchor Links (%d):%s\n", colorBold, colorYellow, len(links), colorReset)
		fmt.Printf("  %sThese are in-page navigation links%s\n", colorGray, colorReset)
	}

	fmt.Println()
}

func extractHost(urlStr string) string {
	// Simple extraction
	start := 0
	if len(urlStr) > 8 && urlStr[:8] == "https://" {
		start = 8
	} else if len(urlStr) > 7 && urlStr[:7] == "http://" {
		start = 7
	}

	end := start
	for end < len(urlStr) && urlStr[end] != '/' && urlStr[end] != '?' {
		end++
	}

	return urlStr[start:end]
}

func extractEmail(mailto string) string {
	if len(mailto) > 7 && mailto[:7] == "mailto:" {
		email := mailto[7:]
		// Remove query params
		for i, c := range email {
			if c == '?' {
				return email[:i]
			}
		}
		return email
	}
	return mailto
}

func extractPhone(tel string) string {
	if len(tel) > 4 && tel[:4] == "tel:" {
		return tel[4:]
	}
	return tel
}
