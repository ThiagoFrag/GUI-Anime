package anilist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AniList GraphQL API client
// Fornece imagens de alta qualidade e banners para uma UI rica

const apiURL = "https://graphql.anilist.co"

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
	cache      = make(map[string]*AnimeMedia)
	cacheMutex sync.RWMutex
)

// AnimeMedia representa os dados do anime da AniList
type AnimeMedia struct {
	ID          int    `json:"id"`
	MALID       int    `json:"idMal"`
	Title       Title  `json:"title"`
	Description string `json:"description"`
	BannerImage string `json:"bannerImage"`
	CoverImage  struct {
		ExtraLarge string `json:"extraLarge"`
		Large      string `json:"large"`
		Medium     string `json:"medium"`
		Color      string `json:"color"` // Cor predominante para UI
	} `json:"coverImage"`
	Genres       []string `json:"genres"`
	Episodes     int      `json:"episodes"`
	Duration     int      `json:"duration"`
	Status       string   `json:"status"`
	Season       string   `json:"season"`
	SeasonYear   int      `json:"seasonYear"`
	AverageScore int      `json:"averageScore"`
	Popularity   int      `json:"popularity"`
	Studios      struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"studios"`
	NextAiringEpisode *struct {
		Episode         int `json:"episode"`
		TimeUntilAiring int `json:"timeUntilAiring"`
	} `json:"nextAiringEpisode"`
	Trailer *struct {
		ID   string `json:"id"`
		Site string `json:"site"`
	} `json:"trailer"`
}

type Title struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
	Native  string `json:"native"`
}

// GetBestTitle retorna o melhor título disponível
func (m *AnimeMedia) GetBestTitle() string {
	if m.Title.English != "" {
		return m.Title.English
	}
	if m.Title.Romaji != "" {
		return m.Title.Romaji
	}
	return m.Title.Native
}

// GetBestImage retorna a melhor imagem disponível
func (m *AnimeMedia) GetBestImage() string {
	if m.CoverImage.ExtraLarge != "" {
		return m.CoverImage.ExtraLarge
	}
	if m.CoverImage.Large != "" {
		return m.CoverImage.Large
	}
	return m.CoverImage.Medium
}

// GetTrailerURL retorna URL do trailer se disponível
func (m *AnimeMedia) GetTrailerURL() string {
	if m.Trailer == nil {
		return ""
	}
	if m.Trailer.Site == "youtube" {
		return fmt.Sprintf("https://www.youtube.com/watch?v=%s", m.Trailer.ID)
	}
	return ""
}

// GraphQL query para buscar anime
const searchQuery = `
query ($search: String, $page: Int, $perPage: Int) {
  Page(page: $page, perPage: $perPage) {
    media(search: $search, type: ANIME, sort: POPULARITY_DESC) {
      id
      idMal
      title {
        romaji
        english
        native
      }
      description(asHtml: false)
      bannerImage
      coverImage {
        extraLarge
        large
        medium
        color
      }
      genres
      episodes
      duration
      status
      season
      seasonYear
      averageScore
      popularity
      studios(isMain: true) {
        nodes {
          name
        }
      }
      nextAiringEpisode {
        episode
        timeUntilAiring
      }
      trailer {
        id
        site
      }
    }
  }
}
`

const trendingQuery = `
query ($page: Int, $perPage: Int) {
  Page(page: $page, perPage: $perPage) {
    media(type: ANIME, sort: TRENDING_DESC) {
      id
      idMal
      title {
        romaji
        english
        native
      }
      description(asHtml: false)
      bannerImage
      coverImage {
        extraLarge
        large
        medium
        color
      }
      genres
      episodes
      duration
      status
      season
      seasonYear
      averageScore
      popularity
      studios(isMain: true) {
        nodes {
          name
        }
      }
      nextAiringEpisode {
        episode
        timeUntilAiring
      }
      trailer {
        id
        site
      }
    }
  }
}
`

const popularQuery = `
query ($page: Int, $perPage: Int) {
  Page(page: $page, perPage: $perPage) {
    media(type: ANIME, sort: POPULARITY_DESC) {
      id
      idMal
      title {
        romaji
        english
        native
      }
      description(asHtml: false)
      bannerImage
      coverImage {
        extraLarge
        large
        medium
        color
      }
      genres
      episodes
      duration
      status
      season
      seasonYear
      averageScore
      popularity
      studios(isMain: true) {
        nodes {
          name
        }
      }
      nextAiringEpisode {
        episode
        timeUntilAiring
      }
      trailer {
        id
        site
      }
    }
  }
}
`

// Response wrapper para GraphQL
type graphQLResponse struct {
	Data struct {
		Page struct {
			Media []*AnimeMedia `json:"media"`
		} `json:"Page"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// SearchAnime busca anime pelo título com imagens HD
func SearchAnime(title string, limit int) ([]*AnimeMedia, error) {
	if limit <= 0 {
		limit = 10
	}

	// Verifica cache
	cacheKey := fmt.Sprintf("search:%s:%d", strings.ToLower(title), limit)
	cacheMutex.RLock()
	if cached, ok := cache[cacheKey]; ok {
		cacheMutex.RUnlock()
		return []*AnimeMedia{cached}, nil
	}
	cacheMutex.RUnlock()

	variables := map[string]interface{}{
		"search":  title,
		"page":    1,
		"perPage": limit,
	}

	results, err := executeQuery(searchQuery, variables)
	if err != nil {
		return nil, err
	}

	// Salva no cache
	for _, anime := range results {
		cacheMutex.Lock()
		cache[fmt.Sprintf("id:%d", anime.ID)] = anime
		if anime.MALID > 0 {
			cache[fmt.Sprintf("mal:%d", anime.MALID)] = anime
		}
		cacheMutex.Unlock()
	}

	return results, nil
}

// GetTrending retorna os animes em alta (trending)
func GetTrending(limit int) ([]*AnimeMedia, error) {
	if limit <= 0 {
		limit = 20
	}

	variables := map[string]interface{}{
		"page":    1,
		"perPage": limit,
	}

	return executeQuery(trendingQuery, variables)
}

// GetPopular retorna os animes mais populares
func GetPopular(limit int) ([]*AnimeMedia, error) {
	if limit <= 0 {
		limit = 20
	}

	variables := map[string]interface{}{
		"page":    1,
		"perPage": limit,
	}

	return executeQuery(popularQuery, variables)
}

// GetAnimeByMALID busca anime pelo ID do MyAnimeList
func GetAnimeByMALID(malID int) (*AnimeMedia, error) {
	cacheKey := fmt.Sprintf("mal:%d", malID)

	cacheMutex.RLock()
	if cached, ok := cache[cacheKey]; ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	query := `
	query ($malId: Int) {
	  Media(idMal: $malId, type: ANIME) {
		id
		idMal
		title {
		  romaji
		  english
		  native
		}
		description(asHtml: false)
		bannerImage
		coverImage {
		  extraLarge
		  large
		  medium
		  color
		}
		genres
		episodes
		duration
		status
		season
		seasonYear
		averageScore
		popularity
		studios(isMain: true) {
		  nodes {
			name
		  }
		}
		trailer {
		  id
		  site
		}
	  }
	}
	`

	variables := map[string]interface{}{
		"malId": malID,
	}

	body := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonBody, _ := json.Marshal(body)
	resp, err := httpClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Media *AnimeMedia `json:"Media"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Data.Media != nil {
		cacheMutex.Lock()
		cache[cacheKey] = result.Data.Media
		cacheMutex.Unlock()
	}

	return result.Data.Media, nil
}

// GetHDImage busca imagem HD de um anime pelo título
func GetHDImage(title string) (image string, banner string, color string, err error) {
	results, err := SearchAnime(title, 1)
	if err != nil || len(results) == 0 {
		return "", "", "", fmt.Errorf("anime não encontrado")
	}

	anime := results[0]
	return anime.GetBestImage(), anime.BannerImage, anime.CoverImage.Color, nil
}

func executeQuery(query string, variables map[string]interface{}) ([]*AnimeMedia, error) {
	body := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("AniList error: %s", result.Errors[0].Message)
	}

	return result.Data.Page.Media, nil
}
