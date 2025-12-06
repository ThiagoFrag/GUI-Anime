// Package api - stream.go gerencia streaming de vídeos
package api

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"goanime-gui/internal/cache"
	"goanime-gui/internal/proxy"
	"goanime-gui/pkg/smartrouter"
	"goanime-gui/pkg/store"
	"goanime-gui/pkg/videoextractor"

	goanime "github.com/alvarorichard/Goanime/pkg/goanime"
	"github.com/alvarorichard/Goanime/pkg/goanime/types"
)

// StreamService gerencia operações de streaming de vídeo
type StreamService struct {
	client        *goanime.Client
	cache         *cache.Cache
	streamCache   *cache.StreamCache
	sourceTracker *cache.SourceTracker
	streamRouter  *smartrouter.SmartRouter
	proxy         *proxy.Server
	episodesCache map[string][]store.Episode
	mutex         sync.RWMutex
}

// NewStreamService cria um novo serviço de streaming
func NewStreamService() *StreamService {
	return &StreamService{
		client:        goanime.NewClient(),
		cache:         cache.New(),
		streamCache:   cache.NewStreamCache(),
		sourceTracker: cache.NewSourceTracker(),
		streamRouter:  smartrouter.New(),
		proxy:         proxy.New(),
		episodesCache: make(map[string][]store.Episode),
	}
}

// StartProxy inicia o servidor de proxy
func (s *StreamService) StartProxy() error {
	return s.proxy.Start()
}

// GetProxyPort retorna a porta do proxy
func (s *StreamService) GetProxyPort() int {
	return s.proxy.Port()
}

// SmartStreamResult é o resultado da busca inteligente de stream
type SmartStreamResult struct {
	URL      string  `json:"url"`
	Source   string  `json:"source"`
	Duration float64 `json:"duration"`
	Success  bool    `json:"success"`
	Error    string  `json:"error,omitempty"`
}

// GetSmartStream usa o Smart Router para buscar stream
func (s *StreamService) GetSmartStream(animeTitle string, episodeNumber int) (*SmartStreamResult, error) {
	fmt.Printf("[SmartStream] Buscando: %s Ep.%d\n", animeTitle, episodeNumber)

	cacheKey := fmt.Sprintf("smart_stream:%s:%d", strings.ToLower(animeTitle), episodeNumber)

	if cached, ok := s.cache.Get(cacheKey); ok {
		fmt.Println("[SmartStream] Cache hit!")
		return cached.(*SmartStreamResult), nil
	}

	result := s.streamRouter.GetStream(animeTitle, episodeNumber)

	smartResult := &SmartStreamResult{
		URL:      result.URL,
		Source:   result.Source,
		Duration: float64(result.Duration.Milliseconds()),
		Success:  result.Error == nil && result.URL != "",
	}

	if result.Error != nil {
		smartResult.Error = result.Error.Error()
		return smartResult, result.Error
	}

	s.cache.Set(cacheKey, smartResult, cache.TTLStream)

	return smartResult, nil
}

// GetSmartStreamParallel busca em todas as fontes simultaneamente
func (s *StreamService) GetSmartStreamParallel(animeTitle string, episodeNumber int) (*SmartStreamResult, error) {
	cacheKey := fmt.Sprintf("smart_stream:%s:%d", strings.ToLower(animeTitle), episodeNumber)

	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*SmartStreamResult), nil
	}

	result := s.streamRouter.GetStreamParallel(animeTitle, episodeNumber)

	smartResult := &SmartStreamResult{
		URL:      result.URL,
		Source:   result.Source,
		Duration: float64(result.Duration.Milliseconds()),
		Success:  result.Error == nil && result.URL != "",
	}

	if result.Error != nil {
		smartResult.Error = result.Error.Error()
		return smartResult, result.Error
	}

	s.cache.Set(cacheKey, smartResult, cache.TTLStream)

	return smartResult, nil
}

// GetStreamURLForEpisode retorna a URL real do vídeo
func (s *StreamService) GetStreamURLForEpisode(animeURL, episodeURL string, cachedEpisodes []store.Episode) (string, error) {
	fmt.Printf("[GetStreamURL] AnimeURL: %s, EpisodeURL: %s\n", animeURL, episodeURL)

	cacheKey := fmt.Sprintf("stream:%s", episodeURL)

	// Verifica cache de stream
	if url, ok := s.streamCache.GetValidatedURL(cacheKey); ok {
		fmt.Println("[GetStreamURL] Cache hit!")
		return url, nil
	}

	// Determina a fonte
	var primarySource types.Source
	var sourceName string

	if strings.Contains(strings.ToLower(animeURL), "animefire") ||
		strings.Contains(strings.ToLower(episodeURL), "animefire") {
		primarySource = types.SourceAnimeFire
		sourceName = "AnimeFire"
	} else {
		primarySource = types.SourceAllAnime
		sourceName = "AllAnime"
	}

	// Verifica disponibilidade da fonte
	if !s.sourceTracker.IsAvailable(sourceName) {
		altSource := s.sourceTracker.GetAlternative(sourceName)
		if altSource != "" {
			fmt.Printf("[GetStreamURL] Fonte %s em cooldown, usando %s\n", sourceName, altSource)
			sourceName = altSource
			if altSource == "AnimeFire" {
				primarySource = types.SourceAnimeFire
			} else {
				primarySource = types.SourceAllAnime
			}
		}
	}

	// Encontra o episódio no cache
	var targetEpisode *store.Episode
	for i := range cachedEpisodes {
		if cachedEpisodes[i].URL == episodeURL {
			targetEpisode = &cachedEpisodes[i]
			break
		}
	}

	if targetEpisode == nil {
		return "", fmt.Errorf("episódio não encontrado")
	}

	// Atualiza source se especificado no episódio
	if targetEpisode.Source != "" {
		if parsed, err := types.ParseSource(targetEpisode.Source); err == nil {
			primarySource = parsed
		}
	}

	// Busca paralela em todas as fontes
	allSources := []struct {
		name   string
		source types.Source
	}{
		{sourceName, primarySource},
		{"AllAnime", types.SourceAllAnime},
		{"AnimeFire", types.SourceAnimeFire},
	}

	// Remove duplicatas
	uniqueSources := make([]struct {
		name   string
		source types.Source
	}, 0, len(allSources))
	seen := make(map[string]bool)
	for _, src := range allSources {
		if !seen[src.name] {
			seen[src.name] = true
			uniqueSources = append(uniqueSources, src)
		}
	}

	type streamResult struct {
		url    string
		source string
		err    error
	}

	resultChan := make(chan streamResult, len(uniqueSources)+1)

	var wg sync.WaitGroup
	for _, src := range uniqueSources {
		if !s.sourceTracker.IsAvailable(src.name) {
			continue
		}

		wg.Add(1)
		go func(srcName string, srcType types.Source) {
			defer wg.Done()

			url, err := s.tryGetStream(targetEpisode, animeURL, episodeURL, srcType)
			if err == nil && url != "" {
				if valid, _ := proxy.ValidateURL(url); valid {
					resultChan <- streamResult{url: url, source: srcName}
					return
				}
			}
			resultChan <- streamResult{url: "", source: srcName, err: err}
		}(src.name, src.source)
	}

	// Smart Router em paralelo
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := s.streamRouter.GetStream(targetEpisode.Title, targetEpisode.Number)
		if result != nil && result.URL != "" && result.Error == nil {
			if valid, _ := proxy.ValidateURL(result.URL); valid {
				resultChan <- streamResult{url: result.URL, source: "SmartRouter:" + result.Source}
				return
			}
		}
		resultChan <- streamResult{url: "", source: "SmartRouter", err: fmt.Errorf("sem resultado")}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	timeout := time.After(10 * time.Second)
	var lastError error

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				return "", fmt.Errorf("nenhuma fonte disponível: %v", lastError)
			}

			if result.url != "" {
				s.streamCache.Set(cacheKey, result.url, result.source, cache.TTLStream)
				s.sourceTracker.RecordSuccess(result.source)
				fmt.Printf("[GetStreamURL] ✓ Sucesso com %s!\n", result.source)
				return result.url, nil
			} else if result.err != nil {
				lastError = result.err
				s.sourceTracker.RecordFailure(result.source, result.err.Error())
			}

		case <-timeout:
			return "", fmt.Errorf("timeout: nenhuma fonte respondeu")
		}
	}
}

// tryGetStream tenta obter stream de uma fonte específica
func (s *StreamService) tryGetStream(episode *store.Episode, animeURL, episodeURL string, source types.Source) (string, error) {
	anime := &types.Anime{
		Name:   episode.Title,
		URL:    animeURL,
		Source: source.String(),
	}

	ep := &types.Episode{
		Number: strconv.Itoa(episode.Number),
		URL:    episodeURL,
		Num:    episode.Number,
	}

	streamURL, _, err := s.client.GetEpisodeStreamURL(anime, ep, &goanime.StreamOptions{
		Quality: "best",
		Mode:    "sub",
	})

	if err == nil && streamURL != "" {
		return streamURL, nil
	}

	// Fallback para AnimeFire
	if source == types.SourceAnimeFire {
		streamURL, err = videoextractor.ExtractVideoURL(episodeURL)
		if err == nil && streamURL != "" {
			return streamURL, nil
		}
	}

	return "", fmt.Errorf("fonte %s falhou: %v", source, err)
}

// GetSourceStats retorna estatísticas das fontes
func (s *StreamService) GetSourceStats() map[string]interface{} {
	stats := s.streamRouter.GetAllStats()
	result := make(map[string]interface{})

	for name, stat := range stats {
		avgLatency := float64(0)
		if stat.SuccessCount > 0 {
			avgLatency = float64(stat.TotalLatency) / float64(stat.SuccessCount)
		}

		result[name] = map[string]interface{}{
			"totalRequests": stat.TotalRequests,
			"successCount":  stat.SuccessCount,
			"failureCount":  stat.FailureCount,
			"avgLatencyMs":  avgLatency,
			"isCircuitOpen": stat.IsCircuitOpen,
			"lastSuccess":   stat.LastSuccess.Format(time.RFC3339),
			"lastFailure":   stat.LastFailure.Format(time.RFC3339),
		}
	}

	return result
}

// ResetCircuits reseta todos os circuit breakers
func (s *StreamService) ResetCircuits() {
	s.streamRouter.ResetAllCircuits()
	s.sourceTracker.Reset()
}

// SetEpisodesCache define o cache de episódios
func (s *StreamService) SetEpisodesCache(url string, episodes []store.Episode) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.episodesCache[url] = episodes
}

// GetEpisodesCache retorna episódios do cache
func (s *StreamService) GetEpisodesCache(url string) ([]store.Episode, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	eps, ok := s.episodesCache[url]
	return eps, ok
}

// ClearCache limpa todos os caches de stream
func (s *StreamService) ClearCache() {
	s.cache.Clear()
	s.streamCache.Clear()
	s.mutex.Lock()
	s.episodesCache = make(map[string][]store.Episode)
	s.mutex.Unlock()
}
