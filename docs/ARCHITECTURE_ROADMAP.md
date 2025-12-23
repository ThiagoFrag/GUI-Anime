# 🏗️ GoAnime Architecture Roadmap
## Baseado na Filosofia Mihon/Aniyomi

---

## 📊 Análise do Estado Atual

### ✅ O que já temos de bom

| Componente | Status | Observação |
|------------|--------|------------|
| **Provider Interface** | ✅ Existe | `pkg/scrapers/types.go` - Interface `Provider` bem definida |
| **Registry Pattern** | ✅ Existe | `ProviderRegistry` para múltiplos scrapers |
| **AniList Integration** | ✅ Parcial | Busca de metadados/imagens, mas SEM tracking |
| **Local Watch History** | ✅ Existe | `WatchedEpisode` em `pkg/store/data.go` |
| **Multi-source Support** | ✅ Existe | Nyaa + RedeTorrent + AnimeFire + AllAnime |
| **Smart Router** | ✅ Existe | `pkg/smartrouter` para fallback automático |

### ❌ O que FALTA (Comparando com Mihon)

| Feature | Mihon | GoAnime | Prioridade |
|---------|-------|---------|------------|
| **Sistema de Extensions** | APKs externos | Hardcoded | 🔴 CRÍTICO |
| **Repo de Extensions** | keiyoushi/extensions | N/A | 🔴 CRÍTICO |
| **AniList Tracking** | Bidirecional | Apenas leitura | 🟡 ALTO |
| **MAL Tracking** | ✅ | ❌ | 🟡 ALTO |
| **Background Updates** | ✅ | ❌ | 🟡 ALTO |
| **Categorias/Tags** | ✅ | ❌ | 🟢 MÉDIO |
| **Download Manager** | ✅ | Parcial (TorBox) | 🟢 MÉDIO |

---

## 🎯 Roadmap de Implementação

### Fase 1: Sistema de Extensions (CRÍTICO)
> **Objetivo:** Desacoplar scrapers do binário principal

#### 1.1 Extension Manifest (`extension.json`)
```json
{
  "id": "com.goanime.animefox",
  "name": "AnimeFox",
  "version": "1.0.0",
  "minAppVersion": "2.0.0",
  "language": "pt-BR",
  "nsfw": false,
  "author": "GoAnime Community",
  "icon": "https://animefox.tv/favicon.ico",
  "sourceUrl": "https://raw.githubusercontent.com/goanime/extensions/main/animefox/source.lua"
}
```

#### 1.2 Nova Estrutura de Diretórios
```
GoAnimeGUI/
├── pkg/
│   ├── extensions/           # ← NOVO: Engine de extensions
│   │   ├── loader.go         # Carrega extensions de disco/URL
│   │   ├── runtime.go        # Executa scripts Lua/JS
│   │   ├── types.go          # Contratos de Extension
│   │   ├── repository.go     # Gerencia repos remotos
│   │   └── sandbox.go        # Isolamento de segurança
│   ├── scrapers/             # Mantido como fallback/built-in
│   └── ...
├── extensions/               # Extensions instaladas localmente
│   ├── animefox/
│   │   ├── extension.json
│   │   └── source.lua
│   └── animefire/
│       ├── extension.json
│       └── source.lua
```

#### 1.3 Interface de Extension
```go
// pkg/extensions/types.go
package extensions

import "context"

// ExtensionSource é a interface que toda extension deve implementar
type ExtensionSource interface {
    // Metadados
    GetInfo() ExtensionInfo
    
    // Busca
    Search(ctx context.Context, query string) ([]AnimeEntry, error)
    GetLatest(ctx context.Context, page int) ([]AnimeEntry, error)
    GetPopular(ctx context.Context, page int) ([]AnimeEntry, error)
    
    // Detalhes
    GetAnimeDetails(ctx context.Context, url string) (*AnimeDetails, error)
    GetEpisodes(ctx context.Context, animeURL string) ([]Episode, error)
    
    // Streams
    GetVideoSources(ctx context.Context, episodeURL string) ([]VideoSource, error)
}

type ExtensionInfo struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Language    string   `json:"language"`
    BaseURL     string   `json:"baseUrl"`
    HasLatest   bool     `json:"hasLatest"`
    HasPopular  bool     `json:"hasPopular"`
    Filters     []Filter `json:"filters,omitempty"`
}

type VideoSource struct {
    URL      string            `json:"url"`
    Quality  string            `json:"quality"`
    Format   string            `json:"format"` // "hls", "dash", "mp4"
    Headers  map[string]string `json:"headers,omitempty"`
    Subtitles []Subtitle       `json:"subtitles,omitempty"`
}
```

#### 1.4 Runtime Lua (Recomendado)
```go
// pkg/extensions/runtime.go
package extensions

import (
    "context"
    lua "github.com/yuin/gopher-lua"
)

type LuaExtension struct {
    state  *lua.LState
    info   ExtensionInfo
    script string
}

func NewLuaExtension(script string) (*LuaExtension, error) {
    L := lua.NewState()
    
    // Injeta funções HTTP seguras
    L.SetGlobal("http_get", L.NewFunction(safeHTTPGet))
    L.SetGlobal("http_post", L.NewFunction(safeHTTPPost))
    L.SetGlobal("parse_html", L.NewFunction(parseHTML))
    L.SetGlobal("json_decode", L.NewFunction(jsonDecode))
    
    // Executa script
    if err := L.DoString(script); err != nil {
        return nil, err
    }
    
    return &LuaExtension{state: L, script: script}, nil
}
```

#### 1.5 Exemplo de Extension em Lua
```lua
-- extensions/animefox/source.lua

Extension = {
    id = "com.goanime.animefox",
    name = "AnimeFox",
    baseUrl = "https://animefox.tv",
    language = "pt-BR"
}

function search(query)
    local url = Extension.baseUrl .. "/pesquisa?q=" .. url_encode(query)
    local html = http_get(url)
    local doc = parse_html(html)
    
    local results = {}
    for _, item in ipairs(doc:select(".anime-card")) do
        table.insert(results, {
            title = item:select(".title"):text(),
            url = item:select("a"):attr("href"),
            image = item:select("img"):attr("src")
        })
    end
    return results
end

function getEpisodes(animeUrl)
    local html = http_get(animeUrl)
    local doc = parse_html(html)
    
    local episodes = {}
    for _, ep in ipairs(doc:select(".episode-item")) do
        table.insert(episodes, {
            number = tonumber(ep:attr("data-num")),
            url = ep:select("a"):attr("href"),
            title = ep:select(".ep-title"):text()
        })
    end
    return episodes
end

function getVideoSources(episodeUrl)
    local html = http_get(episodeUrl)
    local doc = parse_html(html)
    
    -- Extrai player iframe
    local iframe = doc:select("#player iframe"):attr("src")
    local playerHtml = http_get(iframe)
    
    -- Extrai m3u8/mp4
    local sources = {}
    for url in playerHtml:gmatch('https://[^"]+%.m3u8[^"]*') do
        table.insert(sources, {
            url = url,
            quality = "auto",
            format = "hls"
        })
    end
    return sources
end
```

---

### Fase 2: Tracking Bidirecional

#### 2.1 AniList OAuth + Tracking
```go
// pkg/anilist/tracking.go
package anilist

// UpdateProgress atualiza o progresso no AniList
func (c *Client) UpdateProgress(mediaID int, episode int) error {
    mutation := `
    mutation ($mediaId: Int, $progress: Int) {
      SaveMediaListEntry(mediaId: $mediaId, progress: $progress) {
        id
        progress
        status
      }
    }`
    
    variables := map[string]interface{}{
        "mediaId":  mediaID,
        "progress": episode,
    }
    
    return c.graphqlMutation(mutation, variables)
}

// SyncLibrary sincroniza biblioteca local com AniList
func (c *Client) SyncLibrary() error {
    // 1. Busca lista do usuário no AniList
    // 2. Compara com local
    // 3. Merge inteligente (maior progresso vence)
    return nil
}
```

#### 2.2 MAL Integration
```go
// pkg/mal/mal.go
package mal

import (
    "golang.org/x/oauth2"
)

type Client struct {
    token *oauth2.Token
    http  *http.Client
}

func (c *Client) UpdateAnimeStatus(malID int, episode int, status string) error {
    // PATCH https://api.myanimelist.net/v2/anime/{anime_id}/my_list_status
    return nil
}
```

#### 2.3 Unified Tracking Interface
```go
// pkg/tracking/tracker.go
package tracking

type Tracker interface {
    Name() string
    Login() error
    UpdateProgress(mediaID int, episode int) error
    GetLibrary() ([]LibraryEntry, error)
    Search(query string) ([]SearchResult, error)
}

type TrackerManager struct {
    trackers map[string]Tracker // "anilist", "mal", "kitsu"
}

func (m *TrackerManager) BroadcastProgress(title string, episode int) {
    // Atualiza todos os trackers habilitados em paralelo
    for _, tracker := range m.trackers {
        go tracker.UpdateProgress(...)
    }
}
```

---

### Fase 3: Background Updates

#### 3.1 Update Checker Service
```go
// pkg/updates/checker.go
package updates

type UpdateChecker struct {
    db        *store.Database
    sources   []extensions.ExtensionSource
    interval  time.Duration
    notifyCh  chan<- UpdateNotification
}

type UpdateNotification struct {
    AnimeID      string
    AnimeTitle   string
    NewEpisodes  []int
    Source       string
}

func (u *UpdateChecker) Start(ctx context.Context) {
    ticker := time.NewTicker(u.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            u.checkAllLibrary()
        case <-ctx.Done():
            return
        }
    }
}

func (u *UpdateChecker) checkAllLibrary() {
    library := u.db.GetLibrary()
    
    for _, anime := range library {
        // Busca último episódio conhecido
        lastKnown := anime.LastEpisode
        
        // Busca episódios atuais na fonte
        source := u.getSource(anime.SourceID)
        episodes, _ := source.GetEpisodes(ctx, anime.URL)
        
        if len(episodes) > lastKnown {
            u.notifyCh <- UpdateNotification{
                AnimeID:     anime.ID,
                AnimeTitle:  anime.Title,
                NewEpisodes: getNewEpisodeNumbers(episodes, lastKnown),
            }
        }
    }
}
```

#### 3.2 Frontend Integration (Svelte)
```svelte
<!-- frontend/src/lib/UpdateBadge.svelte -->
<script>
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime';
  
  let updates = [];
  
  onMount(() => {
    EventsOn('anime:new-episodes', (data) => {
      updates = [...updates, data];
      showNotification(data);
    });
  });
  
  function showNotification(update) {
    new Notification('Novo Episódio!', {
      body: `${update.animeTitle} - Ep ${update.newEpisodes.join(', ')}`,
      icon: update.image
    });
  }
</script>

{#if updates.length > 0}
  <div class="update-badge">{updates.length}</div>
{/if}
```

---

### Fase 4: Categorias e Tags

#### 4.1 Database Schema
```go
// pkg/store/categories.go
package store

type Category struct {
    ID       string   `json:"id"`
    Name     string   `json:"name"`
    Color    string   `json:"color"`
    Icon     string   `json:"icon"`
    AnimeIDs []string `json:"animeIds"`
    Order    int      `json:"order"`
}

type LibraryAnime struct {
    ID           string     `json:"id"`
    Title        string     `json:"title"`
    AniListID    int        `json:"anilistId,omitempty"`
    MALID        int        `json:"malId,omitempty"`
    Categories   []string   `json:"categories"`
    Tags         []string   `json:"tags"` // custom tags
    Status       string     `json:"status"` // watching, completed, on_hold, dropped, plan_to_watch
    Progress     int        `json:"progress"`
    TotalEps     int        `json:"totalEps"`
    LastUpdated  time.Time  `json:"lastUpdated"`
    SourceID     string     `json:"sourceId"`
    SourceURL    string     `json:"sourceUrl"`
}
```

---

## 🗂️ Repositório de Extensions (Separado)

### Estrutura Recomendada
```
github.com/goanime/extensions/
├── README.md
├── index.json              # Lista de todas extensions
├── icons/
│   ├── animefox.png
│   └── animefire.png
├── pt-BR/
│   ├── animefox/
│   │   ├── extension.json
│   │   ├── source.lua
│   │   └── icon.png
│   └── animefire/
│       └── ...
├── en/
│   ├── gogoanime/
│   └── zoro/
└── multi/
    ├── nyaa/               # Multi-language
    └── anilist/            # Metadata provider
```

### index.json
```json
{
  "version": 1,
  "lastUpdated": "2025-12-21T00:00:00Z",
  "extensions": [
    {
      "id": "com.goanime.animefox",
      "name": "AnimeFox",
      "version": "1.2.0",
      "language": "pt-BR",
      "nsfw": false,
      "path": "pt-BR/animefox",
      "iconUrl": "icons/animefox.png",
      "changelog": "Corrigido extrator de vídeo"
    }
  ]
}
```

---

## 📋 Checklist de Implementação

### Sprint 1 (2 semanas) - Foundation
- [ ] Criar `pkg/extensions/types.go` com interfaces
- [ ] Implementar `LuaExtension` runtime básico
- [ ] Converter `AnimeFire` scraper atual para Lua como POC
- [ ] UI para listar/instalar extensions

### Sprint 2 (2 semanas) - Tracking
- [ ] AniList OAuth flow completo
- [ ] `UpdateProgress` mutation
- [ ] MAL OAuth + basic tracking
- [ ] UI de tracking na página do anime

### Sprint 3 (1 semana) - Updates
- [ ] Background service de update check
- [ ] Notifications nativas (Windows)
- [ ] Badge de updates na biblioteca

### Sprint 4 (1 semana) - Polish
- [ ] Categorias/Tags na biblioteca
- [ ] Migrar scrapers built-in para extensions
- [ ] Criar repo `goanime/extensions` público

---

## 🔗 Referências

- [Aniyomi Source Interface](https://github.com/aniyomiorg/aniyomi/blob/master/source-api/src/commonMain/kotlin/eu/kanade/tachiyomi/animesource/AnimeSource.kt)
- [Tachiyomi Extensions Repo](https://github.com/keiyoushi/extensions)
- [gopher-lua (Lua 5.1 para Go)](https://github.com/yuin/gopher-lua)
- [AniList GraphQL Docs](https://anilist.github.io/ApiV2-GraphQL-Docs/)
- [MAL API v2 Docs](https://myanimelist.net/apiconfig/references/api/v2)
