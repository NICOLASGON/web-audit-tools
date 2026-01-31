# Web Tools

A collection of command-line SEO and web analysis tools written in Go. These tools help you audit websites, detect issues, and improve search engine optimization.

## Tools

| Tool | Description |
|------|-------------|
| `linkchecker` | Detect broken links (404 errors) |
| `linkanalyzer` | Categorize non-analyzable links (external, mailto, files, etc.) |
| `linkindexer` | Find non-indexable links (nofollow, noindex, robots.txt) |
| `linklatency` | Measure page load times with visual bar graphs |
| `serpreview` | Preview how pages appear in Google search results |
| `linkcanonical` | Verify canonical URL configuration |
| `pagerank` | Calculate internal PageRank scores |
| `metacheck` | Check meta description lengths |
| `siteaudit` | Run a comprehensive SEO audit combining all tools |

## Installation

```bash
# Clone the repository
git clone https://github.com/ngonzalez/web-tools.git
cd web-tools

# Build all tools
go build -o linkchecker ./cmd/linkchecker
go build -o linkanalyzer ./cmd/linkanalyzer
go build -o linkindexer ./cmd/linkindexer
go build -o linklatency ./cmd/linklatency
go build -o serpreview ./cmd/serpreview
go build -o linkcanonical ./cmd/linkcanonical
go build -o pagerank ./cmd/pagerank
go build -o metacheck ./cmd/metacheck
go build -o siteaudit ./cmd/siteaudit

# Or build all at once
go build ./cmd/...
```

## Usage

### LinkChecker - Broken Link Detector

Crawls a website and detects broken links (404 errors).

```bash
./linkchecker [options] <url>

Options:
  -c, --concurrency int   Number of concurrent requests (default 10)
  -t, --timeout int       Request timeout in seconds (default 10)
  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)
  -v, --verbose           Show all visited URLs

Example:
  ./linkchecker https://example.com
  ./linkchecker -c 20 -d 3 -v https://example.com
```

### LinkAnalyzer - Non-Analyzable Links

Detects and categorizes links that cannot be crawled: external links, mailto, tel, JavaScript, file downloads, etc.

```bash
./linkanalyzer [options] <url>

Options:
  -c, --concurrency int   Number of concurrent requests (default 10)
  -t, --timeout int       Request timeout in seconds (default 10)
  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)
  -v, --verbose           Show all visited URLs
  -D, --details           Show detailed breakdown (default true)

Example:
  ./linkanalyzer https://example.com
  ./linkanalyzer -d 2 https://example.com
```

### LinkIndexer - Indexability Checker

Finds links that won't be indexed by search engines.

```bash
./linkindexer [options] <url>

Detects:
  - Links with rel="nofollow"
  - Links with rel="sponsored" or rel="ugc"
  - Pages with <meta name="robots" content="noindex">
  - Pages with X-Robots-Tag: noindex header
  - URLs blocked by robots.txt

Options:
  -c, --concurrency int   Number of concurrent requests (default 10)
  -t, --timeout int       Request timeout in seconds (default 10)
  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)
  -v, --verbose           Show all visited URLs
      --no-robots         Skip robots.txt checking

Example:
  ./linkindexer https://example.com
  ./linkindexer -d 3 --no-robots https://example.com
```

### LinkLatency - Performance Measurement

Measures page load times and displays results as a bar graph sorted by latency.

```bash
./linklatency [options] <url>

Options:
  -c, --concurrency int   Number of concurrent requests (default 10)
  -t, --timeout int       Request timeout in seconds (default 30)
  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)
  -v, --verbose           Show progress while crawling
  -w, --width int         Width of the bar graph (default 30)
  -s, --size              Show page sizes

Example:
  ./linklatency https://example.com
  ./linklatency -d 2 -s https://example.com
```

### SERPreview - Google Search Preview

Shows how a page will appear in Google search results and analyzes SEO metadata.

```bash
./serpreview [options] <url>

Analyzes:
  - Google search result preview (SERP snippet)
  - Title and meta description
  - Open Graph and Twitter Card tags
  - Canonical URL and robots directives
  - Schema.org structured data

Options:
  -t, --timeout int   Request timeout in seconds (default 30)
  -v, --verbose       Verbose output
  -a, --analysis      Show analysis only (no preview)
  -p, --preview       Show preview only (no analysis)

Example:
  ./serpreview https://example.com
  ./serpreview -a https://example.com
```

### LinkCanonical - Canonical URL Verifier

Verifies that all internal links point to canonical URLs.

```bash
./linkcanonical [options] <url>

Detects:
  - Links pointing to non-canonical URLs
  - Links causing redirects to canonical
  - Pages with missing canonical tags
  - Canonical URL mismatches
  - Canonical chains (A→B→C)

Options:
  -c, --concurrency int   Number of concurrent requests (default 10)
  -t, --timeout int       Request timeout in seconds (default 10)
  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)
  -v, --verbose           Show all visited URLs

Example:
  ./linkcanonical https://example.com
  ./linkcanonical -d 3 https://example.com
```

### PageRank - Internal PageRank Calculator

Calculates PageRank scores for all pages based on internal link structure.

```bash
./pagerank [options] <url>

Options:
  -c, --concurrency int   Number of concurrent requests (default 10)
  -t, --timeout int       Request timeout in seconds (default 10)
  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)
  -v, --verbose           Show crawl progress
  -n, --top int           Number of top pages to display (default 20)
      --damping float     Damping factor 0-1 (default 0.85)
      --iter int          Maximum iterations (default 100)
  -w, --width int         Bar graph width (default 20)

Example:
  ./pagerank https://example.com
  ./pagerank -n 50 -d 3 https://example.com
```

### MetaCheck - Meta Description Checker

Checks meta description lengths and identifies pages with descriptions that are too long, too short, missing, or duplicated.

```bash
./metacheck [options] <url>

Recommended meta description length: 70-155 characters
Ideal length: 120-155 characters

Options:
  -c, --concurrency int   Number of concurrent requests (default 10)
  -t, --timeout int       Request timeout in seconds (default 10)
  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)
  -v, --verbose           Show crawl progress
  -a, --all               Show all issues (including short and duplicates)
  -n, --limit int         Max pages per category (default 20)

Example:
  ./metacheck https://example.com
  ./metacheck -a -d 3 https://example.com
```

### SiteAudit - Comprehensive SEO Audit

Runs a complete SEO audit combining all tools and generates a detailed report with scores and recommendations.

```bash
./siteaudit [options] <url>

Performs:
  - Broken links detection (404 errors)
  - Non-analyzable links analysis
  - Indexability issues (nofollow, noindex, robots.txt)
  - Canonical URL verification
  - Performance measurement (page latency)
  - SEO analysis (title, description, OG tags, schema)
  - PageRank calculation (internal link structure)

Options:
  -c, --concurrency int   Number of concurrent requests (default 10)
  -t, --timeout int       Request timeout in seconds (default 15)
  -d, --depth int         Maximum crawl depth, 0 = unlimited (default 0)
  -v, --verbose           Show detailed progress

Example:
  ./siteaudit https://example.com
  ./siteaudit -d 3 -v https://example.com
```

#### Audit Scores

The audit generates scores in four categories:

- **Broken Links** (0-100): Penalizes broken links found on the site
- **SEO** (0-100): Checks title, meta description, canonical, H1, Open Graph, Twitter Cards, and Schema.org
- **Performance** (0-100): Penalizes slow pages (>1s) and very slow pages (>3s)
- **Architecture** (0-100): Penalizes orphan pages, dead-end pages, and canonical issues

The **Overall Score** is a weighted average of all four categories.

#### Exit Codes

| Tool | Exit Code | Meaning |
|------|-----------|---------|
| `linkchecker` | 1 | Broken links found |
| `linkcanonical` | 1 | Canonical issues found |
| `metacheck` | 1 | Too long or missing descriptions |
| `siteaudit` | 1 | Score < 70 |
| `siteaudit` | 2 | Score < 50 |

## Project Structure

```
web-tools/
├── cmd/
│   ├── linkchecker/      # Broken link detector CLI
│   ├── linkanalyzer/     # Non-analyzable links CLI
│   ├── linkindexer/      # Indexability checker CLI
│   ├── linklatency/      # Latency measurement CLI
│   ├── serpreview/       # SERP preview CLI
│   ├── linkcanonical/    # Canonical verifier CLI
│   ├── pagerank/         # PageRank calculator CLI
│   ├── metacheck/        # Meta description checker CLI
│   └── siteaudit/        # Comprehensive audit CLI
├── internal/
│   ├── crawler/          # Web crawler with link extraction
│   ├── analyzer/         # Link type categorization
│   ├── indexer/          # Indexability analysis
│   ├── latency/          # Performance measurement
│   ├── serp/             # SEO metadata extraction
│   ├── canonical/        # Canonical URL verification
│   ├── pagerank/         # PageRank algorithm
│   ├── metacheck/        # Meta description analysis
│   └── audit/            # Comprehensive audit orchestration
├── go.mod
└── README.md
```

## Requirements

- Go 1.21 or later

## Dependencies

- `golang.org/x/net/html` - HTML parsing

## License

MIT License
