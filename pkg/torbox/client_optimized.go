package torbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	BaseURLOptimized = "https://api.torbox.app"
	VersionOptimized = "v1"

	// Limites de rate limiting
	MaxRequestsPerSecond = 5
	MaxRetries           = 3
	RetryBaseDelay       = 500 * time.Millisecond
)

// =============================================================================
// CACHE LRU THREAD-SAFE
// =============================================================================

type LRUCache struct {
	mu       sync.RWMutex
	capacity int
	items    map[string]*cacheEntry
	order    []string // Mais recente no final
	ttl      time.Duration
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	c := &LRUCache{
		capacity: capacity,
		items:    make(map[string]*cacheEntry),
		order:    make([]string, 0, capacity),
		ttl:      ttl,
	}
	go c.cleanup()
	return c
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.items[key]
	if !exists || time.Now().After(entry.expiresAt) {
		if exists {
			c.removeKey(key)
		}
		return nil, false
	}

	// Move para o final (mais recente)
	c.moveToEnd(key)
	return entry.value, true
}

func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Se já existe, atualiza e move para o final
	if _, exists := c.items[key]; exists {
		c.items[key] = &cacheEntry{
			value:     value,
			expiresAt: time.Now().Add(c.ttl),
		}
		c.moveToEnd(key)
		return
	}

	// Se está cheio, remove o mais antigo (primeiro)
	if len(c.order) >= c.capacity {
		oldest := c.order[0]
		c.removeKey(oldest)
	}

	c.items[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.order = append(c.order, key)
}

func (c *LRUCache) moveToEnd(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			c.order = append(c.order, key)
			break
		}
	}
}

func (c *LRUCache) removeKey(key string) {
	delete(c.items, key)
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

func (c *LRUCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.items {
			if now.After(entry.expiresAt) {
				c.removeKey(key)
			}
		}
		c.mu.Unlock()
	}
}

func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheEntry)
	c.order = make([]string, 0, c.capacity)
}

func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// =============================================================================
// RATE LIMITER
// =============================================================================

type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

func NewRateLimiter(requestsPerSecond float64) *RateLimiter {
	return &RateLimiter{
		tokens:     requestsPerSecond,
		maxTokens:  requestsPerSecond * 2, // Burst de 2x
		refillRate: requestsPerSecond,
		lastRefill: time.Now(),
	}
}

func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Refill tokens baseado no tempo passado
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.refillRate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now

	// Se não tem tokens, espera
	if r.tokens < 1 {
		waitTime := time.Duration((1 - r.tokens) / r.refillRate * float64(time.Second))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
		r.tokens = 0
	} else {
		r.tokens--
	}

	return nil
}

// =============================================================================
// CLIENTE TORBOX OTIMIZADO
// =============================================================================

type OptimizedClient struct {
	apiKey      string
	httpClient  *http.Client
	rateLimiter *RateLimiter

	// Caches
	searchCache  *LRUCache
	torrentCache *LRUCache
	linkCache    *LRUCache

	// Cache de lista de torrents (atualizado periodicamente)
	torrentList    []Torrent
	torrentByHash  map[string]*Torrent
	torrentListMu  sync.RWMutex
	lastListUpdate time.Time

	// Métricas
	requestCount int64
	cacheHits    int64
	cacheMisses  int64
	errorCount   int64
}

// NewOptimizedClient cria um cliente TorBox otimizado
func NewOptimizedClient(apiKey string) *OptimizedClient {
	c := &OptimizedClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 10,
				MaxConnsPerHost:     10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
				ForceAttemptHTTP2:   true,
			},
		},
		rateLimiter:   NewRateLimiter(MaxRequestsPerSecond),
		searchCache:   NewLRUCache(100, 15*time.Minute), // 100 buscas, 15min TTL
		torrentCache:  NewLRUCache(50, 5*time.Minute),   // 50 torrents, 5min TTL
		linkCache:     NewLRUCache(200, 30*time.Minute), // 200 links, 30min TTL
		torrentByHash: make(map[string]*Torrent),
	}

	// Inicia atualização periódica da lista de torrents
	go c.periodicTorrentListUpdate()

	return c
}

// periodicTorrentListUpdate atualiza a lista de torrents a cada 30s
func (c *OptimizedClient) periodicTorrentListUpdate() {
	ticker := time.NewTicker(30 * time.Second)

	// Primeira atualização imediata
	c.refreshTorrentList()

	for range ticker.C {
		c.refreshTorrentList()
	}
}

func (c *OptimizedClient) refreshTorrentList() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	torrents, err := c.getTorrentsNoCache(ctx)
	if err != nil {
		log.Printf("[TorBox] Erro ao atualizar lista: %v", err)
		return
	}

	c.torrentListMu.Lock()
	c.torrentList = torrents
	c.torrentByHash = make(map[string]*Torrent)
	for i := range torrents {
		c.torrentByHash[strings.ToLower(torrents[i].Hash)] = &torrents[i]
	}
	c.lastListUpdate = time.Now()
	c.torrentListMu.Unlock()

	log.Printf("[TorBox] Lista atualizada: %d torrents", len(torrents))
}

// =============================================================================
// REQUISIÇÕES COM RETRY E RATE LIMITING
// =============================================================================

func (c *OptimizedClient) request(ctx context.Context, method, endpoint string, body io.Reader, contentType string) (json.RawMessage, error) {
	atomic.AddInt64(&c.requestCount, 1)

	// Rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit timeout: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < MaxRetries; attempt++ {
		if attempt > 0 {
			// Backoff exponencial
			delay := RetryBaseDelay * time.Duration(1<<attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			log.Printf("[TorBox] Retry %d/%d para %s", attempt+1, MaxRetries, endpoint)
		}

		data, err := c.doRequest(ctx, method, endpoint, body, contentType)
		if err == nil {
			return data, nil
		}

		lastErr = err

		// Não retenta para erros que não vão mudar
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") ||
			strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "invalid") {
			break
		}
	}

	atomic.AddInt64(&c.errorCount, 1)
	return nil, lastErr
}

func (c *OptimizedClient) doRequest(ctx context.Context, method, endpoint string, body io.Reader, contentType string) (json.RawMessage, error) {
	reqURL := fmt.Sprintf("%s/%s/api/%s", BaseURLOptimized, VersionOptimized, endpoint)

	// Se body for nil, cria um buffer vazio
	var bodyData []byte
	if body != nil {
		var err error
		bodyData, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	// Log apenas se debug ou erro
	if resp.StatusCode >= 400 {
		log.Printf("[TorBox] %s %s -> %d: %s", method, endpoint, resp.StatusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API error: %s - %s", apiResp.Error, apiResp.Detail)
	}

	return apiResp.Data, nil
}

// =============================================================================
// MÉTODOS DE TORRENT OTIMIZADOS
// =============================================================================

// GetTorrents retorna lista de torrents (do cache local)
func (c *OptimizedClient) GetTorrents(ctx context.Context) ([]Torrent, error) {
	c.torrentListMu.RLock()
	defer c.torrentListMu.RUnlock()

	// Se cache muito antigo, força refresh
	if time.Since(c.lastListUpdate) > 2*time.Minute {
		go c.refreshTorrentList()
	}

	// Retorna cópia para evitar race conditions
	result := make([]Torrent, len(c.torrentList))
	copy(result, c.torrentList)
	return result, nil
}

// getTorrentsNoCache busca torrents diretamente da API (para refresh interno)
func (c *OptimizedClient) getTorrentsNoCache(ctx context.Context) ([]Torrent, error) {
	data, err := c.request(ctx, "GET", "torrents/mylist", nil, "")
	if err != nil {
		return nil, err
	}

	var torrents []Torrent
	if err := json.Unmarshal(data, &torrents); err != nil {
		return nil, fmt.Errorf("erro ao decodificar torrents: %w", err)
	}

	// Marca arquivos de vídeo
	videoExts := []string{".mkv", ".mp4", ".avi", ".webm", ".mov", ".m4v"}
	for i := range torrents {
		for j := range torrents[i].Files {
			name := strings.ToLower(torrents[i].Files[j].Name)
			for _, ext := range videoExts {
				if strings.HasSuffix(name, ext) {
					torrents[i].Files[j].IsPlayable = true
					break
				}
			}
		}
	}

	return torrents, nil
}

// GetTorrentByHash busca torrent por hash (O(1) no cache local)
func (c *OptimizedClient) GetTorrentByHash(hash string) (*Torrent, bool) {
	c.torrentListMu.RLock()
	defer c.torrentListMu.RUnlock()
	t, exists := c.torrentByHash[strings.ToLower(hash)]
	return t, exists
}

// GetTorrent retorna um torrent específico (com cache)
func (c *OptimizedClient) GetTorrent(ctx context.Context, torrentID int) (*Torrent, error) {
	cacheKey := fmt.Sprintf("torrent:%d", torrentID)

	// Verifica cache
	if cached, ok := c.torrentCache.Get(cacheKey); ok {
		atomic.AddInt64(&c.cacheHits, 1)
		return cached.(*Torrent), nil
	}
	atomic.AddInt64(&c.cacheMisses, 1)

	endpoint := fmt.Sprintf("torrents/mylist?id=%d", torrentID)
	data, err := c.request(ctx, "GET", endpoint, nil, "")
	if err != nil {
		return nil, err
	}

	var torrent Torrent
	if err := json.Unmarshal(data, &torrent); err != nil {
		return nil, fmt.Errorf("erro ao decodificar torrent: %w", err)
	}

	c.torrentCache.Set(cacheKey, &torrent)
	return &torrent, nil
}

// AddMagnet adiciona torrent com multipart/form-data
func (c *OptimizedClient) AddMagnet(ctx context.Context, magnet string, seed bool) (*Torrent, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("magnet", magnet); err != nil {
		return nil, err
	}

	seedValue := "1"
	if seed {
		seedValue = "2"
	}
	if err := writer.WriteField("seed", seedValue); err != nil {
		return nil, err
	}

	if err := writer.WriteField("allow_zip", "false"); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	data, err := c.request(ctx, "POST", "torrents/createtorrent", &body, writer.FormDataContentType())
	if err != nil {
		return nil, err
	}

	var result struct {
		TorrentID int  `json:"torrent_id"`
		Queued    bool `json:"queued"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resultado: %w", err)
	}

	// Força refresh da lista
	go c.refreshTorrentList()

	return c.GetTorrent(ctx, result.TorrentID)
}

// CheckCached verifica se hashes estão em cache (com batching)
func (c *OptimizedClient) CheckCached(ctx context.Context, hashes []string) (map[string]bool, error) {
	if len(hashes) == 0 {
		return map[string]bool{}, nil
	}

	// Divide em batches de 50 para evitar URLs muito longas
	batchSize := 50
	result := make(map[string]bool)
	var resultMu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)

	for i := 0; i < len(hashes); i += batchSize {
		end := i + batchSize
		if end > len(hashes) {
			end = len(hashes)
		}
		batch := hashes[i:end]

		g.Go(func() error {
			hashList := strings.Join(batch, ",")
			endpoint := fmt.Sprintf("torrents/checkcached?hash=%s&format=object", url.QueryEscape(hashList))

			data, err := c.request(gctx, "GET", endpoint, nil, "")
			if err != nil {
				return nil // Não falha tudo por um batch
			}

			batchResult := make(map[string]bool)
			if err := json.Unmarshal(data, &batchResult); err != nil {
				return nil
			}

			resultMu.Lock()
			for k, v := range batchResult {
				result[k] = v
			}
			resultMu.Unlock()

			return nil
		})
	}

	g.Wait()
	return result, nil
}

// GetDownloadLink obtém link de stream (com cache)
func (c *OptimizedClient) GetDownloadLink(ctx context.Context, torrentID, fileID int) (string, error) {
	cacheKey := fmt.Sprintf("link:%d:%d", torrentID, fileID)

	if cached, ok := c.linkCache.Get(cacheKey); ok {
		atomic.AddInt64(&c.cacheHits, 1)
		return cached.(string), nil
	}
	atomic.AddInt64(&c.cacheMisses, 1)

	endpoint := fmt.Sprintf("torrents/requestdl?token=%s&torrent_id=%d&file_id=%d", c.apiKey, torrentID, fileID)
	data, err := c.request(ctx, "GET", endpoint, nil, "")
	if err != nil {
		return "", err
	}

	var link string
	if err := json.Unmarshal(data, &link); err != nil {
		return "", fmt.Errorf("erro ao decodificar link: %w", err)
	}

	c.linkCache.Set(cacheKey, link)
	return link, nil
}

// DeleteTorrent remove um torrent
func (c *OptimizedClient) DeleteTorrent(ctx context.Context, torrentID int) error {
	body := fmt.Sprintf(`{"torrent_id": %d, "operation": "delete"}`, torrentID)
	_, err := c.request(ctx, "POST", "torrents/controltorrent", strings.NewReader(body), "application/json")

	if err == nil {
		// Invalida caches
		c.torrentCache.Clear()
		go c.refreshTorrentList()
	}

	return err
}

// GetUser retorna informações do usuário
func (c *OptimizedClient) GetUser(ctx context.Context) (*User, error) {
	data, err := c.request(ctx, "GET", "user/me", nil, "")
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("erro ao decodificar usuário: %w", err)
	}

	planNames := map[int]string{0: "Free", 1: "Essential", 2: "Pro", 3: "Standard"}
	user.PlanName = planNames[user.Plan]
	if user.PlanName == "" {
		user.PlanName = fmt.Sprintf("Plano %d", user.Plan)
	}

	return &user, nil
}

// GetVideoFiles retorna apenas arquivos de vídeo
func (c *OptimizedClient) GetVideoFiles(torrent *Torrent) []TorrentFile {
	var videos []TorrentFile
	videoExts := map[string]bool{
		".mkv": true, ".mp4": true, ".avi": true,
		".webm": true, ".mov": true, ".m4v": true,
	}

	for _, f := range torrent.Files {
		name := strings.ToLower(f.Name)
		for ext := range videoExts {
			if strings.HasSuffix(name, ext) {
				f.IsPlayable = true
				videos = append(videos, f)
				break
			}
		}
	}

	// Ordena por tamanho
	sort.Slice(videos, func(i, j int) bool {
		return videos[i].Size > videos[j].Size
	})

	return videos
}

// =============================================================================
// BUSCA PARALELA OTIMIZADA
// =============================================================================

// SearchAnimeTorrentsParallel busca em múltiplas fontes em paralelo
func (c *OptimizedClient) SearchAnimeTorrentsParallel(ctx context.Context, query string) ([]AnimeTorrent, error) {
	cacheKey := "search:" + strings.ToLower(strings.TrimSpace(query))

	// Verifica cache
	if cached, ok := c.searchCache.Get(cacheKey); ok {
		atomic.AddInt64(&c.cacheHits, 1)
		log.Printf("[TorBox] ⚡ Cache HIT: %s", query)
		return cached.([]AnimeTorrent), nil
	}
	atomic.AddInt64(&c.cacheMisses, 1)

	log.Printf("[TorBox] 🔍 Buscando: %s", query)
	startTime := time.Now()

	var allResults []AnimeTorrent
	var resultsMu sync.Mutex
	var seenHashes = make(map[string]bool)

	g, gctx := errgroup.WithContext(ctx)

	// Goroutine 1: Nyaa.si
	g.Go(func() error {
		results, err := c.searchNyaaOptimized(gctx, query)
		if err != nil {
			log.Printf("[Nyaa] Erro: %v", err)
			return nil
		}

		resultsMu.Lock()
		for _, r := range results {
			hash := strings.ToLower(r.Hash)
			if hash != "" && seenHashes[hash] {
				continue
			}
			seenHashes[hash] = true
			allResults = append(allResults, r)
		}
		resultsMu.Unlock()

		log.Printf("[Nyaa] ✅ %d resultados", len(results))
		return nil
	})

	// Goroutine 2: Nyaa.si BR (português)
	g.Go(func() error {
		brQuery := query + " portuguese"
		results, err := c.searchNyaaOptimized(gctx, brQuery)
		if err != nil {
			return nil
		}

		resultsMu.Lock()
		for _, r := range results {
			hash := strings.ToLower(r.Hash)
			if hash != "" && seenHashes[hash] {
				continue
			}
			seenHashes[hash] = true
			r.Source = "nyaa-br"
			allResults = append(allResults, r)
		}
		resultsMu.Unlock()

		return nil
	})

	g.Wait()

	// Ordena por seeds
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Seeds > allResults[j].Seeds
	})

	// Limita a 30 resultados
	if len(allResults) > 30 {
		allResults = allResults[:30]
	}

	log.Printf("[TorBox] 🏁 Busca concluída em %v - %d resultados", time.Since(startTime), len(allResults))

	// Salva no cache
	c.searchCache.Set(cacheKey, allResults)

	return allResults, nil
}

// searchNyaaOptimized busca otimizada no Nyaa com timeout curto
func (c *OptimizedClient) searchNyaaOptimized(ctx context.Context, query string) ([]AnimeTorrent, error) {
	searchURL := fmt.Sprintf("https://nyaa.si/?f=0&c=1_2&q=%s&s=seeders&o=desc", url.QueryEscape(query))

	// Timeout específico para scraping
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Lê apenas os primeiros 500KB (suficiente para ~50 resultados)
	limitedReader := io.LimitReader(resp.Body, 500*1024)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	return c.parseNyaaResultsOptimized(string(body))
}

// parseNyaaResultsOptimized parser otimizado com regex
func (c *OptimizedClient) parseNyaaResultsOptimized(html string) ([]AnimeTorrent, error) {
	var results []AnimeTorrent

	// Regex pré-compilados para performance
	rowRegex := regexp.MustCompile(`<tr class="(success|default|danger)">([\s\S]*?)</tr>`)
	titleRegex := regexp.MustCompile(`<a href="/view/\d+"[^>]*title="([^"]+)"`)
	magnetRegex := regexp.MustCompile(`href="(magnet:\?xt=urn:btih:[^"]+)"`)
	sizeRegex := regexp.MustCompile(`<td class="text-center">(\d+(?:\.\d+)?\s*[KMGT]iB)</td>`)
	seedsRegex := regexp.MustCompile(`<td class="text-center">(\d+)</td>\s*<td class="text-center">\d+</td>\s*</tr>`)

	rows := rowRegex.FindAllStringSubmatch(html, 50) // Máx 50 resultados

	for _, row := range rows {
		rowHTML := row[2]
		torrent := AnimeTorrent{Source: "nyaa.si"}

		// Título
		if match := titleRegex.FindStringSubmatch(rowHTML); len(match) > 1 {
			torrent.Title = c.decodeHTMLEntities(match[1])
		}

		// Magnet
		if match := magnetRegex.FindStringSubmatch(rowHTML); len(match) > 1 {
			torrent.Magnet = c.decodeHTMLEntities(match[1])

			// Extrai hash
			if idx := strings.Index(torrent.Magnet, "btih:"); idx >= 0 {
				hash := torrent.Magnet[idx+5:]
				if ampIdx := strings.Index(hash, "&"); ampIdx > 0 {
					hash = hash[:ampIdx]
				}
				torrent.Hash = strings.ToLower(hash)
			}
		}

		// Tamanho
		if match := sizeRegex.FindStringSubmatch(rowHTML); len(match) > 1 {
			torrent.Size = match[1]
		}

		// Seeds (último td antes do </tr>)
		if match := seedsRegex.FindStringSubmatch(rowHTML); len(match) > 1 {
			fmt.Sscanf(match[1], "%d", &torrent.Seeds)
		}

		// Qualidade
		torrent.Quality = detectQuality(torrent.Title)

		if torrent.Title != "" && torrent.Magnet != "" {
			results = append(results, torrent)
		}
	}

	return results, nil
}

func (c *OptimizedClient) decodeHTMLEntities(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	return s
}

// =============================================================================
// STREAMING INSTANTÂNEO OTIMIZADO
// =============================================================================

// GetInstantStreamOptimized busca e retorna stream de forma otimizada
func (c *OptimizedClient) GetInstantStreamOptimized(ctx context.Context, query string) (*InstantStreamResult, error) {
	log.Printf("[TorBox] 🎬 Stream para: %s", query)

	// Busca torrents em paralelo
	results, err := c.SearchAnimeTorrentsParallel(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("nenhum torrent encontrado: %s", query)
	}

	// Extrai hashes para verificar cache (máx 20)
	hashes := make([]string, 0, 20)
	hashToResult := make(map[string]*AnimeTorrent)
	for i := range results {
		if results[i].Hash != "" && len(hashes) < 20 {
			hashes = append(hashes, results[i].Hash)
			hashToResult[results[i].Hash] = &results[i]
		}
	}

	// Verifica cache no TorBox em paralelo
	cachedMap, err := c.CheckCached(ctx, hashes)
	if err != nil {
		log.Printf("[TorBox] Cache check error: %v", err)
		cachedMap = make(map[string]bool)
	}

	// Encontra melhor torrent
	qualityScore := map[string]int{"4K": 4, "1080p": 3, "720p": 2, "480p": 1, "Unknown": 0}

	var bestCached, bestUncached *AnimeTorrent

	for _, r := range results {
		hash := strings.ToLower(r.Hash)
		isCached := cachedMap[hash]

		if isCached {
			r.Cached = true
			if bestCached == nil || qualityScore[r.Quality] > qualityScore[bestCached.Quality] {
				bestCached = &r
			}
		} else {
			if bestUncached == nil || qualityScore[r.Quality] > qualityScore[bestUncached.Quality] {
				bestUncached = &r
			}
		}
	}

	// Prioriza torrents em cache
	var selected *AnimeTorrent
	if bestCached != nil {
		selected = bestCached
		log.Printf("[TorBox] ⚡ Cache encontrado: %s (%s)", selected.Title, selected.Quality)
	} else if bestUncached != nil {
		selected = bestUncached
		log.Printf("[TorBox] 📥 Sem cache, usando: %s (%s)", selected.Title, selected.Quality)
	} else {
		selected = &results[0]
	}

	// Adiciona torrent
	torrent, err := c.AddMagnet(ctx, selected.Magnet, false)
	if err != nil {
		return nil, fmt.Errorf("erro ao adicionar: %w", err)
	}

	// Se em cache, pequena espera para processar
	if selected.Cached && !torrent.DownloadFinish {
		time.Sleep(500 * time.Millisecond)
		torrent, _ = c.GetTorrent(ctx, torrent.ID)
	}

	// Pega arquivos de vídeo
	videos := c.GetVideoFiles(torrent)
	if len(videos) == 0 {
		return nil, fmt.Errorf("nenhum vídeo encontrado")
	}

	// Link do melhor arquivo
	bestFile := videos[0]
	streamURL, err := c.GetDownloadLink(ctx, torrent.ID, bestFile.ID)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter link: %w", err)
	}

	quality := detectQuality(bestFile.Name)
	if quality == "Unknown" {
		quality = selected.Quality
	}

	result := &InstantStreamResult{
		Success:   true,
		StreamURL: streamURL,
		TorrentID: torrent.ID,
		FileID:    bestFile.ID,
		FileName:  bestFile.Name,
		FileSize:  bestFile.Size,
		Quality:   quality,
		Cached:    selected.Cached,
		Title:     torrent.Name,
		Hash:      selected.Hash,
	}

	log.Printf("[TorBox] ✅ Stream pronto: %s", bestFile.Name)
	return result, nil
}

// =============================================================================
// MÉTRICAS E UTILITÁRIOS
// =============================================================================

// GetStats retorna estatísticas do cliente
func (c *OptimizedClient) GetStats() map[string]interface{} {
	cacheHitRate := float64(0)
	total := atomic.LoadInt64(&c.cacheHits) + atomic.LoadInt64(&c.cacheMisses)
	if total > 0 {
		cacheHitRate = float64(atomic.LoadInt64(&c.cacheHits)) / float64(total) * 100
	}

	c.torrentListMu.RLock()
	torrentCount := len(c.torrentList)
	lastUpdate := c.lastListUpdate
	c.torrentListMu.RUnlock()

	return map[string]interface{}{
		"requests_total": atomic.LoadInt64(&c.requestCount),
		"cache_hits":     atomic.LoadInt64(&c.cacheHits),
		"cache_misses":   atomic.LoadInt64(&c.cacheMisses),
		"cache_hit_rate": cacheHitRate,
		"error_count":    atomic.LoadInt64(&c.errorCount),
		"search_cache":   c.searchCache.Size(),
		"torrent_cache":  c.torrentCache.Size(),
		"link_cache":     c.linkCache.Size(),
		"torrent_list":   torrentCount,
		"last_update":    lastUpdate,
	}
}

// ClearAllCaches limpa todos os caches
func (c *OptimizedClient) ClearAllCaches() {
	c.searchCache.Clear()
	c.torrentCache.Clear()
	c.linkCache.Clear()
	log.Println("[TorBox] Todos os caches limpos")
}

// ForceRefresh força atualização da lista de torrents
func (c *OptimizedClient) ForceRefresh() {
	c.refreshTorrentList()
}
