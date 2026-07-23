package stream

import (
	"testing"
	"time"
)

func TestStoreCreatesAndFindsEntry(t *testing.T) {
	store := NewStore(time.Minute)
	token := store.Create("https://media.example/song.mp3", "MUSIC_U=value")
	entry, ok := store.Get(token)
	if !ok {
		t.Fatal("token was not found")
	}
	if entry.URL != "https://media.example/song.mp3" || entry.Cookie != "MUSIC_U=value" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
}

func TestStoreExpiresEntry(t *testing.T) {
	store := NewStore(time.Nanosecond)
	token := store.Create("https://media.example/song.mp3", "")
	time.Sleep(time.Millisecond)
	if _, ok := store.Get(token); ok {
		t.Fatal("expired token was returned")
	}
}
