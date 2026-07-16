# Hyacine Server

[简体中文](README.zh-CN.md)

NestJS API for Hyacine.music clients. It provides account authentication, user library data, and music-source adapters.

## Included

- PostgreSQL-backed users, playlists, favourites, listening history, artists, albums, and tracks
- JWT registration, login, token refresh, logout, and current-user endpoints
- NeteaseCloudMusicApi-compatible QR login, recommendations, and personal playlist proxying
- Bilibili Cookie format validation
- CORS, Helmet, DTO validation, and a health endpoint

## Requirements

- Node.js 20 or later
- pnpm 11
- PostgreSQL
- Redis
- A NeteaseCloudMusicApi-compatible provider for Netease features

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
| `NETEASE_API_BASE` | For Netease | Base URL of a NeteaseCloudMusicApi-compatible service. |

Never commit `.env` or production secrets. Set `CORS_ORIGIN` to the exact web origins that need browser access. Mobile clients should use an address reachable from the device when configuring the server in the app.

## API Surface

All routes are prefixed with `/api/v1`.

| Area | Routes |
| --- | --- |
| Health | `GET /health` |
| Authentication | `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout` |
| User | `GET /users/me` |
| Netease | `GET /music-sources/netease/qr`, `GET /music-sources/netease/qr/:key`, `POST /music-sources/netease/recommendations`, `POST /music-sources/netease/playlists` |
| Bilibili | `POST /music-sources/bilibili/validate-cookie` |

Authenticated routes require an access token. DTO validation rejects unknown request fields.

## Music Sources

### Available now

- **Netease Cloud Music**: QR session creation and polling, recommendation playlists, and the signed-in account's playlists. These endpoints require `NETEASE_API_BASE` and a compatible upstream service.
- **Bilibili**: validates that a submitted Cookie contains `SESSDATA` and `bili_jct`. It does not currently provide Bilibili search, playback, favourites, or playlist synchronization.

Netease Cookies are sent by the client to complete individual source requests. This service does not persist them in its database.

### Extending sources

Music providers are isolated in `src/music-sources`. Additional providers can be added as adapters with explicit DTOs, credential handling, and response normalization. Do not represent a provider as supported until its adapter and client workflow are implemented and tested.

## Client

The React Native client is maintained at [Hyacine.music](https://github.com/Ruoxi-TH/Hyacine-music).