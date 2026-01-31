package canonical

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Config holds checker configuration
type Config struct {
	Concurrency   int
	Timeout       time.Duration
	MaxDepth      int
	Verbose       bool
	FollowRedirects bool
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Concurrency:   10,
		Timeout:       10 * time.Second,
		MaxDepth:      0,
		Verbose:       false,
		FollowRedirects: true,
	}
}

// Checker verifies canonical URLs
type Checker struct {
	config       Config
	baseURL      *url.URL
	visited      map[string]bool
	visitedMu    sync.RWMutex
	canonicals   map[string]string // URL -> canonical URL
	canonicalsMu sync.RWMutex
	result       *CanonicalResult
	resultMu     sync.Mutex
	client       *http.Client
	semaphore    chan struct{}
	checkedLinks map[string]bool
	checkedMu    sync.Mutex
}

// New creates a new Checker
func New(config Config) *Checker {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	// Don't follow redirects automatically - we want to detect them
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &Checker{
		config:       config,
		visited:      make(map[string]bool),
		canonicals:   make(map[string]string),
		checkedLinks: make(map[string]bool),
		semaphore:    make(chan struct{}, config.Concurrency),
		client:       client,
	}
}

type urlTask struct {
	url       string
	sourceURL string
	depth     int
}

// Check starts the canonical verification
func (c *Checker) Check(startURL string) (*CanonicalResult, error) {
	parsed, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	c.baseURL = parsed
	c.result = NewCanonicalResult(startURL)

	tasks := make(chan urlTask, 1000)

	c.markVisited(startURL)
	tasks <- urlTask{url: startURL, sourceURL: "", depth: 0}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < c.config.Concurrency; i++ {
		go c.worker(ctx, tasks)
	}

	done := make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			c.visitedMu.RLock()
			visitedCount := len(c.visited)
			c.visitedMu.RUnlock()

			if len(tasks) == 0 && len(c.semaphore) == 0 {
				time.Sleep(500 * time.Millisecond)
				if len(tasks) == 0 && len(c.semaphore) == 0 {
					close(done)
					return
				}
			}

			if visitedCount > 10000 {
				close(done)
				return
			}
		}
	}()

	<-done
	cancel()
	close(tasks)

	c.visitedMu.RLock()
	c.result.TotalPages = len(c.visited)
	c.visitedMu.RUnlock()

	c.checkedMu.Lock()
	c.result.TotalLinks = len(c.checkedLinks)
	c.checkedMu.Unlock()

	return c.result, nil
}

func (c *Checker) worker(ctx context.Context, tasks chan urlTask) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			c.processURL(ctx, task, tasks)
		}
	}
}

func (c *Checker) processURL(ctx context.Context, task urlTask, tasks chan urlTask) {
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return
	}

	if c.config.MaxDepth > 0 && task.depth > c.config.MaxDepth {
		return
	}

	// Fetch the page, following redirects manually
	finalURL, canonical, pageInfo, err := c.fetchPage(ctx, task.url)
	if err != nil {
		if c.config.Verbose {
			printError(task.url, err.Error(), task.depth)
		}
		return
	}

	if c.config.Verbose {
		printProgress(task.url, finalURL, canonical, task.depth)
	}

	// Store canonical for this URL
	c.canonicalsMu.Lock()
	if canonical != "" {
		c.canonicals[task.url] = canonical
		c.canonicals[finalURL] = canonical
	}
	c.canonicalsMu.Unlock()

	// Check if accessed URL matches canonical
	if canonical != "" {
		if !URLsEquivalent(finalURL, canonical) {
			c.resultMu.Lock()
			c.result.AddIssue(CanonicalIssue{
				Type:         IssueCanonicalMismatch,
				SourceURL:    task.sourceURL,
				LinkedURL:    task.url,
				CanonicalURL: canonical,
				FinalURL:     finalURL,
			})
			c.resultMu.Unlock()
		}
	} else {
		// No canonical defined
		c.resultMu.Lock()
		c.result.PagesWithout = append(c.result.PagesWithout, finalURL)
		c.result.AddIssue(CanonicalIssue{
			Type:      IssueMissingCanonical,
			SourceURL: task.sourceURL,
			LinkedURL: task.url,
		})
		c.resultMu.Unlock()
	}

	// Check if there was a redirect
	if task.url != finalURL && task.sourceURL != "" {
		c.resultMu.Lock()
		c.result.AddIssue(CanonicalIssue{
			Type:         IssueRedirectToCanonical,
			SourceURL:    task.sourceURL,
			LinkedURL:    task.url,
			CanonicalURL: canonical,
			FinalURL:     finalURL,
		})
		c.resultMu.Unlock()
	}

	// Process links on this page
	if pageInfo != nil {
		for _, link := range pageInfo.Links {
			c.checkLink(ctx, finalURL, link, tasks, task.depth)
		}
	}
}

func (c *Checker) checkLink(ctx context.Context, sourceURL, linkedURL string, tasks chan urlTask, depth int) {
	// Track checked links
	linkKey := sourceURL + " -> " + linkedURL
	c.checkedMu.Lock()
	if c.checkedLinks[linkKey] {
		c.checkedMu.Unlock()
		return
	}
	c.checkedLinks[linkKey] = true
	c.checkedMu.Unlock()

	// Check if we know the canonical for this URL
	c.canonicalsMu.RLock()
	knownCanonical, hasCanonical := c.canonicals[linkedURL]
	c.canonicalsMu.RUnlock()

	if hasCanonical && !URLsEquivalent(linkedURL, knownCanonical) {
		// Link points to non-canonical URL
		c.resultMu.Lock()
		c.result.AddIssue(CanonicalIssue{
			Type:         IssueNonCanonicalLink,
			SourceURL:    sourceURL,
			LinkedURL:    linkedURL,
			CanonicalURL: knownCanonical,
		})
		c.resultMu.Unlock()
	}

	// Queue for crawling if not visited
	if c.shouldVisit(linkedURL) {
		c.markVisited(linkedURL)
		select {
		case tasks <- urlTask{url: linkedURL, sourceURL: sourceURL, depth: depth + 1}:
		default:
		}
	}
}

func (c *Checker) fetchPage(ctx context.Context, targetURL string) (finalURL, canonical string, pageInfo *PageInfo, err error) {
	currentURL := targetURL
	maxRedirects := 10

	for i := 0; i < maxRedirects; i++ {
		req, err := http.NewRequestWithContext(ctx, "GET", currentURL, nil)
		if err != nil {
			return "", "", nil, err
		}

		req.Header.Set("User-Agent", "CanonicalChecker/1.0")

		resp, err := c.client.Do(req)
		if err != nil {
			return "", "", nil, err
		}

		// Check for redirect
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := resp.Header.Get("Location")
			resp.Body.Close()

			if location == "" {
				return currentURL, "", nil, nil
			}

			// Resolve relative redirect
			base, _ := url.Parse(currentURL)
			redirectURL, err := url.Parse(location)
			if err != nil {
				return currentURL, "", nil, nil
			}

			currentURL = base.ResolveReference(redirectURL).String()
			continue
		}

		if resp.StatusCode >= 400 {
			resp.Body.Close()
			return currentURL, "", nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}

		// Check content type
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			resp.Body.Close()
			return currentURL, "", nil, nil
		}

		// Parse page
		baseURL, _ := url.Parse(currentURL)
		pageInfo = ParsePage(resp.Body, baseURL, currentURL)
		resp.Body.Close()

		return currentURL, pageInfo.CanonicalURL, pageInfo, nil
	}

	return currentURL, "", nil, fmt.Errorf("too many redirects")
}

func (c *Checker) markVisited(url string) {
	c.visitedMu.Lock()
	c.visited[url] = true
	c.visitedMu.Unlock()
}

func (c *Checker) shouldVisit(targetURL string) bool {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	if parsed.Host != c.baseURL.Host {
		return false
	}

	c.visitedMu.RLock()
	visited := c.visited[targetURL]
	c.visitedMu.RUnlock()

	return !visited
}

func printProgress(url, finalURL, canonical string, depth int) {
	indent := strings.Repeat("  ", depth)
	status := colorGreen + "✓" + colorReset
	extra := ""

	if canonical == "" {
		status = colorYellow + "!" + colorReset
		extra = " (no canonical)"
	} else if !URLsEquivalent(url, canonical) {
		status = colorYellow + "→" + colorReset
		extra = fmt.Sprintf(" → %s", canonical)
	}

	fmt.Printf("%s%s %s%s\n", indent, status, url, extra)
}

func printError(url, errMsg string, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s%s✗%s %s - %s\n", indent, colorRed, colorReset, url, errMsg)
}
