package jikan

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Cache global para evitar chamadas repetidas
var (
	posterCache = make(map[string]string)
	failedCache = make(map[string]bool) // Cache de falhas para não retentar
	cacheMutex  sync.RWMutex
	httpClient  = &http.Client{Timeout: 3 * time.Second} // Timeout reduzido para resposta rápida
)

// Estruturas para mapear o JSON da Jikan API
type TopAnimeResponse struct {
	Data []AnimeData `json:"data"`
}

type AnimeData struct {
	MalID  int     `json:"mal_id"`
	Rank   int     `json:"rank"`
	Title  string  `json:"title"`
	Score  float64 `json:"score"`
	Year   int     `json:"year"`
	Images struct {
		Jpg struct {
			LargeImageUrl string `json:"large_image_url"`
			ImageUrl      string `json:"image_url"`
		} `json:"jpg"`
		Webp struct {
			LargeImageUrl string `json:"large_image_url"`
		} `json:"webp"`
	} `json:"images"`
}

// AnimeCard - estrutura simplificada para o Frontend
type AnimeCard struct {
	ID    int     `json:"id"`
	Rank  int     `json:"rank"`
	Title string  `json:"title"`
	Image string  `json:"image"`
	Score float64 `json:"score"`
	Year  int     `json:"year"`
}

// FetchTopAnimes busca os top animes na API (com cache)
func FetchTopAnimes() ([]AnimeCard, error) {
	resp, err := httpClient.Get("https://api.jikan.moe/v4/top/anime?limit=25")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result TopAnimeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	animes := make([]AnimeCard, 0, len(result.Data))
	for _, item := range result.Data {
		img := item.Images.Jpg.LargeImageUrl
		if img == "" {
			img = item.Images.Webp.LargeImageUrl
		}
		if img == "" {
			img = item.Images.Jpg.ImageUrl
		}

		animes = append(animes, AnimeCard{
			ID:    item.MalID,
			Rank:  item.Rank,
			Title: item.Title,
			Image: img,
			Score: item.Score,
			Year:  item.Year,
		})

		// Salva no cache também
		if img != "" {
			cacheMutex.Lock()
			posterCache[normalizeTitle(item.Title)] = img
			cacheMutex.Unlock()
		}
	}

	return animes, nil
}

// FetchPoster busca o poster com fallbacks múltiplos
func FetchPoster(title string) (string, error) {
	normalizedTitle := normalizeTitle(title)

	// Verifica cache de sucesso
	cacheMutex.RLock()
	if cached, ok := posterCache[normalizedTitle]; ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	// Verifica cache de falha
	if failedCache[normalizedTitle] {
		cacheMutex.RUnlock()
		return "", fmt.Errorf("já falhou antes")
	}
	cacheMutex.RUnlock()

	// Tenta buscar
	posterURL := tryFetchPoster(normalizedTitle)

	// Se não achou, tenta com título mais simples
	if posterURL == "" {
		simpleTitle := simplifyTitle(normalizedTitle)
		if simpleTitle != normalizedTitle {
			posterURL = tryFetchPoster(simpleTitle)
		}
	}

	// Salva resultado no cache
	cacheMutex.Lock()
	if posterURL != "" {
		posterCache[normalizedTitle] = posterURL
	} else {
		failedCache[normalizedTitle] = true
	}
	cacheMutex.Unlock()

	if posterURL == "" {
		return "", fmt.Errorf("poster não encontrado")
	}
	return posterURL, nil
}

func tryFetchPoster(title string) string {
	endpoint := fmt.Sprintf("https://api.jikan.moe/v4/anime?q=%s&limit=5&sfw=true", url.QueryEscape(title))

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	var result TopAnimeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	// Procura melhor match
	for _, item := range result.Data {
		if item.Images.Jpg.LargeImageUrl != "" {
			return item.Images.Jpg.LargeImageUrl
		}
		if item.Images.Webp.LargeImageUrl != "" {
			return item.Images.Webp.LargeImageUrl
		}
		if item.Images.Jpg.ImageUrl != "" {
			return item.Images.Jpg.ImageUrl
		}
	}

	return ""
}

// ExtractAnimeName extrai o nome limpo do anime de títulos de torrent
// Ex: "[EMBER] Frieren: Beyond Journey`s End (2023) [BDRip]" -> "Frieren"
func ExtractAnimeName(torrentTitle string) string {
	result := torrentTitle

	// 1. Remove tags de grupo entre colchetes [EMBER], [SubsPlease], etc
	for strings.Contains(result, "[") && strings.Contains(result, "]") {
		start := strings.Index(result, "[")
		end := strings.Index(result, "]")
		if end > start {
			result = result[:start] + result[end+1:]
		} else {
			break
		}
	}
	result = strings.TrimSpace(result)

	// 2. Remove padrões de episódio "S01E01", "- 01", "Episode 1"
	episodePatterns := []string{
		" S0", " S1", " S2", " S3", " - 0", " - 1", " - 2",
		" Episode", " Ep ", " Ep.", " E0", " E1",
	}
	for _, p := range episodePatterns {
		if idx := strings.Index(result, p); idx > 3 {
			result = result[:idx]
			break
		}
	}

	// 3. Remove qualidade e codec "1080p", "720p", "HEVC", "x265", etc
	qualityPatterns := []string{
		" 1080p", " 720p", " 480p", " 2160p", " 4K",
		" HEVC", " x265", " x264", " AVC", " H.264", " H 264",
		" BDRip", " BD", " WEB-DL", " WEBRip", " HDTV",
		" AAC", " FLAC", " DDP", " Dual Audio", " Multi",
		" NF ", " CR ", " AMZN",
	}
	for _, p := range qualityPatterns {
		if idx := indexOfCI(result, p); idx > 3 {
			result = result[:idx]
		}
	}

	// 4. Remove ano (2023), (2024), etc
	if idx := strings.LastIndex(result, " ("); idx > 3 {
		result = result[:idx]
	}

	// 5. Remove títulos alternativos após " - " ou ":" se muito longo
	if len(result) > 30 {
		if idx := strings.Index(result, " - "); idx > 5 {
			result = result[:idx]
		}
	}

	// 6. Pega apenas as primeiras palavras se ainda muito longo
	result = strings.TrimSpace(result)
	if len(result) > 40 {
		words := strings.Fields(result)
		if len(words) > 4 {
			result = strings.Join(words[:4], " ")
		}
	}

	return strings.TrimSpace(result)
}

// normalizeTitle remove sufixos comuns para melhor matching
func normalizeTitle(title string) string {
	// Primeiro tenta extrair nome de torrent
	result := ExtractAnimeName(title)
	if result == "" {
		result = title
	}

	// Remove padrões comuns extras
	patterns := []string{
		" (TV)", " (Dub)", " (Sub)", " (Dublado)", " (Legendado)",
		" Season", " 2nd Season", " 3rd Season", " Final Season",
		" Part", " OVA", " Movie", " Special",
		" Dublado", " Legendado", " todos-os-episodios",
	}
	for _, pattern := range patterns {
		if idx := indexOfCI(result, pattern); idx != -1 {
			result = result[:idx]
		}
	}

	// Remove scores (ex: "8.67  A16")
	if idx := strings.Index(result, "  "); idx != -1 {
		candidate := result[:idx]
		if len(candidate) > 5 {
			result = candidate
		}
	}

	return strings.TrimSpace(result)
}

// simplifyTitle para busca mais agressiva
func simplifyTitle(title string) string {
	// Pega só as primeiras palavras
	words := strings.Fields(title)
	if len(words) > 3 {
		return strings.Join(words[:3], " ")
	}
	if len(words) > 1 {
		return words[0]
	}
	return title
}

// indexOfCI - indexOf case insensitive
func indexOfCI(s, substr string) int {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Index(sLower, substrLower)
}

// ==================== MULTI-SOURCE POSTER FETCHING ====================

// FetchPosterMultiSource tenta múltiplas APIs em PARALELO para obter imagem do anime
// Retorna assim que a primeira API responder com uma imagem válida
func FetchPosterMultiSource(title string) string {
	normalizedTitle := normalizeTitle(title)

	// 1. Verifica cache primeiro (muito rápido)
	cacheMutex.RLock()
	if cached, ok := posterCache[normalizedTitle]; ok {
		cacheMutex.RUnlock()
		return cached
	}
	if failedCache[normalizedTitle] {
		cacheMutex.RUnlock()
		return ""
	}
	cacheMutex.RUnlock()

	// 2. Busca em TODAS as APIs em PARALELO
	simpleTitle := simplifyTitle(normalizedTitle)
	resultChan := make(chan string, 6) // Buffer para todas as tentativas

	// Lança todas as buscas em paralelo
	go func() {
		if poster := tryFetchPoster(normalizedTitle); poster != "" {
			resultChan <- poster
		} else {
			resultChan <- ""
		}
	}()

	go func() {
		if poster, _ := FetchPosterAniList(normalizedTitle); poster != "" {
			resultChan <- poster
		} else {
			resultChan <- ""
		}
	}()

	go func() {
		if poster, _ := FetchPosterKitsu(normalizedTitle); poster != "" {
			resultChan <- poster
		} else {
			resultChan <- ""
		}
	}()

	// Também tenta com título simplificado em paralelo (se diferente)
	if simpleTitle != normalizedTitle {
		go func() {
			if poster := tryFetchPoster(simpleTitle); poster != "" {
				resultChan <- poster
			} else {
				resultChan <- ""
			}
		}()

		go func() {
			if poster, _ := FetchPosterAniList(simpleTitle); poster != "" {
				resultChan <- poster
			} else {
				resultChan <- ""
			}
		}()
	} else {
		// Envia resultados vazios para manter a contagem
		go func() { resultChan <- "" }()
		go func() { resultChan <- "" }()
	}

	// Espera pelo primeiro resultado válido ou timeout (1.5s max)
	timeout := time.After(1500 * time.Millisecond)
	expectedResults := 5
	received := 0

	for received < expectedResults {
		select {
		case poster := <-resultChan:
			received++
			if poster != "" {
				// Encontrou! Cacheia e retorna imediatamente
				cachePoster(normalizedTitle, poster)
				return poster
			}
		case <-timeout:
			// Timeout - marca como falha
			cacheMutex.Lock()
			failedCache[normalizedTitle] = true
			cacheMutex.Unlock()
			return ""
		}
	}

	// Todas as APIs falharam
	cacheMutex.Lock()
	failedCache[normalizedTitle] = true
	cacheMutex.Unlock()
	return ""
}

func cachePoster(title, poster string) {
	cacheMutex.Lock()
	posterCache[title] = poster
	cacheMutex.Unlock()
}

// AniList GraphQL API
type aniListResponse struct {
	Data struct {
		Media struct {
			CoverImage struct {
				Large  string `json:"large"`
				Medium string `json:"medium"`
			} `json:"coverImage"`
		} `json:"Media"`
	} `json:"data"`
}

// FetchPosterAniList busca poster na API do AniList
func FetchPosterAniList(title string) (string, error) {
	query := `query ($search: String) {
		Media(search: $search, type: ANIME) {
			coverImage {
				large
				medium
			}
		}
	}`

	variables := map[string]interface{}{
		"search": title,
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})

	req, err := http.NewRequest("POST", "https://graphql.anilist.co", strings.NewReader(string(reqBody)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("anilist status: %d", resp.StatusCode)
	}

	var result aniListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Data.Media.CoverImage.Large != "" {
		return result.Data.Media.CoverImage.Large, nil
	}
	if result.Data.Media.CoverImage.Medium != "" {
		return result.Data.Media.CoverImage.Medium, nil
	}

	return "", fmt.Errorf("imagem não encontrada")
}

// Kitsu API
type kitsuResponse struct {
	Data []struct {
		Attributes struct {
			PosterImage struct {
				Large    string `json:"large"`
				Original string `json:"original"`
				Medium   string `json:"medium"`
			} `json:"posterImage"`
		} `json:"attributes"`
	} `json:"data"`
}

// FetchPosterKitsu busca poster na API do Kitsu
func FetchPosterKitsu(title string) (string, error) {
	endpoint := fmt.Sprintf("https://kitsu.io/api/edge/anime?filter[text]=%s&page[limit]=3", url.QueryEscape(title))

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.api+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("kitsu status: %d", resp.StatusCode)
	}

	var result kitsuResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Data) > 0 {
		img := result.Data[0].Attributes.PosterImage
		if img.Large != "" {
			return img.Large, nil
		}
		if img.Original != "" {
			return img.Original, nil
		}
		if img.Medium != "" {
			return img.Medium, nil
		}
	}

	return "", fmt.Errorf("imagem não encontrada")
}

// PosterResult resultado da busca de poster
type PosterResult struct {
	Title  string `json:"title"`
	Poster string `json:"poster"`
}

// FetchPostersMultiThread busca posters para múltiplos títulos em paralelo
// Usa worker pool para controle de concorrência (5 workers = 5x mais rápido)
func FetchPostersMultiThread(titles []string, maxWorkers int) []PosterResult {
	if len(titles) == 0 {
		return nil
	}

	if maxWorkers <= 0 {
		maxWorkers = 5 // Default 5 workers simultâneos
	}

	results := make([]PosterResult, len(titles))
	var wg sync.WaitGroup

	// Canal para distribuir trabalho
	jobs := make(chan int, len(titles))

	// Inicia workers
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				title := titles[i]
				poster := FetchPosterMultiSource(title)
				results[i] = PosterResult{
					Title:  title,
					Poster: poster,
				}
			}
		}()
	}

	// Envia trabalhos
	for i := range titles {
		jobs <- i
	}
	close(jobs)

	// Aguarda todos terminarem
	wg.Wait()

	return results
}
