package latency

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

// Config holds the configuration
type Config struct {
	Concurrency int
	Timeout     time.Duration
	MaxDepth    int
	Verbose     bool
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Concurrency: 10,
		Timeout:     30 * time.Second,
		MaxDepth:    0,
		Verbose:     false,
	}
}

// Measurer measures page latencies
type Measurer struct {
	config    Config
	baseURL   *url.URL
	visited   map[string]bool
	visitedMu sync.RWMutex
	result    *LatencyResult
	resultMu  sync.Mutex
	client    *http.Client
	semaphore chan struct{}
}

// New creates a new Measurer
func New(config Config) *Measurer {
	return &Measurer{
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
	url   string
	depth int
}

// Measure starts measuring latencies
func (m *Measurer) Measure(startURL string) (*LatencyResult, error) {
	parsed, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	m.baseURL = parsed
	m.result = NewLatencyResult(startURL)

	tasks := make(chan urlTask, 1000)

	m.markVisited(startURL)
	tasks <- urlTask{url: startURL, depth: 0}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < m.config.Concurrency; i++ {
		go m.worker(ctx, tasks)
	}

	done := make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			m.visitedMu.RLock()
			visitedCount := len(m.visited)
			m.visitedMu.RUnlock()

			if len(tasks) == 0 && len(m.semaphore) == 0 {
				time.Sleep(500 * time.Millisecond)
				if len(tasks) == 0 && len(m.semaphore) == 0 {
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

	m.result.Finalize()
	return m.result, nil
}

func (m *Measurer) worker(ctx context.Context, tasks chan urlTask) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			m.processURL(ctx, task, tasks)
		}
	}
}

func (m *Measurer) processURL(ctx context.Context, task urlTask, tasks chan urlTask) {
	select {
	case m.semaphore <- struct{}{}:
		defer func() { <-m.semaphore }()
	case <-ctx.Done():
		return
	}

	if m.config.MaxDepth > 0 && task.depth > m.config.MaxDepth {
		return
	}

	req, err := http.NewRequestWithContext(ctx, "GET", task.url, nil)
	if err != nil {
		m.addResult(PageLatency{URL: task.url, Error: err.Error()})
		return
	}

	req.Header.Set("User-Agent", "LinkLatency/1.0")

	// Measure timing
	start := time.Now()
	resp, err := m.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		if ctx.Err() != nil {
			return
		}
		m.addResult(PageLatency{
			URL:      task.url,
			Duration: duration,
			Error:    err.Error(),
		})
		if m.config.Verbose {
			printError(task.url, err.Error(), task.depth)
		}
		return
	}

	// Read body to get size and complete timing
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	duration = time.Since(start)

	pageLatency := PageLatency{
		URL:        task.url,
		Duration:   duration,
		StatusCode: resp.StatusCode,
		Size:       int64(len(body)),
	}

	m.addResult(pageLatency)

	if m.config.Verbose {
		printProgress(task.url, resp.StatusCode, duration, task.depth)
	}

	if resp.StatusCode >= 400 {
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if !isHTML(contentType) {
		return
	}

	// Extract links
	links := extractLinks(strings.NewReader(string(body)), m.baseURL)

	for _, link := range links {
		if m.shouldVisit(link) {
			m.markVisited(link)
			select {
			case tasks <- urlTask{url: link, depth: task.depth + 1}:
			default:
			}
		}
	}
}

func (m *Measurer) addResult(page PageLatency) {
	m.resultMu.Lock()
	m.result.AddPage(page)
	m.resultMu.Unlock()
}

func (m *Measurer) markVisited(url string) {
	m.visitedMu.Lock()
	m.visited[url] = true
	m.visitedMu.Unlock()
}

func (m *Measurer) shouldVisit(targetURL string) bool {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	if parsed.Host != m.baseURL.Host {
		return false
	}

	m.visitedMu.RLock()
	visited := m.visited[targetURL]
	m.visitedMu.RUnlock()

	return !visited
}

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

func normalizeURL(href string, baseURL *url.URL) string {
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

	resolved.Fragment = ""
	return resolved.String()
}

func isHTML(contentType string) bool {
	return strings.Contains(contentType, "text/html") ||
		strings.Contains(contentType, "application/xhtml+xml")
}

func printProgress(url string, statusCode int, duration time.Duration, depth int) {
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
	fmt.Printf("%s%s[%d]%s %v %s\n", indent, statusColor, statusCode, colorReset, duration.Round(time.Millisecond), url)
}

func printError(url string, err string, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s%s[ERR]%s %s - %s\n", indent, colorRed, colorReset, url, err)
}
