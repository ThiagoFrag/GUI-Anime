// pkg/scrapers/nyaa.go
// Provider principal e mais estável - Nyaa.si
package scrapers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// NyaaProvider implementa scraping do Nyaa.si
type NyaaProvider struct {
	timeout time.Duration
}

// NewNyaaProvider cria um novo provider do Nyaa
func NewNyaaProvider() *NyaaProvider {
	return &NyaaProvider{
		timeout: 15 * time.Second,
	}
}

// Name retorna o nome do provider
func (n *NyaaProvider) Name() string {
	return "Nyaa"
}

// IsAvailable verifica se o Nyaa está acessível
func (n *NyaaProvider) IsAvailable(ctx context.Context) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "HEAD", "https://nyaa.si", nil)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// Search busca animes no Nyaa.si usando gocolly
func (n *NyaaProvider) Search(ctx context.Context, query string) ([]AnimeResult, error) {
	var results []AnimeResult
	var searchErr error

	c := colly.NewCollector(
		colly.AllowedDomains("nyaa.si"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	c.SetRequestTimeout(n.timeout)

	// Configura contexto para cancelamento
	c.OnRequest(func(r *colly.Request) {
		select {
		case <-ctx.Done():
			r.Abort()
		default:
		}
	})

	// Parser das linhas da tabela do Nyaa
	c.OnHTML("tr.default, tr.success", func(e *colly.HTMLElement) {
		// Título - pega o segundo link (primeiro é categoria)
		title := e.ChildText("td:nth-child(2) a:not(.comments)")
		if title == "" {
			// Fallback para outro seletor
			title = e.ChildAttr("td:nth-child(2) a[href*='/view/']", "title")
		}

		// Magnet link
		magnet := e.ChildAttr("td:nth-child(3) a[href^='magnet:']", "href")

		// Tamanho
		size := e.ChildText("td:nth-child(4)")

		// Seeders e Leechers
		seedersStr := e.ChildText("td:nth-child(6)")
		leechersStr := e.ChildText("td:nth-child(7)")

		seeders, _ := strconv.Atoi(strings.TrimSpace(seedersStr))
		leechers, _ := strconv.Atoi(strings.TrimSpace(leechersStr))

		if title != "" && magnet != "" {
			// Extrai hash do magnet
			hash := extractHash(magnet)

			// Detecta qualidade do título
			quality := detectQuality(title)

			// Detecta se tem áudio português
			hasPTBR, dualAudio, dubbed := detectBRFeatures(title)

			// Calcula score BR
			brScore := 0
			if dualAudio {
				brScore = 100
			} else if dubbed {
				brScore = 90
			} else if hasPTBR {
				brScore = 80
			}

			results = append(results, AnimeResult{
				Title:     strings.TrimSpace(title),
				Magnet:    magnet,
				Hash:      hash,
				Seeders:   seeders,
				Leechers:  leechers,
				Source:    "Nyaa",
				Size:      strings.TrimSpace(size),
				Quality:   quality,
				HasPTBR:   hasPTBR,
				DualAudio: dualAudio,
				IsDubbed:  dubbed,
				BRScore:   brScore,
			})
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		searchErr = fmt.Errorf("erro ao acessar Nyaa: %w", err)
		log.Printf("[Nyaa] Erro: %v", err)
	})

	// URL de busca ordenada por seeders
	encodedQuery := url.QueryEscape(query)
	// f=0 = sem filtro, c=1_2 = Anime - English-translated, s=seeders, o=desc
	searchURL := fmt.Sprintf("https://nyaa.si/?f=0&c=1_2&q=%s&s=seeders&o=desc", encodedQuery)

	log.Printf("[Nyaa] Buscando: %s", searchURL)

	if err := c.Visit(searchURL); err != nil {
		return nil, fmt.Errorf("falha ao visitar Nyaa: %w", err)
	}

	if searchErr != nil {
		return nil, searchErr
	}

	log.Printf("[Nyaa] Encontrados %d resultados para '%s'", len(results), query)
	return results, nil
}

// SearchDualAudio busca especificamente por dual audio
func (n *NyaaProvider) SearchDualAudio(ctx context.Context, query string) ([]AnimeResult, error) {
	return n.Search(ctx, query+" dual audio")
}

// SearchBR busca especificamente por conteúdo brasileiro
func (n *NyaaProvider) SearchBR(ctx context.Context, query string) ([]AnimeResult, error) {
	queries := []string{
		query + " dual audio",
		query + " portuguese",
		query + " PTBR",
		query + " legendado",
	}

	var allResults []AnimeResult
	for _, q := range queries {
		results, err := n.Search(ctx, q)
		if err == nil {
			allResults = append(allResults, results...)
		}
	}

	// Remove duplicatas
	return deduplicateResults(allResults), nil
}

// =============================================================================
// FUNÇÕES AUXILIARES
// =============================================================================

// extractHash extrai o hash do magnet link
func extractHash(magnet string) string {
	if idx := strings.Index(magnet, "btih:"); idx >= 0 {
		hashPart := magnet[idx+5:]
		if ampIdx := strings.Index(hashPart, "&"); ampIdx > 0 {
			return strings.ToLower(hashPart[:ampIdx])
		}
		return strings.ToLower(hashPart)
	}
	return ""
}

// detectQuality detecta a qualidade do vídeo pelo título
func detectQuality(title string) string {
	titleLower := strings.ToLower(title)

	if strings.Contains(titleLower, "2160p") || strings.Contains(titleLower, "4k") || strings.Contains(titleLower, "uhd") {
		return "4K"
	}
	if strings.Contains(titleLower, "1080p") || strings.Contains(titleLower, "fullhd") || strings.Contains(titleLower, "full hd") {
		return "1080p"
	}
	if strings.Contains(titleLower, "720p") || strings.Contains(titleLower, "hd") {
		return "720p"
	}
	if strings.Contains(titleLower, "480p") || strings.Contains(titleLower, "sd") {
		return "480p"
	}

	return "Unknown"
}

// detectBRFeatures detecta características brasileiras
func detectBRFeatures(title string) (hasPTBR, dualAudio, dubbed bool) {
	titleLower := strings.ToLower(title)

	// Dual Audio
	if strings.Contains(titleLower, "dual audio") || strings.Contains(titleLower, "dual-audio") || strings.Contains(titleLower, "dualaudio") {
		dualAudio = true
		hasPTBR = true
	}

	// Dublado
	if strings.Contains(titleLower, "dublado") || strings.Contains(titleLower, "dubbed pt") || strings.Contains(titleLower, "dub pt") {
		dubbed = true
		hasPTBR = true
	}

	// Legendado PT-BR
	if strings.Contains(titleLower, "ptbr") || strings.Contains(titleLower, "pt-br") || strings.Contains(titleLower, "portuguese") || strings.Contains(titleLower, "legendado") {
		hasPTBR = true
	}

	return
}

// deduplicateResults remove resultados duplicados por hash
func deduplicateResults(results []AnimeResult) []AnimeResult {
	seen := make(map[string]bool)
	var unique []AnimeResult

	for _, r := range results {
		key := r.Hash
		if key == "" {
			key = r.Magnet
		}
		if !seen[key] {
			seen[key] = true
			unique = append(unique, r)
		}
	}

	return unique
}
