-- ============================================================================
-- AnimeFire Extension para GoAnime
-- Versão: 1.0.0
-- Fonte: https://animefire.plus
-- Idioma: pt-BR (Dublado e Legendado)
-- ============================================================================

Extension = {
    id = "com.goanime.animefire",
    name = "AnimeFire",
    version = "1.0.0",
    language = "pt-BR",
    baseUrl = "https://animefire.plus",
    iconUrl = "https://animefire.plus/favicon.ico",
    author = "GoAnime Community",
    nsfw = false
}

-- ============================================================================
-- BUSCA
-- ============================================================================
function search(query, page, filters)
    local results = {}
    local url = Extension.baseUrl .. "/pesquisar/" .. url_encode(query)
    
    if page > 1 then
        url = url .. "?page=" .. page
    end
    
    local html = http_get(url)
    if not html then
        return results, false
    end
    
    local doc = parse_html(html)
    
    -- Seletor para cards de anime
    doc:select(".divCardUltimosEps .card"):each(function(i, el)
        local linkEl = el:select("a")
        local imgEl = el:select("img")
        
        local anime = {
            title = trim(el:select(".animeTitle"):text()),
            url = linkEl:attr("href"),
            image = imgEl:attr("src") or imgEl:attr("data-src")
        }
        
        -- Normaliza URL
        if anime.url and not anime.url:match("^https?://") then
            anime.url = Extension.baseUrl .. anime.url
        end
        
        if anime.title and anime.title ~= "" then
            table.insert(results, anime)
        end
    end)
    
    -- Verifica próxima página
    local hasNext = doc:select(".pagination .next"):length() > 0 or
                    doc:select("a[rel='next']"):length() > 0
    
    return results, hasNext
end

-- ============================================================================
-- ÚLTIMOS LANÇAMENTOS
-- ============================================================================
function getLatest(page)
    local results = {}
    local url = Extension.baseUrl
    
    if page > 1 then
        url = url .. "/?page=" .. page
    end
    
    local html = http_get(url)
    if not html then
        return results, false
    end
    
    local doc = parse_html(html)
    
    -- Episódios recentes na home
    doc:select(".divCardUltimosEps .card"):each(function(i, el)
        local linkEl = el:select("a")
        local imgEl = el:select("img")
        
        local anime = {
            title = trim(el:select(".animeTitle"):text()),
            url = linkEl:attr("href"),
            image = imgEl:attr("src") or imgEl:attr("data-src")
        }
        
        if anime.url and not anime.url:match("^https?://") then
            anime.url = Extension.baseUrl .. anime.url
        end
        
        if anime.title and anime.title ~= "" then
            table.insert(results, anime)
        end
    end)
    
    local hasNext = page < 5 -- Limita a 5 páginas
    return results, hasNext
end

-- ============================================================================
-- DETALHES DO ANIME
-- ============================================================================
function getAnimeDetails(url)
    local html = http_get(url)
    if not html then
        return nil
    end
    
    local doc = parse_html(html)
    
    -- Extrai gêneros
    local genres = {}
    doc:select(".animeInfo a[href*='genero']"):each(function(i, el)
        table.insert(genres, trim(el:text()))
    end)
    
    -- Extrai sinopse
    local description = doc:select(".animeDescription"):text()
    if description == "" then
        description = doc:select(".divSinopse"):text()
    end
    
    -- Extrai status
    local status = "ongoing"
    local statusText = doc:select(".animeInfo"):text():lower()
    if statusText:match("completo") or statusText:match("finalizado") then
        status = "completed"
    end
    
    local details = {
        title = trim(doc:select("h1.animeTitle"):text()),
        alternateTitle = trim(doc:select(".animeTitleOriginal"):text()),
        url = url,
        image = doc:select(".animeImg img"):attr("src"),
        description = trim(description),
        status = status,
        genres = genres,
        year = tonumber(doc:select(".animeInfo"):text():match("(%d%d%d%d)")),
        studio = trim(doc:select(".animeInfo a[href*='estudio']"):text())
    }
    
    return details
end

-- ============================================================================
-- LISTA DE EPISÓDIOS
-- ============================================================================
function getEpisodes(animeUrl)
    local episodes = {}
    
    local html = http_get(animeUrl)
    if not html then
        return episodes
    end
    
    local doc = parse_html(html)
    
    -- Tenta diferentes seletores
    local epList = doc:select(".div_video_list a")
    if epList:length() == 0 then
        epList = doc:select(".listaDeEps a")
    end
    if epList:length() == 0 then
        epList = doc:select(".animeVideos a")
    end
    
    epList:each(function(i, el)
        local epUrl = el:attr("href")
        local epText = trim(el:text())
        
        -- Extrai número do episódio
        local epNum = tonumber(epText:match("(%d+)"))
        if not epNum then
            epNum = i
        end
        
        local ep = {
            number = epNum,
            title = "Episódio " .. epNum,
            url = epUrl,
            filler = false
        }
        
        if ep.url and not ep.url:match("^https?://") then
            ep.url = Extension.baseUrl .. ep.url
        end
        
        table.insert(episodes, ep)
    end)
    
    -- Ordena por número
    table.sort(episodes, function(a, b)
        return a.number < b.number
    end)
    
    return episodes
end

-- ============================================================================
-- FONTES DE VÍDEO
-- ============================================================================
function getVideoSources(episodeUrl)
    local sources = {}
    
    local html = http_get(episodeUrl)
    if not html then
        return sources
    end
    
    local doc = parse_html(html)
    
    -- Método 1: Extrai data-video-src dos botões
    doc:select("[data-video-src]"):each(function(i, el)
        local videoUrl = el:attr("data-video-src")
        if videoUrl and videoUrl ~= "" then
            table.insert(sources, {
                url = videoUrl,
                quality = "auto",
                format = detectFormat(videoUrl),
                server = "AnimeFire"
            })
        end
    end)
    
    -- Método 2: Busca no script da página
    if #sources == 0 then
        -- Procura por URLs de vídeo no HTML
        for videoUrl in html:gmatch('https://[^"\'%s]+%.m3u8[^"\'%s]*') do
            table.insert(sources, {
                url = videoUrl,
                quality = "auto",
                format = "hls",
                server = "AnimeFire HLS"
            })
        end
        
        for videoUrl in html:gmatch('https://[^"\'%s]+%.mp4[^"\'%s]*') do
            table.insert(sources, {
                url = videoUrl,
                quality = "720p",
                format = "mp4",
                server = "AnimeFire MP4"
            })
        end
    end
    
    -- Método 3: Busca iframe do player
    if #sources == 0 then
        local iframe = doc:select("iframe"):attr("src")
        if iframe and iframe ~= "" then
            -- Faz requisição ao iframe
            local playerHtml = http_get(iframe)
            if playerHtml then
                for videoUrl in playerHtml:gmatch('https://[^"\'%s]+%.m3u8[^"\'%s]*') do
                    table.insert(sources, {
                        url = videoUrl,
                        quality = "auto",
                        format = "hls",
                        server = "Player"
                    })
                end
            end
        end
    end
    
    -- Método 4: API Beta do AnimeFire
    if #sources == 0 then
        -- Extrai ID do episódio da URL
        local epId = episodeUrl:match("/video/([^/]+)")
        if epId then
            local apiUrl = Extension.baseUrl .. "/api/video/" .. epId
            local apiResponse = http_get(apiUrl)
            if apiResponse then
                local data = json.decode(apiResponse)
                if data and data.data then
                    for _, src in ipairs(data.data) do
                        table.insert(sources, {
                            url = src.src,
                            quality = src.label or "auto",
                            format = detectFormat(src.src),
                            server = "API"
                        })
                    end
                end
            end
        end
    end
    
    return sources
end

-- ============================================================================
-- FUNÇÕES AUXILIARES
-- ============================================================================

-- Detecta formato do vídeo pela URL
function detectFormat(url)
    if url:match("%.m3u8") then
        return "hls"
    elseif url:match("%.mp4") then
        return "mp4"
    elseif url:match("%.mpd") then
        return "dash"
    else
        return "hls" -- Assume HLS por padrão
    end
end
