package domain

type Playlist struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	CoverURL    string `json:"coverUrl"`
	PlayCount   int64  `json:"playCount"`
	TrackCount  int64  `json:"trackCount"`
	Description string `json:"description"`
}

type Track struct {
	ID         int64    `json:"id"`
	Title      string   `json:"title"`
	Artists    []string `json:"artists"`
	Album      string   `json:"album"`
	CoverURL   string   `json:"coverUrl"`
	DurationMS int64    `json:"durationMs"`
	Source     string   `json:"source"`
}
