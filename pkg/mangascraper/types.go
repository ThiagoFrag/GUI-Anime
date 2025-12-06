// Package mangascraper provides a portable manga scraping library.
// It supports multiple sources and can be easily integrated into other projects.
//
// Basic usage:
//
//	scraper := mangascraper.New()
//	mangas, totalPages, err := scraper.GetAllMangas("mangalivre.blog", 1)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// For more examples, see the README.md file.
package mangascraper

import (
	"net/http"
	"time"
)

// Manga represents a manga with its metadata
type Manga struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Image       string   `json:"image"`
	URL         string   `json:"url"`
	LatestChap  string   `json:"latestChapter"`
	Genres      []string `json:"genres"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Rating      float64  `json:"rating"`
	Views       int      `json:"views"`
	Author      string   `json:"author"`
	Source      string   `json:"source"` // Which source this manga came from
}

// Chapter represents a manga chapter
type Chapter struct {
	Number      string  `json:"number"`
	NumberFloat float64 `json:"numberFloat"` // For proper sorting
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Date        string  `json:"date"`
	MangaID     string  `json:"mangaId"`
	MangaName   string  `json:"mangaName"`
}

// Page represents a manga page (single image)
type Page struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// Source represents a manga source/provider
type Source interface {
	// Name returns the unique identifier of the source
	Name() string

	// DisplayName returns a human-readable name
	DisplayName() string

	// BaseURL returns the base URL of the source
	BaseURL() string

	// GetAllMangas returns paginated list of all mangas
	GetAllMangas(page int) ([]Manga, int, error)

	// GetPopularMangas returns popular mangas
	GetPopularMangas() ([]Manga, error)

	// GetLatestUpdates returns recently updated mangas
	GetLatestUpdates() ([]Manga, error)

	// SearchManga searches for mangas by query
	SearchManga(query string) ([]Manga, error)

	// GetMangaDetails returns detailed information about a manga
	GetMangaDetails(mangaURL string) (*Manga, error)

	// GetChapters returns all chapters of a manga
	GetChapters(mangaURL string) ([]Chapter, error)

	// GetChapterPages returns all pages/images of a chapter
	GetChapterPages(chapterURL string) ([]Page, error)

	// GetMangasByGenre returns mangas filtered by genre
	GetMangasByGenre(genre string) ([]Manga, error)

	// GetGenres returns available genres
	GetGenres() ([]string, error)
}

// Config holds configuration options for the scraper
type Config struct {
	// HTTPClient allows using a custom HTTP client
	HTTPClient *http.Client

	// Timeout for HTTP requests (default: 30s)
	Timeout time.Duration

	// UserAgent to use for requests
	UserAgent string

	// EnableCache enables/disables caching (default: true)
	EnableCache bool

	// CacheDir directory to store cache files (default: ~/.mangascraper/cache)
	CacheDir string

	// CacheTTL default cache time-to-live
	CacheTTL time.Duration

	// MaxRetries number of retries for failed requests
	MaxRetries int

	// RetryDelay delay between retries
	RetryDelay time.Duration

	// RateLimit requests per second (0 = no limit)
	RateLimit float64
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:     30 * time.Second,
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		EnableCache: true,
		CacheTTL:    30 * time.Minute,
		MaxRetries:  3,
		RetryDelay:  1 * time.Second,
		RateLimit:   5, // 5 requests per second
	}
}

// SourceInfo contains metadata about a source
type SourceInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	BaseURL     string `json:"baseUrl"`
	Language    string `json:"language"`
	NSFW        bool   `json:"nsfw"`
}

// SearchResult contains search results with source info
type SearchResult struct {
	Mangas []Manga `json:"mangas"`
	Source string  `json:"source"`
	Error  error   `json:"error,omitempty"`
}
