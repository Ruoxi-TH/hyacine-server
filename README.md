# Hyacine Server

[简体中文](README.zh-CN.md)

NestJS API for Hyacine.music clients. It provides account authentication, user library data, and music-source adapters.

## Included

- PostgreSQL-backed users, playlists, favourites, listening history, artists, albums, and tracks
- JWT registration, login, token refresh, logout, and current-user endpoints
- Built-in NeteaseCloudMusicApi service for QR login, recommendations, personal playlists, and track search
- Bilibili login-state validation and video search
- CORS, Helmet, DTO validation, and a health endpoint

## Requirements

- Node.js 20 or later
- pnpm 11
- PostgreSQL
- Redis
- Docker and Docker Compose for the bundled production deployment

## Quick Start

```bash
pnpm install
cp .env.example .env
```

Set the PostgreSQL, Redis, CORS, and JWT values in `.env`. Generate a distinct random value of at least 32 characters for each JWT secret.

For local development, create the database and run:

```bash
pnpm prisma:generate
pnpm prisma:migrate
pnpm start:dev
```

The API listens on `PORT` (default `3000`) and is served below `/api/v1`. Confirm it with:

```bash
curl http://localhost:3000/api/v1/health
```

For production, apply existing migrations and start the compiled application:

```bash
pnpm prisma:generate
pnpm prisma:deploy
pnpm build
pnpm start:prod
```

## Docker Deployment

Docker Compose starts the API, PostgreSQL, Redis, and a NeteaseCloudMusicApi container together. On the target host, clone this repository, create the production environment file, then start it:

```bash
cp .env.deploy.example .env
# Edit .env with strong passwords, JWT secrets, and CORS_ORIGIN.
docker compose up -d --build
curl http://127.0.0.1:3000/api/v1/health
```

The API uses the bundled `netease` service at `http://netease:3000`. Do not set `NETEASE_API_BASE` in this deployment unless intentionally replacing that internal upstream.

### GitHub Actions deployment

`.github/workflows/deploy.yml` is a manual `workflow_dispatch` deployment for an existing Docker host. Create a `production` GitHub environment and set these secrets:

| Secret | Purpose |
| --- | --- |
| `DEPLOY_HOST` | Server hostname or IP. |
| `DEPLOY_USER` | SSH user with Docker access. |
| `DEPLOY_SSH_KEY` | Private SSH key for that user. |
| `DEPLOY_PORT` | Optional SSH port; defaults to `22`. |
| `DEPLOY_PATH` | Absolute path of the cloned `hyacine-server` repository. |

The target directory must contain a server-local `.env` created from `.env.deploy.example`. The workflow fetches `master`, rebuilds Compose services, and verifies the health endpoint.

## Configuration

| Variable | Required | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | Yes | PostgreSQL Prisma connection URL. |
| `REDIS_URL` | Yes | Redis connection URL. |
| `PORT` | No | HTTP port. Defaults to `3000`. |
| `CORS_ORIGIN` | Yes | Comma-separated allowed client origins. |
| `JWT_ACCESS_SECRET` | Yes | Access-token signing secret, at least 32 characters. |
| `JWT_REFRESH_SECRET` | Yes | Refresh-token signing secret, at least 32 characters. |
| `JWT_ACCESS_TTL` | No | Access-token lifetime. Defaults to `15m`. |
| `JWT_REFRESH_TTL` | No | Refresh-token lifetime. Defaults to `30d`. |
| `NETEASE_API_BASE` | No in Compose | Optional override for a NeteaseCloudMusicApi-compatible service. Compose uses the bundled `netease` service. |

Never commit `.env` or production secrets. Set `CORS_ORIGIN` to the exact web origins that need browser access. Mobile clients should use an address reachable from the device when configuring the server in the app.

## API Surface

All routes are prefixed with `/api/v1`.

| Area | Routes |
| --- | --- |
| Health | `GET /health` |
| Authentication | `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout` |
| User | `GET /users/me` |
| Netease | `GET /music-sources/netease/qr`, `GET /music-sources/netease/qr/:key`, `POST /music-sources/netease/recommendations`, `POST /music-sources/netease/playlists`, `POST /music-sources/netease/search` |
| Bilibili | `POST /music-sources/bilibili/validate-cookie`, `POST /music-sources/bilibili/search` |

Authenticated routes require an access token. DTO validation rejects unknown request fields.

## Music Sources

### Available now

- **Netease Cloud Music**: QR session creation and polling, recommendation playlists, the signed-in account's playlists, and track search. Docker Compose provides its NeteaseCloudMusicApi upstream internally.
- **Bilibili**: validates actual authenticated login state through Bilibili's `nav` endpoint and searches public video results. It does not provide audio playback URL resolution, favourites, or playlist synchronization.

Netease Cookies are sent by the client to complete individual source requests. This service does not persist them in its database.

### Extending sources

Music providers are isolated in `src/music-sources`. Additional providers can be added as adapters with explicit DTOs, credential handling, and response normalization. Do not represent a provider as supported until its adapter and client workflow are implemented and tested.

## Client

The React Native client is maintained at [Hyacine.music](https://github.com/Ruoxi-TH/Hyacine-music).