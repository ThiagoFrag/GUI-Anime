package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"GoAnimeGUI/internal/manga"
	"GoAnimeGUI/pkg/anilist"
	"GoAnimeGUI/pkg/animesflix"
	"GoAnimeGUI/pkg/aniskip"
	"GoAnimeGUI/pkg/consumet"
	"GoAnimeGUI/pkg/discord"
	"GoAnimeGUI/pkg/enime"
	"GoAnimeGUI/pkg/gofilecloud"
	"GoAnimeGUI/pkg/jikan"
	"GoAnimeGUI/pkg/smartrouter"
	"GoAnimeGUI/pkg/store"
	"GoAnimeGUI/pkg/videoextractor"

	"github.com/alvarorichard/Goanime/pkg/goanime"
	"github.com/alvarorichard/Goanime/pkg/goanime/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// CacheEntry representa uma entrada de cache com TTL e validaÃ§Ã£o
type CacheEntry struct {
	Data        interface{}
	ExpiresAt   time.Time
	URL         string    // URL original para validaÃ§Ã£o
	LastValidAt time.Time // Ãšltima vez que a URL foi validada
	FailCount   int       // NÃºmero de falhas consecutivas
	Source      string    // Fonte do stream (AnimeFire, AllAnime, etc)
}

// IsExpired verifica se a entrada expirou
func (c *CacheEntry) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// NeedsValidation verifica se a URL precisa ser revalidada
// Valida a cada 2 minutos ou se houve falhas
func (c *CacheEntry) NeedsValidation() bool {
	// Se expirou, precisa de novo fetch, nÃ£o apenas validaÃ§Ã£o
	if c.IsExpired() {
		return false
	}
	// Se nunca foi validado
	if c.LastValidAt.IsZero() {
		return true
	}
	// Se teve falhas recentes, valida mais frequentemente
	validationInterval := 2 * time.Minute
	if c.FailCount > 0 {
		validationInterval = 30 * time.Second
	}
	return time.Since(c.LastValidAt) > validationInterval
}

// StreamCacheEntry Ã© uma entrada de cache especÃ­fica para streams com mais metadados
type StreamCacheEntry struct {
	URL         string
	Source      string
	Quality     string
	Referer     string
	ExpiresAt   time.Time
	LastValidAt time.Time
	FailCount   int
	IsValidated bool
}

// SourceFailure rastreia falhas de fontes especÃ­ficas
type SourceFailure struct {
	Source     string
	FailedAt   time.Time
	FailCount  int
	LastError  string
	RetryAfter time.Time
}

// toTitleCase converte string para Title Case (substitui strings.Title deprecated)
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// PrefetchRequest representa um pedido de pré-carregamento de episódio
type PrefetchRequest struct {
	AnimeURL   string
	EpisodeURL string
	EpisodeNum int
}

// StreamResult representa o resultado de uma busca de stream
type StreamResult struct {
	URL    string
	Source string
	Error  error
}

type App struct {
	ctx              context.Context
	client           *goanime.Client
	animesflixClient *animesflix.Client
	gofileClient     *gofilecloud.Client
	User             *store.UserData

	// Smart Router para fontes de vÃ­deo
	streamRouter *smartrouter.SmartRouter

	// Cache unificado com TTL
	cache      map[string]*CacheEntry
	cacheMutex sync.RWMutex

	// Cache especÃ­ficos de alta performance (sem TTL para itens crÃ­ticos)
	episodesCache  map[string][]store.Episode
	urlCache       map[string]string
	topAnimesCache []store.SavedAnime
	trendingCache  []*AniListAnime

	// Cache inteligente de streams com validaÃ§Ã£o
	streamCache      map[string]*StreamCacheEntry
	streamCacheMutex sync.RWMutex

	// Rastreamento de falhas por fonte
	sourceFailures      map[string]*SourceFailure
	sourceFailuresMutex sync.RWMutex

	// Prefetch de episÃ³dios (carrega prÃ³ximos episÃ³dios em background)
	prefetchQueue  chan PrefetchRequest
	prefetchActive map[string]bool
	prefetchMutex  sync.RWMutex

	// Cache para imagens HD do AniList
	hdImageCache map[string]*anilist.AnimeMedia

	// Proxy de vÃ­deo para contornar CORS
	proxyServer     *http.Server
	proxyPort       int
	currentVideoURL string
	proxyMutex      sync.RWMutex

	// Cache de imagens de mangÃ¡ para carregamento rÃ¡pido
	imageCache      map[string][]byte
	imageCacheMutex sync.RWMutex
	imageClient     *http.Client

	// Estado de inicializaÃ§Ã£o
	initialized bool
	initMutex   sync.RWMutex

	// HTTP client para validaÃ§Ã£o de URLs
	validationClient *http.Client

	// Cliente de Manga com cache e worker pool
	mangaClient     *manga.MangaClient
	mangaCache      *manga.MangaCache
	mangaWorkerPool *manga.WorkerPool
	mangaAggregator *manga.MangaAggregator // Agregador de múltiplas fontes

	// Seeding Worker para contribuição comunitária
	seedingWorker *SeedingWorker
}

// Cache TTLs
const (
	CacheTTLSearch   = 10 * time.Minute // Buscas
	CacheTTLTrending = 30 * time.Minute // Trending
	CacheTTLTop      = 1 * time.Hour    // Top animes
	CacheTTLEpisodes = 1 * time.Hour    // EpisÃ³dios (aumentado para nÃ£o perder dados)
	CacheTTLStream   = 10 * time.Minute // URLs de stream (aumentado)
)

func NewApp() *App {
	app := &App{
		cache:          make(map[string]*CacheEntry),
		episodesCache:  make(map[string][]store.Episode),
		urlCache:       make(map[string]string),
		hdImageCache:   make(map[string]*anilist.AnimeMedia),
		streamCache:    make(map[string]*StreamCacheEntry),
		sourceFailures: make(map[string]*SourceFailure),
		imageCache:     make(map[string][]byte),
		imageClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
		validationClient: &http.Client{
			Timeout: 3 * time.Second, // Reduzido para resposta mais rÃ¡pida
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Permite redirecionamentos normalmente
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		prefetchQueue:  make(chan PrefetchRequest, 10),
		prefetchActive: make(map[string]bool),
	}

	// Inicializa o Smart Router com circuit breaker
	app.streamRouter = smartrouter.New(smartrouter.Config{
		CircuitThreshold: 3,                // 3 falhas abre o circuit
		CircuitResetTime: 30 * time.Second, // Tenta resetar apÃ³s 30s
		DefaultTimeout:   5 * time.Second,
	})

	// Adiciona fontes de streaming em ordem de prioridade
	// Prioridade 1: Enime API (mais rÃ¡pida, timeout curto)
	app.streamRouter.AddSource(smartrouter.StreamSource{
		Name:     "Enime",
		Priority: 1,
		Timeout:  3 * time.Second, // Timeout curto para nÃ£o travar
		Fetcher: func(ctx context.Context, title string, ep int) (string, error) {
			return enime.FindAndGetStreamWithContext(ctx, title, ep)
		},
	})

	// Prioridade 2: Consumet API (fallback confiÃ¡vel)
	app.streamRouter.AddSource(smartrouter.StreamSource{
		Name:     "Consumet",
		Priority: 2,
		Timeout:  5 * time.Second,
		Fetcher: func(ctx context.Context, title string, ep int) (string, error) {
			url, _, err := consumet.FindAnimeAndGetStream(title, ep)
			return url, err
		},
	})

	// Inicializa o cliente de Manga com cache e worker pool
	app.mangaClient = manga.NewMangaClient()
	app.mangaCache = manga.NewMangaCache()
	app.mangaWorkerPool = manga.NewWorkerPool(app.mangaClient, app.mangaCache, 4) // 4 workers paralelos
	app.mangaAggregator = manga.NewMangaAggregator()                              // Agregador com todas as fontes

	return app
}

// getCache recupera um item do cache se nÃ£o expirou
func (a *App) getCache(key string) (interface{}, bool) {
	a.cacheMutex.RLock()
	defer a.cacheMutex.RUnlock()

	if entry, ok := a.cache[key]; ok && !entry.IsExpired() {
		return entry.Data, true
	}
	return nil, false
}

// setCache armazena um item no cache com TTL
func (a *App) setCache(key string, data interface{}, ttl time.Duration) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	a.cache[key] = &CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// cleanExpiredCache limpa entradas expiradas (chamar periodicamente)
func (a *App) cleanExpiredCache() {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	count := 0
	for key, entry := range a.cache {
		if entry.IsExpired() {
			delete(a.cache, key)
			count++
		}
	}
	if count > 0 {
		fmt.Printf("[Cache] Limpou %d entradas expiradas\n", count)
	}
}

// ClearEpisodesCache limpa o cache de episÃ³dios para forÃ§ar recarga
func (a *App) ClearEpisodesCache() {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	a.episodesCache = make(map[string][]store.Episode)
	fmt.Println("[Cache] Cache de episÃ³dios limpo")
}

// ClearAllCache limpa todo o cache (Ãºtil para resolver problemas)
func (a *App) ClearAllCache() {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	a.cache = make(map[string]*CacheEntry)
	a.episodesCache = make(map[string][]store.Episode)
	a.urlCache = make(map[string]string)

	// Limpa tambÃ©m o cache de streams e falhas
	a.streamCacheMutex.Lock()
	a.streamCache = make(map[string]*StreamCacheEntry)
	a.streamCacheMutex.Unlock()

	a.sourceFailuresMutex.Lock()
	a.sourceFailures = make(map[string]*SourceFailure)
	a.sourceFailuresMutex.Unlock()

	fmt.Println("[Cache] Todo o cache foi limpo")
}

// === SISTEMA DE CACHE INTELIGENTE COM VALIDAÃ‡ÃƒO ===

// ValidateStreamURL verifica se uma URL de stream ainda Ã© acessÃ­vel
// Usa HEAD request para ser rÃ¡pido e nÃ£o consumir banda
func (a *App) ValidateStreamURL(url string) (bool, error) {
	if url == "" {
		return false, fmt.Errorf("URL vazia")
	}

	fmt.Printf("[ValidateURL] Verificando: %s\n", url)

	// Cria request HEAD (nÃ£o baixa o conteÃºdo, sÃ³ verifica headers)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, err
	}

	// Configura headers baseado no tipo de URL
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	if strings.Contains(url, "lightspeedst.net") || strings.Contains(url, "animefire") {
		req.Header.Set("Referer", "https://animefire.plus/")
		req.Header.Set("Origin", "https://animefire.plus")
	} else if strings.Contains(url, "sharepoint") || strings.Contains(url, "microsoft") {
		req.Header.Set("Referer", "https://myanime.sharepoint.com/")
	} else if strings.Contains(url, "allanime") || strings.Contains(url, "gogoanime") {
		req.Header.Set("Referer", "https://allanime.to/")
	}

	resp, err := a.validationClient.Do(req)
	if err != nil {
		fmt.Printf("[ValidateURL] Erro na requisiÃ§Ã£o: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	// Status 2xx ou 3xx Ã© vÃ¡lido
	isValid := resp.StatusCode >= 200 && resp.StatusCode < 400
	fmt.Printf("[ValidateURL] Status: %d, VÃ¡lido: %v\n", resp.StatusCode, isValid)

	return isValid, nil
}

// GetValidatedStreamCache obtÃ©m stream do cache, validando se ainda funciona
func (a *App) GetValidatedStreamCache(key string) (string, bool) {
	a.streamCacheMutex.RLock()
	entry, exists := a.streamCache[key]
	a.streamCacheMutex.RUnlock()

	if !exists {
		return "", false
	}

	// Verifica se expirou
	if time.Now().After(entry.ExpiresAt) {
		fmt.Printf("[SmartCache] Cache expirado para: %s\n", key)
		return "", false
	}

	// Verifica se precisa revalidar
	if !entry.IsValidated || time.Since(entry.LastValidAt) > 2*time.Minute {
		fmt.Printf("[SmartCache] Validando URL do cache: %s\n", entry.URL)

		// Valida em goroutine para nÃ£o bloquear, mas retorna o cache atual
		go func(e *StreamCacheEntry, k string) {
			valid, err := a.ValidateStreamURL(e.URL)

			a.streamCacheMutex.Lock()
			defer a.streamCacheMutex.Unlock()

			if cached, ok := a.streamCache[k]; ok {
				if valid {
					cached.IsValidated = true
					cached.LastValidAt = time.Now()
					cached.FailCount = 0
					fmt.Printf("[SmartCache] URL validada com sucesso: %s\n", cached.URL)
				} else {
					cached.FailCount++
					fmt.Printf("[SmartCache] URL invÃ¡lida (falha %d): %s - %v\n", cached.FailCount, cached.URL, err)

					// Se falhou 3 vezes, remove do cache
					if cached.FailCount >= 3 {
						delete(a.streamCache, k)
						a.recordSourceFailure(cached.Source, "URL invÃ¡lida apÃ³s mÃºltiplas tentativas")
						fmt.Printf("[SmartCache] Cache removido apÃ³s 3 falhas: %s\n", k)
					}
				}
			}
		}(entry, key)
	}

	return entry.URL, true
}

// SetStreamCache armazena URL de stream no cache inteligente
func (a *App) SetStreamCache(key string, url string, source string, ttl time.Duration) {
	a.streamCacheMutex.Lock()
	defer a.streamCacheMutex.Unlock()

	a.streamCache[key] = &StreamCacheEntry{
		URL:         url,
		Source:      source,
		ExpiresAt:   time.Now().Add(ttl),
		LastValidAt: time.Now(),
		IsValidated: true, // Assume vÃ¡lido no momento do cache
		FailCount:   0,
	}

	fmt.Printf("[SmartCache] Stream cacheado: %s -> %s (source: %s)\n", key, url, source)
}

// InvalidateStreamCache invalida uma entrada especÃ­fica do cache de streams
func (a *App) InvalidateStreamCache(key string) {
	a.streamCacheMutex.Lock()
	defer a.streamCacheMutex.Unlock()

	if entry, ok := a.streamCache[key]; ok {
		a.recordSourceFailure(entry.Source, "Cache invalidado manualmente")
		delete(a.streamCache, key)
		fmt.Printf("[SmartCache] Cache invalidado: %s\n", key)
	}
}

// recordSourceFailure registra falha de uma fonte
func (a *App) recordSourceFailure(source string, reason string) {
	a.sourceFailuresMutex.Lock()
	defer a.sourceFailuresMutex.Unlock()

	if failure, exists := a.sourceFailures[source]; exists {
		failure.FailCount++
		failure.FailedAt = time.Now()
		failure.LastError = reason

		// Backoff exponencial: 30s, 1min, 2min, 5min, 10min
		backoffMinutes := []time.Duration{30 * time.Second, 1 * time.Minute, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute}
		backoffIndex := failure.FailCount - 1
		if backoffIndex >= len(backoffMinutes) {
			backoffIndex = len(backoffMinutes) - 1
		}
		failure.RetryAfter = time.Now().Add(backoffMinutes[backoffIndex])

		fmt.Printf("[SourceTracker] Falha %d para %s: %s (retry apÃ³s %v)\n",
			failure.FailCount, source, reason, backoffMinutes[backoffIndex])
	} else {
		a.sourceFailures[source] = &SourceFailure{
			Source:     source,
			FailedAt:   time.Now(),
			FailCount:  1,
			LastError:  reason,
			RetryAfter: time.Now().Add(30 * time.Second),
		}
		fmt.Printf("[SourceTracker] Primeira falha para %s: %s\n", source, reason)
	}
}

// recordSourceSuccess registra sucesso de uma fonte
func (a *App) recordSourceSuccess(source string) {
	a.sourceFailuresMutex.Lock()
	defer a.sourceFailuresMutex.Unlock()

	// Reseta falhas apÃ³s sucesso
	if failure, exists := a.sourceFailures[source]; exists {
		fmt.Printf("[SourceTracker] Fonte %s recuperada apÃ³s %d falhas\n", source, failure.FailCount)
		delete(a.sourceFailures, source)
	}
}

// IsSourceAvailable verifica se uma fonte estÃ¡ disponÃ­vel (nÃ£o em cooldown)
func (a *App) IsSourceAvailable(source string) bool {
	a.sourceFailuresMutex.RLock()
	defer a.sourceFailuresMutex.RUnlock()

	if failure, exists := a.sourceFailures[source]; exists {
		if time.Now().Before(failure.RetryAfter) {
			fmt.Printf("[SourceTracker] Fonte %s em cooldown atÃ© %v\n", source, failure.RetryAfter)
			return false
		}
	}
	return true
}

// GetAlternativeSource retorna a melhor fonte alternativa disponÃ­vel
func (a *App) GetAlternativeSource(excludeSources ...string) string {
	sources := []string{"AllAnime", "AnimeFire", "Enime", "Consumet"}

	excludeMap := make(map[string]bool)
	for _, s := range excludeSources {
		excludeMap[s] = true
	}

	a.sourceFailuresMutex.RLock()
	defer a.sourceFailuresMutex.RUnlock()

	for _, source := range sources {
		if excludeMap[source] {
			continue
		}

		// Verifica se a fonte estÃ¡ disponÃ­vel
		if failure, exists := a.sourceFailures[source]; exists {
			if time.Now().Before(failure.RetryAfter) {
				continue // Ainda em cooldown
			}
		}

		fmt.Printf("[SourceTracker] Fonte alternativa selecionada: %s\n", source)
		return source
	}

	// Se todas estÃ£o em cooldown, retorna a com menor tempo de espera
	var bestSource string
	var earliestRetry time.Time

	for _, source := range sources {
		if excludeMap[source] {
			continue
		}

		if failure, exists := a.sourceFailures[source]; exists {
			if bestSource == "" || failure.RetryAfter.Before(earliestRetry) {
				bestSource = source
				earliestRetry = failure.RetryAfter
			}
		} else {
			return source // Fonte sem falhas registradas
		}
	}

	return bestSource
}

// SourceStatus representa o status de uma fonte de vÃ­deo para o frontend
type SourceStatus struct {
	Name        string `json:"name"`
	IsAvailable bool   `json:"isAvailable"`
	FailCount   int    `json:"failCount"`
	LastError   string `json:"lastError,omitempty"`
	RetryAfter  string `json:"retryAfter,omitempty"`
	CachedURLs  int    `json:"cachedUrls"`
}

// CacheStats representa estatÃ­sticas do cache para o frontend
type CacheStats struct {
	Sources      []SourceStatus `json:"sources"`
	TotalStreams int            `json:"totalStreams"`
	TotalCache   int            `json:"totalCache"`
}

// GetCacheStats retorna estatÃ­sticas do cache e status das fontes
func (a *App) GetCacheStats() CacheStats {
	stats := CacheStats{
		Sources: make([]SourceStatus, 0),
	}

	// Conta caches
	a.cacheMutex.RLock()
	stats.TotalCache = len(a.cache)
	a.cacheMutex.RUnlock()

	a.streamCacheMutex.RLock()
	stats.TotalStreams = len(a.streamCache)

	// Conta URLs por fonte
	sourceCounts := make(map[string]int)
	for _, entry := range a.streamCache {
		sourceCounts[entry.Source]++
	}
	a.streamCacheMutex.RUnlock()

	// Status de cada fonte
	sources := []string{"AllAnime", "AnimeFire", "Enime", "Consumet"}

	a.sourceFailuresMutex.RLock()
	defer a.sourceFailuresMutex.RUnlock()

	for _, name := range sources {
		status := SourceStatus{
			Name:        name,
			IsAvailable: true,
			CachedURLs:  sourceCounts[name],
		}

		if failure, exists := a.sourceFailures[name]; exists {
			status.FailCount = failure.FailCount
			status.LastError = failure.LastError

			if time.Now().Before(failure.RetryAfter) {
				status.IsAvailable = false
				status.RetryAfter = failure.RetryAfter.Format("15:04:05")
			}
		}

		stats.Sources = append(stats.Sources, status)
	}

	return stats
}

// ResetSourceFailures reseta todas as falhas de fontes (Ãºtil para debug)
func (a *App) ResetSourceFailures() {
	a.sourceFailuresMutex.Lock()
	defer a.sourceFailuresMutex.Unlock()

	a.sourceFailures = make(map[string]*SourceFailure)
	fmt.Println("[SourceTracker] Todas as falhas foram resetadas")
}

// startVideoProxy inicia um servidor HTTP local para fazer proxy do vÃ­deo
func (a *App) startVideoProxy() error {
	if a.proxyServer != nil {
		return nil // JÃ¡ estÃ¡ rodando
	}

	// Encontra uma porta livre
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("erro ao encontrar porta livre: %w", err)
	}
	a.proxyPort = listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/video", a.handleVideoProxy)
	mux.HandleFunc("/proxy/", a.handleGenericProxy)         // Para segmentos HLS
	mux.HandleFunc("/manga-image", a.handleMangaImageProxy) // Para imagens de mangÃ¡ com cache

	a.proxyServer = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", a.proxyPort),
		Handler: mux,
	}

	go func() {
		fmt.Printf("[VideoProxy] Iniciando servidor na porta %d\n", a.proxyPort)
		if err := a.proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[VideoProxy] Erro: %v\n", err)
		}
	}()

	// Espera o servidor iniciar
	time.Sleep(100 * time.Millisecond)
	return nil
}

// handleGenericProxy faz proxy de qualquer URL (para segmentos HLS)
func (a *App) handleGenericProxy(w http.ResponseWriter, r *http.Request) {
	// URL estÃ¡ no path: /proxy/https://...
	targetURL := strings.TrimPrefix(r.URL.Path, "/proxy/")
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	if targetURL == "" {
		http.Error(w, "URL nÃ£o especificada", http.StatusBadRequest)
		return
	}

	fmt.Printf("[GenericProxy] Proxy de: %s\n", targetURL)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		http.Error(w, "Erro ao criar request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Copia Range header se presente
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[GenericProxy] Erro: %v\n", err)
		http.Error(w, "Erro ao acessar recurso", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Headers CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")

	// Copia headers da resposta
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleMangaImageProxy faz proxy de imagens de mangÃ¡ com cache em memÃ³ria
func (a *App) handleMangaImageProxy(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")
	referer := r.URL.Query().Get("referer")

	if imageURL == "" {
		http.Error(w, "URL nÃ£o especificada", http.StatusBadRequest)
		return
	}

	// Verifica cache
	a.imageCacheMutex.RLock()
	cachedData, exists := a.imageCache[imageURL]
	a.imageCacheMutex.RUnlock()

	if exists && len(cachedData) > 0 {
		// Serve do cache
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(cachedData)
		return
	}

	// Baixa a imagem
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		http.Error(w, "Erro ao criar request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")

	if referer != "" {
		req.Header.Set("Referer", referer)
	} else {
		// Extrai referer da URL
		parsed, _ := url.Parse(imageURL)
		if parsed != nil {
			req.Header.Set("Referer", fmt.Sprintf("%s://%s/", parsed.Scheme, parsed.Host))
		}
	}

	resp, err := a.imageClient.Do(req)
	if err != nil {
		fmt.Printf("[MangaImageProxy] Erro ao baixar: %v\n", err)
		http.Error(w, "Erro ao baixar imagem", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		http.Error(w, "Imagem nÃ£o encontrada", resp.StatusCode)
		return
	}

	// LÃª a imagem
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Erro ao ler imagem", http.StatusInternalServerError)
		return
	}

	// Salva no cache (limite de 100MB total)
	a.imageCacheMutex.Lock()
	// Limpa cache se muito grande (simples, pode melhorar depois)
	if len(a.imageCache) > 500 {
		// Remove metade das entradas mais antigas
		count := 0
		for k := range a.imageCache {
			if count > 250 {
				break
			}
			delete(a.imageCache, k)
			count++
		}
	}
	a.imageCache[imageURL] = data
	a.imageCacheMutex.Unlock()

	// Detecta content-type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	// Responde
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Write(data)
}

// handleVideoProxy faz proxy do vÃ­deo remoto para o cliente local
func (a *App) handleVideoProxy(w http.ResponseWriter, r *http.Request) {
	a.proxyMutex.RLock()
	videoURL := a.currentVideoURL
	a.proxyMutex.RUnlock()

	if videoURL == "" {
		http.Error(w, "Nenhum vÃ­deo configurado", http.StatusBadRequest)
		return
	}

	fmt.Printf("[VideoProxy] Fazendo proxy de: %s\n", videoURL)

	// Cria request para o servidor remoto
	client := &http.Client{
		Timeout: 0, // Sem timeout para streaming
	}

	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		http.Error(w, "Erro ao criar request", http.StatusInternalServerError)
		return
	}

	// Copia headers da requisiÃ§Ã£o original (para suportar Range requests)
	for key, values := range r.Header {
		for _, value := range values {
			if key == "Range" || key == "Accept" || key == "Accept-Encoding" {
				req.Header.Add(key, value)
			}
		}
	}

	// Headers comuns para parecer um navegador real
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Define Referer baseado na URL de origem
	if strings.Contains(videoURL, "lightspeedst.net") {
		// LightSpeed CDN - precisa do referer correto do AnimeFire
		req.Header.Set("Referer", "https://animefire.plus/")
		req.Header.Set("Origin", "https://animefire.plus")
	} else if strings.Contains(videoURL, "animefire") {
		req.Header.Set("Referer", "https://animefire.plus/")
		req.Header.Set("Origin", "https://animefire.plus")
	} else if strings.Contains(videoURL, "sharepoint") || strings.Contains(videoURL, "microsoft") {
		// SharePoint precisa de headers especÃ­ficos
		req.Header.Set("Referer", "https://myanime.sharepoint.com/")
		req.Header.Set("Origin", "https://myanime.sharepoint.com")
		req.Header.Set("Accept", "*/*")
	} else if strings.Contains(videoURL, "allanime") || strings.Contains(videoURL, "gogoanime") {
		req.Header.Set("Referer", "https://allanime.to/")
		req.Header.Set("Origin", "https://allanime.to")
	} else {
		req.Header.Set("Referer", "https://google.com/")
	}

	// Headers adicionais para compatibilidade
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Dest", "video")
	req.Header.Set("Sec-Fetch-Mode", "no-cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[VideoProxy] Erro na requisiÃ§Ã£o: %v\n", err)
		http.Error(w, "Erro ao acessar vÃ­deo", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Verifica se a resposta foi bem sucedida
	if resp.StatusCode >= 400 {
		fmt.Printf("[VideoProxy] Servidor remoto retornou erro: %d %s\n", resp.StatusCode, resp.Status)
		// Se for um erro de autenticaÃ§Ã£o, tenta sem proxy
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			fmt.Printf("[VideoProxy] Erro de autenticaÃ§Ã£o - URL pode requerer acesso direto\n")
		}
	}

	// Headers CORS e de resposta
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range, Accept, Content-Type")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")

	// Se for m3u8, reescreve as URLs para usar nosso proxy
	isM3U8 := strings.Contains(videoURL, ".m3u8") || strings.Contains(resp.Header.Get("Content-Type"), "mpegurl")

	if isM3U8 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Erro ao ler m3u8", http.StatusInternalServerError)
			return
		}

		// Reescreve URLs no m3u8 para usar nosso proxy
		content := string(body)
		lines := strings.Split(content, "\n")
		var newLines []string

		baseURL := videoURL[:strings.LastIndex(videoURL, "/")+1]

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				// MantÃ©m comentÃ¡rios e linhas vazias
				newLines = append(newLines, line)
			} else {
				// Ã‰ uma URL de segmento
				var fullURL string
				if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
					fullURL = line
				} else {
					fullURL = baseURL + line
				}
				// Reescreve para usar nosso proxy
				proxyURL := fmt.Sprintf("http://127.0.0.1:%d/proxy/%s", a.proxyPort, fullURL)
				newLines = append(newLines, proxyURL)
			}
		}

		newContent := strings.Join(newLines, "\n")
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(newContent)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(newContent))
		return
	}

	// Copia headers da resposta remota
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Garante Content-Type correto para MP4
	if strings.HasSuffix(videoURL, ".mp4") {
		w.Header().Set("Content-Type", "video/mp4")
	}

	w.WriteHeader(resp.StatusCode)

	// Faz streaming do corpo
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		fmt.Printf("[VideoProxy] Erro no streaming: %v\n", err)
	}
}

// GetProxyURLForVideo retorna a URL do proxy local para um vÃ­deo
func (a *App) GetProxyURLForVideo(videoURL string) (string, error) {
	// Inicia o proxy se ainda nÃ£o estiver rodando
	if err := a.startVideoProxy(); err != nil {
		return "", err
	}

	// Configura a URL atual
	a.proxyMutex.Lock()
	a.currentVideoURL = videoURL
	a.proxyMutex.Unlock()

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d/video", a.proxyPort)
	fmt.Printf("[GetProxyURLForVideo] Proxy URL: %s -> %s\n", proxyURL, videoURL)
	return proxyURL, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.client = goanime.NewClient()
	a.User = store.LoadUser()

	// Inicializa Discord OAuth
	initDiscordOAuth()

	// PrÃ©-carrega dados em background para inicializaÃ§Ã£o rÃ¡pida
	go a.preloadData()

	// Limpa cache expirado periodicamente
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			a.cleanExpiredCache()
		}
	}()
}

// preloadData carrega dados em background para melhor UX
// NÃ£o bloqueia - carrega progressivamente e emite eventos
func (a *App) preloadData() {
	// Carrega trending do AniList PRIMEIRO (prioridade alta - usado no hero)
	// Ã‰ o mais rÃ¡pido, entÃ£o carrega primeiro
	go func() {
		if animes, err := a.fetchTrendingInternal(15); err == nil && len(animes) > 0 {
			a.cacheMutex.Lock()
			a.trendingCache = animes
			a.cacheMutex.Unlock()
			fmt.Println("[preload] Trending carregado:", len(animes))
		}
	}()

	// Carrega top animes em background (prioridade baixa)
	// Usa versÃ£o otimizada que nÃ£o bloqueia
	go func() {
		if animes, err := a.fetchTopAnimesOptimized(); err == nil && len(animes) > 0 {
			a.cacheMutex.Lock()
			a.topAnimesCache = animes
			a.cacheMutex.Unlock()
			fmt.Println("[preload] Top animes carregado:", len(animes))
		}
	}()

	// Marca como inicializado imediatamente - dados carregam em background
	a.initMutex.Lock()
	a.initialized = true
	a.initMutex.Unlock()

	fmt.Println("[startup] PrÃ©-carregamento iniciado em background")
}

// fetchTopAnimesOptimized busca top animes de forma otimizada
// Primeiro tenta Jikan (rÃ¡pido), depois enriquece com fontes reais em background
func (a *App) fetchTopAnimesOptimized() ([]store.SavedAnime, error) {
	// FASE 1: Busca rÃ¡pida do Jikan (jÃ¡ tem imagens)
	jikanAnimes, err := jikan.FetchTopAnimes()
	if err == nil && len(jikanAnimes) > 0 {
		result := make([]store.SavedAnime, 0, 20)
		for i, item := range jikanAnimes {
			if i >= 20 {
				break
			}
			result = append(result, store.SavedAnime{
				Title: item.Title,
				Image: item.Image,
				URL:   "", // SerÃ¡ preenchido quando o usuÃ¡rio clicar
			})
		}

		// FASE 2: Em background, busca URLs reais (nÃ£o bloqueia)
		go a.enrichTopAnimesWithURLs(result)

		return result, nil
	}

	// Fallback: busca nas fontes reais (mais lento)
	return a.fetchTopAnimesInternal()
}

// enrichTopAnimesWithURLs busca URLs reais em background
func (a *App) enrichTopAnimesWithURLs(animes []store.SavedAnime) {
	if a.client == nil {
		a.client = goanime.NewClient()
	}

	sem := make(chan struct{}, 3) // Apenas 3 paralelos para nÃ£o sobrecarregar
	var wg sync.WaitGroup

	for i := range animes {
		if animes[i].URL != "" {
			continue
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results, err := a.client.SearchAnime(animes[idx].Title, nil)
			if err != nil || len(results) == 0 {
				return
			}

			// Atualiza no cache
			a.cacheMutex.Lock()
			if idx < len(a.topAnimesCache) {
				a.topAnimesCache[idx].URL = results[0].URL
				a.topAnimesCache[idx].Source = results[0].Source
			}
			a.cacheMutex.Unlock()
		}(i)
	}

	wg.Wait()
	fmt.Println("[preload] URLs dos top animes enriquecidas")
}

// fetchTrendingInternal busca trending sem cache
func (a *App) fetchTrendingInternal(limit int) ([]*AniListAnime, error) {
	results, err := anilist.GetTrending(limit)
	if err != nil {
		return nil, err
	}

	animes := make([]*AniListAnime, 0, len(results))
	for _, m := range results {
		animes = append(animes, convertAniListToFrontend(m))
	}
	return animes, nil
}

// === WINDOW CONTROLS ===

// SetFullscreen coloca a janela em tela cheia
func (a *App) SetFullscreen(fullscreen bool) {
	if fullscreen {
		runtime.WindowFullscreen(a.ctx)
	} else {
		runtime.WindowUnfullscreen(a.ctx)
	}
}

// ToggleFullscreen alterna entre tela cheia e janela normal
func (a *App) ToggleFullscreen() {
	runtime.WindowToggleMaximise(a.ctx)
}

// IsFullscreen verifica se a janela estÃ¡ em tela cheia
func (a *App) WindowMaximise() {
	runtime.WindowMaximise(a.ctx)
}

func (a *App) WindowUnmaximise() {
	runtime.WindowUnmaximise(a.ctx)
}

// --- FUNÃ‡Ã•ES EXPORTADAS ---

func (a *App) GetCurrentUser() *store.UserData {
	return a.User
}

func (a *App) CreateUser(username string, avatar string) *store.UserData {
	newUser := &store.UserData{
		Username:     username,
		Avatar:       avatar,
		History:      []store.SavedAnime{},
		Favorites:    []store.SavedAnime{},
		WatchHistory: []store.WatchedEpisode{},
		Settings:     store.GetDefaultSettings(),
	}
	a.User = newUser
	store.SaveUser(a.User)
	return a.User
}

// === FAVORITOS ===

// GetFavorites retorna a lista de favoritos do utilizador
func (a *App) GetFavorites() []store.SavedAnime {
	if a.User == nil {
		return []store.SavedAnime{}
	}
	return a.User.Favorites
}

// AddToFavorites adiciona um anime aos favoritos
func (a *App) AddToFavorites(anime store.SavedAnime) bool {
	if a.User == nil {
		return false
	}

	// Verifica se jÃ¡ existe
	for _, fav := range a.User.Favorites {
		if fav.URL == anime.URL || fav.Title == anime.Title {
			return false // JÃ¡ existe
		}
	}

	a.User.Favorites = append(a.User.Favorites, anime)
	store.SaveUser(a.User)
	return true
}

// RemoveFromFavorites remove um anime dos favoritos
func (a *App) RemoveFromFavorites(animeURL string) bool {
	if a.User == nil {
		return false
	}

	for i, fav := range a.User.Favorites {
		if fav.URL == animeURL {
			a.User.Favorites = append(a.User.Favorites[:i], a.User.Favorites[i+1:]...)
			store.SaveUser(a.User)
			return true
		}
	}
	return false
}

// IsFavorite verifica se um anime estÃ¡ nos favoritos
func (a *App) IsFavorite(animeURL string) bool {
	if a.User == nil {
		return false
	}
	for _, fav := range a.User.Favorites {
		if fav.URL == animeURL {
			return true
		}
	}
	return false
}

// === HISTÃ“RICO DE VISUALIZAÃ‡ÃƒO ===

// GetWatchHistory retorna os Ãºltimos episÃ³dios assistidos
func (a *App) GetWatchHistory() []store.WatchedEpisode {
	if a.User == nil {
		return []store.WatchedEpisode{}
	}
	return a.User.WatchHistory
}

// AddToWatchHistory adiciona um episÃ³dio ao histÃ³rico
func (a *App) AddToWatchHistory(episode store.WatchedEpisode) {
	if a.User == nil {
		return
	}

	// Define timestamp se nÃ£o especificado
	if episode.WatchedAt == "" {
		episode.WatchedAt = time.Now().Format(time.RFC3339)
	}

	// Remove entrada duplicada (mesmo episÃ³dio)
	for i, e := range a.User.WatchHistory {
		if e.EpisodeURL == episode.EpisodeURL {
			a.User.WatchHistory = append(a.User.WatchHistory[:i], a.User.WatchHistory[i+1:]...)
			break
		}
	}

	// Adiciona no inÃ­cio (mais recente primeiro)
	a.User.WatchHistory = append([]store.WatchedEpisode{episode}, a.User.WatchHistory...)

	// Limita a 50 entradas
	if len(a.User.WatchHistory) > 50 {
		a.User.WatchHistory = a.User.WatchHistory[:50]
	}

	store.SaveUser(a.User)
}

// === CONFIGURAÃ‡Ã•ES ===

// GetSettings retorna as configuraÃ§Ãµes do utilizador
func (a *App) GetSettings() store.UserSettings {
	if a.User == nil {
		return store.GetDefaultSettings()
	}
	return a.User.Settings
}

// SaveSettings guarda as configuraÃ§Ãµes do utilizador
func (a *App) SaveSettings(settings store.UserSettings) bool {
	if a.User == nil {
		return false
	}
	a.User.Settings = settings
	store.SaveUser(a.User)
	return true
}

// === EXPORT / IMPORT ===

// ExportUserData exporta todos os dados do utilizador como JSON string
func (a *App) ExportUserData() (string, error) {
	if a.User == nil {
		return "", fmt.Errorf("utilizador nÃ£o encontrado")
	}

	data, err := json.MarshalIndent(a.User, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ImportUserData importa dados do utilizador a partir de JSON string
func (a *App) ImportUserData(jsonData string) error {
	var userData store.UserData
	if err := json.Unmarshal([]byte(jsonData), &userData); err != nil {
		return fmt.Errorf("erro ao processar JSON: %w", err)
	}

	// Valida dados mÃ­nimos
	if userData.Username == "" {
		return fmt.Errorf("nome de utilizador invÃ¡lido")
	}

	// Garante que settings tem valores padrÃ£o se vazios
	if userData.Settings.ContentLanguage == "" {
		userData.Settings.ContentLanguage = "all"
	}

	a.User = &userData
	store.SaveUser(a.User)
	return nil
}

// === ANILIST API (Imagens HD e Trending) ===

// AniListAnime representa um anime com dados do AniList para o frontend
type AniListAnime struct {
	ID           int      `json:"id"`
	MalID        int      `json:"malId"`
	Title        string   `json:"title"`
	TitleEnglish string   `json:"titleEnglish"`
	TitleNative  string   `json:"titleNative"`
	Description  string   `json:"description"`
	Image        string   `json:"image"`  // Cover HD
	Banner       string   `json:"banner"` // Banner para hero section
	Color        string   `json:"color"`  // Cor predominante
	Genres       []string `json:"genres"`
	Episodes     int      `json:"episodes"`
	Duration     int      `json:"duration"`
	Status       string   `json:"status"`
	Season       string   `json:"season"`
	Year         int      `json:"year"`
	Score        int      `json:"score"` // AverageScore (0-100)
	Popularity   int      `json:"popularity"`
	Studio       string   `json:"studio"`
	TrailerURL   string   `json:"trailerUrl"`
	IsAiring     bool     `json:"isAiring"`
	NextEpisode  int      `json:"nextEpisode"`
}

// convertAniListToFrontend converte AniList media para estrutura do frontend
func convertAniListToFrontend(m *anilist.AnimeMedia) *AniListAnime {
	if m == nil {
		return nil
	}

	studio := ""
	if len(m.Studios.Nodes) > 0 {
		studio = m.Studios.Nodes[0].Name
	}

	nextEp := 0
	isAiring := m.Status == "RELEASING"
	if m.NextAiringEpisode != nil {
		nextEp = m.NextAiringEpisode.Episode
	}

	return &AniListAnime{
		ID:           m.ID,
		MalID:        m.MALID,
		Title:        m.GetBestTitle(),
		TitleEnglish: m.Title.English,
		TitleNative:  m.Title.Native,
		Description:  m.Description,
		Image:        m.GetBestImage(),
		Banner:       m.BannerImage,
		Color:        m.CoverImage.Color,
		Genres:       m.Genres,
		Episodes:     m.Episodes,
		Duration:     m.Duration,
		Status:       m.Status,
		Season:       m.Season,
		Year:         m.SeasonYear,
		Score:        m.AverageScore,
		Popularity:   m.Popularity,
		Studio:       studio,
		TrailerURL:   m.GetTrailerURL(),
		IsAiring:     isAiring,
		NextEpisode:  nextEp,
	}
}

// GetTrendingAnimes retorna animes em alta do AniList (com imagens HD)
func (a *App) GetTrendingAnimes(limit int) ([]*AniListAnime, error) {
	if limit <= 0 {
		limit = 15
	}

	// Verifica cache prÃ©-carregado primeiro (mais rÃ¡pido)
	a.cacheMutex.RLock()
	if len(a.trendingCache) > 0 {
		cached := a.trendingCache
		a.cacheMutex.RUnlock()
		fmt.Println("[GetTrendingAnimes] Retornando cache prÃ©-carregado")
		if len(cached) > limit {
			return cached[:limit], nil
		}
		return cached, nil
	}
	a.cacheMutex.RUnlock()

	// Verifica cache com TTL
	cacheKey := fmt.Sprintf("trending:%d", limit)
	if cached, ok := a.getCache(cacheKey); ok {
		fmt.Println("[GetTrendingAnimes] Retornando cache TTL")
		return cached.([]*AniListAnime), nil
	}

	// Busca e cacheia
	animes, err := a.fetchTrendingInternal(limit)
	if err != nil {
		return nil, err
	}

	a.setCache(cacheKey, animes, CacheTTLTrending)

	// Atualiza cache prÃ©-carregado tambÃ©m
	a.cacheMutex.Lock()
	a.trendingCache = animes
	a.cacheMutex.Unlock()

	return animes, nil
}

// GetPopularAnimes retorna animes populares do AniList com cache
func (a *App) GetPopularAnimes(limit int) ([]*AniListAnime, error) {
	if limit <= 0 {
		limit = 20
	}

	// Verifica cache
	cacheKey := fmt.Sprintf("popular:%d", limit)
	if cached, ok := a.getCache(cacheKey); ok {
		fmt.Println("[GetPopularAnimes] Retornando cache")
		return cached.([]*AniListAnime), nil
	}

	results, err := anilist.GetPopular(limit)
	if err != nil {
		return nil, err
	}

	animes := make([]*AniListAnime, 0, len(results))
	for _, m := range results {
		animes = append(animes, convertAniListToFrontend(m))
	}

	// Salva no cache por 30 minutos
	a.setCache(cacheKey, animes, CacheTTLTrending)
	fmt.Printf("[GetPopularAnimes] Cached %d animes\n", len(animes))

	return animes, nil
}

// SearchAniList busca animes no AniList (imagens HD)
func (a *App) SearchAniList(query string, limit int) ([]*AniListAnime, error) {
	if limit <= 0 {
		limit = 10
	}

	results, err := anilist.SearchAnime(query, limit)
	if err != nil {
		return nil, err
	}

	animes := make([]*AniListAnime, 0, len(results))
	for _, m := range results {
		animes = append(animes, convertAniListToFrontend(m))
	}

	return animes, nil
}

// GetAnimeHDImage busca imagem HD de um anime pelo tÃ­tulo
func (a *App) GetAnimeHDImage(title string) (map[string]string, error) {
	image, banner, color, err := anilist.GetHDImage(title)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"image":  image,
		"banner": banner,
		"color":  color,
	}, nil
}

// === CONSUMET API (Fallback de Streaming) ===

// ConsometAnime representa um anime do Consumet
type ConsumetAnime struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Image         string   `json:"image"`
	TotalEpisodes int      `json:"totalEpisodes"`
	SubOrDub      string   `json:"subOrDub"`
	Genres        []string `json:"genres"`
	Description   string   `json:"description"`
	Provider      string   `json:"provider"`
}

// ConsumetEpisode representa um episÃ³dio do Consumet
type ConsumetEpisode struct {
	ID       string `json:"id"`
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Provider string `json:"provider"`
}

// SearchConsumet busca animes via Consumet API (fallback)
func (a *App) SearchConsumet(query string) ([]ConsumetAnime, error) {
	result, err := consumet.Search(query, consumet.ProviderGogoanime)
	if err != nil {
		return nil, err
	}

	animes := make([]ConsumetAnime, 0, len(result.Results))
	for _, r := range result.Results {
		animes = append(animes, ConsumetAnime{
			ID:            r.ID,
			Title:         r.Title,
			Image:         r.Image,
			TotalEpisodes: r.TotalEpisodes,
			SubOrDub:      r.SubOrDub,
			Genres:        r.Genres,
			Description:   r.Description,
			Provider:      consumet.ProviderGogoanime,
		})
	}

	return animes, nil
}

// GetConsumetEpisodes busca episÃ³dios de um anime via Consumet
func (a *App) GetConsumetEpisodes(animeID string, provider string) ([]ConsumetEpisode, error) {
	if provider == "" {
		provider = consumet.ProviderGogoanime
	}

	details, err := consumet.GetAnimeInfo(animeID, provider)
	if err != nil {
		return nil, err
	}

	episodes := make([]ConsumetEpisode, 0, len(details.Episodes))
	for _, ep := range details.Episodes {
		episodes = append(episodes, ConsumetEpisode{
			ID:       ep.ID,
			Number:   ep.Number,
			Title:    ep.Title,
			Provider: provider,
		})
	}

	return episodes, nil
}

// GetConsumetStream busca URL de streaming via Consumet (fallback)
func (a *App) GetConsumetStream(episodeID string, provider string) (string, error) {
	if provider == "" {
		provider = consumet.ProviderGogoanime
	}

	url, isM3U8, err := consumet.GetBestStream(episodeID, provider)
	if err != nil {
		return "", err
	}

	fmt.Printf("[Consumet] Stream encontrado: %s (m3u8: %v)\n", url, isM3U8)
	return url, nil
}

// GetStreamWithFallback tenta mÃºltiplas fontes para obter stream
func (a *App) GetStreamWithFallback(animeTitle string, episodeNumber int) (string, error) {
	fmt.Printf("[GetStreamWithFallback] Buscando stream para: %s Ep.%d\n", animeTitle, episodeNumber)

	// Verifica cache
	cacheKey := fmt.Sprintf("stream_fb:%s:%d", strings.ToLower(animeTitle), episodeNumber)
	if cached, ok := a.getCache(cacheKey); ok {
		fmt.Println("[GetStreamWithFallback] Cache hit!")
		return cached.(string), nil
	}

	// Tenta via Consumet como fallback
	url, _, err := consumet.FindAnimeAndGetStream(animeTitle, episodeNumber)
	if err != nil {
		return "", fmt.Errorf("nenhuma fonte encontrada: %w", err)
	}

	// Salva no cache
	a.setCache(cacheKey, url, CacheTTLStream)

	return url, nil
}

// ============================================================================
// SMART ROUTER - Busca inteligente de streams com circuit breaker
// ============================================================================

// SmartStreamResult Ã© o resultado da busca inteligente de stream
type SmartStreamResult struct {
	URL      string  `json:"url"`
	Source   string  `json:"source"`
	Duration float64 `json:"duration"` // em milissegundos
	Success  bool    `json:"success"`
	Error    string  `json:"error,omitempty"`
}

// GetSmartStream usa o Smart Router para buscar stream com fallback automÃ¡tico
// Esta funÃ§Ã£o tenta mÃºltiplas fontes com timeout e circuit breaker
func (a *App) GetSmartStream(animeTitle string, episodeNumber int) (*SmartStreamResult, error) {
	fmt.Printf("[SmartStream] Buscando: %s Ep.%d\n", animeTitle, episodeNumber)

	// Verifica cache primeiro
	cacheKey := fmt.Sprintf("smart_stream:%s:%d", strings.ToLower(animeTitle), episodeNumber)
	if cached, ok := a.getCache(cacheKey); ok {
		fmt.Println("[SmartStream] Cache hit!")
		result := cached.(*SmartStreamResult)
		return result, nil
	}

	// Usa o Smart Router
	result := a.streamRouter.GetStream(animeTitle, episodeNumber)

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

	// Salva no cache
	a.setCache(cacheKey, smartResult, CacheTTLStream)

	return smartResult, nil
}

// GetSmartStreamParallel busca em todas as fontes simultaneamente
// Retorna assim que a primeira fonte responder com sucesso
func (a *App) GetSmartStreamParallel(animeTitle string, episodeNumber int) (*SmartStreamResult, error) {
	fmt.Printf("[SmartStreamParallel] Buscando: %s Ep.%d\n", animeTitle, episodeNumber)

	// Verifica cache
	cacheKey := fmt.Sprintf("smart_stream:%s:%d", strings.ToLower(animeTitle), episodeNumber)
	if cached, ok := a.getCache(cacheKey); ok {
		fmt.Println("[SmartStreamParallel] Cache hit!")
		return cached.(*SmartStreamResult), nil
	}

	// Busca em paralelo
	result := a.streamRouter.GetStreamParallel(animeTitle, episodeNumber)

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

	// Salva no cache
	a.setCache(cacheKey, smartResult, CacheTTLStream)

	return smartResult, nil
}

// GetStreamSourceStats retorna estatÃ­sticas das fontes de streaming
func (a *App) GetStreamSourceStats() map[string]interface{} {
	stats := a.streamRouter.GetAllStats()

	result := make(map[string]interface{})
	for name, s := range stats {
		avgLatency := float64(0)
		if s.SuccessCount > 0 {
			avgLatency = float64(s.TotalLatency) / float64(s.SuccessCount)
		}

		result[name] = map[string]interface{}{
			"totalRequests": s.TotalRequests,
			"successCount":  s.SuccessCount,
			"failureCount":  s.FailureCount,
			"avgLatencyMs":  avgLatency,
			"isCircuitOpen": s.IsCircuitOpen,
			"lastSuccess":   s.LastSuccess.Format(time.RFC3339),
			"lastFailure":   s.LastFailure.Format(time.RFC3339),
		}
	}

	return result
}

// ResetStreamCircuits reseta todos os circuit breakers
func (a *App) ResetStreamCircuits() {
	a.streamRouter.ResetAllCircuits()
	fmt.Println("[SmartRouter] Todos os circuits foram resetados")
}

// ============================================================================
// ANISKIP - Pular abertura/encerramento automaticamente
// ============================================================================

// SkipTimesResult contÃ©m os timestamps para pular abertura/encerramento
type SkipTimesResult struct {
	HasOpening    bool    `json:"hasOpening"`
	OpeningStart  float64 `json:"openingStart"`
	OpeningEnd    float64 `json:"openingEnd"`
	HasEnding     bool    `json:"hasEnding"`
	EndingStart   float64 `json:"endingStart"`
	EndingEnd     float64 `json:"endingEnd"`
	HasRecap      bool    `json:"hasRecap"`
	RecapStart    float64 `json:"recapStart"`
	RecapEnd      float64 `json:"recapEnd"`
	EpisodeLength float64 `json:"episodeLength"`
}

// GetSkipTimes busca os timestamps de abertura/encerramento para um episÃ³dio
// Usa a AniSkip API (requer MAL ID do anime)
func (a *App) GetSkipTimes(malID int, episodeNumber int) (*SkipTimesResult, error) {
	fmt.Printf("[AniSkip] Buscando skip times: MAL ID=%d, Ep=%d\n", malID, episodeNumber)

	// Verifica cache
	cacheKey := fmt.Sprintf("skip:%d:%d", malID, episodeNumber)
	if cached, ok := a.getCache(cacheKey); ok {
		fmt.Println("[AniSkip] Cache hit!")
		return cached.(*SkipTimesResult), nil
	}

	skipTimes, err := aniskip.GetSkipTimes(malID, episodeNumber, 0)
	if err != nil {
		return nil, err
	}

	result := &SkipTimesResult{
		HasOpening:    skipTimes.HasOpening,
		OpeningStart:  skipTimes.OpeningStart,
		OpeningEnd:    skipTimes.OpeningEnd,
		HasEnding:     skipTimes.HasEnding,
		EndingStart:   skipTimes.EndingStart,
		EndingEnd:     skipTimes.EndingEnd,
		HasRecap:      skipTimes.HasRecap,
		RecapStart:    skipTimes.RecapStart,
		RecapEnd:      skipTimes.RecapEnd,
		EpisodeLength: skipTimes.EpisodeLength,
	}

	// Salva no cache (30 minutos - skip times nÃ£o mudam)
	a.setCache(cacheKey, result, 30*time.Minute)

	return result, nil
}

// GetSkipTimesAsync busca skip times de forma assÃ­ncrona
// Retorna imediatamente e o resultado Ã© obtido depois
func (a *App) GetSkipTimesAsync(malID int, episodeNumber int) {
	go func() {
		result, err := a.GetSkipTimes(malID, episodeNumber)
		if err != nil {
			fmt.Printf("[AniSkip] Erro async: %v\n", err)
			return
		}

		// Emite evento para o frontend
		runtime.EventsEmit(a.ctx, "skipTimesReady", map[string]interface{}{
			"malID":         malID,
			"episodeNumber": episodeNumber,
			"skipTimes":     result,
		})
	}()
}

// ============================================================================
// ENIME API - Fonte de vÃ­deo rÃ¡pida
// ============================================================================

// EnimeAnime representa um anime da Enime API
type EnimeAnime struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	TitleNative string   `json:"titleNative"`
	Image       string   `json:"image"`
	Banner      string   `json:"banner"`
	AnilistID   int      `json:"anilistId"`
	MalID       int      `json:"malId"`
	Episodes    int      `json:"episodes"`
	Status      string   `json:"status"`
	Genre       []string `json:"genre"`
}

// SearchEnime busca animes na Enime API
func (a *App) SearchEnime(query string) ([]EnimeAnime, error) {
	animes, err := enime.Search(query)
	if err != nil {
		return nil, err
	}

	result := make([]EnimeAnime, 0, len(animes))
	for _, anime := range animes {
		title := anime.Title.Romaji
		if title == "" {
			title = anime.Title.English
		}
		if title == "" {
			title = anime.Title.Native
		}

		result = append(result, EnimeAnime{
			ID:          anime.ID,
			Title:       title,
			TitleNative: anime.Title.Native,
			Image:       anime.CoverImage,
			Banner:      anime.BannerImage,
			AnilistID:   anime.AnilistID,
			MalID:       anime.MalID,
			Episodes:    anime.TotalEpisodes,
			Status:      anime.Status,
			Genre:       anime.Genre,
		})
	}

	return result, nil
}

// GetEnimeStream busca stream diretamente da Enime API
func (a *App) GetEnimeStream(animeTitle string, episodeNumber int) (string, error) {
	return enime.FindAndGetStream(animeTitle, episodeNumber)
}

// normalizeAnimeName normaliza o nome do anime para comparaÃ§Ã£o
func normalizeAnimeName(name string) string {
	// Remove prefixos de fonte
	name = regexp.MustCompile(`^\[.*?\]\s*`).ReplaceAllString(name, "")
	// Remove sufixos comuns
	name = regexp.MustCompile(`\s*\((?:Dublado|Legendado|Dub|Sub|TV|OVA|Movie)\).*$`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`\s*-\s*(?:Season|Part|Temporada).*$`).ReplaceAllString(name, "")
	// Remove nÃºmeros de episÃ³dios
	name = regexp.MustCompile(`\s*\(\d+\s*episodes?\).*$`).ReplaceAllString(name, "")
	// Normaliza espaÃ§os e lowercase
	name = strings.TrimSpace(strings.ToLower(name))
	// Remove caracteres especiais
	name = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	return name
}

// BuscarAnimes - busca RÃPIDA em ambas as fontes com cache otimizado
func (a *App) BuscarAnimes(termo string) ([]store.SavedAnime, error) {
	termoLower := strings.TrimSpace(strings.ToLower(termo))
	if termoLower == "" {
		return []store.SavedAnime{}, nil
	}

	// Verifica cache TTL
	cacheKey := "search:" + termoLower
	if cached, ok := a.getCache(cacheKey); ok {
		fmt.Printf("[BuscarAnimes] Cache hit para: %s\n", termoLower)
		return cached.([]store.SavedAnime), nil
	}

	if a.client == nil {
		a.client = goanime.NewClient()
	}

	// Busca em paralelo nas duas fontes com timeout curto
	type searchResult struct {
		animes []*types.Anime
		source string
		err    error
	}

	resultChan := make(chan searchResult, 4)

	// AnimeFlix/GoFile Cache (brasileira com cache)
	go func() {
		if a.animesflixClient == nil {
			a.animesflixClient = animesflix.NewClient()
		}
		if a.animesflixClient.IsAvailable() {
			cacheResult, err := a.animesflixClient.SearchCache(termo)
			if err == nil && cacheResult != nil && len(cacheResult.Results) > 0 {
				converted := make([]*types.Anime, 0, len(cacheResult.Results))
				for _, ca := range cacheResult.Results {
					converted = append(converted, &types.Anime{
						Name:     ca.AnimeTitle,
						URL:      ca.GetURL(),
						ImageURL: "",
					})
				}
				resultChan <- searchResult{converted, "AnimeFlix", nil}
				return
			}
		}
		resultChan <- searchResult{nil, "AnimeFlix", fmt.Errorf("AnimeFlix indisponivel")}
	}()

	// GoFileCloud (biblioteca local com uploads)
	go func() {
		if a.gofileClient == nil {
			a.gofileClient = gofilecloud.NewClient()
		}
		if a.gofileClient.IsAvailable() {
			cacheResult, err := a.gofileClient.SearchCache(termo)
			if err == nil && cacheResult != nil && len(cacheResult.Results) > 0 {
				converted := make([]*types.Anime, 0, len(cacheResult.Results))
				for _, ca := range cacheResult.Results {
					animeURL := fmt.Sprintf("gofilecloud://%s/%d", ca.Slug, ca.ID)
					converted = append(converted, &types.Anime{
						Name:     ca.Name,
						URL:      animeURL,
						ImageURL: ca.Cover,
					})
				}
				resultChan <- searchResult{converted, "GoFileCloud", nil}
				return
			}
		}
		resultChan <- searchResult{nil, "GoFileCloud", fmt.Errorf("GoFileCloud indisponivel")}
	}()

	// AllAnime (inglÃªs)
	go func() {
		srcAllAnime := types.SourceAllAnime
		animes, err := a.client.SearchAnime(termo, &srcAllAnime)
		resultChan <- searchResult{animes, "AllAnime", err}
	}()

	// AnimeFire (portuguÃªs)
	go func() {
		srcAnimeFire := types.SourceAnimeFire
		animes, err := a.client.SearchAnime(termo, &srcAnimeFire)
		resultChan <- searchResult{animes, "AnimeFire", err}
	}()

	// Coleta resultados com timeout de 4 segundos
	animeMap := make(map[string]*store.SavedAnime)
	timeout := time.After(4 * time.Second)
	received := 0

	for received < 4 {
		select {
		case res := <-resultChan:
			received++
			if res.err != nil {
				fmt.Printf("[BuscarAnimes] %s erro: %v\n", res.source, res.err)
				continue
			}
			fmt.Printf("[BuscarAnimes] %s: %d resultados\n", res.source, len(res.animes))

			lang := "en"
			if res.source == "AnimeFire" {
				lang = "pt-BR"
			}

			for _, anime := range res.animes {
				if anime == nil {
					continue
				}
				normalized := normalizeAnimeName(anime.Name)
				if normalized == "" {
					continue
				}

				if existing, ok := animeMap[normalized]; ok {
					existing.Sources = append(existing.Sources, store.AnimeSource{
						Name:     res.source,
						Language: lang,
						URL:      anime.URL,
					})
					if existing.Image == "" && anime.ImageURL != "" {
						existing.Image = anime.ImageURL
					}
				} else {
					animeMap[normalized] = &store.SavedAnime{
						Title: anime.Name,
						Image: anime.ImageURL,
						URL:   anime.URL,
						Sources: []store.AnimeSource{{
							Name:     res.source,
							Language: lang,
							URL:      anime.URL,
						}},
					}
				}
			}
		case <-timeout:
			fmt.Println("[BuscarAnimes] Timeout - usando resultados parciais")
			received = 3
		}
	}

	// Converte para slice
	final := make([]store.SavedAnime, 0, len(animeMap))
	for _, anime := range animeMap {
		final = append(final, *anime)
	}

	// Ordena por nÃºmero de fontes
	sort.Slice(final, func(i, j int) bool {
		return len(final[i].Sources) > len(final[j].Sources)
	})

	// Busca imagens em paralelo para animes sem imagem (mÃ¡ximo 10 concurrent)
	var imgWg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Aumentado para 10 goroutines simultÃ¢neas

	for i := range final {
		if final[i].Image == "" {
			imgWg.Add(1)
			go func(idx int) {
				defer imgWg.Done()
				semaphore <- struct{}{}        // Adquire slot
				defer func() { <-semaphore }() // Libera slot

				poster := jikan.FetchPosterMultiSource(final[idx].Title)
				if poster != "" {
					final[idx].Image = poster
					fmt.Printf("[BuscarAnimes] Imagem encontrada para: %s\n", final[idx].Title)
				}
			}(i)
		}
	}

	// Espera busca de imagens (com timeout de 2.5s - APIs jÃ¡ tem timeout interno)
	done := make(chan struct{})
	go func() {
		imgWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Todas as imagens foram buscadas
	case <-time.After(2500 * time.Millisecond):
		fmt.Println("[BuscarAnimes] Timeout na busca de imagens")
	}

	// Salva no cache com TTL
	a.setCache(cacheKey, final, CacheTTLSearch)

	fmt.Printf("[BuscarAnimes] Total: %d animes\n", len(final))
	return final, nil
}

// GetTopAnimes - retorna cache prÃ©-carregado ou busca
func (a *App) GetTopAnimes() ([]store.SavedAnime, error) {
	a.cacheMutex.RLock()
	if len(a.topAnimesCache) > 0 {
		cached := a.topAnimesCache
		a.cacheMutex.RUnlock()
		fmt.Println("[GetTopAnimes] Retornando cache")
		return cached, nil
	}
	a.cacheMutex.RUnlock()

	animes, err := a.fetchTopAnimesInternal()
	if err != nil {
		return nil, err
	}

	a.cacheMutex.Lock()
	a.topAnimesCache = animes
	a.cacheMutex.Unlock()

	return animes, nil
}

func (a *App) fetchTopAnimesInternal() ([]store.SavedAnime, error) {
	// Lista de animes populares para buscar nas fontes reais
	popularTitles := []string{
		"Frieren", "Jujutsu Kaisen", "One Piece", "Demon Slayer",
		"My Hero Academia", "Attack on Titan", "Chainsaw Man",
		"Spy x Family", "Blue Lock", "Solo Leveling",
		"Dragon Ball", "Naruto", "Bleach", "One Punch Man",
		"Mob Psycho", "Vinland Saga", "Black Clover", "Hunter x Hunter",
		"Death Note", "Fullmetal Alchemist",
	}

	if a.client == nil {
		a.client = goanime.NewClient()
	}

	result := make([]store.SavedAnime, 0, 20)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // Limita paralelismo

	for _, title := range popularTitles {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Busca na fonte
			results, err := a.client.SearchAnime(t, nil)
			if err != nil || len(results) == 0 {
				return
			}

			anime := results[0]

			// Busca imagem
			img := jikan.FetchPosterMultiSource(anime.Name)
			if img == "" {
				img = "https://via.placeholder.com/225x318?text=" + strings.ReplaceAll(t, " ", "+")
			}

			mu.Lock()
			// Evita duplicatas
			for _, existing := range result {
				if normalizeAnimeName(existing.Title) == normalizeAnimeName(anime.Name) {
					mu.Unlock()
					return
				}
			}
			result = append(result, store.SavedAnime{
				Title:  anime.Name,
				Image:  img,
				URL:    anime.URL,
				Source: anime.Source,
			})
			mu.Unlock()
		}(title)
	}

	wg.Wait()

	// Se nÃ£o conseguiu nenhum, fallback para Jikan
	if len(result) == 0 {
		animes, err := jikan.FetchTopAnimes()
		if err != nil {
			return nil, err
		}
		for i, item := range animes {
			if i >= 20 {
				break
			}
			result = append(result, store.SavedAnime{
				Title: item.Title,
				Image: item.Image,
				URL:   "",
			})
		}
	}

	return result, nil
}

// GetAnimeURL - com cache
func (a *App) GetAnimeURL(title string) (string, error) {
	// Verifica cache
	a.cacheMutex.RLock()
	if url, ok := a.urlCache[title]; ok {
		a.cacheMutex.RUnlock()
		return url, nil
	}
	a.cacheMutex.RUnlock()

	if a.client == nil {
		a.client = goanime.NewClient()
	}

	searchResults, err := a.client.SearchAnime(title, nil)
	if err != nil || len(searchResults) == 0 || searchResults[0] == nil {
		return "", fmt.Errorf("anime nÃ£o encontrado")
	}

	url := searchResults[0].URL

	// Salva no cache
	a.cacheMutex.Lock()
	a.urlCache[title] = url
	a.cacheMutex.Unlock()

	return url, nil
}

// BuscarAnimesMulti - busca MÃšLTIPLOS termos em PARALELO (para gÃªneros)
// Retorna todos os resultados combinados sem duplicatas
func (a *App) BuscarAnimesMulti(termos []string) ([]store.SavedAnime, error) {
	if len(termos) == 0 {
		return []store.SavedAnime{}, nil
	}

	fmt.Printf("[BuscarAnimesMulti] Buscando %d termos em paralelo: %v\n", len(termos), termos)

	type searchResult struct {
		animes []store.SavedAnime
		termo  string
		err    error
	}

	resultChan := make(chan searchResult, len(termos))

	// LanÃ§a TODAS as buscas em paralelo
	for _, termo := range termos {
		go func(t string) {
			animes, err := a.BuscarAnimes(t)
			resultChan <- searchResult{animes, t, err}
		}(termo)
	}

	// Coleta resultados com timeout
	timeout := time.After(8 * time.Second)
	seenTitles := make(map[string]bool)
	allResults := make([]store.SavedAnime, 0)
	received := 0

	for received < len(termos) {
		select {
		case res := <-resultChan:
			received++
			if res.err != nil {
				fmt.Printf("[BuscarAnimesMulti] Erro em '%s': %v\n", res.termo, res.err)
				continue
			}
			fmt.Printf("[BuscarAnimesMulti] '%s': %d resultados\n", res.termo, len(res.animes))

			// Adiciona sem duplicatas
			for _, anime := range res.animes {
				key := strings.ToLower(anime.Title)
				if !seenTitles[key] {
					seenTitles[key] = true
					allResults = append(allResults, anime)
				}
			}
		case <-timeout:
			fmt.Printf("[BuscarAnimesMulti] Timeout apÃ³s %d/%d buscas\n", received, len(termos))
			received = len(termos)
		}
	}

	fmt.Printf("[BuscarAnimesMulti] Total: %d animes Ãºnicos\n", len(allResults))
	return allResults, nil
}

// GetEpisodes - otimizado com cache e busca paralela
func (a *App) GetEpisodes(seriesURL string) ([]store.Episode, error) {
	if seriesURL == "" {
		return nil, fmt.Errorf("URL invÃ¡lida")
	}

	// Verifica cache
	a.cacheMutex.RLock()
	if eps, ok := a.episodesCache[seriesURL]; ok && len(eps) > 0 {
		a.cacheMutex.RUnlock()
		fmt.Printf("[GetEpisodes] Cache hit: %d episÃ³dios\n", len(eps))
		return eps, nil
	}
	a.cacheMutex.RUnlock()

	if a.client == nil {
		a.client = goanime.NewClient()
	}

	// Busca em TODAS as fontes em PARALELO
	sources := a.client.GetAvailableSources()
	type epResult struct {
		episodes []*types.Episode
		source   types.Source
		err      error
	}

	resultChan := make(chan epResult, len(sources))

	for _, src := range sources {
		go func(s types.Source) {
			eps, err := a.client.GetAnimeEpisodes(seriesURL, s)
			resultChan <- epResult{eps, s, err}
		}(src)
	}

	// Coleta resultados com timeout
	timeout := time.After(5 * time.Second)
	var bestEpisodes []store.Episode
	received := 0

	for received < len(sources) {
		select {
		case res := <-resultChan:
			received++
			if res.err != nil || len(res.episodes) == 0 {
				continue
			}
			mapped := a.convertEpisodes(res.episodes, res.source.String())
			// Usa o resultado com mais episÃ³dios
			if len(mapped) > len(bestEpisodes) {
				bestEpisodes = mapped
				fmt.Printf("[GetEpisodes] %s: %d episÃ³dios (melhor atÃ© agora)\n", res.source, len(mapped))
			}
		case <-timeout:
			fmt.Println("[GetEpisodes] Timeout - usando melhores resultados")
			received = len(sources)
		}
	}

	if len(bestEpisodes) > 0 {
		a.cacheMutex.Lock()
		a.episodesCache[seriesURL] = bestEpisodes
		a.cacheMutex.Unlock()
		return bestEpisodes, nil
	}

	// Fallback heurÃ­stico
	return a.getEpisodesFallback(seriesURL)
}

// GetEpisodesForSource - busca episÃ³dios de uma fonte especÃ­fica
func (a *App) GetEpisodesForSource(sourceURL string, sourceName string) ([]store.Episode, error) {
	if sourceURL == "" {
		return nil, fmt.Errorf("URL invÃ¡lida")
	}

	cacheKey := fmt.Sprintf("%s:%s", sourceName, sourceURL)

	// Verifica cache
	a.cacheMutex.RLock()
	if eps, ok := a.episodesCache[cacheKey]; ok && len(eps) > 0 {
		a.cacheMutex.RUnlock()
		fmt.Printf("[GetEpisodesForSource] Cache hit: %d episÃ³dios de %s\n", len(eps), sourceName)
		return eps, nil
	}
	a.cacheMutex.RUnlock()

	if a.client == nil {
		a.client = goanime.NewClient()
	}

	// Caso especial para AnimeFlix - usa cliente proprio
	if strings.ToLower(sourceName) == "animesflix" || strings.Contains(strings.ToLower(sourceURL), "animesflix") {
		if a.animesflixClient == nil {
			a.animesflixClient = animesflix.NewClient()
		}
		details, err := a.animesflixClient.GetAnimeDetails(sourceURL)
		if err == nil && details != nil {
			var allEps []store.Episode
			for _, season := range details.Seasons {
				for _, ep := range season.Episodes {
					allEps = append(allEps, store.Episode{
						Number: func() int { n, _ := strconv.Atoi(ep.Number); return n }(),
						Title:  ep.Title,
						URL:    ep.URL,
						Source: "AnimeFlix",
					})
				}
			}
			// Salva no cache
			a.cacheMutex.Lock()
			a.episodesCache[cacheKey] = allEps
			a.cacheMutex.Unlock()
			fmt.Printf("[GetEpisodesForSource] AnimeFlix: %d episodios\n", len(allEps))
			return allEps, nil
		}
		fmt.Printf("[GetEpisodesForSource] AnimeFlix erro: %v\n", err)
	}

	// Determina a fonte correta
	var source types.Source
	switch strings.ToLower(sourceName) {
	case "allanime":
		source = types.SourceAllAnime
	case "animefire":
		source = types.SourceAnimeFire
	default:
		source = types.SourceAllAnime
	}

	fmt.Printf("[GetEpisodesForSource] Buscando de %s: %s\n", sourceName, sourceURL)

	eps, err := a.client.GetAnimeEpisodes(sourceURL, source)
	if err != nil {
		fmt.Printf("[GetEpisodesForSource] Erro: %v\n", err)
		// Tenta fallback
		return a.getEpisodesFallback(sourceURL)
	}

	mapped := a.convertEpisodes(eps, sourceName)

	// Salva no cache
	a.cacheMutex.Lock()
	a.episodesCache[cacheKey] = mapped
	a.cacheMutex.Unlock()

	fmt.Printf("[GetEpisodesForSource] Encontrados: %d episÃ³dios\n", len(mapped))
	return mapped, nil
}

func (a *App) convertEpisodes(eps []*types.Episode, source string) []store.Episode {
	mapped := make([]store.Episode, 0, len(eps))
	for _, te := range eps {
		if te == nil {
			continue
		}

		title := ""
		if te.Title != nil {
			if te.Title.English != "" {
				title = te.Title.English
			} else if te.Title.Romaji != "" {
				title = te.Title.Romaji
			} else if te.Title.Japanese != "" {
				title = te.Title.Japanese
			}
		}
		if title == "" {
			title = te.Number
		}

		num := te.Num
		if num == 0 {
			if v, err := strconv.Atoi(te.Number); err == nil {
				num = v
			}
		}

		mapped = append(mapped, store.Episode{
			Title:  title,
			URL:    te.URL,
			Season: 1,
			Number: num,
			Source: source,
		})
	}
	sort.Slice(mapped, func(i, j int) bool { return mapped[i].Number < mapped[j].Number })
	return mapped
}

func (a *App) getEpisodesFallback(seriesURL string) ([]store.Episode, error) {
	fmt.Printf("[GetEpisodes] Fallback para: %s\n", seriesURL)

	fallbackSource := "AnimeFire"
	if strings.Contains(strings.ToLower(seriesURL), "allanime") {
		fallbackSource = "AllAnime"
	}

	clientHTTP := &http.Client{Timeout: 10 * time.Second}
	resp, err := clientHTTP.Get(seriesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	// 1) procura JSON-LD
	eps := parseJSONLDScripts(seriesURL, html)
	if len(eps) > 0 {
		for i := range eps {
			eps[i].Source = fallbackSource
		}
		fmt.Printf("[GetEpisodes] Encontrados %d episÃ³dios via JSON-LD\n", len(eps))
		a.episodesCache[seriesURL] = eps
		return eps, nil
	}

	// 2) procura arrays JS
	eps = parseJSArrays(seriesURL, html)
	if len(eps) > 0 {
		for i := range eps {
			eps[i].Source = fallbackSource
		}
		fmt.Printf("[GetEpisodes] Encontrados %d episÃ³dios via JS arrays\n", len(eps))
		a.episodesCache[seriesURL] = eps
		return eps, nil
	}

	// 3) procura data-attributes
	eps = parseDataAttributes(seriesURL, html)
	if len(eps) > 0 {
		for i := range eps {
			eps[i].Source = fallbackSource
		}
		fmt.Printf("[GetEpisodes] Encontrados %d episÃ³dios via data-attributes\n", len(eps))
		a.episodesCache[seriesURL] = eps
		return eps, nil
	}

	// 4) Para AnimeFire: busca especÃ­fica por links de episÃ³dios
	if strings.Contains(seriesURL, "animefire") {
		eps := a.parseAnimeFireEpisodes(seriesURL, html)
		if len(eps) > 0 {
			for i := range eps {
				eps[i].Source = fallbackSource
			}
			fmt.Printf("[GetEpisodes] Encontrados %d episÃ³dios via AnimeFire parser\n", len(eps))
			a.episodesCache[seriesURL] = eps
			return eps, nil
		}
	}

	// 5) fallback clÃ¡ssico: procura <a> mas com filtros mais rigorosos
	re := regexp.MustCompile(`(?i)<a[^>]*href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	matches := re.FindAllStringSubmatch(html, -1)
	episodesMap := make(map[int]store.Episode)

	// Extrai o slug base do anime da URL
	baseSlug := ""
	if strings.Contains(seriesURL, "animefire") {
		// Ex: https://animefire.plus/animes/frieren-todos-os-episodios -> frieren
		parts := strings.Split(seriesURL, "/")
		for _, p := range parts {
			if strings.Contains(p, "-todos-os-episodios") {
				baseSlug = strings.Replace(p, "-todos-os-episodios", "", 1)
				break
			}
		}
	}

	for _, m := range matches {
		href := m[1]
		text := stripTags(m[2])
		href = normalizeURL(seriesURL, href)

		// Ignora URLs que nÃ£o sÃ£o do mesmo anime
		if baseSlug != "" && !strings.Contains(href, baseSlug) {
			continue
		}

		// Ignora URLs de outros sites (youtube, etc)
		if strings.Contains(href, "youtube") || strings.Contains(href, "blogger") ||
			strings.Contains(href, "google") || strings.Contains(href, "facebook") {
			continue
		}

		// Procura padrÃ£o de episÃ³dio: /anime-slug/NUMERO
		epNumRe := regexp.MustCompile(`/([^/]+)/(\d+)$`)
		epMatch := epNumRe.FindStringSubmatch(href)
		if epMatch != nil {
			num, _ := strconv.Atoi(epMatch[2])
			if num > 0 && num < 2000 {
				// Evita duplicatas - mantém pelo número
				if _, exists := episodesMap[num]; !exists {
					// Extrai nome do anime do slug
					animeSlug := epMatch[1]
					var title string

					// Formata título bonito
					if text != "" && len(text) < 100 {
						title = text
					} else {
						// Converte slug para título
						niceName := strings.ReplaceAll(animeSlug, "-", " ")
						niceName = toTitleCase(niceName)
						title = fmt.Sprintf("%s - Episodio %d", niceName, num)
					}

					episodesMap[num] = store.Episode{
						Title:  title,
						URL:    href,
						Season: 1,
						Number: num,
						Source: fallbackSource,
					}
				}
			}
		}
	}

	var final []store.Episode
	for _, e := range episodesMap {
		final = append(final, e)
	}
	sort.Slice(final, func(i, j int) bool { return final[i].Number < final[j].Number })

	fmt.Printf("[GetEpisodes] Encontrados %d possÃ­veis episÃ³dios (heurÃ­stica)\n", len(final))
	a.episodesCache[seriesURL] = final
	return final, nil
}

// parseAnimeFireEpisodes extrai episÃ³dios especificamente do AnimeFire
func (a *App) parseAnimeFireEpisodes(baseURL, html string) []store.Episode {
	var episodes []store.Episode

	// Extrai o slug base do anime (remove -todos-os-episodios)
	// Ex: https://animefire.plus/animes/jujutsu-kaisen-dublado-todos-os-episodios -> jujutsu-kaisen-dublado
	baseSlug := ""
	parts := strings.Split(baseURL, "/")
	for _, p := range parts {
		if strings.Contains(p, "-todos-os-episodios") {
			baseSlug = strings.Replace(p, "-todos-os-episodios", "", 1)
			break
		}
	}

	fmt.Printf("[parseAnimeFireEpisodes] URL: %s, baseSlug: %s\n", baseURL, baseSlug)

	if baseSlug == "" {
		// Tenta extrair o slug de outra forma - Ãºltimo segmento da URL
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			baseSlug = strings.TrimSuffix(lastPart, "-todos-os-episodios")
			fmt.Printf("[parseAnimeFireEpisodes] Fallback slug: %s\n", baseSlug)
		}
		if baseSlug == "" {
			return nil
		}
	}

	seen := make(map[int]bool)

	// MÃ©todo 1: Procura divNumEP que Ã© o container de episÃ³dios do AnimeFire
	// <div class="divNumEP"><a href="https://animefire.plus/animes/slug/1">1</a></div>
	divPattern := regexp.MustCompile(`class="divNumEP"[^>]*>.*?<a[^>]*href=["']([^"']+)["'][^>]*>`)
	divMatches := divPattern.FindAllStringSubmatch(html, -1)

	fmt.Printf("[parseAnimeFireEpisodes] divNumEP matches: %d\n", len(divMatches))

	for _, m := range divMatches {
		if len(m) >= 2 {
			url := m[1]
			// Extrai nÃºmero do episÃ³dio da URL (Ãºltimo segmento)
			urlParts := strings.Split(strings.TrimSuffix(url, "/"), "/")
			if len(urlParts) > 0 {
				numStr := urlParts[len(urlParts)-1]
				num, err := strconv.Atoi(numStr)
				if err == nil && num > 0 && num < 2000 && !seen[num] {
					seen[num] = true
					niceName := strings.ReplaceAll(baseSlug, "-", " ")
					niceName = toTitleCase(niceName)
					title := fmt.Sprintf("%s - EpisÃ³dio %d", niceName, num)
					episodes = append(episodes, store.Episode{
						Title:  title,
						URL:    url,
						Season: 1,
						Number: num,
					})
				}
			}
		}
	}

	// MÃ©todo 2: Procura padrÃ£o mais genÃ©rico de links /animes/SLUG/NUMERO
	if len(episodes) == 0 {
		// PadrÃ£o genÃ©rico para links de episÃ³dios AnimeFire
		genericPattern := regexp.MustCompile(`href=["'](https?://[^"']*animefire[^"']*/animes/[^/]+/(\d+))["']`)
		genericMatches := genericPattern.FindAllStringSubmatch(html, -1)

		fmt.Printf("[parseAnimeFireEpisodes] Generic pattern matches: %d\n", len(genericMatches))

		for _, m := range genericMatches {
			if len(m) >= 3 {
				url := m[1]
				num, _ := strconv.Atoi(m[2])
				// Verifica se o URL contÃ©m parte do slug (flexÃ­vel)
				slugWords := strings.Split(baseSlug, "-")
				matchCount := 0
				for _, word := range slugWords {
					if len(word) > 2 && strings.Contains(strings.ToLower(url), strings.ToLower(word)) {
						matchCount++
					}
				}
				// Se pelo menos metade das palavras combinam, consideramos vÃ¡lido
				if matchCount >= len(slugWords)/2 || strings.Contains(url, baseSlug) {
					if num > 0 && num < 2000 && !seen[num] {
						seen[num] = true
						niceName := strings.ReplaceAll(baseSlug, "-", " ")
						niceName = toTitleCase(niceName)
						title := fmt.Sprintf("%s - EpisÃ³dio %d", niceName, num)
						episodes = append(episodes, store.Episode{
							Title:  title,
							URL:    url,
							Season: 1,
							Number: num,
						})
					}
				}
			}
		}
	}

	// MÃ©todo 3: Se ainda vazio, procura qualquer link para animes com nÃºmero no final
	if len(episodes) == 0 {
		anyEpPattern := regexp.MustCompile(`href=["'](https?://animefire\.[^"']+/animes/[^"']+/(\d+))["']`)
		anyMatches := anyEpPattern.FindAllStringSubmatch(html, -1)

		fmt.Printf("[parseAnimeFireEpisodes] Any episode pattern matches: %d\n", len(anyMatches))

		for _, m := range anyMatches {
			if len(m) >= 3 {
				url := m[1]
				num, _ := strconv.Atoi(m[2])
				if num > 0 && num < 2000 && !seen[num] {
					seen[num] = true
					niceName := strings.ReplaceAll(baseSlug, "-", " ")
					niceName = toTitleCase(niceName)
					title := fmt.Sprintf("%s - EpisÃ³dio %d", niceName, num)
					episodes = append(episodes, store.Episode{
						Title:  title,
						URL:    url,
						Season: 1,
						Number: num,
					})
				}
			}
		}
	}

	fmt.Printf("[parseAnimeFireEpisodes] Total episÃ³dios encontrados: %d\n", len(episodes))

	sort.Slice(episodes, func(i, j int) bool { return episodes[i].Number < episodes[j].Number })
	return episodes
}

// stripTags remove tags HTML simples do texto
func stripTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}

// normalizeURL normaliza hrefs relativos para absolutos usando base
func normalizeURL(base, href string) string {
	if href == "" {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "http") {
		return href
	}
	// relative
	if strings.HasPrefix(href, "/") {
		if strings.HasPrefix(base, "http") {
			parts := strings.SplitN(base, "/", 4)
			if len(parts) >= 3 {
				return parts[0] + "//" + parts[2] + href
			}
		}
	}
	if strings.HasSuffix(base, "/") {
		return base + href
	}
	return base + "/" + href
}

// parseJSONLDScripts tenta extrair episÃ³dios de <script type="application/ld+json"> contendo Episode
func parseJSONLDScripts(baseURL, html string) []store.Episode {
	re := regexp.MustCompile(`(?is)<script[^>]+type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
	matches := re.FindAllStringSubmatch(html, -1)
	var out []store.Episode
	for _, m := range matches {
		var js interface{}
		txt := strings.TrimSpace(m[1])
		if txt == "" {
			continue
		}
		// try decode
		if err := json.Unmarshal([]byte(txt), &js); err != nil {
			continue
		}
		// walk structure to find episodes
		// support either @graph or itemListElement or direct episode array
		var find func(interface{})
		find = func(v interface{}) {
			switch vv := v.(type) {
			case map[string]interface{}:
				// if this object is an Episode
				if t, ok := vv["@type"]; ok {
					if ts, ok2 := t.(string); ok2 && (strings.EqualFold(ts, "Episode") || strings.Contains(strings.ToLower(ts), "episode")) {
						title := ""
						if nm, ok := vv["name"].(string); ok {
							title = nm
						}
						url := ""
						if u, ok := vv["url"].(string); ok {
							url = normalizeURL(baseURL, u)
						}
						num := 0
						if n, ok := vv["episodeNumber"]; ok {
							switch tn := n.(type) {
							case float64:
								num = int(tn)
							case string:
								if v2, err := strconv.Atoi(tn); err == nil {
									num = v2
								}
							}
						}
						out = append(out, store.Episode{Title: title, URL: url, Season: 1, Number: num})
					}
				}
				for _, v2 := range vv {
					find(v2)
				}
			case []interface{}:
				for _, v2 := range vv {
					find(v2)
				}
			}
		}
		find(js)
	}
	return out
}

// parseJSArrays tenta extrair arrays JS com chave episodes ou var episodes = [...] JSON-like
func parseJSArrays(baseURL, html string) []store.Episode {
	// procura por "episodes": [ ... ] ou var episodes = [...];
	re := regexp.MustCompile(`(?is)(?:"episodes"\s*:\s*|var\s+episodes\s*=\s*)(\[.*?\])`)
	m := re.FindStringSubmatch(html)
	if len(m) < 2 {
		return nil
	}
	arrText := m[1]
	// attempt to sanitize single quotes -> double quotes
	arrText = strings.ReplaceAll(arrText, "'", "\"")
	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(arrText), &items); err != nil {
		// best effort: try to extract URLs via regex inside array
		hrefRe := regexp.MustCompile(`(?i)href\s*[:=]\s*["']([^"']+)["']`)
		hm := hrefRe.FindAllStringSubmatch(arrText, -1)
		var out []store.Episode
		for _, hh := range hm {
			u := normalizeURL(baseURL, hh[1])
			out = append(out, store.Episode{Title: "", URL: u, Season: 1})
		}
		return out
	}
	var out []store.Episode
	for _, it := range items {
		title := ""
		if t, ok := it["title"].(string); ok {
			title = t
		}
		url := ""
		if u, ok := it["url"].(string); ok {
			url = normalizeURL(baseURL, u)
		}
		num := 0
		if n, ok := it["number"]; ok {
			switch tn := n.(type) {
			case float64:
				num = int(tn)
			case string:
				if v2, err := strconv.Atoi(tn); err == nil {
					num = v2
				}
			}
		}
		out = append(out, store.Episode{Title: title, URL: url, Season: 1, Number: num})
	}
	return out
}

// parseDataAttributes procura por links em elementos com data-episode/data-url
func parseDataAttributes(baseURL, html string) []store.Episode {
	re := regexp.MustCompile(`(?i)data-(?:episode|url)=["']([^"']+)["']`)
	m := re.FindAllStringSubmatch(html, -1)
	var out []store.Episode
	for _, mm := range m {
		u := normalizeURL(baseURL, mm[1])
		out = append(out, store.Episode{Title: "", URL: u, Season: 1})
	}
	return out
}

// PlayVideo recebe o link e o tÃ­tulo e manda pro MPV
// Usa a implementaÃ§Ã£o robusta de PlayAnime que procura o MPV em vÃ¡rios locais
func (a *App) PlayVideo(url string, title string) error {
	fmt.Printf("[PlayVideo] Frontend pediu play: %s (URL: %s)\n", title, url)
	return a.PlayAnime(url)
}

// PlayAnime reproduz o anime no MPV (com suporte a yt-dlp para URLs complexas)
func (a *App) PlayAnime(url string) error {
	if url == "" {
		return fmt.Errorf("URL invÃ¡lida")
	}

	fmt.Printf("Iniciando MPV com URL: %s\n", url)

	// Encontra o caminho do MPV
	mpvPath := a.findMPVPath()
	if mpvPath == "" {
		return fmt.Errorf("MPV nÃ£o encontrado. Instale o MPV ou coloque na pasta bin/")
	}

	// Argumentos base do MPV
	args := []string{
		"--force-window=immediate",
		"--hwdec=auto",
		"--vo=gpu",
	}

	// SÃ³ usa yt-dlp se a URL NÃƒO for um stream direto
	isDirectStream := strings.HasSuffix(url, ".mp4") ||
		strings.HasSuffix(url, ".m3u8") ||
		strings.HasSuffix(url, ".webm") ||
		strings.Contains(url, ".mp4?") ||
		strings.Contains(url, ".m3u8?")

	if !isDirectStream && (strings.Contains(url, "/video/") || strings.Contains(url, "/embed/")) {
		// Verifica se yt-dlp estÃ¡ disponÃ­vel
		ytdlpPath := a.findYtdlpPath()
		if ytdlpPath != "" {
			fmt.Printf("[PlayAnime] URL Ã© pÃ¡gina web, usando yt-dlp: %s\n", ytdlpPath)
			args = append(args, "--ytdl-path="+ytdlpPath)
			args = append(args, "--ytdl-format=best")
		} else {
			fmt.Println("[PlayAnime] yt-dlp nÃ£o encontrado, tentando direto...")
		}
	} else {
		fmt.Printf("[PlayAnime] URL Ã© stream direto, reproduzindo diretamente\n")
	}

	args = append(args, url)

	fmt.Printf("Executando: %s %v\n", mpvPath, args)
	cmd := exec.Command(mpvPath, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erro ao iniciar MPV: %v", err)
	}

	fmt.Printf("Sucesso! Reproduzindo no MPV: %s\n", url)
	return nil
}

// IsMPVInstalled verifica se o MPV está instalado e disponível
func (a *App) IsMPVInstalled() bool {
	path := a.findMPVPath()
	return path != ""
}

// findMPVPath procura o MPV em vários locais
func (a *App) findMPVPath() string {
	// 1) Caminho salvo pelo usuário
	if a.User != nil && a.User.MPVPath != "" {
		if _, err := os.Stat(a.User.MPVPath); err == nil {
			return a.User.MPVPath
		}
	}

	// 2) Caminho do instalador (registro do Windows)
	if path := a.getMPVPathFromRegistry(); path != "" {
		return path
	}

	// 3) mpv no PATH
	if path, err := exec.LookPath("mpv"); err == nil {
		return path
	}

	// 4) Caminhos possíveis
	possiblePaths := []string{}

	// Diretório do executável (instalador coloca aqui)
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		possiblePaths = append(possiblePaths,
			filepath.Join(exeDir, "mpv", "mpv.exe"),
			filepath.Join(exeDir, "bin", "mpv.exe"),
		)
	}

	// Diretório atual
	if dir, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths,
			filepath.Join(dir, "mpv", "mpv.exe"),
			filepath.Join(dir, "bin", "mpv.exe"),
		)
	}

	// Caminhos padrão do sistema
	username := os.Getenv("USERNAME")
	localAppData := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("PROGRAMFILES")

	possiblePaths = append(possiblePaths,
		filepath.Join(localAppData, "Programs", "GoAnime", "mpv", "mpv.exe"),
		filepath.Join(localAppData, "mpv", "mpv.exe"),
		filepath.Join(programFiles, "mpv", "mpv.exe"),
		"C:\\Program Files\\mpv\\mpv.exe",
		"C:\\Users\\"+username+"\\AppData\\Local\\mpv\\mpv.exe",
		"bin/mpv.exe",
	)

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			// Salva para próxima vez
			if a.User != nil {
				a.User.MPVPath = path
				store.SaveUser(a.User)
			}
			return path
		}
	}

	return ""
}

// getMPVPathFromRegistry obtém o caminho do MPV salvo pelo instalador
func (a *App) getMPVPathFromRegistry() string {
	// Tenta ler do registro do Windows (definido pelo instalador)
	// HKCU\Software\GoAnime\MPVPath
	cmd := exec.Command("reg", "query", "HKCU\\Software\\GoAnime", "/v", "MPVPath")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse output: "    MPVPath    REG_SZ    C:\path\to\mpv.exe"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "MPVPath") && strings.Contains(line, "REG_SZ") {
			parts := strings.Split(line, "REG_SZ")
			if len(parts) > 1 {
				path := strings.TrimSpace(parts[1])
				if _, err := os.Stat(path); err == nil {
					return path
				}
			}
		}
	}

	return ""
}

// findYtdlpPath procura o yt-dlp
func (a *App) findYtdlpPath() string {
	// No PATH
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		return path
	}

	// Caminhos comuns
	possiblePaths := []string{}

	if dir, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths,
			filepath.Join(dir, "bin", "yt-dlp.exe"),
			filepath.Join(dir, "yt-dlp.exe"),
		)
	}

	possiblePaths = append(possiblePaths,
		"C:\\Program Files\\yt-dlp\\yt-dlp.exe",
		"C:\\Users\\"+os.Getenv("USERNAME")+"\\AppData\\Local\\yt-dlp\\yt-dlp.exe",
	)

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// GetStreamURLForEpisode retorna a URL real do vÃ­deo (WebSprit, CDN) usando a biblioteca GoAnime
// Implementa cache inteligente com validaÃ§Ã£o de URL e fallback automÃ¡tico entre fontes
func (a *App) GetStreamURLForEpisode(animeURL string, episodeURL string) (string, error) {
	if a.client == nil {
		a.client = goanime.NewClient()
	}

	fmt.Printf("[GetStreamURLForEpisode] AnimeURL: %s, EpisodeURL: %s\n", animeURL, episodeURL)

	// === CACHE INTELIGENTE COM VALIDAÃ‡ÃƒO ===
	cacheKey := fmt.Sprintf("stream:%s", episodeURL)

	// Primeiro, tenta o cache inteligente de streams
	if cachedURL, ok := a.GetValidatedStreamCache(cacheKey); ok {
		fmt.Println("[GetStreamURLForEpisode] Cache inteligente hit!")
		return cachedURL, nil
	}

	// Fallback para cache antigo (migraÃ§Ã£o)
	if cached, ok := a.getCache(cacheKey); ok {
		cachedURL := cached.(string)
		fmt.Println("[GetStreamURLForEpisode] Cache legado hit, validando...")

		// Valida a URL do cache antigo
		if valid, _ := a.ValidateStreamURL(cachedURL); valid {
			// Migra para novo cache inteligente
			a.SetStreamCache(cacheKey, cachedURL, "legacy", CacheTTLStream)
			return cachedURL, nil
		}
		fmt.Println("[GetStreamURLForEpisode] URL do cache legado invÃ¡lida, buscando nova...")
	}

	// === DETECÃ‡ÃƒO DE FONTE ===
	var primarySource types.Source
	var sourceName string
	if strings.Contains(strings.ToLower(animeURL), "animefire") || strings.Contains(strings.ToLower(episodeURL), "animefire") {
		primarySource = types.SourceAnimeFire
		sourceName = "AnimeFire"
	} else {
		primarySource = types.SourceAllAnime
		sourceName = "AllAnime"
	}

	// Verifica se a fonte primÃ¡ria estÃ¡ disponÃ­vel (nÃ£o em cooldown)
	if !a.IsSourceAvailable(sourceName) {
		altSource := a.GetAlternativeSource(sourceName)
		if altSource != "" {
			fmt.Printf("[GetStreamURLForEpisode] Fonte %s em cooldown, tentando %s\n", sourceName, altSource)
			sourceName = altSource
			if altSource == "AnimeFire" {
				primarySource = types.SourceAnimeFire
			} else {
				primarySource = types.SourceAllAnime
			}
		}
	}

	// === BUSCA EPISÃ“DIO NO CACHE ===
	var cachedEpisodes []store.Episode
	var exists bool

	a.cacheMutex.RLock()
	cachedEpisodes, exists = a.episodesCache[animeURL]
	if !exists {
		for key, eps := range a.episodesCache {
			if strings.Contains(key, animeURL) || strings.HasSuffix(key, animeURL) {
				cachedEpisodes = eps
				exists = true
				break
			}
		}
	}
	a.cacheMutex.RUnlock()

	if !exists || len(cachedEpisodes) == 0 {
		fmt.Printf("[GetStreamURLForEpisode] EpisÃ³dios nÃ£o encontrados no cache para: %s\n", animeURL)
		return "", fmt.Errorf("episÃ³dios nÃ£o encontrados no cache")
	}

	var targetEpisode *store.Episode
	for i := range cachedEpisodes {
		if cachedEpisodes[i].URL == episodeURL {
			targetEpisode = &cachedEpisodes[i]
			break
		}
	}

	if targetEpisode == nil {
		fmt.Printf("[GetStreamURLForEpisode] EpisÃ³dio nÃ£o encontrado: %s\n", episodeURL)
		return "", fmt.Errorf("episÃ³dio nÃ£o encontrado")
	}

	fmt.Printf("[GetStreamURLForEpisode] EpisÃ³dio: %s (NÃºmero: %d, Source: '%s')\n",
		targetEpisode.Title, targetEpisode.Number, targetEpisode.Source)

	if targetEpisode.Source != "" {
		if parsed, err := types.ParseSource(targetEpisode.Source); err == nil {
			primarySource = parsed
		}
	}

	fmt.Printf("[GetStreamURLForEpisode] Usando source: %v\n", primarySource)

	// === BUSCA PARALELA DE TODAS AS FONTES ===
	// LanÃ§a goroutines para buscar de todas as fontes simultaneamente
	// Retorna assim que a primeira encontrar uma URL vÃ¡lida

	allSources := []struct {
		name   string
		source types.Source
	}{
		{sourceName, primarySource}, // Fonte primÃ¡ria primeiro
		{"AllAnime", types.SourceAllAnime},
		{"AnimeFire", types.SourceAnimeFire},
	}

	// Remove duplicatas (se primarySource jÃ¡ estÃ¡ na lista)
	uniqueSources := make([]struct {
		name   string
		source types.Source
	}, 0, len(allSources))
	seen := make(map[string]bool)
	for _, s := range allSources {
		if !seen[s.name] {
			seen[s.name] = true
			uniqueSources = append(uniqueSources, s)
		}
	}

	type streamResult struct {
		url    string
		source string
		err    error
	}

	resultChan := make(chan streamResult, len(uniqueSources)+1) // +1 para Smart Router

	// LanÃ§a goroutines para cada fonte
	var wg sync.WaitGroup
	for _, src := range uniqueSources {
		if !a.IsSourceAvailable(src.name) {
			fmt.Printf("[GetStreamURLForEpisode] Fonte %s em cooldown, pulando\n", src.name)
			continue
		}

		wg.Add(1)
		go func(srcName string, srcType types.Source) {
			defer wg.Done()

			fmt.Printf("[GetStreamURLForEpisode] [Parallel] Buscando de %s...\n", srcName)
			url, err := a.tryGetStreamFromSource(targetEpisode, animeURL, episodeURL, srcType)

			if err == nil && url != "" {
				// Valida a URL rapidamente (timeout curto)
				if valid, _ := a.ValidateStreamURL(url); valid {
					resultChan <- streamResult{url: url, source: srcName, err: nil}
					return
				}
				resultChan <- streamResult{url: "", source: srcName, err: fmt.Errorf("URL invÃ¡lida")}
				return
			}
			resultChan <- streamResult{url: "", source: srcName, err: err}
		}(src.name, src.source)
	}

	// TambÃ©m tenta o Smart Router em paralelo
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("[GetStreamURLForEpisode] [Parallel] Buscando via Smart Router...")
		result := a.streamRouter.GetStream(targetEpisode.Title, targetEpisode.Number)
		if result != nil && result.URL != "" && result.Error == nil {
			if valid, _ := a.ValidateStreamURL(result.URL); valid {
				resultChan <- streamResult{url: result.URL, source: "SmartRouter:" + result.Source, err: nil}
				return
			}
		}
		resultChan <- streamResult{url: "", source: "SmartRouter", err: fmt.Errorf("sem resultado")}
	}()

	// Fecha o canal quando todas as goroutines terminarem
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Espera pelo primeiro resultado vÃ¡lido ou timeout
	timeout := time.After(10 * time.Second)
	var lastError error

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				// Canal fechado, todas as fontes falharam
				return "", fmt.Errorf("nÃ£o foi possÃ­vel obter stream URL de nenhuma fonte: %v", lastError)
			}

			if result.url != "" {
				// Encontrou! Cacheia e retorna imediatamente
				a.SetStreamCache(cacheKey, result.url, result.source, CacheTTLStream)
				a.recordSourceSuccess(result.source)
				fmt.Printf("[GetStreamURLForEpisode] âœ“ Sucesso com %s (paralelo)!\n", result.source)

				// Inicia prefetch dos prÃ³ximos episÃ³dios em background
				go a.prefetchNextEpisodes(animeURL, cachedEpisodes, targetEpisode.Number)

				return result.url, nil
			} else if result.err != nil {
				lastError = result.err
				a.recordSourceFailure(result.source, result.err.Error())
			}

		case <-timeout:
			return "", fmt.Errorf("timeout: nenhuma fonte respondeu a tempo")
		}
	}
}

// prefetchNextEpisodes prÃ©-carrega URLs dos prÃ³ximos episÃ³dios em background
func (a *App) prefetchNextEpisodes(animeURL string, episodes []store.Episode, currentEpNum int) {
	// PrÃ©-carrega os prÃ³ximos 2 episÃ³dios
	for _, ep := range episodes {
		if ep.Number > currentEpNum && ep.Number <= currentEpNum+2 {
			cacheKey := fmt.Sprintf("stream:%s", ep.URL)

			// Verifica se jÃ¡ estÃ¡ no cache
			if _, ok := a.GetValidatedStreamCache(cacheKey); ok {
				continue
			}

			// Verifica se jÃ¡ estÃ¡ em prefetch
			a.prefetchMutex.RLock()
			inProgress := a.prefetchActive[cacheKey]
			a.prefetchMutex.RUnlock()
			if inProgress {
				continue
			}

			// Marca como em progresso
			a.prefetchMutex.Lock()
			a.prefetchActive[cacheKey] = true
			a.prefetchMutex.Unlock()

			fmt.Printf("[Prefetch] PrÃ©-carregando episÃ³dio %d...\n", ep.Number)

			// Busca em background (sem esperar)
			go func(episode store.Episode) {
				defer func() {
					a.prefetchMutex.Lock()
					delete(a.prefetchActive, fmt.Sprintf("stream:%s", episode.URL))
					a.prefetchMutex.Unlock()
				}()

				// Usa apenas a fonte primÃ¡ria para prefetch (mais rÃ¡pido)
				var source types.Source
				if strings.Contains(episode.URL, "animefire") {
					source = types.SourceAnimeFire
				} else {
					source = types.SourceAllAnime
				}

				url, err := a.tryGetStreamFromSource(&episode, animeURL, episode.URL, source)
				if err == nil && url != "" {
					if valid, _ := a.ValidateStreamURL(url); valid {
						key := fmt.Sprintf("stream:%s", episode.URL)
						a.SetStreamCache(key, url, source.String(), CacheTTLStream)
						fmt.Printf("[Prefetch] âœ“ EpisÃ³dio %d prÃ©-carregado!\n", episode.Number)
					}
				}
			}(ep)
		}
	}
}

// tryGetStreamFromSource tenta obter stream de uma fonte especÃ­fica
func (a *App) tryGetStreamFromSource(episode *store.Episode, animeURL, episodeURL string, source types.Source) (string, error) {
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

	// Tenta usar o mÃ©todo da biblioteca primeiro
	streamURL, metadata, err := a.client.GetEpisodeStreamURL(anime, ep, &goanime.StreamOptions{
		Quality: "best",
		Mode:    "sub",
	})

	if err == nil && streamURL != "" {
		fmt.Printf("[tryGetStreamFromSource] Stream URL obtida de %s: %s\n", source, streamURL)
		if metadata != nil {
			fmt.Printf("[tryGetStreamFromSource] Metadata: %v\n", metadata)
		}
		return streamURL, nil
	}

	// FALLBACK: Para AnimeFire, usa extrator HTML customizado
	if source == types.SourceAnimeFire {
		fmt.Printf("[tryGetStreamFromSource] Usando extrator HTML para AnimeFire...\n")
		streamURL, err = videoextractor.ExtractVideoURL(episodeURL)
		if err != nil {
			return "", fmt.Errorf("falha na extraÃ§Ã£o HTML: %w", err)
		}
		if streamURL != "" {
			fmt.Printf("[tryGetStreamFromSource] Stream extraÃ­do do HTML: %s\n", streamURL)
			return streamURL, nil
		}
	}

	return "", fmt.Errorf("fonte %s falhou: %v", source, err)
}

// AssistirEpisodio Ã© a funÃ§Ã£o MÃGICA que o Svelte vai chamar ao clicar no episÃ³dio
func (a *App) AssistirEpisodio(animeURL string, episodeURL string, episodeTitle string) error {
	fmt.Printf("[AssistirEpisodio] Iniciando processo para: %s\n", episodeTitle)

	// 1. Extrai o link real do vÃ­deo (MP4/M3U8) usando sua funÃ§Ã£o existente
	streamURL, err := a.GetStreamURLForEpisode(animeURL, episodeURL)
	if err != nil {
		fmt.Printf("[AssistirEpisodio] Falha ao extrair link: %v\n", err)
		return fmt.Errorf("nÃ£o foi possÃ­vel extrair o vÃ­deo: %v", err)
	}

	if streamURL == "" {
		return fmt.Errorf("link de vÃ­deo retornado vazio")
	}

	fmt.Printf("[AssistirEpisodio] Link extraÃ­do com sucesso: %s\n", streamURL)

	// 2. Manda o MPV tocar o link REAL do vÃ­deo
	return a.PlayVideo(streamURL, episodeTitle)
}

// ============================================
// DISCORD INTEGRATION
// ============================================

// DiscordRecommendation representa uma recomendaÃ§Ã£o para o frontend
type DiscordRecommendation struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	UserAvatar string  `json:"userAvatar"`
	AnimeTitle string  `json:"animeTitle"`
	AnimeImage string  `json:"animeImage"`
	AnimeScore float64 `json:"animeScore"`
	Message    string  `json:"message"`
	Timestamp  int64   `json:"timestamp"` // Unix timestamp em ms
	Likes      int     `json:"likes"`
	LikedByMe  bool    `json:"likedByMe"`
}

// DiscordStatus retorna o status da conexÃ£o Discord
type DiscordStatus struct {
	Connected   bool   `json:"connected"`
	WebhookURL  string `json:"webhookUrl"`
	Username    string `json:"username"`
	ServerName  string `json:"serverName"`
	ChannelName string `json:"channelName"`
}

// GetDiscordStatus retorna o status atual do Discord
func (a *App) GetDiscordStatus() DiscordStatus {
	bot := discord.GetBot()
	return DiscordStatus{
		Connected:   bot.IsConnected(),
		ServerName:  "GoAnime Community",
		ChannelName: "#recomendaÃ§Ãµes",
	}
}

// ConnectDiscord configura a conexÃ£o com Discord via webhook
func (a *App) ConnectDiscord(webhookURL string) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook URL Ã© obrigatÃ³rio")
	}

	bot := discord.GetBot()
	bot.Configure(webhookURL)

	// Carrega recomendaÃ§Ãµes de exemplo para demonstraÃ§Ã£o
	if len(bot.GetRecommendations()) == 0 {
		bot.AddMockRecommendations()
	}

	fmt.Println("[Discord] Conectado com sucesso!")
	return nil
}

// DisconnectDiscord desconecta do Discord
func (a *App) DisconnectDiscord() {
	bot := discord.GetBot()
	bot.Configure("")
	fmt.Println("[Discord] Desconectado")
}

// GetDiscordRecommendations retorna as recomendaÃ§Ãµes do Discord
func (a *App) GetDiscordRecommendations() []DiscordRecommendation {
	bot := discord.GetBot()
	recs := bot.GetRecommendations()

	result := make([]DiscordRecommendation, len(recs))
	for i, rec := range recs {
		result[i] = DiscordRecommendation{
			ID:         rec.ID,
			Username:   rec.Username,
			UserAvatar: rec.UserAvatar,
			AnimeTitle: rec.AnimeTitle,
			AnimeImage: rec.AnimeImage,
			AnimeScore: rec.AnimeScore,
			Message:    rec.Message,
			Timestamp:  rec.Timestamp.UnixMilli(),
			Likes:      rec.Likes,
			LikedByMe:  rec.LikedByMe,
		}
	}

	return result
}

// SendDiscordRecommendation envia uma recomendaÃ§Ã£o para o Discord
func (a *App) SendDiscordRecommendation(animeTitle, animeImage string, animeScore float64, message string) error {
	bot := discord.GetBot()

	if !bot.IsConnected() {
		return fmt.Errorf("discord não está conectado")
	}

	username := "UsuÃ¡rio"
	if a.User != nil && a.User.Username != "" {
		username = a.User.Username
	}

	rec := discord.Recommendation{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		UserID:     "local",
		Username:   username,
		UserAvatar: "https://cdn.discordapp.com/embed/avatars/0.png",
		AnimeTitle: animeTitle,
		AnimeImage: animeImage,
		AnimeScore: animeScore,
		Message:    message,
		Timestamp:  time.Now(),
	}

	err := bot.SendRecommendation(rec)
	if err != nil {
		fmt.Printf("[Discord] Erro ao enviar recomendaÃ§Ã£o: %v\n", err)
		return err
	}

	fmt.Printf("[Discord] RecomendaÃ§Ã£o enviada: %s\n", animeTitle)
	return nil
}

// LikeDiscordRecommendation adiciona um like a uma recomendaÃ§Ã£o
func (a *App) LikeDiscordRecommendation(recID string) bool {
	bot := discord.GetBot()
	return bot.LikeRecommendation(recID)
}

// SimulateDiscordConnect simula conexÃ£o para demonstraÃ§Ã£o (sem webhook real)
func (a *App) SimulateDiscordConnect() error {
	bot := discord.GetBot()

	// Configura como conectado (modo demo)
	bot.Configure("demo")

	// Carrega recomendaÃ§Ãµes de exemplo
	bot.AddMockRecommendations()

	fmt.Println("[Discord] Modo demonstraÃ§Ã£o ativado com recomendaÃ§Ãµes de exemplo")
	return nil
}

// DiscordOAuthConfig contÃ©m as configuraÃ§Ãµes do OAuth2
type DiscordOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// ConfiguraÃ§Ãµes do aplicativo Discord OAuth2
// Para o app funcionar, vocÃª precisa criar um aplicativo em https://discord.com/developers/applications
// e preencher as credenciais abaixo ou criar um arquivo discord_config.json
var discordOAuth = DiscordOAuthConfig{
	ClientID:     "", // Preencha com seu Client ID ou use discord_config.json
	ClientSecret: "", // Preencha com seu Client Secret ou use discord_config.json
	RedirectURI:  "http://localhost:9876/callback",
}

// InitDiscordOAuth inicializa as credenciais do Discord OAuth
func initDiscordOAuth() {
	// Tenta carregar do arquivo de configuraÃ§Ã£o
	configPath := "discord_config.json"

	// Verifica se o arquivo existe
	if data, err := os.ReadFile(configPath); err == nil {
		var config struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}
		if json.Unmarshal(data, &config) == nil {
			if config.ClientID != "" {
				discordOAuth.ClientID = config.ClientID
			}
			if config.ClientSecret != "" {
				discordOAuth.ClientSecret = config.ClientSecret
			}
			fmt.Println("[Discord] Credenciais carregadas do arquivo discord_config.json")
		}
	}

	// Fallback para variÃ¡veis de ambiente
	if discordOAuth.ClientID == "" {
		if envID := os.Getenv("DISCORD_CLIENT_ID"); envID != "" {
			discordOAuth.ClientID = envID
		}
	}
	if discordOAuth.ClientSecret == "" {
		if envSecret := os.Getenv("DISCORD_CLIENT_SECRET"); envSecret != "" {
			discordOAuth.ClientSecret = envSecret
		}
	}

	// Log do status
	if discordOAuth.ClientID != "" && discordOAuth.ClientSecret != "" {
		fmt.Printf("[Discord] OAuth configurado: Client ID = %s...%s\n",
			discordOAuth.ClientID[:8],
			discordOAuth.ClientID[len(discordOAuth.ClientID)-4:])
	} else {
		fmt.Println("[Discord] âš ï¸ Credenciais nÃ£o configuradas! Crie um arquivo discord_config.json")
	}
}

// SaveDiscordConfig salva as credenciais do Discord
func (a *App) SaveDiscordConfig(clientID, clientSecret string) error {
	config := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile("discord_config.json", data, 0600); err != nil {
		return err
	}

	// Atualiza as credenciais em memÃ³ria
	discordOAuth.ClientID = clientID
	discordOAuth.ClientSecret = clientSecret

	fmt.Println("[Discord] Credenciais salvas com sucesso!")
	return nil
}

// GetDiscordConfigStatus retorna o status da configuraÃ§Ã£o do Discord
func (a *App) GetDiscordConfigStatus() map[string]interface{} {
	return map[string]interface{}{
		"configured":  discordOAuth.ClientID != "" && discordOAuth.ClientSecret != "",
		"hasClientId": discordOAuth.ClientID != "",
		"hasSecret":   discordOAuth.ClientSecret != "",
		"redirectUri": discordOAuth.RedirectURI,
	}
}

// DiscordUserInfo representa as informaÃ§Ãµes do usuÃ¡rio para o frontend
type DiscordUserInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatarUrl"`
	Connected bool   `json:"connected"`
}

// GetDiscordOAuthURL retorna a URL para iniciar o fluxo OAuth2
func (a *App) GetDiscordOAuthURL() string {
	// Scopes necessÃ¡rios: identify para obter informaÃ§Ãµes do usuÃ¡rio
	scopes := "identify"

	authURL := fmt.Sprintf(
		"https://discord.com/api/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s",
		discordOAuth.ClientID,
		url.QueryEscape(discordOAuth.RedirectURI),
		url.QueryEscape(scopes),
	)

	return authURL
}

// StartDiscordOAuth inicia o fluxo OAuth2 abrindo o navegador
func (a *App) StartDiscordOAuth() error {
	authURL := a.GetDiscordOAuthURL()

	// Abre o navegador com a URL de autorizaÃ§Ã£o
	runtime.BrowserOpenURL(a.ctx, authURL)

	fmt.Println("[Discord OAuth] Aguardando autorizaÃ§Ã£o do usuÃ¡rio...")

	// Inicia o servidor de callback em uma goroutine
	go func() {
		bot := discord.GetBot()

		// Aguarda o callback
		code, err := bot.OAuth2CallbackServer(discordOAuth.ClientID, discordOAuth.ClientSecret, discordOAuth.RedirectURI)
		if err != nil {
			fmt.Printf("[Discord OAuth] Erro: %v\n", err)
			return
		}

		// Para funcionar sem client_secret, usamos o token implÃ­cito
		// Ou buscamos o usuÃ¡rio diretamente com o cÃ³digo
		user, err := a.CompleteDiscordOAuthWithCode(code)
		if err != nil {
			fmt.Printf("[Discord OAuth] Erro ao completar OAuth: %v\n", err)
			return
		}

		fmt.Printf("[Discord OAuth] UsuÃ¡rio conectado: %s\n", user.Username)

		// Emite evento para o frontend
		runtime.EventsEmit(a.ctx, "discord:connected", user)
	}()

	return nil
}

// CompleteDiscordOAuthWithCode completa o fluxo OAuth2 usando o cÃ³digo
func (a *App) CompleteDiscordOAuthWithCode(code string) (*DiscordUserInfo, error) {
	bot := discord.GetBot()

	// Se temos client_secret, troca o cÃ³digo por token
	if discordOAuth.ClientSecret != "" {
		accessToken, err := bot.ExchangeCodeForToken(
			discordOAuth.ClientID,
			discordOAuth.ClientSecret,
			code,
			discordOAuth.RedirectURI,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao trocar cÃ³digo por token: %w", err)
		}

		user, err := bot.FetchUserFromToken(accessToken)
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar usuÃ¡rio: %w", err)
		}

		bot.SetCurrentUser(user)

		// Carrega recomendaÃ§Ãµes de exemplo
		if len(bot.GetRecommendations()) == 0 {
			bot.AddMockRecommendations()
		}

		return &DiscordUserInfo{
			ID:        user.ID,
			Username:  user.GlobalName,
			AvatarURL: user.AvatarURL,
			Connected: true,
		}, nil
	}

	// Modo sem client_secret: usamos o usuÃ¡rio simulado baseado no cÃ³digo
	// Para produÃ§Ã£o, vocÃª deve ter o client_secret configurado
	fmt.Println("[Discord OAuth] Modo sem client_secret - usando perfil simulado")

	mockUser := &discord.DiscordUser{
		ID:         "oauth_user",
		Username:   "DiscordUser",
		GlobalName: "Discord User",
		AvatarURL:  "https://cdn.discordapp.com/embed/avatars/0.png",
	}

	bot.SetCurrentUser(mockUser)

	// Carrega recomendaÃ§Ãµes de exemplo
	if len(bot.GetRecommendations()) == 0 {
		bot.AddMockRecommendations()
	}

	return &DiscordUserInfo{
		ID:        mockUser.ID,
		Username:  mockUser.GlobalName,
		AvatarURL: mockUser.AvatarURL,
		Connected: true,
	}, nil
}

// GetDiscordUser retorna o usuÃ¡rio Discord atualmente conectado
func (a *App) GetDiscordUser() *DiscordUserInfo {
	bot := discord.GetBot()
	user := bot.GetCurrentUser()

	if user == nil {
		return &DiscordUserInfo{Connected: false}
	}

	displayName := user.GlobalName
	if displayName == "" {
		displayName = user.Username
	}

	return &DiscordUserInfo{
		ID:        user.ID,
		Username:  displayName,
		AvatarURL: user.AvatarURL,
		Connected: true,
	}
}

// DisconnectDiscordUser desconecta o usuÃ¡rio Discord OAuth
func (a *App) DisconnectDiscordUser() {
	bot := discord.GetBot()
	bot.Disconnect()
	runtime.EventsEmit(a.ctx, "discord:disconnected", nil)
	fmt.Println("[Discord] UsuÃ¡rio desconectado")
}

// SetDiscordClientSecret configura o client secret do OAuth2
func (a *App) SetDiscordClientSecret(secret string) {
	discordOAuth.ClientSecret = secret
}

// ============================================
// DISCORD LINKING SYSTEM (VinculaÃ§Ã£o por CÃ³digo)
// ============================================

// DiscordLinkInfo representa informaÃ§Ãµes da conta vinculada
type DiscordLinkInfo struct {
	IsLinked    bool   `json:"isLinked"`
	UserID      string `json:"userId"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	LinkedAt    string `json:"linkedAt"`
	ShowStatus  bool   `json:"showStatus"`
	ShareAnimes bool   `json:"shareAnimes"`
}

// DiscordFriendActivity representa atividade de um amigo
type DiscordFriendActivity struct {
	UserID     string `json:"userId"`
	Username   string `json:"username"`
	Avatar     string `json:"avatar"`
	AnimeTitle string `json:"animeTitle"`
	EpisodeNum int    `json:"episodeNum"`
	AnimeImage string `json:"animeImage"`
	IsOnline   bool   `json:"isOnline"`
}

// GetDiscordLinkStatus retorna o status da vinculaÃ§Ã£o
func (a *App) GetDiscordLinkStatus() DiscordLinkInfo {
	ls := discord.GetLinkingSystem()
	account := ls.GetLinkedAccount()

	if account == nil {
		return DiscordLinkInfo{IsLinked: false}
	}

	return DiscordLinkInfo{
		IsLinked:    true,
		UserID:      account.UserID,
		Username:    account.Username,
		Avatar:      account.Avatar,
		LinkedAt:    account.LinkedAt.Format("02/01/2006"),
		ShowStatus:  account.ShowStatus,
		ShareAnimes: account.ShareAnimes,
	}
}

// LinkDiscordWithCode vincula a conta Discord usando um cÃ³digo
func (a *App) LinkDiscordWithCode(code string) (DiscordLinkInfo, error) {
	ls := discord.GetLinkingSystem()

	account, err := ls.LinkWithCode(code)
	if err != nil {
		return DiscordLinkInfo{IsLinked: false}, err
	}

	// Emite evento de conexÃ£o
	runtime.EventsEmit(a.ctx, "discord:linked", map[string]string{
		"username": account.Username,
		"userId":   account.UserID,
	})

	return DiscordLinkInfo{
		IsLinked:    true,
		UserID:      account.UserID,
		Username:    account.Username,
		Avatar:      account.Avatar,
		LinkedAt:    account.LinkedAt.Format("02/01/2006"),
		ShowStatus:  account.ShowStatus,
		ShareAnimes: account.ShareAnimes,
	}, nil
}

// UnlinkDiscord desvincula a conta Discord
func (a *App) UnlinkDiscord() error {
	ls := discord.GetLinkingSystem()

	if err := ls.Unlink(); err != nil {
		return err
	}

	runtime.EventsEmit(a.ctx, "discord:unlinked", nil)
	return nil
}

// GetDiscordServerInvite retorna o link do servidor Discord
func (a *App) GetDiscordServerInvite() string {
	return discord.GetServerInviteLink()
}

// GenerateDiscordLinkCode gera um cÃ³digo de vinculaÃ§Ã£o (para debug/teste)
func (a *App) GenerateDiscordLinkCode() string {
	return discord.GenerateLinkCode()
}

// UpdateDiscordWatchingStatus atualiza o status de "assistindo"
func (a *App) UpdateDiscordWatchingStatus(animeTitle string, episodeNum int, animeImage string, totalEpisodes int) {
	ls := discord.GetLinkingSystem()

	status := discord.WatchingStatus{
		AnimeTitle:    animeTitle,
		EpisodeNum:    episodeNum,
		AnimeImage:    animeImage,
		TotalEpisodes: totalEpisodes,
	}

	if err := ls.UpdateWatchingStatus(status); err != nil {
		fmt.Printf("[Discord] Erro ao atualizar status: %v\n", err)
	}
}

// GetDiscordFriendsActivity busca atividade dos amigos
func (a *App) GetDiscordFriendsActivity() []DiscordFriendActivity {
	ls := discord.GetLinkingSystem()

	activities, err := ls.GetFriendsActivity()
	if err != nil {
		fmt.Printf("[Discord] Erro ao buscar amigos: %v\n", err)
		return []DiscordFriendActivity{}
	}

	result := make([]DiscordFriendActivity, len(activities))
	for i, act := range activities {
		result[i] = DiscordFriendActivity{
			UserID:     act.UserID,
			Username:   act.Username,
			Avatar:     act.Avatar,
			AnimeTitle: act.AnimeTitle,
			EpisodeNum: act.EpisodeNum,
			AnimeImage: act.AnimeImage,
			IsOnline:   act.IsOnline,
		}
	}

	return result
}

// SetDiscordShowStatus habilita/desabilita compartilhamento de status
func (a *App) SetDiscordShowStatus(enabled bool) error {
	ls := discord.GetLinkingSystem()
	return ls.SetShowStatus(enabled)
}

// SetDiscordShareAnimes habilita/desabilita compartilhamento de animes
func (a *App) SetDiscordShareAnimes(enabled bool) error {
	ls := discord.GetLinkingSystem()
	return ls.SetShareAnimes(enabled)
}

// ==================== MANGA METHODS ====================

// MangaInfo representa um mangÃ¡ para o frontend
type MangaInfo struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Image       string   `json:"image"`
	URL         string   `json:"url"`
	LatestChap  string   `json:"latestChapter"`
	Genres      []string `json:"genres"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Source      string   `json:"source"` // Fonte do mangÃ¡ (mangalivre.to, mangalivre.blog)
}

// MangaChapterInfo representa um capÃ­tulo para o frontend
type MangaChapterInfo struct {
	Number    string `json:"number"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Date      string `json:"date"`
	MangaID   string `json:"mangaId"`
	MangaName string `json:"mangaName"`
}

// MangaPageInfo representa uma pÃ¡gina de mangÃ¡ para o frontend
type MangaPageInfo struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// MangaSourceInfo representa informações sobre uma fonte de mangá
type MangaSourceInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// AnimeSourceInfo representa informações sobre uma fonte de anime
type AnimeSourceInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"` // "pt" ou "en"
	Priority    int    `json:"priority"`
}

// getSourceFromMangaURL determina a fonte a partir da URL
func getSourceFromMangaURL(mangaURL string) string {
	if strings.Contains(mangaURL, "mangalivre.blog") {
		return "mangalivre.blog"
	}
	return "mangalivre.to"
}

// convertMangaToInfo converte manga.Manga para MangaInfo incluindo a fonte
func convertMangaToInfo(m manga.Manga, source string) MangaInfo {
	if source == "" {
		source = getSourceFromMangaURL(m.URL)
	}
	return MangaInfo{
		ID:          m.ID,
		Title:       m.Title,
		Image:       m.Image,
		URL:         m.URL,
		LatestChap:  m.LatestChap,
		Genres:      m.Genres,
		Description: m.Description,
		Status:      m.Status,
		Source:      source,
	}
}

// GetPopularMangas retorna os mangÃ¡s populares (fonte padrÃ£o: mangalivre.to)
func (a *App) GetPopularMangas() []MangaInfo {
	return a.GetPopularMangasFromSource("mangalivre.to")
}

// GetPopularMangasFromSource retorna os mangÃ¡s populares de uma fonte especÃ­fica
func (a *App) GetPopularMangasFromSource(sourceName string) []MangaInfo {
	fmt.Printf("[GetPopularMangasFromSource] Buscando mangÃ¡s populares da fonte %s...\n", sourceName)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	source, ok := a.mangaAggregator.GetSource(sourceName)
	if !ok {
		fmt.Printf("[GetPopularMangasFromSource] Fonte nÃ£o encontrada: %s\n", sourceName)
		return []MangaInfo{}
	}

	mangas, err := source.GetPopularMangas()
	if err != nil {
		fmt.Printf("[GetPopularMangasFromSource] Erro: %v\n", err)
		return []MangaInfo{}
	}

	result := make([]MangaInfo, len(mangas))
	for i, m := range mangas {
		result[i] = convertMangaToInfo(m, sourceName)
	}

	fmt.Printf("[GetPopularMangasFromSource] Retornando %d mangÃ¡s da fonte %s\n", len(result), sourceName)
	return result
}

// GetPopularMangasAllSources retorna os mangÃ¡s populares de TODAS as fontes combinadas
func (a *App) GetPopularMangasAllSources() []MangaInfo {
	fmt.Println("[GetPopularMangasAllSources] Buscando mangÃ¡s populares de todas as fontes...")

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	sources := a.mangaAggregator.GetSources()
	var allMangas []MangaInfo

	for _, sourceName := range sources {
		mangas := a.GetPopularMangasFromSource(sourceName)
		allMangas = append(allMangas, mangas...)
	}

	fmt.Printf("[GetPopularMangasAllSources] Retornando %d mangÃ¡s de todas as fontes\n", len(allMangas))
	return allMangas
}

// GetLatestMangas retorna os mangÃ¡s com atualizaÃ§Ãµes recentes (fonte padrÃ£o: mangalivre.to)
func (a *App) GetLatestMangas() []MangaInfo {
	return a.GetLatestMangasFromSource("mangalivre.to")
}

// GetLatestMangasFromSource retorna os mangÃ¡s com atualizaÃ§Ãµes recentes de uma fonte especÃ­fica
func (a *App) GetLatestMangasFromSource(sourceName string) []MangaInfo {
	fmt.Printf("[GetLatestMangasFromSource] Buscando Ãºltimas atualizaÃ§Ãµes da fonte %s...\n", sourceName)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	source, ok := a.mangaAggregator.GetSource(sourceName)
	if !ok {
		fmt.Printf("[GetLatestMangasFromSource] Fonte nÃ£o encontrada: %s\n", sourceName)
		return []MangaInfo{}
	}

	mangas, err := source.GetLatestUpdates()
	if err != nil {
		fmt.Printf("[GetLatestMangasFromSource] Erro: %v\n", err)
		return []MangaInfo{}
	}

	result := make([]MangaInfo, len(mangas))
	for i, m := range mangas {
		result[i] = convertMangaToInfo(m, sourceName)
	}

	fmt.Printf("[GetLatestMangasFromSource] Retornando %d mangÃ¡s da fonte %s\n", len(result), sourceName)
	return result
}

// SearchMangas busca mangÃ¡s por termo (fonte padrÃ£o: mangalivre.to)
func (a *App) SearchMangas(query string) []MangaInfo {
	return a.SearchMangasFromSource(query, "mangalivre.to")
}

// SearchMangasFromSource busca mangÃ¡s por termo em uma fonte especÃ­fica
func (a *App) SearchMangasFromSource(query string, sourceName string) []MangaInfo {
	fmt.Printf("[SearchMangasFromSource] Buscando '%s' na fonte %s...\n", query, sourceName)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	source, ok := a.mangaAggregator.GetSource(sourceName)
	if !ok {
		fmt.Printf("[SearchMangasFromSource] Fonte nÃ£o encontrada: %s\n", sourceName)
		return []MangaInfo{}
	}

	mangas, err := source.SearchManga(query)
	if err != nil {
		fmt.Printf("[SearchMangasFromSource] Erro: %v\n", err)
		return []MangaInfo{}
	}

	result := make([]MangaInfo, len(mangas))
	for i, m := range mangas {
		result[i] = convertMangaToInfo(m, sourceName)
	}

	fmt.Printf("[SearchMangasFromSource] Retornando %d mangÃ¡s da fonte %s\n", len(result), sourceName)
	return result
}

// GetMangaDetails obtÃ©m detalhes completos de um mangÃ¡ com cache
func (a *App) GetMangaDetails(mangaURL string) *MangaInfo {
	fmt.Printf("[GetMangaDetails] Obtendo detalhes: %s\n", mangaURL)

	// Usa o agregador para detectar a fonte correta
	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	sourceName := getSourceFromMangaURL(mangaURL)
	mangaDetails, err := a.mangaAggregator.GetMangaDetails(mangaURL)
	if err != nil {
		fmt.Printf("[GetMangaDetails] Erro: %v\n", err)
		return nil
	}

	info := convertMangaToInfo(*mangaDetails, sourceName)
	return &info
}

// GetMangaChapters obtÃ©m a lista de capÃ­tulos de um mangÃ¡ (detecta fonte pela URL)
func (a *App) GetMangaChapters(mangaURL string) []MangaChapterInfo {
	fmt.Printf("[GetMangaChapters] Obtendo capÃ­tulos: %s\n", mangaURL)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	chapters, err := a.mangaAggregator.GetChapters(mangaURL)
	if err != nil {
		fmt.Printf("[GetMangaChapters] Erro: %v\n", err)
		return []MangaChapterInfo{}
	}

	result := make([]MangaChapterInfo, len(chapters))
	for i, ch := range chapters {
		result[i] = MangaChapterInfo{
			Number:    ch.Number,
			Title:     ch.Title,
			URL:       ch.URL,
			Date:      ch.Date,
			MangaID:   ch.MangaID,
			MangaName: ch.MangaName,
		}
	}

	fmt.Printf("[GetMangaChapters] Retornando %d capÃ­tulos\n", len(result))
	return result
}

// GetChapterPages obtÃ©m as pÃ¡ginas (imagens) de um capÃ­tulo (detecta fonte pela URL)
func (a *App) GetChapterPages(chapterURL string) []MangaPageInfo {
	fmt.Printf("[GetChapterPages] Obtendo pÃ¡ginas: %s\n", chapterURL)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	// Inicia proxy se nÃ£o estiver rodando
	if a.proxyPort == 0 {
		a.startVideoProxy()
	}

	// Usa o agregador que detecta a fonte correta
	pages, err := a.mangaAggregator.GetChapterPages(chapterURL)
	if err != nil {
		fmt.Printf("[GetChapterPages] Erro: %v\n", err)
		return []MangaPageInfo{}
	}

	// Extrai referer do chapterURL
	referer := "https://mangalivre.to/"
	if parsed, err := url.Parse(chapterURL); err == nil {
		referer = fmt.Sprintf("%s://%s/", parsed.Scheme, parsed.Host)
	}

	result := make([]MangaPageInfo, len(pages))
	for i, p := range pages {
		// Usa proxy local para cache e evitar CORS/hotlink protection
		proxyURL := p.URL
		if a.proxyPort > 0 {
			proxyURL = fmt.Sprintf("http://127.0.0.1:%d/manga-image?url=%s&referer=%s",
				a.proxyPort,
				url.QueryEscape(p.URL),
				url.QueryEscape(referer))
		}
		result[i] = MangaPageInfo{
			Number: p.Number,
			URL:    proxyURL,
		}
	}

	// PrÃ©-carrega imagens em background para cache
	go func() {
		for _, p := range pages {
			a.preloadMangaImage(p.URL, referer)
		}
	}()

	fmt.Printf("[GetChapterPages] Retornando %d pÃ¡ginas via proxy\n", len(result))
	return result
}

// preloadMangaImage prÃ©-carrega uma imagem de mangÃ¡ no cache
func (a *App) preloadMangaImage(imageURL, referer string) {
	a.imageCacheMutex.RLock()
	_, exists := a.imageCache[imageURL]
	a.imageCacheMutex.RUnlock()

	if exists {
		return // JÃ¡ estÃ¡ no cache
	}

	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("Referer", referer)

	resp, err := a.imageClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		data, err := io.ReadAll(resp.Body)
		if err == nil && len(data) > 0 {
			a.imageCacheMutex.Lock()
			a.imageCache[imageURL] = data
			a.imageCacheMutex.Unlock()
		}
	}
}

// GetMangasByGenre retorna mangÃ¡s de um gÃªnero especÃ­fico
func (a *App) GetMangasByGenre(genre string) []MangaInfo {
	fmt.Printf("[GetMangasByGenre] Buscando gÃªnero: %s\n", genre)

	if a.mangaClient == nil {
		a.mangaClient = manga.NewMangaClient()
	}

	mangas, err := a.mangaClient.GetMangasByGenre(genre)
	if err != nil {
		fmt.Printf("[GetMangasByGenre] Erro: %v\n", err)
		return []MangaInfo{}
	}

	result := make([]MangaInfo, len(mangas))
	for i, m := range mangas {
		result[i] = MangaInfo{
			ID:         m.ID,
			Title:      m.Title,
			Image:      m.Image,
			URL:        m.URL,
			LatestChap: m.LatestChap,
			Genres:     m.Genres,
		}
	}

	fmt.Printf("[GetMangasByGenre] Retornando %d mangÃ¡s\n", len(result))
	return result
}

// MangaListResult representa o resultado de listagem de mangÃ¡s com paginaÃ§Ã£o
type MangaListResult struct {
	Mangas     []MangaInfo `json:"mangas"`
	TotalPages int         `json:"totalPages"`
	Page       int         `json:"page"`
}

// GetAllMangas retorna todos os mangÃ¡s com paginaÃ§Ã£o
func (a *App) GetAllMangas(page int) MangaListResult {
	fmt.Printf("[GetAllMangas] Buscando pÃ¡gina %d...\n", page)

	if a.mangaClient == nil {
		a.mangaClient = manga.NewMangaClient()
	}

	mangas, totalPages, err := a.mangaClient.GetAllMangas(page)
	if err != nil {
		fmt.Printf("[GetAllMangas] Erro: %v\n", err)
		return MangaListResult{Mangas: []MangaInfo{}, TotalPages: 0, Page: page}
	}

	result := make([]MangaInfo, len(mangas))
	for i, m := range mangas {
		result[i] = MangaInfo{
			ID:         m.ID,
			Title:      m.Title,
			Image:      m.Image,
			URL:        m.URL,
			LatestChap: m.LatestChap,
			Genres:     m.Genres,
		}
	}

	fmt.Printf("[GetAllMangas] PÃ¡gina %d: Retornando %d mangÃ¡s (total pÃ¡ginas: %d)\n", page, len(result), totalPages)
	return MangaListResult{Mangas: result, TotalPages: totalPages, Page: page}
}

// GetAllMangasComplete busca TODOS os mangÃ¡s de TODAS as fontes
func (a *App) GetAllMangasComplete() []MangaInfo {
	fmt.Println("[GetAllMangasComplete] Buscando TODOS os mangÃ¡s de todas as fontes (paralelo)...")

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	sources := a.mangaAggregator.GetSources()
	var allMangas []MangaInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, sourceName := range sources {
		source, ok := a.mangaAggregator.GetSource(sourceName)
		if !ok {
			continue
		}

		wg.Add(1)
		go func(sourceName string, source manga.MangaSource) {
			defer wg.Done()
			localMangas := []MangaInfo{}
			var pageWg sync.WaitGroup
			pageChan := make(chan []MangaInfo, 10)

			// Busca atÃ© 10 pÃ¡ginas em paralelo
			for page := 1; page <= 10; page++ {
				pageWg.Add(1)
				go func(page int) {
					defer pageWg.Done()
					mangas, totalPages, err := source.GetAllMangas(page)
					if err != nil {
						fmt.Printf("[GetAllMangasComplete] Erro na fonte %s pÃ¡gina %d: %v\n", sourceName, page, err)
						return
					}
					temp := []MangaInfo{}
					for _, m := range mangas {
						temp = append(temp, convertMangaToInfo(m, sourceName))
					}
					pageChan <- temp
					if page >= totalPages {
						// NÃ£o busca mais pÃ¡ginas se chegou ao fim
						return
					}
				}(page)
			}

			pageWg.Wait()
			close(pageChan)
			for ms := range pageChan {
				localMangas = append(localMangas, ms...)
			}
			mu.Lock()
			allMangas = append(allMangas, localMangas...)
			mu.Unlock()
		}(sourceName, source)
	}

	wg.Wait()
	fmt.Printf("[GetAllMangasComplete] TOTAL FINAL: %d mangÃ¡s de todas as fontes\n", len(allMangas))
	return allMangas
}

// GetAllMangasFromSourceComplete busca TODOS os mangÃ¡s de UMA fonte especÃ­fica
func (a *App) GetAllMangasFromSourceComplete(sourceName string) []MangaInfo {
	fmt.Printf("[GetAllMangasFromSourceComplete] Buscando todos os mangÃ¡s da fonte %s...\n", sourceName)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	source, ok := a.mangaAggregator.GetSource(sourceName)
	if !ok {
		fmt.Printf("[GetAllMangasFromSourceComplete] Fonte nÃ£o encontrada: %s\n", sourceName)
		return []MangaInfo{}
	}

	var allMangas []MangaInfo

	// Busca todas as pÃ¡ginas
	for page := 1; page <= 50; page++ { // Limite de 50 pÃ¡ginas
		mangas, totalPages, err := source.GetAllMangas(page)
		if err != nil {
			fmt.Printf("[GetAllMangasFromSourceComplete] Erro na pÃ¡gina %d: %v\n", page, err)
			break
		}

		for _, m := range mangas {
			allMangas = append(allMangas, convertMangaToInfo(m, sourceName))
		}

		if page >= totalPages {
			break
		}
	}

	fmt.Printf("[GetAllMangasFromSourceComplete] Total: %d mangÃ¡s da fonte %s\n", len(allMangas), sourceName)
	return allMangas
}

// normalizeMangaTitle normaliza o tÃ­tulo para comparaÃ§Ã£o
func normalizeMangaTitle(title string) string {
	// Remove caracteres especiais e converte para minÃºsculas
	title = strings.ToLower(title)
	title = strings.TrimSpace(title)

	// Remove caracteres especiais comuns
	replacer := strings.NewReplacer(
		":", "", "-", "", "â€“", "", "â€”", "",
		"!", "", "?", "", ".", "", ",", "",
		"'", "", "'", "", "\"", "",
		"(", "", ")", "", "[", "", "]", "",
	)
	title = replacer.Replace(title)

	// Remove espaÃ§os extras
	title = strings.Join(strings.Fields(title), " ")

	return title
}

// extractChapterCount extrai o nÃºmero de capÃ­tulos do latestChapter
func extractChapterCount(latestChap string) int {
	if latestChap == "" {
		return 0
	}

	// Tenta extrair nÃºmero do formato "Cap. 123" ou "CapÃ­tulo 123" etc
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(latestChap, -1)

	if len(matches) > 0 {
		// Pega o maior nÃºmero encontrado
		maxNum := 0
		for _, m := range matches {
			if num, err := strconv.Atoi(m); err == nil && num > maxNum {
				maxNum = num
			}
		}
		return maxNum
	}
	return 0
}

// GetMergedMangas retorna mangÃ¡s de todas as fontes, mesclando duplicatas e escolhendo a com mais capÃ­tulos
func (a *App) GetMergedMangas(limit int) []MangaInfo {
	fmt.Printf("[GetMergedMangas] Buscando e mesclando mangÃ¡s de todas as fontes (limite: %d)...\n", limit)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	sources := a.mangaAggregator.GetSources()

	// Mapa para armazenar mangÃ¡s por tÃ­tulo normalizado
	mangaMap := make(map[string]MangaInfo)

	for _, sourceName := range sources {
		source, ok := a.mangaAggregator.GetSource(sourceName)
		if !ok {
			continue
		}

		// Busca populares e Ãºltimos de cada fonte
		populares, _ := source.GetPopularMangas()
		ultimos, _ := source.GetLatestUpdates()

		// Combina ambas as listas
		allFromSource := append(populares, ultimos...)

		for _, m := range allFromSource {
			info := convertMangaToInfo(m, sourceName)
			normalizedTitle := normalizeMangaTitle(info.Title)

			// Verifica se jÃ¡ existe
			if existing, exists := mangaMap[normalizedTitle]; exists {
				// Compara quantidade de capÃ­tulos
				existingChapters := extractChapterCount(existing.LatestChap)
				newChapters := extractChapterCount(info.LatestChap)

				// Escolhe o que tem mais capÃ­tulos
				if newChapters > existingChapters {
					mangaMap[normalizedTitle] = info
					fmt.Printf("[GetMergedMangas] '%s': %s (%d caps) > %s (%d caps)\n",
						info.Title, sourceName, newChapters, existing.Source, existingChapters)
				}
			} else {
				mangaMap[normalizedTitle] = info
			}
		}
	}

	// Converte mapa para slice
	var result []MangaInfo
	for _, m := range mangaMap {
		// Filtra conteÃºdo adulto
		if !isAdultManga(m.Genres) {
			result = append(result, m)
		}
	}

	// Ordena por nÃºmero de capÃ­tulos (mais capÃ­tulos primeiro)
	sort.Slice(result, func(i, j int) bool {
		return extractChapterCount(result[i].LatestChap) > extractChapterCount(result[j].LatestChap)
	})

	// Aplica limite
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	fmt.Printf("[GetMergedMangas] Retornando %d mangÃ¡s mesclados\n", len(result))
	return result
}

// GetMergedMangasComplete retorna TODOS os mangÃ¡s mesclados de todas as fontes
func (a *App) GetMergedMangasComplete() []MangaInfo {
	fmt.Println("[GetMergedMangasComplete] Buscando e mesclando TODOS os mangÃ¡s...")

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	sources := a.mangaAggregator.GetSources()

	// Mapa para armazenar mangÃ¡s por tÃ­tulo normalizado
	mangaMap := make(map[string]MangaInfo)

	for _, sourceName := range sources {
		source, ok := a.mangaAggregator.GetSource(sourceName)
		if !ok {
			continue
		}

		// Busca todas as pÃ¡ginas (limite de 20 por fonte para performance)
		for page := 1; page <= 20; page++ {
			mangas, totalPages, err := source.GetAllMangas(page)
			if err != nil {
				break
			}

			for _, m := range mangas {
				info := convertMangaToInfo(m, sourceName)
				normalizedTitle := normalizeMangaTitle(info.Title)

				if existing, exists := mangaMap[normalizedTitle]; exists {
					existingChapters := extractChapterCount(existing.LatestChap)
					newChapters := extractChapterCount(info.LatestChap)

					if newChapters > existingChapters {
						mangaMap[normalizedTitle] = info
					}
				} else {
					mangaMap[normalizedTitle] = info
				}
			}

			if page >= totalPages {
				break
			}
		}
	}

	// Converte e filtra
	var result []MangaInfo
	for _, m := range mangaMap {
		result = append(result, m)
	}

	// Ordena por capÃ­tulos
	sort.Slice(result, func(i, j int) bool {
		return extractChapterCount(result[i].LatestChap) > extractChapterCount(result[j].LatestChap)
	})

	fmt.Printf("[GetMergedMangasComplete] Retornando %d mangÃ¡s mesclados\n", len(result))
	return result
}

// adultGenres sÃ£o os gÃªneros considerados conteÃºdo adulto (+18) - APENAS HENTAI
var adultGenres = []string{
	"hentai", "+18", "r18", "r-18", "sexo explicito", "sexo explÃ­cito",
}

// isAdultManga verifica se um mangÃ¡ contÃ©m gÃªneros adultos (hentai)
func isAdultManga(genres []string) bool {
	for _, g := range genres {
		genreLower := strings.ToLower(strings.TrimSpace(g))
		for _, adult := range adultGenres {
			if strings.Contains(genreLower, adult) {
				return true
			}
		}
	}
	return false
}

// GetPopularMangasSafe retorna mangÃ¡s populares SEM conteÃºdo adulto
func (a *App) GetPopularMangasSafe() []MangaInfo {
	fmt.Println("[GetPopularMangasSafe] Buscando mangÃ¡s populares (SFW)...")

	allMangas := a.GetPopularMangas()
	var safeMangas []MangaInfo

	for _, m := range allMangas {
		if !isAdultManga(m.Genres) {
			safeMangas = append(safeMangas, m)
		}
	}

	fmt.Printf("[GetPopularMangasSafe] Retornando %d mangÃ¡s seguros de %d total\n", len(safeMangas), len(allMangas))
	return safeMangas
}

// GetPopularMangasAdult retorna APENAS mangÃ¡s populares com conteÃºdo adulto
func (a *App) GetPopularMangasAdult() []MangaInfo {
	fmt.Println("[GetPopularMangasAdult] Buscando mangÃ¡s adultos (+18)...")

	allMangas := a.GetPopularMangas()
	var adultMangas []MangaInfo

	for _, m := range allMangas {
		if isAdultManga(m.Genres) {
			adultMangas = append(adultMangas, m)
		}
	}

	fmt.Printf("[GetPopularMangasAdult] Retornando %d mangÃ¡s adultos de %d total\n", len(adultMangas), len(allMangas))
	return adultMangas
}

// GetAllMangasSafe retorna TODOS os mangÃ¡s SEM conteÃºdo adulto
func (a *App) GetAllMangasSafe() []MangaInfo {
	fmt.Println("[GetAllMangasSafe] Buscando todos os mangÃ¡s (SFW)...")

	allMangas := a.GetAllMangasComplete()
	var safeMangas []MangaInfo

	for _, m := range allMangas {
		if !isAdultManga(m.Genres) {
			safeMangas = append(safeMangas, m)
		}
	}

	fmt.Printf("[GetAllMangasSafe] Retornando %d mangÃ¡s seguros de %d total\n", len(safeMangas), len(allMangas))
	return safeMangas
}

// GetAllMangasAdult retorna APENAS mangÃ¡s com conteÃºdo adulto
func (a *App) GetAllMangasAdult() []MangaInfo {
	fmt.Println("[GetAllMangasAdult] Buscando mangÃ¡s adultos (+18)...")

	allMangas := a.GetAllMangasComplete()
	var adultMangas []MangaInfo

	for _, m := range allMangas {
		if isAdultManga(m.Genres) {
			adultMangas = append(adultMangas, m)
		}
	}

	fmt.Printf("[GetAllMangasAdult] Retornando %d mangÃ¡s adultos de %d total\n", len(adultMangas), len(allMangas))
	return adultMangas
}

// GetFeaturedMangas retorna mangÃ¡s em destaque de TODAS as fontes (populares + Ãºltimas atualizaÃ§Ãµes, sem +18)
func (a *App) GetFeaturedMangas(limit int) []MangaInfo {
	return a.GetFeaturedMangasFromSource(limit, "")
}

// GetFeaturedMangasFromSource retorna mangÃ¡s em destaque de uma fonte especÃ­fica
// Se sourceName for vazio, busca de todas as fontes
func (a *App) GetFeaturedMangasFromSource(limit int, sourceName string) []MangaInfo {
	fmt.Printf("[GetFeaturedMangasFromSource] Buscando %d mangÃ¡s em destaque (fonte: %s)...\n", limit, sourceName)

	if limit <= 0 {
		limit = 24 // PadrÃ£o: 24 mangÃ¡s
	}

	var populares []MangaInfo

	if sourceName == "" || sourceName == "all" {
		// Busca de todas as fontes
		populares = a.GetPopularMangasAllSources()
	} else {
		// Busca de uma fonte especÃ­fica
		populares = a.GetPopularMangasFromSource(sourceName)
	}

	// Filtra conteÃºdo adulto
	var safePopulares []MangaInfo
	for _, m := range populares {
		if !isAdultManga(m.Genres) {
			safePopulares = append(safePopulares, m)
		}
	}

	// Se tiver mangÃ¡s suficientes, retorna
	if len(safePopulares) >= limit {
		return safePopulares[:limit]
	}

	// SenÃ£o, completa com os Ãºltimos updates
	var latests []MangaInfo
	if sourceName == "" || sourceName == "all" {
		// Busca Ãºltimos de todas as fontes
		if a.mangaAggregator == nil {
			a.mangaAggregator = manga.NewMangaAggregator()
		}
		sources := a.mangaAggregator.GetSources()
		for _, src := range sources {
			srcLatests := a.GetLatestMangasFromSource(src)
			latests = append(latests, srcLatests...)
		}
	} else {
		latests = a.GetLatestMangasFromSource(sourceName)
	}

	seen := make(map[string]bool)

	// Marca os populares como vistos
	for _, m := range safePopulares {
		seen[m.URL] = true
	}

	// Adiciona os Ãºltimos que nÃ£o sÃ£o duplicados e nÃ£o sÃ£o adultos
	for _, m := range latests {
		if !seen[m.URL] && !isAdultManga(m.Genres) {
			safePopulares = append(safePopulares, m)
			if len(safePopulares) >= limit {
				break
			}
		}
	}

	fmt.Printf("[GetFeaturedMangasFromSource] Retornando %d mangÃ¡s em destaque\n", len(safePopulares))
	return safePopulares
}

// GetMangaGenres retorna a lista de gÃªneros disponÃ­veis
func (a *App) GetMangaGenres() []string {
	fmt.Println("[GetMangaGenres] Buscando gÃªneros...")

	if a.mangaClient == nil {
		a.mangaClient = manga.NewMangaClient()
	}

	genres, err := a.mangaClient.GetGenres()
	if err != nil {
		fmt.Printf("[GetMangaGenres] Erro: %v\n", err)
		return []string{}
	}

	fmt.Printf("[GetMangaGenres] Retornando %d gÃªneros\n", len(genres))
	return genres
}

// ============== FUNÃ‡Ã•ES DE MÃšLTIPLAS FONTES DE MANGÃ ==============

// GetMangaSourcesInfo retorna informaÃ§Ãµes detalhadas sobre as fontes disponÃ­veis
func (a *App) GetMangaSourcesInfo() []MangaSourceInfo {
	sources := a.GetMangaSources()
	result := make([]MangaSourceInfo, len(sources))

	for i, s := range sources {
		switch s {
		case "mangalivre.to":
			result[i] = MangaSourceInfo{
				ID:          s,
				Name:        "MangaLivre.to",
				Description: "Fonte principal com grande acervo de mangÃ¡s em portuguÃªs",
				URL:         "https://mangalivre.to",
			}
		case "mangalivre.blog":
			result[i] = MangaSourceInfo{
				ID:          s,
				Name:        "MangaLivre.blog",
				Description: "Fonte alternativa com mangÃ¡s atualizados frequentemente",
				URL:         "https://mangalivre.blog",
			}
		default:
			result[i] = MangaSourceInfo{
				ID:          s,
				Name:        s,
				Description: "Fonte de mangÃ¡s",
				URL:         "",
			}
		}
	}

	return result
}

// GetMangaSources retorna a lista de fontes disponíveis
func (a *App) GetMangaSources() []string {
	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}
	return a.mangaAggregator.GetSources()
}

// ============== FUNÇÕES DE MÚLTIPLAS FONTES DE ANIME ==============

// GetAnimeSourcesInfo retorna informações detalhadas sobre as fontes de anime disponíveis
func (a *App) GetAnimeSourcesInfo() []AnimeSourceInfo {
	sources := []AnimeSourceInfo{
		{
			ID:          "enime",
			Name:        "Enime",
			Description: "Fonte rápida com animes legendados em inglês",
			Language:    "en",
			Priority:    1,
		},
		{
			ID:          "consumet",
			Name:        "Consumet",
			Description: "Fonte confiável com grande acervo",
			Language:    "en",
			Priority:    2,
		},
		{
			ID:          "torbox",
			Name:        "TorBox",
			Description: "Streaming via torrents em alta qualidade (requer API key)",
			Language:    "multi",
			Priority:    3,
		},
	}
	return sources
}

// GetAnimeSources retorna a lista de IDs das fontes de anime disponíveis
func (a *App) GetAnimeSources() []string {
	return []string{"enime", "consumet", "torbox"}
}

// GetMangasFromSource busca mangÃ¡s de uma fonte especÃ­fica
func (a *App) GetMangasFromSource(sourceName string, page int) MangaListResult {
	fmt.Printf("[GetMangasFromSource] Buscando pÃ¡gina %d da fonte %s...\n", page, sourceName)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	mangas, totalPages, err := a.mangaAggregator.GetAllMangasFromSource(sourceName, page)
	if err != nil {
		fmt.Printf("[GetMangasFromSource] Erro: %v\n", err)
		return MangaListResult{Mangas: []MangaInfo{}, TotalPages: 0, Page: page}
	}

	result := make([]MangaInfo, len(mangas))
	for i, m := range mangas {
		result[i] = MangaInfo{
			ID:         m.ID,
			Title:      m.Title,
			Image:      m.Image,
			URL:        m.URL,
			LatestChap: m.LatestChap,
			Genres:     m.Genres,
		}
	}

	fmt.Printf("[GetMangasFromSource] Fonte %s, PÃ¡gina %d: %d mangÃ¡s\n", sourceName, page, len(result))
	return MangaListResult{Mangas: result, TotalPages: totalPages, Page: page}
}

// GetMangasFromAllSources busca mangÃ¡s de todas as fontes combinadas
func (a *App) GetMangasFromAllSources(page int) MangaListResult {
	fmt.Printf("[GetMangasFromAllSources] Buscando pÃ¡gina %d de todas as fontes...\n", page)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	mangas, totalPages, err := a.mangaAggregator.GetAllMangasFromAllSources(page)
	if err != nil {
		fmt.Printf("[GetMangasFromAllSources] Erro: %v\n", err)
		return MangaListResult{Mangas: []MangaInfo{}, TotalPages: 0, Page: page}
	}

	result := make([]MangaInfo, len(mangas))
	for i, m := range mangas {
		result[i] = MangaInfo{
			ID:         m.ID,
			Title:      m.Title,
			Image:      m.Image,
			URL:        m.URL,
			LatestChap: m.LatestChap,
			Genres:     m.Genres,
		}
	}

	fmt.Printf("[GetMangasFromAllSources] PÃ¡gina %d: %d mangÃ¡s de todas as fontes\n", page, len(result))
	return MangaListResult{Mangas: result, TotalPages: totalPages, Page: page}
}

// SearchMangasAllSources busca mangÃ¡s em todas as fontes
func (a *App) SearchMangasAllSources(query string) []MangaInfo {
	fmt.Printf("[SearchMangasAllSources] Buscando '%s' em todas as fontes...\n", query)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	mangas, err := a.mangaAggregator.SearchAllSources(query)
	if err != nil {
		fmt.Printf("[SearchMangasAllSources] Erro: %v\n", err)
		return []MangaInfo{}
	}

	result := make([]MangaInfo, len(mangas))
	for i, m := range mangas {
		result[i] = MangaInfo{
			ID:         m.ID,
			Title:      m.Title,
			Image:      m.Image,
			URL:        m.URL,
			LatestChap: m.LatestChap,
			Genres:     m.Genres,
		}
	}

	fmt.Printf("[SearchMangasAllSources] Encontrados %d mangÃ¡s\n", len(result))
	return result
}

// GetMangaDetailsAuto obtÃ©m detalhes de um mangÃ¡ (detecta fonte pela URL automaticamente)
func (a *App) GetMangaDetailsAuto(mangaURL string) *MangaInfo {
	fmt.Printf("[GetMangaDetailsAuto] Obtendo detalhes: %s\n", mangaURL)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	mangaDetails, err := a.mangaAggregator.GetMangaDetails(mangaURL)
	if err != nil {
		fmt.Printf("[GetMangaDetailsAuto] Erro: %v\n", err)
		return nil
	}

	return &MangaInfo{
		ID:          mangaDetails.ID,
		Title:       mangaDetails.Title,
		Image:       mangaDetails.Image,
		URL:         mangaDetails.URL,
		Genres:      mangaDetails.Genres,
		Description: mangaDetails.Description,
		Status:      mangaDetails.Status,
	}
}

// GetMangaChaptersAuto obtÃ©m capÃ­tulos de um mangÃ¡ (detecta fonte pela URL automaticamente)
func (a *App) GetMangaChaptersAuto(mangaURL string) []MangaChapterInfo {
	fmt.Printf("[GetMangaChaptersAuto] Obtendo capÃ­tulos: %s\n", mangaURL)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	chapters, err := a.mangaAggregator.GetChapters(mangaURL)
	if err != nil {
		fmt.Printf("[GetMangaChaptersAuto] Erro: %v\n", err)
		return []MangaChapterInfo{}
	}

	result := make([]MangaChapterInfo, len(chapters))
	for i, ch := range chapters {
		result[i] = MangaChapterInfo{
			Number:    ch.Number,
			Title:     ch.Title,
			URL:       ch.URL,
			Date:      ch.Date,
			MangaID:   ch.MangaID,
			MangaName: ch.MangaName,
		}
	}

	fmt.Printf("[GetMangaChaptersAuto] Retornando %d capÃ­tulos\n", len(result))
	return result
}

// GetChapterPagesAuto obtÃ©m pÃ¡ginas de um capÃ­tulo (detecta fonte pela URL automaticamente)
func (a *App) GetChapterPagesAuto(chapterURL string) []MangaPageInfo {
	fmt.Printf("[GetChapterPagesAuto] Obtendo pÃ¡ginas: %s\n", chapterURL)

	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	// Inicia proxy se nÃ£o estiver rodando
	if a.proxyPort == 0 {
		a.startVideoProxy()
	}

	pages, err := a.mangaAggregator.GetChapterPages(chapterURL)
	if err != nil {
		fmt.Printf("[GetChapterPagesAuto] Erro: %v\n", err)
		return []MangaPageInfo{}
	}

	// Extrai referer do chapterURL
	referer := "https://mangalivre.to/"
	if parsed, err := url.Parse(chapterURL); err == nil {
		referer = fmt.Sprintf("%s://%s/", parsed.Scheme, parsed.Host)
	}

	result := make([]MangaPageInfo, len(pages))
	for i, p := range pages {
		// Usa proxy local para cache e evitar CORS/hotlink protection
		proxyURL := p.URL
		if a.proxyPort > 0 {
			proxyURL = fmt.Sprintf("http://127.0.0.1:%d/manga-image?url=%s&referer=%s",
				a.proxyPort,
				url.QueryEscape(p.URL),
				url.QueryEscape(referer))
		}
		result[i] = MangaPageInfo{
			Number: p.Number,
			URL:    proxyURL,
		}
	}

	// PrÃ©-carrega imagens em background para cache
	go func() {
		for _, p := range pages {
			a.preloadMangaImage(p.URL, referer)
		}
	}()

	fmt.Printf("[GetChapterPagesAuto] Retornando %d pÃ¡ginas via proxy\n", len(result))
	return result
}

// GetMergedMangasWithBestSource busca mangÃ¡s de todas as fontes e mescla inteligentemente
// escolhendo a versÃ£o que tem mais capÃ­tulos quando hÃ¡ duplicatas
func (a *App) GetMergedMangasWithBestSource() []MangaInfo {
	fmt.Println("[GetMergedMangasWithBestSource] Iniciando merge inteligente de todas as fontes...")

	// Inicializa o aggregator se necessÃ¡rio
	if a.mangaAggregator == nil {
		a.mangaAggregator = manga.NewMangaAggregator()
	}

	// FunÃ§Ã£o para normalizar tÃ­tulos para comparaÃ§Ã£o
	normalizeTitleForComparison := func(title string) string {
		// Remove caracteres especiais e converte para minÃºsculas
		title = strings.ToLower(title)
		reg := regexp.MustCompile(`[^a-z0-9\s]`)
		title = reg.ReplaceAllString(title, "")
		title = strings.TrimSpace(title)
		// Remove espaÃ§os extras
		reg = regexp.MustCompile(`\s+`)
		title = reg.ReplaceAllString(title, " ")
		return title
	}

	// FunÃ§Ã£o para extrair nÃºmero de capÃ­tulos do campo LatestChap
	extractChapterCount := func(latestChap string) int {
		if latestChap == "" {
			return 0
		}
		// Tenta extrair nÃºmero do formato "Cap. XXX" ou "CapÃ­tulo XXX"
		reg := regexp.MustCompile(`(\d+)`)
		matches := reg.FindStringSubmatch(latestChap)
		if len(matches) > 1 {
			num, err := strconv.Atoi(matches[1])
			if err == nil {
				return num
			}
		}
		return 0
	}

	// Mapa para armazenar o melhor mangÃ¡ por tÃ­tulo normalizado
	bestMangaByTitle := make(map[string]MangaInfo)

	// Busca de cada fonte
	sourceNames := a.mangaAggregator.GetSources()
	for _, sourceName := range sourceNames {
		fmt.Printf("[GetMergedMangasWithBestSource] Buscando da fonte: %s\n", sourceName)

		source, ok := a.mangaAggregator.GetSource(sourceName)
		if !ok {
			fmt.Printf("[GetMergedMangasWithBestSource] Fonte nÃ£o encontrada: %s\n", sourceName)
			continue
		}

		// Busca todos os mangÃ¡s (pÃ¡gina por pÃ¡gina)
		page := 1
		for {
			mangas, totalPages, err := source.GetAllMangas(page)
			if err != nil {
				fmt.Printf("[GetMergedMangasWithBestSource] Erro na fonte %s pÃ¡gina %d: %v\n", sourceName, page, err)
				break
			}

			for _, m := range mangas {
				info := convertMangaToInfo(m, sourceName)
				normalizedTitle := normalizeTitleForComparison(info.Title)

				if existing, exists := bestMangaByTitle[normalizedTitle]; exists {
					// Compara nÃºmero de capÃ­tulos
					existingChapters := extractChapterCount(existing.LatestChap)
					newChapters := extractChapterCount(info.LatestChap)

					if newChapters > existingChapters {
						fmt.Printf("[GetMergedMangasWithBestSource] Substituindo '%s': %s(%d caps) -> %s(%d caps)\n",
							info.Title, existing.Source, existingChapters, sourceName, newChapters)
						bestMangaByTitle[normalizedTitle] = info
					}
				} else {
					bestMangaByTitle[normalizedTitle] = info
				}
			}

			if page >= totalPages {
				break
			}
			page++
		}
	}

	// Converte mapa para slice
	result := make([]MangaInfo, 0, len(bestMangaByTitle))
	for _, manga := range bestMangaByTitle {
		result = append(result, manga)
	}

	// Ordena por tÃ­tulo
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Title) < strings.ToLower(result[j].Title)
	})

	fmt.Printf("[GetMergedMangasWithBestSource] Total apÃ³s merge: %d mangÃ¡s Ãºnicos\n", len(result))
	return result
}
