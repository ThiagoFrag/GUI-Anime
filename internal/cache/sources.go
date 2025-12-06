// Package cache - sources.go gerencia falhas e cooldowns de fontes de streaming
package cache

import (
	"sync"
	"time"
)

// SourceFailure rastreia falhas de uma fonte de streaming
type SourceFailure struct {
	Source      string
	FailedAt    time.Time
	CooldownEnd time.Time
	FailCount   int
	LastError   string
}

// SourceTracker gerencia o estado de múltiplas fontes de streaming
type SourceTracker struct {
	failures map[string]*SourceFailure
	mutex    sync.RWMutex
}

// NewSourceTracker cria um novo rastreador de fontes
func NewSourceTracker() *SourceTracker {
	return &SourceTracker{
		failures: make(map[string]*SourceFailure),
	}
}

// RecordFailure registra uma falha de uma fonte
func (st *SourceTracker) RecordFailure(source, errorMsg string) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	now := time.Now()

	if existing, ok := st.failures[source]; ok {
		existing.FailCount++
		existing.FailedAt = now
		existing.LastError = errorMsg

		// Cooldown exponencial: 30s, 1min, 2min, 5min, 10min
		cooldowns := []time.Duration{
			30 * time.Second,
			1 * time.Minute,
			2 * time.Minute,
			5 * time.Minute,
			10 * time.Minute,
		}

		idx := existing.FailCount - 1
		if idx >= len(cooldowns) {
			idx = len(cooldowns) - 1
		}

		existing.CooldownEnd = now.Add(cooldowns[idx])
	} else {
		st.failures[source] = &SourceFailure{
			Source:      source,
			FailedAt:    now,
			CooldownEnd: now.Add(30 * time.Second),
			FailCount:   1,
			LastError:   errorMsg,
		}
	}
}

// RecordSuccess registra sucesso de uma fonte (reseta falhas)
func (st *SourceTracker) RecordSuccess(source string) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	delete(st.failures, source)
}

// IsAvailable verifica se uma fonte está disponível (não em cooldown)
func (st *SourceTracker) IsAvailable(source string) bool {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	failure, exists := st.failures[source]
	if !exists {
		return true
	}

	return time.Now().After(failure.CooldownEnd)
}

// GetAlternative retorna uma fonte alternativa
func (st *SourceTracker) GetAlternative(currentSource string) string {
	alternatives := map[string]string{
		"AllAnime":  "AnimeFire",
		"AnimeFire": "AllAnime",
		"Consumet":  "AllAnime",
		"Enime":     "AnimeFire",
	}

	alt, ok := alternatives[currentSource]
	if ok && st.IsAvailable(alt) {
		return alt
	}

	// Retorna a primeira fonte disponível
	for source := range alternatives {
		if source != currentSource && st.IsAvailable(source) {
			return source
		}
	}

	return ""
}

// GetCooldownRemaining retorna tempo restante do cooldown
func (st *SourceTracker) GetCooldownRemaining(source string) time.Duration {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	failure, exists := st.failures[source]
	if !exists {
		return 0
	}

	remaining := time.Until(failure.CooldownEnd)
	if remaining < 0 {
		return 0
	}

	return remaining
}

// SourceStatus representa o status de uma fonte para estatísticas
type SourceStatus struct {
	Source       string    `json:"source"`
	Failures     int       `json:"failures"`
	LastFailure  time.Time `json:"lastFailure"`
	CooldownEnd  time.Time `json:"cooldownEnd"`
	IsInCooldown bool      `json:"isInCooldown"`
	LastError    string    `json:"lastError"`
}

// GetAllStatus retorna status de todas as fontes
func (st *SourceTracker) GetAllStatus() []SourceStatus {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	now := time.Now()
	result := make([]SourceStatus, 0, len(st.failures))

	for _, failure := range st.failures {
		result = append(result, SourceStatus{
			Source:       failure.Source,
			Failures:     failure.FailCount,
			LastFailure:  failure.FailedAt,
			CooldownEnd:  failure.CooldownEnd,
			IsInCooldown: now.Before(failure.CooldownEnd),
			LastError:    failure.LastError,
		})
	}

	return result
}

// Reset reseta o estado de todas as fontes
func (st *SourceTracker) Reset() {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	st.failures = make(map[string]*SourceFailure)
}

// ResetSource reseta o estado de uma fonte específica
func (st *SourceTracker) ResetSource(source string) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	delete(st.failures, source)
}
