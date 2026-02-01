package migration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// Config holds the migration checker configuration
type Config struct {
	Concurrency int
	Timeout     time.Duration
	MaxDepth    int // 0 means unlimited
	Verbose     bool
	UseHEAD     bool // Use HEAD requests instead of GET for checking
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Concurrency: 10,
		Timeout:     10 * time.Second,
		MaxDepth:    0,
		Verbose:     false,
		UseHEAD:     true,
	}
}

// Migrator checks for lost links between old and new site
type Migrator struct {
	config       Config
	oldBaseURL   *url.URL
	newBaseURL   *url.URL
	visited      map[string]bool
	visitedMu    sync.RWMutex
	collectedURLs []string
	collectedMu  sync.Mutex
	lostLinks    []LostLink
	lostMu       sync.Mutex
	validCount   int
	validMu      sync.Mutex
	client       *http.Client
	semaphore    chan struct{}
}

// New creates a new Migrator instance
func New(config Config) *Migrator {
	return &Migrator{
		config:        config,
		visited:       make(map[string]bool),
		collectedURLs: make([]string, 0),
		semaphore:     make(chan struct{}, config.Concurrency),
		client: &http.Client{
			Timeout: config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// urlTask represents a URL to be crawled with its metadata
type urlTask struct {
	url       string
	sourceURL string
	depth     int
}

// Check performs the migration check between old and new site
func (m *Migrator) Check(oldSiteURL, newSiteURL string) (*MigrationResult, error) {
	// Parse old site URL
	oldParsed, err := url.Parse(oldSiteURL)
	if err != nil {
		return nil, fmt.Errorf("invalid old site URL: %w", err)
	}
	if oldParsed.Scheme != "http" && oldParsed.Scheme != "https" {
		return nil, fmt.Errorf("old site URL must use http or https scheme")
	}
	m.oldBaseURL = oldParsed

	// Parse new site URL
	newParsed, err := url.Parse(newSiteURL)
	if err != nil {
		return nil, fmt.Errorf("invalid new site URL: %w", err)
	}
	if newParsed.Scheme != "http" && newParsed.Scheme != "https" {
		return nil, fmt.Errorf("new site URL must use http or https scheme")
	}
	m.newBaseURL = newParsed

	// Phase 1: Crawl old site to collect all URLs
	if m.config.Verbose {
		fmt.Printf("\n%sPhase 1: Crawling old site...%s\n\n", colorCyan, colorReset)
	}
	err = m.crawlOldSite()
	if err != nil {
		return nil, fmt.Errorf("failed to crawl old site: %w", err)
	}

	// Phase 2: Check each URL on new site
	if m.config.Verbose {
		fmt.Printf("\n%sPhase 2: Checking URLs on new site...%s\n\n", colorCyan, colorReset)
	}
	m.checkNewSite()

	m.visitedMu.RLock()
	totalCrawled := len(m.visited)
	m.visitedMu.RUnlock()

	m.collectedMu.Lock()
	totalChecked := len(m.collectedURLs)
	m.collectedMu.Unlock()

	m.validMu.Lock()
	validLinks := m.validCount
	m.validMu.Unlock()

	return &MigrationResult{
		OldSiteURL:   oldSiteURL,
		NewSiteURL:   newSiteURL,
		TotalCrawled: totalCrawled,
		TotalChecked: totalChecked,
		LostLinks:    m.lostLinks,
		ValidLinks:   validLinks,
	}, nil
}

// crawlOldSite crawls the old site and collects all internal URLs
func (m *Migrator) crawlOldSite() error {
	startURL := m.oldBaseURL.String()

	// Channel for URLs to process
	tasks := make(chan urlTask, 1000)

	// Start with the initial URL
	m.markVisited(startURL)
	m.addCollectedURL(startURL)
	tasks <- urlTask{url: startURL, sourceURL: "", depth: 0}

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Worker pool
	for i := 0; i < m.config.Concurrency; i++ {
		go m.crawlWorker(ctx, tasks)
	}

	// Monitor completion
	done := make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			m.visitedMu.RLock()
			visitedCount := len(m.visited)
			m.visitedMu.RUnlock()

			// Check if there are pending tasks
			if len(tasks) == 0 && len(m.semaphore) == 0 {
				// Give a moment for any in-flight work
				time.Sleep(500 * time.Millisecond)
				if len(tasks) == 0 && len(m.semaphore) == 0 {
					close(done)
					return
				}
			}

			// Safety timeout
			if visitedCount > 10000 {
				close(done)
				return
			}
		}
	}()

	<-done
	cancel()
	close(tasks)

	return nil
}

// crawlWorker processes URLs from the task channel for crawling
func (m *Migrator) crawlWorker(ctx context.Context, tasks chan urlTask) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			m.processCrawlURL(ctx, task, tasks)
		}
	}
}

// processCrawlURL fetches and processes a single URL during crawling
func (m *Migrator) processCrawlURL(ctx context.Context, task urlTask, tasks chan urlTask) {
	// Acquire semaphore
	select {
	case m.semaphore <- struct{}{}:
		defer func() { <-m.semaphore }()
	case <-ctx.Done():
		return
	}

	// Check depth limit
	if m.config.MaxDepth > 0 && task.depth > m.config.MaxDepth {
		return
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", task.url, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "LinkMigration/1.0")

	resp, err := m.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		return
	}
	defer resp.Body.Close()

	if m.config.Verbose {
		fmt.Printf("  [%d] %s\n", resp.StatusCode, truncateURL(task.url, 70))
	}

	// Skip error pages
	if resp.StatusCode >= 400 {
		return
	}

	// Only parse HTML content for links
	contentType := resp.Header.Get("Content-Type")
	if !isHTML(contentType) {
		return
	}

	// Parse and extract links
	links := extractLinks(resp.Body, m.oldBaseURL)

	// Queue new links
	for _, link := range links {
		if m.shouldVisit(link) {
			m.markVisited(link)
			m.addCollectedURL(link)

			// Try to send task, skip if channel is full
			select {
			case tasks <- urlTask{url: link, sourceURL: task.url, depth: task.depth + 1}:
			default:
				// Channel full, skip this link
			}
		}
	}
}

// checkNewSite checks all collected URLs on the new site
func (m *Migrator) checkNewSite() {
	m.collectedMu.Lock()
	urls := make([]string, len(m.collectedURLs))
	copy(urls, m.collectedURLs)
	m.collectedMu.Unlock()

	// Channel for URLs to check
	checkTasks := make(chan string, len(urls))

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fill the task channel
	for _, u := range urls {
		checkTasks <- u
	}
	close(checkTasks)

	// Worker pool for checking
	var wg sync.WaitGroup
	for i := 0; i < m.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.checkWorker(ctx, checkTasks)
		}()
	}

	wg.Wait()
}

// checkWorker processes URLs to check on the new site
func (m *Migrator) checkWorker(ctx context.Context, tasks chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case oldURL, ok := <-tasks:
			if !ok {
				return
			}
			m.checkURL(ctx, oldURL)
		}
	}
}

// checkURL checks if a URL from the old site exists on the new site
func (m *Migrator) checkURL(ctx context.Context, oldURL string) {
	// Map old URL to new URL
	newURL := m.mapURL(oldURL)

	// Acquire semaphore
	select {
	case m.semaphore <- struct{}{}:
		defer func() { <-m.semaphore }()
	case <-ctx.Done():
		return
	}

	// Create request - use HEAD if configured, otherwise GET
	method := "GET"
	if m.config.UseHEAD {
		method = "HEAD"
	}

	req, err := http.NewRequestWithContext(ctx, method, newURL, nil)
	if err != nil {
		m.addLostLink(oldURL, newURL, 0, err.Error())
		if m.config.Verbose {
			PrintError(oldURL, newURL, err.Error())
		}
		return
	}

	req.Header.Set("User-Agent", "LinkMigration/1.0")

	resp, err := m.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		m.addLostLink(oldURL, newURL, 0, err.Error())
		if m.config.Verbose {
			PrintError(oldURL, newURL, err.Error())
		}
		return
	}
	defer resp.Body.Close()

	// Check if the URL is valid on new site
	if resp.StatusCode >= 400 {
		m.addLostLink(oldURL, newURL, resp.StatusCode, "")
		if m.config.Verbose {
			PrintProgress(oldURL, newURL, resp.StatusCode, true)
		}
	} else {
		m.validMu.Lock()
		m.validCount++
		m.validMu.Unlock()
		if m.config.Verbose {
			PrintProgress(oldURL, newURL, resp.StatusCode, false)
		}
	}
}

// mapURL maps a URL from the old site to the new site
func (m *Migrator) mapURL(oldURL string) string {
	parsed, err := url.Parse(oldURL)
	if err != nil {
		return oldURL
	}

	// Replace the host with the new site host
	parsed.Scheme = m.newBaseURL.Scheme
	parsed.Host = m.newBaseURL.Host

	// Keep the path and query the same
	return parsed.String()
}

// markVisited marks a URL as visited (thread-safe)
func (m *Migrator) markVisited(u string) {
	m.visitedMu.Lock()
	m.visited[u] = true
	m.visitedMu.Unlock()
}

// addCollectedURL adds a URL to the collected list (thread-safe)
func (m *Migrator) addCollectedURL(u string) {
	m.collectedMu.Lock()
	m.collectedURLs = append(m.collectedURLs, u)
	m.collectedMu.Unlock()
}

// shouldVisit checks if a URL should be visited
func (m *Migrator) shouldVisit(targetURL string) bool {
	// Check if it's an internal link
	if !isSameDomain(targetURL, m.oldBaseURL) {
		return false
	}

	// Check if already visited
	m.visitedMu.RLock()
	visited := m.visited[targetURL]
	m.visitedMu.RUnlock()

	return !visited
}

// addLostLink adds a lost link to the results (thread-safe)
func (m *Migrator) addLostLink(oldURL, newURL string, statusCode int, errMsg string) {
	m.lostMu.Lock()
	m.lostLinks = append(m.lostLinks, LostLink{
		OldURL:     oldURL,
		NewURL:     newURL,
		StatusCode: statusCode,
		Error:      errMsg,
	})
	m.lostMu.Unlock()
}

// extractLinks parses HTML content and extracts all href links
func extractLinks(body io.Reader, baseURL *url.URL) []string {
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

// isSameDomain checks if the given URL belongs to the same domain as the base URL
func isSameDomain(targetURL string, baseURL *url.URL) bool {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	return parsed.Host == baseURL.Host
}

// isHTML checks if the content type indicates HTML content
func isHTML(contentType string) bool {
	return strings.Contains(contentType, "text/html") ||
		strings.Contains(contentType, "application/xhtml+xml")
}
