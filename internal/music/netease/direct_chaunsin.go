package netease

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ncmapi "github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
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

// DirectClient uses the upstream Go WEAPI implementation. It creates a new
// client and cookie jar for every request, preventing cross-user cookie leaks.
type DirectClient struct{ timeout time.Duration }

func NewDirectClient(timeout time.Duration) *DirectClient { return &DirectClient{timeout: timeout} }

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
	out := make([]Playlist, 0, len(response.Playlist))
	for _, playlist := range response.Playlist {
		description := ""
		if playlist.Description != nil {
			description = *playlist.Description
		}
		out = append(out, Playlist{ID: playlist.Id, Name: playlist.Name, CoverURL: playlist.CoverImgUrl, PlayCount: playlist.PlayCount, TrackCount: playlist.TrackCount, Description: description})
	}
	return out, nil
}

func (c *DirectClient) clientForCookie(rawCookie string) (*ncmapi.Client, error) {
	client, err := ncmapi.NewClient(&ncmapi.Config{Timeout: c.timeout, Retry: 1}, nil)
	if err != nil {
		return nil, err
	}
	uri, _ := url.Parse("https://music.163.com")
	cookies := append(parseCookies(rawCookie), &http.Cookie{Name: "os", Value: "pc"})
	client.SetCookies(uri, cookies)
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
