package netease

import (
	"testing"

	ncmlog "github.com/chaunsin/netease-cloud-music/pkg/log"
)

func TestEnsureNeteaseLogger(t *testing.T) {
	ncmlog.Default = nil
	ensureNeteaseLogger()
	if ncmlog.Default == nil {
		t.Fatal("expected package logger to be initialized")
	}
	// Second call must remain idempotent.
	ensureNeteaseLogger()
	if ncmlog.Default == nil {
		t.Fatal("logger was cleared after second ensure call")
	}
}

func TestParseCookies(t *testing.T) {
	cookies := parseCookies("MUSIC_U=token; __csrf=value; invalid")
	if len(cookies) != 2 {
		t.Fatalf("count = %d", len(cookies))
	}
	if cookies[0].Name != "MUSIC_U" || cookies[0].Value != "token" {
		t.Fatalf("first = %#v", cookies[0])
	}
	if cookies[1].Name != "__csrf" || cookies[1].Value != "value" {
		t.Fatalf("second = %#v", cookies[1])
	}
}

func TestDailySongCover(t *testing.T) {
	const imageID int64 = 109951171234567890
	const fallback = "https://music.163.com/api/img/blob/109951171234567890"

	tests := []struct {
		name   string
		picURL string
		picStr string
		pic    int64
		want   string
	}{
		{name: "uses pic URL", picURL: "https://p3.music.126.net/cover.jpg", picStr: "123", pic: imageID, want: "https://p3.music.126.net/cover.jpg"},
		{name: "uses numeric pic str", picStr: "109951171234567890", want: fallback},
		{name: "uses pic ID", pic: imageID, want: fallback},
		{name: "omits missing cover", want: ""},
		{name: "ignores invalid pic str", picStr: "not-an-id", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := dailySongCover(test.picURL, test.picStr, test.pic); got != test.want {
				t.Fatalf("dailySongCover() = %q, want %q", got, test.want)
			}
		})
	}
}
