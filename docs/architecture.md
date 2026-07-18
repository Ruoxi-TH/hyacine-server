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
