package netease

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPClientForwardsCookie(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/account" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Cookie") != "MUSIC_U=test" {
			t.Fatalf("cookie = %q", r.Header.Get("Cookie"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200}`))
	}))
	defer upstream.Close()

	client := NewHTTPClient(upstream.URL, time.Second)
	body, err := client.Get(context.Background(), "/user/account", "MUSIC_U=test")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"code":200}` {
		t.Fatalf("body = %s", body)
	}
}

func TestHTTPClientRejectsUpstreamErrors(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer upstream.Close()
	if _, err := NewHTTPClient(upstream.URL, time.Second).Get(context.Background(), "/x", ""); err == nil {
		t.Fatal("expected upstream error")
	}
}
