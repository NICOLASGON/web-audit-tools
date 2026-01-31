package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Config holds the crawler configuration
type Config struct {
	Concurrency int
	Timeout     time.Duration
	MaxDepth    int // 0 means unlimited
	Verbose     bool
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Concurrency: 10,
		Timeout:     10 * time.Second,
		MaxDepth:    0,
		Verbose:     false,
	}
}

// Crawler is a concurrent web crawler for finding broken links
type Crawler struct {
	config     Config
	baseURL    *url.URL
	visited    map[string]bool
	visitedMu  sync.RWMutex
	broken     []BrokenLink
	brokenMu   sync.Mutex
	client     *http.Client
	semaphore  chan struct{}
	wg         sync.WaitGroup
	totalCount int
	countMu    sync.Mutex
}

// New creates a new Crawler instance
func New(config Config) *Crawler {
	return &Crawler{
		config:    config,
		visited:   make(map[string]bool),
		semaphore: make(chan struct{}, config.Concurrency),
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

// Crawl starts crawling from the given URL and returns the results
func (c *Crawler) Crawl(startURL string) (*CrawlResult, error) {
	parsed, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	c.baseURL = parsed

	// Channel for URLs to process
	tasks := make(chan urlTask, 1000)

	// Start with the initial URL
	c.markVisited(startURL)
	tasks <- urlTask{url: startURL, sourceURL: "", depth: 0}

	// Track active workers
	var activeWorkers sync.WaitGroup

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Worker pool
	for i := 0; i < c.config.Concurrency; i++ {
		go c.worker(ctx, tasks, &activeWorkers)
	}

	// Wait for all work to complete
	activeWorkers.Add(1)
	go func() {
		defer activeWorkers.Done()
		// Initial task is already sent, wait a bit for it to be picked up
		time.Sleep(100 * time.Millisecond)
	}()

	// Monitor completion
	done := make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			c.visitedMu.RLock()
			visitedCount := len(c.visited)
			c.visitedMu.RUnlock()

			// Check if there are pending tasks
			if len(tasks) == 0 && len(c.semaphore) == 0 {
				// Give a moment for any in-flight work
				time.Sleep(500 * time.Millisecond)
				if len(tasks) == 0 && len(c.semaphore) == 0 {
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

	c.visitedMu.RLock()
	totalVisited := len(c.visited)
	c.visitedMu.RUnlock()

	return &CrawlResult{
		StartURL:     startURL,
		TotalVisited: totalVisited,
		BrokenLinks:  c.broken,
	}, nil
}

// worker processes URLs from the task channel
func (c *Crawler) worker(ctx context.Context, tasks chan urlTask, activeWorkers *sync.WaitGroup) {
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

// processURL fetches and processes a single URL
func (c *Crawler) processURL(ctx context.Context, task urlTask, tasks chan urlTask) {
	// Acquire semaphore
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return
	}

	// Check depth limit
	if c.config.MaxDepth > 0 && task.depth > c.config.MaxDepth {
		return
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", task.url, nil)
	if err != nil {
		c.addBrokenLink(task.sourceURL, task.url, 0, err.Error())
		return
	}

	req.Header.Set("User-Agent", "LinkChecker/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		if c.config.Verbose {
			PrintError(task.url, err.Error(), task.depth)
		}
		if task.sourceURL != "" {
			c.addBrokenLink(task.sourceURL, task.url, 0, err.Error())
		}
		return
	}
	defer resp.Body.Close()

	if c.config.Verbose {
		PrintProgress(task.url, resp.StatusCode, task.depth)
	}

	// Check for broken link
	if resp.StatusCode >= 400 {
		if task.sourceURL != "" {
			c.addBrokenLink(task.sourceURL, task.url, resp.StatusCode, "")
		} else {
			// The start URL itself is broken
			c.addBrokenLink(task.url, task.url, resp.StatusCode, "start URL returned error")
		}
		return
	}

	// Only parse HTML content for links
	contentType := resp.Header.Get("Content-Type")
	if !isHTML(contentType) {
		return
	}

	// Parse and extract links
	links := ExtractLinks(resp.Body, c.baseURL)

	// Queue new links
	for _, link := range links {
		if c.shouldVisit(link) {
			c.markVisited(link)

			// Try to send task, skip if channel is full
			select {
			case tasks <- urlTask{url: link, sourceURL: task.url, depth: task.depth + 1}:
			default:
				// Channel full, skip this link
			}
		}
	}
}

// markVisited marks a URL as visited (thread-safe)
func (c *Crawler) markVisited(url string) {
	c.visitedMu.Lock()
	c.visited[url] = true
	c.visitedMu.Unlock()
}

// shouldVisit checks if a URL should be visited
func (c *Crawler) shouldVisit(targetURL string) bool {
	// Check if it's an internal link
	if !IsSameDomain(targetURL, c.baseURL) {
		return false
	}

	// Check if already visited
	c.visitedMu.RLock()
	visited := c.visited[targetURL]
	c.visitedMu.RUnlock()

	return !visited
}

// addBrokenLink adds a broken link to the results (thread-safe)
func (c *Crawler) addBrokenLink(sourceURL, brokenURL string, statusCode int, errMsg string) {
	c.brokenMu.Lock()
	c.broken = append(c.broken, BrokenLink{
		SourceURL:  sourceURL,
		BrokenURL:  brokenURL,
		StatusCode: statusCode,
		Error:      errMsg,
	})
	c.brokenMu.Unlock()
}

// isHTML checks if the content type indicates HTML content
func isHTML(contentType string) bool {
	return len(contentType) >= 9 && contentType[:9] == "text/html" ||
		len(contentType) >= 21 && contentType[:21] == "application/xhtml+xml"
}
