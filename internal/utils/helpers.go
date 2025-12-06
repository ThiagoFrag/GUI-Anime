// Package utils fornece funções utilitárias comuns
package utils

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// NormalizeAnimeName normaliza o nome do anime para comparação
func NormalizeAnimeName(name string) string {
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

// StripTags remove tags HTML simples do texto
func StripTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}

// NormalizeURL normaliza hrefs relativos para absolutos usando base
func NormalizeURL(base, href string) string {
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

// ExtractEpisodeNumber tenta extrair o número do episódio de uma string
func ExtractEpisodeNumber(s string) int {
	// Padrões comuns: "Ep 1", "Episode 1", "1", "E01", "#1"
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)ep(?:isode|isódio)?\s*(\d+)`),
		regexp.MustCompile(`(?i)e(\d+)`),
		regexp.MustCompile(`#(\d+)`),
		regexp.MustCompile(`(\d+)$`),
	}

	for _, pattern := range patterns {
		if match := pattern.FindStringSubmatch(s); len(match) >= 2 {
			if num, err := strconv.Atoi(match[1]); err == nil {
				return num
			}
		}
	}

	return 0
}

// TruncateString trunca uma string para um tamanho máximo
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}

// ContainsAny verifica se a string contém algum dos substrings
func ContainsAny(s string, substrs ...string) bool {
	sLower := strings.ToLower(s)
	for _, substr := range substrs {
		if strings.Contains(sLower, strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

// SafeJSONMarshal faz marshal de JSON ignorando erros
func SafeJSONMarshal(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// SafeJSONUnmarshal faz unmarshal de JSON ignorando erros
func SafeJSONUnmarshal(data string, v interface{}) bool {
	if err := json.Unmarshal([]byte(data), v); err != nil {
		return false
	}
	return true
}

// CleanTitle limpa um título de anime para exibição
func CleanTitle(title string) string {
	// Remove tags HTML
	title = StripTags(title)

	// Remove caracteres de controle
	title = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, title)

	// Trim espaços
	title = strings.TrimSpace(title)

	return title
}

// IsVideoURL verifica se uma URL parece ser de um vídeo
func IsVideoURL(url string) bool {
	lowerURL := strings.ToLower(url)

	videoExtensions := []string{
		".mp4", ".m3u8", ".webm", ".mkv", ".avi", ".mov",
		".mp4?", ".m3u8?", ".webm?",
	}

	for _, ext := range videoExtensions {
		if strings.Contains(lowerURL, ext) {
			return true
		}
	}

	return false
}

// ExtractDomain extrai o domínio de uma URL
func ExtractDomain(urlStr string) string {
	// Remove protocolo
	if idx := strings.Index(urlStr, "://"); idx != -1 {
		urlStr = urlStr[idx+3:]
	}

	// Remove path
	if idx := strings.Index(urlStr, "/"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	// Remove porta
	if idx := strings.Index(urlStr, ":"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	return urlStr
}

// SlugToTitle converte um slug para título legível
func SlugToTitle(slug string) string {
	// Substitui hífens e underscores por espaços
	title := strings.ReplaceAll(slug, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Capitaliza cada palavra
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// UniqueStrings remove duplicatas de um slice de strings
func UniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))

	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}
