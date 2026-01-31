package canonical

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// PageInfo contains parsed page information
type PageInfo struct {
	URL          string
	CanonicalURL string
	Links        []string
}

// ParsePage extracts canonical and links from HTML
func ParsePage(body io.Reader, baseURL *url.URL, pageURL string) *PageInfo {
	info := &PageInfo{
		URL: pageURL,
	}

	doc, err := html.Parse(body)
	if err != nil {
		return info
	}

	var parseNode func(*html.Node)
	parseNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "link":
				rel := strings.ToLower(getAttr(n, "rel"))
				if rel == "canonical" {
					href := getAttr(n, "href")
					if href != "" {
						info.CanonicalURL = resolveURL(href, baseURL)
					}
				}

			case "a":
				href := getAttr(n, "href")
				if href != "" {
					resolved := resolveURL(href, baseURL)
					if resolved != "" && isSameDomain(resolved, baseURL) {
						info.Links = append(info.Links, resolved)
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseNode(c)
		}
	}

	parseNode(doc)
	return info
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == key {
			return attr.Val
		}
	}
	return ""
}

func resolveURL(href string, baseURL *url.URL) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}

	lower := strings.ToLower(href)
	if strings.HasPrefix(lower, "javascript:") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(href, "#") {
		return ""
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}

	resolved := baseURL.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}

	// Remove fragment
	resolved.Fragment = ""

	return resolved.String()
}

func isSameDomain(targetURL string, baseURL *url.URL) bool {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	return parsed.Host == baseURL.Host
}

// NormalizeURL normalizes URL for comparison
func NormalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Lowercase scheme and host
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)

	// Remove default ports
	host := parsed.Host
	if strings.HasSuffix(host, ":80") && parsed.Scheme == "http" {
		parsed.Host = strings.TrimSuffix(host, ":80")
	}
	if strings.HasSuffix(host, ":443") && parsed.Scheme == "https" {
		parsed.Host = strings.TrimSuffix(host, ":443")
	}

	// Remove fragment
	parsed.Fragment = ""

	// Sort query parameters for consistent comparison
	if parsed.RawQuery != "" {
		values := parsed.Query()
		parsed.RawQuery = values.Encode()
	}

	return parsed.String()
}

// URLsEquivalent checks if two URLs are equivalent
func URLsEquivalent(url1, url2 string) bool {
	// Normalize both
	n1 := NormalizeURL(url1)
	n2 := NormalizeURL(url2)

	if n1 == n2 {
		return true
	}

	// Also check with/without trailing slash
	n1Slash := strings.TrimSuffix(n1, "/")
	n2Slash := strings.TrimSuffix(n2, "/")

	return n1Slash == n2Slash
}
