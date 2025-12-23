# 🎬 GoAnime GUI

Aplicativo desktop multiplataforma para assistir animes com streaming de alta qualidade, upscaling 4K com shaders de IA, e integração com diversos serviços.

![GoAnime](https://img.shields.io/badge/GoAnime-v2.0.0-blue)
![Wails](https://img.shields.io/badge/Wails-v2.11.0-green)
![Svelte](https://img.shields.io/badge/Svelte-v5-orange)

## ✨ Funcionalidades

### 🎥 Player
- **Player 4K integrado** com upscaling via shaders Anime4K
- Modos de qualidade: Low, Medium, High
- Suporte a legendas externas (.srt, .ass, .vtt)
- Pular intro/outro automático (AniSkip)
- Atalhos de teclado personalizados

### 📺 Streaming
- Múltiplas fontes de anime (AniList, Consumet, AnimeFire, AllAnime)
- TorBox integration para torrents
- VPS streaming pipeline
- Cache inteligente de streams

### 📖 Mangá
- Leitor de mangá integrado
- Múltiplas fontes (MangaLivre, etc.)
- Favoritos e histórico de leitura

### 👥 Social
- Sistema de amigos
- Compartilhar o que está assistindo
- Integração Discord RPC

### ⚙️ Configurações
- Seeding automático
- Limites de CPU/banda configuráveis
- Temas claro/escuro
- Exportar/Importar dados do usuário

## 🛠️ Requisitos

- Windows 10/11
- [Go 1.21+](https://golang.org/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

## 🚀 Desenvolvimento

```bash
# Clonar o repositório
git clone https://github.com/seu-usuario/GoAnimeGUI.git
cd GoAnimeGUI

# Instalar dependências
go mod tidy
cd frontend && npm install && cd ..

# Rodar em modo desenvolvimento
wails dev
```

## 📦 Build

```bash
# Build de produção
wails build

# Build com instalador
cd installer
.\build_installer.ps1
```

## 📁 Estrutura do Projeto

```
GoAnimeGUI/
├── app.go              # Lógica principal do aplicativo
├── main.go             # Ponto de entrada
├── player_methods.go   # Integração com Player 4K
├── torbox_methods.go   # API TorBox
├── remote_api.go       # API VPS remota
├── seeding.go          # Worker de seeding
├── social_methods.go   # Sistema social
├── frontend/           # Interface Svelte
│   └── src/
│       └── App.svelte  # Componente principal
├── pkg/                # Pacotes internos
│   ├── anilist/        # API AniList
│   ├── consumet/       # API Consumet
│   ├── embeddedplayer/ # Player MPV integrado
│   └── store/          # Armazenamento local
├── installer/          # Scripts de instalação
└── bin/                # Binários (MPV, player4k)
```

## ⌨️ Atalhos do Player

| Tecla | Ação |
|-------|------|
| `ESPAÇO` | Play/Pause |
| `← →` | Seek -5s/+5s |
| `↑ ↓` | Volume +/- |
| `I` | Pular intro (85s) |
| `F` | Tela cheia |
| `S` | Screenshot |
| `M` | Mute |
| `V` | Mostrar/ocultar legendas |
| `J` | Próxima legenda |
| `A` | Próximo áudio |
| `[ ]` | Velocidade -/+ |
| `Q` | Fechar |

## 📜 Licença

MIT License - Veja [LICENSE](LICENSE) para mais detalhes.

## 🙏 Créditos

- [Wails](https://wails.io/) - Framework desktop
- [Svelte](https://svelte.dev/) - Framework UI
- [MPV](https://mpv.io/) - Player de vídeo
- [Anime4K](https://github.com/bloc97/Anime4K) - Shaders de upscaling
- [AniSkip](https://github.com/lexesjan/typescript-aniskip-extension) - Dados de skip
