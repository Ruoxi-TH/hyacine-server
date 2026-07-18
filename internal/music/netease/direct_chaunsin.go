//go:build chaunsin

package netease

// This file is intentionally build-tagged until the repository baseline moves
// to Go 1.25. The upstream github.com/chaunsin/netease-cloud-music module
// currently requires Go 1.25 and provides direct WEAPI/EAPI encryption,
// per-client cookie jars, QR login, account data, playlists, search, and
// SongPlayerV1. The HTTP API uses Client so this implementation can replace
// HTTPClient without changing route contracts or stream token behavior.
//
// Enable only after updating go.mod and CI to Go 1.25:
//   go build -tags chaunsin ./cmd/hyacine-server
//
// The direct provider must construct a separate upstream client for every
// incoming source cookie. Sharing a cookie jar across mobile users would leak
// account state between requests.
