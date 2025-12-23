// pkg/scrapers/types.go
// Contrato único para todos os providers de scraping
package scrapers

import "context"

// AnimeResult representa um resultado de busca de anime
type AnimeResult struct {
	Title    string `json:"title"`
	Magnet   string `json:"magnet"`
	Hash     string `json:"hash"`
	Seeders  int    `json:"seeders"`
	Leechers int    `json:"leechers"`
	Source   string `json:"source"` // "Nyaa", "RedeTorrent", "AnimesTorrent", etc.
	Size     string `json:"size"`
	Quality  string `json:"quality"` // "1080p", "720p", "4K"

	// Metadados BR
	HasPTBR   bool `json:"has_ptbr"`
	DualAudio bool `json:"dual_audio"`
	IsDubbed  bool `json:"is_dubbed"`
	BRScore   int  `json:"br_score"` // 0-100, maior = melhor para BR
}

// Provider é a interface que todo scraper deve implementar
type Provider interface {
	// Name retorna o nome do provider (para logs)
	Name() string

	// Search busca animes pela query
	Search(ctx context.Context, query string) ([]AnimeResult, error)

	// IsAvailable verifica se o provider está funcionando
	IsAvailable(ctx context.Context) bool
}

// ProviderRegistry gerencia múltiplos providers
type ProviderRegistry struct {
	providers []Provider
}

// NewProviderRegistry cria um novo registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make([]Provider, 0),
	}
}

// Register adiciona um provider ao registry
func (r *ProviderRegistry) Register(p Provider) {
	r.providers = append(r.providers, p)
}

// SearchAll busca em todos os providers em paralelo
func (r *ProviderRegistry) SearchAll(ctx context.Context, query string) []AnimeResult {
	var allResults []AnimeResult
	resultChan := make(chan []AnimeResult, len(r.providers))

	for _, p := range r.providers {
		go func(provider Provider) {
			results, err := provider.Search(ctx, query)
			if err == nil {
				resultChan <- results
			} else {
				resultChan <- nil
			}
		}(p)
	}

	// Coleta resultados
	for range r.providers {
		if results := <-resultChan; results != nil {
			allResults = append(allResults, results...)
		}
	}

	return allResults
}

// GetAvailableProviders retorna apenas providers funcionais
func (r *ProviderRegistry) GetAvailableProviders(ctx context.Context) []Provider {
	var available []Provider
	for _, p := range r.providers {
		if p.IsAvailable(ctx) {
			available = append(available, p)
		}
	}
	return available
}
