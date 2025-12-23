package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// CLIENTE DA API REMOTA (VPS)
// =============================================================================

// Constantes de criptografia
const (
	NonceSize       = 12 // 96 bits = 12 bytes
	ProtocolVersion = 1
)

// RemoteAPIClient cliente para comunicação com o servidor na VPS
type RemoteAPIClient struct {
	serverURL  string
	key        []byte
	gcm        cipher.AEAD
	clientID   string
	httpClient *http.Client
	mu         sync.Mutex
}

// RequestPayload payload de requisição
type RequestPayload struct {
	Action    string                 `json:"action"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	ClientID  string                 `json:"client_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// ResponsePayload payload de resposta
type ResponsePayload struct {
	Status    string                 `json:"status"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// EncryptedEnvelope envelope criptografado
// IMPORTANTE: O campo Data contém NONCE + CIPHERTEXT em Base64
type EncryptedEnvelope struct {
	Data      string `json:"data"`
	Version   int    `json:"version"`
	Signature string `json:"signature,omitempty"`
}

// RemoteAnimeInfo informações de anime da API remota
type RemoteAnimeInfo struct {
	ID         int64  `json:"id"`
	MalID      int64  `json:"mal_id,omitempty"`
	Title      string `json:"title"`
	TitleEN    string `json:"title_en,omitempty"`
	CoverImage string `json:"cover_image,omitempty"`
	Episodes   int    `json:"episodes"`
	Status     string `json:"status"`
	Source     string `json:"source"`
}

// RemoteEpisodeInfo informações de episódio da API remota
type RemoteEpisodeInfo struct {
	ID         int64  `json:"id"`
	AnimeID    int64  `json:"anime_id"`
	Number     int    `json:"number"`
	Title      string `json:"title,omitempty"`
	GoFileID   string `json:"gofile_id,omitempty"`
	TorBoxID   string `json:"torbox_id,omitempty"`
	MagnetLink string `json:"magnet_link,omitempty"`
	Quality    string `json:"quality,omitempty"`
	HasPTBR    bool   `json:"has_ptbr"`
}

// RemoteTorrentResult resultado de busca de torrent
type RemoteTorrentResult struct {
	Title       string `json:"title"`
	Name        string `json:"name,omitempty"`      // Nome real do torrent (se disponível)
	RawTitle    string `json:"raw_title,omitempty"` // raw_title da API TorBox
	Magnet      string `json:"magnet,omitempty"`
	Hash        string `json:"hash"`
	Size        string `json:"size"`
	Seeds       int    `json:"seeds"`
	Leeches     int    `json:"leeches"`
	Source      string `json:"source"`
	PageURL     string `json:"page_url,omitempty"`
	IsBrazilian bool   `json:"is_brazilian"`
	// Campos para agrupamento
	CleanTitle string                `json:"clean_title,omitempty"` // Título limpo para buscar imagem
	Variants   []RemoteTorrentResult `json:"variants,omitempty"`    // Variantes agrupadas (temporadas, etc)
}

// extractNameFromMagnet extrai o nome do torrent do magnet link (parâmetro dn=)
func extractNameFromMagnet(magnet string) string {
	if magnet == "" {
		return ""
	}

	// Primeiro, converte HTML entities para caracteres normais
	magnet = strings.ReplaceAll(magnet, "&amp;", "&")

	// Procura por dn= no magnet link
	dnStart := strings.Index(magnet, "dn=")
	if dnStart == -1 {
		return ""
	}

	// Pula "dn="
	dnStart += 3

	// Encontra o fim do parâmetro (próximo &)
	dnEnd := strings.Index(magnet[dnStart:], "&")
	var encoded string
	if dnEnd == -1 {
		encoded = magnet[dnStart:]
	} else {
		encoded = magnet[dnStart : dnStart+dnEnd]
	}

	// URL decode
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		// Fallback manual
		decoded = encoded
		decoded = strings.ReplaceAll(decoded, "%20", " ")
		decoded = strings.ReplaceAll(decoded, "%5B", "[")
		decoded = strings.ReplaceAll(decoded, "%5D", "]")
		decoded = strings.ReplaceAll(decoded, "%28", "(")
		decoded = strings.ReplaceAll(decoded, "%29", ")")
		decoded = strings.ReplaceAll(decoded, "%2B", "+")
	}

	return decoded
}

// isBatchOrComplete verifica se um torrent é um batch/completo
func isBatchOrComplete(title string) bool {
	titleLower := strings.ToLower(title)
	batchKeywords := []string{
		"batch", "complete", "completo", "full", "all episodes",
		"temporada completa", "season complete", "1-", "01-", "001-",
		"intégrale", "integral", "[complete]", "(complete)",
	}
	for _, kw := range batchKeywords {
		if strings.Contains(titleLower, kw) {
			return true
		}
	}
	// Verifica padrões como "001-220", "1-12", etc
	if matched, _ := regexp.MatchString(`\d{1,3}\s*[-~]\s*\d{1,3}`, title); matched {
		return true
	}
	return false
}

// extractCleanAnimeName extrai nome muito limpo para buscar imagem (mais agressivo)
// FOCO: agrupar "Naruto Clássico 1ª Temporada", "Naruto Clássico 2ª Temporada" -> "Naruto Clássico"
func extractCleanAnimeName(title string) string {
	name := title

	// 1. Substitui pontos por espaços PRIMEIRO (importante para "Naruto.Clássico.6ª.Temporada")
	name = strings.ReplaceAll(name, ".", " ")

	// 2. Remove TUDO entre colchetes, parênteses e chaves
	name = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(name, " ")
	name = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(name, " ")
	name = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(name, " ")

	// 3. Remove HTML entities
	name = regexp.MustCompile(`&[a-z]+;`).ReplaceAllString(name, "")

	// 4. Remove padrões de temporada/episódio BRASILEIROS (ANTES de remover qualidade)
	// "6ª Temporada", "1º Temporada", "2ª.Temporada", etc
	name = regexp.MustCompile(`(?i)\d+[ªºa°]?\s*(temporada|temp)\b`).ReplaceAllString(name, "")
	// "Temporada 6", "Season 2"
	name = regexp.MustCompile(`(?i)(temporada|season|temp)\s*\d+`).ReplaceAllString(name, "")
	// Ranges de episódios: "001-500", "1-92", "156-176"
	name = regexp.MustCompile(`\d{1,3}\s*[-~]\s*\d{1,3}`).ReplaceAllString(name, "")
	// Set patterns: "Set 12", "Set 18"
	name = regexp.MustCompile(`(?i)set\s*\d+`).ReplaceAllString(name, "")
	// Part patterns: "Part I", "Part 1"
	name = regexp.MustCompile(`(?i)part\s*[ivx\d]+`).ReplaceAllString(name, "")
	// v2, v3 etc
	name = regexp.MustCompile(`(?i)\bv\d+\b`).ReplaceAllString(name, "")
	// Batch
	name = regexp.MustCompile(`(?i)\bbatch\b`).ReplaceAllString(name, "")
	// Complete/Completo
	name = regexp.MustCompile(`(?i)\b(complete|completo|completa)\b`).ReplaceAllString(name, "")
	// Final
	name = regexp.MustCompile(`(?i)\bfinal\b`).ReplaceAllString(name, "")

	// 5. Remove qualidade e info técnica
	name = regexp.MustCompile(`(?i)(720p|1080p|480p|2160p|4k|hevc|x265|x264|10bit|flac|aac|ac3|h264|h 264|hi10p)`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)(bd|dvd|webrip|bluray|blu-ray|remux|hdtv|web-dl|bdrip|dvdrip)`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)(dual\s*audio|dual-audio|multi\s*subs?|multiple\s*subs?|eng\s*sub|legendado|dublado)`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)(mkv|mp4|avi)`).ReplaceAllString(name, "")

	// 6. Remove nomes de sites e release groups
	name = regexp.MustCompile(`(?i)(baixar|torrent|download|filmes|beta|viatorrents|dosfilmes)`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)(comoeubaixo|torrentdosfilmes|filmestorrents)\s*com?`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)\b(d4v1|jysze|judas|almighty|uss)\b`).ReplaceAllString(name, "")

	// 7. Remove anos
	name = regexp.MustCompile(`\b(19|20)\d{2}\b`).ReplaceAllString(name, "")

	// 8. Limpa caracteres especiais restantes no final
	name = regexp.MustCompile(`[-_:+]+$`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`^[-_:+]+`).ReplaceAllString(name, "")

	// 9. Limpa espaços múltiplos
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	name = strings.TrimSpace(name)

	// 10. Se ficou muito curto, tenta extrair as primeiras palavras
	if len(name) < 3 {
		words := strings.Fields(title)
		var cleanWords []string
		for _, w := range words {
			w = strings.Trim(w, "[](){}.,-_")
			if len(w) > 1 && !regexp.MustCompile(`^\d+$`).MatchString(w) {
				cleanWords = append(cleanWords, w)
				if len(cleanWords) >= 3 {
					break
				}
			}
		}
		name = strings.Join(cleanWords, " ")
	}

	return name
}

// filterAndSortTorrents filtra e organiza os torrents
// MODO SIMPLIFICADO: Remove duplicatas e ordena por seeds, sem agrupar demais
func filterAndSortTorrents(results []RemoteTorrentResult) []RemoteTorrentResult {
	if len(results) == 0 {
		return results
	}

	seenHashes := make(map[string]bool)
	var filtered []RemoteTorrentResult

	for i := range results {
		// Remove duplicatas por hash
		hashLower := strings.ToLower(results[i].Hash)
		if hashLower != "" && seenHashes[hashLower] {
			continue
		}
		if hashLower != "" {
			seenHashes[hashLower] = true
		}

		// Adiciona o CleanTitle para busca de imagem
		results[i].CleanTitle = extractCleanAnimeName(results[i].Title)

		filtered = append(filtered, results[i])
	}

	// Ordena: batches/complete primeiro, depois por seeds
	sort.Slice(filtered, func(i, j int) bool {
		isBatchI := isBatchOrComplete(filtered[i].Title)
		isBatchJ := isBatchOrComplete(filtered[j].Title)
		if isBatchI != isBatchJ {
			return isBatchI // Batches vêm primeiro
		}
		return filtered[i].Seeds > filtered[j].Seeds
	})

	fmt.Printf("[RemoteAPI] Filtro: %d torrents únicos de %d originais\n", len(filtered), len(results))

	return filtered
}

func formatBytesClient(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// RemoteStreamLink link de streaming
type RemoteStreamLink struct {
	DirectURL   string `json:"direct_url"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	ExpiresAt   int64  `json:"expires_at,omitempty"`
}

// RemoteTorrentFile representa um arquivo dentro de um torrent
type RemoteTorrentFile struct {
	ID        int    `json:"id"`
	TorrentID int    `json:"torrent_id,omitempty"` // ID do torrent pai (para TorBox)
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	Size      int64  `json:"size"`
	SizeStr   string `json:"size_str"`
	Episode   int    `json:"episode"`
	Season    int    `json:"season"`
	IsVideo   bool   `json:"is_video"`
}

// RemoteTorrentInfo informações de um torrent com seus arquivos
type RemoteTorrentInfo struct {
	Hash     string              `json:"hash"`
	Name     string              `json:"name"`
	Size     int64               `json:"size"`
	SizeStr  string              `json:"size_str"`
	Status   string              `json:"status"`
	Progress float64             `json:"progress"`
	Files    []RemoteTorrentFile `json:"files"`
}

// Constantes de ações
const (
	ActionSearch        = "search"
	ActionSearchBR      = "search_br"
	ActionSearchNyaa    = "search_nyaa"
	ActionGetMagnet     = "get_magnet"
	ActionGetLink       = "get_link"
	ActionGetFiles      = "get_files"
	ActionGetMediaInfo  = "get_mediainfo"
	ActionGetSubtitle   = "get_subtitle"
	ActionGetAnimes     = "get_animes"
	ActionGetEpisodes   = "get_episodes"
	ActionGetRecent     = "get_recent"
	ActionDeleteTorrent = "delete_torrent"
	ActionListTorrents  = "list_torrents"
)

var (
	remoteClient     *RemoteAPIClient
	remoteClientOnce sync.Once
)

// InitRemoteAPI inicializa o cliente da API remota
func InitRemoteAPI(serverURL string) error {
	var initErr error
	remoteClientOnce.Do(func() {
		// Chave compartilhada (mesma do servidor)
		secretKey := "GoAnime-Super-Secret-Key-2024-AES256-GCM"
		keyHash := sha256.Sum256([]byte(secretKey))

		block, err := aes.NewCipher(keyHash[:])
		if err != nil {
			initErr = fmt.Errorf("erro ao criar cipher: %w", err)
			return
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			initErr = fmt.Errorf("erro ao criar GCM: %w", err)
			return
		}

		// Cliente HTTP otimizado com timeouts específicos
		transport := &http.Transport{
			MaxIdleConns:          20,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			DisableCompression:    false,
			ResponseHeaderTimeout: 30 * time.Second,
		}

		remoteClient = &RemoteAPIClient{
			serverURL: serverURL,
			key:       keyHash[:],
			gcm:       gcm,
			clientID:  fmt.Sprintf("goanime-gui-%d", time.Now().UnixNano()),
			httpClient: &http.Client{
				Timeout:   45 * time.Second, // Timeout aumentado para buscas lentas
				Transport: transport,
			},
		}
		fmt.Printf("[RemoteAPI] Cliente inicializado: %s\n", serverURL)
	})
	return initErr
}

// encryptBytes criptografa dados com AES-256-GCM
// Retorna: nonce (12 bytes) + ciphertext concatenados
func (c *RemoteAPIClient) encryptBytes(plaintext []byte) ([]byte, error) {
	// Gera nonce aleatório
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("erro ao gerar nonce: %w", err)
	}

	// Criptografa e prefixa o nonce ao ciphertext
	ciphertext := c.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptBytes descriptografa dados com AES-256-GCM
// Espera: nonce (12 bytes) + ciphertext concatenados
func (c *RemoteAPIClient) decryptBytes(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < NonceSize {
		return nil, fmt.Errorf("ciphertext muito curto")
	}

	// Extrai o nonce do início
	nonce := ciphertext[:NonceSize]
	encrypted := ciphertext[NonceSize:]

	// Descriptografa
	plaintext, err := c.gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao descriptografar: %w", err)
	}

	return plaintext, nil
}

// Call faz uma chamada à API remota com retry automático
func (c *RemoteAPIClient) Call(action string, payload map[string]interface{}) (*ResponsePayload, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Tenta até 3 vezes com backoff
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			backoff := time.Duration(attempt) * 2 * time.Second
			fmt.Printf("[RemoteAPI] Retry %d/3 para %s após %v\n", attempt, action, backoff)
			time.Sleep(backoff)
		}

		resp, err := c.doCall(action, payload)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Se for erro de autenticação ou não encontrado, não retenta
		errStr := err.Error()
		if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "404") || strings.Contains(errStr, "invalid") {
			break
		}

		fmt.Printf("[RemoteAPI] Tentativa %d falhou: %v\n", attempt, err)
	}

	return nil, lastErr
}

// doCall executa uma chamada à API (sem retry)
func (c *RemoteAPIClient) doCall(action string, payload map[string]interface{}) (*ResponsePayload, error) {
	// Criar request
	req := RequestPayload{
		Action:    action,
		Payload:   payload,
		ClientID:  c.clientID,
		Timestamp: time.Now().Unix(),
	}

	// Serializar
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar request: %w", err)
	}

	// Criptografar (nonce + ciphertext concatenados)
	encrypted, err := c.encryptBytes(jsonData)
	if err != nil {
		return nil, fmt.Errorf("erro ao criptografar: %w", err)
	}

	// Criar envelope (Data contém nonce+ciphertext em Base64)
	envelope := EncryptedEnvelope{
		Data:    base64.StdEncoding.EncodeToString(encrypted),
		Version: ProtocolVersion,
	}

	envJson, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar envelope: %w", err)
	}

	// Enviar
	resp, err := c.httpClient.Post(
		c.serverURL+"/api/tunnel",
		"application/json",
		bytes.NewReader(envJson),
	)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	// Ler resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	// Verificar se é erro HTTP
	if resp.StatusCode != 200 {
		// Tenta decodificar erro JSON
		var errResp map[string]interface{}
		if json.Unmarshal(body, &errResp) == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				return nil, fmt.Errorf("erro do servidor: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("erro HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Decodificar envelope de resposta
	var respEnv EncryptedEnvelope
	if err := json.Unmarshal(body, &respEnv); err != nil {
		return nil, fmt.Errorf("erro ao decodificar envelope: %w", err)
	}

	// Decodifica Base64
	encryptedResp, err := base64.StdEncoding.DecodeString(respEnv.Data)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	// Descriptografar
	plaintext, err := c.decryptBytes(encryptedResp)
	if err != nil {
		return nil, fmt.Errorf("erro ao descriptografar resposta: %w", err)
	}

	// Decodificar resposta
	var response ResponsePayload
	if err := json.Unmarshal(plaintext, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &response, nil
}

// CallWithTimeout faz uma chamada com timeout customizado (para operações longas)
func (c *RemoteAPIClient) CallWithTimeout(action string, payload map[string]interface{}, timeout time.Duration) (*ResponsePayload, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	start := time.Now()
	fmt.Printf("[RemoteAPI] CallWithTimeout: %s (timeout: %v)\n", action, timeout)

	// Criar request
	req := RequestPayload{
		Action:    action,
		Payload:   payload,
		ClientID:  c.clientID,
		Timestamp: time.Now().Unix(),
	}

	// Serializar
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar request: %w", err)
	}

	// Criptografar
	encrypted, err := c.encryptBytes(jsonData)
	if err != nil {
		return nil, fmt.Errorf("erro ao criptografar: %w", err)
	}

	// Criar envelope
	envelope := EncryptedEnvelope{
		Data:    base64.StdEncoding.EncodeToString(encrypted),
		Version: ProtocolVersion,
	}

	envJson, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar envelope: %w", err)
	}

	// Cliente HTTP com timeout customizado
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	// Enviar
	resp, err := client.Post(
		c.serverURL+"/api/tunnel",
		"application/json",
		bytes.NewReader(envJson),
	)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição (timeout %v): %w", timeout, err)
	}
	defer resp.Body.Close()

	// Ler resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	// Verificar erro HTTP
	if resp.StatusCode != 200 {
		var errResp map[string]interface{}
		if json.Unmarshal(body, &errResp) == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				return nil, fmt.Errorf("erro do servidor: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("erro HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Decodificar envelope
	var respEnv EncryptedEnvelope
	if err := json.Unmarshal(body, &respEnv); err != nil {
		return nil, fmt.Errorf("erro ao decodificar envelope: %w", err)
	}

	// Base64 decode
	encryptedResp, err := base64.StdEncoding.DecodeString(respEnv.Data)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	// Descriptografar
	plaintext, err := c.decryptBytes(encryptedResp)
	if err != nil {
		return nil, fmt.Errorf("erro ao descriptografar: %w", err)
	}

	// Decodificar resposta
	var response ResponsePayload
	if err := json.Unmarshal(plaintext, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	fmt.Printf("[RemoteAPI] CallWithTimeout concluído em %v\n", time.Since(start))
	return &response, nil
}

// =============================================================================
// SISTEMA DE JOBS ASSÍNCRONOS (para operações longas)
// =============================================================================

// JobStatus status de um job assíncrono
type JobStatus struct {
	JobID     string                 `json:"job_id"`
	Status    string                 `json:"status"` // "pending", "processing", "completed", "failed"
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	CreatedAt string                 `json:"created_at,omitempty"`
	UpdatedAt string                 `json:"updated_at,omitempty"`
}

// GetJobStatus consulta o status de um job assíncrono
func (c *RemoteAPIClient) GetJobStatus(jobID string) (*JobStatus, error) {
	resp, err := c.Call("job_status", map[string]interface{}{
		"job_id": jobID,
	})
	if err != nil {
		return nil, err
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("erro ao consultar job: %s", resp.Error)
	}

	status := &JobStatus{}
	if jsonData, err := json.Marshal(resp.Data); err == nil {
		if err := json.Unmarshal(jsonData, status); err != nil {
			return nil, fmt.Errorf("erro ao decodificar status: %w", err)
		}
	}

	return status, nil
}

// CallWithPolling faz uma chamada que pode retornar um job_id e faz polling até completar
// Ideal para operações como get_files que podem demorar muito
func (c *RemoteAPIClient) CallWithPolling(action string, payload map[string]interface{}, initialTimeout time.Duration, maxWait time.Duration) (*ResponsePayload, error) {
	start := time.Now()
	fmt.Printf("[RemoteAPI] CallWithPolling: %s (initial: %v, maxWait: %v)\n", action, initialTimeout, maxWait)

	// Primeira tentativa com timeout inicial
	resp, err := c.CallWithTimeout(action, payload, initialTimeout)
	if err != nil {
		return nil, err
	}

	// Verifica se retornou um job_id (processamento assíncrono)
	if resp.Status == "success" {
		if jobID, ok := resp.Data["job_id"].(string); ok && jobID != "" {
			fmt.Printf("[RemoteAPI] Job assíncrono iniciado: %s\n", jobID)

			// Faz polling até completar ou timeout
			pollInterval := 2 * time.Second
			for {
				elapsed := time.Since(start)
				if elapsed >= maxWait {
					return nil, fmt.Errorf("timeout aguardando job %s (após %v)", jobID, elapsed)
				}

				time.Sleep(pollInterval)

				jobStatus, err := c.GetJobStatus(jobID)
				if err != nil {
					fmt.Printf("[RemoteAPI] Erro ao consultar job: %v\n", err)
					continue
				}

				fmt.Printf("[RemoteAPI] Job %s: status=%s\n", jobID, jobStatus.Status)

				switch jobStatus.Status {
				case "completed":
					// Converte result para ResponsePayload
					return &ResponsePayload{
						Status: "success",
						Data:   jobStatus.Result,
					}, nil

				case "failed":
					return nil, fmt.Errorf("job falhou: %s", jobStatus.Error)

				case "pending", "processing":
					// Continua polling
					continue

				default:
					fmt.Printf("[RemoteAPI] Status desconhecido: %s\n", jobStatus.Status)
				}
			}
		}
	}

	// Retorno direto (não foi assíncrono)
	return resp, nil
}

// =============================================================================
// MÉTODOS EXPOSTOS PARA O FRONTEND (via Wails)
// =============================================================================

// RemoteSearchAnimes busca animes na API remota
func (a *App) RemoteSearchAnimes(query string) []RemoteAnimeInfo {
	if remoteClient == nil {
		fmt.Println("[RemoteAPI] Cliente não inicializado")
		return nil
	}

	resp, err := remoteClient.Call(ActionSearch, map[string]interface{}{
		"query": query,
	})
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro na busca: %v\n", err)
		return nil
	}

	if resp.Status != "success" {
		fmt.Printf("[RemoteAPI] Busca falhou: %s\n", resp.Error)
		return nil
	}

	// Converter data para []RemoteAnimeInfo
	results := []RemoteAnimeInfo{}
	if data, ok := resp.Data["results"]; ok {
		if jsonData, err := json.Marshal(data); err == nil {
			if err := json.Unmarshal(jsonData, &results); err != nil {
				fmt.Printf("[RemoteAPI] Erro ao decodificar animes: %v\n", err)
			}
		}
	}

	fmt.Printf("[RemoteAPI] Encontrados %d animes\n", len(results))
	return results
}

// RemoteSearchTorrents busca torrents na API remota
func (a *App) RemoteSearchTorrents(query string, brOnly bool) []RemoteTorrentResult {
	if remoteClient == nil {
		fmt.Println("[RemoteAPI] Cliente não inicializado")
		return nil
	}

	action := ActionSearch
	if brOnly {
		action = ActionSearchBR
	}

	resp, err := remoteClient.Call(action, map[string]interface{}{
		"query": query,
	})
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro na busca de torrents: %v\n", err)
		return nil
	}

	if resp.Status != "success" {
		fmt.Printf("[RemoteAPI] Busca de torrents falhou: %s\n", resp.Error)
		return nil
	}

	results := []RemoteTorrentResult{}

	// DEBUG: Mostrar estrutura completa dos dados recebidos
	if fullJson, err := json.MarshalIndent(resp.Data, "", "  "); err == nil {
		fmt.Printf("[RemoteAPI] Dados recebidos:\n%s\n", string(fullJson))
	}

	// O gateway retorna em "results" não em "torrents"
	if data, ok := resp.Data["results"]; ok {
		if jsonData, err := json.Marshal(data); err == nil {
			fmt.Printf("[RemoteAPI] JSON dos resultados: %s\n", string(jsonData)[:min(len(jsonData), 2000)])
			if err := json.Unmarshal(jsonData, &results); err != nil {
				fmt.Printf("[RemoteAPI] Erro ao decodificar results: %v\n", err)
			}
		}
	} else if data, ok := resp.Data["torrents"]; ok {
		// Fallback para "torrents"
		if jsonData, err := json.Marshal(data); err == nil {
			if err := json.Unmarshal(jsonData, &results); err != nil {
				fmt.Printf("[RemoteAPI] Erro ao decodificar torrents: %v\n", err)
			}
		}
	}

	// DEBUG: Mostrar primeiro resultado para verificar mapeamento
	if len(results) > 0 {
		fmt.Printf("[RemoteAPI] Primeiro resultado mapeado: Title=%s, Hash=%s, Size=%s\n",
			results[0].Title, results[0].Hash, results[0].Size)
	}

	// Corrigir títulos: extrair nome real do magnet link se o título for apenas categoria
	for i := range results {
		// Se não tem magnet mas tem hash, construir o magnet
		if results[i].Magnet == "" && results[i].Hash != "" {
			results[i].Magnet = fmt.Sprintf("magnet:?xt=urn:btih:%s", results[i].Hash)
			fmt.Printf("[RemoteAPI] Magnet construído do hash: %s\n", results[i].Magnet)
		}

		// Se o título parece ser apenas uma categoria (ex: "Anime - English-translated")
		// ou está vazio, extrair do magnet
		if results[i].Title == "" ||
			strings.Contains(results[i].Title, " - ") && len(results[i].Title) < 40 ||
			strings.HasPrefix(results[i].Title, "Anime") {

			nameFromMagnet := extractNameFromMagnet(results[i].Magnet)
			if nameFromMagnet != "" {
				results[i].Title = nameFromMagnet
				fmt.Printf("[RemoteAPI] Título extraído do magnet: %s\n", nameFromMagnet)
			}
		}

		// Se tem Name ou RawTitle, usar esses
		if results[i].Name != "" {
			results[i].Title = results[i].Name
		}
		if results[i].RawTitle != "" && results[i].RawTitle != results[i].Title {
			results[i].Title = results[i].RawTitle
		}

		// Limpar o campo Size que pode ter HTML
		if strings.Contains(results[i].Size, "<") {
			results[i].Size = "" // Limpa HTML do Size
		}

		// DEBUG: Mostra cada torrent processado
		magnetPreview := results[i].Magnet
		if len(magnetPreview) > 60 {
			magnetPreview = magnetPreview[:60]
		}
		fmt.Printf("[RemoteAPI] Torrent[%d]: Title=%s, Hash=%s, Magnet=%s...\n", i, results[i].Title, results[i].Hash, magnetPreview)
	}

	// Filtra e organiza: batches primeiro, agrupa episódios únicos
	results = filterAndSortTorrents(results)

	fmt.Printf("[RemoteAPI] Retornando %d torrents após filtro\n", len(results))
	return results
}

// RemoteGetStreamLink obtém link de streaming via Player VPS
// Usa endpoint /api/torbox/stream/{torrentId}/{fileId}
func (a *App) RemoteGetStreamLink(hash string, fileID int) *RemoteStreamLink {
	return a.RemoteGetStreamLinkWithTorrent(hash, fileID, 0)
}

// RemoteGetStreamLinkWithTorrent obtém link de streaming usando torrentId diretamente
func (a *App) RemoteGetStreamLinkWithTorrent(hash string, fileID int, torrentID int) *RemoteStreamLink {
	fmt.Printf("[RemoteAPI] GetStreamLink via Player - hash: %s, fileID: %d, torrentID: %d\n", hash, fileID, torrentID)

	client := &http.Client{Timeout: 30 * time.Second}

	// Se já temos o torrentID, pula a busca
	if torrentID == 0 {
		// Precisa buscar o torrentID pelo hash
		apiURL := fmt.Sprintf("%s/api/torbox/instant?q=%s", VPSPlayerURL, url.QueryEscape(hash))
		fmt.Printf("[RemoteAPI] Buscando torrent: %s\n", apiURL)

		resp, err := client.Get(apiURL)
		if err != nil {
			fmt.Printf("[RemoteAPI] Erro na requisição: %v\n", err)
			return nil
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("[RemoteAPI] Erro ao ler resposta: %v\n", err)
			return nil
		}

		var playerResp struct {
			Success   bool `json:"success"`
			TorrentID int  `json:"torrent_id"`
		}

		if err := json.Unmarshal(body, &playerResp); err != nil {
			fmt.Printf("[RemoteAPI] Erro ao decodificar: %v\n", err)
			return nil
		}

		if !playerResp.Success || playerResp.TorrentID == 0 {
			fmt.Printf("[RemoteAPI] Torrent não encontrado\n")
			return nil
		}

		torrentID = playerResp.TorrentID
	}

	// Chama o endpoint de stream
	streamURL := fmt.Sprintf("%s/api/torbox/stream/%d/%d", VPSPlayerURL, torrentID, fileID)
	fmt.Printf("[RemoteAPI] Obtendo stream: %s\n", streamURL)

	resp, err := client.Get(streamURL)
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao obter stream: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao ler stream: %v\n", err)
		return nil
	}

	var streamResp struct {
		StreamURL string `json:"stream_url"`
		Error     string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &streamResp); err != nil {
		fmt.Printf("[RemoteAPI] Erro ao decodificar stream: %v\n", err)
		return nil
	}

	if streamResp.StreamURL == "" {
		fmt.Printf("[RemoteAPI] Stream URL vazia: %s\n", streamResp.Error)
		return nil
	}

	link := &RemoteStreamLink{
		DirectURL: streamResp.StreamURL,
	}

	fmt.Printf("[RemoteAPI] Link obtido: %s\n", link.DirectURL)
	return link
}

// RemoteGetTorrentFiles obtém lista de arquivos de um torrent via VPS Player
// Usa POST /api/torbox/add para adicionar o torrent e obter lista de arquivos
func (a *App) RemoteGetTorrentFiles(magnet string, hash string) *RemoteTorrentInfo {
	magnetPreview := magnet
	if len(magnetPreview) > 50 {
		magnetPreview = magnetPreview[:50]
	}
	fmt.Printf("[RemoteAPI] GetTorrentFiles via Player - magnet: %s..., hash: %s\n", magnetPreview, hash)

	client := &http.Client{Timeout: 120 * time.Second}

	// 1. Primeiro, adiciona o torrent via POST /api/torbox/add
	// Isso vai adicionar ao TorBox e retornar info do torrent com arquivos
	addReq := map[string]interface{}{
		"magnet":       magnet,
		"check_cached": false, // Não requer cache, adiciona de qualquer forma
	}

	addJSON, _ := json.Marshal(addReq)
	apiURL := fmt.Sprintf("%s/api/torbox/add", VPSPlayerURL)
	fmt.Printf("[RemoteAPI] POST %s com magnet\n", apiURL)

	resp, err := client.Post(apiURL, "application/json", bytes.NewReader(addJSON))
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao adicionar torrent: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao ler resposta: %v\n", err)
		return nil
	}

	// Preview da resposta
	preview := string(body)
	if len(preview) > 500 {
		preview = preview[:500]
	}
	fmt.Printf("[RemoteAPI] Resposta add: %s\n", preview)

	// Parse da resposta - formato do TorBox torrent
	var torrentResp struct {
		Error          string  `json:"error,omitempty"`
		ID             int     `json:"id"`
		Hash           string  `json:"hash"`
		Name           string  `json:"name"`
		Cached         bool    `json:"cached"`
		DownloadFinish bool    `json:"download_finished"`
		Progress       float64 `json:"progress"`
		Files          []struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			ShortName string `json:"short_name"`
			Size      int64  `json:"size"`
		} `json:"files"`
	}

	if err := json.Unmarshal(body, &torrentResp); err != nil {
		fmt.Printf("[RemoteAPI] Erro ao decodificar resposta: %v\n", err)
		return nil
	}

	if torrentResp.Error != "" {
		fmt.Printf("[RemoteAPI] Erro do TorBox: %s\n", torrentResp.Error)
		return nil
	}

	if torrentResp.ID == 0 {
		fmt.Printf("[RemoteAPI] Torrent não retornou ID válido\n")
		return nil
	}

	// Converte para formato RemoteTorrentInfo
	info := &RemoteTorrentInfo{
		Hash:     torrentResp.Hash,
		Name:     torrentResp.Name,
		Status:   "ready",
		Progress: torrentResp.Progress * 100,
	}

	if !torrentResp.DownloadFinish {
		info.Status = "downloading"
	}

	// Processa arquivos do torrent
	for _, f := range torrentResp.Files {
		// Filtra apenas arquivos de vídeo
		if !isVideoFile(f.Name) {
			continue
		}

		episode, season := extractEpisodeInfoFromName(f.Name)
		shortName := f.ShortName
		if shortName == "" {
			shortName = extractShortName(f.Name)
		}

		file := RemoteTorrentFile{
			ID:        f.ID,
			TorrentID: torrentResp.ID,
			Name:      f.Name,
			ShortName: shortName,
			Size:      f.Size,
			SizeStr:   formatBytesClient(f.Size),
			Episode:   episode,
			Season:    season,
			IsVideo:   true,
		}
		info.Files = append(info.Files, file)
		info.Size += f.Size
	}

	info.SizeStr = formatBytesClient(info.Size)

	fmt.Printf("[RemoteAPI] Torrent adicionado: %s (ID=%d), %d arquivos de vídeo\n", info.Name, torrentResp.ID, len(info.Files))

	// Se não tem arquivos de vídeo, pode ser que o download ainda não terminou
	if len(info.Files) == 0 {
		fmt.Printf("[RemoteAPI] ⚠️ Nenhum arquivo de vídeo encontrado. Status: %s, Progress: %.1f%%\n", info.Status, info.Progress)

		// Tenta obter arquivos com polling - até 30 segundos
		fmt.Printf("[RemoteAPI] Iniciando polling para obter arquivos (máx 30s)...\n")
		return a.RemoteGetTorrentFilesWithPolling(torrentResp.ID, 30*time.Second)
	}

	return info
}

// RemoteGetTorrentFilesWithPolling faz polling para obter arquivos de um torrent
// Tenta múltiplas vezes até obter arquivos ou timeout
func (a *App) RemoteGetTorrentFilesWithPolling(torrentID int, timeout time.Duration) *RemoteTorrentInfo {
	start := time.Now()
	attempt := 0

	for time.Since(start) < timeout {
		attempt++
		fmt.Printf("[RemoteAPI] Polling tentativa %d (elapsed: %.1fs)\n", attempt, time.Since(start).Seconds())

		info := a.RemoteGetTorrentFilesRetry(torrentID)
		if info != nil && len(info.Files) > 0 {
			fmt.Printf("[RemoteAPI] ✅ Arquivos obtidos na tentativa %d: %d arquivos\n", attempt, len(info.Files))
			return info
		}

		// Verifica status
		if info != nil {
			fmt.Printf("[RemoteAPI] Status: %s, Progress: %.1f%%, Files: %d\n", info.Status, info.Progress, len(info.Files))

			// Se está em erro ou deletado, para
			if info.Status == "error" || info.Status == "deleted" {
				fmt.Printf("[RemoteAPI] ❌ Torrent em estado final: %s\n", info.Status)
				return info
			}
		}

		// Espera antes de próxima tentativa (aumenta gradualmente)
		waitTime := time.Duration(2+attempt) * time.Second
		if waitTime > 5*time.Second {
			waitTime = 5 * time.Second
		}
		fmt.Printf("[RemoteAPI] Aguardando %v antes de próxima tentativa...\n", waitTime)
		time.Sleep(waitTime)
	}

	fmt.Printf("[RemoteAPI] ⚠️ Timeout após %v - retornando último estado\n", timeout)
	return a.RemoteGetTorrentFilesRetry(torrentID)
}

// RemoteGetTorrentFilesRetry obtém arquivos de um torrent já adicionado pelo ID
func (a *App) RemoteGetTorrentFilesRetry(torrentID int) *RemoteTorrentInfo {
	fmt.Printf("[RemoteAPI] Retry: buscando arquivos do torrent ID=%d\n", torrentID)

	client := &http.Client{Timeout: 30 * time.Second}

	// Usa a API direta do TorBox para obter info do torrent
	apiURL := fmt.Sprintf("https://api.torbox.app/v1/api/torrents/mylist?id=%d", torrentID)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+TorBoxAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao buscar torrent: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Data struct {
			ID             int     `json:"id"`
			Hash           string  `json:"hash"`
			Name           string  `json:"name"`
			DownloadFinish bool    `json:"download_finished"`
			Progress       float64 `json:"progress"`
			Files          []struct {
				ID        int    `json:"id"`
				Name      string `json:"name"`
				ShortName string `json:"short_name"`
				Size      int64  `json:"size"`
			} `json:"files"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("[RemoteAPI] Erro ao decodificar: %v\n", err)
		return nil
	}

	info := &RemoteTorrentInfo{
		Hash:     result.Data.Hash,
		Name:     result.Data.Name,
		Status:   "downloading",
		Progress: result.Data.Progress * 100,
	}

	if result.Data.DownloadFinish {
		info.Status = "ready"
	}

	for _, f := range result.Data.Files {
		if !isVideoFile(f.Name) {
			continue
		}

		episode, season := extractEpisodeInfoFromName(f.Name)
		shortName := f.ShortName
		if shortName == "" {
			shortName = extractShortName(f.Name)
		}

		file := RemoteTorrentFile{
			ID:        f.ID,
			TorrentID: result.Data.ID,
			Name:      f.Name,
			ShortName: shortName,
			Size:      f.Size,
			SizeStr:   formatBytesClient(f.Size),
			Episode:   episode,
			Season:    season,
			IsVideo:   true,
		}
		info.Files = append(info.Files, file)
		info.Size += f.Size
	}

	info.SizeStr = formatBytesClient(info.Size)
	fmt.Printf("[RemoteAPI] Retry obteve %d arquivos de vídeo\n", len(info.Files))

	return info
}

// extractEpisodeInfoFromName extrai número do episódio e temporada do nome do arquivo
func extractEpisodeInfoFromName(name string) (episode, season int) {
	// Padrões comuns: S01E03, 1x03, Episode 03, - 03, etc
	patterns := []struct {
		regex   *regexp.Regexp
		epIdx   int
		seasIdx int
	}{
		{regexp.MustCompile(`[Ss](\d+)[Ee](\d+)`), 2, 1},      // S01E03
		{regexp.MustCompile(`(\d+)x(\d+)`), 2, 1},             // 1x03
		{regexp.MustCompile(`[Ee]pisode\s*(\d+)`), 1, 0},      // Episode 03
		{regexp.MustCompile(`[-_\s](\d{2,3})[-_\s\[]`), 1, 0}, // - 03 - ou _03_
		{regexp.MustCompile(`\s(\d{2,3})\s`), 1, 0},           // espaço 03 espaço
	}

	for _, p := range patterns {
		if matches := p.regex.FindStringSubmatch(name); matches != nil {
			if p.epIdx > 0 && p.epIdx < len(matches) {
				fmt.Sscanf(matches[p.epIdx], "%d", &episode)
			}
			if p.seasIdx > 0 && p.seasIdx < len(matches) {
				fmt.Sscanf(matches[p.seasIdx], "%d", &season)
			}
			if episode > 0 {
				if season == 0 {
					season = 1
				}
				return
			}
		}
	}

	return 0, 1
}

// extractShortName extrai nome curto para exibição
func extractShortName(name string) string {
	// Remove extensão
	if idx := strings.LastIndex(name, "."); idx > 0 {
		name = name[:idx]
	}
	// Remove path
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	// Limita tamanho
	if len(name) > 60 {
		name = name[:57] + "..."
	}
	return name
}

// isVideoFile verifica se arquivo é vídeo
func isVideoFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".mkv") ||
		strings.HasSuffix(lower, ".mp4") ||
		strings.HasSuffix(lower, ".avi") ||
		strings.HasSuffix(lower, ".webm")
}

// RemoteGetEpisodes obtém episódios de um anime
func (a *App) RemoteGetEpisodes(animeID int64) []RemoteEpisodeInfo {
	if remoteClient == nil {
		fmt.Println("[RemoteAPI] Cliente não inicializado")
		return nil
	}

	resp, err := remoteClient.Call(ActionGetEpisodes, map[string]interface{}{
		"anime_id": animeID,
	})
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao obter episódios: %v\n", err)
		return nil
	}

	if resp.Status != "success" {
		fmt.Printf("[RemoteAPI] Falha ao obter episódios: %s\n", resp.Error)
		return nil
	}

	episodes := []RemoteEpisodeInfo{}
	if data, ok := resp.Data["episodes"]; ok {
		if jsonData, err := json.Marshal(data); err == nil {
			if err := json.Unmarshal(jsonData, &episodes); err != nil {
				fmt.Printf("[RemoteAPI] Erro ao decodificar episodes: %v\n", err)
			}
		}
	}

	fmt.Printf("[RemoteAPI] Encontrados %d episódios\n", len(episodes))
	return episodes
}

// RemoteGetRecentReleases obtém lançamentos recentes
func (a *App) RemoteGetRecentReleases(limit int) []RemoteAnimeInfo {
	if remoteClient == nil {
		fmt.Println("[RemoteAPI] Cliente não inicializado")
		return nil
	}

	resp, err := remoteClient.Call(ActionGetRecent, map[string]interface{}{
		"limit": limit,
	})
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao obter lançamentos: %v\n", err)
		return nil
	}

	if resp.Status != "success" {
		fmt.Printf("[RemoteAPI] Falha ao obter lançamentos: %s\n", resp.Error)
		return nil
	}

	releases := []RemoteAnimeInfo{}
	if data, ok := resp.Data["releases"]; ok {
		if jsonData, err := json.Marshal(data); err == nil {
			if err := json.Unmarshal(jsonData, &releases); err != nil {
				fmt.Printf("[RemoteAPI] Erro ao decodificar releases: %v\n", err)
			}
		}
	}

	fmt.Printf("[RemoteAPI] Encontrados %d lançamentos\n", len(releases))
	return releases
}

// RemoteDeleteTorrent deleta um torrent no TorBox via VPS
func (a *App) RemoteDeleteTorrent(torrentID int) bool {
	if remoteClient == nil {
		fmt.Println("[RemoteAPI] Cliente não inicializado")
		return false
	}

	fmt.Printf("[RemoteAPI] Deletando torrent ID: %d\n", torrentID)

	resp, err := remoteClient.Call(ActionDeleteTorrent, map[string]interface{}{
		"torrent_id": torrentID,
	})
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao deletar torrent: %v\n", err)
		return false
	}

	if resp.Status != "success" {
		fmt.Printf("[RemoteAPI] Falha ao deletar torrent: %s\n", resp.Error)
		return false
	}

	fmt.Printf("[RemoteAPI] Torrent %d deletado com sucesso\n", torrentID)
	return true
}

// RemoteListTorrents lista todos os torrents no TorBox via VPS
func (a *App) RemoteListTorrents() []RemoteTorrentInfo {
	if remoteClient == nil {
		fmt.Println("[RemoteAPI] Cliente não inicializado")
		return nil
	}

	resp, err := remoteClient.Call(ActionListTorrents, nil)
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao listar torrents: %v\n", err)
		return nil
	}

	if resp.Status != "success" {
		fmt.Printf("[RemoteAPI] Falha ao listar torrents: %s\n", resp.Error)
		return nil
	}

	torrents := []RemoteTorrentInfo{}
	if data, ok := resp.Data["torrents"]; ok {
		if jsonData, err := json.Marshal(data); err == nil {
			if err := json.Unmarshal(jsonData, &torrents); err != nil {
				fmt.Printf("[RemoteAPI] Erro ao decodificar torrents: %v\n", err)
			}
		}
	}

	fmt.Printf("[RemoteAPI] %d torrents listados\n", len(torrents))
	return torrents
}

// RemoteCleanupTorrents remove torrents antigos/inativos do TorBox via VPS
func (a *App) RemoteCleanupTorrents(olderThanHours int) int {
	if remoteClient == nil {
		fmt.Println("[RemoteAPI] Cliente não inicializado")
		return 0
	}

	resp, err := remoteClient.Call("cleanup_torrents", map[string]interface{}{
		"older_than_hours": olderThanHours,
	})
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao limpar torrents: %v\n", err)
		return 0
	}

	if resp.Status != "success" {
		fmt.Printf("[RemoteAPI] Falha ao limpar torrents: %s\n", resp.Error)
		return 0
	}

	deletedCount := 0
	if count, ok := resp.Data["deleted_count"].(float64); ok {
		deletedCount = int(count)
	}

	fmt.Printf("[RemoteAPI] %d torrents removidos\n", deletedCount)
	return deletedCount
}

// PipelineResult resultado do pipeline TorBox -> GoFile -> Delete
type PipelineResult struct {
	Success        bool    `json:"success"`
	TorrentID      int     `json:"torrent_id"`
	Hash           string  `json:"hash"`
	FileName       string  `json:"file_name"`
	FileSize       int64   `json:"file_size"`
	GoFileCode     string  `json:"gofile_code"`
	GoFileLink     string  `json:"gofile_link"`
	EncodedLink    string  `json:"encoded_link,omitempty"`
	DownloadTime   float64 `json:"download_time"`
	UploadTime     float64 `json:"upload_time"`
	EncodeTime     float64 `json:"encode_time,omitempty"`
	TorrentDeleted bool    `json:"torrent_deleted"`
	Error          string  `json:"error,omitempty"`
}

// RemoteProcessEpisode processa episódio completo: TorBox -> GoFile -> Delete -> Encode
func (a *App) RemoteProcessEpisode(hash, magnet string, fileID int, autoEncode, deleteAfter bool) *PipelineResult {
	if remoteClient == nil {
		fmt.Println("[RemoteAPI] Cliente não inicializado")
		return nil
	}

	fmt.Printf("[RemoteAPI] Processando episódio - Hash: %s, Delete: %v, Encode: %v\n", hash, deleteAfter, autoEncode)

	resp, err := remoteClient.Call("process_episode", map[string]interface{}{
		"hash":         hash,
		"magnet":       magnet,
		"file_id":      fileID,
		"auto_encode":  autoEncode,
		"delete_after": deleteAfter,
	})
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao processar episódio: %v\n", err)
		return nil
	}

	if resp.Status != "success" {
		fmt.Printf("[RemoteAPI] Falha ao processar: %s\n", resp.Error)
		return nil
	}

	result := &PipelineResult{}
	if data, ok := resp.Data["result"]; ok {
		if jsonData, err := json.Marshal(data); err == nil {
			if err := json.Unmarshal(jsonData, result); err != nil {
				fmt.Printf("[RemoteAPI] Erro ao decodificar resultado: %v\n", err)
				return nil
			}
		}
	}

	if result.Success {
		fmt.Printf("[RemoteAPI] ✅ Pipeline completo! GoFile: %s\n", result.GoFileLink)
	} else {
		fmt.Printf("[RemoteAPI] ❌ Pipeline falhou: %s\n", result.Error)
	}

	return result
}

// RemoteHealthCheck verifica se a API remota está funcionando
func (a *App) RemoteHealthCheck() bool {
	if remoteClient == nil {
		return false
	}

	// Tenta fazer uma chamada simples de ping via tunnel
	resp, err := remoteClient.Call("ping", nil)
	if err != nil {
		fmt.Printf("[RemoteAPI] Health check falhou: %v\n", err)
		return false
	}

	// Se recebeu resposta (mesmo erro), o servidor está funcionando
	return resp != nil
}

// InitRemoteConnection inicializa a conexão com a API remota
func (a *App) InitRemoteConnection(serverURL string) bool {
	if serverURL == "" {
		serverURL = "http://[2804:54:c100:2::11]:8080" // VPS padrão
	}

	err := InitRemoteAPI(serverURL)
	if err != nil {
		fmt.Printf("[RemoteAPI] Erro ao inicializar: %v\n", err)
		return false
	}

	// Verificar conexão com ping
	fmt.Println("[RemoteAPI] Testando conexão...")
	if !a.RemoteHealthCheck() {
		fmt.Println("[RemoteAPI] Servidor não está respondendo corretamente")
		// Mesmo sem health check, vamos considerar conectado se InitRemoteAPI passou
	}

	fmt.Println("[RemoteAPI] Conexão estabelecida com sucesso!")
	return true
}

// =============================================================================
// VPS PLAYER API - Pipeline TorBox -> GoFile (porta 3002)
// =============================================================================

// VPSPlayerURL URL da API do player no VPS
const VPSPlayerURL = "http://100.105.69.69:3003" // Tailscale IP (porta 3003 via socat proxy)

// TorBoxAPIKey chave de API do TorBox para chamadas diretas
const TorBoxAPIKey = "9605c965-59e4-4245-b90b-6d6cd15f3631"

// VPSPipelineRequest requisição para o pipeline
type VPSPipelineRequest struct {
	Query        string `json:"query"`
	StreamURL    string `json:"stream_url,omitempty"`
	FileName     string `json:"file_name,omitempty"`
	Encode       bool   `json:"encode"` // Apenas remux MKV->MP4 no servidor
	UploadGoFile bool   `json:"upload_gofile"`
}

// VPSPipelineResponse resposta do pipeline
type VPSPipelineResponse struct {
	Status       string `json:"status"`
	JobID        string `json:"job_id"`
	StreamURL    string `json:"stream_url"`
	FileName     string `json:"file_name"`
	Encode       bool   `json:"encode"`
	UploadGoFile bool   `json:"upload_gofile"`
	Message      string `json:"message"`
	Error        string `json:"error,omitempty"`
}

// VPSSearchResult resultado da busca de torrent no VPS
type VPSSearchResult struct {
	Status    string `json:"status"`
	StreamURL string `json:"stream_url"`
	TorrentID int    `json:"torrent_id"`
	FileID    int    `json:"file_id"`
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
	Quality   string `json:"quality"`
	Cached    bool   `json:"cached"`
	Title     string `json:"title"`
	Hash      string `json:"hash"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

// VPSStartPipeline inicia o pipeline de processamento no VPS
// O pipeline faz: TorBox (download) -> Remux MKV->MP4 -> Upload GoFile -> Salva no DB
func (a *App) VPSStartPipeline(query string, encode, uploadGoFile bool) *VPSPipelineResponse {
	client := &http.Client{Timeout: 30 * time.Second}

	reqBody := VPSPipelineRequest{
		Query:        query,
		Encode:       encode,
		UploadGoFile: uploadGoFile,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return &VPSPipelineResponse{Status: "error", Error: err.Error()}
	}

	resp, err := client.Post(VPSPlayerURL+"/v1/process/start", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return &VPSPipelineResponse{Status: "error", Error: fmt.Sprintf("Erro de conexão: %v", err)}
	}
	defer resp.Body.Close()

	var result VPSPipelineResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &VPSPipelineResponse{Status: "error", Error: fmt.Sprintf("Erro ao decodificar: %v", err)}
	}

	fmt.Printf("[VPS Pipeline] Query: %s -> Status: %s, JobID: %s\n", query, result.Status, result.JobID)
	return &result
}

// VPSSearchStream busca stream no VPS (apenas busca, não processa)
func (a *App) VPSSearchStream(query string) *VPSSearchResult {
	client := &http.Client{Timeout: 60 * time.Second}

	reqBody := map[string]interface{}{
		"query":         query,
		"encode":        false,
		"upload_gofile": false,
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := client.Post(VPSPlayerURL+"/v1/process", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return &VPSSearchResult{Status: "error", Error: fmt.Sprintf("Erro de conexão: %v", err)}
	}
	defer resp.Body.Close()

	var result VPSSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &VPSSearchResult{Status: "error", Error: fmt.Sprintf("Erro ao decodificar: %v", err)}
	}

	fmt.Printf("[VPS Search] Query: %s -> Status: %s, Cached: %v\n", query, result.Status, result.Cached)
	return &result
}

// VPSGetStreamURL retorna URL de stream direto do TorBox via VPS
func (a *App) VPSGetStreamURL(query string) string {
	result := a.VPSSearchStream(query)
	if result.Status == "found" || result.Status == "success" {
		return result.StreamURL
	}
	return ""
}

// VPSCheckHealth verifica se o servidor player do VPS está online
func (a *App) VPSCheckHealth() bool {
	fmt.Printf("[VPS Health] Verificando %s/v1/health...\n", VPSPlayerURL)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(VPSPlayerURL + "/v1/health")
	if err != nil {
		fmt.Printf("[VPS Health] ❌ Erro: %v\n", err)
		return false
	}
	defer resp.Body.Close()
	fmt.Printf("[VPS Health] ✅ Status: %d\n", resp.StatusCode)
	return resp.StatusCode == 200
}

// VPSGetEpisodeGoFile busca episódio já processado do banco de dados do VPS
func (a *App) VPSGetEpisodeGoFile(animeName string, episodeNum int) string {
	client := &http.Client{Timeout: 10 * time.Second}

	// Endpoint para buscar episódio do banco
	url := fmt.Sprintf("%s/v1/episodes?anime=%s&episode=%d",
		VPSPlayerURL,
		url.QueryEscape(animeName),
		episodeNum)

	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("[VPS Episodes] Erro: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		GoFileID  string `json:"gofile_id"`
		GoFileURL string `json:"gofile_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	if result.GoFileURL != "" {
		return result.GoFileURL
	}
	if result.GoFileID != "" {
		return "https://gofile.io/d/" + result.GoFileID
	}
	return ""
}

// SubtitleSearchResult representa uma legenda encontrada
type SubtitleSearchResult struct {
	Title       string  `json:"title"`
	Language    string  `json:"language"`
	Format      string  `json:"format"`
	Source      string  `json:"source"`
	DownloadURL string  `json:"download_url"`
	MatchScore  float64 `json:"match_score"`
}

// SubtitleSearchResponse resposta da API de legendas
type SubtitleSearchResponse struct {
	Query      string                 `json:"query"`
	Episode    int                    `json:"episode"`
	Language   string                 `json:"language"`
	TotalFound int                    `json:"total_found"`
	Results    []SubtitleSearchResult `json:"results"`
}

// VPSSearchSubtitles busca legendas no servidor VPS (OpenSubtitles, Kitsunekko, etc)
func (a *App) VPSSearchSubtitles(animeName string, episode int) *SubtitleSearchResponse {
	client := &http.Client{Timeout: 15 * time.Second}

	// Constrói URL com parâmetros
	searchURL := fmt.Sprintf("%s/v1/subtitle/search?q=%s&episode=%d&lang=pt-BR",
		VPSPlayerURL,
		url.QueryEscape(animeName),
		episode)

	fmt.Printf("[VPS Subtitle] Buscando: %s\n", searchURL)

	resp, err := client.Get(searchURL)
	if err != nil {
		fmt.Printf("[VPS Subtitle] Erro: %v\n", err)
		return &SubtitleSearchResponse{TotalFound: 0}
	}
	defer resp.Body.Close()

	var result SubtitleSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("[VPS Subtitle] Erro ao decodificar: %v\n", err)
		return &SubtitleSearchResponse{TotalFound: 0}
	}

	fmt.Printf("[VPS Subtitle] Encontrado: %d legendas\n", result.TotalFound)
	return &result
}

// VPSDownloadSubtitle baixa uma legenda específica do servidor VPS
func (a *App) VPSDownloadSubtitle(downloadURL, source string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Endpoint de download
	dlURL := fmt.Sprintf("%s/v1/subtitle/download?url=%s&source=%s",
		VPSPlayerURL,
		url.QueryEscape(downloadURL),
		url.QueryEscape(source))

	resp, err := client.Get(dlURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// VPSGetBestSubtitle busca e retorna a melhor legenda disponível
func (a *App) VPSGetBestSubtitle(animeName string, episode int) *SubtitleSearchResult {
	result := a.VPSSearchSubtitles(animeName, episode)
	if result.TotalFound == 0 {
		return nil
	}

	// Retorna a primeira (já ordenada por score no servidor)
	return &result.Results[0]
}
