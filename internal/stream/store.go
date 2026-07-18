package stream

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Entry struct {
	URL     string
	Cookie  string
	Expires time.Time
}

type Store struct {
	mu      sync.Mutex
	entries map[string]Entry
	ttl     time.Duration
}

func NewStore(ttl time.Duration) *Store {
	return &Store{entries: make(map[string]Entry), ttl: ttl}
}

func (s *Store) Create(url, cookie string) string {
	token := newToken()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpiredLocked(time.Now())
	s.entries[token] = Entry{URL: url, Cookie: cookie, Expires: time.Now().Add(s.ttl)}
	return token
}

func (s *Store) Get(token string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[token]
	if !ok || time.Now().After(entry.Expires) {
		delete(s.entries, token)
		return Entry{}, false
	}
	return entry, true
}

func (s *Store) removeExpiredLocked(now time.Time) {
	for token, entry := range s.entries {
		if now.After(entry.Expires) {
			delete(s.entries, token)
		}
	}
}

func newToken() string {
	bytes := make([]byte, 24)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
