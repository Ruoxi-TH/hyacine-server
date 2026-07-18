package netease

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	ncmapi "github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
	ncmlog "github.com/chaunsin/netease-cloud-music/pkg/log"
)

var ErrNoPlayableURL = errors.New("Netease returned no playable URL")

type Profile struct {
	UserID    int64
	Nickname  string
	AvatarURL string
}
type Track struct {
	ID         int64
	Title      string
	Artists    []string
	CoverURL   string
	DurationMS int64
}
type Playlist struct {
	ID          int64
	Name        string
	CoverURL    string
	PlayCount   int64
	TrackCount  int64
	Description string
}
type QRStatus struct {
	Status  string
	Cookie  string
	Message string
}

// DirectClient creates a client and cookie jar for each request, preventing
// third-party credentials from crossing user boundaries.
type DirectClient struct{ timeout time.Duration }

var ensureNeteaseLoggerOnce sync.Once

func NewDirectClient(timeout time.Duration) *DirectClient {
	ensureNeteaseLogger()
	return &DirectClient{timeout: timeout}
}

// The upstream library panics if package-level log.Default is nil when Request
// emits debug messages. Initialize a quiet stdout logger once per process.
func ensureNeteaseLogger() {
	ensureNeteaseLoggerOnce.Do(func() {
		if ncmlog.Default != nil {
			return
		}
		ncmlog.Default = ncmlog.New(&ncmlog.Config{
			App:    "hyacine-server",
			Format: "text",
			Level:  "error",
			Stdout: true,
		})
	})
}

func (c *DirectClient) PlayURL(ctx context.Context, id int64, level, rawCookie string) (string, int, error) {
	client, err := c.clientForCookie(rawCookie)
	if err != nil {
		return "", 0, err
	}
	defer client.Close(ctx)
	response, err := weapi.New(client).SongPlayerV1(ctx, &weapi.SongPlayerV1Req{Ids: types.IntsString{id}, Level: types.Level(level), EncodeType: "mp3"})
	if err != nil {
		return "", 0, err
	}
	if response.Code != http.StatusOK || len(response.Data) == 0 || response.Data[0].Url == "" {
		return "", 0, ErrNoPlayableURL
	}
	return response.Data[0].Url, int(response.Data[0].Br), nil
}

func (c *DirectClient) Profile(ctx context.Context, rawCookie string) (Profile, error) {
	client, err := c.clientForCookie(rawCookie)
	if err != nil {
		return Profile{}, err
	}
	defer client.Close(ctx)
	response, err := weapi.New(client).GetUserInfo(ctx, &weapi.GetUserInfoReq{})
	if err != nil {
		return Profile{}, err
	}
	if response.Code != http.StatusOK || response.Profile == nil {
		return Profile{}, errors.New("Netease account is unavailable")
	}
	return Profile{UserID: response.Profile.UserId, Nickname: response.Profile.Nickname, AvatarURL: response.Profile.AvatarUrl}, nil
}

func (c *DirectClient) DailySongs(ctx context.Context, rawCookie string) ([]Track, error) {
	client, err := c.clientForCookie(rawCookie)
	if err != nil {
		return nil, err
	}
	defer client.Close(ctx)
	response, err := weapi.New(client).RecommendSongs(ctx, &weapi.RecommendSongsReq{})
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		return nil, errors.New("Netease daily recommendations are unavailable")
	}
	out := make([]Track, 0, len(response.Data.DailySongs))
	for _, song := range response.Data.DailySongs {
		artists := make([]string, 0, len(song.Ar))
		for _, artist := range song.Ar {
			artists = append(artists, artist.Name)
		}
		out = append(out, Track{ID: song.Id, Title: song.Name, Artists: artists, CoverURL: song.Al.PicUrl, DurationMS: song.Dt})
	}
	return out, nil
}

func (c *DirectClient) Playlists(ctx context.Context, rawCookie string) ([]Playlist, error) {
	profile, err := c.Profile(ctx, rawCookie)
	if err != nil {
		return nil, err
	}
	client, err := c.clientForCookie(rawCookie)
	if err != nil {
		return nil, err
	}
	defer client.Close(ctx)
	response, err := weapi.New(client).Playlist(ctx, &weapi.PlaylistReq{Uid: strconv.FormatInt(profile.UserID, 10), Limit: "1000"})
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		return nil, errors.New("Netease playlists are unavailable")
	}
	return playlists(response.Playlist), nil
}

func (c *DirectClient) Search(ctx context.Context, keywords string, limit int, rawCookie string) ([]Track, error) {
	if strings.TrimSpace(keywords) == "" {
		return []Track{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 30
	}
	var response struct {
		Code   int64 `json:"code"`
		Result struct {
			Songs []song `json:"songs"`
		} `json:"result"`
	}
	if err := c.weapiRequest(ctx, rawCookie, "https://music.163.com/weapi/cloudsearch/get/web", map[string]any{"s": keywords, "type": 1, "limit": limit, "offset": 0, "total": true}, &response); err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		return nil, errors.New("Netease search is unavailable")
	}
	out := make([]Track, 0, len(response.Result.Songs))
	for _, item := range response.Result.Songs {
		out = append(out, item.track())
	}
	return out, nil
}

func (c *DirectClient) Recommendations(ctx context.Context, rawCookie string) ([]Playlist, error) {
	var response struct {
		Code      int64      `json:"code"`
		Recommend []playlist `json:"recommend"`
	}
	if err := c.weapiRequest(ctx, rawCookie, "https://music.163.com/weapi/v1/discovery/recommend/resource", map[string]any{}, &response); err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		return nil, errors.New("Netease playlist recommendations are unavailable")
	}
	out := make([]Playlist, 0, len(response.Recommend))
	for _, item := range response.Recommend {
		out = append(out, item.playlist())
	}
	return out, nil
}

func (c *DirectClient) CreatePlaylist(ctx context.Context, name, rawCookie string) (Playlist, error) {
	if strings.TrimSpace(name) == "" {
		return Playlist{}, errors.New("playlist name is required")
	}
	var response struct {
		Code     int64    `json:"code"`
		Playlist playlist `json:"playlist"`
	}
	if err := c.weapiRequest(ctx, rawCookie, "https://music.163.com/weapi/playlist/create", map[string]any{"name": name}, &response); err != nil {
		return Playlist{}, err
	}
	if response.Code != http.StatusOK {
		return Playlist{}, errors.New("Netease playlist creation is unavailable")
	}
	return response.Playlist.playlist(), nil
}

func (c *DirectClient) CreateQR(ctx context.Context) (string, string, error) {
	client, err := c.clientForCookie("")
	if err != nil {
		return "", "", err
	}
	defer client.Close(ctx)
	api := weapi.New(client)
	// Type 1 is the web client flow used by the existing mobile QR page.
	for _, qrType := range []int64{1, 3} {
		key, err := api.QrcodeCreateKey(ctx, &weapi.QrcodeCreateKeyReq{Type: qrType})
		if err != nil {
			return "", "", err
		}
		if key.UniKey != "" && (key.Code == 0 || key.Code == http.StatusOK) {
			return key.UniKey, "https://music.163.com/login?codekey=" + url.QueryEscape(key.UniKey), nil
		}
	}
	return "", "", errors.New("Netease returned no QR key")
}

func (c *DirectClient) CheckQR(ctx context.Context, key string) (QRStatus, error) {
	client, err := c.clientForCookie("")
	if err != nil {
		return QRStatus{}, err
	}
	defer client.Close(ctx)
	response, err := weapi.New(client).QrcodeCheck(ctx, &weapi.QrcodeCheckReq{Key: key, Type: 1})
	if err != nil {
		return QRStatus{}, err
	}
	switch response.Code {
	case 803:
		uri, _ := url.Parse("https://music.163.com")
		return QRStatus{Status: "confirmed", Cookie: cookieString(client.GetCookies(uri))}, nil
	case 800:
		return QRStatus{Status: "expired", Message: response.Message}, nil
	default:
		return QRStatus{Status: "pending", Message: response.Message}, nil
	}
}

type song struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	DurationMS int64  `json:"dt"`
	Album      struct {
		PicURL string `json:"picUrl"`
	} `json:"al"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"ar"`
}

func (s song) track() Track {
	artists := make([]string, 0, len(s.Artists))
	for _, artist := range s.Artists {
		artists = append(artists, artist.Name)
	}
	return Track{ID: s.ID, Title: s.Name, Artists: artists, CoverURL: s.Album.PicURL, DurationMS: s.DurationMS}
}

type playlist struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	CoverURL    string `json:"coverImgUrl"`
	PicURL      string `json:"picUrl"`
	PlayCount   int64  `json:"playCount"`
	TrackCount  int64  `json:"trackCount"`
	Description string `json:"description"`
	Copywriter  string `json:"copywriter"`
}

func (p playlist) playlist() Playlist {
	image := p.CoverURL
	if image == "" {
		image = p.PicURL
	}
	description := p.Description
	if description == "" {
		description = p.Copywriter
	}
	return Playlist{ID: p.ID, Name: p.Name, CoverURL: image, PlayCount: p.PlayCount, TrackCount: p.TrackCount, Description: description}
}
func playlists(items []weapi.PlaylistRespList) []Playlist {
	out := make([]Playlist, 0, len(items))
	for _, item := range items {
		description := ""
		if item.Description != nil {
			description = *item.Description
		}
		out = append(out, Playlist{ID: item.Id, Name: item.Name, CoverURL: item.CoverImgUrl, PlayCount: item.PlayCount, TrackCount: item.TrackCount, Description: description})
	}
	return out
}

func (c *DirectClient) weapiRequest(ctx context.Context, rawCookie, endpoint string, request, response any) error {
	client, err := c.clientForCookie(rawCookie)
	if err != nil {
		return err
	}
	defer client.Close(ctx)
	_, err = client.Request(ctx, endpoint, request, response, ncmapi.NewOptions())
	return err
}
func (c *DirectClient) clientForCookie(rawCookie string) (*ncmapi.Client, error) {
	ensureNeteaseLogger()
	client, err := ncmapi.NewClient(&ncmapi.Config{Timeout: c.timeout, Retry: 1}, ncmlog.Default)
	if err != nil {
		return nil, err
	}
	uri, _ := url.Parse("https://music.163.com")
	client.SetCookies(uri, append(parseCookies(rawCookie), &http.Cookie{Name: "os", Value: "pc"}))
	return client, nil
}
func parseCookies(raw string) []*http.Cookie {
	var cookies []*http.Cookie
	for _, item := range strings.Split(raw, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(item), "=")
		if ok && name != "" {
			cookies = append(cookies, &http.Cookie{Name: name, Value: value})
		}
	}
	return cookies
}
func cookieString(cookies []*http.Cookie) string {
	parts := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie.Name != "" && cookie.Value != "" {
			parts = append(parts, cookie.Name+"="+cookie.Value)
		}
	}
	return strings.Join(parts, "; ")
}
