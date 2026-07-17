# Hyacine Server

[English](README.md)

为 Hyacine.music 客户端提供的 NestJS API，负责账户认证、用户音乐库数据和音乐源适配。

## 已提供能力

- 单容器模式下用 SQLite 持久化用户、歌单、收藏、播放历史、歌手、专辑和歌曲
- JWT 注册、登录、刷新令牌、退出登录和当前用户接口
- 内置 NeteaseCloudMusicApi：扫码登录、推荐歌单、个人歌单、搜索和播放地址
- Bilibili Cookie 校验、搜索和播放地址尝试
- CORS、Helmet、DTO 校验和健康检查

## 生产部署（推荐）：一个容器

GitHub Actions 会构建并推送**单容器镜像**，镜像内已包含：

- API
- SQLite
- Redis
- 网易云 NeteaseCloudMusicApi

镜像地址：

```text
ghcr.io/ruoxi-th/hyacine-server:latest
```

工作流：`.github/workflows/build-image.yml`  
构建成功后会自动把 GHCR 包设为 Public。

### 宝塔 / 任意 Docker 主机

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

检查：

```bash
docker ps
curl -sS http://127.0.0.1:3000/api/v1/health
```

手机端后端地址：

```text
http://你的公网IP:3000
```

**不要**填 `127.0.0.1` / `localhost`。安全组 / 宝塔防火墙放行 TCP `3000`。

一键脚本（同样是 pull + 单容器 run）：

```bash
bash scripts/baota-run.sh
```

## 本地开发

依赖：

- Node.js 20+
- pnpm 11
- Redis（或直接用单容器镜像）

```bash
pnpm install
cp .env.example .env
# 配置 DATABASE_URL / REDIS_URL / CORS_ORIGIN / JWT 密钥
pnpm prisma:generate
pnpm prisma:migrate
pnpm start:dev
```

健康检查：

```bash
curl http://localhost:3000/api/v1/health
```

## 可选：多服务 Compose

`docker-compose.yml` 仍可拆成 API + Postgres + Redis + 网易云 四个服务。宝塔场景优先用上面的**单容器镜像**。

```bash
cp .env.deploy.example .env
docker compose up -d --build
```

## 环境变量

| 变量 | 必填 | 用途 |
| --- | --- | --- |
| `DATABASE_URL` | 是 | Prisma 连接。单容器默认 `file:/data/hyacine.db`。 |
| `REDIS_URL` | 是 | Redis 连接。单容器默认 `redis://127.0.0.1:6379`。 |
| `PORT` | 否 | HTTP 端口，默认 `3000`。 |
| `CORS_ORIGIN` | 是 | 允许的来源；手机调试可用 `*`。 |
| `JWT_ACCESS_SECRET` | 是 | Access Token 密钥，≥ 32 字符。 |
| `JWT_REFRESH_SECRET` | 是 | Refresh Token 密钥，≥ 32 字符。 |
| `JWT_ACCESS_TTL` | 否 | 默认 `15m`。 |
| `JWT_REFRESH_TTL` | 否 | 默认 `30d`。 |
| `NETEASE_API_BASE` | 单容器：是 | 网易云上游，默认 `http://127.0.0.1:3001`。 |

不要提交 `.env` 或生产密钥。

## 接口

所有路由前缀：`/api/v1`。

| 范围 | 路由 |
| --- | --- |
| 健康检查 | `GET /health` |
| 认证 | `POST /auth/register`、`POST /auth/login`、`POST /auth/refresh`、`POST /auth/logout` |
| 用户 | `GET /users/me` |
| 网易云 | `GET /music-sources/netease/qr`、`GET /music-sources/netease/qr/:key`、`POST /music-sources/netease/recommendations`、`POST /music-sources/netease/playlists`、`POST /music-sources/netease/search`、`POST /music-sources/netease/play-url` |
| Bilibili | `POST /music-sources/bilibili/validate-cookie`、`POST /music-sources/bilibili/search`、`POST /music-sources/bilibili/play-url` |

## 音乐源

- **网易云**：扫码登录、推荐、歌单、搜索、播放地址（内置 NeteaseCloudMusicApi）。
- **Bilibili**：Cookie/`nav` 校验、搜索、播放地址尝试；尚未达到 NeriPlayer 完整 WBI/Ticket 级别。

第三方 Cookie 由客户端按请求传入，服务端不落库。

## 客户端

React Native 客户端：[Hyacine.music](https://github.com/Ruoxi-TH/Hyacine-music)。
