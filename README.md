# Hyacine Server

[简体中文](README.zh-CN.md)

NestJS API for Hyacine.music clients. It provides account authentication, user library data, and music-source adapters.

## Included

- SQLite-backed users, playlists, favourites, listening history, artists, albums, and tracks (single-container mode)
- JWT registration, login, token refresh, logout, and current-user endpoints
- Built-in NeteaseCloudMusicApi for QR login, recommendations, personal playlists, search, and play-url
- Bilibili cookie validation, search, and play-url attempt
- CORS, Helmet, DTO validation, and a health endpoint

## Production (recommended): one container

GitHub Actions builds a single image that already contains:

- API
- SQLite
- Redis
- NeteaseCloudMusicApi

Image:

```text
ghcr.io/ruoxi-th/hyacine-server:latest
```

Workflow: `.github/workflows/build-image.yml`  
After a successful build, the package is set to public automatically.

### Baota / any Docker host

```bash
docker pull ghcr.io/ruoxi-th/hyacine-server:latest

docker rm -f hyacine 2>/dev/null || true

docker run -d \
  --name hyacine \
  --restart unless-stopped \
  -p 3000:3000 \
  -e PORT=3000 \
  -e DATABASE_URL=file:/data/hyacine.db \
  -e REDIS_URL=redis://127.0.0.1:6379 \
  -e NETEASE_API_BASE=http://127.0.0.1:3001 \
  -e CORS_ORIGIN=* \
  -e JWT_ACCESS_SECRET=replace_with_random_32+_chars_access \
  -e JWT_REFRESH_SECRET=replace_with_random_32+_chars_refresh \
  -v /www/wwwroot/hyacine-data:/data \
  ghcr.io/ruoxi-th/hyacine-server:latest
```

Check:

```bash
docker ps
curl -sS http://127.0.0.1:3000/api/v1/health
```

Mobile client backend URL:

```text
http://YOUR_PUBLIC_IP:3000
```

Do **not** use `127.0.0.1` / `localhost` on the phone. Open TCP `3000` in the firewall/security group.

One-liner (same behaviour):

```bash
bash scripts/baota-run.sh
```

## Local development

Requirements:

- Node.js 20+
- pnpm 11
- Redis (or use single-container for everything)

```bash
pnpm install
cp .env.example .env
# set DATABASE_URL / REDIS_URL / CORS_ORIGIN / JWT secrets
pnpm prisma:generate
pnpm prisma:migrate
pnpm start:dev
```

Health:

```bash
curl http://localhost:3000/api/v1/health
```

## Optional: multi-service Compose

`docker-compose.yml` can still start API + Postgres + Redis + Netease as separate services. Prefer the **one-container** image above for Baota.

```bash
cp .env.deploy.example .env
docker compose up -d --build
```

## Configuration

| Variable | Required | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | Yes | Prisma URL. Single container uses `file:/data/hyacine.db`. |
| `REDIS_URL` | Yes | Redis URL. Single container uses `redis://127.0.0.1:6379`. |
| `PORT` | No | HTTP port, default `3000`. |
| `CORS_ORIGIN` | Yes | Allowed origins; `*` is fine for mobile testing. |
| `JWT_ACCESS_SECRET` | Yes | Access-token secret, ≥ 32 chars. |
| `JWT_REFRESH_SECRET` | Yes | Refresh-token secret, ≥ 32 chars. |
| `JWT_ACCESS_TTL` | No | Default `15m`. |
| `JWT_REFRESH_TTL` | No | Default `30d`. |
| `NETEASE_API_BASE` | Single container: yes | Upstream for Netease API, default `http://127.0.0.1:3001`. |

Never commit `.env` or production secrets.

## API surface

All routes are under `/api/v1`.

| Area | Routes |
| --- | --- |
| Health | `GET /health` |
| Auth | `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout` |
| User | `GET /users/me` |
| Netease | `GET /music-sources/netease/qr`, `GET /music-sources/netease/qr/:key`, `POST /music-sources/netease/recommendations`, `POST /music-sources/netease/playlists`, `POST /music-sources/netease/search`, `POST /music-sources/netease/play-url` |
| Bilibili | `POST /music-sources/bilibili/validate-cookie`, `POST /music-sources/bilibili/search`, `POST /music-sources/bilibili/play-url` |

## Music sources

- **Netease**: QR login, recommendations, playlists, search, play-url (via built-in NeteaseCloudMusicApi).
- **Bilibili**: cookie/nav validation, search, play-url attempt. Full WBI/ticket parity with NeriPlayer is not complete.

Cookies for third-party sources are sent by the client per request and are not stored in the DB.

## Client

React Native client: [Hyacine.music](https://github.com/Ruoxi-TH/Hyacine-music).
