package pagerank

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// PageScore represents a page and its PageRank score
type PageScore struct {
	URL         string
	Score       float64
	InLinks     int // Number of incoming links
	OutLinks    int // Number of outgoing links
}

// Graph represents the link graph
type Graph struct {
	Pages      map[string]int      // URL -> index
	Indices    []string            // index -> URL
	OutLinks   [][]int             // adjacency list (outgoing)
	InLinks    [][]int             // adjacency list (incoming)
	OutDegree  []int               // number of outgoing links per page
}

// NewGraph creates a new graph
func NewGraph() *Graph {
	return &Graph{
		Pages: make(map[string]int),
	}
}

// AddPage adds a page to the graph
func (g *Graph) AddPage(url string) int {
	if idx, exists := g.Pages[url]; exists {
		return idx
	}

	idx := len(g.Indices)
	g.Pages[url] = idx
	g.Indices = append(g.Indices, url)
	g.OutLinks = append(g.OutLinks, nil)
	g.InLinks = append(g.InLinks, nil)
	g.OutDegree = append(g.OutDegree, 0)

	return idx
}

// AddLink adds a directed link from -> to
func (g *Graph) AddLink(from, to string) {
	fromIdx := g.AddPage(from)
	toIdx := g.AddPage(to)

	// Check if link already exists
	for _, existing := range g.OutLinks[fromIdx] {
		if existing == toIdx {
			return
		}
	}

	g.OutLinks[fromIdx] = append(g.OutLinks[fromIdx], toIdx)
	g.InLinks[toIdx] = append(g.InLinks[toIdx], fromIdx)
	g.OutDegree[fromIdx]++
}

// Size returns the number of pages
func (g *Graph) Size() int {
	return len(g.Indices)
}

// PageRankResult holds the computation results
type PageRankResult struct {
	StartURL    string
	TotalPages  int
	TotalLinks  int
	Iterations  int
	Converged   bool
	DampingFactor float64
	Scores      []PageScore
}

// ANSI colors
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

// PrintSummary displays the PageRank results
func (r *PageRankResult) PrintSummary(topN int, barWidth int) {
	fmt.Println()
	fmt.Printf("%s%s=== PageRank Analysis ===%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Start URL: %s%s%s\n", colorBlue, r.StartURL, colorReset)
	fmt.Printf("Pages analyzed: %s%d%s\n", colorGreen, r.TotalPages, colorReset)
	fmt.Printf("Internal links: %s%d%s\n", colorGreen, r.TotalLinks, colorReset)
	fmt.Printf("Damping factor: %s%.2f%s\n", colorYellow, r.DampingFactor, colorReset)
	fmt.Printf("Iterations: %s%d%s", colorYellow, r.Iterations, colorReset)
	if r.Converged {
		fmt.Printf(" %s(converged)%s\n", colorGreen, colorReset)
	} else {
		fmt.Printf(" %s(max reached)%s\n", colorYellow, colorReset)
	}

	// Statistics
	if len(r.Scores) > 0 {
		var sum, max, min float64
		min = r.Scores[0].Score
		for _, s := range r.Scores {
			sum += s.Score
			if s.Score > max {
				max = s.Score
			}
			if s.Score < min {
				min = s.Score
			}
		}
		avg := sum / float64(len(r.Scores))

		fmt.Println()
		fmt.Printf("%s%sStatistics:%s\n", colorBold, colorYellow, colorReset)
		fmt.Printf("  Max score: %s%.6f%s\n", colorGreen, max, colorReset)
		fmt.Printf("  Min score: %s%.6f%s\n", colorRed, min, colorReset)
		fmt.Printf("  Avg score: %s%.6f%s\n", colorYellow, avg, colorReset)
		fmt.Printf("  Sum: %s%.4f%s (should be ~1.0)\n", colorGray, sum, colorReset)
	}

	// Sort by score descending
	sorted := make([]PageScore, len(r.Scores))
	copy(sorted, r.Scores)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	// Display top N
	displayCount := topN
	if displayCount <= 0 || displayCount > len(sorted) {
		displayCount = len(sorted)
	}

	fmt.Println()
	fmt.Printf("%s%sTop %d pages by PageRank:%s\n", colorBold, colorPurple, displayCount, colorReset)
	fmt.Println()

	// Find max score for scaling
	maxScore := sorted[0].Score

	// Header
	fmt.Printf("%s%3s  %-8s  %-*s  %s   %s%s\n",
		colorGray,
		"#",
		"Score",
		barWidth, "PageRank",
		"In",
		"URL",
		colorReset)
	fmt.Printf("%s%s%s\n", colorGray, strings.Repeat("─", 80), colorReset)

	for i := 0; i < displayCount; i++ {
		page := sorted[i]
		r.printPageBar(i+1, page, maxScore, barWidth)
	}

	if len(sorted) > displayCount {
		fmt.Println()
		fmt.Printf("%s... and %d more pages%s\n", colorGray, len(sorted)-displayCount, colorReset)
	}

	// Show pages with highest incoming links
	r.printTopByInLinks(sorted, 5)

	// Show potential issues
	r.printIssues(sorted)

	fmt.Println()
}

func (r *PageRankResult) printPageBar(rank int, page PageScore, maxScore float64, barWidth int) {
	// Calculate bar length
	barLen := int(math.Round(float64(barWidth) * page.Score / maxScore))
	if barLen < 1 {
		barLen = 1
	}

	// Color based on relative score
	var barColor string
	ratio := page.Score / maxScore
	switch {
	case ratio >= 0.8:
		barColor = colorGreen
	case ratio >= 0.5:
		barColor = colorYellow
	case ratio >= 0.2:
		barColor = colorPurple
	default:
		barColor = colorGray
	}

	bar := strings.Repeat("█", barLen)
	emptyBar := strings.Repeat("░", barWidth-barLen)

	// Truncate URL
	url := page.URL
	maxURLLen := 50
	if len(url) > maxURLLen {
		url = url[:maxURLLen-3] + "..."
	}

	fmt.Printf("%s%3d%s  %s%.6f%s  %s%s%s%s  %s%3d%s  %s\n",
		colorYellow, rank, colorReset,
		colorCyan, page.Score, colorReset,
		barColor, bar, colorGray, emptyBar,
		colorBlue, page.InLinks, colorReset,
		url)
}

func (r *PageRankResult) printTopByInLinks(sorted []PageScore, topN int) {
	// Sort by incoming links
	byInLinks := make([]PageScore, len(sorted))
	copy(byInLinks, sorted)
	sort.Slice(byInLinks, func(i, j int) bool {
		return byInLinks[i].InLinks > byInLinks[j].InLinks
	})

	if len(byInLinks) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("%s%sTop %d pages by incoming links:%s\n", colorBold, colorYellow, topN, colorReset)

	displayCount := topN
	if displayCount > len(byInLinks) {
		displayCount = len(byInLinks)
	}

	for i := 0; i < displayCount; i++ {
		page := byInLinks[i]
		url := page.URL
		if len(url) > 60 {
			url = url[:57] + "..."
		}
		fmt.Printf("  %s%3d links%s  %s\n", colorBlue, page.InLinks, colorReset, url)
	}
}

func (r *PageRankResult) printIssues(sorted []PageScore) {
	// Find orphan pages (no incoming links)
	var orphans []string
	for _, page := range sorted {
		if page.InLinks == 0 {
			orphans = append(orphans, page.URL)
		}
	}

	// Find dead ends (no outgoing links)
	var deadEnds []string
	for _, page := range sorted {
		if page.OutLinks == 0 {
			deadEnds = append(deadEnds, page.URL)
		}
	}

	if len(orphans) > 0 || len(deadEnds) > 0 {
		fmt.Println()
		fmt.Printf("%s%sPotential issues:%s\n", colorBold, colorRed, colorReset)

		if len(orphans) > 0 {
			fmt.Printf("\n  %sOrphan pages%s (no incoming links): %d\n", colorYellow, colorReset, len(orphans))
			for i, url := range orphans {
				if i >= 3 {
					fmt.Printf("    %s... and %d more%s\n", colorGray, len(orphans)-3, colorReset)
					break
				}
				if len(url) > 60 {
					url = url[:57] + "..."
				}
				fmt.Printf("    • %s\n", url)
			}
		}

		if len(deadEnds) > 0 {
			fmt.Printf("\n  %sDead-end pages%s (no outgoing links): %d\n", colorYellow, colorReset, len(deadEnds))
			for i, url := range deadEnds {
				if i >= 3 {
					fmt.Printf("    %s... and %d more%s\n", colorGray, len(deadEnds)-3, colorReset)
					break
				}
				if len(url) > 60 {
					url = url[:57] + "..."
				}
				fmt.Printf("    • %s\n", url)
			}
		}
	}
}
