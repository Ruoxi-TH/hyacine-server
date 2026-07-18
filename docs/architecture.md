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

`internal/music/netease.Client` decouples the HTTP contract from the upstream implementation. `HTTPClient` is the currently compiled adapter and talks to the configured compatible Netease API service. The planned direct adapter uses the MIT-licensed `github.com/chaunsin/netease-cloud-music` WEAPI/EAPI implementation. Its current release requires Go 1.25, while this repository is validated on Go 1.22. The direct adapter must only be enabled after the toolchain and CI baseline are upgraded and must create a separate upstream cookie jar for each request cookie.
