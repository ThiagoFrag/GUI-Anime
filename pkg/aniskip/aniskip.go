package aniskip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AniSkip API - Retorna timestamps de abertura/encerramento
// Documentação: https://api.aniskip.com/api-docs
const (
	BaseURL = "https://api.aniskip.com/v2"
)

var (
	httpClient = &http.Client{Timeout: 5 * time.Second}
	cache      = make(map[string]*SkipTimes)
	cacheMutex sync.RWMutex
)

// SkipType representa o tipo de segmento a pular
type SkipType string

const (
	SkipTypeOpening SkipType = "op"
	SkipTypeEnding  SkipType = "ed"
	SkipTypeMixed   SkipType = "mixed-op"
	SkipTypeRecap   SkipType = "recap"
)

// SkipInterval representa um intervalo de tempo para pular
type SkipInterval struct {
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
}

// SkipResult representa um resultado da API
type SkipResult struct {
	Interval      SkipInterval `json:"interval"`
	SkipType      string       `json:"skipType"`
	SkipID        string       `json:"skipId"`
	EpisodeLength float64      `json:"episodeLength"`
}

// APIResponse representa a resposta completa da API
type APIResponse struct {
	Found      bool         `json:"found"`
	Results    []SkipResult `json:"results"`
	Message    string       `json:"message,omitempty"`
	StatusCode int          `json:"statusCode,omitempty"`
}

// SkipTimes contém os timestamps de abertura e encerramento
type SkipTimes struct {
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
// malID: ID do anime no MyAnimeList
// episodeNumber: Número do episódio
// episodeLength: Duração do episódio em segundos (opcional, use 0 para ignorar)
func GetSkipTimes(malID int, episodeNumber int, episodeLength float64) (*SkipTimes, error) {
	if malID <= 0 || episodeNumber <= 0 {
		return nil, fmt.Errorf("malID e episodeNumber devem ser maiores que 0")
	}

	// Verifica cache
	cacheKey := fmt.Sprintf("%d:%d", malID, episodeNumber)
	cacheMutex.RLock()
	if cached, ok := cache[cacheKey]; ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	// Monta a URL
	// Tipos: op, ed, mixed-op, mixed-ed, recap
	types := "op,ed,recap"
	url := fmt.Sprintf("%s/skip-times/%d/%d?types=%s", BaseURL, malID, episodeNumber, types)

	if episodeLength > 0 {
		url += fmt.Sprintf("&episodeLength=%.0f", episodeLength)
	}

	fmt.Printf("[AniSkip] Buscando skip times: MAL ID=%d, Ep=%d\n", malID, episodeNumber)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// Não encontrou - salva no cache como vazio
		empty := &SkipTimes{}
		cacheMutex.Lock()
		cache[cacheKey] = empty
		cacheMutex.Unlock()
		return empty, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP: %d", resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	// Processa os resultados
	skipTimes := &SkipTimes{}

	for _, result := range apiResp.Results {
		switch result.SkipType {
		case "op", "mixed-op":
			skipTimes.HasOpening = true
			skipTimes.OpeningStart = result.Interval.StartTime
			skipTimes.OpeningEnd = result.Interval.EndTime
		case "ed", "mixed-ed":
			skipTimes.HasEnding = true
			skipTimes.EndingStart = result.Interval.StartTime
			skipTimes.EndingEnd = result.Interval.EndTime
		case "recap":
			skipTimes.HasRecap = true
			skipTimes.RecapStart = result.Interval.StartTime
			skipTimes.RecapEnd = result.Interval.EndTime
		}

		if result.EpisodeLength > 0 {
			skipTimes.EpisodeLength = result.EpisodeLength
		}
	}

	// Salva no cache
	cacheMutex.Lock()
	cache[cacheKey] = skipTimes
	cacheMutex.Unlock()

	fmt.Printf("[AniSkip] Encontrado: Opening=%v (%.1f-%.1f), Ending=%v (%.1f-%.1f)\n",
		skipTimes.HasOpening, skipTimes.OpeningStart, skipTimes.OpeningEnd,
		skipTimes.HasEnding, skipTimes.EndingStart, skipTimes.EndingEnd)

	return skipTimes, nil
}

// GetSkipTimesAsync busca skip times de forma assíncrona
func GetSkipTimesAsync(malID int, episodeNumber int, episodeLength float64) <-chan *SkipTimes {
	ch := make(chan *SkipTimes, 1)

	go func() {
		skipTimes, err := GetSkipTimes(malID, episodeNumber, episodeLength)
		if err != nil {
			fmt.Printf("[AniSkip] Erro: %v\n", err)
			ch <- nil
		} else {
			ch <- skipTimes
		}
		close(ch)
	}()

	return ch
}

// ClearCache limpa o cache de skip times
func ClearCache() {
	cacheMutex.Lock()
	cache = make(map[string]*SkipTimes)
	cacheMutex.Unlock()
}
