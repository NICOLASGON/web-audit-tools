package serp

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Config holds fetcher configuration
type Config struct {
	Timeout time.Duration
	Verbose bool
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Timeout: 30 * time.Second,
		Verbose: false,
	}
}

// Fetcher fetches and analyzes pages
type Fetcher struct {
	config Config
	client *http.Client
}

// New creates a new Fetcher
func New(config Config) *Fetcher {
	return &Fetcher{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Analyze fetches a URL and extracts SEO metadata
func (f *Fetcher) Analyze(targetURL string) (*PageMeta, error) {
	// Validate URL
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	if f.config.Verbose {
		fmt.Printf("%sFetching %s...%s\n", colorGray, targetURL, colorReset)
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use a browser-like user agent to get the real page
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SERPreview/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "application/xhtml") {
		return nil, fmt.Errorf("not an HTML page: %s", contentType)
	}

	// Get the final URL (after redirects)
	finalURL := resp.Request.URL.String()

	// Parse the page
	meta := ExtractMeta(resp.Body, finalURL)

	// Check X-Robots-Tag header
	xRobots := resp.Header.Get("X-Robots-Tag")
	if xRobots != "" {
		if meta.Robots != "" {
			meta.Robots += ", " + xRobots
		} else {
			meta.Robots = xRobots
		}
	}

	return meta, nil
}
