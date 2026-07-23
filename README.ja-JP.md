# Hyacine Server · 風菫音楽バックエンド

[简体中文](README.zh-CN.md) · [English](README.md)

Hyacine Server は、モバイルアプリに音楽ソース連携、再生 URL 解決、サービス状態確認を提供する Go HTTP バックエンドです。現在のサービスはステートレスで、端末内プロフィール、音楽ソース認証情報、お気に入り、再生履歴を保存しません。

## 実装済み

- リクエスト単位の Cookie Jar を使う NetEase Cloud Music WEAPI 直接接続
- `NETEASE_API_BASE` による互換アップストリームモード
- QR ログイン、プロフィール、デイリー曲、おすすめ、プレイリスト詳細
- 検索、再生 URL 解決、短時間ストリームトークン、HTTP Range 転送
- 翻訳付き同期歌詞
- 閲覧専用コメント
- Bilibili 認証情報検証、検索、再生
- CORS と構造化ヘルス・機能レスポンス

## JB

JB は将来のバックエンド拡張境界として予約されています。現在、実運用ルート、認証情報形式、永続化モデル、アダプターはありません。プロトコル、認証方法、データ所有規則が確定した後に `internal/music/jb` へ実装します。NetEase/Bilibili の認証情報を再利用せず、ログやヘルスレスポンスへ秘密情報を出力してはいけません。

## 管理画面とデータ境界

バックエンドには `GET /api/v1/health` が実装済みです。アプリの管理画面はこの API を実際に呼び出し、到達可否、遅延、NetEase 動作モード、機能フラグを表示します。

ユーザー名、アバター、端末内お気に入り、再生履歴、認証情報の有無、クライアントログは現在の端末から読み取ります。バックエンドにはユーザーデータベース、リモートログストア、複数ユーザー向け管理コンソールはありません。未認証の公開 API へ端末データを送信しません。

## 起動

Go 1.25 以降が必要です。`PORT` の既定値は `3000`、`NETEASE_API_BASE` は任意です。

```bash
PORT=3000 ./run.sh
curl -fsS http://127.0.0.1:3000/api/v1/health
```

スマートフォンには `http://SERVER_IP:3000` のような端末から到達可能なアドレスを設定してください。

## 主なルート

すべて `/api/v1` プレフィックスを使用します。

| 分類 | メソッドとルート |
| --- | --- |
| ヘルス | `GET /health` |
| NetEase QR | `GET /music-sources/netease/qr`、`GET /music-sources/netease/qr/:key` |
| プロフィール | `POST /music-sources/netease/profile` |
| おすすめ | `POST /music-sources/netease/recommendations`、`/daily-songs` |
| プレイリスト | `POST /music-sources/netease/playlists`、`/playlists/detail`、`/playlists/create` |
| 検索 | `POST /music-sources/netease/search` |
| 歌詞 | `POST /music-sources/netease/lyrics` |
| コメント | `POST /music-sources/netease/comments` |
| 再生 | `POST /music-sources/netease/play-url`、`GET /music-sources/netease/stream/:token` |
| Bilibili | `POST /music-sources/bilibili/validate-cookie`、`/search`、`/play-url` |

コメント API は閲覧専用です。

## ライセンス

本プロジェクトは [MIT License](LICENSE) で公開されています。ライセンス条件に従い、利用、複製、変更、統合、公開、配布、再許諾、販売が可能です。本ソフトウェアに保証はありません。