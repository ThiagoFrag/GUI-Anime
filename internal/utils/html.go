// Package utils - html.go contém funções para parsing de HTML
package utils

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// EpisodeData representa dados de um episódio extraído do HTML
type EpisodeData struct {
	Title  string
	URL    string
	Number int
	Season int
}

// ParseJSONLDScripts tenta extrair episódios de <script type="application/ld+json"> contendo Episode
func ParseJSONLDScripts(baseURL, html string) []EpisodeData {
	re := regexp.MustCompile(`(?is)<script[^>]+type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
	matches := re.FindAllStringSubmatch(html, -1)

	var out []EpisodeData

	for _, m := range matches {
		var js interface{}
		txt := strings.TrimSpace(m[1])
		if txt == "" {
			continue
		}

		if err := json.Unmarshal([]byte(txt), &js); err != nil {
			continue
		}

		// Função recursiva para encontrar episódios
		var find func(interface{})
		find = func(v interface{}) {
			switch vv := v.(type) {
			case map[string]interface{}:
				if t, ok := vv["@type"]; ok {
					if ts, ok2 := t.(string); ok2 && (strings.EqualFold(ts, "Episode") || strings.Contains(strings.ToLower(ts), "episode")) {
						title := ""
						if nm, ok := vv["name"].(string); ok {
							title = nm
						}
						url := ""
						if u, ok := vv["url"].(string); ok {
							url = NormalizeURL(baseURL, u)
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
						out = append(out, EpisodeData{Title: title, URL: url, Season: 1, Number: num})
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

// ParseJSArrays tenta extrair arrays JS com chave episodes
func ParseJSArrays(baseURL, html string) []EpisodeData {
	re := regexp.MustCompile(`(?is)(?:"episodes"\s*:\s*|var\s+episodes\s*=\s*)(\[.*?\])`)
	m := re.FindStringSubmatch(html)

	if len(m) < 2 {
		return nil
	}

	arrText := m[1]
	arrText = strings.ReplaceAll(arrText, "'", "\"")

	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(arrText), &items); err != nil {
		// Fallback: tenta extrair URLs via regex
		hrefRe := regexp.MustCompile(`(?i)href\s*[:=]\s*["']([^"']+)["']`)
		hm := hrefRe.FindAllStringSubmatch(arrText, -1)

		var out []EpisodeData
		for _, hh := range hm {
			u := NormalizeURL(baseURL, hh[1])
			out = append(out, EpisodeData{URL: u, Season: 1})
		}
		return out
	}

	var out []EpisodeData
	for _, it := range items {
		title := ""
		if t, ok := it["title"].(string); ok {
			title = t
		}

		url := ""
		if u, ok := it["url"].(string); ok {
			url = NormalizeURL(baseURL, u)
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

		out = append(out, EpisodeData{Title: title, URL: url, Season: 1, Number: num})
	}

	return out
}

// ParseDataAttributes procura por links em elementos com data-episode/data-url
func ParseDataAttributes(baseURL, html string) []EpisodeData {
	re := regexp.MustCompile(`(?i)data-(?:episode|url)=["']([^"']+)["']`)
	m := re.FindAllStringSubmatch(html, -1)

	var out []EpisodeData
	for _, mm := range m {
		u := NormalizeURL(baseURL, mm[1])
		out = append(out, EpisodeData{URL: u, Season: 1})
	}

	return out
}

// ParseAnimeFireEpisodes extrai episódios especificamente do AnimeFire
func ParseAnimeFireEpisodes(baseURL, html string) []EpisodeData {
	var episodes []EpisodeData

	// Extrai o slug base do anime
	baseSlug := extractAnimeFireSlug(baseURL)
	if baseSlug == "" {
		return nil
	}

	seen := make(map[int]bool)

	// Método 1: divNumEP (container de episódios do AnimeFire)
	divPattern := regexp.MustCompile(`class="divNumEP"[^>]*>.*?<a[^>]*href=["']([^"']+)["'][^>]*>`)
	divMatches := divPattern.FindAllStringSubmatch(html, -1)

	for _, m := range divMatches {
		if len(m) >= 2 {
			url := m[1]
			urlParts := strings.Split(strings.TrimSuffix(url, "/"), "/")
			if len(urlParts) > 0 {
				numStr := urlParts[len(urlParts)-1]
				num, err := strconv.Atoi(numStr)
				if err == nil && num > 0 && num < 2000 && !seen[num] {
					seen[num] = true
					title := SlugToTitle(baseSlug) + " - Episódio " + strconv.Itoa(num)
					episodes = append(episodes, EpisodeData{
						Title:  title,
						URL:    url,
						Season: 1,
						Number: num,
					})
				}
			}
		}
	}

	// Método 2: Padrão genérico
	if len(episodes) == 0 {
		genericPattern := regexp.MustCompile(`href=["'](https?://[^"']*animefire[^"']*/animes/[^/]+/(\d+))["']`)
		genericMatches := genericPattern.FindAllStringSubmatch(html, -1)

		for _, m := range genericMatches {
			if len(m) >= 3 {
				url := m[1]
				num, _ := strconv.Atoi(m[2])

				if num > 0 && num < 2000 && !seen[num] && strings.Contains(url, baseSlug) {
					seen[num] = true
					title := SlugToTitle(baseSlug) + " - Episódio " + strconv.Itoa(num)
					episodes = append(episodes, EpisodeData{
						Title:  title,
						URL:    url,
						Season: 1,
						Number: num,
					})
				}
			}
		}
	}

	return episodes
}

// extractAnimeFireSlug extrai o slug do anime de uma URL do AnimeFire
func extractAnimeFireSlug(url string) string {
	parts := strings.Split(url, "/")

	for _, p := range parts {
		if strings.Contains(p, "-todos-os-episodios") {
			return strings.Replace(p, "-todos-os-episodios", "", 1)
		}
	}

	// Fallback: último segmento
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		return strings.TrimSuffix(lastPart, "-todos-os-episodios")
	}

	return ""
}

// ExtractVideoFromHTML tenta extrair URL de vídeo de HTML de uma página
func ExtractVideoFromHTML(html string) string {
	// Padrões comuns de vídeo embutido
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`<source[^>]+src=["']([^"']+\.(?:mp4|m3u8|webm)[^"']*)["']`),
		regexp.MustCompile(`<video[^>]+src=["']([^"']+\.(?:mp4|m3u8|webm)[^"']*)["']`),
		regexp.MustCompile(`["']file["']\s*:\s*["']([^"']+\.(?:mp4|m3u8|webm)[^"']*)["']`),
		regexp.MustCompile(`["']src["']\s*:\s*["']([^"']+\.(?:mp4|m3u8|webm)[^"']*)["']`),
		regexp.MustCompile(`(https?://[^"'\s]+\.(?:mp4|m3u8|webm)[^"'\s]*)`),
	}

	for _, pattern := range patterns {
		if match := pattern.FindStringSubmatch(html); len(match) >= 2 {
			return match[1]
		}
	}

	return ""
}
