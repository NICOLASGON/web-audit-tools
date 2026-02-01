package migration

import (
	"fmt"
	"strings"
)

// LostLink represents a URL that exists on the old site but is not available on the new site
type LostLink struct {
	OldURL     string
	NewURL     string
	StatusCode int
	Error      string
}

// MigrationResult holds the complete results of a migration check
type MigrationResult struct {
	OldSiteURL   string
	NewSiteURL   string
	TotalCrawled int
	TotalChecked int
	LostLinks    []LostLink
	ValidLinks   int
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorGray   = "\033[90m"
)

// PrintSummary displays the migration check results in a formatted way
func (r *MigrationResult) PrintSummary() {
	fmt.Println()
	fmt.Printf("%s%s=== Migration Check Summary ===%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Old site: %s%s%s\n", colorBlue, r.OldSiteURL, colorReset)
	fmt.Printf("New site: %s%s%s\n", colorBlue, r.NewSiteURL, colorReset)
	fmt.Println()
	fmt.Printf("Pages crawled on old site: %s%d%s\n", colorGreen, r.TotalCrawled, colorReset)
	fmt.Printf("URLs checked on new site:  %s%d%s\n", colorGreen, r.TotalChecked, colorReset)
	fmt.Printf("Valid links:               %s%d%s\n", colorGreen, r.ValidLinks, colorReset)
	fmt.Println()

	if len(r.LostLinks) == 0 {
		fmt.Printf("%s%s✓ All links are available on the new site!%s\n", colorBold, colorGreen, colorReset)
		return
	}

	fmt.Printf("%s%s✗ Found %d lost link(s):%s\n\n", colorBold, colorRed, len(r.LostLinks), colorReset)

	// Group by status code
	byStatus := make(map[int][]LostLink)
	var errorLinks []LostLink

	for _, link := range r.LostLinks {
		if link.Error != "" {
			errorLinks = append(errorLinks, link)
		} else {
			byStatus[link.StatusCode] = append(byStatus[link.StatusCode], link)
		}
	}

	// Print 404 errors first (most common migration issue)
	if links, ok := byStatus[404]; ok {
		fmt.Printf("%s--- 404 Not Found (%d) ---%s\n\n", colorYellow, len(links), colorReset)
		for _, link := range links {
			printLostLink(link)
		}
		delete(byStatus, 404)
	}

	// Print other status codes
	for status, links := range byStatus {
		fmt.Printf("%s--- HTTP %d (%d) ---%s\n\n", colorYellow, status, len(links), colorReset)
		for _, link := range links {
			printLostLink(link)
		}
	}

	// Print connection errors
	if len(errorLinks) > 0 {
		fmt.Printf("%s--- Connection Errors (%d) ---%s\n\n", colorYellow, len(errorLinks), colorReset)
		for _, link := range errorLinks {
			printLostLink(link)
		}
	}
}

func printLostLink(link LostLink) {
	fmt.Printf("  %s%s%s\n", colorRed, link.OldURL, colorReset)
	fmt.Printf("    → %s%s%s\n", colorGray, link.NewURL, colorReset)
	if link.StatusCode > 0 {
		fmt.Printf("    Status: %s%d%s\n", colorRed, link.StatusCode, colorReset)
	}
	if link.Error != "" {
		fmt.Printf("    Error: %s\n", link.Error)
	}
	fmt.Println()
}

// PrintProgress displays progress information during the check
func PrintProgress(oldURL, newURL string, statusCode int, isLost bool) {
	status := fmt.Sprintf("%d", statusCode)
	var statusColor string

	if isLost {
		statusColor = colorRed
	} else {
		statusColor = colorGreen
	}

	// Truncate URLs if too long
	displayOld := truncateURL(oldURL, 60)
	displayNew := truncateURL(newURL, 60)

	fmt.Printf("%s[%s]%s %s → %s\n", statusColor, status, colorReset, displayOld, displayNew)
}

// PrintError displays an error during URL check
func PrintError(oldURL, newURL, errMsg string) {
	displayOld := truncateURL(oldURL, 60)
	fmt.Printf("%s[ERR]%s %s - %s\n", colorRed, colorReset, displayOld, errMsg)
}

func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}

// ExportCSV exports the lost links to CSV format
func (r *MigrationResult) ExportCSV() string {
	var sb strings.Builder
	sb.WriteString("old_url,new_url,status_code,error\n")

	for _, link := range r.LostLinks {
		errField := strings.ReplaceAll(link.Error, "\"", "'")
		sb.WriteString(fmt.Sprintf("\"%s\",\"%s\",%d,\"%s\"\n",
			link.OldURL, link.NewURL, link.StatusCode, errField))
	}

	return sb.String()
}
