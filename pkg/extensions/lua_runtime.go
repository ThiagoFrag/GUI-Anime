package extensions

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	lua "github.com/yuin/gopher-lua"
	luajson "layeh.com/gopher-json"
)

// LuaExtension implementa ExtensionSource usando scripts Lua
type LuaExtension struct {
	state  *lua.LState
	info   ExtensionInfo
	script string
}

// Opções de configuração para o runtime Lua
var (
	luaHTTPTimeout = 15 * time.Second
	luaHTTPClient  = &http.Client{Timeout: luaHTTPTimeout}
)

// NewLuaExtension cria uma nova extension a partir de um script Lua
func NewLuaExtension(script string) (*LuaExtension, error) {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: false,
	})

	// Remove bibliotecas perigosas
	L.SetGlobal("os", lua.LNil)
	L.SetGlobal("io", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("dofile", lua.LNil)

	// Registra JSON encoder/decoder
	luajson.Preload(L)
	L.DoString(`json = require("json")`)

	// Registra funções HTTP seguras
	L.SetGlobal("http_get", L.NewFunction(luaHTTPGet))
	L.SetGlobal("http_post", L.NewFunction(luaHTTPPost))
	L.SetGlobal("url_encode", L.NewFunction(luaURLEncode))

	// Registra funções de parsing HTML
	L.SetGlobal("parse_html", L.NewFunction(luaParseHTML))

	// Registra utilitários de string
	L.SetGlobal("trim", L.NewFunction(luaTrim))
	L.SetGlobal("split", L.NewFunction(luaSplit))
	L.SetGlobal("match", L.NewFunction(luaMatch))
	L.SetGlobal("match_all", L.NewFunction(luaMatchAll))

	// Executa o script
	if err := L.DoString(script); err != nil {
		L.Close()
		return nil, fmt.Errorf("erro ao executar script: %w", err)
	}

	// Extrai informações da extension
	ext := &LuaExtension{
		state:  L,
		script: script,
	}

	if err := ext.extractInfo(); err != nil {
		L.Close()
		return nil, err
	}

	return ext, nil
}

// Close libera recursos do runtime Lua
func (e *LuaExtension) Close() {
	if e.state != nil {
		e.state.Close()
	}
}

// GetInfo implementa ExtensionSource
func (e *LuaExtension) GetInfo() ExtensionInfo {
	return e.info
}

// Search implementa ExtensionSource
func (e *LuaExtension) Search(ctx context.Context, query string, page int, filters map[string]string) ([]AnimeEntry, bool, error) {
	L := e.state

	// Chama função Lua: search(query, page, filters) -> results, hasNext
	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("search"),
		NRet:    2,
		Protect: true,
	}, lua.LString(query), lua.LNumber(page), filtersToTable(L, filters)); err != nil {
		return nil, false, fmt.Errorf("erro ao chamar search: %w", err)
	}

	hasNext := L.ToBool(-1)
	L.Pop(1)

	results, err := tableToAnimeEntries(L, -1)
	L.Pop(1)

	return results, hasNext, err
}

// GetLatest implementa ExtensionSource
func (e *LuaExtension) GetLatest(ctx context.Context, page int) ([]AnimeEntry, bool, error) {
	L := e.state

	fn := L.GetGlobal("getLatest")
	if fn == lua.LNil {
		return nil, false, fmt.Errorf("função getLatest não implementada")
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    2,
		Protect: true,
	}, lua.LNumber(page)); err != nil {
		return nil, false, err
	}

	hasNext := L.ToBool(-1)
	L.Pop(1)

	results, err := tableToAnimeEntries(L, -1)
	L.Pop(1)

	return results, hasNext, err
}

// GetPopular implementa ExtensionSource
func (e *LuaExtension) GetPopular(ctx context.Context, page int) ([]AnimeEntry, bool, error) {
	L := e.state

	fn := L.GetGlobal("getPopular")
	if fn == lua.LNil {
		return nil, false, fmt.Errorf("função getPopular não implementada")
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    2,
		Protect: true,
	}, lua.LNumber(page)); err != nil {
		return nil, false, err
	}

	hasNext := L.ToBool(-1)
	L.Pop(1)

	results, err := tableToAnimeEntries(L, -1)
	L.Pop(1)

	return results, hasNext, err
}

// GetAnimeDetails implementa ExtensionSource
func (e *LuaExtension) GetAnimeDetails(ctx context.Context, url string) (*AnimeDetails, error) {
	L := e.state

	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("getAnimeDetails"),
		NRet:    1,
		Protect: true,
	}, lua.LString(url)); err != nil {
		return nil, err
	}

	result, err := tableToAnimeDetails(L, -1)
	L.Pop(1)

	return result, err
}

// GetEpisodes implementa ExtensionSource
func (e *LuaExtension) GetEpisodes(ctx context.Context, animeURL string) ([]Episode, error) {
	L := e.state

	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("getEpisodes"),
		NRet:    1,
		Protect: true,
	}, lua.LString(animeURL)); err != nil {
		return nil, err
	}

	result, err := tableToEpisodes(L, -1)
	L.Pop(1)

	return result, err
}

// GetVideoSources implementa ExtensionSource
func (e *LuaExtension) GetVideoSources(ctx context.Context, episodeURL string) ([]VideoSource, error) {
	L := e.state

	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("getVideoSources"),
		NRet:    1,
		Protect: true,
	}, lua.LString(episodeURL)); err != nil {
		return nil, err
	}

	result, err := tableToVideoSources(L, -1)
	L.Pop(1)

	return result, err
}

// --- Métodos privados ---

func (e *LuaExtension) extractInfo() error {
	L := e.state

	extTable := L.GetGlobal("Extension")
	if extTable == lua.LNil {
		return fmt.Errorf("extension table not found in script")
	}

	tbl, ok := extTable.(*lua.LTable)
	if !ok {
		return fmt.Errorf("extension is not a table")
	}

	e.info = ExtensionInfo{
		ID:         getStringField(tbl, "id"),
		Name:       getStringField(tbl, "name"),
		Version:    getStringField(tbl, "version"),
		Language:   getStringField(tbl, "language"),
		BaseURL:    getStringField(tbl, "baseUrl"),
		IconURL:    getStringField(tbl, "iconUrl"),
		Author:     getStringField(tbl, "author"),
		NSFW:       getBoolField(tbl, "nsfw"),
		HasLatest:  L.GetGlobal("getLatest") != lua.LNil,
		HasPopular: L.GetGlobal("getPopular") != lua.LNil,
		HasSearch:  L.GetGlobal("search") != lua.LNil,
	}

	if e.info.ID == "" {
		return fmt.Errorf("extension.id is required")
	}
	if e.info.Name == "" {
		e.info.Name = e.info.ID
	}
	if e.info.Version == "" {
		e.info.Version = "1.0.0"
	}

	return nil
}

// --- Funções Lua globais ---

func luaHTTPGet(L *lua.LState) int {
	urlStr := L.CheckString(1)
	headers := L.OptTable(2, nil)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	if headers != nil {
		headers.ForEach(func(k, v lua.LValue) {
			req.Header.Set(k.String(), v.String())
		})
	}

	resp, err := luaHTTPClient.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	html, _ := doc.Html()
	L.Push(lua.LString(html))
	return 1
}

func luaHTTPPost(L *lua.LState) int {
	urlStr := L.CheckString(1)
	body := L.CheckString(2)
	headers := L.OptTable(3, nil)

	req, err := http.NewRequest("POST", urlStr, strings.NewReader(body))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if headers != nil {
		headers.ForEach(func(k, v lua.LValue) {
			req.Header.Set(k.String(), v.String())
		})
	}

	resp, err := luaHTTPClient.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	html, _ := doc.Html()
	L.Push(lua.LString(html))
	return 1
}

func luaURLEncode(L *lua.LState) int {
	s := L.CheckString(1)
	L.Push(lua.LString(url.QueryEscape(s)))
	return 1
}

func luaParseHTML(L *lua.LState) int {
	html := L.CheckString(1)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Retorna um userdata que wrapa o documento
	ud := L.NewUserData()
	ud.Value = doc.Selection

	// Registra metatable com métodos
	mt := L.NewTypeMetatable("html_document")
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"select": htmlSelect,
		"text":   htmlText,
		"attr":   htmlAttr,
		"html":   htmlHTML,
		"each":   htmlEach,
		"first":  htmlFirst,
		"last":   htmlLast,
		"length": htmlLength,
	}))
	L.SetMetatable(ud, mt)

	L.Push(ud)
	return 1
}

func luaTrim(L *lua.LState) int {
	s := L.CheckString(1)
	L.Push(lua.LString(strings.TrimSpace(s)))
	return 1
}

func luaSplit(L *lua.LState) int {
	s := L.CheckString(1)
	sep := L.CheckString(2)
	parts := strings.Split(s, sep)

	tbl := L.NewTable()
	for i, p := range parts {
		tbl.RawSetInt(i+1, lua.LString(p))
	}
	L.Push(tbl)
	return 1
}

func luaMatch(L *lua.LState) int {
	s := L.CheckString(1)
	pattern := L.CheckString(2)

	// Usa strings.Contains para pattern simples ou regexp para complexo
	if strings.Contains(s, pattern) {
		L.Push(lua.LString(pattern))
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

func luaMatchAll(L *lua.LState) int {
	// Simplificado - retorna todas as ocorrências
	s := L.CheckString(1)
	pattern := L.CheckString(2)

	tbl := L.NewTable()
	idx := 1
	for _, match := range strings.Split(s, pattern) {
		if match != "" {
			tbl.RawSetInt(idx, lua.LString(match))
			idx++
		}
	}
	L.Push(tbl)
	return 1
}

// --- Funções HTML ---

func htmlSelect(L *lua.LState) int {
	ud := L.CheckUserData(1)
	selector := L.CheckString(2)

	sel, ok := ud.Value.(*goquery.Selection)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}

	newSel := sel.Find(selector)
	newUD := L.NewUserData()
	newUD.Value = newSel
	L.SetMetatable(newUD, L.GetTypeMetatable("html_document"))
	L.Push(newUD)
	return 1
}

func htmlText(L *lua.LState) int {
	ud := L.CheckUserData(1)
	sel, ok := ud.Value.(*goquery.Selection)
	if !ok {
		L.Push(lua.LString(""))
		return 1
	}
	L.Push(lua.LString(strings.TrimSpace(sel.Text())))
	return 1
}

func htmlAttr(L *lua.LState) int {
	ud := L.CheckUserData(1)
	attr := L.CheckString(2)

	sel, ok := ud.Value.(*goquery.Selection)
	if !ok {
		L.Push(lua.LString(""))
		return 1
	}

	val, _ := sel.Attr(attr)
	L.Push(lua.LString(val))
	return 1
}

func htmlHTML(L *lua.LState) int {
	ud := L.CheckUserData(1)
	sel, ok := ud.Value.(*goquery.Selection)
	if !ok {
		L.Push(lua.LString(""))
		return 1
	}
	html, _ := sel.Html()
	L.Push(lua.LString(html))
	return 1
}

func htmlEach(L *lua.LState) int {
	ud := L.CheckUserData(1)
	fn := L.CheckFunction(2)

	sel, ok := ud.Value.(*goquery.Selection)
	if !ok {
		return 0
	}

	sel.Each(func(i int, s *goquery.Selection) {
		itemUD := L.NewUserData()
		itemUD.Value = s
		L.SetMetatable(itemUD, L.GetTypeMetatable("html_document"))

		L.Push(fn)
		L.Push(lua.LNumber(i + 1))
		L.Push(itemUD)
		L.PCall(2, 0, nil)
	})

	return 0
}

func htmlFirst(L *lua.LState) int {
	ud := L.CheckUserData(1)
	sel, ok := ud.Value.(*goquery.Selection)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}

	newUD := L.NewUserData()
	newUD.Value = sel.First()
	L.SetMetatable(newUD, L.GetTypeMetatable("html_document"))
	L.Push(newUD)
	return 1
}

func htmlLast(L *lua.LState) int {
	ud := L.CheckUserData(1)
	sel, ok := ud.Value.(*goquery.Selection)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}

	newUD := L.NewUserData()
	newUD.Value = sel.Last()
	L.SetMetatable(newUD, L.GetTypeMetatable("html_document"))
	L.Push(newUD)
	return 1
}

func htmlLength(L *lua.LState) int {
	ud := L.CheckUserData(1)
	sel, ok := ud.Value.(*goquery.Selection)
	if !ok {
		L.Push(lua.LNumber(0))
		return 1
	}
	L.Push(lua.LNumber(sel.Length()))
	return 1
}

// --- Helpers de conversão ---

func getStringField(tbl *lua.LTable, key string) string {
	v := tbl.RawGetString(key)
	if s, ok := v.(lua.LString); ok {
		return string(s)
	}
	return ""
}

func getBoolField(tbl *lua.LTable, key string) bool {
	v := tbl.RawGetString(key)
	if b, ok := v.(lua.LBool); ok {
		return bool(b)
	}
	return false
}

func filtersToTable(L *lua.LState, filters map[string]string) *lua.LTable {
	tbl := L.NewTable()
	for k, v := range filters {
		tbl.RawSetString(k, lua.LString(v))
	}
	return tbl
}

func tableToAnimeEntries(L *lua.LState, idx int) ([]AnimeEntry, error) {
	v := L.Get(idx)
	if v == lua.LNil {
		return nil, nil
	}

	tbl, ok := v.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("esperado tabela, recebeu %T", v)
	}

	var entries []AnimeEntry
	tbl.ForEach(func(_, v lua.LValue) {
		if item, ok := v.(*lua.LTable); ok {
			entries = append(entries, AnimeEntry{
				Title:       getStringField(item, "title"),
				URL:         getStringField(item, "url"),
				Image:       getStringField(item, "image"),
				Description: getStringField(item, "description"),
				Status:      getStringField(item, "status"),
			})
		}
	})

	return entries, nil
}

func tableToAnimeDetails(L *lua.LState, idx int) (*AnimeDetails, error) {
	v := L.Get(idx)
	if v == lua.LNil {
		return nil, nil
	}

	tbl, ok := v.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("esperado tabela, recebeu %T", v)
	}

	details := &AnimeDetails{
		Title:          getStringField(tbl, "title"),
		AlternateTitle: getStringField(tbl, "alternateTitle"),
		URL:            getStringField(tbl, "url"),
		Image:          getStringField(tbl, "image"),
		Banner:         getStringField(tbl, "banner"),
		Description:    getStringField(tbl, "description"),
		Status:         getStringField(tbl, "status"),
		Studio:         getStringField(tbl, "studio"),
	}

	if year := tbl.RawGetString("year"); year != lua.LNil {
		if n, ok := year.(lua.LNumber); ok {
			details.Year = int(n)
		}
	}

	if rating := tbl.RawGetString("rating"); rating != lua.LNil {
		if n, ok := rating.(lua.LNumber); ok {
			details.Rating = float64(n)
		}
	}

	if genres := tbl.RawGetString("genres"); genres != lua.LNil {
		if genresTbl, ok := genres.(*lua.LTable); ok {
			genresTbl.ForEach(func(_, v lua.LValue) {
				if s, ok := v.(lua.LString); ok {
					details.Genres = append(details.Genres, string(s))
				}
			})
		}
	}

	return details, nil
}

func tableToEpisodes(L *lua.LState, idx int) ([]Episode, error) {
	v := L.Get(idx)
	if v == lua.LNil {
		return nil, nil
	}

	tbl, ok := v.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("esperado tabela, recebeu %T", v)
	}

	var episodes []Episode
	tbl.ForEach(func(_, v lua.LValue) {
		if item, ok := v.(*lua.LTable); ok {
			ep := Episode{
				Title:     getStringField(item, "title"),
				URL:       getStringField(item, "url"),
				Thumbnail: getStringField(item, "thumbnail"),
				Filler:    getBoolField(item, "filler"),
			}

			if num := item.RawGetString("number"); num != lua.LNil {
				if n, ok := num.(lua.LNumber); ok {
					ep.Number = int(n)
				}
			}

			episodes = append(episodes, ep)
		}
	})

	return episodes, nil
}

func tableToVideoSources(L *lua.LState, idx int) ([]VideoSource, error) {
	v := L.Get(idx)
	if v == lua.LNil {
		return nil, nil
	}

	tbl, ok := v.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("esperado tabela, recebeu %T", v)
	}

	var sources []VideoSource
	tbl.ForEach(func(_, v lua.LValue) {
		if item, ok := v.(*lua.LTable); ok {
			src := VideoSource{
				URL:     getStringField(item, "url"),
				Quality: getStringField(item, "quality"),
				Format:  getStringField(item, "format"),
				Server:  getStringField(item, "server"),
			}

			// Headers
			if headers := item.RawGetString("headers"); headers != lua.LNil {
				if headersTbl, ok := headers.(*lua.LTable); ok {
					src.Headers = make(map[string]string)
					headersTbl.ForEach(func(k, v lua.LValue) {
						src.Headers[k.String()] = v.String()
					})
				}
			}

			// Subtitles
			if subs := item.RawGetString("subtitles"); subs != lua.LNil {
				if subsTbl, ok := subs.(*lua.LTable); ok {
					subsTbl.ForEach(func(_, v lua.LValue) {
						if subItem, ok := v.(*lua.LTable); ok {
							src.Subtitles = append(src.Subtitles, Subtitle{
								URL:      getStringField(subItem, "url"),
								Language: getStringField(subItem, "language"),
								Label:    getStringField(subItem, "label"),
								Format:   getStringField(subItem, "format"),
								Default:  getBoolField(subItem, "default"),
							})
						}
					})
				}
			}

			sources = append(sources, src)
		}
	})

	return sources, nil
}
