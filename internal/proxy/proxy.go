// Package proxy fornece um servidor proxy para contornar CORS em streams de vídeo
package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Server representa o servidor de proxy de vídeo
type Server struct {
	server       *http.Server
	port         int
	currentVideo string
	mutex        sync.RWMutex
	client       *http.Client
}

// New cria um novo servidor de proxy
func New() *Server {
	return &Server{
		port: 0,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Start inicia o servidor de proxy em uma porta disponível
func (s *Server) Start() error {
	// Encontra uma porta disponível
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("erro ao criar listener: %w", err)
	}

	s.port = listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/video", s.handleVideo)
	mux.HandleFunc("/proxy", s.handleGenericProxy)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		fmt.Printf("[VideoProxy] Servidor iniciado na porta %d\n", s.port)
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[VideoProxy] Erro: %v\n", err)
		}
	}()

	return nil
}

// Stop para o servidor de proxy
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// Port retorna a porta em que o servidor está rodando
func (s *Server) Port() int {
	return s.port
}

// SetCurrentVideo define a URL do vídeo atual
func (s *Server) SetCurrentVideo(videoURL string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.currentVideo = videoURL
}

// GetCurrentVideo retorna a URL do vídeo atual
func (s *Server) GetCurrentVideo() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.currentVideo
}

// GetProxyURL retorna a URL do proxy para um vídeo
func (s *Server) GetProxyURL(videoURL string) string {
	if s.port == 0 {
		return videoURL
	}
	return fmt.Sprintf("http://127.0.0.1:%d/proxy?url=%s", s.port, url.QueryEscape(videoURL))
}

// handleVideo serve o vídeo atual via proxy
func (s *Server) handleVideo(w http.ResponseWriter, r *http.Request) {
	videoURL := s.GetCurrentVideo()
	if videoURL == "" {
		http.Error(w, "Nenhum vídeo configurado", http.StatusNotFound)
		return
	}

	s.proxyRequest(w, r, videoURL)
}

// handleGenericProxy faz proxy de qualquer URL passada como parâmetro
func (s *Server) handleGenericProxy(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, "URL não especificada", http.StatusBadRequest)
		return
	}

	s.proxyRequest(w, r, targetURL)
}

// handleHealth endpoint de health check
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// proxyRequest faz o proxy da requisição para a URL de destino
func (s *Server) proxyRequest(w http.ResponseWriter, r *http.Request, targetURL string) {
	// Cria requisição para o destino
	req, err := http.NewRequest(r.Method, targetURL, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao criar requisição: %v", err), http.StatusInternalServerError)
		return
	}

	// Copia headers relevantes
	for _, header := range []string{"Range", "Accept", "Accept-Encoding"} {
		if v := r.Header.Get(header); v != "" {
			req.Header.Set(header, v)
		}
	}

	// Headers para parecer um navegador
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", extractReferer(targetURL))

	// Faz a requisição
	resp, err := s.client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao buscar vídeo: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copia headers da resposta
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Adiciona headers CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")

	w.WriteHeader(resp.StatusCode)

	// Stream do conteúdo
	_, _ = io.Copy(w, resp.Body)
}

// extractReferer extrai o referer base da URL
func extractReferer(targetURL string) string {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s://%s/", parsed.Scheme, parsed.Host)
}

// ValidateURL verifica se uma URL de vídeo é válida com HEAD request
func ValidateURL(videoURL string) (bool, error) {
	if videoURL == "" {
		return false, fmt.Errorf("URL vazia")
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("muitos redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("HEAD", videoURL, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Considera válido se status é 2xx ou 3xx
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		// Verifica content-type para vídeo
		contentType := resp.Header.Get("Content-Type")
		isVideo := strings.Contains(contentType, "video") ||
			strings.Contains(contentType, "mpegurl") ||
			strings.Contains(contentType, "octet-stream") ||
			strings.HasSuffix(videoURL, ".mp4") ||
			strings.HasSuffix(videoURL, ".m3u8")

		return isVideo || contentType == "", nil
	}

	return false, fmt.Errorf("status %d", resp.StatusCode)
}
