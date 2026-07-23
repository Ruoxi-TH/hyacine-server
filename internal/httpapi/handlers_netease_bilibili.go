package httpapi

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hyacine-go-server/internal/music/netease"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type requestBody struct {
	Offset     int    `json:"offset"`
	Cookie     string `json:"cookie"`
	Keywords   string `json:"keywords"`
	Limit      int    `json:"limit"`
	ID         int64  `json:"-"`
	BilibiliID string `json:"-"`
	CID        string `json:"cid"`
	Level      string `json:"level"`
	Name       string `json:"name"`
}

// The existing mobile client sends a numeric Netease ID and a string Bilibili BV ID.
func (b *requestBody) UnmarshalJSON(data []byte) error {
	var raw struct {
		Offset   int             `json:"offset"`
		Cookie   string          `json:"cookie"`
		Keywords string          `json:"keywords"`
		Limit    int             `json:"limit"`
		ID       json.RawMessage `json:"id"`
		CID      string          `json:"cid"`
		Level    string          `json:"level"`
		Name     string          `json:"name"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	b.Offset, b.Cookie, b.Keywords, b.Limit, b.CID, b.Level, b.Name = raw.Offset, raw.Cookie, raw.Keywords, raw.Limit, raw.CID, raw.Level, raw.Name
	if len(raw.ID) == 0 || string(raw.ID) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw.ID, &b.ID); err == nil {
		return nil
	}
	if err := json.Unmarshal(raw.ID, &b.BilibiliID); err != nil {
		return errors.New("id must be a number or string")
	}
	return nil
}

func (s *App) health(w http.ResponseWriter, _ *http.Request) {
	capabilities := map[string]bool{
		"qr": true, "profile": true, "dailySongs": true, "playlists": true,
		"recommendations": true, "search": true, "createPlaylist": true, "lyrics": true,
	}
	if s.directNetease != nil {
		capabilities = map[string]bool{
			"qr": true, "profile": true, "dailySongs": true, "playlists": true,
			"recommendations": true, "search": true, "createPlaylist": true, "lyrics": true,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "timestamp": time.Now().UTC().Format(time.RFC3339), "netease": map[string]any{"direct": s.directNetease != nil, "capabilities": capabilities}})
}
func (s *App) neteaseQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if s.directNetease != nil {
		key, qrURL, err := s.directNetease.CreateQR(r.Context())
		if err != nil {
			providerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"key": key, "qrUrl": qrURL})
		return
	}
	keyResp, err := s.providerGet("/login/qr/key?timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), "")
	if err != nil {
		providerError(w, err)
		return
	}
	var key struct {
		Data struct {
			Unikey string `json:"unikey"`
		} `json:"data"`
	}
	if json.Unmarshal(keyResp, &key) != nil || key.Data.Unikey == "" {
		providerError(w, errors.New("provider returned no QR key"))
		return
	}
	payload, err := s.providerGet("/login/qr/create?key="+url.QueryEscape(key.Data.Unikey)+"&qrimg=true&timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), "")
	if err != nil {
		providerError(w, err)
		return
	}
	var qr struct {
		Data struct {
			QRURL string `json:"qrurl"`
		} `json:"data"`
	}
	if json.Unmarshal(payload, &qr) != nil || qr.Data.QRURL == "" {
		providerError(w, errors.New("provider returned no QR URL"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"key": key.Data.Unikey, "qrUrl": qr.Data.QRURL})
}
func (s *App) neteaseQRPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/api/v1/music-sources/netease/qr/")
	if s.directNetease != nil {
		result, err := s.directNetease.CheckQR(r.Context(), key)
		if err != nil {
			providerError(w, err)
			return
		}
		payload := map[string]string{"status": result.Status, "message": result.Message}
		if result.Cookie != "" {
			payload["cookie"] = result.Cookie
		}
		writeJSON(w, http.StatusOK, payload)
		return
	}
	data, err := s.providerGet("/login/qr/check?key="+url.QueryEscape(key)+"&timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), "")
	if err != nil {
		providerError(w, err)
		return
	}
	var result struct {
		Code    int    `json:"code"`
		Cookie  string `json:"cookie"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(data, &result)
	if result.Code == 803 && result.Cookie != "" {
		writeJSON(w, 200, map[string]string{"status": "confirmed", "cookie": result.Cookie})
		return
	}
	if result.Code == 800 {
		writeJSON(w, 200, map[string]string{"status": "expired", "message": result.Message})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "pending", "message": result.Message})
}
func (s *App) neteaseProfile(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if s.directNetease != nil {
		profile, err := s.directNetease.Profile(r.Context(), desktopCookie(body.Cookie))
		if err != nil {
			providerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"userId": profile.UserID, "nickname": profile.Nickname, "avatarUrl": httpsURL(profile.AvatarURL)})
		return
	}
	data, err := s.providerGet("/user/account?timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), body.Cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	var result struct {
		Account struct {
			ID int64 `json:"id"`
		} `json:"account"`
		Profile struct {
			UserID    int64  `json:"userId"`
			Nickname  string `json:"nickname"`
			AvatarURL string `json:"avatarUrl"`
		} `json:"profile"`
	}
	_ = json.Unmarshal(data, &result)
	id := result.Profile.UserID
	if id == 0 {
		id = result.Account.ID
	}
	if id == 0 {
		providerError(w, errors.New("Netease account is unavailable"))
		return
	}
	writeJSON(w, 200, map[string]any{"userId": id, "nickname": result.Profile.Nickname, "avatarUrl": httpsURL(result.Profile.AvatarURL)})
}
func (s *App) neteaseRecommendations(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if s.directNetease != nil {
		playlists, err := s.directNetease.Recommendations(r.Context(), desktopCookie(body.Cookie))
		if err != nil {
			providerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, playlistResponse(playlists))
		return
	}
	s.convertPlaylists(w, "/top/playlist?cat=%E5%85%A8%E9%83%A8&order=hot&limit=100&offset=0&timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), body.Cookie, "playlists")
}
func (s *App) neteasePlaylists(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if s.directNetease != nil {
		playlists, err := s.directNetease.Playlists(r.Context(), desktopCookie(body.Cookie))
		if err != nil {
			providerError(w, err)
			return
		}
		out := make([]map[string]any, 0, len(playlists))
		for _, playlist := range playlists {
			out = append(out, map[string]any{"id": playlist.ID, "name": playlist.Name, "coverUrl": cover(playlist.CoverURL), "playCount": playlist.PlayCount, "trackCount": playlist.TrackCount, "description": playlist.Description})
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	account, err := s.providerGet("/user/account?timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), body.Cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	var a struct {
		Account struct {
			ID int64 `json:"id"`
		} `json:"account"`
		Profile struct {
			UserID int64 `json:"userId"`
		} `json:"profile"`
	}
	_ = json.Unmarshal(account, &a)
	id := a.Profile.UserID
	if id == 0 {
		id = a.Account.ID
	}
	if id == 0 {
		providerError(w, errors.New("Netease account is unavailable"))
		return
	}
	s.convertPlaylists(w, "/user/playlist?uid="+strconv.FormatInt(id, 10)+"&timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), body.Cookie, "playlist")
}
func (s *App) convertPlaylists(w http.ResponseWriter, path, cookie, key string) {
	data, err := s.providerGet(path, cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		providerError(w, errors.New("invalid provider response"))
		return
	}
	var items []map[string]any
	_ = json.Unmarshal(raw[key], &items)
	out := make([]map[string]any, 0, len(items))
	for _, x := range items {
		id := number(x["id"])
		name, _ := x["name"].(string)
		if id == 0 || name == "" {
			continue
		}
		image, _ := x["picUrl"].(string)
		if image == "" {
			image, _ = x["coverImgUrl"].(string)
		}
		pc := number(x["playcount"])
		if pc == 0 {
			pc = number(x["playCount"])
		}
		out = append(out, map[string]any{"id": id, "name": name, "coverUrl": cover(image), "playCount": pc, "trackCount": number(x["trackCount"]), "description": str(x["copywriter"], str(x["description"], ""))})
	}
	writeJSON(w, 200, out)
}
func (s *App) neteaseDailySongs(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if s.directNetease != nil {
		songs, err := s.directNetease.DailySongs(r.Context(), desktopCookie(body.Cookie))
		if err != nil {
			providerError(w, err)
			return
		}
		out := make([]map[string]any, 0, len(songs))
		for _, song := range songs {
			out = append(out, map[string]any{"id": song.ID, "title": song.Title, "artists": song.Artists, "coverUrl": cover(song.CoverURL), "durationMs": song.DurationMS})
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	data, err := s.providerGet("/recommend/songs?timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), body.Cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	var raw struct {
		Offset   int             `json:"offset"`
		Data struct {
			DailySongs []map[string]any `json:"dailySongs"`
		} `json:"data"`
	}
	_ = json.Unmarshal(data, &raw)
	writeJSON(w, http.StatusOK, tracks(raw.Data.DailySongs))
}
func (s *App) neteaseLyrics(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if body.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "song id is required"})
		return
	}
	if s.directNetease != nil {
		lyrics, err := s.directNetease.Lyrics(r.Context(), body.ID, desktopCookie(body.Cookie))
		if err != nil {
			providerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"lyric": lyrics.Text, "translation": lyrics.Translation})
		return
	}
	data, err := s.providerGet("/lyric?id="+strconv.FormatInt(body.ID, 10), body.Cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	var raw struct {
		Offset   int             `json:"offset"`
		Lrc struct {
			Lyric string `json:"lyric"`
		} `json:"lrc"`
		TLyric struct {
			Lyric string `json:"lyric"`
		} `json:"tlyric"`
	}
	if json.Unmarshal(data, &raw) != nil {
		providerError(w, errors.New("invalid lyric response"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"lyric": raw.Lrc.Lyric, "translation": raw.TLyric.Lyric})
}

func (s *App) neteaseComments(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if body.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "song id is required"})
		return
	}
	limit := body.Limit
	if limit <= 0 {
		limit = 30
	}
	if s.directNetease != nil {
		page, err := s.directNetease.Comments(r.Context(), body.ID, limit, body.Offset, desktopCookie(body.Cookie))
		if err != nil {
			providerError(w, err)
			return
		}
		items := make([]map[string]any, 0, len(page.Comments))
		for _, item := range page.Comments {
			items = append(items, map[string]any{"id": item.ID, "nickname": item.Nickname, "avatarUrl": cover(item.AvatarURL), "content": item.Content, "time": item.Time, "timeText": item.TimeText, "likedCount": item.LikedCount, "location": item.Location})
		}
		writeJSON(w, http.StatusOK, map[string]any{"total": page.Total, "more": page.More, "comments": items})
		return
	}
	data, err := s.providerGet("/comment/music?id="+strconv.FormatInt(body.ID, 10)+"&limit="+strconv.Itoa(limit), body.Cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	var payload any
	if json.Unmarshal(data, &payload) != nil {
		providerError(w, errors.New("invalid comments response"))
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *App) neteaseSearch(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if s.directNetease != nil {
		results, err := s.directNetease.Search(r.Context(), body.Keywords, body.Limit, desktopCookie(body.Cookie))
		if err != nil {
			providerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, trackResponse(results))
		return
	}
	limit := body.Limit
	if limit <= 0 {
		limit = 20
	}
	path := "/cloudsearch?keywords=" + url.QueryEscape(body.Keywords) + "&limit=" + strconv.Itoa(limit) + "&timestamp=" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	data, err := s.providerGet(path, body.Cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	var raw struct {
		Offset   int             `json:"offset"`
		Result struct {
			Songs []map[string]any `json:"songs"`
		} `json:"result"`
	}
	_ = json.Unmarshal(data, &raw)
	writeJSON(w, 200, tracks(raw.Result.Songs))
}
func (s *App) neteasePlayURL(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if body.ID <= 0 {
		writeJSON(w, 400, map[string]string{"message": "id is required"})
		return
	}
	cookie := desktopCookie(body.Cookie)
	levels := []string{body.Level, "exhigh", "higher", "standard"}
	if s.directNetease != nil {
		for _, level := range levels {
			if level == "" {
				continue
			}
			mediaURL, br, err := s.directNetease.PlayURL(r.Context(), body.ID, level, cookie)
			if err == nil && mediaURL != "" {
				s.createStreamResponse(w, mediaURL, br, cookie)
				return
			}
		}
		providerError(w, errors.New("Failed to get Netease play URL"))
		return
	}
	seen := make(map[string]bool)
	for _, level := range levels {
		if level == "" || seen[level] {
			continue
		}
		seen[level] = true
		data, err := s.providerGet("/song/url/v1?id="+strconv.FormatInt(body.ID, 10)+"&level="+url.QueryEscape(level)+"&encodeType=mp3&timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), cookie)
		if err != nil {
			continue
		}
		if mediaURL, br := parseNeteasePlayURL(data); mediaURL != "" {
			s.createStreamResponse(w, mediaURL, br, cookie)
			return
		}
	}
	data, err := s.providerGet("/song/url?id="+strconv.FormatInt(body.ID, 10)+"&timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), cookie)
	if err == nil {
		if mediaURL, br := parseNeteasePlayURL(data); mediaURL != "" {
			s.createStreamResponse(w, mediaURL, br, cookie)
			return
		}
	}
	providerError(w, errors.New("Failed to get Netease play URL"))
}
func parseNeteasePlayURL(data []byte) (string, int) {
	var raw struct {
		Offset   int             `json:"offset"`
		Data []struct {
			URL  string `json:"url"`
			BR   int    `json:"br"`
			Code int    `json:"code"`
		} `json:"data"`
	}
	if json.Unmarshal(data, &raw) != nil || len(raw.Data) == 0 || raw.Data[0].Code != 200 {
		return "", 0
	}
	return raw.Data[0].URL, raw.Data[0].BR
}

func (s *App) createStreamResponse(w http.ResponseWriter, mediaURL string, br int, cookie string) {
	token := s.streams.Create(mediaURL, cookie)
	// Return a path relative to /api/v1 so mobile clients that prefix apiBase
	// do not produce /api/v1/api/v1/... stream URLs.
	writeJSON(w, http.StatusOK, map[string]any{"url": "/music-sources/netease/stream/" + token, "br": br})
}

func (s *App) neteaseStream(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/api/v1/music-sources/netease/stream/")
	item, found := s.streams.Get(token)
	if !found {
		writeJSON(w, 404, map[string]string{"message": "Netease stream has expired"})
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, item.URL, nil)
	if err != nil {
		providerError(w, err)
		return
	}
	req.Header.Set("Referer", "https://music.163.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) NeteaseMusicDesktop/2.3.17.1034")
	req.Header.Set("Accept", "*/*")
	if item.Cookie != "" {
		req.Header.Set("Cookie", item.Cookie)
	}
	if v := r.Header.Get("Range"); v != "" {
		req.Header.Set("Range", v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		providerError(w, err)
		return
	}
	defer resp.Body.Close()
	for _, h := range []string{"Accept-Ranges", "Content-Length", "Content-Range", "Content-Type"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
func (s *App) neteasePlaylistDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID     int64  `json:"id"`
		Cookie string `json:"cookie"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.ID <= 0 {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	tracks, err := s.directNetease.PlaylistDetail(r.Context(), body.ID, desktopCookie(body.Cookie))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(tracks))
	for _, t := range tracks {
		artists := make([]string, 0, len(t.Artists))
		for _, a := range t.Artists {
			artists = append(artists, a)
		}
		out = append(out, map[string]any{"id": t.ID, "title": t.Title, "artists": artists, "coverUrl": cover(t.CoverURL), "durationMs": t.DurationMS})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *App) neteaseCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		writeJSON(w, 400, map[string]string{"message": "name is required"})
		return
	}
	if s.directNetease != nil {
		playlist, err := s.directNetease.CreatePlaylist(r.Context(), body.Name, desktopCookie(body.Cookie))
		if err != nil {
			providerError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, playlistResponse([]netease.Playlist{playlist})[0])
		return
	}
	data, err := s.providerGet("/playlist/create?name="+url.QueryEscape(body.Name)+"&timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), body.Cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	var raw struct {
		Offset   int             `json:"offset"`
		Playlist map[string]any `json:"playlist"`
	}
	_ = json.Unmarshal(data, &raw)
	x := raw.Playlist
	writeJSON(w, 201, map[string]any{"id": number(x["id"]), "name": str(x["name"], ""), "coverUrl": cover(str(x["coverImgUrl"], "")), "playCount": number(x["playCount"]), "trackCount": number(x["trackCount"]), "description": str(x["description"], "")})
}
func (s *App) neteaseDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if body.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "id is required"})
		return
	}
	if s.directNetease != nil {
		if err := s.directNetease.DeletePlaylist(r.Context(), body.ID, desktopCookie(body.Cookie)); err != nil {
			providerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
		return
	}
	if _, err := s.providerGet("/playlist/delete?id="+strconv.FormatInt(body.ID, 10)+"&timestamp="+strconv.FormatInt(time.Now().UnixMilli(), 10), body.Cookie); err != nil {
		providerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (s *App) neteaseToggleFavorite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cookie string `json:"cookie"`
		ID     int64  `json:"id"`
		Remove bool   `json:"remove"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "valid song id is required"})
		return
	}
	if s.directNetease == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"message": "favorites require direct Netease mode"})
		return
	}
	playlists, err := s.directNetease.Playlists(r.Context(), desktopCookie(body.Cookie))
	if err != nil {
		providerError(w, err)
		return
	}
	var playlistID int64
	for _, playlist := range playlists {
		if playlist.Name == "收藏风堇音乐" {
			playlistID = playlist.ID
			break
		}
	}
	if playlistID == 0 {
		created, createErr := s.directNetease.CreatePlaylist(r.Context(), "收藏风堇音乐", desktopCookie(body.Cookie))
		if createErr != nil {
			providerError(w, createErr)
			return
		}
		playlistID = created.ID
	}
	operation := "add"
	if body.Remove {
		operation = "del"
	}
	if err := s.directNetease.ManipulatePlaylistTracks(r.Context(), playlistID, body.ID, operation, desktopCookie(body.Cookie)); err != nil {
		providerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"favorite": !body.Remove, "playlistId": playlistID})
}

func (s *App) bilibiliValidateCookie(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if !strings.Contains(body.Cookie, "SESSDATA=") || !strings.Contains(body.Cookie, "bili_jct=") {
		writeJSON(w, http.StatusOK, map[string]bool{"valid": false})
		return
	}
	data, err := bilibiliGet("/x/web-interface/nav", body.Cookie)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]bool{"valid": false})
		return
	}
	var result struct {
		Code int `json:"code"`
		Data struct {
			IsLogin bool `json:"isLogin"`
		} `json:"data"`
	}
	_ = json.Unmarshal(data, &result)
	writeJSON(w, http.StatusOK, map[string]bool{"valid": result.Code == 0 && result.Data.IsLogin})
}

func (s *App) bilibiliSearch(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if !strings.Contains(body.Cookie, "SESSDATA=") {
		providerError(w, errors.New("请先在音乐源页绑定有效的哔哩哔哩账号"))
		return
	}
	limit := body.Limit
	if limit <= 0 || limit > 20 {
		limit = 20
	}
	path, err := bilibiliSignedPath("/x/web-interface/wbi/search/type", map[string]string{"keyword": body.Keywords, "search_type": "video", "page": "1", "page_size": strconv.Itoa(limit), "tids": "3"}, body.Cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	data, err := bilibiliGet(path, body.Cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	var raw struct {
		Offset   int             `json:"offset"`
		Code int `json:"code"`
		Data struct {
			Result []struct {
				BVID     string `json:"bvid"`
				Title    string `json:"title"`
				Author   string `json:"author"`
				Pic      string `json:"pic"`
				Duration string `json:"duration"`
				Type     string `json:"type"`
				TypeName string `json:"typename"`
			} `json:"result"`
		} `json:"data"`
	}
	_ = json.Unmarshal(data, &raw)
	if raw.Code != 0 {
		providerError(w, errors.New("Bilibili search is unavailable"))
		return
	}
	out := make([]map[string]any, 0, len(raw.Data.Result))
	for _, item := range raw.Data.Result {
		if item.BVID == "" || item.Title == "" || item.Type != "video" {
			continue
		}
		if item.TypeName != "" && !strings.Contains(item.TypeName, "音乐") {
			continue
		}
		pic := item.Pic
		if strings.HasPrefix(pic, "//") {
			pic = "https:" + pic
		}
		out = append(out, map[string]any{"id": item.BVID, "title": stripHTML(item.Title), "artists": []string{item.Author}, "coverUrl": pic, "duration": item.Duration, "source": "bilibili"})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *App) bilibiliPlayURL(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if !strings.Contains(body.Cookie, "SESSDATA=") {
		providerError(w, errors.New("请先在音乐源页绑定有效的哔哩哔哩账号"))
		return
	}
	bvid := strings.TrimSpace(body.BilibiliID)
	if bvid == "" {
		providerError(w, errors.New("Bilibili bvid is required"))
		return
	}
	cid := strings.TrimSpace(body.CID)
	if cid == "" {
		view, err := bilibiliGet("/x/web-interface/view?bvid="+url.QueryEscape(bvid), body.Cookie)
		if err != nil {
			providerError(w, err)
			return
		}
		var page struct {
			Code int `json:"code"`
			Data struct {
				CID int64 `json:"cid"`
			} `json:"data"`
		}
		_ = json.Unmarshal(view, &page)
		if page.Code != 0 || page.Data.CID == 0 {
			providerError(w, errors.New("Failed to resolve Bilibili cid"))
			return
		}
		cid = strconv.FormatInt(page.Data.CID, 10)
	}
	data, err := bilibiliGet("/x/player/playurl?bvid="+url.QueryEscape(bvid)+"&cid="+url.QueryEscape(cid)+"&fnver=0&qn=80&fnval=4048&fourk=1", body.Cookie)
	if err != nil {
		providerError(w, err)
		return
	}
	var play struct {
		Code int `json:"code"`
		Data struct {
			DURL []struct {
				URL string `json:"url"`
			} `json:"durl"`
			Dash struct {
				Audio []struct {
					ID        int      `json:"id"`
					BaseURL   string   `json:"baseUrl"`
					BackupURL []string `json:"backupUrl"`
				} `json:"audio"`
			} `json:"dash"`
		} `json:"data"`
	}
	_ = json.Unmarshal(data, &play)
	if play.Code != 0 {
		providerError(w, errors.New("Bilibili playurl is unavailable"))
		return
	}
	if len(play.Data.Dash.Audio) > 0 {
		a := play.Data.Dash.Audio[0]
		media := a.BaseURL
		if media == "" && len(a.BackupURL) > 0 {
			media = a.BackupURL[0]
		}
		if media != "" {
			token := s.streams.Create(media, body.Cookie)
			writeJSON(w, 200, map[string]any{"url": "/api/v1/music-sources/bilibili/stream/" + token, "quality": "dash_" + strconv.Itoa(a.ID), "cid": cid, "proxied": true})
			return
		}
	}
	if len(play.Data.DURL) > 0 && play.Data.DURL[0].URL != "" {
		token := s.streams.Create(play.Data.DURL[0].URL, body.Cookie)
		writeJSON(w, 200, map[string]any{"url": "/api/v1/music-sources/bilibili/stream/" + token, "quality": "durl", "cid": cid, "proxied": true})
		return
	}
}

func (s *App) bilibiliStream(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/api/v1/music-sources/bilibili/stream/")
	item, found := s.streams.Get(token)
	if !found {
		writeJSON(w, 404, map[string]string{"message": "Bilibili stream has expired"})
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, item.URL, nil)
	if err != nil {
		providerError(w, err)
		return
	}
	req.Header.Set("Referer", "https://www.bilibili.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	if v := r.Header.Get("Range"); v != "" {
		req.Header.Set("Range", v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		providerError(w, err)
		return
	}
	defer resp.Body.Close()
	for _, h := range []string{"Accept-Ranges", "Content-Length", "Content-Range", "Content-Type"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func bilibiliGet(path, cookie string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.bilibili.com"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", cookie)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("Bilibili returned HTTP " + strconv.Itoa(resp.StatusCode))
	}
	return data, nil
}

func bilibiliSignedPath(path string, params map[string]string, cookie string) (string, error) {
	data, err := bilibiliGet("/x/web-interface/nav", cookie)
	if err != nil {
		return "", err
	}
	var nav struct {
		Data struct {
			WBI struct {
				ImgURL string `json:"img_url"`
				SubURL string `json:"sub_url"`
			} `json:"wbi_img"`
		} `json:"data"`
	}
	if json.Unmarshal(data, &nav) != nil {
		return "", errors.New("invalid Bilibili WBI response")
	}
	fileKey := func(v string) string {
		parts := strings.Split(strings.TrimSuffix(v, ".png"), "/")
		return parts[len(parts)-1]
	}
	source := fileKey(nav.Data.WBI.ImgURL) + fileKey(nav.Data.WBI.SubURL)
	table := []int{46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49, 33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40, 61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11, 36, 20, 34, 44, 52}
	var mixin strings.Builder
	for _, i := range table {
		if i < len(source) {
			mixin.WriteByte(source[i])
		}
		if mixin.Len() == 32 {
			break
		}
	}
	values := url.Values{}
	for k, v := range params {
		values.Set(k, strings.NewReplacer("!", "", "'", "", "(", "", ")", "", "*", "").Replace(v))
	}
	values.Set("wts", strconv.FormatInt(time.Now().Unix(), 10))
	query := values.Encode()
	sum := md5.Sum([]byte(query + mixin.String()))
	return path + "?" + query + "&w_rid=" + hex.EncodeToString(sum[:]), nil
}

func stripHTML(s string) string {
	for {
		start := strings.Index(s, "<")
		if start < 0 {
			return s
		}
		end := strings.Index(s[start:], ">")
		if end < 0 {
			return s
		}
		s = s[:start] + s[start+end+1:]
	}
}

func (s *App) providerGet(path, cookie string) ([]byte, error) {
	if s.netease == nil {
		return nil, errors.New("this Netease endpoint still requires NETEASE_API_BASE; direct mode currently supports playback")
	}
	return s.netease.Get(context.Background(), path, cookie)
}
func decodeBody(w http.ResponseWriter, r *http.Request) (requestBody, bool) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return requestBody{}, false
	}
	var b requestBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeJSON(w, 400, map[string]string{"message": "invalid JSON request"})
		return b, false
	}
	return b, true
}
func tracks(songs []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(songs))
	for _, s := range songs {
		id := number(s["id"])
		title := str(s["name"], "")
		if id == 0 || title == "" {
			continue
		}
		artists := []string{}
		for _, key := range []string{"ar", "artists"} {
			if values, ok := s[key].([]any); ok {
				for _, v := range values {
					if m, ok := v.(map[string]any); ok && str(m["name"], "") != "" {
						artists = append(artists, str(m["name"], ""))
					}
				}
			}
		}
		album := ""
		image := str(s["picUrl"], "")
		for _, key := range []string{"al", "album"} {
			if m, ok := s[key].(map[string]any); ok {
				album = str(m["name"], album)
				for _, field := range []string{"picUrl", "blurPicUrl", "pic_str"} {
					if image == "" {
						image = str(m[field], image)
					}
				}
				if image == "" {
					if picID := number(m["pic"]); picID > 0 {
						image = "https://music.163.com/api/img/blob/" + strconv.FormatInt(picID, 10)
					}
				}
			}
		}
		out = append(out, map[string]any{"id": id, "title": title, "artists": artists, "album": album, "coverUrl": cover(image), "durationMs": number(s["dt"]), "source": "netease"})
	}
	return out
}
func trackResponse(items []netease.Track) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{"id": item.ID, "title": item.Title, "artists": item.Artists, "coverUrl": cover(item.CoverURL), "durationMs": item.DurationMS, "source": "netease"})
	}
	return out
}
func playlistResponse(items []netease.Playlist) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{"id": item.ID, "name": item.Name, "coverUrl": cover(item.CoverURL), "playCount": item.PlayCount, "trackCount": item.TrackCount, "description": item.Description})
	}
	return out
}
func httpsURL(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "//") {
		v = "https:" + v
	}
	if strings.HasPrefix(v, "http://") {
		v = "https://" + strings.TrimPrefix(v, "http://")
	}
	return v
}

func cover(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "//") {
		v = "https:" + v
	}
	if strings.HasPrefix(v, "http://") {
		// Android cleartext blocking commonly drops http:// music.126.net covers.
		v = "https://" + strings.TrimPrefix(v, "http://")
	}
	if strings.HasPrefix(v, "https://") {
		if strings.Contains(v, "param=") {
			return v
		}
		if strings.Contains(v, "?") {
			return v + "&param=400y400"
		}
		return v + "?param=400y400"
	}
	return v
}
func number(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}
func str(v any, fallback string) string {
	if x, ok := v.(string); ok {
		return x
	}
	return fallback
}
func desktopCookie(cookie string) string {
	if !strings.Contains(cookie, "os=") {
		if cookie != "" {
			cookie += "; "
		}
		cookie += "os=pc"
	}
	return cookie
}
func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
}
func providerError(w http.ResponseWriter, err error) {
	log.Printf("provider error: %v", err)
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"message": err.Error()})
}
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
