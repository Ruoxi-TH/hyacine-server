package netease

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	ncmapi "github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
)

var ErrNoPlayableURL = errors.New("Netease returned no playable URL")

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
