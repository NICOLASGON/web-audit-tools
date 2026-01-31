package serp

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// PageMeta holds extracted SEO metadata
type PageMeta struct {
	URL             string
	Title           string
	MetaDescription string
	OGTitle         string
	OGDescription   string
	OGImage         string
	OGType          string
	OGSiteName      string
	Canonical       string
	H1              string
	Favicon         string
	Lang            string
	Charset         string

	// Twitter cards
	TwitterCard        string
	TwitterTitle       string
	TwitterDescription string
	TwitterImage       string

	// Schema.org
	SchemaTypes []string

	// Robots
	Robots      string
	GoogleBot   string
}

// SERPPreview represents how the page will appear in Google
type SERPPreview struct {
	DisplayURL    string
	Title         string
	TitleTruncated bool
	Description   string
	DescTruncated bool
	Favicon       string
	SiteName      string
	Date          string
}

// Limits for Google SERP display
const (
	TitleMaxPixels = 600  // ~60 chars
	TitleMaxChars  = 60
	DescMaxPixels  = 920  // ~155 chars
	DescMaxChars   = 155
)

// ANSI colors
const (
	colorReset   = "\033[0m"
	colorBlue    = "\033[34m"
	colorGreen   = "\033[32m"
	colorGray    = "\033[90m"
	colorYellow  = "\033[33m"
	colorRed     = "\033[31m"
	colorCyan    = "\033[36m"
	colorMagenta = "\033[35m"
	colorBold    = "\033[1m"
	colorItalic  = "\033[3m"
	colorUnder   = "\033[4m"
)

// GeneratePreview creates a SERP preview from metadata
func (m *PageMeta) GeneratePreview() *SERPPreview {
	preview := &SERPPreview{}

	// Title: prefer <title>, fallback to og:title, then h1
	preview.Title = m.Title
	if preview.Title == "" {
		preview.Title = m.OGTitle
	}
	if preview.Title == "" {
		preview.Title = m.H1
	}

	// Truncate title if needed
	if utf8.RuneCountInString(preview.Title) > TitleMaxChars {
		preview.Title = truncateString(preview.Title, TitleMaxChars-3) + "..."
		preview.TitleTruncated = true
	}

	// Description: prefer meta description, fallback to og:description
	preview.Description = m.MetaDescription
	if preview.Description == "" {
		preview.Description = m.OGDescription
	}

	// Truncate description if needed
	if utf8.RuneCountInString(preview.Description) > DescMaxChars {
		preview.Description = truncateString(preview.Description, DescMaxChars-3) + "..."
		preview.DescTruncated = true
	}

	// URL formatting (Google style breadcrumb)
	preview.DisplayURL = formatGoogleURL(m.URL)

	// Site name
	preview.SiteName = m.OGSiteName

	// Favicon
	preview.Favicon = m.Favicon

	return preview
}

// PrintGooglePreview displays the SERP preview
func (p *SERPPreview) PrintGooglePreview() {
	fmt.Println()
	fmt.Printf("%s%s┌─────────────────────────────────────────────────────────────────────────────┐%s\n", colorBold, colorGray, colorReset)
	fmt.Printf("%s%s│%s  %sGoogle%s Search Preview                                                       %s│%s\n", colorBold, colorGray, colorReset, colorBlue, colorReset, colorGray, colorReset)
	fmt.Printf("%s%s├─────────────────────────────────────────────────────────────────────────────┤%s\n", colorBold, colorGray, colorReset)
	fmt.Printf("%s%s│%s                                                                             %s│%s\n", colorBold, colorGray, colorReset, colorGray, colorReset)

	// Favicon + URL line
	favicon := "○"
	if p.Favicon != "" {
		favicon = "●"
	}
	siteName := p.SiteName
	if siteName == "" {
		siteName = extractDomain(p.DisplayURL)
	}

	fmt.Printf("%s%s│%s  %s%s%s %s%s%s                                    %s│%s\n",
		colorBold, colorGray, colorReset,
		colorGray, favicon, colorReset,
		colorGray, siteName, colorReset,
		colorGray, colorReset)

	// URL breadcrumb
	fmt.Printf("%s%s│%s    %s%s%s\n",
		colorBold, colorGray, colorReset,
		colorGreen, p.DisplayURL, colorReset)

	// Title
	titleIndicator := ""
	if p.TitleTruncated {
		titleIndicator = fmt.Sprintf(" %s(truncated)%s", colorYellow, colorReset)
	}
	fmt.Printf("%s%s│%s  %s%s%s%s%s\n",
		colorBold, colorGray, colorReset,
		colorBlue, colorUnder, p.Title, colorReset, titleIndicator)

	// Description
	descIndicator := ""
	if p.DescTruncated {
		descIndicator = fmt.Sprintf(" %s(truncated)%s", colorYellow, colorReset)
	}
	if p.Description != "" {
		// Wrap description
		wrapped := wrapText(p.Description, 70)
		lines := strings.Split(wrapped, "\n")
		for i, line := range lines {
			suffix := ""
			if i == len(lines)-1 {
				suffix = descIndicator
			}
			fmt.Printf("%s%s│%s  %s%s%s%s\n",
				colorBold, colorGray, colorReset,
				colorGray, line, colorReset, suffix)
		}
	} else {
		fmt.Printf("%s%s│%s  %s(no description)%s\n",
			colorBold, colorGray, colorReset,
			colorRed, colorReset)
	}

	fmt.Printf("%s%s│%s                                                                             %s│%s\n", colorBold, colorGray, colorReset, colorGray, colorReset)
	fmt.Printf("%s%s└─────────────────────────────────────────────────────────────────────────────┘%s\n", colorBold, colorGray, colorReset)
}

// PrintMetaAnalysis displays detailed meta analysis
func (m *PageMeta) PrintMetaAnalysis() {
	fmt.Println()
	fmt.Printf("%s%s=== SEO Analysis ===%s\n", colorBold, colorCyan, colorReset)
	fmt.Println()

	// Title analysis
	fmt.Printf("%s%sTitle:%s\n", colorBold, colorYellow, colorReset)
	if m.Title != "" {
		titleLen := utf8.RuneCountInString(m.Title)
		status := colorGreen + "✓" + colorReset
		warning := ""
		if titleLen > TitleMaxChars {
			status = colorRed + "✗" + colorReset
			warning = fmt.Sprintf(" %s(too long: %d/%d chars)%s", colorRed, titleLen, TitleMaxChars, colorReset)
		} else if titleLen < 30 {
			status = colorYellow + "!" + colorReset
			warning = fmt.Sprintf(" %s(too short: %d chars, recommended: 30-60)%s", colorYellow, titleLen, colorReset)
		}
		fmt.Printf("  %s %s%s\n", status, m.Title, warning)
		fmt.Printf("    %sLength: %d characters%s\n", colorGray, titleLen, colorReset)
	} else {
		fmt.Printf("  %s✗%s %sMissing!%s\n", colorRed, colorReset, colorRed, colorReset)
	}

	// Meta description analysis
	fmt.Println()
	fmt.Printf("%s%sMeta Description:%s\n", colorBold, colorYellow, colorReset)
	if m.MetaDescription != "" {
		descLen := utf8.RuneCountInString(m.MetaDescription)
		status := colorGreen + "✓" + colorReset
		warning := ""
		if descLen > DescMaxChars {
			status = colorRed + "✗" + colorReset
			warning = fmt.Sprintf(" %s(too long: %d/%d chars)%s", colorRed, descLen, DescMaxChars, colorReset)
		} else if descLen < 70 {
			status = colorYellow + "!" + colorReset
			warning = fmt.Sprintf(" %s(too short: %d chars, recommended: 70-155)%s", colorYellow, descLen, colorReset)
		}
		wrapped := wrapText(m.MetaDescription, 65)
		lines := strings.Split(wrapped, "\n")
		fmt.Printf("  %s %s%s\n", status, lines[0], warning)
		for _, line := range lines[1:] {
			fmt.Printf("    %s\n", line)
		}
		fmt.Printf("    %sLength: %d characters%s\n", colorGray, descLen, colorReset)
	} else {
		fmt.Printf("  %s✗%s %sMissing! Google will use a page excerpt.%s\n", colorRed, colorReset, colorRed, colorReset)
	}

	// Canonical URL
	fmt.Println()
	fmt.Printf("%s%sCanonical URL:%s\n", colorBold, colorYellow, colorReset)
	if m.Canonical != "" {
		if m.Canonical == m.URL {
			fmt.Printf("  %s✓%s %s (self-referencing)\n", colorGreen, colorReset, m.Canonical)
		} else {
			fmt.Printf("  %s!%s %s\n", colorYellow, colorReset, m.Canonical)
			fmt.Printf("    %sDiffers from current URL!%s\n", colorYellow, colorReset)
		}
	} else {
		fmt.Printf("  %s!%s %sNot defined%s\n", colorYellow, colorReset, colorYellow, colorReset)
	}

	// H1
	fmt.Println()
	fmt.Printf("%s%sH1:%s\n", colorBold, colorYellow, colorReset)
	if m.H1 != "" {
		fmt.Printf("  %s✓%s %s\n", colorGreen, colorReset, m.H1)
	} else {
		fmt.Printf("  %s!%s %sNo H1 found%s\n", colorYellow, colorReset, colorYellow, colorReset)
	}

	// Open Graph
	fmt.Println()
	fmt.Printf("%s%sOpen Graph:%s\n", colorBold, colorYellow, colorReset)
	ogItems := []struct {
		name  string
		value string
	}{
		{"og:title", m.OGTitle},
		{"og:description", m.OGDescription},
		{"og:image", m.OGImage},
		{"og:type", m.OGType},
		{"og:site_name", m.OGSiteName},
	}

	hasOG := false
	for _, item := range ogItems {
		if item.value != "" {
			hasOG = true
			value := item.value
			if len(value) > 60 {
				value = value[:57] + "..."
			}
			fmt.Printf("  %s✓%s %s: %s\n", colorGreen, colorReset, item.name, value)
		}
	}
	if !hasOG {
		fmt.Printf("  %s!%s %sNo Open Graph tags%s\n", colorYellow, colorReset, colorYellow, colorReset)
	}

	// Twitter Cards
	fmt.Println()
	fmt.Printf("%s%sTwitter Cards:%s\n", colorBold, colorYellow, colorReset)
	twItems := []struct {
		name  string
		value string
	}{
		{"twitter:card", m.TwitterCard},
		{"twitter:title", m.TwitterTitle},
		{"twitter:description", m.TwitterDescription},
		{"twitter:image", m.TwitterImage},
	}

	hasTW := false
	for _, item := range twItems {
		if item.value != "" {
			hasTW = true
			value := item.value
			if len(value) > 60 {
				value = value[:57] + "..."
			}
			fmt.Printf("  %s✓%s %s: %s\n", colorGreen, colorReset, item.name, value)
		}
	}
	if !hasTW {
		fmt.Printf("  %s!%s %sNo Twitter Card tags%s\n", colorYellow, colorReset, colorYellow, colorReset)
	}

	// Robots directives
	fmt.Println()
	fmt.Printf("%s%sRobots:%s\n", colorBold, colorYellow, colorReset)
	if m.Robots != "" {
		status := colorGreen + "✓" + colorReset
		if strings.Contains(strings.ToLower(m.Robots), "noindex") {
			status = colorRed + "✗" + colorReset
		}
		fmt.Printf("  %s meta robots: %s\n", status, m.Robots)
	}
	if m.GoogleBot != "" {
		status := colorGreen + "✓" + colorReset
		if strings.Contains(strings.ToLower(m.GoogleBot), "noindex") {
			status = colorRed + "✗" + colorReset
		}
		fmt.Printf("  %s googlebot: %s\n", status, m.GoogleBot)
	}
	if m.Robots == "" && m.GoogleBot == "" {
		fmt.Printf("  %s✓%s No restrictions (indexable)\n", colorGreen, colorReset)
	}

	// Schema.org
	if len(m.SchemaTypes) > 0 {
		fmt.Println()
		fmt.Printf("%s%sSchema.org (JSON-LD):%s\n", colorBold, colorYellow, colorReset)
		for _, t := range m.SchemaTypes {
			fmt.Printf("  %s✓%s %s\n", colorGreen, colorReset, t)
		}
	}

	fmt.Println()
}

// Helper functions

func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

func formatGoogleURL(rawURL string) string {
	// Remove protocol
	url := rawURL
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Split into parts
	parts := strings.Split(url, "/")
	if len(parts) == 1 {
		return url
	}

	// Format as breadcrumb: domain › path › path
	domain := parts[0]
	path := parts[1:]

	// Filter empty parts
	var cleanPath []string
	for _, p := range path {
		if p != "" {
			cleanPath = append(cleanPath, p)
		}
	}

	if len(cleanPath) == 0 {
		return domain
	}

	return domain + " › " + strings.Join(cleanPath, " › ")
}

func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		domain := parts[0]
		// Remove www.
		domain = strings.TrimPrefix(domain, "www.")
		return domain
	}
	return url
}

func wrapText(text string, width int) string {
	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		wordLen := utf8.RuneCountInString(word)

		if lineLen+wordLen+1 > width && lineLen > 0 {
			result.WriteString("\n")
			lineLen = 0
		}

		if lineLen > 0 {
			result.WriteString(" ")
			lineLen++
		}

		result.WriteString(word)
		lineLen += wordLen

		_ = i
	}

	return result.String()
}
