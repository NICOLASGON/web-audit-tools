package crawler

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// ExtractLinks parses HTML content and extracts all href links
func ExtractLinks(body io.Reader, baseURL *url.URL) []string {
	var links []string
	tokenizer := html.NewTokenizer(body)

	for {
		tokenType := tokenizer.Next()

		switch tokenType {
		case html.ErrorToken:
			return links

		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()

			if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						link := normalizeURL(attr.Val, baseURL)
						if link != "" {
							links = append(links, link)
						}
						break
					}
				}
			}
		}
	}
}

// normalizeURL converts a potentially relative URL to an absolute URL
// and filters out non-HTTP URLs
func normalizeURL(href string, baseURL *url.URL) string {
	href = strings.TrimSpace(href)

	// Skip empty links
	if href == "" {
		return ""
	}

	// Skip anchors, javascript, mailto, tel, and data URLs
	lowerHref := strings.ToLower(href)
	skipPrefixes := []string{"#", "javascript:", "mailto:", "tel:", "data:", "file:"}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(lowerHref, prefix) {
			return ""
		}
	}

	// Parse the href
	parsedURL, err := url.Parse(href)
	if err != nil {
		return ""
	}

	// Resolve relative URLs against the base URL
	resolvedURL := baseURL.ResolveReference(parsedURL)

	// Only keep HTTP and HTTPS URLs
	if resolvedURL.Scheme != "http" && resolvedURL.Scheme != "https" {
		return ""
	}

	// Remove fragment
	resolvedURL.Fragment = ""

	return resolvedURL.String()
}

// IsSameDomain checks if the given URL belongs to the same domain as the base URL
func IsSameDomain(targetURL string, baseURL *url.URL) bool {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	return parsed.Host == baseURL.Host
}
