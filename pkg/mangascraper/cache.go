package mangascraper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents a cached item with TTL
type CacheEntry struct {
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expiresAt"`
	CreatedAt time.Time   `json:"createdAt"`
}

// IsExpired checks if the entry has expired
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Cache provides persistent caching for manga data
type Cache struct {
	data      map[string]*CacheEntry
	mutex     sync.RWMutex
	cacheFile string
	dirty     bool
	enabled   bool
}

// CacheTTL constants
const (
	TTLMangaList     = 30 * time.Minute // Manga list
	TTLMangaDetails  = 60 * time.Minute // Manga details
	TTLMangaChapters = 15 * time.Minute // Chapters (updates more frequently)
	TTLMangaPages    = 24 * time.Hour   // Chapter pages (rarely change)
	TTLMangaSearch   = 10 * time.Minute // Search results
)

// NewCache creates a new cache instance
func NewCache(cacheDir string) *Cache {
	if cacheDir == "" {
		cacheDir = getDefaultCacheDir()
	}

	cacheFile := filepath.Join(cacheDir, "mangascraper_cache.json")

	cache := &Cache{
		data:      make(map[string]*CacheEntry),
		cacheFile: cacheFile,
		enabled:   true,
	}

	cache.loadFromDisk()
	go cache.periodicMaintenance()

	return cache
}

// NewDisabledCache creates a cache that is disabled (no-op)
func NewDisabledCache() *Cache {
	return &Cache{
		data:    make(map[string]*CacheEntry),
		enabled: false,
	}
}

func getDefaultCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	cacheDir := filepath.Join(homeDir, ".mangascraper", "cache")
	os.MkdirAll(cacheDir, 0755)
	return cacheDir
}

// Get retrieves a value from cache if it exists and is not expired
func (c *Cache) Get(key string) (interface{}, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists || entry.IsExpired() {
		return nil, false
	}

	return entry.Value, true
}

// Set stores a value in cache with TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	if !c.enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = &CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
	c.dirty = true
}

// GetMangas retrieves manga list from cache
func (c *Cache) GetMangas(key string) ([]Manga, bool) {
	value, exists := c.Get(key)
	if !exists {
		return nil, false
	}

	switch v := value.(type) {
	case []Manga:
		return v, true
	case []interface{}:
		mangas := make([]Manga, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				manga := mangaFromMap(m)
				mangas = append(mangas, manga)
			}
		}
		return mangas, len(mangas) > 0
	}

	return nil, false
}

// SetMangas stores manga list in cache
func (c *Cache) SetMangas(key string, mangas []Manga, ttl time.Duration) {
	c.Set(key, mangas, ttl)
}

// GetChapters retrieves chapters from cache
func (c *Cache) GetChapters(key string) ([]Chapter, bool) {
	value, exists := c.Get(key)
	if !exists {
		return nil, false
	}

	switch v := value.(type) {
	case []Chapter:
		return v, true
	case []interface{}:
		chapters := make([]Chapter, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				chapter := chapterFromMap(m)
				chapters = append(chapters, chapter)
			}
		}
		return chapters, len(chapters) > 0
	}

	return nil, false
}

// SetChapters stores chapters in cache
func (c *Cache) SetChapters(key string, chapters []Chapter, ttl time.Duration) {
	c.Set(key, chapters, ttl)
}

// GetPages retrieves pages from cache
func (c *Cache) GetPages(key string) ([]Page, bool) {
	value, exists := c.Get(key)
	if !exists {
		return nil, false
	}

	switch v := value.(type) {
	case []Page:
		return v, true
	case []interface{}:
		pages := make([]Page, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				page := pageFromMap(m)
				pages = append(pages, page)
			}
		}
		return pages, len(pages) > 0
	}

	return nil, false
}

// SetPages stores pages in cache
func (c *Cache) SetPages(key string, pages []Page, ttl time.Duration) {
	c.Set(key, pages, ttl)
}

// Delete removes an entry from cache
func (c *Cache) Delete(key string) {
	if !c.enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.data, key)
	c.dirty = true
}

// Clear removes all entries from cache
func (c *Cache) Clear() {
	if !c.enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data = make(map[string]*CacheEntry)
	c.dirty = true
	c.saveToDisk()
}

// Size returns the number of entries in cache
func (c *Cache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.data)
}

// loadFromDisk loads cache from disk
func (c *Cache) loadFromDisk() {
	if !c.enabled || c.cacheFile == "" {
		return
	}

	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		return
	}

	var entries map[string]*CacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return
	}

	c.mutex.Lock()
	c.data = entries
	c.mutex.Unlock()

	c.cleanExpired()
}

// saveToDisk saves cache to disk
func (c *Cache) saveToDisk() {
	if !c.enabled || c.cacheFile == "" {
		return
	}

	c.mutex.RLock()
	data, err := json.Marshal(c.data)
	c.mutex.RUnlock()

	if err != nil {
		return
	}

	dir := filepath.Dir(c.cacheFile)
	os.MkdirAll(dir, 0755)

	os.WriteFile(c.cacheFile, data, 0644)
	c.dirty = false
}

// cleanExpired removes expired entries
func (c *Cache) cleanExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
			c.dirty = true
		}
	}
}

// periodicMaintenance runs periodic cache maintenance
func (c *Cache) periodicMaintenance() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanExpired()
		if c.dirty {
			c.saveToDisk()
		}
	}
}

// Helper functions for JSON conversion

func mangaFromMap(m map[string]interface{}) Manga {
	manga := Manga{}

	if v, ok := m["id"].(string); ok {
		manga.ID = v
	}
	if v, ok := m["title"].(string); ok {
		manga.Title = v
	}
	if v, ok := m["image"].(string); ok {
		manga.Image = v
	}
	if v, ok := m["url"].(string); ok {
		manga.URL = v
	}
	if v, ok := m["latestChapter"].(string); ok {
		manga.LatestChap = v
	}
	if v, ok := m["description"].(string); ok {
		manga.Description = v
	}
	if v, ok := m["status"].(string); ok {
		manga.Status = v
	}
	if v, ok := m["author"].(string); ok {
		manga.Author = v
	}
	if v, ok := m["source"].(string); ok {
		manga.Source = v
	}
	if v, ok := m["rating"].(float64); ok {
		manga.Rating = v
	}
	if v, ok := m["views"].(float64); ok {
		manga.Views = int(v)
	}
	if v, ok := m["genres"].([]interface{}); ok {
		for _, g := range v {
			if gs, ok := g.(string); ok {
				manga.Genres = append(manga.Genres, gs)
			}
		}
	}

	return manga
}

func chapterFromMap(m map[string]interface{}) Chapter {
	chapter := Chapter{}

	if v, ok := m["number"].(string); ok {
		chapter.Number = v
	}
	if v, ok := m["numberFloat"].(float64); ok {
		chapter.NumberFloat = v
	}
	if v, ok := m["title"].(string); ok {
		chapter.Title = v
	}
	if v, ok := m["url"].(string); ok {
		chapter.URL = v
	}
	if v, ok := m["date"].(string); ok {
		chapter.Date = v
	}
	if v, ok := m["mangaId"].(string); ok {
		chapter.MangaID = v
	}
	if v, ok := m["mangaName"].(string); ok {
		chapter.MangaName = v
	}

	return chapter
}

func pageFromMap(m map[string]interface{}) Page {
	page := Page{}

	if v, ok := m["number"].(float64); ok {
		page.Number = int(v)
	}
	if v, ok := m["url"].(string); ok {
		page.URL = v
	}

	return page
}
