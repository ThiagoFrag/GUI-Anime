# GoAnime Installer

Este diretório contém os arquivos necessários para criar o instalador do GoAnime.

## 📦 O que está incluído no instalador

- **GoAnimeGUI.exe** - Aplicativo principal
- **MPV Player** - Reprodutor de vídeo otimizado
- **Anime4K Shaders** - Upscaling de anime em tempo real (38 shaders)
- **FSR** - AMD FidelityFX Super Resolution
- **FSRCNNX** - Neural network upscaler
- **Configurações otimizadas** - mpv.conf e input.conf pré-configurados

## 🛠️ Requisitos para compilar

1. **Inno Setup 6** - Compilador do instalador
   - Download: https://jrsoftware.org/isdl.php
   - Ou via Chocolatey: `choco install innosetup`
   - Ou via Winget: `winget install JRSoftware.InnoSetup`

2. **Go 1.21+** - Para compilar o aplicativo
3. **Wails CLI** - `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
4. **Node.js 18+** - Para o frontend

## 🚀 Como compilar o instalador

### Método 1: Script automático (recomendado)

```powershell
cd installer
.\build_installer.ps1
```

Opções:
- `-SkipBuild` - Pula a compilação do Wails (usa executável existente)
- `-Version "2.1.0"` - Define a versão do instalador

### Método 2: Manual

1. Compile o aplicativo:
```powershell
cd ..
wails build -clean
```

2. Copie os arquivos do MPV para `installer/mpv/`
3. Copie os shaders para `installer/shaders/`
4. Abra `GoAnimeGUI_Setup.iss` no Inno Setup e compile

## 📁 Estrutura de arquivos

```
installer/
├── GoAnimeGUI_Setup.iss     # Script do Inno Setup
├── build_installer.ps1      # Script de build automatizado
├── README.md                # Este arquivo
├── mpv/
│   ├── mpv.exe              # Player
│   ├── d3dcompiler_43.dll   # Dependência
│   └── portable_config/
│       ├── mpv.conf         # Configuração do MPV
│       ├── input.conf       # Atalhos de teclado
│       ├── scripts/         # Scripts Lua
│       └── fonts/           # Fontes para legendas
└── shaders/
    ├── Anime4K/             # Shaders Anime4K
    ├── FSR.glsl             # AMD FSR
    └── FSRCNNX_x2_*.glsl    # Neural network upscaler
```

## 📋 Atalhos de teclado incluídos

| Tecla | Função |
|-------|--------|
| `1` | Anime4K Mode A (Rápido) |
| `2` | Anime4K Mode B (Qualidade) |
| `3` | Anime4K Mode C (Alta Qualidade) |
| `4` | AMD FSR Upscaling |
| `5` | FSRCNNX Neural Network |
| `0` | Modo Performance (sem shaders) |
| `Ctrl+RIGHT` | Pula 90s (skip opening) |
| `f` | Fullscreen |
| `i` | Estatísticas de vídeo |

## 🔧 Personalização

### Mudar versão
Edite a linha no arquivo `.iss`:
```
#define MyAppVersion "2.0.0"
```

### Adicionar mais shaders
Coloque os arquivos `.glsl` em `shaders/` e adicione ao script:
```
Source: "shaders\MeuShader.glsl"; DestDir: "{app}\shaders"; Flags: ignoreversion
```

### Mudar ícone
Substitua `../build/appicon.ico` pelo seu ícone.

## 📤 Saída

O instalador será criado em:
```
../dist/GoAnime_Setup_v2.0.0.exe
```

Tamanho estimado: 50-80 MB (dependendo do MPV e shaders)

## 🐛 Solução de problemas

### "Inno Setup não encontrado"
Instale o Inno Setup 6 de https://jrsoftware.org/isdl.php

### "GoAnimeGUI.exe não encontrado"
Execute `wails build` primeiro ou use a flag `-SkipBuild` se já existe.

### "MPV não encontrado"
Copie manualmente os arquivos do MPV para `installer/mpv/`

## 📝 Notas

- O instalador NÃO requer privilégios de administrador por padrão
- Instala em `%LOCALAPPDATA%\Programs\GoAnime` ou permite escolher
- Cria atalhos no Menu Iniciar e Desktop
- Registra caminhos no registro para o app encontrar o MPV
- Inclui desinstalador completo
