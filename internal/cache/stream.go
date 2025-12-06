// Package cache - stream.go contém cache especializado para streams de vídeo
package cache

import (
	"sync"
	"time"
)

// StreamEntry representa um stream cacheado com metadados de validação
type StreamEntry struct {
	URL           string
	Source        string
	CachedAt      time.Time
	ExpiresAt     time.Time
	LastValidated time.Time
	IsValid       bool
}

// IsExpired verifica se o cache expirou
func (s *StreamEntry) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// NeedsRevalidation verifica se precisa revalidar a URL
func (s *StreamEntry) NeedsRevalidation() bool {
	return time.Since(s.LastValidated) > 5*time.Minute
}

// StreamCache é um cache especializado para URLs de stream
type StreamCache struct {
	data  map[string]*StreamEntry
	mutex sync.RWMutex
}

// NewStreamCache cria um novo cache de streams
func NewStreamCache() *StreamCache {
	sc := &StreamCache{
		data: make(map[string]*StreamEntry),
	}

	// Inicia limpeza periódica
	go sc.periodicCleanup()

	return sc
}

// Get retorna um stream cacheado se existir e não estiver expirado
func (sc *StreamCache) Get(key string) (*StreamEntry, bool) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	entry, exists := sc.data[key]
	if !exists || entry.IsExpired() {
		return nil, false
	}

	return entry, true
}

// Set armazena um stream no cache
func (sc *StreamCache) Set(key, url, source string, ttl time.Duration) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	now := time.Now()
	sc.data[key] = &StreamEntry{
		URL:           url,
		Source:        source,
		CachedAt:      now,
		ExpiresAt:     now.Add(ttl),
		LastValidated: now,
		IsValid:       true,
	}
}

// UpdateValidation atualiza o timestamp de validação
func (sc *StreamCache) UpdateValidation(key string, isValid bool) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	if entry, exists := sc.data[key]; exists {
		entry.LastValidated = time.Now()
		entry.IsValid = isValid
	}
}

// Delete remove um stream do cache
func (sc *StreamCache) Delete(key string) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	delete(sc.data, key)
}

// Clear limpa todo o cache de streams
func (sc *StreamCache) Clear() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.data = make(map[string]*StreamEntry)
}

// Size retorna o número de streams cacheados
func (sc *StreamCache) Size() int {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	return len(sc.data)
}

// periodicCleanup remove streams expirados periodicamente
func (sc *StreamCache) periodicCleanup() {
	ticker := time.NewTicker(3 * time.Minute)
	for range ticker.C {
		sc.mutex.Lock()
		now := time.Now()
		for key, entry := range sc.data {
			if now.After(entry.ExpiresAt) {
				delete(sc.data, key)
			}
		}
		sc.mutex.Unlock()
	}
}

// GetValidatedURL retorna a URL se válida, ou string vazia
func (sc *StreamCache) GetValidatedURL(key string) (string, bool) {
	entry, exists := sc.Get(key)
	if !exists {
		return "", false
	}

	if !entry.IsValid {
		return "", false
	}

	return entry.URL, true
}
