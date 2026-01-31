package indexer

import (
	"bufio"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RobotsChecker checks URLs against robots.txt rules
type RobotsChecker struct {
	rules      []disallowRule
	loaded     bool
	loadError  error
}

type disallowRule struct {
	path string
}

// NewRobotsChecker creates a new robots.txt checker
func NewRobotsChecker() *RobotsChecker {
	return &RobotsChecker{}
}

// Load fetches and parses robots.txt from the given base URL
func (r *RobotsChecker) Load(baseURL *url.URL, timeout time.Duration) error {
	robotsURL := &url.URL{
		Scheme: baseURL.Scheme,
		Host:   baseURL.Host,
		Path:   "/robots.txt",
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(robotsURL.String())
	if err != nil {
		r.loadError = err
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// No robots.txt or error - assume everything is allowed
		r.loaded = true
		return nil
	}

	// Parse robots.txt
	scanner := bufio.NewScanner(resp.Body)
	inUserAgentAll := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split directive and value
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		directive := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch directive {
		case "user-agent":
			// Check if this applies to all bots
			inUserAgentAll = value == "*"
		case "disallow":
			if inUserAgentAll && value != "" {
				r.rules = append(r.rules, disallowRule{path: value})
			}
		}
	}

	r.loaded = true
	return scanner.Err()
}

// IsBlocked checks if a URL is blocked by robots.txt
func (r *RobotsChecker) IsBlocked(targetURL string) bool {
	if !r.loaded || len(r.rules) == 0 {
		return false
	}

	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	path := parsed.Path
	if path == "" {
		path = "/"
	}

	for _, rule := range r.rules {
		if matchesRule(path, rule.path) {
			return true
		}
	}

	return false
}

// GetRules returns the parsed disallow rules
func (r *RobotsChecker) GetRules() []string {
	rules := make([]string, len(r.rules))
	for i, rule := range r.rules {
		rules[i] = "Disallow: " + rule.path
	}
	return rules
}

// matchesRule checks if a path matches a robots.txt rule
func matchesRule(path, rule string) bool {
	// Handle wildcard at end
	if strings.HasSuffix(rule, "*") {
		prefix := strings.TrimSuffix(rule, "*")
		return strings.HasPrefix(path, prefix)
	}

	// Handle $ anchor (exact match)
	if strings.HasSuffix(rule, "$") {
		exact := strings.TrimSuffix(rule, "$")
		return path == exact
	}

	// Simple prefix match
	return strings.HasPrefix(path, rule)
}
