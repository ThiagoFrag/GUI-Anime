package manga

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkerPool gerencia requisições paralelas com rate limiting
type WorkerPool struct {
	client       *MangaClient
	cache        *MangaCache
	workers      int
	rateLimiter  chan struct{}
	requestDelay time.Duration
	mutex        sync.Mutex
}

// Task representa uma tarefa para o worker pool
type Task struct {
	Type     string // "mangas", "details", "chapters", "pages"
	URL      string // URL para processar
	Page     int    // Número da página (para paginação)
	Callback func(interface{}, error)
}

// BatchResult representa o resultado de uma operação em lote
type BatchResult struct {
	Mangas   []Manga
	Chapters []MangaChapter
	Pages    []MangaPage
	Errors   []error
}

// NewWorkerPool cria um novo pool de workers
func NewWorkerPool(client *MangaClient, cache *MangaCache, workers int) *WorkerPool {
	if workers <= 0 {
		workers = 3 // Padrão: 3 workers simultâneos
	}

	return &WorkerPool{
		client:       client,
		cache:        cache,
		workers:      workers,
		rateLimiter:  make(chan struct{}, workers),
		requestDelay: 200 * time.Millisecond, // 200ms entre requisições
	}
}

// acquire obtém um slot no rate limiter
func (wp *WorkerPool) acquire() {
	wp.rateLimiter <- struct{}{}
}

// release libera um slot no rate limiter
func (wp *WorkerPool) release() {
	<-wp.rateLimiter
}

// FetchAllMangasParallel busca todos os mangás em paralelo
// Estratégia: busca páginas em lotes paralelos até não encontrar mais novos mangás
func (wp *WorkerPool) FetchAllMangasParallel(ctx context.Context, maxPages int) ([]Manga, error) {
	// Verifica cache primeiro
	cacheKey := "all_mangas_complete"
	if cached, ok := wp.cache.GetMangas(cacheKey); ok && len(cached) > 0 {
		fmt.Printf("[WorkerPool] Retornando %d mangás do cache\n", len(cached))
		return cached, nil
	}

	if maxPages <= 0 {
		maxPages = 25 // Limite padrão de segurança
	}

	allMangas := make([]Manga, 0, 150)
	seenURLs := make(map[string]bool)
	var allMangasMu sync.Mutex

	// Processa páginas em lotes paralelos
	batchSize := wp.workers // Número de páginas por lote
	page := 1
	emptyPages := 0

	for page <= maxPages && emptyPages < 2 {
		// Define o lote atual
		endPage := page + batchSize - 1
		if endPage > maxPages {
			endPage = maxPages
		}

		type pageResult struct {
			page   int
			mangas []Manga
			err    error
		}

		results := make(chan pageResult, batchSize)
		var wg sync.WaitGroup

		// Lança workers para as páginas do lote
		for p := page; p <= endPage; p++ {
			wg.Add(1)
			go func(pageNum int) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					results <- pageResult{page: pageNum, err: ctx.Err()}
					return
				default:
				}

				wp.acquire()
				defer wp.release()

				// Delay escalonado para não sobrecarregar
				time.Sleep(time.Duration(pageNum-page) * 100 * time.Millisecond)

				pageMangas, _, err := wp.client.GetAllMangas(pageNum)
				results <- pageResult{page: pageNum, mangas: pageMangas, err: err}
			}(p)
		}

		// Fecha canal quando todos terminarem
		go func() {
			wg.Wait()
			close(results)
		}()

		// Coleta resultados do lote
		batchEmpty := true
		for result := range results {
			if result.err != nil {
				fmt.Printf("[WorkerPool] Erro na página %d: %v\n", result.page, result.err)
				continue
			}

			if len(result.mangas) > 0 {
				batchEmpty = false
				allMangasMu.Lock()
				for _, m := range result.mangas {
					if !seenURLs[m.URL] {
						seenURLs[m.URL] = true
						allMangas = append(allMangas, m)
					}
				}
				fmt.Printf("[WorkerPool] Página %d: +%d mangás (total: %d)\n", result.page, len(result.mangas), len(allMangas))
				allMangasMu.Unlock()
			}
		}

		if batchEmpty {
			emptyPages++
		} else {
			emptyPages = 0
		}

		page = endPage + 1
	}

	fmt.Printf("[WorkerPool] TOTAL: %d mangás encontrados em %d páginas\n", len(allMangas), page-1)

	// Salva no cache
	if len(allMangas) > 0 {
		wp.cache.SetMangas(cacheKey, allMangas, TTLMangaList)
	}

	return allMangas, nil
}

// FetchChaptersWithCache busca capítulos com cache
func (wp *WorkerPool) FetchChaptersWithCache(mangaURL string) ([]MangaChapter, error) {
	cacheKey := fmt.Sprintf("chapters:%s", mangaURL)

	// Verifica cache
	if cached, ok := wp.cache.GetChapters(cacheKey); ok && len(cached) > 0 {
		fmt.Printf("[WorkerPool] Retornando %d capítulos do cache\n", len(cached))
		return cached, nil
	}

	// Busca do servidor
	wp.acquire()
	defer wp.release()

	time.Sleep(wp.requestDelay)
	chapters, err := wp.client.GetChapters(mangaURL)
	if err != nil {
		return nil, err
	}

	// Salva no cache
	wp.cache.SetChapters(cacheKey, chapters, TTLMangaChapters)

	return chapters, nil
}

// FetchDetailsWithCache busca detalhes do mangá com cache
func (wp *WorkerPool) FetchDetailsWithCache(mangaURL string) (*Manga, error) {
	cacheKey := fmt.Sprintf("details:%s", mangaURL)

	// Verifica cache
	if cached, ok := wp.cache.Get(cacheKey); ok {
		if manga, ok := cached.(*Manga); ok && manga != nil {
			fmt.Printf("[WorkerPool] Retornando detalhes do cache: %s\n", manga.Title)
			return manga, nil
		}
		// Tenta converter de map
		if m, ok := cached.(map[string]interface{}); ok {
			manga := mangaFromMap(m)
			return &manga, nil
		}
	}

	// Busca do servidor
	wp.acquire()
	defer wp.release()

	time.Sleep(wp.requestDelay)
	manga, err := wp.client.GetMangaDetails(mangaURL)
	if err != nil {
		return nil, err
	}

	// Salva no cache
	wp.cache.Set(cacheKey, manga, TTLMangaDetails)

	return manga, nil
}

// FetchPagesWithCache busca páginas do capítulo com cache
func (wp *WorkerPool) FetchPagesWithCache(chapterURL string) ([]MangaPage, error) {
	cacheKey := fmt.Sprintf("pages:%s", chapterURL)

	// Verifica cache
	if cached, ok := wp.cache.GetPages(cacheKey); ok && len(cached) > 0 {
		fmt.Printf("[WorkerPool] Retornando %d páginas do cache\n", len(cached))
		return cached, nil
	}

	// Busca do servidor
	wp.acquire()
	defer wp.release()

	time.Sleep(wp.requestDelay)
	pages, err := wp.client.GetChapterPages(chapterURL)
	if err != nil {
		return nil, err
	}

	// Salva no cache (longo TTL pois páginas raramente mudam)
	wp.cache.SetPages(cacheKey, pages, TTLMangaPages)

	return pages, nil
}

// PreloadMangasDetails pré-carrega detalhes de vários mangás em paralelo
func (wp *WorkerPool) PreloadMangasDetails(ctx context.Context, mangaURLs []string) {
	var wg sync.WaitGroup

	for _, url := range mangaURLs {
		wg.Add(1)
		go func(mangaURL string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			// Já está em cache?
			cacheKey := fmt.Sprintf("details:%s", mangaURL)
			if _, ok := wp.cache.Get(cacheKey); ok {
				return
			}

			wp.acquire()
			defer wp.release()

			time.Sleep(wp.requestDelay)
			manga, err := wp.client.GetMangaDetails(mangaURL)
			if err == nil && manga != nil {
				wp.cache.Set(cacheKey, manga, TTLMangaDetails)
			}
		}(url)
	}

	wg.Wait()
}

// BatchFetchChapters busca capítulos de vários mangás em paralelo
func (wp *WorkerPool) BatchFetchChapters(ctx context.Context, mangaURLs []string) map[string][]MangaChapter {
	results := make(map[string][]MangaChapter)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for _, url := range mangaURLs {
		wg.Add(1)
		go func(mangaURL string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			chapters, err := wp.FetchChaptersWithCache(mangaURL)
			if err == nil && len(chapters) > 0 {
				mutex.Lock()
				results[mangaURL] = chapters
				mutex.Unlock()
			}
		}(url)
	}

	wg.Wait()
	return results
}

// ClearCache limpa o cache
func (wp *WorkerPool) ClearCache() {
	wp.cache.Clear()
}

// CacheStats retorna estatísticas do cache
func (wp *WorkerPool) CacheStats() (total, expired, valid int) {
	return wp.cache.Stats()
}
