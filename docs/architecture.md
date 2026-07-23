# Architecture

## Ownership boundaries

- `cmd/hyacine-server`: process entry point.
- `internal/config`: validated environment configuration.
- `internal/httpapi`: `/api/v1` routes, CORS, response conversion, and stream proxy.
- `internal/music/netease`: direct WEAPI and compatible-upstream Netease adapters.
- `internal/music/bilibili`: Bilibili integration boundary.
- `internal/stream`: short-lived media URL token store.
- `internal/store` and `migrations`: reserved persistence boundary; unused by the current stateless runtime.

## Request and privacy model

Third-party credentials arrive in individual requests. Direct Netease calls create a separate cookie jar per request so accounts cannot share provider state. Credentials are not part of health output and must never be written to logs. The App administration page only reports whether a credential exists on the phone; it does not send credential content to the health endpoint.

## Netease

When `NETEASE_API_BASE` is unset, `DirectClient` uses `chaunsin/netease-cloud-music` WEAPI calls for implemented capabilities. When configured, compatible-upstream HTTP routes remain available. Both modes preserve Hyacine's public API contract and short-lived stream proxy.

Current public capabilities include profile, discovery, playlists, search, playback, lyrics, and read-only comments. Comment mutation is intentionally outside the current scope.

## App administration contract

`GET /api/v1/health` is the only backend status contract consumed by the current App administration screen. It returns service status, timestamp, Netease mode, and capability flags. User counts and client logs are calculated locally on the phone; there is no server-side user enumeration API.

## JB extension boundary

JB is reserved as a future provider boundary. Until its protocol and authentication contract are specified, there is no `internal/music/jb` implementation and no `/api/v1/music-sources/jb` route. A future implementation must use request-scoped credentials, redact secrets, advertise exact capabilities through health output, and avoid coupling JB state to Netease or Bilibili adapters.