# GoAnimeGUI - Estrutura Interna

Esta pasta contém a lógica de negócio do backend organizada em módulos.

## 📁 Estrutura

```
internal/
├── api/           # Handlers de API para o frontend
│   ├── anime.go   # Busca de animes, episódios
│   ├── anilist.go # Integração AniList (trending, imagens HD)
│   ├── discord.go # Integração Discord (vinculação, recomendações)
│   ├── stream.go  # Streaming de vídeos (smart router)
│   └── user.go    # Gerenciamento de usuário (favoritos, histórico)
│
├── cache/         # Sistema de cache
│   ├── cache.go   # Cache genérico com TTL
│   ├── stream.go  # Cache especializado para streams
│   └── sources.go # Rastreamento de falhas de fontes
│
├── player/        # Reprodução de vídeo
│   └── mpv.go     # Integração com MPV player
│
├── proxy/         # Proxy de vídeo
│   └── proxy.go   # Servidor proxy para CORS bypass
│
├── types/         # Tipos comuns
│   └── types.go   # Structs compartilhadas
│
└── utils/         # Utilitários
    ├── helpers.go # Funções auxiliares
    └── html.go    # Parsing de HTML
```

## 🔧 Módulos

### cache/
Sistema de cache thread-safe com TTL automático.

```go
import "goanime-gui/internal/cache"

c := cache.New()
c.Set("key", value, cache.TTLSearch)
val, ok := c.Get("key")
```

### api/
Services para diferentes funcionalidades:

```go
import "goanime-gui/internal/api"

// Anime
animeService := api.NewAnimeService()
animes, _ := animeService.Search("Frieren")

// AniList
anilistService := api.NewAniListService()
trending, _ := anilistService.GetTrending(10)

// Stream
streamService := api.NewStreamService()
result, _ := streamService.GetSmartStream("Frieren", 1)

// Discord
discordService := api.NewDiscordService()
status := discordService.GetLinkStatus()
```

### player/
Integração com MPV:

```go
import "goanime-gui/internal/player"

mpv := player.New()
mpv.FindMPV("")
mpv.Play(url, player.DefaultOptions())
```

### proxy/
Servidor proxy para CORS:

```go
import "goanime-gui/internal/proxy"

server := proxy.New()
server.Start()
proxyURL := server.GetProxyURL(videoURL)
```

## 📊 Constantes de TTL

| Tipo | TTL | Uso |
|------|-----|-----|
| TTLSearch | 10 min | Cache de busca |
| TTLTrending | 30 min | Trending/Popular |
| TTLStream | 15 min | URLs de stream |
| TTLEpisodes | 30 min | Lista de episódios |
| TTLImages | 60 min | Imagens HD |

## 🔄 Circuit Breaker

O `SourceTracker` implementa backoff exponencial:

| Falhas | Cooldown |
|--------|----------|
| 1 | 30 segundos |
| 2 | 1 minuto |
| 3 | 2 minutos |
| 4 | 5 minutos |
| 5+ | 10 minutos |

## 🛠️ Desenvolvimento

Para adicionar um novo módulo:

1. Crie uma nova pasta em `internal/`
2. Implemente a lógica no pacote
3. Exporte as funções necessárias
4. Integre no `app.go`
