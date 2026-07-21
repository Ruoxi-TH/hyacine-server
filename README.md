# Hyacine Server

[简体中文](README.zh-CN.md) · [日本語](README.ja-JP.md)

Go HTTP backend for [风堇音乐 / Hyacine Music](https://github.com/Ruoxi-TH/Hyacine-music). The current service is stateless and does not persist mobile profiles, source credentials, favorites, or listening history.

## Implemented capabilities

- Netease Cloud Music direct WEAPI integration with request-scoped cookie jars
- Optional compatible upstream mode through `NETEASE_API_BASE`
- QR login and polling, profile, daily songs, recommendations, playlists and playlist details
- Search, playback URL resolution, short-lived stream tokens, HTTP Range forwarding
- Timed lyrics with translation
- Read-only song comments with nickname, avatar, body, time, location, and like count
- Bilibili credential validation, search, and playback
- CORS and structured health/capability response

## JB

JB integration is documented as a reserved backend extension boundary. No production JB route, credential format, persistence model, or provider adapter is implemented in the current source. Add a concrete adapter under `internal/music/jb` only after the JB protocol, authentication method, and data ownership rules are defined. Do not overload Netease/Bilibili cookies or expose raw JB credentials in logs or health responses.

## Privacy and administration

The mobile App includes an on-device administration screen that calls `GET /api/v1/health` and displays backend reachability, latency, Netease direct/upstream mode, and advertised capabilities. User profile, favorites, listening history, and diagnostic logs shown there remain on the phone. The backend does not currently provide a remote user database or admin dashboard.

Music-service cookies are accepted only for the duration of the corresponding request. They must not be logged or persisted without an explicit encrypted account-storage design.

## Requirements and run

- Go 1.25+
- `PORT` defaults to `3000`
- `NETEASE_API_BASE` is optional

### Configuration file

Create `config.json` in the working directory or `/etc/hyacine/config.json`:

```json
{
  "port": 3000,
  "netease_api_base": "",
  "log_level": "info",
  "cors": {
    "enabled": true,
    "origins": ["*"]
  },
  "stream": {
    "buffer_size": 32768,
    "timeout": 30
  }
}
```

**Priority**: Environment variables > config.json > defaults

### Run

```bash
PORT=3000 ./run.sh
curl -fsS http://127.0.0.1:3000/api/v1/health
```

Compatibility mode:

```bash
NETEASE_API_BASE=http://127.0.0.1:3001 PORT=3000 ./run.sh
```

## Routes

All routes use the `/api/v1` prefix.

| Area | Method and route |
| --- | --- |
| Health | `GET /health` |
| Netease QR | `GET /music-sources/netease/qr`, `GET /music-sources/netease/qr/:key` |
| Profile | `POST /music-sources/netease/profile` |
| Discovery | `POST /music-sources/netease/recommendations`, `/daily-songs` |
| Playlists | `POST /music-sources/netease/playlists`, `/playlists/detail`, `/playlists/create` |
| Search | `POST /music-sources/netease/search` |
| Lyrics | `POST /music-sources/netease/lyrics` |
| Comments | `POST /music-sources/netease/comments` |
| Playback | `POST /music-sources/netease/play-url`, `GET /music-sources/netease/stream/:token` |
| Bilibili | `POST /music-sources/bilibili/validate-cookie`, `/search`, `/play-url` |

Comments are read-only. The endpoint does not post, delete, or like comments.

## Layout

```text
cmd/hyacine-server/       executable entry point
internal/config/          environment loading
internal/httpapi/         routes, CORS, conversion, stream proxy
internal/music/netease/   direct and compatible Netease adapters
internal/music/bilibili/  Bilibili adapter boundary
internal/stream/          short-lived media token store
internal/store/           reserved server persistence boundary
migrations/               reserved versioned database migrations
docs/                     architecture documentation
```

## Acknowledgements

Direct Netease integration uses [chaunsin/netease-cloud-music](https://github.com/chaunsin/netease-cloud-music) under its MIT license.

## Verification status

Formatting and source checks are separate from deployment and real-account validation. In environments that cannot download the Go 1.25 toolchain, `go test ./...` cannot complete until toolchain access is restored.

## License

This project is released under the [MIT License](LICENSE). You may use, copy, modify, merge, publish, distribute, sublicense, and sell copies of the software as permitted by the license. The software is provided without warranty.