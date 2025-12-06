package manga

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry representa uma entrada no cache com TTL
type CacheEntry struct {
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expiresAt"`
	CreatedAt time.Time   `json:"createdAt"`
}

// IsExpired verifica se a entrada expirou
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// MangaCache é um cache persistente para mangás
type MangaCache struct {
	data      map[string]*CacheEntry
	mutex     sync.RWMutex
	cacheFile string
	dirty     bool
}

// Constantes de TTL para cache de mangá
const (
	TTLMangaList     = 30 * time.Minute // Lista de mangás
	TTLMangaDetails  = 60 * time.Minute // Detalhes do mangá
	TTLMangaChapters = 15 * time.Minute // Capítulos (atualiza mais frequente)
	TTLMangaPages    = 24 * time.Hour   // Páginas do capítulo (raramente mudam)
	TTLMangaSearch   = 10 * time.Minute // Resultados de busca
)

// NewMangaCache cria um novo cache de mangá
func NewMangaCache() *MangaCache {
	cacheDir := getCacheDir()
	cacheFile := filepath.Join(cacheDir, "manga_cache.json")

	cache := &MangaCache{
		data:      make(map[string]*CacheEntry),
		cacheFile: cacheFile,
	}

	// Carrega cache do disco
	cache.loadFromDisk()

	// Inicia salvamento periódico e limpeza
	go cache.periodicMaintenance()

	return cache
}

// getCacheDir retorna o diretório de cache
func getCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	cacheDir := filepath.Join(homeDir, ".goanime", "cache")
	os.MkdirAll(cacheDir, 0755)
	return cacheDir
}

// Get retorna um valor do cache se existir e não estiver expirado
func (c *MangaCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	if entry.IsExpired() {
		return nil, false
	}

	return entry.Value, true
}

// Set adiciona um valor ao cache com TTL
func (c *MangaCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = &CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
	c.dirty = true
}

// GetMangas retorna lista de mangás do cache
func (c *MangaCache) GetMangas(key string) ([]Manga, bool) {
	value, exists := c.Get(key)
	if !exists {
		return nil, false
	}

	// Tenta converter para []Manga
	switch v := value.(type) {
	case []Manga:
		return v, true
	case []interface{}:
		// Reconverte de JSON
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

// SetMangas salva lista de mangás no cache
func (c *MangaCache) SetMangas(key string, mangas []Manga, ttl time.Duration) {
	c.Set(key, mangas, ttl)
}

// GetChapters retorna capítulos do cache
func (c *MangaCache) GetChapters(key string) ([]MangaChapter, bool) {
	value, exists := c.Get(key)
	if !exists {
		return nil, false
	}

	switch v := value.(type) {
	case []MangaChapter:
		return v, true
	case []interface{}:
		chapters := make([]MangaChapter, 0, len(v))
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

// SetChapters salva capítulos no cache
func (c *MangaCache) SetChapters(key string, chapters []MangaChapter, ttl time.Duration) {
	c.Set(key, chapters, ttl)
}

// GetPages retorna páginas do cache
func (c *MangaCache) GetPages(key string) ([]MangaPage, bool) {
	value, exists := c.Get(key)
	if !exists {
		return nil, false
	}

	switch v := value.(type) {
	case []MangaPage:
		return v, true
	case []interface{}:
		pages := make([]MangaPage, 0, len(v))
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

// SetPages salva páginas no cache
func (c *MangaCache) SetPages(key string, pages []MangaPage, ttl time.Duration) {
	c.Set(key, pages, ttl)
}

// Delete remove uma entrada do cache
func (c *MangaCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.data, key)
	c.dirty = true
}

// Clear limpa todo o cache
func (c *MangaCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data = make(map[string]*CacheEntry)
	c.dirty = true
}

// Stats retorna estatísticas do cache
func (c *MangaCache) Stats() (total int, expired int, valid int) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, entry := range c.data {
		total++
		if entry.IsExpired() {
			expired++
		} else {
			valid++
		}
	}
	return
}

// loadFromDisk carrega o cache do disco
func (c *MangaCache) loadFromDisk() {
	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		return // Arquivo não existe, cache vazio
	}

	var cacheData map[string]*CacheEntry
	if err := json.Unmarshal(data, &cacheData); err != nil {
		fmt.Printf("[MangaCache] Erro ao carregar cache: %v\n", err)
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Filtra entradas expiradas ao carregar
	for key, entry := range cacheData {
		if !entry.IsExpired() {
			c.data[key] = entry
		}
	}

	fmt.Printf("[MangaCache] Carregado %d entradas válidas do disco\n", len(c.data))
}

// saveToDisk salva o cache no disco
func (c *MangaCache) saveToDisk() {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.dirty {
		return
	}

	// Filtra entradas expiradas antes de salvar
	validData := make(map[string]*CacheEntry)
	for key, entry := range c.data {
		if !entry.IsExpired() {
			validData[key] = entry
		}
	}

	data, err := json.Marshal(validData)
	if err != nil {
		fmt.Printf("[MangaCache] Erro ao serializar cache: %v\n", err)
		return
	}

	if err := os.WriteFile(c.cacheFile, data, 0644); err != nil {
		fmt.Printf("[MangaCache] Erro ao salvar cache: %v\n", err)
		return
	}

	c.dirty = false
	fmt.Printf("[MangaCache] Salvo %d entradas no disco\n", len(validData))
}

// periodicMaintenance faz limpeza e salvamento periódico
func (c *MangaCache) periodicMaintenance() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
		c.saveToDisk()
	}
}

// cleanup remove entradas expiradas
func (c *MangaCache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	removed := 0
	for key, entry := range c.data {
		if entry.IsExpired() {
			delete(c.data, key)
			removed++
		}
	}

	if removed > 0 {
		c.dirty = true
		fmt.Printf("[MangaCache] Limpeza: removidas %d entradas expiradas\n", removed)
	}
}

// Funções auxiliares para conversão de mapas

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
	if v, ok := m["rating"].(float64); ok {
		manga.Rating = v
	}
	if v, ok := m["views"].(float64); ok {
		manga.Views = int(v)
	}
	if genres, ok := m["genres"].([]interface{}); ok {
		for _, g := range genres {
			if gs, ok := g.(string); ok {
				manga.Genres = append(manga.Genres, gs)
			}
		}
	}
	return manga
}

func chapterFromMap(m map[string]interface{}) MangaChapter {
	chapter := MangaChapter{}
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

func pageFromMap(m map[string]interface{}) MangaPage {
	page := MangaPage{}
	if v, ok := m["number"].(float64); ok {
		page.Number = int(v)
	}
	if v, ok := m["url"].(string); ok {
		page.URL = v
	}
	return page
}
