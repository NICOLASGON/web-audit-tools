package metacheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// Config holds checker configuration
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
		Timeout:     10 * time.Second,
		MaxDepth:    0,
		Verbose:     false,
	}
}

// Checker analyzes meta descriptions
type Checker struct {
	config    Config
	baseURL   *url.URL
	visited   map[string]bool
	visitedMu sync.RWMutex
	result    *MetaResult
	resultMu  sync.Mutex
	client    *http.Client
	semaphore chan struct{}
}

// New creates a new Checker
func New(config Config) *Checker {
	return &Checker{
		config:    config,
		visited:   make(map[string]bool),
		semaphore: make(chan struct{}, config.Concurrency),
		client: &http.Client{
			Timeout: config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
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

// Check starts the meta description analysis
func (c *Checker) Check(startURL string) (*MetaResult, error) {
	parsed, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	c.baseURL = parsed
	c.result = NewMetaResult(startURL)

	tasks := make(chan urlTask, 1000)

	c.markVisited(startURL)
	tasks <- urlTask{url: startURL, depth: 0}

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

	c.result.Finalize()

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

	req, err := http.NewRequestWithContext(ctx, "GET", task.url, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "MetaChecker/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		if c.config.Verbose {
			printError(task.url, err.Error(), task.depth)
		}
		return
	}
	defer resp.Body.Close()

	if c.config.Verbose {
		printProgress(task.url, resp.StatusCode, task.depth)
	}

	if resp.StatusCode >= 400 {
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return
	}

	// Parse page
	pageMeta, links := c.parsePage(resp.Body, task.url)

	// Add to results
	c.resultMu.Lock()
	c.result.AddPage(pageMeta)
	c.resultMu.Unlock()

	// Queue new links
	for _, link := range links {
		if c.shouldVisit(link) {
			c.markVisited(link)
			select {
			case tasks <- urlTask{url: link, depth: task.depth + 1}:
			default:
			}
		}
	}
}

func (c *Checker) parsePage(body io.Reader, pageURL string) (PageMeta, []string) {
	meta := PageMeta{
		URL: pageURL,
	}
	var links []string

	doc, err := html.Parse(body)
	if err != nil {
		return meta, links
	}

	var parseNode func(*html.Node)
	parseNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					meta.Title = strings.TrimSpace(n.FirstChild.Data)
					meta.TitleLength = utf8.RuneCountInString(meta.Title)
				}

			case "meta":
				name := strings.ToLower(getAttr(n, "name"))
				if name == "description" {
					content := getAttr(n, "content")
					meta.Description = strings.TrimSpace(content)
					meta.DescLength = utf8.RuneCountInString(meta.Description)
				}

			case "a":
				href := getAttr(n, "href")
				if href != "" {
					resolved := c.resolveURL(href)
					if resolved != "" {
						links = append(links, resolved)
					}
				}
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			parseNode(child)
		}
	}

	parseNode(doc)
	return meta, links
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == key {
			return attr.Val
		}
	}
	return ""
}

func (c *Checker) resolveURL(href string) string {
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

	resolved := c.baseURL.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}

	// Only internal links
	if resolved.Host != c.baseURL.Host {
		return ""
	}

	resolved.Fragment = ""
	return resolved.String()
}

func (c *Checker) markVisited(url string) {
	c.visitedMu.Lock()
	c.visited[url] = true
	c.visitedMu.Unlock()
}

func (c *Checker) shouldVisit(targetURL string) bool {
	c.visitedMu.RLock()
	visited := c.visited[targetURL]
	c.visitedMu.RUnlock()
	return !visited
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

func printError(url string, errMsg string, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s%s[ERR]%s %s - %s\n", indent, colorRed, colorReset, url, errMsg)
}
