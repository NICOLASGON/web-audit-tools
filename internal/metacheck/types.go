package metacheck

import (
	"fmt"
	"sort"
	"strings"
)

// Status of a meta description
type Status int

const (
	StatusOK Status = iota
	StatusTooLong
	StatusTooShort
	StatusMissing
	StatusDuplicate
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusTooLong:
		return "Too long"
	case StatusTooShort:
		return "Too short"
	case StatusMissing:
		return "Missing"
	case StatusDuplicate:
		return "Duplicate"
	default:
		return "Unknown"
	}
}

func (s Status) Color() string {
	switch s {
	case StatusOK:
		return colorGreen
	case StatusTooLong:
		return colorRed
	case StatusTooShort:
		return colorYellow
	case StatusMissing:
		return colorRed
	case StatusDuplicate:
		return colorPurple
	default:
		return colorGray
	}
}

// Limits for meta description
const (
	DescMinLength = 70
	DescMaxLength = 155
	DescIdealMin  = 120
	DescIdealMax  = 155
)

// PageMeta holds metadata for a page
type PageMeta struct {
	URL         string
	Title       string
	TitleLength int
	Description string
	DescLength  int
	Status      Status
}

// MetaResult holds the analysis results
type MetaResult struct {
	StartURL    string
	TotalPages  int

	// Counts
	OKCount       int
	TooLongCount  int
	TooShortCount int
	MissingCount  int
	DuplicateCount int

	// Pages by status
	TooLong   []PageMeta
	TooShort  []PageMeta
	Missing   []PageMeta
	Duplicate []PageMeta
	OK        []PageMeta

	// All pages
	AllPages []PageMeta

	// Duplicate tracking
	DescriptionMap map[string][]string // description -> URLs
}

// NewMetaResult creates a new result
func NewMetaResult(startURL string) *MetaResult {
	return &MetaResult{
		StartURL:       startURL,
		DescriptionMap: make(map[string][]string),
	}
}

// AddPage adds a page to the results
func (r *MetaResult) AddPage(page PageMeta) {
	r.TotalPages++
	r.AllPages = append(r.AllPages, page)

	// Track duplicates
	if page.Description != "" {
		r.DescriptionMap[page.Description] = append(r.DescriptionMap[page.Description], page.URL)
	}
}

// Finalize calculates final stats and categorizes pages
func (r *MetaResult) Finalize() {
	// Find duplicates first
	duplicateDescs := make(map[string]bool)
	for desc, urls := range r.DescriptionMap {
		if len(urls) > 1 {
			duplicateDescs[desc] = true
		}
	}

	// Categorize pages
	for i := range r.AllPages {
		page := &r.AllPages[i]

		// Determine status
		if page.Description == "" {
			page.Status = StatusMissing
			r.MissingCount++
			r.Missing = append(r.Missing, *page)
		} else if duplicateDescs[page.Description] {
			page.Status = StatusDuplicate
			r.DuplicateCount++
			r.Duplicate = append(r.Duplicate, *page)
		} else if page.DescLength > DescMaxLength {
			page.Status = StatusTooLong
			r.TooLongCount++
			r.TooLong = append(r.TooLong, *page)
		} else if page.DescLength < DescMinLength {
			page.Status = StatusTooShort
			r.TooShortCount++
			r.TooShort = append(r.TooShort, *page)
		} else {
			page.Status = StatusOK
			r.OKCount++
			r.OK = append(r.OK, *page)
		}
	}

	// Sort too long by length descending
	sort.Slice(r.TooLong, func(i, j int) bool {
		return r.TooLong[i].DescLength > r.TooLong[j].DescLength
	})

	// Sort too short by length ascending
	sort.Slice(r.TooShort, func(i, j int) bool {
		return r.TooShort[i].DescLength < r.TooShort[j].DescLength
	})
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
func (r *MetaResult) PrintSummary(showAll bool, limit int) {
	fmt.Println()
	fmt.Printf("%s%s=== Meta Description Analysis ===%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("URL: %s%s%s\n", colorBlue, r.StartURL, colorReset)
	fmt.Printf("Pages analyzed: %s%d%s\n", colorGreen, r.TotalPages, colorReset)
	fmt.Println()

	// Summary
	fmt.Printf("%s%sSummary:%s\n", colorBold, colorYellow, colorReset)
	fmt.Printf("  %s✓ OK (70-155 chars):%s      %s%d%s\n", colorGreen, colorReset, colorBold, r.OKCount, colorReset)
	fmt.Printf("  %s✗ Too long (>155):%s       %s%d%s\n", colorRed, colorReset, colorBold, r.TooLongCount, colorReset)
	fmt.Printf("  %s! Too short (<70):%s       %s%d%s\n", colorYellow, colorReset, colorBold, r.TooShortCount, colorReset)
	fmt.Printf("  %s✗ Missing:%s                %s%d%s\n", colorRed, colorReset, colorBold, r.MissingCount, colorReset)
	fmt.Printf("  %s⚠ Duplicate:%s              %s%d%s\n", colorPurple, colorReset, colorBold, r.DuplicateCount, colorReset)

	// Show bar chart
	r.printDistributionChart()

	// Too long descriptions (main focus)
	if len(r.TooLong) > 0 {
		fmt.Println()
		fmt.Printf("%s%s=== Too Long Descriptions (%d) ===%s\n", colorBold, colorRed, len(r.TooLong), colorReset)
		fmt.Printf("%sRecommended limit is %d characters%s\n", colorGray, DescMaxLength, colorReset)
		fmt.Println()

		displayCount := limit
		if displayCount <= 0 || displayCount > len(r.TooLong) {
			displayCount = len(r.TooLong)
		}

		for i := 0; i < displayCount; i++ {
			page := r.TooLong[i]
			r.printPageDetail(page, true)
		}

		if len(r.TooLong) > displayCount {
			fmt.Printf("\n%s... and %d more pages%s\n", colorGray, len(r.TooLong)-displayCount, colorReset)
		}
	}

	// Missing descriptions
	if len(r.Missing) > 0 {
		fmt.Println()
		fmt.Printf("%s%s=== Missing Descriptions (%d) ===%s\n", colorBold, colorRed, len(r.Missing), colorReset)
		fmt.Println()

		displayCount := limit
		if displayCount <= 0 || displayCount > len(r.Missing) {
			displayCount = len(r.Missing)
		}

		for i := 0; i < displayCount; i++ {
			page := r.Missing[i]
			url := page.URL
			if len(url) > 70 {
				url = url[:67] + "..."
			}
			fmt.Printf("  %s✗%s %s\n", colorRed, colorReset, url)
		}

		if len(r.Missing) > displayCount {
			fmt.Printf("\n%s... and %d more pages%s\n", colorGray, len(r.Missing)-displayCount, colorReset)
		}
	}

	// Too short descriptions
	if len(r.TooShort) > 0 && showAll {
		fmt.Println()
		fmt.Printf("%s%s=== Too Short Descriptions (%d) ===%s\n", colorBold, colorYellow, len(r.TooShort), colorReset)
		fmt.Printf("%sRecommended minimum is %d characters%s\n", colorGray, DescMinLength, colorReset)
		fmt.Println()

		displayCount := limit
		if displayCount <= 0 || displayCount > len(r.TooShort) {
			displayCount = len(r.TooShort)
		}

		for i := 0; i < displayCount; i++ {
			page := r.TooShort[i]
			r.printPageDetail(page, false)
		}

		if len(r.TooShort) > displayCount {
			fmt.Printf("\n%s... and %d more pages%s\n", colorGray, len(r.TooShort)-displayCount, colorReset)
		}
	}

	// Duplicates
	if len(r.DescriptionMap) > 0 && showAll {
		hasDuplicates := false
		for _, urls := range r.DescriptionMap {
			if len(urls) > 1 {
				hasDuplicates = true
				break
			}
		}

		if hasDuplicates {
			fmt.Println()
			fmt.Printf("%s%s=== Duplicate Descriptions ===%s\n", colorBold, colorPurple, colorReset)
			fmt.Println()

			count := 0
			for desc, urls := range r.DescriptionMap {
				if len(urls) > 1 {
					count++
					if count > limit && limit > 0 {
						remaining := 0
						for _, u := range r.DescriptionMap {
							if len(u) > 1 {
								remaining++
							}
						}
						fmt.Printf("\n%s... and %d more groups%s\n", colorGray, remaining-limit, colorReset)
						break
					}

					truncDesc := desc
					if len(truncDesc) > 60 {
						truncDesc = truncDesc[:57] + "..."
					}
					fmt.Printf("  %s\"%s\"%s\n", colorGray, truncDesc, colorReset)
					fmt.Printf("  %sUsed on %d pages:%s\n", colorPurple, len(urls), colorReset)
					for j, url := range urls {
						if j >= 3 {
							fmt.Printf("    %s... and %d more%s\n", colorGray, len(urls)-3, colorReset)
							break
						}
						if len(url) > 65 {
							url = url[:62] + "..."
						}
						fmt.Printf("    • %s\n", url)
					}
					fmt.Println()
				}
			}
		}
	}

	// Recommendations
	r.printRecommendations()

	fmt.Println()
}

func (r *MetaResult) printDistributionChart() {
	if r.TotalPages == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("%s%sDistribution:%s\n", colorBold, colorYellow, colorReset)

	barWidth := 40

	// OK
	okPct := float64(r.OKCount) / float64(r.TotalPages)
	okBar := int(okPct * float64(barWidth))
	fmt.Printf("  OK        %s%s%s%s %d (%.0f%%)\n",
		colorGreen, strings.Repeat("█", okBar), colorGray, strings.Repeat("░", barWidth-okBar),
		r.OKCount, okPct*100)

	// Too long
	longPct := float64(r.TooLongCount) / float64(r.TotalPages)
	longBar := int(longPct * float64(barWidth))
	fmt.Printf("  Long      %s%s%s%s %d (%.0f%%)\n",
		colorRed, strings.Repeat("█", longBar), colorGray, strings.Repeat("░", barWidth-longBar),
		r.TooLongCount, longPct*100)

	// Too short
	shortPct := float64(r.TooShortCount) / float64(r.TotalPages)
	shortBar := int(shortPct * float64(barWidth))
	fmt.Printf("  Short     %s%s%s%s %d (%.0f%%)\n",
		colorYellow, strings.Repeat("█", shortBar), colorGray, strings.Repeat("░", barWidth-shortBar),
		r.TooShortCount, shortPct*100)

	// Missing
	missPct := float64(r.MissingCount) / float64(r.TotalPages)
	missBar := int(missPct * float64(barWidth))
	fmt.Printf("  Missing   %s%s%s%s %d (%.0f%%)\n",
		colorRed, strings.Repeat("█", missBar), colorGray, strings.Repeat("░", barWidth-missBar),
		r.MissingCount, missPct*100)
}

func (r *MetaResult) printPageDetail(page PageMeta, showExcess bool) {
	url := page.URL
	if len(url) > 70 {
		url = url[:67] + "..."
	}

	// Length indicator
	lengthColor := colorGreen
	if page.DescLength > DescMaxLength {
		lengthColor = colorRed
	} else if page.DescLength < DescMinLength {
		lengthColor = colorYellow
	}

	fmt.Printf("  %s[%d chars]%s %s\n", lengthColor, page.DescLength, colorReset, url)

	// Show description with truncation point
	if page.Description != "" {
		desc := page.Description
		if showExcess && len(desc) > DescMaxLength {
			// Show where it gets cut
			visible := desc[:DescMaxLength]
			excess := desc[DescMaxLength:]
			fmt.Printf("    %s\"%s%s%s%s\"%s\n",
				colorGray, visible, colorRed, excess, colorGray, colorReset)
			fmt.Printf("    %s↑ Cut at %d characters (+%d excess)%s\n",
				colorRed, DescMaxLength, len(excess), colorReset)
		} else {
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			fmt.Printf("    %s\"%s\"%s\n", colorGray, desc, colorReset)
		}
	}
	fmt.Println()
}

func (r *MetaResult) printRecommendations() {
	issues := r.TooLongCount + r.MissingCount
	if issues == 0 && r.DuplicateCount == 0 {
		fmt.Println()
		fmt.Printf("%s%s✓ All meta descriptions are properly configured!%s\n", colorBold, colorGreen, colorReset)
		return
	}

	fmt.Println()
	fmt.Printf("%s%sRecommendations:%s\n", colorBold, colorCyan, colorReset)

	if r.TooLongCount > 0 {
		fmt.Printf("\n  %s1. Too long descriptions (%d)%s\n", colorYellow, r.TooLongCount, colorReset)
		fmt.Printf("     Shorten them to maximum %d characters.\n", DescMaxLength)
		fmt.Printf("     Google truncates longer descriptions with \"...\"\n")
	}

	if r.MissingCount > 0 {
		fmt.Printf("\n  %s2. Missing descriptions (%d)%s\n", colorYellow, r.MissingCount, colorReset)
		fmt.Printf("     Add a unique meta description on each page.\n")
		fmt.Printf("     Without one, Google uses a page excerpt.\n")
	}

	if r.DuplicateCount > 0 {
		fmt.Printf("\n  %s3. Duplicate descriptions (%d)%s\n", colorYellow, r.DuplicateCount, colorReset)
		fmt.Printf("     Each page should have a unique description.\n")
		fmt.Printf("     Duplicates hurt CTR in search results.\n")
	}

	if r.TooShortCount > 0 {
		fmt.Printf("\n  %s4. Too short descriptions (%d)%s\n", colorYellow, r.TooShortCount, colorReset)
		fmt.Printf("     Aim for %d-%d characters for optimal descriptions.\n", DescIdealMin, DescIdealMax)
	}
}
