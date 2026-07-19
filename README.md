# Hyacine Go Backend

Standalone Go HTTP backend for the [风堇音乐](https://github.com/Ruoxi-TH/Hyacine-music) mobile client. No Docker, Node.js, Prisma, Redis, or database runtime dependency. The current release is stateless; the mobile client stores account profile, preferences, history, and source credentials locally.

## Current version: 0.3.0

### Implemented

- **Netease Cloud Music direct playback**: Go WEAPI client with per-request Cookie Jar, no shared account state
- **QR login, profile, recommendations, daily songs, playlists, search, lyrics**
- **Playlist detail endpoint**: `POST /api/v1/music-sources/netease/playlists/detail` returns full track list
- **Timed lyrics with translation**: `POST /api/v1/music-sources/netease/lyrics`
- **Bilibili cookie validation, search, playback**
- **Short-lived audio stream proxy** with Range header forwarding
- **CORS and health endpoint**

### In progress / TODO

- Server-side account persistence (SQLite migrations scaffold exists)
- Go test suite (toolchain download timeout in CI)
- Real-account CDN Range proxy end-to-end validation

## Project layout

```text
cmd/hyacine-server/       Application entry point
internal/config/          Environment loading and validation
internal/domain/          Shared API data models
internal/httpapi/         Versioned routes, CORS, stream proxy
internal/music/           Netease and Bilibili adapter boundaries
internal/stream/          Short-lived audio stream token boundary
internal/store/           Future server-side account/library persistence
migrations/               Future versioned SQLite migrations
docs/                     Architecture documentation
```

See [docs/architecture.md](docs/architecture.md) for ownership rules.

## Requirements

- Go 1.25+ when building from source
- Optional: `NETEASE_API_BASE` set to a compatible upstream, for example `http://127.0.0.1:3001`

Without `NETEASE_API_BASE`, Netease playback uses the MIT-licensed `chaunsin/netease-cloud-music` Go WEAPI client directly. Each play request gets a separate upstream Cookie Jar built from the submitted music-service cookie. The optional compatible upstream mode remains available for QR login, account/profile, playlists, recommendations, and search while those endpoints are migrated one by one.

## Acknowledgements

Netease direct playback is implemented with reference to [chaunsin/netease-cloud-music](https://github.com/chaunsin/netease-cloud-music), an MIT-licensed Go project. Hyacine uses only the provider capabilities needed for this backend, currently the WEAPI `SongPlayerV1` playback flow and per-request cookie handling.

## Run from source

```bash
PORT=3000 ./run.sh
# Optional compatibility mode:
# NETEASE_API_BASE=http://127.0.0.1:3001 PORT=3000 ./run.sh
```

## Run in the background

```bash
NETEASE_API_BASE=http://127.0.0.1:3001 PORT=3000 \
  nohup ./run.sh >/tmp/hyacine-go.log 2>&1 &
curl -fsS http://127.0.0.1:3000/api/v1/health
```

For a mobile device, configure the backend URL as `http://SERVER_IP:3000`. Do not use `localhost` or `127.0.0.1` in the mobile app.

## Environment

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `NETEASE_API_BASE` | No | None | Enables compatible upstream mode for Netease endpoints not yet migrated to direct Go client |
| `PORT` | No | `3000` | HTTP listen port |

## Routes

All routes begin with `/api/v1`.

| Feature | Route |
| --- | --- |
| Health | `GET /health` |
| Netease QR login | `GET /music-sources/netease/qr`, `GET /music-sources/netease/qr/:key` |
| Netease data | `POST /music-sources/netease/profile`, `/recommendations`, `/daily-songs`, `/playlists`, `/playlists/detail`, `/playlists/create`, `/search`, `/lyrics` |
| Netease playback | `POST /music-sources/netease/play-url`, `GET /music-sources/netease/stream/:token` |
| Bilibili | `POST /music-sources/bilibili/validate-cookie`, `/search`, `/play-url` |

## CI & Release

- **CI** (`.github/workflows/ci.yml`): runs `go test ./...` and `go build` on every push to `go-backend-rewrite`
- **Release** (`.github/workflows/release.yml`): builds static binaries for 5 platforms when a `v*` tag is pushed, attaches them to a GitHub Release

## License

See [LICENSE](LICENSE).