# 风堇音乐后端 · Hyacine Server

[English](README.md) · [日本語](README.ja-JP.md)

风堇音乐后端是为移动端提供音乐源接入、播放地址解析和服务状态检查的 Go HTTP 服务。当前运行模式是无状态服务，不持久化手机中的个人资料、音乐源凭据、本地收藏或听歌历史。

## 已实现能力

- 网易云音乐 WEAPI 直连，每次请求使用独立 Cookie Jar
- 可选的 `NETEASE_API_BASE` 兼容上游模式
- 网易云扫码登录、资料、每日推荐、推荐歌单、歌单详情
- 搜索、播放地址解析、短时流媒体令牌和 HTTP Range 转发
- 带翻译的定时歌词
- 网易云歌曲只读评论
- Bilibili 凭据验证、搜索和播放
- CORS 与结构化健康状态、能力响应

## JB

JB 是预留的后端扩展边界。目前没有生产路由、凭据格式、持久化模型或适配器。在明确 JB 的协议、鉴权方式和数据归属规则后，才应在 `internal/music/jb` 中实现。不得复用网易云或 Bilibili 凭据，也不得在日志或健康响应中泄露原始凭据。

## 管理后台与数据边界

后端已经实现 `GET /api/v1/health`。App 管理后台会真实调用该接口，显示后端可达性、延迟、网易云直连或兼容上游模式及能力标记。

用户昵称、头像、本地收藏数量、听歌历史、音乐源凭据存在状态和客户端日志由 App 从当前手机读取。后端目前没有用户数据库、远程日志库或服务端多用户管理后台。为了避免未鉴权的数据泄露，不会把手机数据通过公开管理接口上传到服务器。

## 运行

要求 Go 1.25 或更高版本。`PORT` 默认值为 `3000`，`NETEASE_API_BASE` 可选。

```bash
PORT=3000 ./run.sh
curl -fsS http://127.0.0.1:3000/api/v1/health
```

手机中的后端地址必须是手机能够访问的地址，例如 `http://SERVER_IP:3000`，不能填写电脑自身的 `localhost`。

## 主要路由

所有路由使用 `/api/v1` 前缀。

| 分类 | 方法与路径 |
| --- | --- |
| 健康状态 | `GET /health` |
| 网易云扫码 | `GET /music-sources/netease/qr`、`GET /music-sources/netease/qr/:key` |
| 用户资料 | `POST /music-sources/netease/profile` |
| 推荐 | `POST /music-sources/netease/recommendations`、`/daily-songs` |
| 歌单 | `POST /music-sources/netease/playlists`、`/playlists/detail`、`/playlists/create` |
| 搜索 | `POST /music-sources/netease/search` |
| 歌词 | `POST /music-sources/netease/lyrics` |
| 评论 | `POST /music-sources/netease/comments` |
| 播放 | `POST /music-sources/netease/play-url`、`GET /music-sources/netease/stream/:token` |
| Bilibili | `POST /music-sources/bilibili/validate-cookie`、`/search`、`/play-url` |

评论接口只读，不支持发布、删除或点赞。

## 项目结构

```text
cmd/hyacine-server/       程序入口
internal/config/          环境配置
internal/httpapi/         路由、CORS、响应转换和流代理
internal/music/netease/   网易云直连与兼容适配器
internal/music/bilibili/  Bilibili 适配边界
internal/stream/          短时媒体令牌
internal/store/           预留服务端持久化边界
migrations/               预留数据库迁移
docs/                     架构文档
```

## 许可证

本项目采用 [MIT 许可证](LICENSE) 开源。你可以在许可证允许的范围内使用、复制、修改、合并、发布、分发、再许可及销售本软件副本；本软件不提供任何形式的担保。