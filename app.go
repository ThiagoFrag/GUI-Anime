package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

	"GoAnimeGUI/pkg/anilist"
	"GoAnimeGUI/pkg/aniskip"
	"GoAnimeGUI/pkg/consumet"
	"GoAnimeGUI/pkg/discord"
	"GoAnimeGUI/pkg/enime"
	"GoAnimeGUI/pkg/jikan"
	"GoAnimeGUI/pkg/smartrouter"
	"GoAnimeGUI/pkg/store"
	"GoAnimeGUI/pkg/videoextractor"

	"github.com/alvarorichard/Goanime/pkg/goanime"
	"github.com/alvarorichard/Goanime/pkg/goanime/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// CacheEntry representa uma entrada de cache com TTL e validação
type CacheEntry struct {
	Data        interface{}
	ExpiresAt   time.Time
	URL         string    // URL original para validação
	LastValidAt time.Time // Última vez que a URL foi validada
	FailCount   int       // Número de falhas consecutivas
	Source      string    // Fonte do stream (AnimeFire, AllAnime, etc)
}

// IsExpired verifica se a entrada expirou
func (c *CacheEntry) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// NeedsValidation verifica se a URL precisa ser revalidada
// Valida a cada 2 minutos ou se houve falhas
func (c *CacheEntry) NeedsValidation() bool {
	// Se expirou, precisa de novo fetch, não apenas validação
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

// StreamCacheEntry é uma entrada de cache específica para streams com mais metadados
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

// SourceFailure rastreia falhas de fontes específicas
type SourceFailure struct {
	Source     string
	FailedAt   time.Time
	FailCount  int
	LastError  string
	RetryAfter time.Time
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
	ctx    context.Context
	client *goanime.Client
	User   *store.UserData

	// Smart Router para fontes de vídeo
	streamRouter *smartrouter.SmartRouter

	// Cache unificado com TTL
	cache      map[string]*CacheEntry
	cacheMutex sync.RWMutex

	// Cache específicos de alta performance (sem TTL para itens críticos)
	episodesCache  map[string][]store.Episode
	urlCache       map[string]string
	topAnimesCache []store.SavedAnime
	trendingCache  []*AniListAnime

	// Cache inteligente de streams com validação
	streamCache      map[string]*StreamCacheEntry
	streamCacheMutex sync.RWMutex

	// Rastreamento de falhas por fonte
	sourceFailures      map[string]*SourceFailure
	sourceFailuresMutex sync.RWMutex

	// Prefetch de episódios (carrega próximos episódios em background)
	prefetchQueue  chan PrefetchRequest
	prefetchActive map[string]bool
	prefetchMutex  sync.RWMutex

	// Cache para imagens HD do AniList
	hdImageCache map[string]*anilist.AnimeMedia

	// Proxy de vídeo para contornar CORS
	proxyServer     *http.Server
	proxyPort       int
	currentVideoURL string
	proxyMutex      sync.RWMutex

	// Estado de inicialização
	initialized bool
	initMutex   sync.RWMutex

	// HTTP client para validação de URLs
	validationClient *http.Client
}

// Cache TTLs
const (
	CacheTTLSearch   = 10 * time.Minute // Buscas
	CacheTTLTrending = 30 * time.Minute // Trending
	CacheTTLTop      = 1 * time.Hour    // Top animes
	CacheTTLEpisodes = 1 * time.Hour    // Episódios (aumentado para não perder dados)
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
		validationClient: &http.Client{
			Timeout: 3 * time.Second, // Reduzido para resposta mais rápida
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
		CircuitResetTime: 30 * time.Second, // Tenta resetar após 30s
		DefaultTimeout:   5 * time.Second,
	})

	// Adiciona fontes de streaming em ordem de prioridade
	// Prioridade 1: Enime API (mais rápida, timeout curto)
	app.streamRouter.AddSource(smartrouter.StreamSource{
		Name:     "Enime",
		Priority: 1,
		Timeout:  3 * time.Second, // Timeout curto para não travar
		Fetcher: func(ctx context.Context, title string, ep int) (string, error) {
			return enime.FindAndGetStreamWithContext(ctx, title, ep)
		},
	})

	// Prioridade 2: Consumet API (fallback confiável)
	app.streamRouter.AddSource(smartrouter.StreamSource{
		Name:     "Consumet",
		Priority: 2,
		Timeout:  5 * time.Second,
		Fetcher: func(ctx context.Context, title string, ep int) (string, error) {
			url, _, err := consumet.FindAnimeAndGetStream(title, ep)
			return url, err
		},
	})

	return app
}

// getCache recupera um item do cache se não expirou
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

// ClearEpisodesCache limpa o cache de episódios para forçar recarga
func (a *App) ClearEpisodesCache() {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	a.episodesCache = make(map[string][]store.Episode)
	fmt.Println("[Cache] Cache de episódios limpo")
}

// ClearAllCache limpa todo o cache (útil para resolver problemas)
func (a *App) ClearAllCache() {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	a.cache = make(map[string]*CacheEntry)
	a.episodesCache = make(map[string][]store.Episode)
	a.urlCache = make(map[string]string)

	// Limpa também o cache de streams e falhas
	a.streamCacheMutex.Lock()
	a.streamCache = make(map[string]*StreamCacheEntry)
	a.streamCacheMutex.Unlock()

	a.sourceFailuresMutex.Lock()
	a.sourceFailures = make(map[string]*SourceFailure)
	a.sourceFailuresMutex.Unlock()

	fmt.Println("[Cache] Todo o cache foi limpo")
}

// === SISTEMA DE CACHE INTELIGENTE COM VALIDAÇÃO ===

// ValidateStreamURL verifica se uma URL de stream ainda é acessível
// Usa HEAD request para ser rápido e não consumir banda
func (a *App) ValidateStreamURL(url string) (bool, error) {
	if url == "" {
		return false, fmt.Errorf("URL vazia")
	}

	fmt.Printf("[ValidateURL] Verificando: %s\n", url)

	// Cria request HEAD (não baixa o conteúdo, só verifica headers)
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
		fmt.Printf("[ValidateURL] Erro na requisição: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	// Status 2xx ou 3xx é válido
	isValid := resp.StatusCode >= 200 && resp.StatusCode < 400
	fmt.Printf("[ValidateURL] Status: %d, Válido: %v\n", resp.StatusCode, isValid)

	return isValid, nil
}

// GetValidatedStreamCache obtém stream do cache, validando se ainda funciona
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

		// Valida em goroutine para não bloquear, mas retorna o cache atual
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
					fmt.Printf("[SmartCache] URL inválida (falha %d): %s - %v\n", cached.FailCount, cached.URL, err)

					// Se falhou 3 vezes, remove do cache
					if cached.FailCount >= 3 {
						delete(a.streamCache, k)
						a.recordSourceFailure(cached.Source, "URL inválida após múltiplas tentativas")
						fmt.Printf("[SmartCache] Cache removido após 3 falhas: %s\n", k)
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
		IsValidated: true, // Assume válido no momento do cache
		FailCount:   0,
	}

	fmt.Printf("[SmartCache] Stream cacheado: %s -> %s (source: %s)\n", key, url, source)
}

// InvalidateStreamCache invalida uma entrada específica do cache de streams
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

		fmt.Printf("[SourceTracker] Falha %d para %s: %s (retry após %v)\n",
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

	// Reseta falhas após sucesso
	if failure, exists := a.sourceFailures[source]; exists {
		fmt.Printf("[SourceTracker] Fonte %s recuperada após %d falhas\n", source, failure.FailCount)
		delete(a.sourceFailures, source)
	}
}

// IsSourceAvailable verifica se uma fonte está disponível (não em cooldown)
func (a *App) IsSourceAvailable(source string) bool {
	a.sourceFailuresMutex.RLock()
	defer a.sourceFailuresMutex.RUnlock()

	if failure, exists := a.sourceFailures[source]; exists {
		if time.Now().Before(failure.RetryAfter) {
			fmt.Printf("[SourceTracker] Fonte %s em cooldown até %v\n", source, failure.RetryAfter)
			return false
		}
	}
	return true
}

// GetAlternativeSource retorna a melhor fonte alternativa disponível
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

		// Verifica se a fonte está disponível
		if failure, exists := a.sourceFailures[source]; exists {
			if time.Now().Before(failure.RetryAfter) {
				continue // Ainda em cooldown
			}
		}

		fmt.Printf("[SourceTracker] Fonte alternativa selecionada: %s\n", source)
		return source
	}

	// Se todas estão em cooldown, retorna a com menor tempo de espera
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

// SourceStatus representa o status de uma fonte de vídeo para o frontend
type SourceStatus struct {
	Name        string `json:"name"`
	IsAvailable bool   `json:"isAvailable"`
	FailCount   int    `json:"failCount"`
	LastError   string `json:"lastError,omitempty"`
	RetryAfter  string `json:"retryAfter,omitempty"`
	CachedURLs  int    `json:"cachedUrls"`
}

// CacheStats representa estatísticas do cache para o frontend
type CacheStats struct {
	Sources      []SourceStatus `json:"sources"`
	TotalStreams int            `json:"totalStreams"`
	TotalCache   int            `json:"totalCache"`
}

// GetCacheStats retorna estatísticas do cache e status das fontes
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

// ResetSourceFailures reseta todas as falhas de fontes (útil para debug)
func (a *App) ResetSourceFailures() {
	a.sourceFailuresMutex.Lock()
	defer a.sourceFailuresMutex.Unlock()

	a.sourceFailures = make(map[string]*SourceFailure)
	fmt.Println("[SourceTracker] Todas as falhas foram resetadas")
}

// startVideoProxy inicia um servidor HTTP local para fazer proxy do vídeo
func (a *App) startVideoProxy() error {
	if a.proxyServer != nil {
		return nil // Já está rodando
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
	mux.HandleFunc("/proxy/", a.handleGenericProxy) // Para segmentos HLS

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
	// URL está no path: /proxy/https://...
	targetURL := strings.TrimPrefix(r.URL.Path, "/proxy/")
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	if targetURL == "" {
		http.Error(w, "URL não especificada", http.StatusBadRequest)
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

// handleVideoProxy faz proxy do vídeo remoto para o cliente local
func (a *App) handleVideoProxy(w http.ResponseWriter, r *http.Request) {
	a.proxyMutex.RLock()
	videoURL := a.currentVideoURL
	a.proxyMutex.RUnlock()

	if videoURL == "" {
		http.Error(w, "Nenhum vídeo configurado", http.StatusBadRequest)
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

	// Copia headers da requisição original (para suportar Range requests)
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
		// SharePoint precisa de headers específicos
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
		fmt.Printf("[VideoProxy] Erro na requisição: %v\n", err)
		http.Error(w, "Erro ao acessar vídeo", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Verifica se a resposta foi bem sucedida
	if resp.StatusCode >= 400 {
		fmt.Printf("[VideoProxy] Servidor remoto retornou erro: %d %s\n", resp.StatusCode, resp.Status)
		// Se for um erro de autenticação, tenta sem proxy
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			fmt.Printf("[VideoProxy] Erro de autenticação - URL pode requerer acesso direto\n")
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
		body, err := ioutil.ReadAll(resp.Body)
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
				// Mantém comentários e linhas vazias
				newLines = append(newLines, line)
			} else {
				// É uma URL de segmento
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

// GetProxyURLForVideo retorna a URL do proxy local para um vídeo
func (a *App) GetProxyURLForVideo(videoURL string) (string, error) {
	// Inicia o proxy se ainda não estiver rodando
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

	// Pré-carrega dados em background para inicialização rápida
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
// Não bloqueia - carrega progressivamente e emite eventos
func (a *App) preloadData() {
	// Carrega trending do AniList PRIMEIRO (prioridade alta - usado no hero)
	// É o mais rápido, então carrega primeiro
	go func() {
		if animes, err := a.fetchTrendingInternal(15); err == nil && len(animes) > 0 {
			a.cacheMutex.Lock()
			a.trendingCache = animes
			a.cacheMutex.Unlock()
			fmt.Println("[preload] Trending carregado:", len(animes))
		}
	}()

	// Carrega top animes em background (prioridade baixa)
	// Usa versão otimizada que não bloqueia
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

	fmt.Println("[startup] Pré-carregamento iniciado em background")
}

// fetchTopAnimesOptimized busca top animes de forma otimizada
// Primeiro tenta Jikan (rápido), depois enriquece com fontes reais em background
func (a *App) fetchTopAnimesOptimized() ([]store.SavedAnime, error) {
	// FASE 1: Busca rápida do Jikan (já tem imagens)
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
				URL:   "", // Será preenchido quando o usuário clicar
			})
		}

		// FASE 2: Em background, busca URLs reais (não bloqueia)
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

	sem := make(chan struct{}, 3) // Apenas 3 paralelos para não sobrecarregar
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

// IsFullscreen verifica se a janela está em tela cheia
func (a *App) WindowMaximise() {
	runtime.WindowMaximise(a.ctx)
}

func (a *App) WindowUnmaximise() {
	runtime.WindowUnmaximise(a.ctx)
}

// --- FUNÇÕES EXPORTADAS ---

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

	// Verifica se já existe
	for _, fav := range a.User.Favorites {
		if fav.URL == anime.URL || fav.Title == anime.Title {
			return false // Já existe
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

// IsFavorite verifica se um anime está nos favoritos
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

// === HISTÓRICO DE VISUALIZAÇÃO ===

// GetWatchHistory retorna os últimos episódios assistidos
func (a *App) GetWatchHistory() []store.WatchedEpisode {
	if a.User == nil {
		return []store.WatchedEpisode{}
	}
	return a.User.WatchHistory
}

// AddToWatchHistory adiciona um episódio ao histórico
func (a *App) AddToWatchHistory(episode store.WatchedEpisode) {
	if a.User == nil {
		return
	}

	// Define timestamp se não especificado
	if episode.WatchedAt == "" {
		episode.WatchedAt = time.Now().Format(time.RFC3339)
	}

	// Remove entrada duplicada (mesmo episódio)
	for i, e := range a.User.WatchHistory {
		if e.EpisodeURL == episode.EpisodeURL {
			a.User.WatchHistory = append(a.User.WatchHistory[:i], a.User.WatchHistory[i+1:]...)
			break
		}
	}

	// Adiciona no início (mais recente primeiro)
	a.User.WatchHistory = append([]store.WatchedEpisode{episode}, a.User.WatchHistory...)

	// Limita a 50 entradas
	if len(a.User.WatchHistory) > 50 {
		a.User.WatchHistory = a.User.WatchHistory[:50]
	}

	store.SaveUser(a.User)
}

// === CONFIGURAÇÕES ===

// GetSettings retorna as configurações do utilizador
func (a *App) GetSettings() store.UserSettings {
	if a.User == nil {
		return store.GetDefaultSettings()
	}
	return a.User.Settings
}

// SaveSettings guarda as configurações do utilizador
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
		return "", fmt.Errorf("utilizador não encontrado")
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

	// Valida dados mínimos
	if userData.Username == "" {
		return fmt.Errorf("nome de utilizador inválido")
	}

	// Garante que settings tem valores padrão se vazios
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

	// Verifica cache pré-carregado primeiro (mais rápido)
	a.cacheMutex.RLock()
	if len(a.trendingCache) > 0 {
		cached := a.trendingCache
		a.cacheMutex.RUnlock()
		fmt.Println("[GetTrendingAnimes] Retornando cache pré-carregado")
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

	// Atualiza cache pré-carregado também
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

// GetAnimeHDImage busca imagem HD de um anime pelo título
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

// ConsumetEpisode representa um episódio do Consumet
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

// GetConsumetEpisodes busca episódios de um anime via Consumet
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

// GetStreamWithFallback tenta múltiplas fontes para obter stream
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

// SmartStreamResult é o resultado da busca inteligente de stream
type SmartStreamResult struct {
	URL      string  `json:"url"`
	Source   string  `json:"source"`
	Duration float64 `json:"duration"` // em milissegundos
	Success  bool    `json:"success"`
	Error    string  `json:"error,omitempty"`
}

// GetSmartStream usa o Smart Router para buscar stream com fallback automático
// Esta função tenta múltiplas fontes com timeout e circuit breaker
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

// GetStreamSourceStats retorna estatísticas das fontes de streaming
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

// SkipTimesResult contém os timestamps para pular abertura/encerramento
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

// GetSkipTimes busca os timestamps de abertura/encerramento para um episódio
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

	// Salva no cache (30 minutos - skip times não mudam)
	a.setCache(cacheKey, result, 30*time.Minute)

	return result, nil
}

// GetSkipTimesAsync busca skip times de forma assíncrona
// Retorna imediatamente e o resultado é obtido depois
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
// ENIME API - Fonte de vídeo rápida
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

// normalizeAnimeName normaliza o nome do anime para comparação
func normalizeAnimeName(name string) string {
	// Remove prefixos de fonte
	name = regexp.MustCompile(`^\[.*?\]\s*`).ReplaceAllString(name, "")
	// Remove sufixos comuns
	name = regexp.MustCompile(`\s*\((?:Dublado|Legendado|Dub|Sub|TV|OVA|Movie)\).*$`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`\s*-\s*(?:Season|Part|Temporada).*$`).ReplaceAllString(name, "")
	// Remove números de episódios
	name = regexp.MustCompile(`\s*\(\d+\s*episodes?\).*$`).ReplaceAllString(name, "")
	// Normaliza espaços e lowercase
	name = strings.TrimSpace(strings.ToLower(name))
	// Remove caracteres especiais
	name = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	return name
}

// BuscarAnimes - busca RÁPIDA em ambas as fontes com cache otimizado
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

	resultChan := make(chan searchResult, 2)

	// AllAnime (inglês)
	go func() {
		srcAllAnime := types.SourceAllAnime
		animes, err := a.client.SearchAnime(termo, &srcAllAnime)
		resultChan <- searchResult{animes, "AllAnime", err}
	}()

	// AnimeFire (português)
	go func() {
		srcAnimeFire := types.SourceAnimeFire
		animes, err := a.client.SearchAnime(termo, &srcAnimeFire)
		resultChan <- searchResult{animes, "AnimeFire", err}
	}()

	// Coleta resultados com timeout de 4 segundos
	animeMap := make(map[string]*store.SavedAnime)
	timeout := time.After(4 * time.Second)
	received := 0

	for received < 2 {
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
			received = 2
		}
	}

	// Converte para slice
	final := make([]store.SavedAnime, 0, len(animeMap))
	for _, anime := range animeMap {
		final = append(final, *anime)
	}

	// Ordena por número de fontes
	sort.Slice(final, func(i, j int) bool {
		return len(final[i].Sources) > len(final[j].Sources)
	})

	// Busca imagens em paralelo para animes sem imagem (máximo 10 concurrent)
	var imgWg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Aumentado para 10 goroutines simultâneas

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

	// Espera busca de imagens (com timeout de 2.5s - APIs já tem timeout interno)
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

// GetTopAnimes - retorna cache pré-carregado ou busca
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

	// Se não conseguiu nenhum, fallback para Jikan
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
		return "", fmt.Errorf("anime não encontrado")
	}

	url := searchResults[0].URL

	// Salva no cache
	a.cacheMutex.Lock()
	a.urlCache[title] = url
	a.cacheMutex.Unlock()

	return url, nil
}

// BuscarAnimesMulti - busca MÚLTIPLOS termos em PARALELO (para gêneros)
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

	// Lança TODAS as buscas em paralelo
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
			fmt.Printf("[BuscarAnimesMulti] Timeout após %d/%d buscas\n", received, len(termos))
			received = len(termos)
		}
	}

	fmt.Printf("[BuscarAnimesMulti] Total: %d animes únicos\n", len(allResults))
	return allResults, nil
}

// GetEpisodes - otimizado com cache e busca paralela
func (a *App) GetEpisodes(seriesURL string) ([]store.Episode, error) {
	if seriesURL == "" {
		return nil, fmt.Errorf("URL inválida")
	}

	// Verifica cache
	a.cacheMutex.RLock()
	if eps, ok := a.episodesCache[seriesURL]; ok && len(eps) > 0 {
		a.cacheMutex.RUnlock()
		fmt.Printf("[GetEpisodes] Cache hit: %d episódios\n", len(eps))
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
			// Usa o resultado com mais episódios
			if len(mapped) > len(bestEpisodes) {
				bestEpisodes = mapped
				fmt.Printf("[GetEpisodes] %s: %d episódios (melhor até agora)\n", res.source, len(mapped))
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

	// Fallback heurístico
	return a.getEpisodesFallback(seriesURL)
}

// GetEpisodesForSource - busca episódios de uma fonte específica
func (a *App) GetEpisodesForSource(sourceURL string, sourceName string) ([]store.Episode, error) {
	if sourceURL == "" {
		return nil, fmt.Errorf("URL inválida")
	}

	cacheKey := fmt.Sprintf("%s:%s", sourceName, sourceURL)

	// Verifica cache
	a.cacheMutex.RLock()
	if eps, ok := a.episodesCache[cacheKey]; ok && len(eps) > 0 {
		a.cacheMutex.RUnlock()
		fmt.Printf("[GetEpisodesForSource] Cache hit: %d episódios de %s\n", len(eps), sourceName)
		return eps, nil
	}
	a.cacheMutex.RUnlock()

	if a.client == nil {
		a.client = goanime.NewClient()
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

	fmt.Printf("[GetEpisodesForSource] Encontrados: %d episódios\n", len(mapped))
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

	body, err := ioutil.ReadAll(resp.Body)
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
		fmt.Printf("[GetEpisodes] Encontrados %d episódios via JSON-LD\n", len(eps))
		a.episodesCache[seriesURL] = eps
		return eps, nil
	}

	// 2) procura arrays JS
	eps = parseJSArrays(seriesURL, html)
	if len(eps) > 0 {
		for i := range eps {
			eps[i].Source = fallbackSource
		}
		fmt.Printf("[GetEpisodes] Encontrados %d episódios via JS arrays\n", len(eps))
		a.episodesCache[seriesURL] = eps
		return eps, nil
	}

	// 3) procura data-attributes
	eps = parseDataAttributes(seriesURL, html)
	if len(eps) > 0 {
		for i := range eps {
			eps[i].Source = fallbackSource
		}
		fmt.Printf("[GetEpisodes] Encontrados %d episódios via data-attributes\n", len(eps))
		a.episodesCache[seriesURL] = eps
		return eps, nil
	}

	// 4) Para AnimeFire: busca específica por links de episódios
	if strings.Contains(seriesURL, "animefire") {
		eps := a.parseAnimeFireEpisodes(seriesURL, html)
		if len(eps) > 0 {
			for i := range eps {
				eps[i].Source = fallbackSource
			}
			fmt.Printf("[GetEpisodes] Encontrados %d episódios via AnimeFire parser\n", len(eps))
			a.episodesCache[seriesURL] = eps
			return eps, nil
		}
	}

	// 5) fallback clássico: procura <a> mas com filtros mais rigorosos
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

		// Ignora URLs que não são do mesmo anime
		if baseSlug != "" && !strings.Contains(href, baseSlug) {
			continue
		}

		// Ignora URLs de outros sites (youtube, etc)
		if strings.Contains(href, "youtube") || strings.Contains(href, "blogger") ||
			strings.Contains(href, "google") || strings.Contains(href, "facebook") {
			continue
		}

		// Procura padrão de episódio: /anime-slug/NUMERO
		epNumRe := regexp.MustCompile(`/([^/]+)/(\d+)$`)
		epMatch := epNumRe.FindStringSubmatch(href)
		if epMatch != nil {
			num, _ := strconv.Atoi(epMatch[2])
			if num > 0 && num < 2000 {
				// Evita duplicatas - mantém pelo número
				if _, exists := episodesMap[num]; !exists {
					// Extrai nome do anime do slug
					animeSlug := epMatch[1]
					title := fmt.Sprintf("Episódio %d", num)

					// Formata título bonito
					if text != "" && len(text) < 100 {
						title = text
					} else {
						// Converte slug para título
						niceName := strings.ReplaceAll(animeSlug, "-", " ")
						niceName = strings.Title(niceName)
						title = fmt.Sprintf("%s - Episódio %d", niceName, num)
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

	fmt.Printf("[GetEpisodes] Encontrados %d possíveis episódios (heurística)\n", len(final))
	a.episodesCache[seriesURL] = final
	return final, nil
}

// parseAnimeFireEpisodes extrai episódios especificamente do AnimeFire
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
		// Tenta extrair o slug de outra forma - último segmento da URL
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

	// Método 1: Procura divNumEP que é o container de episódios do AnimeFire
	// <div class="divNumEP"><a href="https://animefire.plus/animes/slug/1">1</a></div>
	divPattern := regexp.MustCompile(`class="divNumEP"[^>]*>.*?<a[^>]*href=["']([^"']+)["'][^>]*>`)
	divMatches := divPattern.FindAllStringSubmatch(html, -1)

	fmt.Printf("[parseAnimeFireEpisodes] divNumEP matches: %d\n", len(divMatches))

	for _, m := range divMatches {
		if len(m) >= 2 {
			url := m[1]
			// Extrai número do episódio da URL (último segmento)
			urlParts := strings.Split(strings.TrimSuffix(url, "/"), "/")
			if len(urlParts) > 0 {
				numStr := urlParts[len(urlParts)-1]
				num, err := strconv.Atoi(numStr)
				if err == nil && num > 0 && num < 2000 && !seen[num] {
					seen[num] = true
					niceName := strings.ReplaceAll(baseSlug, "-", " ")
					niceName = strings.Title(niceName)
					title := fmt.Sprintf("%s - Episódio %d", niceName, num)
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

	// Método 2: Procura padrão mais genérico de links /animes/SLUG/NUMERO
	if len(episodes) == 0 {
		// Padrão genérico para links de episódios AnimeFire
		genericPattern := regexp.MustCompile(`href=["'](https?://[^"']*animefire[^"']*/animes/[^/]+/(\d+))["']`)
		genericMatches := genericPattern.FindAllStringSubmatch(html, -1)

		fmt.Printf("[parseAnimeFireEpisodes] Generic pattern matches: %d\n", len(genericMatches))

		for _, m := range genericMatches {
			if len(m) >= 3 {
				url := m[1]
				num, _ := strconv.Atoi(m[2])
				// Verifica se o URL contém parte do slug (flexível)
				slugWords := strings.Split(baseSlug, "-")
				matchCount := 0
				for _, word := range slugWords {
					if len(word) > 2 && strings.Contains(strings.ToLower(url), strings.ToLower(word)) {
						matchCount++
					}
				}
				// Se pelo menos metade das palavras combinam, consideramos válido
				if matchCount >= len(slugWords)/2 || strings.Contains(url, baseSlug) {
					if num > 0 && num < 2000 && !seen[num] {
						seen[num] = true
						niceName := strings.ReplaceAll(baseSlug, "-", " ")
						niceName = strings.Title(niceName)
						title := fmt.Sprintf("%s - Episódio %d", niceName, num)
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

	// Método 3: Se ainda vazio, procura qualquer link para animes com número no final
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
					niceName = strings.Title(niceName)
					title := fmt.Sprintf("%s - Episódio %d", niceName, num)
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

	fmt.Printf("[parseAnimeFireEpisodes] Total episódios encontrados: %d\n", len(episodes))

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

// parseJSONLDScripts tenta extrair episódios de <script type="application/ld+json"> contendo Episode
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

// PlayVideo recebe o link e o título e manda pro MPV
// Usa a implementação robusta de PlayAnime que procura o MPV em vários locais
func (a *App) PlayVideo(url string, title string) error {
	fmt.Printf("[PlayVideo] Frontend pediu play: %s (URL: %s)\n", title, url)
	return a.PlayAnime(url)
}

// PlayAnime reproduz o anime no MPV (com suporte a yt-dlp para URLs complexas)
func (a *App) PlayAnime(url string) error {
	if url == "" {
		return fmt.Errorf("URL inválida")
	}

	fmt.Printf("Iniciando MPV com URL: %s\n", url)

	// Encontra o caminho do MPV
	mpvPath := a.findMPVPath()
	if mpvPath == "" {
		return fmt.Errorf("MPV não encontrado. Instale o MPV ou coloque na pasta bin/")
	}

	// Argumentos base do MPV
	args := []string{
		"--force-window=immediate",
		"--hwdec=auto",
		"--vo=gpu",
	}

	// Só usa yt-dlp se a URL NÃO for um stream direto
	isDirectStream := strings.HasSuffix(url, ".mp4") ||
		strings.HasSuffix(url, ".m3u8") ||
		strings.HasSuffix(url, ".webm") ||
		strings.Contains(url, ".mp4?") ||
		strings.Contains(url, ".m3u8?")

	if !isDirectStream && (strings.Contains(url, "/video/") || strings.Contains(url, "/embed/")) {
		// Verifica se yt-dlp está disponível
		ytdlpPath := a.findYtdlpPath()
		if ytdlpPath != "" {
			fmt.Printf("[PlayAnime] URL é página web, usando yt-dlp: %s\n", ytdlpPath)
			args = append(args, "--ytdl-path="+ytdlpPath)
			args = append(args, "--ytdl-format=best")
		} else {
			fmt.Println("[PlayAnime] yt-dlp não encontrado, tentando direto...")
		}
	} else {
		fmt.Printf("[PlayAnime] URL é stream direto, reproduzindo diretamente\n")
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

// findMPVPath procura o MPV em vários locais
func (a *App) findMPVPath() string {
	// 1) Caminho salvo pelo usuário
	if a.User != nil && a.User.MPVPath != "" {
		if _, err := os.Stat(a.User.MPVPath); err == nil {
			return a.User.MPVPath
		}
	}

	// 2) mpv no PATH
	if path, err := exec.LookPath("mpv"); err == nil {
		return path
	}

	// 3) caminhos possíveis
	possiblePaths := []string{}

	// Diretório atual
	if dir, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(dir, "bin", "mpv.exe"))
	}

	possiblePaths = append(possiblePaths,
		"bin/mpv.exe",
		"C:\\Program Files\\mpv\\mpv.exe",
		"C:\\Users\\"+os.Getenv("USERNAME")+"\\AppData\\Local\\mpv\\mpv.exe",
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

// GetStreamURLForEpisode retorna a URL real do vídeo (WebSprit, CDN) usando a biblioteca GoAnime
// Implementa cache inteligente com validação de URL e fallback automático entre fontes
func (a *App) GetStreamURLForEpisode(animeURL string, episodeURL string) (string, error) {
	if a.client == nil {
		a.client = goanime.NewClient()
	}

	fmt.Printf("[GetStreamURLForEpisode] AnimeURL: %s, EpisodeURL: %s\n", animeURL, episodeURL)

	// === CACHE INTELIGENTE COM VALIDAÇÃO ===
	cacheKey := fmt.Sprintf("stream:%s", episodeURL)

	// Primeiro, tenta o cache inteligente de streams
	if cachedURL, ok := a.GetValidatedStreamCache(cacheKey); ok {
		fmt.Println("[GetStreamURLForEpisode] Cache inteligente hit!")
		return cachedURL, nil
	}

	// Fallback para cache antigo (migração)
	if cached, ok := a.getCache(cacheKey); ok {
		cachedURL := cached.(string)
		fmt.Println("[GetStreamURLForEpisode] Cache legado hit, validando...")

		// Valida a URL do cache antigo
		if valid, _ := a.ValidateStreamURL(cachedURL); valid {
			// Migra para novo cache inteligente
			a.SetStreamCache(cacheKey, cachedURL, "legacy", CacheTTLStream)
			return cachedURL, nil
		}
		fmt.Println("[GetStreamURLForEpisode] URL do cache legado inválida, buscando nova...")
	}

	// === DETECÇÃO DE FONTE ===
	var primarySource types.Source
	var sourceName string
	if strings.Contains(strings.ToLower(animeURL), "animefire") || strings.Contains(strings.ToLower(episodeURL), "animefire") {
		primarySource = types.SourceAnimeFire
		sourceName = "AnimeFire"
	} else {
		primarySource = types.SourceAllAnime
		sourceName = "AllAnime"
	}

	// Verifica se a fonte primária está disponível (não em cooldown)
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

	// === BUSCA EPISÓDIO NO CACHE ===
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
		fmt.Printf("[GetStreamURLForEpisode] Episódios não encontrados no cache para: %s\n", animeURL)
		return "", fmt.Errorf("episódios não encontrados no cache")
	}

	var targetEpisode *store.Episode
	for i := range cachedEpisodes {
		if cachedEpisodes[i].URL == episodeURL {
			targetEpisode = &cachedEpisodes[i]
			break
		}
	}

	if targetEpisode == nil {
		fmt.Printf("[GetStreamURLForEpisode] Episódio não encontrado: %s\n", episodeURL)
		return "", fmt.Errorf("episódio não encontrado")
	}

	fmt.Printf("[GetStreamURLForEpisode] Episódio: %s (Número: %d, Source: '%s')\n",
		targetEpisode.Title, targetEpisode.Number, targetEpisode.Source)

	if targetEpisode.Source != "" {
		if parsed, err := types.ParseSource(targetEpisode.Source); err == nil {
			primarySource = parsed
		}
	}

	fmt.Printf("[GetStreamURLForEpisode] Usando source: %v\n", primarySource)

	// === BUSCA PARALELA DE TODAS AS FONTES ===
	// Lança goroutines para buscar de todas as fontes simultaneamente
	// Retorna assim que a primeira encontrar uma URL válida

	allSources := []struct {
		name   string
		source types.Source
	}{
		{sourceName, primarySource}, // Fonte primária primeiro
		{"AllAnime", types.SourceAllAnime},
		{"AnimeFire", types.SourceAnimeFire},
	}

	// Remove duplicatas (se primarySource já está na lista)
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

	// Lança goroutines para cada fonte
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
				resultChan <- streamResult{url: "", source: srcName, err: fmt.Errorf("URL inválida")}
				return
			}
			resultChan <- streamResult{url: "", source: srcName, err: err}
		}(src.name, src.source)
	}

	// Também tenta o Smart Router em paralelo
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

	// Espera pelo primeiro resultado válido ou timeout
	timeout := time.After(10 * time.Second)
	var lastError error

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				// Canal fechado, todas as fontes falharam
				return "", fmt.Errorf("não foi possível obter stream URL de nenhuma fonte: %v", lastError)
			}

			if result.url != "" {
				// Encontrou! Cacheia e retorna imediatamente
				a.SetStreamCache(cacheKey, result.url, result.source, CacheTTLStream)
				a.recordSourceSuccess(result.source)
				fmt.Printf("[GetStreamURLForEpisode] ✓ Sucesso com %s (paralelo)!\n", result.source)

				// Inicia prefetch dos próximos episódios em background
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

// prefetchNextEpisodes pré-carrega URLs dos próximos episódios em background
func (a *App) prefetchNextEpisodes(animeURL string, episodes []store.Episode, currentEpNum int) {
	// Pré-carrega os próximos 2 episódios
	for _, ep := range episodes {
		if ep.Number > currentEpNum && ep.Number <= currentEpNum+2 {
			cacheKey := fmt.Sprintf("stream:%s", ep.URL)

			// Verifica se já está no cache
			if _, ok := a.GetValidatedStreamCache(cacheKey); ok {
				continue
			}

			// Verifica se já está em prefetch
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

			fmt.Printf("[Prefetch] Pré-carregando episódio %d...\n", ep.Number)

			// Busca em background (sem esperar)
			go func(episode store.Episode) {
				defer func() {
					a.prefetchMutex.Lock()
					delete(a.prefetchActive, fmt.Sprintf("stream:%s", episode.URL))
					a.prefetchMutex.Unlock()
				}()

				// Usa apenas a fonte primária para prefetch (mais rápido)
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
						fmt.Printf("[Prefetch] ✓ Episódio %d pré-carregado!\n", episode.Number)
					}
				}
			}(ep)
		}
	}
}

// tryGetStreamFromSource tenta obter stream de uma fonte específica
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

	// Tenta usar o método da biblioteca primeiro
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
			return "", fmt.Errorf("falha na extração HTML: %w", err)
		}
		if streamURL != "" {
			fmt.Printf("[tryGetStreamFromSource] Stream extraído do HTML: %s\n", streamURL)
			return streamURL, nil
		}
	}

	return "", fmt.Errorf("fonte %s falhou: %v", source, err)
}

// AssistirEpisodio é a função MÁGICA que o Svelte vai chamar ao clicar no episódio
func (a *App) AssistirEpisodio(animeURL string, episodeURL string, episodeTitle string) error {
	fmt.Printf("[AssistirEpisodio] Iniciando processo para: %s\n", episodeTitle)

	// 1. Extrai o link real do vídeo (MP4/M3U8) usando sua função existente
	streamURL, err := a.GetStreamURLForEpisode(animeURL, episodeURL)
	if err != nil {
		fmt.Printf("[AssistirEpisodio] Falha ao extrair link: %v\n", err)
		return fmt.Errorf("não foi possível extrair o vídeo: %v", err)
	}

	if streamURL == "" {
		return fmt.Errorf("link de vídeo retornado vazio")
	}

	fmt.Printf("[AssistirEpisodio] Link extraído com sucesso: %s\n", streamURL)

	// 2. Manda o MPV tocar o link REAL do vídeo
	return a.PlayVideo(streamURL, episodeTitle)
}

// ============================================
// DISCORD INTEGRATION
// ============================================

// DiscordRecommendation representa uma recomendação para o frontend
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

// DiscordStatus retorna o status da conexão Discord
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
		ChannelName: "#recomendações",
	}
}

// ConnectDiscord configura a conexão com Discord via webhook
func (a *App) ConnectDiscord(webhookURL string) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook URL é obrigatório")
	}

	bot := discord.GetBot()
	bot.Configure(webhookURL)

	// Carrega recomendações de exemplo para demonstração
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

// GetDiscordRecommendations retorna as recomendações do Discord
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

// SendDiscordRecommendation envia uma recomendação para o Discord
func (a *App) SendDiscordRecommendation(animeTitle, animeImage string, animeScore float64, message string) error {
	bot := discord.GetBot()

	if !bot.IsConnected() {
		return fmt.Errorf("Discord não está conectado")
	}

	username := "Usuário"
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
		fmt.Printf("[Discord] Erro ao enviar recomendação: %v\n", err)
		return err
	}

	fmt.Printf("[Discord] Recomendação enviada: %s\n", animeTitle)
	return nil
}

// LikeDiscordRecommendation adiciona um like a uma recomendação
func (a *App) LikeDiscordRecommendation(recID string) bool {
	bot := discord.GetBot()
	return bot.LikeRecommendation(recID)
}

// SimulateDiscordConnect simula conexão para demonstração (sem webhook real)
func (a *App) SimulateDiscordConnect() error {
	bot := discord.GetBot()

	// Configura como conectado (modo demo)
	bot.Configure("demo")

	// Carrega recomendações de exemplo
	bot.AddMockRecommendations()

	fmt.Println("[Discord] Modo demonstração ativado com recomendações de exemplo")
	return nil
}

// DiscordOAuthConfig contém as configurações do OAuth2
type DiscordOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// Configurações do aplicativo Discord OAuth2
// Para o app funcionar, você precisa criar um aplicativo em https://discord.com/developers/applications
// e preencher as credenciais abaixo ou criar um arquivo discord_config.json
var discordOAuth = DiscordOAuthConfig{
	ClientID:     "", // Preencha com seu Client ID ou use discord_config.json
	ClientSecret: "", // Preencha com seu Client Secret ou use discord_config.json
	RedirectURI:  "http://localhost:9876/callback",
}

// InitDiscordOAuth inicializa as credenciais do Discord OAuth
func initDiscordOAuth() {
	// Tenta carregar do arquivo de configuração
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

	// Fallback para variáveis de ambiente
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
		fmt.Println("[Discord] ⚠️ Credenciais não configuradas! Crie um arquivo discord_config.json")
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

	// Atualiza as credenciais em memória
	discordOAuth.ClientID = clientID
	discordOAuth.ClientSecret = clientSecret

	fmt.Println("[Discord] Credenciais salvas com sucesso!")
	return nil
}

// GetDiscordConfigStatus retorna o status da configuração do Discord
func (a *App) GetDiscordConfigStatus() map[string]interface{} {
	return map[string]interface{}{
		"configured":  discordOAuth.ClientID != "" && discordOAuth.ClientSecret != "",
		"hasClientId": discordOAuth.ClientID != "",
		"hasSecret":   discordOAuth.ClientSecret != "",
		"redirectUri": discordOAuth.RedirectURI,
	}
}

// DiscordUserInfo representa as informações do usuário para o frontend
type DiscordUserInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatarUrl"`
	Connected bool   `json:"connected"`
}

// GetDiscordOAuthURL retorna a URL para iniciar o fluxo OAuth2
func (a *App) GetDiscordOAuthURL() string {
	// Scopes necessários: identify para obter informações do usuário
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

	// Abre o navegador com a URL de autorização
	runtime.BrowserOpenURL(a.ctx, authURL)

	fmt.Println("[Discord OAuth] Aguardando autorização do usuário...")

	// Inicia o servidor de callback em uma goroutine
	go func() {
		bot := discord.GetBot()

		// Aguarda o callback
		code, err := bot.OAuth2CallbackServer(discordOAuth.ClientID, discordOAuth.ClientSecret, discordOAuth.RedirectURI)
		if err != nil {
			fmt.Printf("[Discord OAuth] Erro: %v\n", err)
			return
		}

		// Para funcionar sem client_secret, usamos o token implícito
		// Ou buscamos o usuário diretamente com o código
		user, err := a.CompleteDiscordOAuthWithCode(code)
		if err != nil {
			fmt.Printf("[Discord OAuth] Erro ao completar OAuth: %v\n", err)
			return
		}

		fmt.Printf("[Discord OAuth] Usuário conectado: %s\n", user.Username)

		// Emite evento para o frontend
		runtime.EventsEmit(a.ctx, "discord:connected", user)
	}()

	return nil
}

// CompleteDiscordOAuthWithCode completa o fluxo OAuth2 usando o código
func (a *App) CompleteDiscordOAuthWithCode(code string) (*DiscordUserInfo, error) {
	bot := discord.GetBot()

	// Se temos client_secret, troca o código por token
	if discordOAuth.ClientSecret != "" {
		accessToken, err := bot.ExchangeCodeForToken(
			discordOAuth.ClientID,
			discordOAuth.ClientSecret,
			code,
			discordOAuth.RedirectURI,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao trocar código por token: %w", err)
		}

		user, err := bot.FetchUserFromToken(accessToken)
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
		}

		bot.SetCurrentUser(user)

		// Carrega recomendações de exemplo
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

	// Modo sem client_secret: usamos o usuário simulado baseado no código
	// Para produção, você deve ter o client_secret configurado
	fmt.Println("[Discord OAuth] Modo sem client_secret - usando perfil simulado")

	mockUser := &discord.DiscordUser{
		ID:         "oauth_user",
		Username:   "DiscordUser",
		GlobalName: "Discord User",
		AvatarURL:  "https://cdn.discordapp.com/embed/avatars/0.png",
	}

	bot.SetCurrentUser(mockUser)

	// Carrega recomendações de exemplo
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

// GetDiscordUser retorna o usuário Discord atualmente conectado
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

// DisconnectDiscordUser desconecta o usuário Discord OAuth
func (a *App) DisconnectDiscordUser() {
	bot := discord.GetBot()
	bot.Disconnect()
	runtime.EventsEmit(a.ctx, "discord:disconnected", nil)
	fmt.Println("[Discord] Usuário desconectado")
}

// SetDiscordClientSecret configura o client secret do OAuth2
func (a *App) SetDiscordClientSecret(secret string) {
	discordOAuth.ClientSecret = secret
}

// ============================================
// DISCORD LINKING SYSTEM (Vinculação por Código)
// ============================================

// DiscordLinkInfo representa informações da conta vinculada
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

// GetDiscordLinkStatus retorna o status da vinculação
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

// LinkDiscordWithCode vincula a conta Discord usando um código
func (a *App) LinkDiscordWithCode(code string) (DiscordLinkInfo, error) {
	ls := discord.GetLinkingSystem()

	account, err := ls.LinkWithCode(code)
	if err != nil {
		return DiscordLinkInfo{IsLinked: false}, err
	}

	// Emite evento de conexão
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

// GenerateDiscordLinkCode gera um código de vinculação (para debug/teste)
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
