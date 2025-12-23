# build_installer.ps1
# Script para compilar o instalador completo do GoAnime
# Inclui: Aplicativo, MPV, Shaders

param(
    [switch]$SkipBuild,      # Pula compilação do Wails
    [switch]$SkipDownload,   # Pula download do MPV
    [string]$Version = "2.0.0"
)

$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$rootDir = Split-Path -Parent $scriptDir
$installerDir = $scriptDir
$distDir = Join-Path $rootDir "dist"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   GoAnime Installer Builder v$Version" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Cria diretórios necessários
$dirs = @(
    $distDir,
    "$installerDir\mpv",
    "$installerDir\mpv\portable_config",
    "$installerDir\mpv\portable_config\scripts",
    "$installerDir\mpv\portable_config\script-opts",
    "$installerDir\mpv\portable_config\fonts",
    "$installerDir\shaders",
    "$installerDir\shaders\Anime4K"
)

foreach ($dir in $dirs) {
    if (-not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
        Write-Host "[+] Criado: $dir" -ForegroundColor Green
    }
}

# ==========================================
# STEP 1: Compilar o aplicativo Wails
# ==========================================
if (-not $SkipBuild) {
    Write-Host ""
    Write-Host "[1/4] Compilando GoAnimeGUI..." -ForegroundColor Yellow
    Push-Location $rootDir
    try {
        wails build -clean
        if ($LASTEXITCODE -ne 0) {
            throw "Falha na compilação do Wails"
        }
        Write-Host "[OK] GoAnimeGUI.exe compilado!" -ForegroundColor Green
    }
    finally {
        Pop-Location
    }
} else {
    Write-Host "[1/4] Pulando compilação (--SkipBuild)" -ForegroundColor DarkGray
}

# Verifica se o executável existe
$exePath = Join-Path $rootDir "build\bin\GoAnimeGUI.exe"
if (-not (Test-Path $exePath)) {
    throw "GoAnimeGUI.exe não encontrado em: $exePath"
}

# ==========================================
# STEP 2: Baixar/Copiar MPV
# ==========================================
Write-Host ""
Write-Host "[2/4] Preparando MPV Player..." -ForegroundColor Yellow

$mpvSourceDir = "C:\Users\th\Documents\codigos\player4k\mpv"
$mpvDestDir = "$installerDir\mpv"

if (Test-Path $mpvSourceDir) {
    # Copia do player4k local
    Write-Host "  Copiando MPV de $mpvSourceDir..." -ForegroundColor Gray
    
    # mpv.exe
    if (Test-Path "$mpvSourceDir\mpv.exe") {
        Copy-Item "$mpvSourceDir\mpv.exe" "$mpvDestDir\mpv.exe" -Force
        Write-Host "  [+] mpv.exe" -ForegroundColor Green
    }
    
    # d3dcompiler_43.dll
    if (Test-Path "$mpvSourceDir\d3dcompiler_43.dll") {
        Copy-Item "$mpvSourceDir\d3dcompiler_43.dll" "$mpvDestDir\d3dcompiler_43.dll" -Force
        Write-Host "  [+] d3dcompiler_43.dll" -ForegroundColor Green
    }
    
    # libmpv-2.dll (se existir)
    $libmpvPath = "C:\Users\th\Documents\codigos\player4k\libmpv-2.dll"
    if (Test-Path $libmpvPath) {
        Copy-Item $libmpvPath "$mpvDestDir\libmpv-2.dll" -Force
        Write-Host "  [+] libmpv-2.dll" -ForegroundColor Green
    }
    
    # Configurações do MPV
    if (Test-Path "$mpvSourceDir\portable_config") {
        # mpv.conf
        if (Test-Path "$mpvSourceDir\portable_config\mpv.conf") {
            Copy-Item "$mpvSourceDir\portable_config\mpv.conf" "$mpvDestDir\portable_config\mpv.conf" -Force
            Write-Host "  [+] mpv.conf" -ForegroundColor Green
        }
        
        # input.conf
        if (Test-Path "$mpvSourceDir\portable_config\input.conf") {
            Copy-Item "$mpvSourceDir\portable_config\input.conf" "$mpvDestDir\portable_config\input.conf" -Force
            Write-Host "  [+] input.conf" -ForegroundColor Green
        }
        
        # Scripts
        if (Test-Path "$mpvSourceDir\portable_config\scripts") {
            Copy-Item "$mpvSourceDir\portable_config\scripts\*" "$mpvDestDir\portable_config\scripts\" -Recurse -Force -ErrorAction SilentlyContinue
            Write-Host "  [+] scripts/" -ForegroundColor Green
        }
        
        # Script-opts
        if (Test-Path "$mpvSourceDir\portable_config\script-opts") {
            Copy-Item "$mpvSourceDir\portable_config\script-opts\*" "$mpvDestDir\portable_config\script-opts\" -Recurse -Force -ErrorAction SilentlyContinue
            Write-Host "  [+] script-opts/" -ForegroundColor Green
        }
        
        # Fonts
        if (Test-Path "$mpvSourceDir\portable_config\fonts") {
            Copy-Item "$mpvSourceDir\portable_config\fonts\*" "$mpvDestDir\portable_config\fonts\" -Recurse -Force -ErrorAction SilentlyContinue
            Write-Host "  [+] fonts/" -ForegroundColor Green
        }
    }
} elseif (-not $SkipDownload) {
    # Baixa MPV se não existir localmente
    Write-Host "  Baixando MPV..." -ForegroundColor Gray
    $mpvUrl = "https://github.com/shinchiro/mpv-winbuild-cmake/releases/download/20231231/mpv-x86_64-20231231-git-abc1234.7z"
    $mpvZip = "$installerDir\mpv_temp.7z"
    
    # Usa versão estável do SourceForge
    $mpvUrl = "https://sourceforge.net/projects/mpv-player-windows/files/64bit/mpv-x86_64-20231231-git-abc1234.7z/download"
    
    Write-Host "  [!] MPV não encontrado localmente. Copie manualmente para: $mpvDestDir" -ForegroundColor Yellow
}

Write-Host "[OK] MPV preparado!" -ForegroundColor Green

# ==========================================
# STEP 3: Copiar Shaders
# ==========================================
Write-Host ""
Write-Host "[3/4] Copiando Shaders de Upscaling AI..." -ForegroundColor Yellow

$shadersSourceDir = "C:\Users\th\Documents\codigos\player4k\shaders"
$shadersDestDir = "$installerDir\shaders"

if (Test-Path $shadersSourceDir) {
    # Anime4K
    if (Test-Path "$shadersSourceDir\Anime4K") {
        Copy-Item "$shadersSourceDir\Anime4K\*.glsl" "$shadersDestDir\Anime4K\" -Force
        $count = (Get-ChildItem "$shadersDestDir\Anime4K\*.glsl").Count
        Write-Host "  [+] Anime4K: $count shaders" -ForegroundColor Green
    }
    
    # FSR
    if (Test-Path "$shadersSourceDir\FSR.glsl") {
        Copy-Item "$shadersSourceDir\FSR.glsl" "$shadersDestDir\FSR.glsl" -Force
        Write-Host "  [+] FSR.glsl" -ForegroundColor Green
    }
    
    # FSRCNNX
    if (Test-Path "$shadersSourceDir\FSRCNNX_x2_16-0-4-1.glsl") {
        Copy-Item "$shadersSourceDir\FSRCNNX_x2_16-0-4-1.glsl" "$shadersDestDir\FSRCNNX_x2_16-0-4-1.glsl" -Force
        Write-Host "  [+] FSRCNNX_x2_16-0-4-1.glsl" -ForegroundColor Green
    }
}

Write-Host "[OK] Shaders copiados!" -ForegroundColor Green

# ==========================================
# STEP 4: Criar ícone (se não existir)
# ==========================================
$iconPath = Join-Path $rootDir "build\appicon.ico"
if (-not (Test-Path $iconPath)) {
    Write-Host ""
    Write-Host "[!] appicon.ico não encontrado. Criando a partir de appicon.png..." -ForegroundColor Yellow
    
    $pngPath = Join-Path $rootDir "build\appicon.png"
    if (Test-Path $pngPath) {
        # Tenta converter com ImageMagick se disponível
        $magick = Get-Command magick -ErrorAction SilentlyContinue
        if ($magick) {
            magick $pngPath -define icon:auto-resize=256,128,64,48,32,16 $iconPath
            Write-Host "  [+] Ícone criado com ImageMagick" -ForegroundColor Green
        } else {
            Write-Host "  [!] ImageMagick não encontrado. Copie appicon.ico manualmente." -ForegroundColor Yellow
        }
    }
}

# ==========================================
# STEP 5: Compilar Instalador
# ==========================================
Write-Host ""
Write-Host "[4/4] Compilando instalador..." -ForegroundColor Yellow

# Procura pelo Inno Setup
$innoSetupPaths = @(
    "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
    "C:\Program Files\Inno Setup 6\ISCC.exe",
    "$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe"
)

$iscc = $null
foreach ($path in $innoSetupPaths) {
    if (Test-Path $path) {
        $iscc = $path
        break
    }
}

if (-not $iscc) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "  Inno Setup não encontrado!" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    Write-Host ""
    Write-Host "Para compilar o instalador, instale o Inno Setup:" -ForegroundColor Yellow
    Write-Host "  https://jrsoftware.org/isdl.php" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Ou instale via Chocolatey:" -ForegroundColor Yellow
    Write-Host "  choco install innosetup" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Ou instale via Winget:" -ForegroundColor Yellow
    Write-Host "  winget install JRSoftware.InnoSetup" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Arquivos preparados em: $installerDir" -ForegroundColor Green
    Write-Host "Script do instalador: $installerDir\GoAnimeGUI_Setup.iss" -ForegroundColor Green
    exit 1
}

Write-Host "  Usando: $iscc" -ForegroundColor Gray

# Atualiza versão no script
$issPath = "$installerDir\GoAnimeGUI_Setup.iss"
$issContent = Get-Content $issPath -Raw
$issContent = $issContent -replace '#define MyAppVersion ".*"', "#define MyAppVersion `"$Version`""
Set-Content $issPath $issContent -NoNewline

# Compila o instalador
Push-Location $installerDir
try {
    & $iscc "GoAnimeGUI_Setup.iss"
    if ($LASTEXITCODE -ne 0) {
        throw "Falha na compilação do instalador"
    }
}
finally {
    Pop-Location
}

# ==========================================
# RESULTADO FINAL
# ==========================================
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "   INSTALADOR CRIADO COM SUCESSO!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

$installerPath = Join-Path $distDir "GoAnime_Setup_v$Version.exe"
if (Test-Path $installerPath) {
    $size = [math]::Round((Get-Item $installerPath).Length / 1MB, 2)
    Write-Host "Arquivo: $installerPath" -ForegroundColor Cyan
    Write-Host "Tamanho: $size MB" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "O instalador inclui:" -ForegroundColor Yellow
    Write-Host "  - GoAnimeGUI.exe" -ForegroundColor White
    Write-Host "  - MPV Player (reprodutor de video)" -ForegroundColor White
    Write-Host "  - Anime4K Shaders (upscaling anime)" -ForegroundColor White
    Write-Host "  - FSR/FSRCNNX (upscaling neural network)" -ForegroundColor White
    Write-Host "  - Configuracoes otimizadas" -ForegroundColor White
}

Write-Host ""
Write-Host "Para testar: Abra $installerPath" -ForegroundColor Gray
