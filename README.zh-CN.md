# Hyacine Server

[English](README.md)

为 Hyacine.music 客户端提供的 NestJS API，负责账户认证、用户音乐库数据和音乐源适配。

## 已提供能力

- PostgreSQL 持久化用户、歌单、收藏、播放历史、歌手、专辑和歌曲
- JWT 注册、登录、刷新令牌、退出登录和当前用户接口
- 兼容 NeteaseCloudMusicApi 的网易云扫码登录、推荐歌单和个人歌单代理
- Bilibili Cookie 格式校验
- CORS、Helmet、DTO 校验和健康检查

## 依赖

- Node.js 20 或更高版本
- pnpm 11
- PostgreSQL
- Redis
- 用于网易云功能的 NeteaseCloudMusicApi 兼容服务

## 快速开始

```bash
pnpm install
cp .env.example .env
```

在 `.env` 中配置 PostgreSQL、Redis、CORS 与 JWT。两个 JWT 密钥必须分别使用长度至少为 32 字符的随机值。

本地开发时，创建数据库后运行：

```bash
pnpm prisma:generate
pnpm prisma:migrate
pnpm start:dev
```

API 监听 `PORT`，默认是 `3000`，所有路由位于 `/api/v1` 下。可通过以下命令确认服务可用：

```bash
curl http://localhost:3000/api/v1/health
```

生产环境执行已有迁移并启动构建产物：

```bash
pnpm prisma:generate
pnpm prisma:deploy
pnpm build
pnpm start:prod
```

## 环境变量

| 变量 | 必填 | 用途 |
| --- | --- | --- |
| `DATABASE_URL` | 是 | PostgreSQL Prisma 连接地址。 |
| `REDIS_URL` | 是 | Redis 连接地址。 |
| `PORT` | 否 | HTTP 端口，默认 `3000`。 |
| `CORS_ORIGIN` | 是 | 允许访问的 Web 来源，多个来源以逗号分隔。 |
| `JWT_ACCESS_SECRET` | 是 | 至少 32 字符的 Access Token 签名密钥。 |
| `JWT_REFRESH_SECRET` | 是 | 至少 32 字符的 Refresh Token 签名密钥。 |
| `JWT_ACCESS_TTL` | 否 | Access Token 有效期，默认 `15m`。 |
| `JWT_REFRESH_TTL` | 否 | Refresh Token 有效期，默认 `30d`。 |
| `NETEASE_API_BASE` | 网易云功能需要 | NeteaseCloudMusicApi 兼容服务的基础地址。 |

不要提交 `.env` 或生产密钥。`CORS_ORIGIN` 应只填写需要浏览器访问的确切来源。移动端首次设置时，填写设备能够访问的服务器地址。

## 接口

所有路由均有 `/api/v1` 前缀。

| 范围 | 路由 |
| --- | --- |
| 健康检查 | `GET /health` |
| 认证 | `POST /auth/register`、`POST /auth/login`、`POST /auth/refresh`、`POST /auth/logout` |
| 用户 | `GET /users/me` |
| 网易云 | `GET /music-sources/netease/qr`、`GET /music-sources/netease/qr/:key`、`POST /music-sources/netease/recommendations`、`POST /music-sources/netease/playlists` |
| Bilibili | `POST /music-sources/bilibili/validate-cookie` |

需要认证的接口必须携带 Access Token。DTO 校验会拒绝未声明的请求字段。

## 音乐源

### 当前已接入

- **网易云音乐**：创建和轮询扫码会话、读取推荐歌单、读取当前登录账号的个人歌单。依赖 `NETEASE_API_BASE` 与兼容的上游服务。
- **Bilibili**：只校验提交的 Cookie 是否包含 `SESSDATA` 和 `bili_jct`。目前不提供 Bilibili 搜索、播放、收藏或歌单同步。

网易云 Cookie 由客户端在单次音乐源请求中传入，服务端不会将其写入数据库。

### 扩展第三方平台

音乐源位于 `src/music-sources`。可以按适配器方式添加其他平台，并为凭据、DTO 和响应格式建立明确边界。未实现且未测试完整客户端流程的平台不能视为已支持。

## 客户端

React Native 客户端位于 [Hyacine.music](https://github.com/Ruoxi-TH/Hyacine-music)。