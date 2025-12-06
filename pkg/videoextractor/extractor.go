package videoextractor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return nil // Segue redirects
	},
}

// ExtractVideoURL extrai a URL do vídeo de uma página do AnimeFire
func ExtractVideoURL(pageURL string) (string, error) {
	fmt.Printf("[ExtractVideoURL] Extraindo de: %s\n", pageURL)

	// Headers para parecer um navegador real
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Referer", "https://animefire.plus/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro ao buscar página: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	html := string(body)

	// MÉTODO 1: Procura data-video-src (AnimeFire usa isso) - pode ser página intermediária
	dataVideoPattern := regexp.MustCompile(`data-video-src=["']([^"']+)["']`)
	if matches := dataVideoPattern.FindStringSubmatch(html); len(matches) > 1 {
		videoURL := matches[1]
		fmt.Printf("[ExtractVideoURL] Encontrado data-video-src: %s\n", videoURL)

		// Se for uma página /video/, precisamos extrair o stream real dela
		if strings.Contains(videoURL, "/video/") {
			return extractStreamFromVideoPage(videoURL)
		}
		return decodeVideoURL(videoURL)
	}

	// MÉTODO 2: Procura URLs m3u8 diretamente (stream HLS)
	m3u8Pattern := regexp.MustCompile(`["'](https?://[^"'\s]+\.m3u8[^"'\s]*)["']`)
	if matches := m3u8Pattern.FindAllStringSubmatch(html, -1); len(matches) > 0 {
		for _, match := range matches {
			url := match[1]
			if !isAdURL(url) {
				fmt.Printf("[ExtractVideoURL] Stream m3u8: %s\n", url)
				return url, nil
			}
		}
	}

	// MÉTODO 3: Procura array de vídeos em JavaScript (AnimeFire)
	videoArrayPattern := regexp.MustCompile(`(?:video|sources|quality)\s*[=:]\s*\[([^\]]+)\]`)
	if matches := videoArrayPattern.FindStringSubmatch(html); len(matches) > 1 {
		urls := extractURLsFromArray(matches[1])
		if len(urls) > 0 {
			fmt.Printf("[ExtractVideoURL] Encontrado em array JS: %s\n", urls[0])
			return urls[0], nil
		}
	}

	// MÉTODO 4: Procura URLs diretas de vídeo (.mp4, .m3u8)
	directURLPattern := regexp.MustCompile(`["'](https?://[^"'\s]+\.(?:mp4|m3u8)[^"'\s]*)["']`)
	matches := directURLPattern.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		url := match[1]
		// Ignora URLs de propaganda
		if !isAdURL(url) {
			fmt.Printf("[ExtractVideoURL] URL direta: %s\n", url)
			return url, nil
		}
	}

	// MÉTODO 5: Procura player iframe src
	iframePattern := regexp.MustCompile(`<iframe[^>]+src=["']([^"']+(?:video|player|embed)[^"']*)["']`)
	if matches := iframePattern.FindStringSubmatch(html); len(matches) > 1 {
		iframeSrc := matches[1]
		fmt.Printf("[ExtractVideoURL] Encontrado iframe: %s\n", iframeSrc)
		// Tenta extrair do iframe
		return ExtractVideoURL(iframeSrc)
	}

	// MÉTODO 6: Procura base64 encoded video URLs (alguns sites usam)
	base64Pattern := regexp.MustCompile(`atob\(["']([A-Za-z0-9+/=]+)["']\)`)
	if matches := base64Pattern.FindAllStringSubmatch(html, -1); len(matches) > 0 {
		for _, match := range matches {
			decoded, err := base64.StdEncoding.DecodeString(match[1])
			if err == nil && strings.Contains(string(decoded), "http") {
				fmt.Printf("[ExtractVideoURL] URL base64: %s\n", string(decoded))
				return string(decoded), nil
			}
		}
	}

	// MÉTODO 7: Procura JSON com vídeo
	jsonPattern := regexp.MustCompile(`\{[^{}]*"(?:src|url|file|video)":\s*"(https?://[^"]+)"[^{}]*\}`)
	if matches := jsonPattern.FindStringSubmatch(html); len(matches) > 1 {
		fmt.Printf("[ExtractVideoURL] URL em JSON: %s\n", matches[1])
		return matches[1], nil
	}

	return "", fmt.Errorf("nenhuma URL de vídeo encontrada")
}

// extractStreamFromVideoPage busca o stream real da página de vídeo do AnimeFire
func extractStreamFromVideoPage(videoPageURL string) (string, error) {
	fmt.Printf("[extractStreamFromVideoPage] Acessando: %s\n", videoPageURL)

	req, err := http.NewRequest("GET", videoPageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Referer", "https://animefire.plus/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	content := string(body)
	fmt.Printf("[extractStreamFromVideoPage] Resposta (%d bytes): %s\n", len(content), content[:min(200, len(content))])

	// VERIFICAR SE É RESPOSTA JSON (API do AnimeFire)
	if strings.HasPrefix(strings.TrimSpace(content), "{") {
		// Estrutura: {"data":[{"src":"URL","label":"QUALITY"},...],...}
		type VideoSource struct {
			Src   string `json:"src"`
			Label string `json:"label"`
		}
		type VideoResponse struct {
			Data []VideoSource `json:"data"`
		}

		var response VideoResponse
		if err := json.Unmarshal(body, &response); err == nil && len(response.Data) > 0 {
			// Ordena por qualidade (maior primeiro)
			qualityOrder := map[string]int{
				"1080p": 1080,
				"720p":  720,
				"480p":  480,
				"360p":  360,
				"fhd":   1080,
				"hd":    720,
				"sd":    360,
			}

			var bestURL string
			var bestQuality int

			for _, source := range response.Data {
				url := strings.ReplaceAll(source.Src, "\\/", "/")
				label := strings.ToLower(source.Label)

				quality := 360 // Default
				if q, ok := qualityOrder[label]; ok {
					quality = q
				} else if strings.Contains(label, "1080") {
					quality = 1080
				} else if strings.Contains(label, "720") {
					quality = 720
				} else if strings.Contains(label, "480") {
					quality = 480
				}

				// Também verifica pela URL
				if strings.Contains(url, "/fhd/") || strings.Contains(url, "1080") {
					quality = max(quality, 1080)
				} else if strings.Contains(url, "/hd/") || strings.Contains(url, "720") {
					quality = max(quality, 720)
				} else if strings.Contains(url, "/sd/") || strings.Contains(url, "360") {
					quality = max(quality, 360)
				}

				if quality > bestQuality || bestURL == "" {
					bestURL = url
					bestQuality = quality
				}
			}

			if bestURL != "" {
				fmt.Printf("[extractStreamFromVideoPage] Stream encontrado (JSON API): %s (%dp)\n", bestURL, bestQuality)
				return bestURL, nil
			}
		}

		// Fallback: regex para encontrar src com maior qualidade
		srcPattern := regexp.MustCompile(`"src"\s*:\s*"([^"]+)"`)
		matches := srcPattern.FindAllStringSubmatch(content, -1)

		var bestURL string
		for _, match := range matches {
			url := strings.ReplaceAll(match[1], "\\/", "/")

			// Prioriza HD sobre SD
			if strings.Contains(url, "/hd/") || strings.Contains(url, "/fhd/") {
				bestURL = url
				break
			}
			if bestURL == "" {
				bestURL = url
			}
		}

		if bestURL != "" {
			fmt.Printf("[extractStreamFromVideoPage] Stream encontrado (regex fallback): %s\n", bestURL)
			return bestURL, nil
		}
	}

	// Procura stream HLS (.m3u8) no HTML
	m3u8Pattern := regexp.MustCompile(`["'](https?://[^"'\s]+\.m3u8[^"'\s]*)["']`)
	if matches := m3u8Pattern.FindAllStringSubmatch(content, -1); len(matches) > 0 {
		for _, match := range matches {
			url := match[1]
			if !isAdURL(url) {
				fmt.Printf("[extractStreamFromVideoPage] Stream m3u8: %s\n", url)
				return url, nil
			}
		}
	}

	// Procura MP4 direto
	mp4Pattern := regexp.MustCompile(`["'](https?://[^"'\s]+\.mp4[^"'\s]*)["']`)
	if matches := mp4Pattern.FindAllStringSubmatch(content, -1); len(matches) > 0 {
		for _, match := range matches {
			url := strings.ReplaceAll(match[1], "\\/", "/")
			if !isAdURL(url) {
				fmt.Printf("[extractStreamFromVideoPage] Stream mp4: %s\n", url)
				return url, nil
			}
		}
	}

	// Procura array de fontes de vídeo
	sourcesPattern := regexp.MustCompile(`(?:sources|videos?)\s*[=:]\s*\[([^\]]+)\]`)
	if matches := sourcesPattern.FindStringSubmatch(content); len(matches) > 1 {
		urls := extractURLsFromArray(matches[1])
		for _, url := range urls {
			if strings.Contains(url, ".m3u8") || strings.Contains(url, ".mp4") {
				fmt.Printf("[extractStreamFromVideoPage] Stream do array: %s\n", url)
				return url, nil
			}
		}
	}

	// Fallback: retorna a própria URL da página de vídeo
	fmt.Printf("[extractStreamFromVideoPage] Fallback: usando URL da página\n")
	return videoPageURL, nil
}

// min retorna o menor de dois inteiros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// decodeVideoURL decodifica URLs que podem estar encoded
func decodeVideoURL(url string) (string, error) {
	// Tenta decodificar base64 se necessário
	if !strings.HasPrefix(url, "http") {
		decoded, err := base64.StdEncoding.DecodeString(url)
		if err == nil && strings.HasPrefix(string(decoded), "http") {
			return string(decoded), nil
		}
	}
	return url, nil
}

// extractURLsFromArray extrai URLs de um array JavaScript
func extractURLsFromArray(arrayContent string) []string {
	var urls []string

	// Tenta parsear como JSON array
	var jsonArray []interface{}
	if err := json.Unmarshal([]byte("["+arrayContent+"]"), &jsonArray); err == nil {
		for _, item := range jsonArray {
			switch v := item.(type) {
			case string:
				if strings.HasPrefix(v, "http") {
					urls = append(urls, v)
				}
			case map[string]interface{}:
				for _, val := range v {
					if s, ok := val.(string); ok && strings.HasPrefix(s, "http") {
						urls = append(urls, s)
					}
				}
			}
		}
	}

	// Fallback: regex para URLs
	if len(urls) == 0 {
		urlPattern := regexp.MustCompile(`["'](https?://[^"']+)["']`)
		matches := urlPattern.FindAllStringSubmatch(arrayContent, -1)
		for _, m := range matches {
			urls = append(urls, m[1])
		}
	}

	return urls
}

// isAdURL verifica se a URL parece ser de propaganda
func isAdURL(url string) bool {
	adPatterns := []string{
		"googleads", "doubleclick", "adservice",
		"ad.", "ads.", "/ad/", "/ads/",
		"banner", "popup", "tracking",
		"analytics", "pixel",
	}
	lowerURL := strings.ToLower(url)
	for _, pattern := range adPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}
	return false
}
