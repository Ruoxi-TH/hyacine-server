package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hyacine-go-server/internal/config"
	"hyacine-go-server/internal/stream"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	res := httptest.NewRecorder()
	NewRouter(config.Config{Port: "3000", NeteaseAPIBase: "http://127.0.0.1:3001"}).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body struct {
		Status  string `json:"status"`
		Netease struct {
			Capabilities map[string]bool `json:"capabilities"`
		} `json:"netease"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" {
		t.Fatalf("status body = %q", body.Status)
	}
	if !body.Netease.Capabilities["search"] {
		t.Fatal("compatibility mode must expose search")
	}
}

func TestHealthDirectModeCapabilities(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	res := httptest.NewRecorder()
	NewRouter(config.Config{Port: "3000"}).ServeHTTP(res, req)

	var body struct {
		Netease struct {
			Direct       bool            `json:"direct"`
			Capabilities map[string]bool `json:"capabilities"`
		} `json:"netease"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Netease.Direct || !body.Netease.Capabilities["dailySongs"] || !body.Netease.Capabilities["playlists"] {
		t.Fatalf("unexpected direct capabilities: %#v", body.Netease)
	}
	for _, capability := range []string{"qr", "profile", "dailySongs", "playlists", "recommendations", "search", "createPlaylist"} {
		if !body.Netease.Capabilities[capability] {
			t.Fatalf("direct capability %q was disabled: %#v", capability, body.Netease.Capabilities)
		}
	}
}

func TestRequestBodyAcceptsNeteaseAndBilibiliIDs(t *testing.T) {
	var netease requestBody
	if err := json.NewDecoder(strings.NewReader(`{"id":123,"cookie":"a"}`)).Decode(&netease); err != nil {
		t.Fatal(err)
	}
	if netease.ID != 123 || netease.BilibiliID != "" {
		t.Fatalf("unexpected Netease body: %#v", netease)
	}

	var bilibili requestBody
	if err := json.NewDecoder(strings.NewReader(`{"id":"BV1xx411c7mD","cid":"9"}`)).Decode(&bilibili); err != nil {
		t.Fatal(err)
	}
	if bilibili.BilibiliID != "BV1xx411c7mD" || bilibili.CID != "9" {
		t.Fatalf("unexpected Bilibili body: %#v", bilibili)
	}
}

func TestCoverUsesHTTPSAndParam(t *testing.T) {
	got := cover("http://p3.music.126.net/example.jpg")
	want := "https://p3.music.126.net/example.jpg?param=400y400"
	if got != want {
		t.Fatalf("cover = %q, want %q", got, want)
	}
	if cover("https://p3.music.126.net/example.jpg?param=300y300") != "https://p3.music.126.net/example.jpg?param=300y300" {
		t.Fatal("cover should keep existing param")
	}
}

func TestCreateStreamResponseIsRelativeToAPIBase(t *testing.T) {
	s := &server{streams: stream.NewStore(0)}
	res := httptest.NewRecorder()
	s.createStreamResponse(res, "https://m701.music.126.net/song.mp3", 128000, "MUSIC_U=1")
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	var body struct {
		URL string `json:"url"`
		BR  int    `json:"br"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.BR != 128000 {
		t.Fatalf("br = %d", body.BR)
	}
	if !strings.HasPrefix(body.URL, "/music-sources/netease/stream/") {
		t.Fatalf("url = %q", body.URL)
	}
	if strings.HasPrefix(body.URL, "/api/v1/") {
		t.Fatalf("url should not include /api/v1 prefix: %q", body.URL)
	}
}
