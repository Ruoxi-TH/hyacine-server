# Architecture

- `cmd/hyacine-server`: executable entry point.
- `internal/config`: validated runtime configuration.
- `internal/httpapi`: `/api/v1` HTTP routes, CORS, response handling, and stream proxy.
- `internal/domain`: public music data models.
- `internal/music/netease`: direct Netease adapter boundary.
- `internal/music/bilibili`: Bilibili adapter boundary.
- `internal/stream`: short-lived media token abstraction.
- `internal/store`: future persistent account and library storage.

The service keeps third-party cookies request-scoped. It must never persist a client music-service cookie without an explicit account-storage feature and encryption design.

## Netease provider migration

`internal/music/netease.Client` decouples the HTTP contract from the compatible upstream implementation. The service baseline is Go 1.25. `DirectClient` implements playback with reference to the MIT-licensed [chaunsin/netease-cloud-music](https://github.com/chaunsin/netease-cloud-music) WEAPI `SongPlayerV1` flow when `NETEASE_API_BASE` is unset. The integration uses only the required Go provider packages, creates a separate upstream cookie jar for each request cookie, and retains Hyacine's own stream-token and Range-proxy contract. `HTTPClient` remains available when `NETEASE_API_BASE` is set and continues to serve QR login, profile, playlists, recommendations, and search during their incremental direct-client migration. Direct playback is build-tested but still requires real Cookie and device playback validation.
