package mangascraper

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// baseSource provides common functionality for all sources
type baseSource struct {
	name        string
	displayName string
	baseURL     string
	config      *Config
	httpClient  *http.Client
}

func newBaseSource(name, displayName, baseURL string, config *Config) *baseSource {
	if config == nil {
		config = DefaultConfig()
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		}
	}

	return &baseSource{
		name:        name,
		displayName: displayName,
		baseURL:     baseURL,
		config:      config,
		httpClient:  httpClient,
	}
}

func (s *baseSource) Name() string        { return s.name }
func (s *baseSource) DisplayName() string { return s.displayName }
func (s *baseSource) BaseURL() string     { return s.baseURL }

// makeRequest makes an HTTP request with appropriate headers
func (s *baseSource) makeRequest(urlStr string) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(s.config.RetryDelay)
		}

		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", s.config.UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Set("Referer", s.baseURL)
		req.Header.Set("Cache-Control", "no-cache")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		_ = resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return nil, fmt.Errorf("max retries exceeded: %v", lastErr)
}

// fetchDocument fetches and parses HTML document
func (s *baseSource) fetchDocument(urlStr string) (*goquery.Document, error) {
	resp, err := s.makeRequest(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return goquery.NewDocumentFromReader(resp.Body)
}

// Helper functions

// normalizeURL ensures URL is absolute
func normalizeURL(urlStr, baseURL string) string {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return ""
	}

	if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		return urlStr
	}

	if strings.HasPrefix(urlStr, "//") {
		return "https:" + urlStr
	}

	if strings.HasPrefix(urlStr, "/") {
		return strings.TrimSuffix(baseURL, "/") + urlStr
	}

	return baseURL + "/" + urlStr
}

// normalizeImageURL normalizes image URLs
func normalizeImageURL(imgURL, baseURL string) string {
	if imgURL == "" {
		return ""
	}

	imgURL = strings.TrimSpace(imgURL)

	// Remove data: URLs
	if strings.HasPrefix(imgURL, "data:") {
		return ""
	}

	return normalizeURL(imgURL, baseURL)
}

// getImageSrc extracts image source from multiple attributes
func getImageSrc(img *goquery.Selection) string {
	// Priority: data-src, data-lazy-src, src
	for _, attr := range []string{"data-src", "data-lazy-src", "data-original", "src"} {
		if src, exists := img.Attr(attr); exists {
			src = strings.TrimSpace(src)
			if src != "" && !strings.HasPrefix(src, "data:") {
				return src
			}
		}
	}
	return ""
}

// extractMangaID extracts manga ID from URL
func extractMangaID(mangaURL string) string {
	// Remove trailing slash and query params
	urlStr := strings.Split(mangaURL, "?")[0]
	urlStr = strings.TrimSuffix(urlStr, "/")

	// Get last part of path
	parts := strings.Split(urlStr, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if part != "" && part != "manga" {
			return part
		}
	}

	return ""
}

// parseChapterNumber extracts float number from chapter string
func parseChapterNumber(chapterStr string) float64 {
	// Remove common prefixes
	chapterStr = strings.TrimSpace(chapterStr)
	chapterStr = strings.TrimPrefix(strings.ToLower(chapterStr), "capítulo")
	chapterStr = strings.TrimPrefix(strings.ToLower(chapterStr), "capitulo")
	chapterStr = strings.TrimPrefix(strings.ToLower(chapterStr), "cap.")
	chapterStr = strings.TrimPrefix(strings.ToLower(chapterStr), "cap")
	chapterStr = strings.TrimPrefix(chapterStr, ".")
	chapterStr = strings.TrimSpace(chapterStr)

	// Extract number
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(chapterStr)
	if len(matches) > 1 {
		if num, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return num
		}
	}

	return 0
}

// containsString checks if slice contains a string
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
