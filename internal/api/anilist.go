// Package api - anilist.go fornece integração com a API do AniList
package api

import (
	"fmt"
	"sync"

	"GoAnimeGUI/internal/cache"
	"GoAnimeGUI/pkg/anilist"
)

// AniListAnime representa um anime com dados do AniList
type AniListAnime struct {
	ID           int      `json:"id"`
	MalID        int      `json:"malId"`
	Title        string   `json:"title"`
	TitleEnglish string   `json:"titleEnglish"`
	TitleNative  string   `json:"titleNative"`
	Description  string   `json:"description"`
	Image        string   `json:"image"`
	Banner       string   `json:"banner"`
	Color        string   `json:"color"`
	Genres       []string `json:"genres"`
	Episodes     int      `json:"episodes"`
	Duration     int      `json:"duration"`
	Status       string   `json:"status"`
	Season       string   `json:"season"`
	Year         int      `json:"year"`
	Score        int      `json:"score"`
	Popularity   int      `json:"popularity"`
	Studio       string   `json:"studio"`
	TrailerURL   string   `json:"trailerUrl"`
	IsAiring     bool     `json:"isAiring"`
	NextEpisode  int      `json:"nextEpisode"`
}

// AniListService gerencia operações com a API do AniList
type AniListService struct {
	cache         *cache.Cache
	trendingCache []*AniListAnime
	mutex         sync.RWMutex
}

// NewAniListService cria um novo serviço AniList
func NewAniListService() *AniListService {
	return &AniListService{
		cache:         cache.New(),
		trendingCache: nil,
	}
}

// GetTrending retorna animes em alta
func (s *AniListService) GetTrending(limit int) ([]*AniListAnime, error) {
	if limit <= 0 {
		limit = 15
	}

	// Verifica cache pré-carregado
	s.mutex.RLock()
	if len(s.trendingCache) > 0 {
		cached := s.trendingCache
		s.mutex.RUnlock()
		fmt.Println("[AniList] Retornando cache pré-carregado")
		if len(cached) > limit {
			return cached[:limit], nil
		}
		return cached, nil
	}
	s.mutex.RUnlock()

	// Verifica cache TTL
	cacheKey := fmt.Sprintf("trending:%d", limit)
	if cached, ok := s.cache.Get(cacheKey); ok {
		fmt.Println("[AniList] Retornando cache TTL")
		return cached.([]*AniListAnime), nil
	}

	// Busca da API
	animes, err := s.fetchTrending(limit)
	if err != nil {
		return nil, err
	}

	s.cache.Set(cacheKey, animes, cache.TTLTrending)

	s.mutex.Lock()
	s.trendingCache = animes
	s.mutex.Unlock()

	return animes, nil
}

// GetPopular retorna animes populares
func (s *AniListService) GetPopular(limit int) ([]*AniListAnime, error) {
	if limit <= 0 {
		limit = 20
	}

	cacheKey := fmt.Sprintf("popular:%d", limit)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.([]*AniListAnime), nil
	}

	results, err := anilist.GetPopular(limit)
	if err != nil {
		return nil, err
	}

	animes := make([]*AniListAnime, 0, len(results))
	for _, m := range results {
		animes = append(animes, convertAniListMedia(m))
	}

	s.cache.Set(cacheKey, animes, cache.TTLTrending)
	return animes, nil
}

// Search busca animes no AniList
func (s *AniListService) Search(query string, limit int) ([]*AniListAnime, error) {
	if limit <= 0 {
		limit = 10
	}

	results, err := anilist.SearchAnime(query, limit)
	if err != nil {
		return nil, err
	}

	animes := make([]*AniListAnime, 0, len(results))
	for _, m := range results {
		animes = append(animes, convertAniListMedia(m))
	}

	return animes, nil
}

// GetHDImage busca imagem HD de um anime pelo título
func (s *AniListService) GetHDImage(title string) (map[string]string, error) {
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

// PreloadTrending pré-carrega trending em background
func (s *AniListService) PreloadTrending() {
	go func() {
		animes, err := s.fetchTrending(20)
		if err != nil {
			fmt.Printf("[AniList] Erro no preload: %v\n", err)
			return
		}

		s.mutex.Lock()
		s.trendingCache = animes
		s.mutex.Unlock()
		fmt.Printf("[AniList] Preload concluído: %d animes\n", len(animes))
	}()
}

// SetTrendingCache define o cache de trending diretamente
func (s *AniListService) SetTrendingCache(animes []*AniListAnime) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.trendingCache = animes
}

// GetTrendingCache retorna o cache atual de trending
func (s *AniListService) GetTrendingCache() []*AniListAnime {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.trendingCache
}

// fetchTrending busca trending da API
func (s *AniListService) fetchTrending(limit int) ([]*AniListAnime, error) {
	results, err := anilist.GetTrending(limit)
	if err != nil {
		return nil, err
	}

	animes := make([]*AniListAnime, 0, len(results))
	for _, m := range results {
		animes = append(animes, convertAniListMedia(m))
	}

	return animes, nil
}

// convertAniListMedia converte media do AniList para o tipo da API
func convertAniListMedia(m *anilist.AnimeMedia) *AniListAnime {
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

// AniSkipResult contém timestamps para pular abertura/encerramento
type AniSkipResult struct {
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

// AniSkipService gerencia skip times
type AniSkipService struct {
	cache *cache.Cache
}

// NewAniSkipService cria um novo serviço AniSkip
func NewAniSkipService() *AniSkipService {
	return &AniSkipService{
		cache: cache.New(),
	}
}

// GetSkipTimes busca timestamps de skip para um episódio
func (s *AniSkipService) GetSkipTimes(malID, episodeNumber int) (*AniSkipResult, error) {
	cacheKey := fmt.Sprintf("skip:%d:%d", malID, episodeNumber)

	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*AniSkipResult), nil
	}

	// Importa o pacote aniskip localmente para evitar dependência circular
	// Aqui você chamaria a função real do pacote aniskip
	// Por ora, retorna nil para indicar que não há skip times

	return nil, fmt.Errorf("skip times não disponível")
}

// GetSkipTimesAsync busca skip times de forma assíncrona
func (s *AniSkipService) GetSkipTimesAsync(malID, episodeNumber int, callback func(*AniSkipResult, error)) {
	go func() {
		result, err := s.GetSkipTimes(malID, episodeNumber)
		if callback != nil {
			callback(result, err)
		}
	}()
}
