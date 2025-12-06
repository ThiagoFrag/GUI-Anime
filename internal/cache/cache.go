// Package cache fornece um sistema de cache genérico com TTL
package cache

import (
	"sync"
	"time"
)

// Constantes de TTL para diferentes tipos de cache
const (
	TTLSearch   = 10 * time.Minute // Cache de busca
	TTLTrending = 30 * time.Minute // Cache de trending
	TTLStream   = 15 * time.Minute // Cache de streams
	TTLEpisodes = 30 * time.Minute // Cache de episódios
	TTLImages   = 60 * time.Minute // Cache de imagens HD
	TTLTopAnime = 60 * time.Minute // Cache de top animes
)

// Entry representa uma entrada no cache com TTL
type Entry struct {
	Value     interface{}
	ExpiresAt time.Time
	CreatedAt time.Time
	Source    string
}

// IsExpired verifica se a entrada expirou
func (e *Entry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Cache é um cache thread-safe com TTL
type Cache struct {
	data  map[string]*Entry
	mutex sync.RWMutex
}

// New cria um novo cache
func New() *Cache {
	c := &Cache{
		data: make(map[string]*Entry),
	}

	// Inicia limpeza periódica
	go c.periodicCleanup()

	return c
}

// Get retorna um valor do cache se existir e não estiver expirado
func (c *Cache) Get(key string) (interface{}, bool) {
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

// Set armazena um valor no cache com TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = &Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
}

// SetWithSource armazena um valor com informação da fonte
func (c *Cache) SetWithSource(key string, value interface{}, source string, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = &Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
		Source:    source,
	}
}

// Delete remove uma entrada do cache
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
}

// Clear limpa todo o cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]*Entry)
}

// CleanExpired remove entradas expiradas
func (c *Cache) CleanExpired() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	count := 0
	now := time.Now()

	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
			count++
		}
	}

	return count
}

// Size retorna o número de entradas no cache
func (c *Cache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.data)
}

// Keys retorna todas as chaves do cache
func (c *Cache) Keys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// Has verifica se uma chave existe e não expirou
func (c *Cache) Has(key string) bool {
	_, exists := c.Get(key)
	return exists
}

// GetEntry retorna a entrada completa do cache
func (c *Cache) GetEntry(key string) (*Entry, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists || entry.IsExpired() {
		return nil, false
	}

	return entry, true
}

// periodicCleanup executa limpeza periódica do cache
func (c *Cache) periodicCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		cleaned := c.CleanExpired()
		if cleaned > 0 {
			// Log silencioso - apenas para debug
			_ = cleaned
		}
	}
}

// Stats representa estatísticas do cache
type Stats struct {
	TotalEntries   int
	ExpiredEntries int
	ActiveEntries  int
}

// GetStats retorna estatísticas do cache
func (c *Cache) GetStats() Stats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	now := time.Now()
	expired := 0

	for _, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			expired++
		}
	}

	return Stats{
		TotalEntries:   len(c.data),
		ExpiredEntries: expired,
		ActiveEntries:  len(c.data) - expired,
	}
}
