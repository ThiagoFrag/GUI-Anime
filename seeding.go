package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// SeedingWorker gerencia o processo de semeamento comunitário
type SeedingWorker struct {
	app         *App
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	running     bool
	mu          sync.RWMutex
	currentJob  *SeedJob
	stats       SeedingStats
	tempDir     string
	ffmpegPath  string
	jobQueue    chan *SeedJob
	resultQueue chan *SeedResult
}

// SeedJob representa um trabalho de semeamento
type SeedJob struct {
	ID          string    `json:"id"`
	AnimeName   string    `json:"anime_name"`
	Episode     int       `json:"episode"`
	Quality     string    `json:"quality"`
	TorrentHash string    `json:"torrent_hash"`
	StreamURL   string    `json:"stream_url"`
	FileSize    int64     `json:"file_size"`
	Status      string    `json:"status"` // pending, downloading, encoding, uploading, completed, error
	Progress    float64   `json:"progress"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
}

// SeedResult resultado do processamento
type SeedResult struct {
	JobID     string `json:"job_id"`
	Success   bool   `json:"success"`
	GoFileURL string `json:"gofile_url,omitempty"`
	Error     string `json:"error,omitempty"`
	Duration  int64  `json:"duration_ms"`
	BytesUp   int64  `json:"bytes_uploaded"`
}

// SeedingStats estatísticas de semeamento
// NOTA: Os nomes JSON correspondem ao que o frontend espera
type SeedingStats struct {
	JobsCompleted      int    `json:"jobsCompleted"`          // Episódios processados
	Errors             int    `json:"errors"`                 // Número de erros (antes JobsFailed)
	TotalBytesUploaded int64  `json:"totalBytesUploaded"`     // Total de bytes enviados
	CurrentJob         string `json:"currentJob,omitempty"`   // Job atual em texto legível
	CurrentJobID       string `json:"currentJobId,omitempty"` // ID do job atual
	LastJobTime        int64  `json:"lastJobTime"`            // Tempo do último job em ms
	AverageJobTime     int64  `json:"averageJobTime"`         // Tempo médio por job em ms
	IsRunning          bool   `json:"isRunning"`              // Se está rodando
}

// NewSeedingWorker cria um novo worker de semeamento
func NewSeedingWorker(app *App) *SeedingWorker {
	return &SeedingWorker{
		app:         app,
		tempDir:     filepath.Join(os.TempDir(), "goanime_seeding"),
		ffmpegPath:  findFFmpegPath(),
		jobQueue:    make(chan *SeedJob, 10),
		resultQueue: make(chan *SeedResult, 10),
	}
}

// findFFmpegPath encontra o FFmpeg no sistema
func findFFmpegPath() string {
	// Tenta no PATH
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path
	}

	// Locais comuns no Windows
	if runtime.GOOS == "windows" {
		paths := []string{
			"C:\\ffmpeg\\bin\\ffmpeg.exe",
			"C:\\Program Files\\ffmpeg\\bin\\ffmpeg.exe",
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "WinGet", "Links", "ffmpeg.exe"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	return ""
}

// Start inicia o worker de semeamento
func (sw *SeedingWorker) Start() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.running {
		return nil
	}

	// Cria diretório temporário
	if err := os.MkdirAll(sw.tempDir, 0750); err != nil {
		return fmt.Errorf("erro ao criar diretório temp: %w", err)
	}

	sw.ctx, sw.cancel = context.WithCancel(context.Background())
	sw.running = true
	sw.stats.IsRunning = true

	// Worker principal
	sw.wg.Add(1)
	go sw.mainLoop()

	// Worker de resultados
	sw.wg.Add(1)
	go sw.resultLoop()

	log.Println("[Seeding] Worker iniciado")
	return nil
}

// Stop para o worker
func (sw *SeedingWorker) Stop() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if !sw.running {
		return
	}

	sw.cancel()
	sw.running = false
	sw.stats.IsRunning = false
	sw.wg.Wait()

	// Limpa diretório temporário
	os.RemoveAll(sw.tempDir)

	log.Println("[Seeding] Worker parado")
}

// IsRunning verifica se está rodando
func (sw *SeedingWorker) IsRunning() bool {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	return sw.running
}

// GetStats retorna estatísticas
func (sw *SeedingWorker) GetStats() SeedingStats {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	return sw.stats
}

// mainLoop loop principal que busca jobs da VPS
func (sw *SeedingWorker) mainLoop() {
	defer sw.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sw.ctx.Done():
			return
		case <-ticker.C:
			// Verifica se deve processar baseado nas configurações
			if !sw.shouldProcess() {
				continue
			}

			// Busca próximo job da VPS
			job, err := sw.fetchNextJob()
			if err != nil {
				log.Printf("[Seeding] Erro ao buscar job: %v", err)
				continue
			}

			if job == nil {
				continue // Sem jobs disponíveis
			}

			// Processa o job
			sw.processJob(job)
		}
	}
}

// shouldProcess verifica se deve processar baseado nas configurações
func (sw *SeedingWorker) shouldProcess() bool {
	settings := sw.app.User.Settings

	// Verifica schedule
	now := time.Now()
	hour := now.Hour()

	switch settings.SeedingSchedule {
	case "night":
		if hour < 0 || hour >= 6 {
			return false
		}
	case "idle":
		// TODO: Verificar se PC está ocioso
		// Por enquanto, assume que está sempre ok
	}

	// Verifica CPU
	// TODO: Verificar uso atual de CPU

	return true
}

// fetchNextJob busca próximo job da VPS
func (sw *SeedingWorker) fetchNextJob() (*SeedJob, error) {
	// Faz requisição para a VPS pedindo próximo arquivo pendente
	req, err := http.NewRequestWithContext(sw.ctx, "POST",
		"http://[2804:54:c100:2::11]:8080/api/tunnel", nil)
	if err != nil {
		return nil, err
	}

	// Monta payload para buscar job
	payload := map[string]interface{}{
		"action":      "claim_encode_job",
		"client_id":   sw.app.User.Username,
		"has_ffmpeg":  sw.ffmpegPath != "",
		"max_size_mb": 2000, // 2GB máximo
	}

	payloadBytes, _ := json.Marshal(payload)
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(string(payloadBytes)))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Silencia erro 401 (autenticação pendente) para não poluir logs
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return nil, nil // Ignora silenciosamente
		}
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result struct {
		Success bool     `json:"success"`
		Job     *SeedJob `json:"job,omitempty"`
		Message string   `json:"message,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, nil // Sem jobs
	}

	return result.Job, nil
}

// processJob processa um job completo
func (sw *SeedingWorker) processJob(job *SeedJob) {
	sw.mu.Lock()
	sw.currentJob = job
	sw.stats.CurrentJobID = job.ID
	sw.stats.CurrentJob = fmt.Sprintf("%s Ep %d", job.AnimeName, job.Episode)
	sw.mu.Unlock()

	startTime := time.Now()
	result := &SeedResult{JobID: job.ID}

	defer func() {
		result.Duration = time.Since(startTime).Milliseconds()
		sw.resultQueue <- result

		sw.mu.Lock()
		sw.currentJob = nil
		sw.stats.CurrentJobID = ""
		sw.stats.CurrentJob = ""
		if result.Success {
			sw.stats.JobsCompleted++
			sw.stats.TotalBytesUploaded += result.BytesUp
		} else {
			sw.stats.Errors++
		}
		sw.stats.LastJobTime = result.Duration
		sw.mu.Unlock()
	}()

	log.Printf("[Seeding] Processando: %s Ep %d", job.AnimeName, job.Episode)

	// 1. Download do arquivo
	job.Status = "downloading"
	localPath, err := sw.downloadFile(job)
	if err != nil {
		job.Status = "error"
		job.Error = fmt.Sprintf("Download falhou: %v", err)
		result.Error = job.Error
		return
	}
	defer os.Remove(localPath)

	// 2. Encode para MP4 (se necessário)
	outputPath := localPath
	if sw.needsEncode(localPath) && sw.ffmpegPath != "" {
		job.Status = "encoding"
		encodedPath, err := sw.encodeToMP4(localPath)
		if err != nil {
			log.Printf("[Seeding] Encode falhou: %v, usando original", err)
		} else {
			os.Remove(localPath)
			outputPath = encodedPath
		}
	}
	defer func() {
		if outputPath != localPath {
			os.Remove(outputPath)
		}
	}()

	// 3. Upload para GoFile
	job.Status = "uploading"
	gofileURL, uploadedBytes, err := sw.uploadToGoFile(outputPath)
	if err != nil {
		job.Status = "error"
		job.Error = fmt.Sprintf("Upload falhou: %v", err)
		result.Error = job.Error
		return
	}

	// 4. Notifica VPS do sucesso
	job.Status = "completed"
	result.Success = true
	result.GoFileURL = gofileURL
	result.BytesUp = uploadedBytes

	sw.notifyCompletion(job, gofileURL)
	log.Printf("[Seeding] ✅ Concluído: %s → %s", job.AnimeName, gofileURL)
}

// downloadFile baixa o arquivo
func (sw *SeedingWorker) downloadFile(job *SeedJob) (string, error) {
	if job.StreamURL == "" {
		return "", fmt.Errorf("URL de stream vazia")
	}

	// Nome do arquivo
	fileName := fmt.Sprintf("%s_E%02d.mkv", sanitizeFileName(job.AnimeName), job.Episode)
	localPath := filepath.Join(sw.tempDir, fileName)

	// HTTP request
	req, err := http.NewRequestWithContext(sw.ctx, "GET", job.StreamURL, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Cria arquivo
	out, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Copia com progresso
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(localPath)
		return "", err
	}

	log.Printf("[Seeding] Baixado: %d MB", written/(1024*1024))
	return localPath, nil
}

// needsEncode verifica se precisa encode
func (sw *SeedingWorker) needsEncode(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".mkv" || ext == ".avi" || ext == ".webm"
}

// encodeToMP4 converte para MP4
func (sw *SeedingWorker) encodeToMP4(inputPath string) (string, error) {
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputPath := filepath.Join(sw.tempDir, baseName+"_encoded.mp4")

	args := []string{
		"-i", inputPath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "192k",
		"-movflags", "+faststart",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(sw.ctx, sw.ffmpegPath, args...)
	_, err := cmd.CombinedOutput()

	if err != nil {
		// Se copy falhou, tenta transcoding
		args = []string{
			"-i", inputPath,
			"-c:v", "libx264",
			"-preset", "fast",
			"-crf", "23",
			"-c:a", "aac",
			"-b:a", "192k",
			"-movflags", "+faststart",
			"-y",
			outputPath,
		}
		cmd = exec.CommandContext(sw.ctx, sw.ffmpegPath, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("ffmpeg: %v - %s", err, string(output))
		}
	}

	return outputPath, nil
}

// uploadToGoFile faz upload para GoFile via API direta
func (sw *SeedingWorker) uploadToGoFile(filePath string) (string, int64, error) {
	// Faz upload direto para GoFile API

	// 1. Busca servidor disponível
	serverResp, err := http.Get("https://api.gofile.io/servers")
	if err != nil {
		return "", 0, fmt.Errorf("erro ao buscar servidor: %w", err)
	}
	defer serverResp.Body.Close()

	var serverData struct {
		Status string `json:"status"`
		Data   struct {
			Servers []struct {
				Name string `json:"name"`
				Zone string `json:"zone"`
			} `json:"servers"`
		} `json:"data"`
	}

	if err := json.NewDecoder(serverResp.Body).Decode(&serverData); err != nil {
		return "", 0, fmt.Errorf("erro ao decodificar servidor: %w", err)
	}

	if serverData.Status != "ok" || len(serverData.Data.Servers) == 0 {
		return "", 0, fmt.Errorf("sem servidores GoFile disponíveis")
	}

	server := serverData.Data.Servers[0].Name
	uploadURL := fmt.Sprintf("https://%s.gofile.io/contents/uploadfile", server)

	// 2. Abre arquivo para upload
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("erro ao abrir arquivo: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	fileSize := stat.Size()

	// 3. Cria multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", 0, err
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", 0, err
	}
	writer.Close()

	// 4. Faz upload
	req, err := http.NewRequestWithContext(sw.ctx, "POST", uploadURL, body)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Minute} // Timeout longo para uploads grandes
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("erro no upload: %w", err)
	}
	defer resp.Body.Close()

	// 5. Processa resposta
	var uploadResp struct {
		Status string `json:"status"`
		Data   struct {
			DownloadPage string `json:"downloadPage"`
			Code         string `json:"code"`
			FileID       string `json:"fileId"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", 0, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if uploadResp.Status != "ok" {
		return "", 0, fmt.Errorf("upload falhou: status %s", uploadResp.Status)
	}

	gofileURL := uploadResp.Data.DownloadPage
	if gofileURL == "" {
		gofileURL = fmt.Sprintf("https://gofile.io/d/%s", uploadResp.Data.Code)
	}

	log.Printf("[Seeding] Upload completo: %s (%d bytes)", gofileURL, fileSize)
	return gofileURL, fileSize, nil
}

// notifyCompletion notifica a VPS da conclusão
func (sw *SeedingWorker) notifyCompletion(job *SeedJob, gofileURL string) {
	payload := map[string]interface{}{
		"action":     "complete_encode_job",
		"job_id":     job.ID,
		"client_id":  sw.app.User.Username,
		"gofile_url": gofileURL,
	}

	payloadBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "http://[2804:54:c100:2::11]:8080/api/tunnel",
		strings.NewReader(string(payloadBytes)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Seeding] Erro ao notificar VPS: %v", err)
		return
	}
	resp.Body.Close()
}

// resultLoop processa resultados
func (sw *SeedingWorker) resultLoop() {
	defer sw.wg.Done()

	for {
		select {
		case <-sw.ctx.Done():
			return
		case result := <-sw.resultQueue:
			// Atualiza estatísticas do usuário
			if result.Success {
				sw.app.User.Settings.SeedingContributed += result.BytesUp
				// Salva configurações atualizadas
				sw.app.SaveSettings(sw.app.User.Settings)

				log.Printf("[Seeding] Contribuição total: %d MB",
					sw.app.User.Settings.SeedingContributed/(1024*1024))
			}
		}
	}
}

// sanitizeFileName remove caracteres inválidos
func sanitizeFileName(name string) string {
	invalid := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// ============================================
// MÉTODOS EXPOSTOS PARA O FRONTEND
// ============================================

// StartSeeding inicia o worker de semeamento
func (a *App) StartSeeding() error {
	if a.seedingWorker == nil {
		a.seedingWorker = NewSeedingWorker(a)
	}
	return a.seedingWorker.Start()
}

// StopSeeding para o worker de semeamento
func (a *App) StopSeeding() {
	if a.seedingWorker != nil {
		a.seedingWorker.Stop()
	}
}

// GetSeedingStats retorna estatísticas de semeamento
func (a *App) GetSeedingStats() SeedingStats {
	if a.seedingWorker == nil {
		return SeedingStats{}
	}
	return a.seedingWorker.GetStats()
}

// IsSeedingRunning verifica se o semeamento está ativo
func (a *App) IsSeedingRunning() bool {
	if a.seedingWorker == nil {
		return false
	}
	return a.seedingWorker.IsRunning()
}

// ToggleSeeding liga/desliga o semeamento
func (a *App) ToggleSeeding(enabled bool) error {
	if enabled {
		return a.StartSeeding()
	}
	a.StopSeeding()
	return nil
}
