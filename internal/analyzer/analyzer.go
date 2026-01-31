package analyzer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Config holds the analyzer configuration
type Config struct {
	Concurrency int
	Timeout     time.Duration
	MaxDepth    int
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

// Analyzer crawls a website and categorizes all links
type Analyzer struct {
	config    Config
	baseURL   *url.URL
	visited   map[string]bool
	visitedMu sync.RWMutex
	result    *AnalysisResult
	resultMu  sync.Mutex
	client    *http.Client
	semaphore chan struct{}
}

// New creates a new Analyzer instance
func New(config Config) *Analyzer {
	return &Analyzer{
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

type urlTask struct {
	url       string
	sourceURL string
	depth     int
}

// Analyze starts analyzing from the given URL
func (a *Analyzer) Analyze(startURL string) (*AnalysisResult, error) {
	parsed, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	a.baseURL = parsed
	a.result = NewAnalysisResult(startURL)

	tasks := make(chan urlTask, 1000)

	a.markVisited(startURL)
	tasks <- urlTask{url: startURL, sourceURL: "", depth: 0}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start workers
	for i := 0; i < a.config.Concurrency; i++ {
		go a.worker(ctx, tasks)
	}

	// Monitor completion
	done := make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			a.visitedMu.RLock()
			visitedCount := len(a.visited)
			a.visitedMu.RUnlock()

			if len(tasks) == 0 && len(a.semaphore) == 0 {
				time.Sleep(500 * time.Millisecond)
				if len(tasks) == 0 && len(a.semaphore) == 0 {
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

	a.visitedMu.RLock()
	a.result.TotalPages = len(a.visited)
	a.visitedMu.RUnlock()

	return a.result, nil
}

func (a *Analyzer) worker(ctx context.Context, tasks chan urlTask) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			a.processURL(ctx, task, tasks)
		}
	}
}

func (a *Analyzer) processURL(ctx context.Context, task urlTask, tasks chan urlTask) {
	select {
	case a.semaphore <- struct{}{}:
		defer func() { <-a.semaphore }()
	case <-ctx.Done():
		return
	}

	if a.config.MaxDepth > 0 && task.depth > a.config.MaxDepth {
		return
	}

	req, err := http.NewRequestWithContext(ctx, "GET", task.url, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "LinkAnalyzer/1.0")

	resp, err := a.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		if a.config.Verbose {
			printError(task.url, err.Error(), task.depth)
		}
		return
	}
	defer resp.Body.Close()

	if a.config.Verbose {
		printProgress(task.url, resp.StatusCode, task.depth)
	}

	if resp.StatusCode >= 400 {
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if !isHTML(contentType) {
		return
	}

	// Extract and classify all links
	links := ExtractAllLinks(resp.Body, a.baseURL, task.url)

	for _, link := range links {
		a.resultMu.Lock()
		a.result.AddLink(link)
		a.resultMu.Unlock()

		// Only queue internal HTML links for further crawling
		if link.Type == LinkTypeInternal && a.shouldVisit(link.URL) {
			a.markVisited(link.URL)
			select {
			case tasks <- urlTask{url: link.URL, sourceURL: task.url, depth: task.depth + 1}:
			default:
			}
		}
	}
}

func (a *Analyzer) markVisited(url string) {
	a.visitedMu.Lock()
	a.visited[url] = true
	a.visitedMu.Unlock()
}

func (a *Analyzer) shouldVisit(targetURL string) bool {
	if !IsSameDomain(targetURL, a.baseURL) {
		return false
	}

	a.visitedMu.RLock()
	visited := a.visited[targetURL]
	a.visitedMu.RUnlock()

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
