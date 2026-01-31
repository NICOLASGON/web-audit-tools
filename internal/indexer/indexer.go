package indexer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Config holds the indexer configuration
type Config struct {
	Concurrency    int
	Timeout        time.Duration
	MaxDepth       int
	Verbose        bool
	CheckRobotsTxt bool
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Concurrency:    10,
		Timeout:        10 * time.Second,
		MaxDepth:       0,
		Verbose:        false,
		CheckRobotsTxt: true,
	}
}

// Indexer analyzes link indexability
type Indexer struct {
	config        Config
	baseURL       *url.URL
	visited       map[string]bool
	visitedMu     sync.RWMutex
	result        *IndexerResult
	resultMu      sync.Mutex
	client        *http.Client
	semaphore     chan struct{}
	robotsChecker *RobotsChecker
	seenLinks     map[string]bool
	seenLinksMu   sync.Mutex
}

// New creates a new Indexer
func New(config Config) *Indexer {
	return &Indexer{
		config:        config,
		visited:       make(map[string]bool),
		seenLinks:     make(map[string]bool),
		semaphore:     make(chan struct{}, config.Concurrency),
		robotsChecker: NewRobotsChecker(),
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

type urlTask struct {
	url       string
	sourceURL string
	depth     int
}

// Analyze starts the indexability analysis
func (idx *Indexer) Analyze(startURL string) (*IndexerResult, error) {
	parsed, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	idx.baseURL = parsed
	idx.result = NewIndexerResult(startURL)

	// Load robots.txt if enabled
	if idx.config.CheckRobotsTxt {
		if idx.config.Verbose {
			fmt.Printf("%sLoading robots.txt...%s\n", colorGray, colorReset)
		}
		if err := idx.robotsChecker.Load(parsed, idx.config.Timeout); err != nil {
			if idx.config.Verbose {
				fmt.Printf("%sCould not load robots.txt: %v%s\n", colorYellow, err, colorReset)
			}
		} else {
			idx.result.RobotsTxtRules = idx.robotsChecker.GetRules()
			if idx.config.Verbose && len(idx.result.RobotsTxtRules) > 0 {
				fmt.Printf("%sFound %d robots.txt rules%s\n", colorGray, len(idx.result.RobotsTxtRules), colorReset)
			}
		}
	}

	tasks := make(chan urlTask, 1000)

	idx.markVisited(startURL)
	tasks <- urlTask{url: startURL, sourceURL: "", depth: 0}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < idx.config.Concurrency; i++ {
		go idx.worker(ctx, tasks)
	}

	done := make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			idx.visitedMu.RLock()
			visitedCount := len(idx.visited)
			idx.visitedMu.RUnlock()

			if len(tasks) == 0 && len(idx.semaphore) == 0 {
				time.Sleep(500 * time.Millisecond)
				if len(tasks) == 0 && len(idx.semaphore) == 0 {
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

	idx.visitedMu.RLock()
	idx.result.TotalPages = len(idx.visited)
	idx.visitedMu.RUnlock()

	idx.seenLinksMu.Lock()
	idx.result.TotalLinks = len(idx.seenLinks)
	idx.seenLinksMu.Unlock()

	idx.result.IndexableLinks = idx.result.TotalLinks - len(idx.result.NonIndexableLinks)

	return idx.result, nil
}

func (idx *Indexer) worker(ctx context.Context, tasks chan urlTask) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			idx.processURL(ctx, task, tasks)
		}
	}
}

func (idx *Indexer) processURL(ctx context.Context, task urlTask, tasks chan urlTask) {
	select {
	case idx.semaphore <- struct{}{}:
		defer func() { <-idx.semaphore }()
	case <-ctx.Done():
		return
	}

	if idx.config.MaxDepth > 0 && task.depth > idx.config.MaxDepth {
		return
	}

	req, err := http.NewRequestWithContext(ctx, "GET", task.url, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "LinkIndexer/1.0")

	resp, err := idx.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		if idx.config.Verbose {
			printError(task.url, err.Error(), task.depth)
		}
		return
	}
	defer resp.Body.Close()

	if idx.config.Verbose {
		printProgress(task.url, resp.StatusCode, task.depth)
	}

	// Check X-Robots-Tag header
	xRobotsTag := strings.ToLower(resp.Header.Get("X-Robots-Tag"))
	hasNoIndexHeader := strings.Contains(xRobotsTag, "noindex")

	if hasNoIndexHeader {
		idx.resultMu.Lock()
		idx.result.PagesWithNoIndex = append(idx.result.PagesWithNoIndex, task.url)
		idx.resultMu.Unlock()
	}

	if resp.StatusCode >= 400 {
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if !isHTML(contentType) {
		return
	}

	// Parse page
	pageInfo := ParsePage(resp.Body, idx.baseURL, task.url)

	// Track noindex pages
	if pageInfo.HasNoIndex {
		idx.resultMu.Lock()
		// Avoid duplicates
		found := false
		for _, p := range idx.result.PagesWithNoIndex {
			if p == task.url {
				found = true
				break
			}
		}
		if !found {
			idx.result.PagesWithNoIndex = append(idx.result.PagesWithNoIndex, task.url)
		}
		idx.resultMu.Unlock()
	}

	// Process each link
	for _, link := range pageInfo.Links {
		// Track unique links
		idx.seenLinksMu.Lock()
		linkKey := task.url + " -> " + link.URL
		if idx.seenLinks[linkKey] {
			idx.seenLinksMu.Unlock()
			continue
		}
		idx.seenLinks[linkKey] = true
		idx.seenLinksMu.Unlock()

		// Check indexability issues
		var reasons []NoIndexReason
		var details string

		if link.IsNoFollow || pageInfo.HasNoFollow {
			reasons = append(reasons, ReasonNoFollow)
		}
		if link.IsSponsored {
			reasons = append(reasons, ReasonSponsored)
		}
		if link.IsUGC {
			reasons = append(reasons, ReasonUGC)
		}

		// Check if target page has noindex (only for internal links we've seen)
		if IsSameDomain(link.URL, idx.baseURL) {
			if idx.config.CheckRobotsTxt && idx.robotsChecker.IsBlocked(link.URL) {
				reasons = append(reasons, ReasonRobotsTxt)
			}
		}

		if len(reasons) > 0 {
			idx.resultMu.Lock()
			idx.result.AddNonIndexable(NonIndexableLink{
				URL:       link.URL,
				SourceURL: task.url,
				Reasons:   reasons,
				Details:   details,
			})
			idx.resultMu.Unlock()
		}

		// Queue internal links for crawling
		if IsSameDomain(link.URL, idx.baseURL) && idx.shouldVisit(link.URL) {
			idx.markVisited(link.URL)
			select {
			case tasks <- urlTask{url: link.URL, sourceURL: task.url, depth: task.depth + 1}:
			default:
			}
		}
	}
}

func (idx *Indexer) markVisited(url string) {
	idx.visitedMu.Lock()
	idx.visited[url] = true
	idx.visitedMu.Unlock()
}

func (idx *Indexer) shouldVisit(targetURL string) bool {
	if !IsSameDomain(targetURL, idx.baseURL) {
		return false
	}

	idx.visitedMu.RLock()
	visited := idx.visited[targetURL]
	idx.visitedMu.RUnlock()

	return !visited
}

func isHTML(contentType string) bool {
	return strings.Contains(contentType, "text/html") ||
		strings.Contains(contentType, "application/xhtml+xml")
}

func printProgress(url string, statusCode int, depth int) {
	var statusColor string
	switch {
	case statusCode >= 200 && statusCode < 300:
		statusColor = colorGreen
	case statusCode >= 300 && statusCode < 400:
		statusColor = colorYellow
	case statusCode >= 400:
		statusColor = colorRed
	default:
		statusColor = colorReset
	}

	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s%s[%d]%s %s\n", indent, statusColor, statusCode, colorReset, url)
}

func printError(url string, err string, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s%s[ERR]%s %s - %s\n", indent, colorRed, colorReset, url, err)
}
