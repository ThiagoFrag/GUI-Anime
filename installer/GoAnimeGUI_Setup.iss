; GoAnimeGUI Installer Script
; Inno Setup Script - https://jrsoftware.org/isinfo.php
; 
; Este instalador inclui:
; - GoAnimeGUI.exe (aplicativo principal)
; - MPV Player (reprodutor de vídeo)
; - Shaders Anime4K, FSR, FSRCNNX (upscaling AI)
; - Configurações otimizadas

#define MyAppName "GoAnime"
#define MyAppVersion "2.0.0"
#define MyAppPublisher "GoAnime Team"
#define MyAppURL "https://github.com/goanime"
#define MyAppExeName "GoAnimeGUI.exe"
#define MyAppIcon "..\build\appicon.ico"

[Setup]
; Identificador único do aplicativo
AppId={{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}

; Diretórios
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes

; Saída
OutputDir=..\dist
OutputBaseFilename=GoAnime_Setup_v{#MyAppVersion}
; SetupIconFile=..\build\appicon.ico

; Compressão (LZMA2 é o melhor para tamanho)
Compression=lzma2/ultra64
SolidCompression=yes
LZMAUseSeparateProcess=yes
LZMANumBlockThreads=4

; Visual
WizardStyle=modern
; WizardImageFile=wizard_image.bmp
; WizardSmallImageFile=wizard_small.bmp

; Privilégios
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog

; Outras opções
UninstallDisplayIcon={app}\{#MyAppExeName}
UninstallDisplayName={#MyAppName}
VersionInfoVersion={#MyAppVersion}
VersionInfoCompany={#MyAppPublisher}
VersionInfoDescription=Instalador do GoAnime - Assista animes com upscaling AI
VersionInfoProductName={#MyAppName}
VersionInfoProductVersion={#MyAppVersion}
MinVersion=10.0

[Languages]
Name: "brazilianportuguese"; MessagesFile: "compiler:Languages\BrazilianPortuguese.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[CustomMessages]
brazilianportuguese.InstallingMPV=Instalando MPV Player...
brazilianportuguese.InstallingShaders=Instalando Shaders de Upscaling AI...
brazilianportuguese.InstallingApp=Instalando GoAnime...
brazilianportuguese.CreatingShortcuts=Criando atalhos...
brazilianportuguese.FinishMessage=GoAnime foi instalado com sucesso!%n%nInclui:%n- Player 4K com upscaling AI%n- Shaders Anime4K, FSR, FSRCNNX%n%nAproveite!
english.InstallingMPV=Installing MPV Player...
english.InstallingShaders=Installing AI Upscaling Shaders...
english.InstallingApp=Installing GoAnime...
english.CreatingShortcuts=Creating shortcuts...
english.FinishMessage=GoAnime has been installed successfully!%n%nIncludes:%n- 4K Player with AI upscaling%n- Anime4K, FSR, FSRCNNX Shaders%n%nEnjoy!

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: checkedonce
Name: "quicklaunchicon"; Description: "{cm:CreateQuickLaunchIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked; OnlyBelowVersion: 6.1

[Files]
; === APLICATIVO PRINCIPAL ===
Source: "..\build\bin\GoAnimeGUI.exe"; DestDir: "{app}"; Flags: ignoreversion; \
    BeforeInstall: StatusMessage(ExpandConstant('{cm:InstallingApp}'))

; === MPV PLAYER ===
Source: "mpv\mpv.exe"; DestDir: "{app}\mpv"; Flags: ignoreversion; \
    BeforeInstall: StatusMessage(ExpandConstant('{cm:InstallingMPV}'))
Source: "mpv\d3dcompiler_43.dll"; DestDir: "{app}\mpv"; Flags: ignoreversion
Source: "mpv\libmpv-2.dll"; DestDir: "{app}\mpv"; Flags: ignoreversion skipifsourcedoesntexist

; === CONFIGURAÇÃO DO MPV ===
Source: "mpv\portable_config\mpv.conf"; DestDir: "{app}\mpv\portable_config"; Flags: ignoreversion
Source: "mpv\portable_config\input.conf"; DestDir: "{app}\mpv\portable_config"; Flags: ignoreversion
Source: "mpv\portable_config\scripts\*"; DestDir: "{app}\mpv\portable_config\scripts"; Flags: ignoreversion recursesubdirs createallsubdirs skipifsourcedoesntexist
Source: "mpv\portable_config\script-opts\*"; DestDir: "{app}\mpv\portable_config\script-opts"; Flags: ignoreversion recursesubdirs createallsubdirs skipifsourcedoesntexist
Source: "mpv\portable_config\fonts\*"; DestDir: "{app}\mpv\portable_config\fonts"; Flags: ignoreversion recursesubdirs createallsubdirs skipifsourcedoesntexist

; === SHADERS DE UPSCALING AI ===
; Anime4K
Source: "shaders\Anime4K\*.glsl"; DestDir: "{app}\shaders\Anime4K"; Flags: ignoreversion; \
    BeforeInstall: StatusMessage(ExpandConstant('{cm:InstallingShaders}'))

; FSR (AMD FidelityFX Super Resolution)
Source: "shaders\FSR.glsl"; DestDir: "{app}\shaders"; Flags: ignoreversion skipifsourcedoesntexist

; FSRCNNX (Neural Network Upscaler)
Source: "shaders\FSRCNNX_x2_16-0-4-1.glsl"; DestDir: "{app}\shaders"; Flags: ignoreversion skipifsourcedoesntexist

; === ÍCONE DO APP ===
; Source: "..\build\appicon.ico"; DestDir: "{app}"; DestName: "icon.ico"; Flags: ignoreversion skipifsourcedoesntexist

[Dirs]
; Cria diretórios para cache e dados do usuário
Name: "{app}\mpv\portable_config\cache"
Name: "{app}\mpv\portable_config\subtitles"
Name: "{app}\downloads"

[Icons]
; Menu Iniciar
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Comment: "Assista animes com qualidade 4K"
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"

; Desktop
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

; Quick Launch
Name: "{userappdata}\Microsoft\Internet Explorer\Quick Launch\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: quicklaunchicon

[Registry]
; Registra o caminho do MPV para o aplicativo encontrar
Root: HKCU; Subkey: "Software\GoAnime"; ValueType: string; ValueName: "InstallPath"; ValueData: "{app}"; Flags: uninsdeletekey
Root: HKCU; Subkey: "Software\GoAnime"; ValueType: string; ValueName: "MPVPath"; ValueData: "{app}\mpv\mpv.exe"; Flags: uninsdeletekey
Root: HKCU; Subkey: "Software\GoAnime"; ValueType: string; ValueName: "ShadersPath"; ValueData: "{app}\shaders"; Flags: uninsdeletekey

[Run]
; Opção de executar após instalação
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
; Limpa cache e arquivos temporários na desinstalação
Type: filesandordirs; Name: "{app}\mpv\portable_config\cache"
Type: filesandordirs; Name: "{app}\downloads"

[Code]
// Mostra mensagem de status durante instalação
procedure StatusMessage(Msg: String);
begin
    WizardForm.StatusLabel.Caption := Msg;
end;

// Verifica se o WebView2 está instalado
function IsWebView2Installed(): Boolean;
var
    Version: String;
begin
    Result := RegQueryStringValue(HKLM, 'SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}', 'pv', Version) or
              RegQueryStringValue(HKLM, 'SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}', 'pv', Version) or
              RegQueryStringValue(HKCU, 'SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}', 'pv', Version);
end;

// Página de boas-vindas customizada
procedure InitializeWizard();
begin
    WizardForm.WelcomeLabel2.Caption := 
        'Este assistente irá instalar o GoAnime no seu computador.' + #13#10 + #13#10 +
        'O pacote inclui:' + #13#10 +
        '• GoAnime - Aplicativo principal' + #13#10 +
        '• MPV Player - Reprodutor de vídeo otimizado' + #13#10 +
        '• Anime4K - Shaders de upscaling para anime' + #13#10 +
        '• FSR/FSRCNNX - Upscaling neural network' + #13#10 + #13#10 +
        'Clique em Avançar para continuar.';
end;

// Verifica requisitos antes de instalar
function InitializeSetup(): Boolean;
begin
    Result := True;
    
    // Avisa se WebView2 não estiver instalado (necessário para Wails)
    if not IsWebView2Installed() then
    begin
        if MsgBox('O Microsoft WebView2 Runtime não foi detectado.' + #13#10 + 
                  'O GoAnime precisa dele para funcionar.' + #13#10 + #13#10 +
                  'Deseja continuar a instalação mesmo assim?' + #13#10 +
                  '(O WebView2 será baixado automaticamente na primeira execução)',
                  mbConfirmation, MB_YESNO) = IDNO then
        begin
            Result := False;
        end;
    end;
end;

// Mensagem final customizada
procedure CurStepChanged(CurStep: TSetupStep);
begin
    if CurStep = ssPostInstall then
    begin
        // Podemos fazer ações pós-instalação aqui se necessário
    end;
end;
