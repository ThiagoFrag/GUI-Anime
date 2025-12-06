# MangaScraper

A portable Go library for scraping manga from multiple sources. Designed to be easily integrated into any Go project.

## Features

- 🔌 **Multi-source support**: Currently supports MangaLivre.to and MangaLivre.blog
- 🚀 **Easy to use**: Simple API with sensible defaults
- 💾 **Built-in caching**: Persistent disk cache with configurable TTL
- ⚙️ **Configurable**: Customize timeouts, retries, rate limiting, and more
- 🔄 **Auto source detection**: Automatically detects the source from URLs
- 🔒 **Thread-safe**: Safe for concurrent use
- 📦 **Zero external dependencies** (except goquery for HTML parsing)

## Installation

```bash
go get github.com/yourusername/mangascraper
```

Or if using within this project:

```go
import "GoAnimeGUI/pkg/mangascraper"
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "GoAnimeGUI/pkg/mangascraper"
)

func main() {
    // Create a new scraper with default settings
    scraper := mangascraper.New()

    // Get list of available sources
    sources := scraper.GetSources()
    fmt.Println("Available sources:", sources)

    // Get mangas from a specific source (page 1)
    mangas, totalPages, err := scraper.GetAllMangas("mangalivre.blog", 1)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d mangas, %d total pages\n", len(mangas), totalPages)

    // Display some mangas
    for i, manga := range mangas[:5] {
        fmt.Printf("%d. %s (%s)\n", i+1, manga.Title, manga.URL)
    }

    // Get chapters from a manga (source auto-detected from URL)
    chapters, err := scraper.GetChapters(mangas[0].URL)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d chapters\n", len(chapters))

    // Get pages from a chapter
    if len(chapters) > 0 {
        pages, err := scraper.GetChapterPages(chapters[0].URL)
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("Chapter has %d pages\n", len(pages))
    }
}
```

## Configuration

You can customize the scraper behavior:

```go
config := &mangascraper.Config{
    Timeout:     60 * time.Second,  // HTTP timeout
    UserAgent:   "MyApp/1.0",       // Custom user agent
    EnableCache: true,               // Enable/disable caching
    CacheDir:    "/tmp/manga-cache", // Custom cache directory
    CacheTTL:    1 * time.Hour,      // Default cache TTL
    MaxRetries:  5,                  // Number of retries for failed requests
    RetryDelay:  2 * time.Second,    // Delay between retries
    RateLimit:   3,                  // Max requests per second
}

scraper := mangascraper.NewWithConfig(config)
```

## API Reference

### Main Types

```go
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
    Source      string   `json:"source"`
}

// Chapter represents a manga chapter
type Chapter struct {
    Number      string  `json:"number"`
    NumberFloat float64 `json:"numberFloat"`
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
```

### Scraper Methods

| Method | Description |
|--------|-------------|
| `New()` | Create scraper with defaults |
| `NewWithConfig(config)` | Create scraper with custom config |
| `GetSources()` | List available sources |
| `GetSourceInfo()` | Get detailed source info |
| `GetAllMangas(source, page)` | Get paginated manga list |
| `GetAllMangasFromAllSources(page)` | Get mangas from all sources |
| `SearchManga(source, query)` | Search in a specific source |
| `SearchAllSources(query)` | Search across all sources |
| `GetMangaDetails(url)` | Get manga details (auto-detect source) |
| `GetChapters(url)` | Get manga chapters (auto-detect source) |
| `GetChapterPages(url)` | Get chapter pages (auto-detect source) |
| `GetPopularMangas(source)` | Get popular mangas |
| `GetLatestUpdates(source)` | Get recently updated mangas |
| `GetMangasByGenre(source, genre)` | Get mangas by genre |
| `GetGenres(source)` | Get available genres |
| `GetAllGenres()` | Get genres from all sources |
| `ClearCache()` | Clear all cached data |

## Adding New Sources

You can create custom sources by implementing the `Source` interface:

```go
type Source interface {
    Name() string
    DisplayName() string
    BaseURL() string
    GetAllMangas(page int) ([]Manga, int, error)
    GetPopularMangas() ([]Manga, error)
    GetLatestUpdates() ([]Manga, error)
    SearchManga(query string) ([]Manga, error)
    GetMangaDetails(mangaURL string) (*Manga, error)
    GetChapters(mangaURL string) ([]Chapter, error)
    GetChapterPages(chapterURL string) ([]Page, error)
    GetMangasByGenre(genre string) ([]Manga, error)
    GetGenres() ([]string, error)
}
```

Then register it:

```go
scraper := mangascraper.New()
scraper.RegisterSource(myCustomSource)
```

## Caching

The library includes a persistent disk cache with automatic TTL management:

- **Manga lists**: 30 minutes
- **Manga details**: 60 minutes
- **Chapters**: 15 minutes
- **Chapter pages**: 24 hours
- **Search results**: 10 minutes

You can disable caching:

```go
config := mangascraper.DefaultConfig()
config.EnableCache = false
scraper := mangascraper.NewWithConfig(config)
```

Or clear the cache manually:

```go
scraper.ClearCache()
```

## Error Handling

The library returns errors for network failures, parsing errors, and invalid sources:

```go
mangas, _, err := scraper.GetAllMangas("invalid-source", 1)
if err != nil {
    // Handle error: "source not found: invalid-source"
}

chapters, err := scraper.GetChapters("https://unknown-site.com/manga/test")
if err != nil {
    // Handle error: "source not found for URL: ..."
}
```

## Thread Safety

The scraper is safe for concurrent use from multiple goroutines:

```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(page int) {
        defer wg.Done()
        mangas, _, _ := scraper.GetAllMangas("mangalivre.blog", page)
        fmt.Printf("Page %d: %d mangas\n", page, len(mangas))
    }(i + 1)
}
wg.Wait()
```

## License

MIT License - feel free to use in your projects!
