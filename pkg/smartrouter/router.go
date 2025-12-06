package smartrouter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// StreamResult representa o resultado de uma busca de stream
type StreamResult struct {
	URL      string
	Source   string
	Referer  string
	Headers  map[string]string
	Error    error
	Duration time.Duration
}

// SourceStats mantém estatísticas de cada fonte
type SourceStats struct {
	TotalRequests int64
	SuccessCount  int64
	FailureCount  int64
	TotalLatency  int64 // em milissegundos
	LastSuccess   time.Time
	LastFailure   time.Time
	IsCircuitOpen bool
	CircuitOpenAt time.Time
}

// StreamSource define uma fonte de streaming
type StreamSource struct {
	Name     string
	Priority int // Menor = maior prioridade
	Timeout  time.Duration
	Fetcher  func(ctx context.Context, animeTitle string, episodeNumber int) (string, error)
}

// SmartRouter gerencia múltiplas fontes de streaming com circuit breaker
type SmartRouter struct {
	sources          []StreamSource
	stats            map[string]*SourceStats
	statsMutex       sync.RWMutex
	circuitThreshold int           // Número de falhas para abrir o circuit
	circuitResetTime time.Duration // Tempo para tentar resetar o circuit
	defaultTimeout   time.Duration
}

// Config para o SmartRouter
type Config struct {
	CircuitThreshold int           // Padrão: 3 falhas
	CircuitResetTime time.Duration // Padrão: 30 segundos
	DefaultTimeout   time.Duration // Padrão: 5 segundos
}

// DefaultConfig retorna configuração padrão
func DefaultConfig() Config {
	return Config{
		CircuitThreshold: 3,
		CircuitResetTime: 30 * time.Second,
		DefaultTimeout:   5 * time.Second,
	}
}

// New cria um novo SmartRouter
func New(config Config) *SmartRouter {
	if config.CircuitThreshold <= 0 {
		config.CircuitThreshold = 3
	}
	if config.CircuitResetTime <= 0 {
		config.CircuitResetTime = 30 * time.Second
	}
	if config.DefaultTimeout <= 0 {
		config.DefaultTimeout = 5 * time.Second
	}

	return &SmartRouter{
		sources:          make([]StreamSource, 0),
		stats:            make(map[string]*SourceStats),
		circuitThreshold: config.CircuitThreshold,
		circuitResetTime: config.CircuitResetTime,
		defaultTimeout:   config.DefaultTimeout,
	}
}

// AddSource adiciona uma nova fonte de streaming
func (r *SmartRouter) AddSource(source StreamSource) {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()

	if source.Timeout <= 0 {
		source.Timeout = r.defaultTimeout
	}

	r.sources = append(r.sources, source)
	r.stats[source.Name] = &SourceStats{}

	// Ordena por prioridade
	for i := 0; i < len(r.sources)-1; i++ {
		for j := i + 1; j < len(r.sources); j++ {
			if r.sources[i].Priority > r.sources[j].Priority {
				r.sources[i], r.sources[j] = r.sources[j], r.sources[i]
			}
		}
	}
}

// isCircuitOpen verifica se o circuit breaker está aberto para uma fonte
func (r *SmartRouter) isCircuitOpen(sourceName string) bool {
	r.statsMutex.RLock()
	defer r.statsMutex.RUnlock()

	stats, ok := r.stats[sourceName]
	if !ok {
		return false
	}

	if !stats.IsCircuitOpen {
		return false
	}

	// Verifica se é hora de tentar resetar
	if time.Since(stats.CircuitOpenAt) > r.circuitResetTime {
		return false // Permite uma tentativa
	}

	return true
}

// recordSuccess registra um sucesso
func (r *SmartRouter) recordSuccess(sourceName string, latency time.Duration) {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()

	stats, ok := r.stats[sourceName]
	if !ok {
		return
	}

	atomic.AddInt64(&stats.TotalRequests, 1)
	atomic.AddInt64(&stats.SuccessCount, 1)
	atomic.AddInt64(&stats.TotalLatency, latency.Milliseconds())
	stats.LastSuccess = time.Now()

	// Fecha o circuit se estava aberto
	if stats.IsCircuitOpen {
		stats.IsCircuitOpen = false
		fmt.Printf("[SmartRouter] Circuit fechado para %s após sucesso\n", sourceName)
	}
}

// recordFailure registra uma falha
func (r *SmartRouter) recordFailure(sourceName string) {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()

	stats, ok := r.stats[sourceName]
	if !ok {
		return
	}

	atomic.AddInt64(&stats.TotalRequests, 1)
	atomic.AddInt64(&stats.FailureCount, 1)
	stats.LastFailure = time.Now()

	// Abre o circuit se atingiu o threshold
	recentFailures := stats.FailureCount - stats.SuccessCount
	if recentFailures >= int64(r.circuitThreshold) && !stats.IsCircuitOpen {
		stats.IsCircuitOpen = true
		stats.CircuitOpenAt = time.Now()
		fmt.Printf("[SmartRouter] Circuit ABERTO para %s após %d falhas\n", sourceName, recentFailures)
	}
}

// GetStream busca o stream usando a lógica de prioridade com fallback
func (r *SmartRouter) GetStream(animeTitle string, episodeNumber int) *StreamResult {
	startTime := time.Now()

	for _, source := range r.sources {
		// Verifica circuit breaker
		if r.isCircuitOpen(source.Name) {
			fmt.Printf("[SmartRouter] Pulando %s (circuit aberto)\n", source.Name)
			continue
		}

		fmt.Printf("[SmartRouter] Tentando %s (timeout: %v)\n", source.Name, source.Timeout)

		// Cria context com timeout
		ctx, cancel := context.WithTimeout(context.Background(), source.Timeout)

		// Canal para receber resultado
		resultChan := make(chan StreamResult, 1)

		go func(src StreamSource) {
			srcStart := time.Now()
			url, err := src.Fetcher(ctx, animeTitle, episodeNumber)
			duration := time.Since(srcStart)

			resultChan <- StreamResult{
				URL:      url,
				Source:   src.Name,
				Error:    err,
				Duration: duration,
			}
		}(source)

		// Espera resultado ou timeout
		select {
		case result := <-resultChan:
			cancel()

			if result.Error == nil && result.URL != "" {
				r.recordSuccess(source.Name, result.Duration)
				result.Duration = time.Since(startTime)
				fmt.Printf("[SmartRouter] ✓ Sucesso com %s em %v\n", source.Name, result.Duration)
				return &result
			}

			r.recordFailure(source.Name)
			fmt.Printf("[SmartRouter] ✗ Falha em %s: %v\n", source.Name, result.Error)

		case <-ctx.Done():
			cancel()
			r.recordFailure(source.Name)
			fmt.Printf("[SmartRouter] ⏱ Timeout em %s após %v\n", source.Name, source.Timeout)
		}
	}

	// Todas as fontes falharam
	return &StreamResult{
		Error:    errors.New("todas as fontes falharam"),
		Duration: time.Since(startTime),
	}
}

// GetStreamParallel busca em todas as fontes simultaneamente e retorna a primeira resposta
func (r *SmartRouter) GetStreamParallel(animeTitle string, episodeNumber int) *StreamResult {
	startTime := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultChan := make(chan StreamResult, len(r.sources))
	activeSources := 0

	for _, source := range r.sources {
		if r.isCircuitOpen(source.Name) {
			continue
		}

		activeSources++
		go func(src StreamSource) {
			srcCtx, srcCancel := context.WithTimeout(ctx, src.Timeout)
			defer srcCancel()

			srcStart := time.Now()
			url, err := src.Fetcher(srcCtx, animeTitle, episodeNumber)
			duration := time.Since(srcStart)

			select {
			case <-ctx.Done():
				// Context cancelado, outra fonte já respondeu
				return
			default:
				resultChan <- StreamResult{
					URL:      url,
					Source:   src.Name,
					Error:    err,
					Duration: duration,
				}
			}
		}(source)
	}

	if activeSources == 0 {
		return &StreamResult{
			Error:    errors.New("nenhuma fonte disponível"),
			Duration: time.Since(startTime),
		}
	}

	// Espera resultados
	var lastError error
	for i := 0; i < activeSources; i++ {
		result := <-resultChan

		if result.Error == nil && result.URL != "" {
			cancel() // Cancela outras requisições
			r.recordSuccess(result.Source, result.Duration)
			result.Duration = time.Since(startTime)
			fmt.Printf("[SmartRouter] ✓ Primeiro sucesso: %s em %v\n", result.Source, result.Duration)
			return &result
		}

		r.recordFailure(result.Source)
		lastError = result.Error
	}

	return &StreamResult{
		Error:    fmt.Errorf("todas as fontes falharam: %w", lastError),
		Duration: time.Since(startTime),
	}
}

// GetStats retorna estatísticas de uma fonte
func (r *SmartRouter) GetStats(sourceName string) *SourceStats {
	r.statsMutex.RLock()
	defer r.statsMutex.RUnlock()

	if stats, ok := r.stats[sourceName]; ok {
		// Retorna uma cópia
		return &SourceStats{
			TotalRequests: atomic.LoadInt64(&stats.TotalRequests),
			SuccessCount:  atomic.LoadInt64(&stats.SuccessCount),
			FailureCount:  atomic.LoadInt64(&stats.FailureCount),
			TotalLatency:  atomic.LoadInt64(&stats.TotalLatency),
			LastSuccess:   stats.LastSuccess,
			LastFailure:   stats.LastFailure,
			IsCircuitOpen: stats.IsCircuitOpen,
			CircuitOpenAt: stats.CircuitOpenAt,
		}
	}
	return nil
}

// GetAllStats retorna estatísticas de todas as fontes
func (r *SmartRouter) GetAllStats() map[string]*SourceStats {
	r.statsMutex.RLock()
	defer r.statsMutex.RUnlock()

	result := make(map[string]*SourceStats)
	for name, stats := range r.stats {
		result[name] = &SourceStats{
			TotalRequests: atomic.LoadInt64(&stats.TotalRequests),
			SuccessCount:  atomic.LoadInt64(&stats.SuccessCount),
			FailureCount:  atomic.LoadInt64(&stats.FailureCount),
			TotalLatency:  atomic.LoadInt64(&stats.TotalLatency),
			LastSuccess:   stats.LastSuccess,
			LastFailure:   stats.LastFailure,
			IsCircuitOpen: stats.IsCircuitOpen,
			CircuitOpenAt: stats.CircuitOpenAt,
		}
	}
	return result
}

// ResetCircuit reseta o circuit breaker de uma fonte
func (r *SmartRouter) ResetCircuit(sourceName string) {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()

	if stats, ok := r.stats[sourceName]; ok {
		stats.IsCircuitOpen = false
		stats.FailureCount = 0
		fmt.Printf("[SmartRouter] Circuit resetado manualmente para %s\n", sourceName)
	}
}

// ResetAllCircuits reseta todos os circuit breakers
func (r *SmartRouter) ResetAllCircuits() {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()

	for name, stats := range r.stats {
		stats.IsCircuitOpen = false
		stats.FailureCount = 0
		fmt.Printf("[SmartRouter] Circuit resetado para %s\n", name)
	}
}
