// Package api fornece handlers de API para o frontend
package api

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"goanime-gui/internal/cache"
	"goanime-gui/internal/utils"
	"goanime-gui/pkg/jikan"
	"goanime-gui/pkg/store"

	goanime "github.com/alvarorichard/Goanime/pkg/goanime"
	"github.com/alvarorichard/Goanime/pkg/goanime/types"
)

// AnimeService gerencia operações relacionadas a animes
type AnimeService struct {
	client       *goanime.Client
	cache        *cache.Cache
	episodeCache map[string][]store.Episode
	urlCache     map[string]string
	mutex        sync.RWMutex
}

// NewAnimeService cria um novo serviço de anime
func NewAnimeService() *AnimeService {
	return &AnimeService{
		client:       goanime.NewClient(),
		cache:        cache.New(),
		episodeCache: make(map[string][]store.Episode),
		urlCache:     make(map[string]string),
	}
}

// Search busca animes em múltiplas fontes
func (s *AnimeService) Search(termo string) ([]store.SavedAnime, error) {
	termoLower := strings.TrimSpace(strings.ToLower(termo))
	if termoLower == "" {
		return []store.SavedAnime{}, nil
	}

	// Verifica cache
	cacheKey := "search:" + termoLower
	if cached, ok := s.cache.Get(cacheKey); ok {
		fmt.Printf("[Search] Cache hit para: %s\n", termoLower)
		return cached.([]store.SavedAnime), nil
	}

	// Busca em paralelo nas duas fontes
	type searchResult struct {
		animes []*types.Anime
		source string
		err    error
	}

	resultChan := make(chan searchResult, 2)

	// AllAnime (inglês)
	go func() {
		srcAllAnime := types.SourceAllAnime
		animes, err := s.client.SearchAnime(termo, &srcAllAnime)
		resultChan <- searchResult{animes, "AllAnime", err}
	}()

	// AnimeFire (português)
	go func() {
		srcAnimeFire := types.SourceAnimeFire
		animes, err := s.client.SearchAnime(termo, &srcAnimeFire)
		resultChan <- searchResult{animes, "AnimeFire", err}
	}()

	// Coleta resultados com timeout
	animeMap := make(map[string]*store.SavedAnime)
	timeout := time.After(4 * time.Second)
	received := 0

	for received < 2 {
		select {
		case res := <-resultChan:
			received++
			if res.err != nil {
				fmt.Printf("[Search] %s erro: %v\n", res.source, res.err)
				continue
			}
			fmt.Printf("[Search] %s: %d resultados\n", res.source, len(res.animes))

			lang := "en"
			if res.source == "AnimeFire" {
				lang = "pt-BR"
			}

			for _, anime := range res.animes {
				if anime == nil {
					continue
				}
				normalized := utils.NormalizeAnimeName(anime.Name)
				if normalized == "" {
					continue
				}

				if existing, ok := animeMap[normalized]; ok {
					existing.Sources = append(existing.Sources, store.AnimeSource{
						Name:     res.source,
						Language: lang,
						URL:      anime.URL,
					})
					if existing.Image == "" && anime.ImageURL != "" {
						existing.Image = anime.ImageURL
					}
				} else {
					animeMap[normalized] = &store.SavedAnime{
						Title: anime.Name,
						Image: anime.ImageURL,
						URL:   anime.URL,
						Sources: []store.AnimeSource{{
							Name:     res.source,
							Language: lang,
							URL:      anime.URL,
						}},
					}
				}
			}
		case <-timeout:
			fmt.Println("[Search] Timeout - usando resultados parciais")
			received = 2
		}
	}

	// Converte para slice
	final := make([]store.SavedAnime, 0, len(animeMap))
	for _, anime := range animeMap {
		final = append(final, *anime)
	}

	// Ordena por número de fontes
	sort.Slice(final, func(i, j int) bool {
		return len(final[i].Sources) > len(final[j].Sources)
	})

	// Busca imagens em paralelo para animes sem imagem
	s.enrichWithImages(final)

	// Salva no cache
	s.cache.Set(cacheKey, final, cache.TTLSearch)

	fmt.Printf("[Search] Total: %d animes\n", len(final))
	return final, nil
}

// SearchMulti busca múltiplos termos em paralelo
func (s *AnimeService) SearchMulti(termos []string) ([]store.SavedAnime, error) {
	if len(termos) == 0 {
		return []store.SavedAnime{}, nil
	}

	fmt.Printf("[SearchMulti] Buscando %d termos\n", len(termos))

	type searchResult struct {
		animes []store.SavedAnime
		termo  string
		err    error
	}

	resultChan := make(chan searchResult, len(termos))

	for _, termo := range termos {
		go func(t string) {
			animes, err := s.Search(t)
			resultChan <- searchResult{animes, t, err}
		}(termo)
	}

	timeout := time.After(8 * time.Second)
	seenTitles := make(map[string]bool)
	allResults := make([]store.SavedAnime, 0)
	received := 0

	for received < len(termos) {
		select {
		case res := <-resultChan:
			received++
			if res.err != nil {
				continue
			}
			for _, anime := range res.animes {
				key := strings.ToLower(anime.Title)
				if !seenTitles[key] {
					seenTitles[key] = true
					allResults = append(allResults, anime)
				}
			}
		case <-timeout:
			received = len(termos)
		}
	}

	return allResults, nil
}

// GetEpisodes busca episódios de um anime
func (s *AnimeService) GetEpisodes(seriesURL string) ([]store.Episode, error) {
	if seriesURL == "" {
		return nil, fmt.Errorf("URL inválida")
	}

	// Verifica cache
	s.mutex.RLock()
	if eps, ok := s.episodeCache[seriesURL]; ok && len(eps) > 0 {
		s.mutex.RUnlock()
		fmt.Printf("[GetEpisodes] Cache hit: %d episódios\n", len(eps))
		return eps, nil
	}
	s.mutex.RUnlock()

	// Busca em todas as fontes em paralelo
	sources := s.client.GetAvailableSources()
	type epResult struct {
		episodes []*types.Episode
		source   types.Source
		err      error
	}

	resultChan := make(chan epResult, len(sources))

	for _, src := range sources {
		go func(source types.Source) {
			eps, err := s.client.GetAnimeEpisodes(seriesURL, source)
			resultChan <- epResult{eps, source, err}
		}(src)
	}

	timeout := time.After(5 * time.Second)
	var bestEpisodes []store.Episode
	received := 0

	for received < len(sources) {
		select {
		case res := <-resultChan:
			received++
			if res.err != nil || len(res.episodes) == 0 {
				continue
			}
			mapped := s.convertEpisodes(res.episodes, res.source.String())
			if len(mapped) > len(bestEpisodes) {
				bestEpisodes = mapped
				fmt.Printf("[GetEpisodes] %s: %d episódios (melhor)\n", res.source, len(mapped))
			}
		case <-timeout:
			received = len(sources)
		}
	}

	if len(bestEpisodes) > 0 {
		s.mutex.Lock()
		s.episodeCache[seriesURL] = bestEpisodes
		s.mutex.Unlock()
		return bestEpisodes, nil
	}

	return nil, fmt.Errorf("nenhum episódio encontrado")
}

// GetAnimeURL busca a URL de um anime pelo título
func (s *AnimeService) GetAnimeURL(title string) (string, error) {
	s.mutex.RLock()
	if url, ok := s.urlCache[title]; ok {
		s.mutex.RUnlock()
		return url, nil
	}
	s.mutex.RUnlock()

	searchResults, err := s.client.SearchAnime(title, nil)
	if err != nil || len(searchResults) == 0 || searchResults[0] == nil {
		return "", fmt.Errorf("anime não encontrado")
	}

	url := searchResults[0].URL

	s.mutex.Lock()
	s.urlCache[title] = url
	s.mutex.Unlock()

	return url, nil
}

// ClearEpisodesCache limpa o cache de episódios
func (s *AnimeService) ClearEpisodesCache() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.episodeCache = make(map[string][]store.Episode)
	fmt.Println("[AnimeService] Cache de episódios limpo")
}

// ClearAllCache limpa todos os caches
func (s *AnimeService) ClearAllCache() {
	s.mutex.Lock()
	s.episodeCache = make(map[string][]store.Episode)
	s.urlCache = make(map[string]string)
	s.mutex.Unlock()
	s.cache.Clear()
	fmt.Println("[AnimeService] Todos os caches limpos")
}

// convertEpisodes converte episódios do tipo da biblioteca para o tipo da store
func (s *AnimeService) convertEpisodes(eps []*types.Episode, source string) []store.Episode {
	mapped := make([]store.Episode, 0, len(eps))

	for _, te := range eps {
		if te == nil {
			continue
		}

		title := ""
		if te.Title != nil {
			if te.Title.English != "" {
				title = te.Title.English
			} else if te.Title.Romaji != "" {
				title = te.Title.Romaji
			} else if te.Title.Japanese != "" {
				title = te.Title.Japanese
			}
		}
		if title == "" {
			title = te.Number
		}

		num := te.Num
		if num == 0 {
			num = utils.ExtractEpisodeNumber(te.Number)
		}

		mapped = append(mapped, store.Episode{
			Title:  title,
			URL:    te.URL,
			Season: 1,
			Number: num,
			Source: source,
		})
	}

	sort.Slice(mapped, func(i, j int) bool {
		return mapped[i].Number < mapped[j].Number
	})

	return mapped
}

// enrichWithImages busca imagens para animes que não têm
func (s *AnimeService) enrichWithImages(animes []store.SavedAnime) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)

	for i := range animes {
		if animes[i].Image == "" {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				poster := jikan.FetchPosterMultiSource(animes[idx].Title)
				if poster != "" {
					animes[idx].Image = poster
				}
			}(i)
		}
	}

	// Espera com timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2500 * time.Millisecond):
		fmt.Println("[enrichWithImages] Timeout na busca de imagens")
	}
}
