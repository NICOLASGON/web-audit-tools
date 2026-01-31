package latency

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// PageLatency holds timing info for a page
type PageLatency struct {
	URL        string
	Duration   time.Duration
	StatusCode int
	Size       int64
	Error      string
}

// LatencyResult holds all results
type LatencyResult struct {
	StartURL   string
	Pages      []PageLatency
	TotalTime  time.Duration
	StartTime  time.Time
	EndTime    time.Time
}

// NewLatencyResult creates a new result
func NewLatencyResult(startURL string) *LatencyResult {
	return &LatencyResult{
		StartURL:  startURL,
		StartTime: time.Now(),
	}
}

// AddPage adds a page timing
func (r *LatencyResult) AddPage(page PageLatency) {
	r.Pages = append(r.Pages, page)
}

// Finalize marks the end of the crawl
func (r *LatencyResult) Finalize() {
	r.EndTime = time.Now()
	r.TotalTime = r.EndTime.Sub(r.StartTime)
}

// SortByLatency sorts pages by duration (slowest first)
func (r *LatencyResult) SortByLatency() {
	sort.Slice(r.Pages, func(i, j int) bool {
		return r.Pages[i].Duration > r.Pages[j].Duration
	})
}

// Stats returns statistics
func (r *LatencyResult) Stats() (min, max, avg time.Duration) {
	if len(r.Pages) == 0 {
		return 0, 0, 0
	}

	min = r.Pages[0].Duration
	max = r.Pages[0].Duration
	var total time.Duration

	for _, p := range r.Pages {
		if p.Error != "" {
			continue
		}
		if p.Duration < min {
			min = p.Duration
		}
		if p.Duration > max {
			max = p.Duration
		}
		total += p.Duration
	}

	successCount := 0
	for _, p := range r.Pages {
		if p.Error == "" {
			successCount++
		}
	}

	if successCount > 0 {
		avg = total / time.Duration(successCount)
	}

	return min, max, avg
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// PrintSummary displays the results with bar graph
func (r *LatencyResult) PrintSummary(barWidth int, showSize bool) {
	fmt.Println()
	fmt.Printf("%s%s=== Latency Analysis ===%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Start URL: %s%s%s\n", colorBlue, r.StartURL, colorReset)
	fmt.Printf("Pages analyzed: %s%d%s\n", colorGreen, len(r.Pages), colorReset)
	fmt.Printf("Total crawl time: %s%v%s\n", colorYellow, r.TotalTime.Round(time.Millisecond), colorReset)

	min, max, avg := r.Stats()
	fmt.Println()
	fmt.Printf("%s%sStatistics:%s\n", colorBold, colorYellow, colorReset)
	fmt.Printf("  Fastest: %s%v%s\n", colorGreen, min.Round(time.Millisecond), colorReset)
	fmt.Printf("  Slowest: %s%v%s\n", colorRed, max.Round(time.Millisecond), colorReset)
	fmt.Printf("  Average: %s%v%s\n", colorYellow, avg.Round(time.Millisecond), colorReset)

	// Sort by latency (slowest first)
	r.SortByLatency()

	fmt.Println()
	fmt.Printf("%s%sPages by Load Time (slowest first):%s\n", colorBold, colorPurple, colorReset)
	fmt.Println()

	// Find max duration for scaling
	var maxDuration time.Duration
	for _, p := range r.Pages {
		if p.Duration > maxDuration {
			maxDuration = p.Duration
		}
	}

	// Calculate URL column width (use remaining space)
	maxURLWidth := 60

	for _, p := range r.Pages {
		r.printPageBar(p, maxDuration, barWidth, maxURLWidth, showSize)
	}

	// Distribution histogram
	r.printDistribution()

	fmt.Println()
}

func (r *LatencyResult) printPageBar(p PageLatency, maxDuration time.Duration, barWidth, maxURLWidth int, showSize bool) {
	// Truncate URL if needed
	url := p.URL
	if len(url) > maxURLWidth {
		url = url[:maxURLWidth-3] + "..."
	}

	// Calculate bar length
	var barLen int
	if maxDuration > 0 {
		barLen = int(float64(barWidth) * float64(p.Duration) / float64(maxDuration))
	}
	if barLen < 1 && p.Error == "" {
		barLen = 1
	}

	// Choose color based on latency
	var barColor string
	ms := p.Duration.Milliseconds()
	switch {
	case p.Error != "":
		barColor = colorRed
	case ms < 200:
		barColor = colorGreen
	case ms < 500:
		barColor = colorYellow
	case ms < 1000:
		barColor = colorPurple
	default:
		barColor = colorRed
	}

	// Build bar
	bar := strings.Repeat("█", barLen)
	emptyBar := strings.Repeat("░", barWidth-barLen)

	// Format duration
	durationStr := fmt.Sprintf("%7v", p.Duration.Round(time.Millisecond))

	// Status indicator
	var status string
	if p.Error != "" {
		status = fmt.Sprintf("%s[ERR]%s", colorRed, colorReset)
		durationStr = fmt.Sprintf("%7s", "---")
	} else if p.StatusCode >= 400 {
		status = fmt.Sprintf("%s[%d]%s", colorRed, p.StatusCode, colorReset)
	} else if p.StatusCode >= 300 {
		status = fmt.Sprintf("%s[%d]%s", colorYellow, p.StatusCode, colorReset)
	} else {
		status = fmt.Sprintf("%s[%d]%s", colorGreen, p.StatusCode, colorReset)
	}

	// Size info
	sizeStr := ""
	if showSize && p.Size > 0 {
		sizeStr = fmt.Sprintf(" %s(%s)%s", colorGray, formatSize(p.Size), colorReset)
	}

	fmt.Printf("%s %s%s%s%s %s %-*s%s\n",
		status,
		barColor, bar, colorGray, emptyBar,
		durationStr,
		maxURLWidth, url,
		sizeStr,
	)
}

func (r *LatencyResult) printDistribution() {
	if len(r.Pages) < 5 {
		return
	}

	fmt.Println()
	fmt.Printf("%s%sLatency Distribution:%s\n", colorBold, colorYellow, colorReset)

	// Define buckets
	buckets := []struct {
		label string
		maxMs int64
		color string
	}{
		{"< 100ms", 100, colorGreen},
		{"100-200ms", 200, colorGreen},
		{"200-500ms", 500, colorYellow},
		{"500ms-1s", 1000, colorPurple},
		{"> 1s", -1, colorRed},
	}

	counts := make([]int, len(buckets))

	for _, p := range r.Pages {
		if p.Error != "" {
			continue
		}
		ms := p.Duration.Milliseconds()
		for i, b := range buckets {
			if b.maxMs == -1 || ms < b.maxMs {
				counts[i]++
				break
			}
		}
	}

	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	barWidth := 30
	for i, b := range buckets {
		if counts[i] == 0 {
			continue
		}
		barLen := 1
		if maxCount > 0 {
			barLen = int(float64(barWidth) * float64(counts[i]) / float64(maxCount))
			if barLen < 1 {
				barLen = 1
			}
		}

		bar := strings.Repeat("█", barLen)
		fmt.Printf("  %s%-12s%s %s%s%s %d\n",
			colorGray, b.label, colorReset,
			b.color, bar, colorReset,
			counts[i],
		)
	}
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
	)

	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
