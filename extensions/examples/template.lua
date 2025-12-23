-- ============================================================================
-- GoAnime Extension Template
-- Versão: 1.0.0
-- ============================================================================
-- Este é um template para criar extensions de anime para o GoAnime.
-- Cada extension é um script Lua que implementa funções específicas para
-- buscar e extrair conteúdo de um site de anime.
--
-- FUNÇÕES OBRIGATÓRIAS:
--   search(query, page, filters) -> results, hasNextPage
--   getAnimeDetails(url) -> details
--   getEpisodes(animeUrl) -> episodes
--   getVideoSources(episodeUrl) -> sources
--
-- FUNÇÕES OPCIONAIS:
--   getLatest(page) -> results, hasNextPage
--   getPopular(page) -> results, hasNextPage
--
-- FUNÇÕES GLOBAIS DISPONÍVEIS:
--   http_get(url, headers?) -> html
--   http_post(url, body, headers?) -> html
--   url_encode(string) -> encoded_string
--   parse_html(html) -> document
--   trim(string) -> trimmed_string
--   split(string, separator) -> table
--   json.decode(string) -> table
--   json.encode(table) -> string
--
-- MÉTODOS DO DOCUMENTO HTML:
--   doc:select(selector) -> selection
--   selection:text() -> string
--   selection:attr(name) -> string
--   selection:html() -> string
--   selection:each(function(i, el)) -> void
--   selection:first() -> selection
--   selection:last() -> selection
--   selection:length() -> number
-- ============================================================================

-- Metadados da Extension (OBRIGATÓRIO)
Extension = {
    id = "com.goanime.template",       -- ID único (com.goanime.nomedosite)
    name = "Template Extension",        -- Nome amigável
    version = "1.0.0",                 -- Versão semântica
    language = "pt-BR",                -- Idioma: pt-BR, en, ja, multi
    baseUrl = "https://example.com",   -- URL base do site
    iconUrl = "",                      -- URL do ícone (opcional)
    author = "GoAnime Community",      -- Autor
    nsfw = false                       -- Conteúdo adulto?
}

-- ============================================================================
-- FUNÇÃO: search
-- Busca animes por query
-- 
-- Parâmetros:
--   query (string): Termo de busca
--   page (number): Número da página (1-indexed)
--   filters (table): Filtros aplicados (ex: {genre="action", year="2024"})
--
-- Retorno:
--   results (table): Lista de animes encontrados
--   hasNextPage (boolean): Se há mais páginas
-- ============================================================================
function search(query, page, filters)
    local results = {}
    
    -- Monta URL de busca
    local url = Extension.baseUrl .. "/search?q=" .. url_encode(query) .. "&page=" .. page
    
    -- Aplica filtros se existirem
    if filters.genre then
        url = url .. "&genre=" .. url_encode(filters.genre)
    end
    
    -- Faz requisição HTTP
    local html = http_get(url)
    if not html then
        return results, false
    end
    
    -- Parse do HTML
    local doc = parse_html(html)
    
    -- Extrai resultados
    -- ADAPTE O SELETOR CSS PARA O SITE ESPECÍFICO
    doc:select(".anime-card"):each(function(i, el)
        local anime = {
            title = el:select(".title"):text(),
            url = el:select("a"):attr("href"),
            image = el:select("img"):attr("src"),
            status = el:select(".status"):text()  -- opcional
        }
        
        -- Normaliza URL relativa
        if anime.url and not anime.url:match("^https?://") then
            anime.url = Extension.baseUrl .. anime.url
        end
        
        table.insert(results, anime)
    end)
    
    -- Verifica se há próxima página
    local hasNext = doc:select(".pagination .next"):length() > 0
    
    return results, hasNext
end

-- ============================================================================
-- FUNÇÃO: getLatest (OPCIONAL)
-- Retorna os últimos lançamentos/atualizações
-- ============================================================================
function getLatest(page)
    local results = {}
    local url = Extension.baseUrl .. "/releases?page=" .. page
    
    local html = http_get(url)
    if not html then
        return results, false
    end
    
    local doc = parse_html(html)
    
    doc:select(".release-item"):each(function(i, el)
        table.insert(results, {
            title = el:select(".title"):text(),
            url = el:select("a"):attr("href"),
            image = el:select("img"):attr("src")
        })
    end)
    
    local hasNext = doc:select(".pagination .next"):length() > 0
    return results, hasNext
end

-- ============================================================================
-- FUNÇÃO: getPopular (OPCIONAL)
-- Retorna os animes mais populares
-- ============================================================================
function getPopular(page)
    local results = {}
    local url = Extension.baseUrl .. "/popular?page=" .. page
    
    local html = http_get(url)
    if not html then
        return results, false
    end
    
    local doc = parse_html(html)
    
    doc:select(".popular-item"):each(function(i, el)
        table.insert(results, {
            title = el:select(".title"):text(),
            url = el:select("a"):attr("href"),
            image = el:select("img"):attr("src")
        })
    end)
    
    local hasNext = doc:select(".pagination .next"):length() > 0
    return results, hasNext
end

-- ============================================================================
-- FUNÇÃO: getAnimeDetails
-- Retorna informações detalhadas de um anime
--
-- Parâmetros:
--   url (string): URL da página do anime
--
-- Retorno:
--   details (table): Informações do anime
-- ============================================================================
function getAnimeDetails(url)
    local html = http_get(url)
    if not html then
        return nil
    end
    
    local doc = parse_html(html)
    
    -- Extrai gêneros
    local genres = {}
    doc:select(".genres a"):each(function(i, el)
        table.insert(genres, el:text())
    end)
    
    -- Extrai detalhes
    local details = {
        title = doc:select("h1.title"):text(),
        alternateTitle = doc:select(".alt-title"):text(),
        url = url,
        image = doc:select(".poster img"):attr("src"),
        banner = doc:select(".banner"):attr("style"):match("url%((.-)%)"),
        description = doc:select(".synopsis"):text(),
        status = doc:select(".status"):text(),  -- "ongoing" ou "completed"
        genres = genres,
        year = tonumber(doc:select(".year"):text()),
        studio = doc:select(".studio"):text(),
        rating = tonumber(doc:select(".rating"):text())
    }
    
    return details
end

-- ============================================================================
-- FUNÇÃO: getEpisodes
-- Retorna a lista de episódios de um anime
--
-- Parâmetros:
--   animeUrl (string): URL da página do anime
--
-- Retorno:
--   episodes (table): Lista de episódios
-- ============================================================================
function getEpisodes(animeUrl)
    local episodes = {}
    
    local html = http_get(animeUrl)
    if not html then
        return episodes
    end
    
    local doc = parse_html(html)
    
    doc:select(".episode-list .episode"):each(function(i, el)
        local ep = {
            number = tonumber(el:attr("data-num")) or i,
            title = el:select(".ep-title"):text(),
            url = el:select("a"):attr("href"),
            thumbnail = el:select("img"):attr("src"),
            filler = el:select(".filler-badge"):length() > 0
        }
        
        -- Normaliza URL
        if ep.url and not ep.url:match("^https?://") then
            ep.url = Extension.baseUrl .. ep.url
        end
        
        table.insert(episodes, ep)
    end)
    
    return episodes
end

-- ============================================================================
-- FUNÇÃO: getVideoSources
-- Extrai as fontes de vídeo de um episódio
--
-- Parâmetros:
--   episodeUrl (string): URL da página do episódio
--
-- Retorno:
--   sources (table): Lista de fontes de vídeo
-- ============================================================================
function getVideoSources(episodeUrl)
    local sources = {}
    
    local html = http_get(episodeUrl)
    if not html then
        return sources
    end
    
    local doc = parse_html(html)
    
    -- Exemplo 1: Extrai URL do player embutido
    local iframe = doc:select("#player iframe"):attr("src")
    if iframe and iframe ~= "" then
        -- Faz requisição ao iframe para extrair m3u8
        local playerHtml = http_get(iframe)
        if playerHtml then
            -- Tenta encontrar URL HLS
            local m3u8 = playerHtml:match('https://[^"\']+%.m3u8[^"\']*')
            if m3u8 then
                table.insert(sources, {
                    url = m3u8,
                    quality = "auto",
                    format = "hls",
                    server = "Principal"
                })
            end
            
            -- Tenta encontrar URL MP4
            local mp4 = playerHtml:match('https://[^"\']+%.mp4[^"\']*')
            if mp4 then
                table.insert(sources, {
                    url = mp4,
                    quality = "720p",
                    format = "mp4",
                    server = "Download"
                })
            end
        end
    end
    
    -- Exemplo 2: Múltiplos servidores
    doc:select(".server-list .server"):each(function(i, el)
        local serverUrl = el:attr("data-url")
        local serverName = el:text()
        
        if serverUrl then
            table.insert(sources, {
                url = serverUrl,
                quality = "auto",
                format = "hls",
                server = serverName
            })
        end
    end)
    
    -- Exemplo 3: Com legendas
    local subUrl = doc:select("#subtitles"):attr("src")
    if subUrl and #sources > 0 then
        sources[1].subtitles = {
            {
                url = subUrl,
                language = "pt-BR",
                label = "Português",
                format = "vtt",
                default = true
            }
        }
    end
    
    return sources
end

-- ============================================================================
-- FUNÇÕES AUXILIARES (use conforme necessário)
-- ============================================================================

-- Extrai número do episódio de uma string
function extractEpisodeNumber(text)
    local num = text:match("Epis[oó]dio%s*(%d+)")
           or text:match("EP%s*(%d+)")
           or text:match("E(%d+)")
           or text:match("(%d+)")
    return tonumber(num)
end

-- Limpa e normaliza título
function cleanTitle(title)
    title = trim(title)
    title = title:gsub("%s+", " ")  -- Remove espaços duplos
    return title
end

-- Extrai dados de JSON embutido na página
function extractJsonData(html, varName)
    local pattern = varName .. "%s*=%s*({.-});?"
    local jsonStr = html:match(pattern)
    if jsonStr then
        return json.decode(jsonStr)
    end
    return nil
end
