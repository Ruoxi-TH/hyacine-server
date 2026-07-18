package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hyacine-go-server/internal/config"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	res := httptest.NewRecorder()
	NewRouter(config.Config{Port: "3000", NeteaseAPIBase: "http://127.0.0.1:3001"}).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status body = %q", body["status"])
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
