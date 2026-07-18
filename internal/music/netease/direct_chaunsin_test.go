package netease

import "testing"

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
