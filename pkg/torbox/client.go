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
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	BaseURL = "https://api.torbox.app"
	Version = "v1"
)

// Cache local para evitar buscas repetidas
var (
	searchCache    = make(map[string]*cachedSearch)
	searchCacheMux sync.RWMutex
	streamCache    = make(map[string]*InstantStreamResult)
	streamCacheMux sync.RWMutex
	cacheDuration  = 30 * time.Minute
)

type cachedSearch struct {
	results   []AnimeTorrent
	timestamp time.Time
}

// Client cliente da API TorBox
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// New cria um novo cliente TorBox
func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ==================== TIPOS ====================

// TorrentFile arquivo dentro de um torrent
type TorrentFile struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	MimeType   string `json:"mimetype"`
	ShortName  string `json:"short_name"`
	IsPlayable bool   // Calculado
}

// Torrent informações de um torrent
type Torrent struct {
	ID             int           `json:"id"`
	Hash           string        `json:"hash"`
	Name           string        `json:"name"`
	Size           int64         `json:"size"`
	Progress       float64       `json:"progress"`
	Status         string        `json:"download_state"`
	Seeds          int           `json:"seeds"`
	Peers          int           `json:"peers"`
	Ratio          float64       `json:"ratio"`
	Speed          int64         `json:"download_speed"`
	UploadSpeed    int64         `json:"upload_speed"`
	ExpiresAt      string        `json:"expires_at"`
	Files          []TorrentFile `json:"files"`
	CreatedAt      string        `json:"created_at"`
	TotalDownload  int64         `json:"total_downloaded"`
	TotalUploaded  int64         `json:"total_uploaded"`
	Cached         bool          `json:"cached"`
	DownloadFinish bool          `json:"download_finished"`
	Active         bool          `json:"active"`
	Availability   float64       `json:"availability"`
}

// User informações do usuário
type User struct {
	ID            int     `json:"id"`
	Email         string  `json:"email"`
	Plan          int     `json:"plan"`
	PlanName      string  // Calculado
	TotalDownload int64   `json:"total_downloaded"`
	CreatedAt     string  `json:"created_at"`
	Premium       bool    `json:"is_subscribed"`
}

// AnimeTorrent resultado de busca de torrent
type AnimeTorrent struct {
	Title   string    `json:"title"`
	Magnet  string    `json:"magnet"`
	Hash    string    `json:"hash"`
	Size    string    `json:"size"`
	Seeds   int       `json:"seeds"`
	Leeches int       `json:"leeches"`
	Date    time.Time `json:"date"`
	Source  string    `json:"source"`
	Quality string    `json:"quality"`
	Cached  bool      `json:"cached"`
}

// InstantStreamResult resultado do streaming instantâneo
type InstantStreamResult struct {
	Success   bool         `json:"success"`
	StreamURL string       `json:"stream_url"`
	TorrentID int          `json:"torrent_id"`
	FileID    int          `json:"file_id"`
	FileName  string       `json:"file_name"`
	FileSize  int64        `json:"file_size"`
	Quality   string       `json:"quality"`
	Cached    bool         `json:"cached"`
	Title     string       `json:"title"`
	Hash      string       `json:"hash"`
	AllFiles  []FileStream `json:"all_files,omitempty"`
}

// FileStream arquivo disponível para stream
type FileStream struct {
	FileID    int    `json:"file_id"`
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
	StreamURL string `json:"stream_url"`
}

// APIResponse resposta genérica da API
type APIResponse struct {
	Success bool            `json:"success"`
	Detail  string          `json:"detail"`
	Error   string          `json:"error"`
	Data    json.RawMessage `json:"data"`
}

// ==================== MÉTODOS PRINCIPAIS ====================

// request faz uma requisição à API
func (c *Client) request(ctx context.Context, method, endpoint string, body io.Reader, contentType string) (json.RawMessage, error) {
	reqURL := fmt.Sprintf("%s/%s/api/%s", BaseURL, Version, endpoint)

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
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

	log.Printf("[TorBox] %s %s -> %d", method, endpoint, resp.StatusCode)

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w (body: %s)", err, string(respBody))
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("erro da API: %s - %s", apiResp.Error, apiResp.Detail)
	}

	return apiResp.Data, nil
}

// GetUser retorna informações do usuário
func (c *Client) GetUser(ctx context.Context) (*User, error) {
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

// GetTorrents lista todos os torrents do usuário
func (c *Client) GetTorrents(ctx context.Context) ([]Torrent, error) {
	data, err := c.request(ctx, "GET", "torrents/mylist", nil, "")
	if err != nil {
		return nil, err
	}

	var torrents []Torrent
	if err := json.Unmarshal(data, &torrents); err != nil {
		return nil, fmt.Errorf("erro ao decodificar torrents: %w", err)
	}

	// Marca arquivos de vídeo como playable
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

// GetTorrent retorna um torrent específico
func (c *Client) GetTorrent(ctx context.Context, torrentID int) (*Torrent, error) {
	endpoint := fmt.Sprintf("torrents/mylist?id=%d", torrentID)
	data, err := c.request(ctx, "GET", endpoint, nil, "")
	if err != nil {
		return nil, err
	}

	var torrent Torrent
	if err := json.Unmarshal(data, &torrent); err != nil {
		return nil, fmt.Errorf("erro ao decodificar torrent: %w", err)
	}

	return &torrent, nil
}

// AddMagnet adiciona um torrent via magnet link usando multipart/form-data
func (c *Client) AddMagnet(ctx context.Context, magnet string, seed bool) (*Torrent, error) {
	// Cria body multipart/form-data (formato correto da API TorBox)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	
	// Campo magnet
	if err := writer.WriteField("magnet", magnet); err != nil {
		return nil, fmt.Errorf("erro ao escrever campo magnet: %w", err)
	}
	
	// Campo seed (1=auto, 2=seed, 3=don't seed)
	seedValue := "1" // auto
	if seed {
		seedValue = "2" // seed
	}
	if err := writer.WriteField("seed", seedValue); err != nil {
		return nil, fmt.Errorf("erro ao escrever campo seed: %w", err)
	}
	
	// Campo allow_zip
	if err := writer.WriteField("allow_zip", "false"); err != nil {
		return nil, fmt.Errorf("erro ao escrever campo allow_zip: %w", err)
	}
	
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("erro ao fechar writer multipart: %w", err)
	}
	
	log.Printf("[TorBox] AddMagnet usando multipart/form-data")
	
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

	return c.GetTorrent(ctx, result.TorrentID)
}

// CheckCached verifica se hashes estão em cache
func (c *Client) CheckCached(ctx context.Context, hashes []string) (map[string]bool, error) {
	if len(hashes) == 0 {
		return map[string]bool{}, nil
	}

	hashList := strings.Join(hashes, ",")
	endpoint := fmt.Sprintf("torrents/checkcached?hash=%s&format=object", url.QueryEscape(hashList))

	data, err := c.request(ctx, "GET", endpoint, nil, "")
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar cache: %w", err)
	}

	return result, nil
}

// GetDownloadLink obtém link direto de download/streaming
func (c *Client) GetDownloadLink(ctx context.Context, torrentID, fileID int) (string, error) {
	endpoint := fmt.Sprintf("torrents/requestdl?token=%s&torrent_id=%d&file_id=%d", c.apiKey, torrentID, fileID)

	data, err := c.request(ctx, "GET", endpoint, nil, "")
	if err != nil {
		return "", err
	}

	var link string
	if err := json.Unmarshal(data, &link); err != nil {
		return "", fmt.Errorf("erro ao decodificar link: %w", err)
	}

	return link, nil
}

// DeleteTorrent remove um torrent
func (c *Client) DeleteTorrent(ctx context.Context, torrentID int) error {
	body := fmt.Sprintf(`{"torrent_id": %d, "operation": "delete"}`, torrentID)
	_, err := c.request(ctx, "POST", "torrents/controltorrent", strings.NewReader(body), "application/json")
	return err
}

// ==================== BUSCA DE ANIME ====================

// SearchAnimeTorrents busca torrents de anime no Nyaa.si
func (c *Client) SearchAnimeTorrents(ctx context.Context, query string) ([]AnimeTorrent, error) {
	// Cache key
	cacheKey := strings.ToLower(strings.TrimSpace(query))
	
	// Verifica cache primeiro
	searchCacheMux.RLock()
	if cached, ok := searchCache[cacheKey]; ok && time.Since(cached.timestamp) < cacheDuration {
		searchCacheMux.RUnlock()
		return cached.results, nil
	}
	searchCacheMux.RUnlock()

	// Busca no Nyaa.si
	results, err := searchNyaa(ctx, query)
	if err != nil {
		return nil, err
	}

	// Salva no cache
	searchCacheMux.Lock()
	searchCache[cacheKey] = &cachedSearch{
		results:   results,
		timestamp: time.Now(),
	}
	searchCacheMux.Unlock()

	return results, nil
}

// searchNyaa busca torrents no Nyaa.si
func searchNyaa(ctx context.Context, query string) ([]AnimeTorrent, error) {
	searchURL := fmt.Sprintf("https://nyaa.si/?f=0&c=1_2&q=%s&s=seeders&o=desc", url.QueryEscape(query))

	req, _ := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return parseNyaaResults(string(body))
}

// parseNyaaResults parseia os resultados do Nyaa
func parseNyaaResults(html string) ([]AnimeTorrent, error) {
	var results []AnimeTorrent

	rows := strings.Split(html, "<tr class=\"")
	
	for _, row := range rows[1:] {
		if !strings.Contains(row, "success") && !strings.Contains(row, "default") && !strings.Contains(row, "danger") {
			continue
		}

		torrent := AnimeTorrent{Source: "nyaa.si"}

		// Localiza o <td colspan="2"> que contém o título
		colspanIdx := strings.Index(row, `colspan="2"`)
		if colspanIdx == -1 {
			continue
		}
		
		// Pega o conteúdo a partir do colspan até </td>
		afterColspan := row[colspanIdx:]
		tdEndIdx := strings.Index(afterColspan, `</td>`)
		if tdEndIdx == -1 {
			continue
		}
		tdContent := afterColspan[:tdEndIdx]
		
		// Procura links /view/ (pula o primeiro que é de comentários)
		viewLinks := strings.Split(tdContent, `<a href="/view/`)
		
		for i := 1; i < len(viewLinks); i++ {
			link := viewLinks[i]
			
			// Pula se for link de comentários
			if strings.Contains(link, "#comments") {
				continue
			}
			
			// Extrai title="..."
			titleIdx := strings.Index(link, `title="`)
			if titleIdx >= 0 {
				titleStart := titleIdx + 7
				rest := link[titleStart:]
				titleEnd := strings.Index(rest, `"`)
				if titleEnd > 0 {
					torrent.Title = rest[:titleEnd]
					break
				}
			}
		}
		
		// Se ainda não tem título, tenta extrair do texto do link
		if torrent.Title == "" {
			// Fallback: pega texto entre > e </a>
			for i := 1; i < len(viewLinks); i++ {
				link := viewLinks[i]
				if strings.Contains(link, "#comments") {
					continue
				}
				closeIdx := strings.Index(link, ">")
				endIdx := strings.Index(link, "</a>")
				if closeIdx >= 0 && endIdx > closeIdx {
					torrent.Title = strings.TrimSpace(link[closeIdx+1 : endIdx])
					break
				}
			}
		}
		
		// Decodifica HTML entities no título
		if torrent.Title != "" {
			torrent.Title = strings.ReplaceAll(torrent.Title, "&amp;", "&")
			torrent.Title = strings.ReplaceAll(torrent.Title, "&lt;", "<")
			torrent.Title = strings.ReplaceAll(torrent.Title, "&gt;", ">")
			torrent.Title = strings.ReplaceAll(torrent.Title, "&quot;", "\"")
			torrent.Title = strings.ReplaceAll(torrent.Title, "&#39;", "'")
			torrent.Title = strings.ReplaceAll(torrent.Title, "&#x27;", "'")
		}

		// Extrai magnet
		if idx := strings.Index(row, `href="magnet:`); idx > 0 {
			end := strings.Index(row[idx+6:], `"`)
			if end > 0 {
				torrent.Magnet = row[idx+6 : idx+6+end]
				// Decodifica HTML entities no magnet
				torrent.Magnet = strings.ReplaceAll(torrent.Magnet, "&amp;", "&")
			}
		}

		// Extrai hash do magnet
		if torrent.Magnet != "" {
			if idx := strings.Index(torrent.Magnet, "btih:"); idx > 0 {
				hash := torrent.Magnet[idx+5:]
				if ampIdx := strings.Index(hash, "&"); ampIdx > 0 {
					hash = hash[:ampIdx]
				}
				torrent.Hash = strings.ToLower(hash)
			}
		}

		// Extrai tamanho - precisa pular o primeiro td que contém links de download
		// Procura por padrão de tamanho como "X GiB" ou "X MiB"
		sizePattern := `<td class="text-center">`
		searchPos := 0
		for i := 0; i < 3; i++ { // Procura nas primeiras 3 células text-center
			idx := strings.Index(row[searchPos:], sizePattern)
			if idx < 0 {
				break
			}
			actualIdx := searchPos + idx
			end := strings.Index(row[actualIdx+len(sizePattern):], `</td>`)
			if end > 0 {
				content := row[actualIdx+len(sizePattern) : actualIdx+len(sizePattern)+end]
				// Verifica se parece tamanho (contém GiB, MiB, KiB)
				if strings.Contains(content, "iB") && !strings.Contains(content, "<") {
					torrent.Size = strings.TrimSpace(content)
					break
				}
			}
			searchPos = actualIdx + len(sizePattern)
		}

		// Extrai seeds - busca pelo padrão com cor verde
		seedPattern := `<td class="text-center">`
		seedIdx := 0
		for i := 0; i < 6; i++ { // Procura nas células
			idx := strings.Index(row[seedIdx:], seedPattern)
			if idx < 0 {
				break
			}
			actualIdx := seedIdx + idx
			end := strings.Index(row[actualIdx+len(seedPattern):], `</td>`)
			if end > 0 {
				content := row[actualIdx+len(seedPattern) : actualIdx+len(seedPattern)+end]
				// Células após o tamanho contêm data e seeds
				// Seeds vem depois da data (que contém data-timestamp)
				if !strings.Contains(content, "<") && !strings.Contains(content, "iB") && !strings.Contains(content, "data-timestamp") {
					var seeds int
					if _, err := fmt.Sscanf(content, "%d", &seeds); err == nil && seeds > 0 {
						torrent.Seeds = seeds
						break
					}
				}
			}
			seedIdx = actualIdx + len(seedPattern)
		}

		// Detecta qualidade
		torrent.Quality = detectQuality(torrent.Title)

		if torrent.Title != "" && torrent.Magnet != "" {
			results = append(results, torrent)
		}
	}

	// Ordena por seeds
	sort.Slice(results, func(i, j int) bool {
		return results[i].Seeds > results[j].Seeds
	})

	// Limita a 10 resultados
	if len(results) > 10 {
		results = results[:10]
	}

	return results, nil
}

// detectQuality detecta a qualidade do vídeo a partir do nome
func detectQuality(name string) string {
	nameLower := strings.ToLower(name)
	if strings.Contains(nameLower, "2160p") || strings.Contains(nameLower, "4k") || strings.Contains(nameLower, "uhd") {
		return "4K"
	} else if strings.Contains(nameLower, "1080p") || strings.Contains(nameLower, "1080") {
		return "1080p"
	} else if strings.Contains(nameLower, "720p") || strings.Contains(nameLower, "720") {
		return "720p"
	} else if strings.Contains(nameLower, "480p") || strings.Contains(nameLower, "480") {
		return "480p"
	}
	return "Unknown"
}

// ==================== STREAMING INSTANTÂNEO ====================

// GetInstantStream busca anime e retorna link direto de streaming
func (c *Client) GetInstantStream(ctx context.Context, query string) (*InstantStreamResult, error) {
	log.Printf("[TorBox] Buscando stream instantâneo para: %s", query)

	// Verifica cache de streams primeiro
	cacheKey := strings.ToLower(strings.TrimSpace(query))
	streamCacheMux.RLock()
	if cached, ok := streamCache[cacheKey]; ok {
		streamCacheMux.RUnlock()
		log.Printf("[TorBox] Stream cache hit: %s", query)
		return cached, nil
	}
	streamCacheMux.RUnlock()

	// Busca torrents
	results, err := c.SearchAnimeTorrents(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("nenhum torrent encontrado para: %s", query)
	}

	// Extrai hashes para verificar cache (máx 10)
	hashes := make([]string, 0, 10)
	hashToResult := make(map[string]*AnimeTorrent)
	for i := range results {
		if results[i].Hash != "" && len(hashes) < 10 {
			hashes = append(hashes, results[i].Hash)
			hashToResult[results[i].Hash] = &results[i]
		}
	}

	// Verifica cache no TorBox
	cachedMap, err := c.CheckCached(ctx, hashes)
	if err != nil {
		log.Printf("[TorBox] Erro ao verificar cache: %v", err)
	}

	// Encontra o melhor torrent em cache
	var bestCached *AnimeTorrent
	qualityScore := map[string]int{"4K": 4, "1080p": 3, "720p": 2, "480p": 1, "Unknown": 0}

	for hash, isCached := range cachedMap {
		if isCached {
			result := hashToResult[hash]
			if result != nil {
				result.Cached = true
				if bestCached == nil || qualityScore[result.Quality] > qualityScore[bestCached.Quality] ||
					(qualityScore[result.Quality] == qualityScore[bestCached.Quality] && result.Seeds > bestCached.Seeds) {
					bestCached = result
				}
			}
		}
	}

	if bestCached == nil {
		// Sem cache, usa o melhor por qualidade/seeds
		sort.Slice(results, func(i, j int) bool {
			if qualityScore[results[i].Quality] != qualityScore[results[j].Quality] {
				return qualityScore[results[i].Quality] > qualityScore[results[j].Quality]
			}
			return results[i].Seeds > results[j].Seeds
		})
		bestCached = &results[0]
		log.Printf("[TorBox] Nenhum cache encontrado, usando: %s", bestCached.Title)
	} else {
		log.Printf("[TorBox] Cache encontrado! %s (Quality: %s)", bestCached.Title, bestCached.Quality)
	}

	// Adiciona o torrent ao TorBox
	torrent, err := c.AddMagnet(ctx, bestCached.Magnet, false)
	if err != nil {
		return nil, fmt.Errorf("erro ao adicionar torrent: %w", err)
	}

	// Se está em cache, espera 1 segundo
	if !torrent.DownloadFinish && bestCached.Cached {
		time.Sleep(1 * time.Second)
		torrent, err = c.GetTorrent(ctx, torrent.ID)
		if err != nil {
			return nil, err
		}
	}

	// Pega arquivos de vídeo
	videos := c.GetVideoFiles(torrent)
	if len(videos) == 0 {
		return nil, fmt.Errorf("nenhum arquivo de vídeo encontrado")
	}

	// Pega link do melhor arquivo
	bestFile := videos[0]
	streamURL, err := c.GetDownloadLink(ctx, torrent.ID, bestFile.ID)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter link de stream: %w", err)
	}

	// Detecta qualidade do nome do arquivo
	quality := detectQuality(bestFile.Name)
	if quality == "Unknown" {
		quality = bestCached.Quality
	}

	result := &InstantStreamResult{
		Success:   true,
		StreamURL: streamURL,
		TorrentID: torrent.ID,
		FileID:    bestFile.ID,
		FileName:  bestFile.Name,
		FileSize:  bestFile.Size,
		Quality:   quality,
		Cached:    bestCached.Cached,
		Title:     torrent.Name,
		Hash:      bestCached.Hash,
	}

	result.AllFiles = append(result.AllFiles, FileStream{
		FileID:    bestFile.ID,
		FileName:  bestFile.Name,
		FileSize:  bestFile.Size,
		StreamURL: streamURL,
	})

	// Salva no cache de streams
	streamCacheMux.Lock()
	streamCache[cacheKey] = result
	streamCacheMux.Unlock()

	log.Printf("[TorBox] ✅ Stream pronto: %s", streamURL)
	return result, nil
}

// GetVideoFiles retorna apenas arquivos de vídeo de um torrent
func (c *Client) GetVideoFiles(torrent *Torrent) []TorrentFile {
	var videos []TorrentFile
	videoExts := []string{".mkv", ".mp4", ".avi", ".webm", ".mov", ".m4v"}

	for _, f := range torrent.Files {
		name := strings.ToLower(f.Name)
		for _, ext := range videoExts {
			if strings.HasSuffix(name, ext) {
				f.IsPlayable = true
				videos = append(videos, f)
				break
			}
		}
	}

	// Ordena por tamanho (maior primeiro)
	sort.Slice(videos, func(i, j int) bool {
		return videos[i].Size > videos[j].Size
	})

	return videos
}

// ClearCache limpa todos os caches
func ClearCache() {
	searchCacheMux.Lock()
	searchCache = make(map[string]*cachedSearch)
	searchCacheMux.Unlock()

	streamCacheMux.Lock()
	streamCache = make(map[string]*InstantStreamResult)
	streamCacheMux.Unlock()

	log.Println("[TorBox] Cache limpo")
}
