package analyzer

import (
	"io"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// File extensions considered as non-HTML files
var fileExtensions = map[string]bool{
	"pdf": true, "doc": true, "docx": true, "xls": true, "xlsx": true,
	"ppt": true, "pptx": true, "odt": true, "ods": true, "odp": true,
	"zip": true, "rar": true, "7z": true, "tar": true, "gz": true,
	"jpg": true, "jpeg": true, "png": true, "gif": true, "svg": true,
	"webp": true, "ico": true, "bmp": true, "tiff": true,
	"mp3": true, "wav": true, "ogg": true, "flac": true,
	"mp4": true, "avi": true, "mov": true, "wmv": true, "webm": true,
	"txt": true, "csv": true, "json": true, "xml": true,
	"exe": true, "dmg": true, "pkg": true, "deb": true, "rpm": true,
}

// ExtractAllLinks parses HTML and extracts all links with their types
func ExtractAllLinks(body io.Reader, baseURL *url.URL, sourceURL string) []Link {
	var links []Link
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
						link := classifyLink(attr.Val, baseURL, sourceURL)
						if link != nil {
							links = append(links, *link)
						}
						break
					}
				}
			}
		}
	}
}

// classifyLink determines the type of a link
func classifyLink(href string, baseURL *url.URL, sourceURL string) *Link {
	href = strings.TrimSpace(href)

	if href == "" {
		return nil
	}

	lowerHref := strings.ToLower(href)

	// Check for special protocols first
	if strings.HasPrefix(lowerHref, "javascript:") {
		return &Link{URL: href, SourceURL: sourceURL, Type: LinkTypeJavaScript}
	}

	if strings.HasPrefix(lowerHref, "mailto:") {
		return &Link{URL: href, SourceURL: sourceURL, Type: LinkTypeMailto}
	}

	if strings.HasPrefix(lowerHref, "tel:") {
		return &Link{URL: href, SourceURL: sourceURL, Type: LinkTypeTel}
	}

	if strings.HasPrefix(lowerHref, "data:") {
		return &Link{URL: href, SourceURL: sourceURL, Type: LinkTypeData}
	}

	// Check for anchor-only links
	if strings.HasPrefix(href, "#") {
		return &Link{URL: href, SourceURL: sourceURL, Type: LinkTypeAnchor}
	}

	// Parse the URL
	parsedURL, err := url.Parse(href)
	if err != nil {
		return &Link{URL: href, SourceURL: sourceURL, Type: LinkTypeOther}
	}

	// Resolve relative URLs
	resolvedURL := baseURL.ResolveReference(parsedURL)

	// Check scheme
	if resolvedURL.Scheme != "http" && resolvedURL.Scheme != "https" {
		return &Link{URL: href, SourceURL: sourceURL, Type: LinkTypeOther}
	}

	// Remove fragment for comparison
	resolvedURL.Fragment = ""
	fullURL := resolvedURL.String()

	// Check if it's a file
	ext := getFileExtension(resolvedURL.Path)
	if ext != "" && fileExtensions[ext] {
		return &Link{URL: fullURL, SourceURL: sourceURL, Type: LinkTypeFile, FileType: ext}
	}

	// Check if internal or external
	if resolvedURL.Host == baseURL.Host {
		return &Link{URL: fullURL, SourceURL: sourceURL, Type: LinkTypeInternal}
	}

	return &Link{URL: fullURL, SourceURL: sourceURL, Type: LinkTypeExternal}
}

// getFileExtension extracts the lowercase file extension from a path
func getFileExtension(urlPath string) string {
	ext := path.Ext(urlPath)
	if ext == "" {
		return ""
	}
	// Remove the leading dot and convert to lowercase
	return strings.ToLower(ext[1:])
}

// IsSameDomain checks if the URL belongs to the same domain
func IsSameDomain(targetURL string, baseURL *url.URL) bool {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	return parsed.Host == baseURL.Host
}
