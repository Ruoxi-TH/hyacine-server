package config

import "testing"

func TestLoadDefaultsPort(t *testing.T) {
	t.Setenv("NETEASE_API_BASE", "http://127.0.0.1:3001/")
	t.Setenv("PORT", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "3000" {
		t.Fatalf("port = %q", cfg.Port)
	}
	if cfg.NeteaseAPIBase != "http://127.0.0.1:3001" {
		t.Fatalf("base = %q", cfg.NeteaseAPIBase)
	}
}

func TestLoadRequiresNeteaseBase(t *testing.T) {
	t.Setenv("NETEASE_API_BASE", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected configuration error")
	}
}
