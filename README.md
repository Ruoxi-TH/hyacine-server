# Hyacine Go Backend

Standalone Go HTTP backend for the Hyacine.music mobile client's music-source features. It has no Docker, Node.js, Prisma, Redis, or database runtime dependency. The current release is stateless because the mobile client stores its account profile, preferences, history, and source credentials locally.

## Project layout

```text
cmd/hyacine-server/       Application entry point
internal/config/          Environment loading and validation
internal/domain/          Shared API data models
internal/httpapi/         Versioned routes, CORS, stream proxy, request tests
internal/music/           Netease and Bilibili adapter boundaries
internal/stream/          Short-lived audio stream token boundary
internal/store/           Future server-side account/library persistence boundary
migrations/               Future versioned SQLite migrations
docs/                     Architecture documentation
```

See [docs/architecture.md](docs/architecture.md) for ownership rules.

## Requirements

- Go 1.25+ when building from source
- Optional: `NETEASE_API_BASE` set to a compatible upstream, for example `http://127.0.0.1:3001`

Without `NETEASE_API_BASE`, Netease playback uses the MIT-licensed `chaunsin/netease-cloud-music` Go WEAPI client directly. Each play request gets a separate upstream Cookie Jar built from the submitted music-service cookie. This prevents account cookies from being shared across users. The optional compatible upstream mode remains available for QR login, account/profile, playlists, recommendations, and search while those endpoints are migrated one by one.

## Netease Reference

Direct Netease playback is implemented with reference to [chaunsin/netease-cloud-music](https://github.com/chaunsin/netease-cloud-music), an MIT-licensed Go project. Hyacine uses only the provider capabilities needed for this backend, currently the WEAPI `SongPlayerV1` playback flow and per-request cookie handling. It does not embed or expose the reference project's CLI, download, sign-in automation, or unrelated features.

The direct path is compiled and tested locally, but real-account Cookie, CDN Range proxy, and mobile-player end-to-end playback still require separate live validation.

## Run from source

```bash
PORT=3000 ./run.sh
# Optional compatibility mode for QR/profile/playlist/search endpoints:
# NETEASE_API_BASE=http://127.0.0.1:3001 PORT=3000 ./run.sh
```

`run.sh` compiles the current source and starts the server. To build once and run the binary directly:

```bash
go build -o hyacine-go-server .
NETEASE_API_BASE=http://127.0.0.1:3001 PORT=3000 ./hyacine-go-server
```

## Run in the background

```bash
NETEASE_API_BASE=http://127.0.0.1:3001 PORT=3000 \
  nohup ./run.sh >/tmp/hyacine-go.log 2>&1 &

curl -fsS http://127.0.0.1:3000/api/v1/health
```

For a mobile device, configure the backend URL as `http://SERVER_IP:3000`. Do not use `localhost` or `127.0.0.1` in the mobile app. Open TCP port `3000` in the host firewall and cloud security group.

## Environment

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `NETEASE_API_BASE` | No | None | Enables the compatible upstream mode for Netease endpoints not yet migrated to the direct Go client. |
| `PORT` | No | `3000` | HTTP listen port. |

## Routes

All routes begin with `/api/v1`.

| Feature | Route |
| --- | --- |
| Health | `GET /health` |
| Netease QR login | `GET /music-sources/netease/qr`, `GET /music-sources/netease/qr/:key` |
| Netease data | `POST /music-sources/netease/profile`, `/recommendations`, `/daily-songs`, `/playlists`, `/playlists/create`, `/search` |
| Netease playback | `POST /music-sources/netease/play-url`, `GET /music-sources/netease/stream/:token` |
| Bilibili validation | `POST /music-sources/bilibili/validate-cookie` |
| Bilibili search | `POST /music-sources/bilibili/search` |
| Bilibili playback | `POST /music-sources/bilibili/play-url` |

Netease playback URLs are temporary. `play-url` returns a 15-minute local stream token instead of exposing the upstream CDN URL. The stream endpoint forwards the client `Range` header, plus the required Cookie, desktop User-Agent, and Referer headers to the upstream audio URL.

Bilibili playback accepts a string BV ID in `id` and an optional `cid`. The service resolves a missing `cid`, prefers DASH audio, and falls back to `durl` when needed.

Source cookies are supplied in request JSON by the mobile client and are never written to disk by this service.

## CI artifacts

GitHub Actions builds static binaries for:

- Linux amd64
- Linux arm64
- Linux armv7
- macOS arm64
- Windows amd64

See [.github/workflows/build-go-backend.yml](.github/workflows/build-go-backend.yml). Artifacts are retained for 14 days per workflow run.