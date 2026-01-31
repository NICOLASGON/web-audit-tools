package serp

import (
	"encoding/json"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// ExtractMeta parses HTML and extracts all SEO-relevant metadata
func ExtractMeta(body io.Reader, pageURL string) *PageMeta {
	meta := &PageMeta{
		URL: pageURL,
	}

	doc, err := html.Parse(body)
	if err != nil {
		return meta
	}

	baseURL, _ := url.Parse(pageURL)

	var parseNode func(*html.Node)
	parseNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					meta.Title = strings.TrimSpace(n.FirstChild.Data)
				}

			case "meta":
				name := strings.ToLower(getAttr(n, "name"))
				property := strings.ToLower(getAttr(n, "property"))
				content := getAttr(n, "content")
				charset := getAttr(n, "charset")

				if charset != "" {
					meta.Charset = charset
				}

				// Standard meta tags
				switch name {
				case "description":
					meta.MetaDescription = content
				case "robots":
					meta.Robots = content
				case "googlebot":
					meta.GoogleBot = content
				}

				// Open Graph
				switch property {
				case "og:title":
					meta.OGTitle = content
				case "og:description":
					meta.OGDescription = content
				case "og:image":
					meta.OGImage = resolveURL(content, baseURL)
				case "og:type":
					meta.OGType = content
				case "og:site_name":
					meta.OGSiteName = content
				}

				// Twitter Cards
				switch name {
				case "twitter:card":
					meta.TwitterCard = content
				case "twitter:title":
					meta.TwitterTitle = content
				case "twitter:description":
					meta.TwitterDescription = content
				case "twitter:image":
					meta.TwitterImage = resolveURL(content, baseURL)
				}

			case "link":
				rel := strings.ToLower(getAttr(n, "rel"))
				href := getAttr(n, "href")

				switch rel {
				case "canonical":
					meta.Canonical = resolveURL(href, baseURL)
				case "icon", "shortcut icon":
					if meta.Favicon == "" {
						meta.Favicon = resolveURL(href, baseURL)
					}
				}

			case "h1":
				if meta.H1 == "" {
					meta.H1 = extractTextContent(n)
				}

			case "html":
				lang := getAttr(n, "lang")
				if lang != "" {
					meta.Lang = lang
				}

			case "script":
				scriptType := getAttr(n, "type")
				if scriptType == "application/ld+json" && n.FirstChild != nil {
					parseJSONLD(n.FirstChild.Data, meta)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseNode(c)
		}
	}

	parseNode(doc)
	return meta
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
	if href == "" {
		return ""
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return href
	}

	return baseURL.ResolveReference(parsed).String()
}

func extractTextContent(n *html.Node) string {
	var text strings.Builder

	var extract func(*html.Node)
	extract = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(n)
	return strings.TrimSpace(text.String())
}

func parseJSONLD(data string, meta *PageMeta) {
	// Try to parse as single object
	var single map[string]interface{}
	if err := json.Unmarshal([]byte(data), &single); err == nil {
		extractSchemaType(single, meta)
		return
	}

	// Try to parse as array
	var arr []map[string]interface{}
	if err := json.Unmarshal([]byte(data), &arr); err == nil {
		for _, item := range arr {
			extractSchemaType(item, meta)
		}
	}
}

func extractSchemaType(obj map[string]interface{}, meta *PageMeta) {
	if t, ok := obj["@type"]; ok {
		switch v := t.(type) {
		case string:
			meta.SchemaTypes = append(meta.SchemaTypes, v)
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					meta.SchemaTypes = append(meta.SchemaTypes, s)
				}
			}
		}
	}

	// Check for @graph
	if graph, ok := obj["@graph"].([]interface{}); ok {
		for _, item := range graph {
			if m, ok := item.(map[string]interface{}); ok {
				extractSchemaType(m, meta)
			}
		}
	}
}
