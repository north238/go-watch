# GoWatch — GitHub Issues

設計ドキュメントに基づく実装タスク一覧。
開発スケジュール (2 Weeks) の Day 単位で Issue を分割。

---

## Issue #1: プロジェクト構成 + SQLite セットアップ

**Labels:** `Week1` `backend` `Day 1-2`

### 説明

GoWatch プロジェクトの骨格を作成する。
Go モジュールの初期化、ディレクトリ構成 (`cmd/` + `internal/`)、SQLite のテーブル作成、
および共有型定義 (`internal/model`) を実装する。
この Issue が完了すると、サーバーが起動して DB に接続できる状態になる。

### やること

- `go mod init` でモジュール作成
- 設計ドキュメント §6 に沿ったディレクトリ構成を作成
- `internal/model/model.go` — `Target`, `CheckResult` 等の型定義
- `migrations/001_init.sql` — `targets` テーブル + `check_results` テーブル + インデックス作成 (§7)
- `internal/store/sqlite.go` — DB 接続初期化、マイグレーション実行
- `cmd/server/main.go` — 最小限のエントリーポイント (DB 接続確認 → ログ出力 → 終了)
- `Dockerfile` + `docker-compose.yml` の雛形

### 受け入れ条件 (AC)

- [ ] `go build ./cmd/server` がエラーなく通る
- [ ] サーバー起動時に SQLite ファイルが作成され、`targets` / `check_results` テーブルが存在する
- [ ] `internal/model` が他パッケージに依存していない
- [ ] `docker compose build` が成功する

---

## Issue #2: URL 管理 REST API (CRUD)

**Labels:** `Week1` `backend` `Day 1-2`

### 説明

監視対象 URL の登録・取得・削除を行う REST API を実装する。
ルーター (chi or net/http) のセットアップと、`internal/store` の CRUD 操作、
`internal/handler/target.go` の HTTP ハンドラーをこの Issue で完成させる。

### やること

- `internal/store/sqlite.go` に CRUD メソッド追加
  - `AddTarget(ctx, url, name) → Target`
  - `ListTargets(ctx) → []Target`
  - `DeleteTarget(ctx, id) → error`
- `internal/handler/target.go` — HTTP ハンドラー
  - `POST /api/targets` — URL 登録 (URL バリデーション含む)
  - `GET /api/targets` — 一覧取得
  - `DELETE /api/targets/{id}` — 削除
- `cmd/server/main.go` にルーター登録を追加
- レスポンスは JSON 形式

### 受け入れ条件 (AC)

- [ ] `curl` で URL の登録・一覧取得・削除が正常に動作する
- [ ] 重複 URL の登録時に適切なエラーレスポンス (409) を返す
- [ ] 不正な URL 形式の登録時にバリデーションエラー (400) を返す
- [ ] 削除時に `check_results` が CASCADE で削除される

---

## Issue #3: ヘルスチェッカー — Worker Pool + Ticker 定期実行

**Labels:** `Week1` `backend` `Day 3-4`

### 説明

GoWatch の中核機能である Worker Pool パターンのヘルスチェッカーを実装する。
`time.Ticker` で 30 秒ごとにチェックサイクルを起動し、
job channel → Worker Pool → result channel の流れで並行にヘルスチェックを実行する。
結果は `check_results` テーブルに保存し、`targets.status` も更新する。

### やること

- `internal/checker/checker.go`
  - Worker Pool 構造体 (worker 数、job/result channel)
  - `Start(ctx)` — Ticker ループ開始 + Worker 起動
  - `Stop()` — 全 goroutine の安全な停止
  - 1 サイクルの処理: URL リスト取得 → job channel 投入 → Worker が HTTP GET → result channel に結果送信
  - ステータス判定ロジック: UP (2xx) / DOWN (エラー or 5xx) / SLOW (2xx かつ >2000ms)
- context 階層の実装 (§5.2)
  - サイクル context: Ticker 周期内に収める
  - 個別 URL context: 5 秒タイムアウト
- `sync.RWMutex` で URL リストへの安全なアクセス
- `internal/store` にチェック結果保存メソッド追加
  - `SaveCheckResult(ctx, result) → error`
  - `UpdateTargetStatus(ctx, targetID, status) → error`
- データ保持: 各ターゲットの直近 1,000 件を超えた古い結果を削除 (§7.4)

### 受け入れ条件 (AC)

- [ ] サーバー起動後、30 秒ごとにチェックサイクルが実行される
- [ ] チェック結果が `check_results` テーブルに保存される
- [ ] `targets.status` が最新のチェック結果で更新される
- [ ] context キャンセル時に実行中のチェックが中断される
- [ ] Worker Pool の goroutine が停止後にリークしていない (テストで確認)
- [ ] 1,000 件超過時に古いレコードが削除される

---

## Issue #4: WebSocket エンドポイント + リアルタイム push

**Labels:** `Week1` `backend` `Day 5`

### 説明

WebSocket の接続管理 (Hub パターン) を実装し、ヘルスチェック結果をクライアントにリアルタイム push する。
§8 のメッセージ設計に沿って、`check_result` / `cycle_start` / `cycle_complete` / `targets_updated` の 4 種類のメッセージを送信する。

### やること

- `internal/websocket/hub.go`
  - Hub 構造体: クライアント接続の管理 (register / unregister / broadcast)
  - `Run(ctx)` — Hub のメインループ
  - `Broadcast(message)` — 全クライアントへ JSON メッセージ送信
- `internal/websocket/client.go`
  - 個別クライアント接続の管理 (writePump)
  - 切断検知とクリーンアップ
- `internal/handler/ws.go` — WebSocket ハンドシェイクハンドラー
- `internal/checker` と `internal/websocket` の連携
  - チェック完了ごとに `check_result` メッセージ push
  - サイクル開始時に `cycle_start`、完了時に `cycle_complete` push
- URL 追加・削除時に `targets_updated` メッセージ push
- チェック結果履歴の取得 API
  - `GET /api/targets/{id}/history` — 指定 URL の直近 N 件の結果取得

### 受け入れ条件 (AC)

- [ ] `wscat` 等で WebSocket 接続が確立できる
- [ ] チェック完了ごとに `check_result` メッセージが届く
- [ ] サイクルの開始・完了メッセージが届く
- [ ] 複数クライアント接続時に全員にメッセージが配信される
- [ ] クライアント切断後に goroutine がリークしない
- [ ] 履歴 API がチェック結果を JSON で返す

---

## Issue #5: Graceful Shutdown + Context 制御

**Labels:** `Week1` `backend` `Day 5`

### 説明

SIGTERM / SIGINT を受けた際に全 goroutine (Ticker、Worker Pool、WebSocket Hub) を安全に停止する graceful shutdown を実装する。
§5.2 の context 階層をアプリ全体レベルで統合する。

### やること

- `cmd/server/main.go` で `signal.NotifyContext` を使ったシグナルハンドリング
- アプリ全体 context → Checker / WebSocket Hub に伝搬
- シャットダウンシーケンスの実装
  1. シグナル受信 → context キャンセル
  2. HTTP サーバーの graceful shutdown (`http.Server.Shutdown`)
  3. Checker 停止 (実行中サイクルの完了待ち)
  4. WebSocket Hub 停止 (全クライアント切断)
  5. DB 接続クローズ
- タイムアウト付き shutdown (全体で 10 秒以内に完了)

### 受け入れ条件 (AC)

- [ ] `kill -TERM <pid>` でサーバーが正常終了する
- [ ] シャットダウン中に実行中のチェックサイクルが完了してから終了する
- [ ] WebSocket 接続が切断される
- [ ] シャットダウン完了のログが出力される
- [ ] 10 秒以内に終了しない場合は強制終了する

---

## Issue #6: React — WebSocket 接続 + ステータス一覧 UI

**Labels:** `Week2` `frontend` `Day 1-2`

### 説明

React + TypeScript (Vite) のフロントエンドプロジェクトを作成し、
WebSocket 接続・URL 追加モーダル・ステータス一覧テーブル・サマリーカードを実装する。
§9 のコンポーネント設計に沿って構築する。

### やること

- Vite + React + TypeScript プロジェクトの初期化
- `src/types/index.ts` — 型定義 (Target, CheckResult, WebSocket メッセージ型)
- `src/hooks/useWebSocket.ts` — WebSocket 接続・自動再接続・メッセージ振り分け
- `src/api/client.ts` — REST API 呼び出し (fetch ラッパー)
- `App.tsx` — `useReducer` によるグローバル State 管理 (§9.3)
- コンポーネント実装
  - `Header.tsx` — タイトル + WebSocket 接続状態インジケーター + 「Add URL」ボタン
  - `SummaryCards.tsx` — UP / DOWN / SLOW の件数 + 最終サイクル情報
  - `TargetTable.tsx` + `TargetRow.tsx` — 監視対象一覧テーブル (ステータス、URL、レスポンスタイム、削除ボタン)
  - `AddTargetModal.tsx` — URL 追加フォーム (URL + 表示名)
- Docker Compose にフロントエンドサービスを追加

### 受け入れ条件 (AC)

- [ ] ブラウザで UI が表示される
- [ ] WebSocket 接続状態が画面上に表示される
- [ ] URL を追加するとテーブルに即座に反映される
- [ ] チェック結果が WebSocket 経由でリアルタイムに更新される
- [ ] サマリーカードがサイクル完了ごとに更新される
- [ ] URL を削除するとテーブルから消え、バックエンドからも削除される

---

## Issue #7: レスポンスタイム推移グラフ + アラート表示

**Labels:** `Week2` `frontend` `Day 3-4`

### 説明

監視対象の行クリックでレスポンスタイム推移グラフを表示する機能と、
ステータスが DOWN に変化した際の UI 通知 (アラート) を実装する。

### やること

- `ResponseChart.tsx`
  - テーブルの行クリックで対象 URL の履歴データを REST API で取得
  - レスポンスタイム推移を折れ線グラフで表示 (recharts 等のライブラリ使用)
  - サイクル完了時にグラフデータを自動更新
  - UP / DOWN / SLOW をグラフ上で色分け表示
- アラート表示
  - WebSocket の `check_result` でステータスが DOWN に変化したタイミングを検知
  - 画面上部にトースト通知を表示 (数秒で自動消去)
  - DOWN のターゲット行をハイライト表示

### 受け入れ条件 (AC)

- [ ] テーブル行クリックでレスポンスタイム推移グラフが表示される
- [ ] グラフがサイクル完了ごとに自動更新される
- [ ] DOWN 検知時にトースト通知が表示される
- [ ] DOWN の行が視覚的にハイライトされている
- [ ] グラフが空の場合 (履歴なし) に適切なメッセージが表示される

---

## Issue #8: README 整備 + Docker Compose 仕上げ + デモ準備

**Labels:** `Week2` `docs` `Day 5`

### 説明

ポートフォリオとして公開するための README 作成、Docker Compose の最終調整、
デモ動画の撮影準備を行う。§12 のポートフォリオストーリーを反映させる。

### やること

- `README.md`
  - プロジェクト概要 + スクリーンショット / GIF
  - 動機 (§12: なぜ Go か、なぜ監視ツールか)
  - アーキテクチャ概要図 (Worker Pool フロー)
  - 技術的ハイライト (§10 の Go 並行処理要素)
  - セットアップ手順 (`docker compose up` で起動)
  - 使い方 (URL 追加 → ダッシュボード確認)
  - 今後の拡張予定 (§4: Out of Scope 項目)
- Docker Compose 最終調整
  - Go バックエンド + React フロントエンドが `docker compose up` で一発起動
  - ヘルスチェック設定
  - ポート設定の確認
- デモ動画撮影
  - URL 追加 → リアルタイムステータス更新 → DOWN 検知アラート → グラフ表示 の一連の流れ

### 受け入れ条件 (AC)

- [ ] `docker compose up` だけでアプリ全体が起動する
- [ ] README に動機・アーキテクチャ・セットアップ手順・技術ハイライトが記載されている
- [ ] README のセットアップ手順に従って第三者が起動できる
- [ ] デモ動画 (or GIF) が README に埋め込まれている

---

## Issue #9: Slack 通知機能の追加

**Labels:** `backend` `feature`

### 説明

DOWN 検知時に Slack へ通知を送る機能を追加する。
通知先を将来 Discord 等に切り替え・追加できる設計にするため、
`internal/checker` に `Notifier` interface を定義し、`internal/notifier` に実装を置く。
通知失敗時はシステムを止めず、Toast でユーザーへフィードバックする。

### やること

- `internal/checker/checker.go` に `Notifier` interface を追加
  - `type Notifier interface { Notify(message string) error }`
  - DOWN 検知時に `Notifier.Notify()` を呼び出す
  - 通知失敗時はログ出力 + Hub 経由で `notification_error` メッセージを送信
- `internal/notifier/slack.go` — SlackNotifier 実装
  - Incoming Webhook を使った通知送信
  - `Notify(message string) error` メソッドを実装
- `internal/websocket/hub.go` に `notification_error` メッセージタイプを追加
- `cmd/server/main.go` で SlackNotifier を Checker に注入
  - `SLACK_WEBHOOK_URL` 環境変数から Webhook URL を取得
  - URL 未設定の場合は通知なしで起動
- `docker-compose.yml` に `SLACK_WEBHOOK_URL` 環境変数を追加

### 受け入れ条件 (AC)

- [ ] DOWN 検知時に Slack へ通知が届く
- [ ] Slack 通知が失敗してもヘルスチェックが継続される
- [ ] 通知失敗時に Toast が表示される
- [ ] `SLACK_WEBHOOK_URL` 未設定でもサーバーが正常起動する
- [ ] `checker` パッケージが `notifier` パッケージを import していない
- [ ] `go build ./cmd/server` がエラーなく通る

### 関連

- 設計ドキュメント §4 (Out of Scope → 拡張候補)
- 将来の拡張: `internal/notifier/discord.go` を追加するだけで Discord 通知に対応できる
