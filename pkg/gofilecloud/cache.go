package gofilecloud

import (
"sync"
"time"
)

// CacheEntry entrada de cache com TTL
type CacheEntry struct {
Data      interface{}
ExpiresAt time.Time
}

// Cache cache thread-safe
type Cache struct {
mu    sync.RWMutex
items map[string]*CacheEntry
}

// CachedAnime anime em cache
type CachedAnime struct {
Anime
Episodes []Episode `json:"episodes"`
}

// CachedSearchResult resultado de busca em cache
type CachedSearchResult struct {
Query     string     `json:"query"`
Results   []CachedAnime `json:"results"`
Total     int        `json:"total"`
CachedAt  time.Time  `json:"cached_at"`
}

// NewCache cria novo cache
func NewCache() *Cache {
return &Cache{
items: make(map[string]*CacheEntry),
}
}

// Get obtem item do cache
func (c *Cache) Get(key string) (interface{}, bool) {
c.mu.RLock()
defer c.mu.RUnlock()

if entry, ok := c.items[key]; ok {
if time.Now().Before(entry.ExpiresAt) {
return entry.Data, true
}
// Expirou, remove
delete(c.items, key)
}
return nil, false
}

// Set armazena item no cache
func (c *Cache) Set(key string, data interface{}, ttl time.Duration) {
c.mu.Lock()
defer c.mu.Unlock()

c.items[key] = &CacheEntry{
Data:      data,
ExpiresAt: time.Now().Add(ttl),
}
}

// Delete remove item do cache
func (c *Cache) Delete(key string) {
c.mu.Lock()
defer c.mu.Unlock()
delete(c.items, key)
}

// Clear limpa todo o cache
func (c *Cache) Clear() {
c.mu.Lock()
defer c.mu.Unlock()
c.items = make(map[string]*CacheEntry)
}

// CleanExpired limpa entradas expiradas
func (c *Cache) CleanExpired() int {
c.mu.Lock()
defer c.mu.Unlock()

count := 0
now := time.Now()
for key, entry := range c.items {
if now.After(entry.ExpiresAt) {
delete(c.items, key)
count++
}
}
return count
}

// SearchCache busca em cache com capacidades avancadas
func (client *Client) SearchCache(query string) (*CachedSearchResult, error) {
// Busca episodios que contem o termo
episodes, err := client.Search(query)
if err != nil {
return nil, err
}

// Agrupa por anime
animeMap := make(map[int64]*CachedAnime)
for _, ep := range episodes {
if existing, ok := animeMap[ep.AnimeID]; ok {
existing.Episodes = append(existing.Episodes, ep)
} else {
animeMap[ep.AnimeID] = &CachedAnime{
Anime: Anime{
ID:        ep.AnimeID,
Name:      ep.AnimeName,
Slug:      ep.AnimeSlug,
Cover:     ep.AnimeCover,
Category:  getCategoryFromLangs(ep.AudioLang, ep.SubLang),
AudioLang: ep.AudioLang,
SubLang:   ep.SubLang,
},
Episodes: []Episode{ep},
}
}
}

// Converte para slice
results := make([]CachedAnime, 0, len(animeMap))
for _, anime := range animeMap {
anime.TotalEps = len(anime.Episodes)
results = append(results, *anime)
}

return &CachedSearchResult{
Query:    query,
Results:  results,
Total:    len(results),
CachedAt: time.Now(),
}, nil
}

// getCategoryFromLangs determina categoria baseado nos idiomas
func getCategoryFromLangs(audioLang, subLang string) string {
if audioLang == "pt-BR" {
return "dubbed-ptbr"
}
if subLang == "pt-BR" {
return "sub-ptbr"
}
if audioLang == "dual" || audioLang == "multi" {
return "dual-audio"
}
return "original"
}
