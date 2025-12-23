package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"GoAnimeGUI/pkg/scrapers"
)

// TorrentSource representa uma fonte de torrent disponivel
type TorrentSource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsBR        bool   `json:"isBr"`
	Available   bool   `json:"available"`
}

// TorrentResult resultado de busca de torrent para o frontend
type TorrentResult struct {
	Title     string `json:"title"`
	Magnet    string `json:"magnet"`
	Hash      string `json:"hash"`
	Size      string `json:"size"`
	Quality   string `json:"quality"`
	Seeders   int    `json:"seeders"`
	Source    string `json:"source"`
	IsBR      bool   `json:"isBr"`
	DualAudio bool   `json:"dualAudio"`
}

// scraperRegistry registry compartilhado
var scraperRegistry *scrapers.ProviderRegistry
var nyaaProvider *scrapers.NyaaProvider
var redeTorrentProvider *scrapers.RedeTorrentProvider

// initScrapers inicializa os scrapers
func initScrapers() {
	if scraperRegistry == nil {
		scraperRegistry = scrapers.NewProviderRegistry()
		nyaaProvider = scrapers.NewNyaaProvider()
		redeTorrentProvider = scrapers.NewRedeTorrentProvider()
		scraperRegistry.Register(nyaaProvider)
		scraperRegistry.Register(redeTorrentProvider)
		fmt.Println("[Scrapers] Nyaa + RedeTorrent inicializados")
	}
}

// GetTorrentSources retorna as fontes de torrent disponiveis
func (a *App) GetTorrentSources() []TorrentSource {
	initScrapers()

	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	sources := []TorrentSource{
		{
			ID:          "all",
			Name:        "Todas as Fontes",
			Description: "Busca em Nyaa + RedeTorrent",
			IsBR:        false,
			Available:   true,
		},
		{
			ID:          "nyaa",
			Name:        "Nyaa.si",
			Description: "Internacional - Dual Audio, Legendado",
			IsBR:        false,
			Available:   nyaaProvider.IsAvailable(ctx),
		},
		{
			ID:          "redetorrent",
			Name:        "RedeTorrent",
			Description: "BR - Dublado e Legendado PT-BR",
			IsBR:        true,
			Available:   redeTorrentProvider.IsAvailable(ctx),
		},
	}

	return sources
}

// isBatchTorrent verifica se é um pack/batch (múltiplos eps) baseado no título/tamanho
func isBatchTorrent(title string, size string) bool {
	titleLower := strings.ToLower(title)

	// Padrões que indicam batch/pack
	batchPatterns := []string{
		"batch", "complete", "completo", "temporada", "season",
		"01-", "1-", "e01-", "ep01-", "episode 01-",
		"s01", "s02", "s1", "s2",
		"~", " - ", "all episodes", "todos",
	}

	for _, pattern := range batchPatterns {
		if strings.Contains(titleLower, pattern) {
			return true
		}
	}

	// Se tamanho > 5GB provavelmente é batch
	if strings.Contains(size, "GiB") || strings.Contains(size, "GB") {
		sizeStr := strings.ReplaceAll(size, "GiB", "")
		sizeStr = strings.ReplaceAll(sizeStr, "GB", "")
		sizeStr = strings.TrimSpace(sizeStr)
		if sizeNum := parseFloat(sizeStr); sizeNum > 5.0 {
			return true
		}
	}

	return false
}

// parseFloat converte string para float
func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// extractBaseAnimeName extrai o nome base do anime removendo episódio/qualidade
func extractBaseAnimeName(title string) string {
	// Remove tags de fansub [SubGroup]
	result := title
	for strings.Contains(result, "[") && strings.Contains(result, "]") {
		start := strings.Index(result, "[")
		end := strings.Index(result, "]")
		if start >= 0 && end > start {
			result = result[:start] + result[end+1:]
		} else {
			break
		}
	}

	// Remove extensão
	result = strings.TrimSuffix(result, ".mkv")
	result = strings.TrimSuffix(result, ".mp4")

	// Remove número de episódio no final: " - 01", " E01", " Ep01", etc
	patterns := []string{
		" - ", " E", " Ep", " EP", " Episode ", " episode ",
	}
	for _, p := range patterns {
		if idx := strings.LastIndex(result, p); idx > 0 {
			suffix := result[idx:]
			// Verifica se depois do pattern tem número
			if len(suffix) > len(p) {
				afterPattern := suffix[len(p):]
				if len(afterPattern) > 0 && afterPattern[0] >= '0' && afterPattern[0] <= '9' {
					result = result[:idx]
					break
				}
			}
		}
	}

	// Remove qualidade (1080p), (720p) etc
	result = strings.TrimSpace(result)
	for strings.HasSuffix(result, ")") {
		start := strings.LastIndex(result, "(")
		if start > 0 {
			result = strings.TrimSpace(result[:start])
		} else {
			break
		}
	}

	return strings.TrimSpace(result)
}

// SearchTorrents busca torrents em todas as fontes ou fonte especifica
// Agrupa resultados por anime, priorizando batches/packs
func (a *App) SearchTorrents(query string, sourceID string) []TorrentResult {
	initScrapers()

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	var rawResults []scrapers.AnimeResult

	switch sourceID {
	case "nyaa":
		results, err := nyaaProvider.Search(ctx, query)
		if err != nil {
			fmt.Printf("[Scrapers] Nyaa erro: %v\n", err)
		} else {
			rawResults = results
			fmt.Printf("[Scrapers] Nyaa: %d resultados\n", len(results))
		}
	case "redetorrent":
		results, err := redeTorrentProvider.Search(ctx, query)
		if err != nil {
			fmt.Printf("[Scrapers] RedeTorrent erro: %v\n", err)
		} else {
			rawResults = results
			fmt.Printf("[Scrapers] RedeTorrent: %d resultados\n", len(results))
		}
	default:
		rawResults = scraperRegistry.SearchAll(ctx, query)
		fmt.Printf("[Scrapers] Todas fontes: %d resultados\n", len(rawResults))
	}

	seen := make(map[string]bool)
	var allResults []TorrentResult

	// Primeiro passo: converter e marcar batches
	for _, r := range rawResults {
		if r.Hash == "" || seen[r.Hash] {
			continue
		}
		seen[r.Hash] = true

		isBR := r.HasPTBR || r.DualAudio || r.IsDubbed || r.BRScore >= 50 ||
			strings.Contains(strings.ToLower(r.Source), "rede")

		allResults = append(allResults, TorrentResult{
			Title:     r.Title,
			Magnet:    r.Magnet,
			Hash:      r.Hash,
			Size:      r.Size,
			Quality:   r.Quality,
			Seeders:   r.Seeders,
			Source:    r.Source,
			IsBR:      isBR,
			DualAudio: r.DualAudio,
		})
	}

	// Segundo passo: agrupar por anime base, priorizar batches
	animeGroups := make(map[string][]TorrentResult)
	for _, r := range allResults {
		baseName := extractBaseAnimeName(r.Title)
		if baseName == "" {
			baseName = r.Title
		}
		// Normaliza para agrupar
		key := strings.ToLower(baseName)
		animeGroups[key] = append(animeGroups[key], r)
	}

	var results []TorrentResult
	seenAnimes := make(map[string]bool)

	// Para cada grupo de anime, pegar o melhor torrent (batch ou mais seeds)
	for animeName, group := range animeGroups {
		// Ordena: BR primeiro, depois batches, depois por seeds
		sort.Slice(group, func(i, j int) bool {
			// BR primeiro
			if group[i].IsBR != group[j].IsBR {
				return group[i].IsBR
			}
			// Batch primeiro
			isBatchI := isBatchTorrent(group[i].Title, group[i].Size)
			isBatchJ := isBatchTorrent(group[j].Title, group[j].Size)
			if isBatchI != isBatchJ {
				return isBatchI
			}
			// Mais seeds
			return group[i].Seeders > group[j].Seeders
		})

		// Adiciona TODOS os batches e os 3 melhores episódios únicos
		addedCount := 0
		for _, r := range group {
			isBatch := isBatchTorrent(r.Title, r.Size)

			// Sempre adiciona batches
			if isBatch {
				if !seenAnimes[r.Hash] {
					results = append(results, r)
					seenAnimes[r.Hash] = true
				}
				continue
			}

			// Para episódios únicos, limite de 3 por anime
			if addedCount < 3 {
				if !seenAnimes[r.Hash] {
					results = append(results, r)
					seenAnimes[r.Hash] = true
					addedCount++
				}
			}
		}

		_ = animeName // usado para debug se precisar
	}

	// Ordena resultado final
	sort.Slice(results, func(i, j int) bool {
		// BR primeiro
		if results[i].IsBR != results[j].IsBR {
			return results[i].IsBR
		}
		// Batches primeiro
		isBatchI := isBatchTorrent(results[i].Title, results[i].Size)
		isBatchJ := isBatchTorrent(results[j].Title, results[j].Size)
		if isBatchI != isBatchJ {
			return isBatchI
		}
		// Mais seeds
		return results[i].Seeders > results[j].Seeders
	})

	fmt.Printf("[Scrapers] %d resultados agrupados para '%s' (fonte: %s, total bruto: %d)\n",
		len(results), query, sourceID, len(allResults))
	return results
}

// SearchTorrentsBR busca apenas em fontes BR (RedeTorrent)
func (a *App) SearchTorrentsBR(query string) []TorrentResult {
	return a.SearchTorrents(query, "redetorrent")
}

// SearchTorrentsNyaa busca apenas no Nyaa
func (a *App) SearchTorrentsNyaa(query string) []TorrentResult {
	return a.SearchTorrents(query, "nyaa")
}

// SearchTorrentsAll busca em todas as fontes
func (a *App) SearchTorrentsAll(query string) []TorrentResult {
	return a.SearchTorrents(query, "all")
}
