# Hyacine Go Backend

Standalone HTTP backend for the Hyacine.music mobile client's music-source features. It is dependency-free at runtime: no Docker, Node.js, Prisma, Redis, or database is required.

## Requirements

- A Netease API-compatible upstream service
- `NETEASE_API_BASE` set to that upstream, for example `http://127.0.0.1:3001`
- Go 1.22+ only when building from source

## Run from source

```bash
NETEASE_API_BASE=http://127.0.0.1:3001 PORT=3000 ./run.sh
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
| `NETEASE_API_BASE` | Yes | None | Base URL of the compatible Netease API service. |
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