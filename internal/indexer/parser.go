package indexer

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// LinkInfo contains information about a link and its indexability
type LinkInfo struct {
	URL        string
	IsNoFollow bool
	IsSponsored bool
	IsUGC      bool
}

// PageInfo contains indexability information about a page
type PageInfo struct {
	URL              string
	Links            []LinkInfo
	HasNoIndex       bool
	HasNoFollow      bool
	CanonicalURL     string
	CanonicalMismatch bool
}

// ParsePage extracts links and indexability info from HTML
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
			case "meta":
				name := getAttr(n, "name")
				content := strings.ToLower(getAttr(n, "content"))

				if strings.ToLower(name) == "robots" {
					if strings.Contains(content, "noindex") {
						info.HasNoIndex = true
					}
					if strings.Contains(content, "nofollow") {
						info.HasNoFollow = true
					}
				}

			case "link":
				rel := strings.ToLower(getAttr(n, "rel"))
				if rel == "canonical" {
					href := getAttr(n, "href")
					if href != "" {
						canonical := resolveURL(href, baseURL)
						info.CanonicalURL = canonical
						// Check if canonical differs from current URL
						if canonical != "" && normalizeURL(canonical) != normalizeURL(pageURL) {
							info.CanonicalMismatch = true
						}
					}
				}

			case "a":
				href := getAttr(n, "href")
				if href == "" {
					break
				}

				resolved := resolveURL(href, baseURL)
				if resolved == "" {
					break
				}

				rel := strings.ToLower(getAttr(n, "rel"))
				relParts := strings.Fields(rel)

				linkInfo := LinkInfo{
					URL: resolved,
				}

				for _, part := range relParts {
					switch part {
					case "nofollow":
						linkInfo.IsNoFollow = true
					case "sponsored":
						linkInfo.IsSponsored = true
					case "ugc":
						linkInfo.IsUGC = true
					}
				}

				info.Links = append(info.Links, linkInfo)
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
		if attr.Key == key {
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

	// Skip non-HTTP links
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

	resolved.Fragment = ""
	return resolved.String()
}

func normalizeURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	// Remove trailing slash for comparison
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	parsed.Fragment = ""
	return parsed.String()
}

// IsSameDomain checks if URL is on the same domain
func IsSameDomain(targetURL string, baseURL *url.URL) bool {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	return parsed.Host == baseURL.Host
}
