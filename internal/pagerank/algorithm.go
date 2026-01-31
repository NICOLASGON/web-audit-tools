package pagerank

import (
	"math"
)

// Config for PageRank computation
type ComputeConfig struct {
	DampingFactor float64 // Usually 0.85
	MaxIterations int     // Maximum iterations
	Tolerance     float64 // Convergence threshold
}

// DefaultComputeConfig returns default computation settings
func DefaultComputeConfig() ComputeConfig {
	return ComputeConfig{
		DampingFactor: 0.85,
		MaxIterations: 100,
		Tolerance:     1e-6,
	}
}

// Compute calculates PageRank scores for all pages in the graph
func Compute(graph *Graph, config ComputeConfig) ([]float64, int, bool) {
	n := graph.Size()
	if n == 0 {
		return nil, 0, true
	}

	d := config.DampingFactor

	// Initialize scores: each page starts with 1/n
	scores := make([]float64, n)
	initialScore := 1.0 / float64(n)
	for i := range scores {
		scores[i] = initialScore
	}

	// Precompute teleport value
	teleport := (1.0 - d) / float64(n)

	// Iterative computation
	newScores := make([]float64, n)
	var iterations int
	var converged bool

	for iter := 0; iter < config.MaxIterations; iter++ {
		iterations = iter + 1

		// Handle dangling nodes (pages with no outgoing links)
		// Their PageRank is distributed evenly to all pages
		var danglingSum float64
		for i := 0; i < n; i++ {
			if graph.OutDegree[i] == 0 {
				danglingSum += scores[i]
			}
		}
		danglingContribution := d * danglingSum / float64(n)

		// Calculate new scores
		for i := 0; i < n; i++ {
			// Start with teleport probability + dangling contribution
			newScores[i] = teleport + danglingContribution

			// Add contributions from incoming links
			for _, j := range graph.InLinks[i] {
				if graph.OutDegree[j] > 0 {
					newScores[i] += d * scores[j] / float64(graph.OutDegree[j])
				}
			}
		}

		// Check convergence (L1 norm of difference)
		var diff float64
		for i := 0; i < n; i++ {
			diff += math.Abs(newScores[i] - scores[i])
		}

		// Swap slices
		scores, newScores = newScores, scores

		if diff < config.Tolerance {
			converged = true
			break
		}
	}

	return scores, iterations, converged
}

// ComputeWithResult calculates PageRank and returns a full result
func ComputeWithResult(graph *Graph, config ComputeConfig, startURL string) *PageRankResult {
	scores, iterations, converged := Compute(graph, config)

	// Count total links
	totalLinks := 0
	for _, outLinks := range graph.OutLinks {
		totalLinks += len(outLinks)
	}

	// Build result
	result := &PageRankResult{
		StartURL:      startURL,
		TotalPages:    graph.Size(),
		TotalLinks:    totalLinks,
		Iterations:    iterations,
		Converged:     converged,
		DampingFactor: config.DampingFactor,
		Scores:        make([]PageScore, graph.Size()),
	}

	for i, url := range graph.Indices {
		result.Scores[i] = PageScore{
			URL:      url,
			Score:    scores[i],
			InLinks:  len(graph.InLinks[i]),
			OutLinks: len(graph.OutLinks[i]),
		}
	}

	return result
}
