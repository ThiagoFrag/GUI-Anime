-- ═══════════════════════════════════════════════════════════════════════════════
--  🎬 GoAnime Player - Script Lua Customizado
--  Logo, OSD personalizado e funcionalidades nativas
-- ═══════════════════════════════════════════════════════════════════════════════

local mp = require 'mp'
local msg = require 'mp.msg'
local utils = require 'mp.utils'
local assdraw = require 'mp.assdraw'

-- Cores do GoAnime
local PINK = "FF6B9D"
local DARK_BG = "0D0D1A"
local WHITE = "FFFFFF"

-- Configurações
local config = {
    show_logo_on_start = true,
    logo_duration = 2.0,
    show_filename_on_load = true,
    auto_skip_intro = false,
    custom_osd = true
}

-- ═══════════════════════════════════════════════════════════════════════════════
--  LOGO/SPLASH DO GOANIME
-- ═══════════════════════════════════════════════════════════════════════════════

local logo_overlay = nil

function show_goanime_logo()
    if not config.show_logo_on_start then return end
    
    local osd_w, osd_h = mp.get_osd_size()
    if not osd_w or osd_w == 0 then
        osd_w, osd_h = 1920, 1080
    end
    
    local ass = assdraw.ass_new()
    
    -- Fundo semi-transparente
    ass:new_event()
    ass:pos(0, 0)
    ass:append("{\\an7\\bord0\\shad0\\1c&H" .. DARK_BG .. "&\\alpha&H40&}")
    ass:draw_start()
    ass:rect_cw(0, 0, osd_w, osd_h)
    ass:draw_stop()
    
    -- Logo texto "GoAnime"
    ass:new_event()
    ass:pos(osd_w/2, osd_h/2 - 40)
    ass:an(5)
    ass:append("{\\fn Segoe UI\\fs72\\b1\\bord3\\shad2\\1c&H" .. PINK .. "&\\3c&H" .. DARK_BG .. "&}")
    ass:append("🎬 GoAnime")
    
    -- Subtítulo
    ass:new_event()
    ass:pos(osd_w/2, osd_h/2 + 30)
    ass:an(5)
    ass:append("{\\fn Segoe UI\\fs28\\b0\\bord1\\shad1\\1c&H" .. WHITE .. "&\\3c&H000000&}")
    ass:append("Player 4K")
    
    -- Versão
    ass:new_event()
    ass:pos(osd_w/2, osd_h/2 + 70)
    ass:an(5)
    ass:append("{\\fn Segoe UI\\fs16\\b0\\bord1\\shad0\\1c&H888888&}")
    ass:append("Upscaling • Interpolação • Deband")
    
    mp.set_osd_ass(osd_w, osd_h, ass.text)
    
    -- Remove depois de X segundos
    mp.add_timeout(config.logo_duration, function()
        mp.set_osd_ass(0, 0, "")
    end)
end

-- ═══════════════════════════════════════════════════════════════════════════════
--  OSD PERSONALIZADO - INFO DO VÍDEO
-- ═══════════════════════════════════════════════════════════════════════════════

function show_video_info()
    local title = mp.get_property("media-title") or mp.get_property("filename")
    local duration = mp.get_property_number("duration") or 0
    local width = mp.get_property_number("width") or 0
    local height = mp.get_property_number("height") or 0
    local fps = mp.get_property_number("estimated-vf-fps") or mp.get_property_number("container-fps") or 0
    
    -- Formata duração
    local dur_min = math.floor(duration / 60)
    local dur_sec = math.floor(duration % 60)
    local dur_str = string.format("%d:%02d", dur_min, dur_sec)
    
    -- Detecta qualidade
    local quality = "SD"
    if height >= 2160 then
        quality = "4K"
    elseif height >= 1440 then
        quality = "2K"
    elseif height >= 1080 then
        quality = "FHD"
    elseif height >= 720 then
        quality = "HD"
    end
    
    -- Limita título
    if #title > 60 then
        title = string.sub(title, 1, 57) .. "..."
    end
    
    local info = string.format("🎬 %s\n%s • %dx%d • %.1f fps • %s", 
        title, quality, width, height, fps, dur_str)
    
    mp.osd_message(info, 3)
end

-- ═══════════════════════════════════════════════════════════════════════════════
--  NOTIFICAÇÕES ESTILIZADAS
-- ═══════════════════════════════════════════════════════════════════════════════

function notify(text, duration)
    duration = duration or 1.5
    mp.osd_message("🎬 " .. text, duration)
end

-- Volume com ícone
function show_volume()
    local vol = mp.get_property_number("volume") or 0
    local mute = mp.get_property_bool("mute")
    
    local icon = "🔊"
    if mute or vol == 0 then
        icon = "🔇"
    elseif vol < 30 then
        icon = "🔈"
    elseif vol < 70 then
        icon = "🔉"
    end
    
    if mute then
        notify(icon .. " Mudo", 1)
    else
        notify(string.format("%s Volume: %d%%", icon, vol), 1)
    end
end

-- Speed com ícone
function show_speed()
    local speed = mp.get_property_number("speed") or 1
    local icon = "⏩"
    if speed < 1 then
        icon = "⏪"
    elseif speed == 1 then
        icon = "▶️"
    end
    notify(string.format("%s Velocidade: %.2fx", icon, speed), 1)
end

-- ═══════════════════════════════════════════════════════════════════════════════
--  ATALHOS CUSTOMIZADOS
-- ═══════════════════════════════════════════════════════════════════════════════

-- Ctrl+I = Info do vídeo
mp.add_key_binding("ctrl+i", "goanime-info", show_video_info)

-- Mostrar logo novamente
mp.add_key_binding("ctrl+l", "goanime-logo", show_goanime_logo)

-- ═══════════════════════════════════════════════════════════════════════════════
--  EVENTOS
-- ═══════════════════════════════════════════════════════════════════════════════

-- Quando arquivo carrega
mp.register_event("file-loaded", function()
    msg.info("GoAnime Player: Arquivo carregado")
    
    -- Mostra info após o logo
    if config.show_filename_on_load then
        mp.add_timeout(config.logo_duration + 0.5, function()
            local title = mp.get_property("media-title") or mp.get_property("filename")
            if #title > 80 then
                title = string.sub(title, 1, 77) .. "..."
            end
            mp.osd_message("🎬 " .. title, 2)
        end)
    end
end)

-- Observer de volume
mp.observe_property("volume", "number", function(name, value)
    -- Não mostra no início
    if mp.get_property_number("time-pos") and mp.get_property_number("time-pos") > 1 then
        -- Volume já é mostrado pelo OSD padrão, não duplicar
    end
end)

-- Observer de velocidade
mp.observe_property("speed", "number", function(name, value)
    if mp.get_property_number("time-pos") and mp.get_property_number("time-pos") > 1 then
        -- show_speed() -- Descomentar se quiser OSD customizado
    end
end)

-- ═══════════════════════════════════════════════════════════════════════════════
--  INICIALIZAÇÃO
-- ═══════════════════════════════════════════════════════════════════════════════

mp.register_event("start-file", function()
    msg.info("═══════════════════════════════════════")
    msg.info("  🎬 GoAnime Player 4K")
    msg.info("  Upscaling de Alta Qualidade")
    msg.info("═══════════════════════════════════════")
    
    -- Mostra logo na primeira reprodução
    mp.add_timeout(0.5, show_goanime_logo)
end)

msg.info("GoAnime Player script loaded!")
