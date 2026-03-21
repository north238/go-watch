# Project Design Document

GoWatch — Web Site Monitoring Tool

Created: 2026-03-17 | Status: Design Complete | Target: 2 weeks

## **1\. Overview**

Go \+ React で構築するポートフォリオプロジェクト。登録した URL を Go の goroutine で定期的に並行ヘルスチェックし、ステータスとレスポンスタイムを WebSocket でリアルタイムにダッシュボード表示する Web アプリケーション。

### Purpose

- Go の並行処理（goroutine / channel / context）を実践的に学ぶ

- 「なぜこれを作ったのか」を語れる題材: 自作アプリの監視という具体的な動機

- 運用・インフラを意識した設計力をポートフォリオで見せる

### Why a monitoring tool?

監視ツールは Go の並行処理が「飾り」ではなく「必然」になる設計。起動から停止まで goroutine が常に動き続けるため、起動・管理・停止の全工程を学べる。また、本番稼働中の自作資産管理アプリが監視対象として存在するため、ストーリーの一貫性がある。

## **2\. Tech Stack**

| Layer     | Technology                 | Notes                                      |
| :-------- | :------------------------- | :----------------------------------------- |
| Backend   | Go (chi or net/http)       | goroutine \+ channel \+ errgroup \+ ticker |
| WebSocket | gorilla/websocket          | ステータス変化をリアルタイム push          |
| Frontend  | React \+ TypeScript (Vite) | ダッシュボード UI                          |
| DB        | SQLite                     | チェック履歴の保存                         |
| Infra     | Docker Compose             | Go \+ React                                |

## **3\. MVP Features**

1. URL 登録 / 削除 — 監視対象の URL 管理（CRUD）

2. 定期ヘルスチェック — time.Ticker で 30秒ごとに全 URL を Worker Pool で並行チェック

3. リアルタイムステータス — 各 URL の状態 (UP / DOWN / SLOW) を WebSocket で 1 件ずつ push

4. レスポンスタイム記録 — SQLite に履歴保存、React 側で推移グラフ表示

5. アラート表示 — DOWN に変わったタイミングで UI 上に通知

## **4\. Out of Scope (MVP)**

以下は MVP では対象外とし、完成後の拡張候補とする。

- メール / Slack 通知
- ユーザー認証
- SSL 証明書チェック
- 本番デプロイ
- テストの網羅（主要部分のみ）
- 手動実行（任意タイミングでのチェック起動）

> ### **手動実行を除外した理由**
>
> 30秒間隔での自動チェックで十分な用途を想定している。手動実行を加えると Ticker 以外のトリガーが生まれ、サイクル重複防止の制御と UI 設計が複雑になる。トレードオフとして意図的に除外している。
>
> ### **Interface を事前定義しない理由**
>
> 通知先（WebSocket / メール / Slack）やチェック種別（HTTP / SSL）の Interface はあえて定義していない。Go の慣習では Interface は消費側が必要になったとき（= テスト時にモックが必要になったとき）に定義する。実装が 1 つしかない MVP 段階での抽象化はコストだけかかる。Go の Interface は後付けで切れる（implements 宣言が不要）ため、追加が確定した時点で定義すれば十分。

## **5\. Architecture**

### **5.1 Worker Pool (Health Checker Core)**

ヘルスチェッカーの中核は Worker Pool パターン。同時実行数を制御し、監視対象が増えても外部サーバーへの負荷を一定に保つ。

### **Processing flow:**

```text
[Ticker: 30秒ごと]
↓
[サイクル実行中チェック] ← sync.Mutex でフラグ確認
│ 実行中なら即スキップ（前サイクル完了を待たない）
↓
[URLリスト取得] ← sync.RWMutex で読み取りロック
↓
[job channel に URL 投入]
↓
[Worker Pool (N本の goroutine)]
│ 各 worker が HTTP GET 実行
│ context.WithTimeout で個別タイムアウト
↓
[result channel に 1 件ずつ送信]
↓
[結果受信 goroutine]
├→ SQLite に保存
└→ WebSocket で即座に push（1件ずつ）
```

## **Design decisions**

### **Worker Pool vs fan-out/fan-in:**

- Worker Pool を採用。同時実行数を制御でき、監視対象の増加に対してスケールする。相手サーバーへの負荷配慮と自サービスのリソース消費の予測可能性が理由。

### **サイクル重複防止（sync.Mutex）:**

- Ticker は 30 秒ごとに無条件で発火するため、前のサイクルが終わっていない場合に次のサイクルが重複して起動する問題が発生する。バッチ処理の二重起動防止と同じ制御で、sync.Mutex によるフラグ管理で実行中なら即スキップする。Laravel の withoutOverlapping() と同じ考え方を Go で自前実装している。

### **手動実行を持たない理由:**

- 30 秒間隔での自動チェックで用途を満たす。手動実行を追加すると Ticker 以外のトリガーが生まれ、サイクル重複制御と UI 設計が複雑になる。トレードオフとして意図的に除外している。

### **Result delivery:**

- 1 件完了ごとに逐次送信。全完了待ちの一括送信より UI のリアルタイム感が向上する。

### **Timeout strategy:**

- context の階層構造を活用。アプリ全体 \> チェックサイクル \> 個別 URL の 3 層。SIGTERM で全階層が連鎖的にキャンセルされる。

### **5.2 Context Hierarchy**

```text
[アプリ全体の context] ← SIGTERM でキャンセル
↓
[1回のチェックサイクルの context] ← Ticker 周期内に収める
↓
[個別 URL の context] ← 1 URL あたり 5 秒タイムアウト
```

- graceful shutdown 時にアプリ全体の context がキャンセルされ、実行中の全チェックが自動的に中断される。Go の context 伝搬の典型的なパターン。

### **5.3 このツールの制約（設計の前提から来るもの）**

- 以下は MVP として割り切った除外項目ではなく、設計の前提から生まれる構造的な制約。面接で「このツールの限界は？」と聞かれた際に説明できる。

### **自分自身の死活監視ができない:**

- GoWatch 自体が落ちた場合、チェックが実行されないため検知できない。監視する側と監視される側が同一プロセスである以上、構造的に解決できない制約。外部から監視してもらう存在が必要になる。

### **スケールアウトができない:**

- SQLite はファイルベースのため、複数サーバーからの同時読み書きができない。同じ URL 群を複数台で分散チェックしたい場合は PostgreSQL 等への移行が必要になる。ただし、監視先が完全に別々の独立したデプロイであれば問題は発生しない。

## **6\. Directory Structure (Go Backend)**

Go コミュニティの慣習に沿った cmd/ \+ internal/ 構成。将来 CLI を追加する拡張性も確保

```text
gowatch/
├── cmd/
│   └── server/
│       └── main.go         # エントリーポイント, DI, graceful shutdown
├── internal/
│   ├── checker/
│   │   ├── checker.go      # Worker Pool + Ticker + サイクル重複防止
│   │   └── checker_test.go
│   ├── model/
│   │   └── model.go        # Target, CheckResult 等の型定義
│   ├── store/
│   │   ├── sqlite.go       # SQLite 操作 (CRUD + 履歴保存)
│   │   └── sqlite_test.go
│   ├── websocket/
│   │   ├── hub.go          # WebSocket 接続管理 + Broadcast
│   │   └── client.go       # 個別クライアント接続
│   └── handler/
│       ├── target.go       # URL 管理 API (CRUD)
│       ├── health.go       # ヘルスチェック結果 API
│       └── ws.go           # WebSocket ハンドシェイク
├── migrations/
│   └── 001_init.sql        # テーブル作成 SQL
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

### **Package responsibilities**

| Package            | Responsibility                                            | Dependencies                     |
| :----------------- | :-------------------------------------------------------- | :------------------------------- |
| cmd/server         | エントリーポイント, DI, graceful shutdown                 | 全パッケージ                     |
| internal/model     | 型定義のみ、ロジックなし                                  | なし                             |
| internal/checker   | Worker Pool, Ticker, ヘルスチェック実行, サイクル重複防止 | model, store, websocket          |
| internal/store     | SQLite への読み書き                                       | model                            |
| internal/websocket | 接続管理, Broadcast                                       | model                            |
| internal/handler   | HTTP ハンドラー (REST API)                                | model, store, websocket, checker |

- model がどこにも依存しないことで循環参照を防ぎ、テストもしやすくなる。

## **7\. Database Design (SQLite)**

### **7.1 targets table**

| Column       | Type                   | Notes                                        |
| :----------- | :--------------------- | :------------------------------------------- |
| id           | TEXT PK                | UUID                                         |
| url          | TEXT NOT NULL UNIQUE   | 監視対象 URL                                 |
| name         | TEXT NOT NULL          | 表示名                                       |
| interval_sec | INTEGER DEFAULT 30     | チェック間隔（MVP ではグローバル 30 秒固定） |
| status       | TEXT DEFAULT 'unknown' | 最新ステータス (up/down/slow)                |
| created_at   | DATETIME               | 作成日時                                     |
| updated_at   | DATETIME               | 更新日時                                     |

### **7.2 check_results table**

| Column           | Type                     | Notes                 |
| :--------------- | :----------------------- | :-------------------- |
| id               | INTEGER PK AUTOINCREMENT | 連番                  |
| target_id        | TEXT FK → targets.id     | ON DELETE CASCADE     |
| status           | TEXT NOT NULL            | up / down / slow      |
| status_code      | INTEGER                  | HTTP ステータスコード |
| response_time_ms | INTEGER NOT NULL         | レスポンスタイム (ms) |
| error            | TEXT                     | エラー内容            |
| checked_at       | DATETIME NOT NULL        | チェック日時          |

### **7.3 Index**

レスポンスタイム推移グラフで「特定 URL の直近 N 件」を頻繁に取得するため、複合インデックスを設定。

```SQL
CREATE INDEX idx_results_target_time
ON check_results(target_id, checked_at DESC);
```

### **7.4 Data Retention**

監視ツールは放置するとデータが無限に増える。MVP では各ターゲットの直近 1,000 件だけ残す簡易クリーンアップをチェック時に実行する。

## **8\. WebSocket Message Design**

全メッセージは type フィールドで種別を判定する JSON 形式。新しいメッセージタイプを追加する際もクライアント側の変更が最小で済む。WebSocket はサーバーからの一方向 push に特化し、URL の CRUD は REST API で行う。

### **8.1 Server → Client Messages**

| type            | Payload                                                                 | Trigger                |
| :-------------- | :---------------------------------------------------------------------- | :--------------------- |
| check_result    | target_id, url, name, status, status_code, response_time_ms, checked_at | 1 件のチェック完了毎   |
| cycle_start     | target_count, started_at                                                | チェックサイクル開始時 |
| cycle_complete  | total, up, down, slow, duration_ms, completed_at                        | チェックサイクル完了時 |
| targets_updated | targets\[\] (id, url, name, status)                                     | URL 追加・削除時       |

### **8.2 Communication Pattern**

- REST API: URL の CRUD（追加・削除・一覧取得・履歴取得）

- WebSocket: サーバーからの一方向 push（チェック結果・サイクル通知）

- React は REST で操作し、WebSocket でリアルタイム更新を受け取る

## **9\. React Frontend Design**

1 ページ構成のダッシュボード。MVP でルーティングは不要。

### **9.1 Component Structure**

```text
src/

├── App.tsx              # WebSocket 接続管理 + レイアウト
├── hooks/
│ └── useWebSocket.ts    # 接続・再接続・メッセージ振り分け
├── components/
│ ├── Header.tsx         # タイトル + 接続状態 + Add URL
│ ├── SummaryCards.tsx   # UP / DOWN / SLOW / Last cycle
│ ├── TargetTable.tsx    # 監視対象一覧テーブル
│ ├── TargetRow.tsx      # 1 行分
│ ├── ResponseChart.tsx  # レスポンスタイム推移グラフ
│ └── AddTargetModal.tsx # URL 追加モーダル
├── api/
│ └── client.ts          # REST API 呼び出し
└── types/
└── index.ts             # 型定義
```

### **9.2 Data Flow per Component**

| Component      | Data source                                   | Update trigger                 |
| :------------- | :-------------------------------------------- | :----------------------------- |
| SummaryCards   | WebSocket: cycle_complete                     | サイクル完了ごと               |
| TargetTable    | 初回: REST GET / 以降: WebSocket check_result | 1 件チェック完了ごと           |
| ResponseChart  | REST GET（履歴データ）                        | 行クリック時 \+ サイクル完了時 |
| AddTargetModal | REST POST                                     | ユーザー操作時                 |

### **9.3 State Management**

React の useReducer で十分。Redux / Zustand は不要

```ts
type State = {
  targets: Map<string, Target>; // id → 最新状態
  connected: boolean; // WebSocket 接続状態
  lastCycle: CycleComplete | null; // 直近のサイクル結果
  selectedTargetId: string | null; // グラフ表示対象
};
```

## **10\. Go Technical Highlights**

ポートフォリオで語るポイントとして以下を意識する。

| Element               | Detail                                                 |
| :-------------------- | :----------------------------------------------------- |
| Worker Pool           | 同時実行数制御、監視対象増加へのスケーラビリティ       |
| time.Ticker           | 定期実行ループ                                         |
| context 階層          | アプリ \> サイクル \> 個別 URL の 3 層タイムアウト制御 |
| graceful shutdown     | SIGTERM で全 goroutine を安全停止                      |
| sync.RWMutex          | 監視対象リストへの並行読み書き制御                     |
| sync.Mutex            | サイクル重複実行防止（バッチの二重起動防止と同じ制御） |
| goroutine リーク防止  | 長時間稼働プロセスでのメモリ管理                       |
| WebSocket × goroutine | 収集の進捗を都度クライアントに push                    |

### **sync パッケージの使い分け:**

- sync.RWMutex — URLリストの並行読み書き制御（読みは複数 goroutine が同時 OK）
- sync.Mutex — サイクルの重複実行防止（単純なフラグ保護）

## **11\. Development Schedule (2 Weeks)**

### **Week 1: Go backend**

| Day     | Task                                                                  |
| :------ | :-------------------------------------------------------------------- |
| Day 1-2 | プロジェクト構成 \+ URL 管理 API \+ SQLite セットアップ               |
| Day 3-4 | ヘルスチェッカー (Worker Pool \+ Ticker 定期実行 \+ サイクル重複防止) |
| Day 5   | WebSocket エンドポイント \+ graceful shutdown \+ context 制御         |

### **Week 2: React frontend \+ integration**

| Day     | Task                                       |
| :------ | :----------------------------------------- |
| Day 1-2 | WebSocket 接続 \+ ステータス一覧 UI        |
| Day 3-4 | レスポンスタイム推移グラフ \+ アラート表示 |
| Day 5   | README 整備 \+ デモ動画撮影                |

## **12\. Portfolio Story**

面接で語る際のストーリーを整理する。

### **「なぜ Go なのか？」**

監視ツールは起動から停止まで goroutine が常に動き続ける設計。goroutine の「起動・管理・停止」を全工程学べる題材として Go を選んだ。

### **「なぜこれを作ったのか？」**

自作の資産管理アプリを本番運用しており、その死活監視を自分で行いたかった。代表的な監視ツールはあるが、自作することで Go の並行処理を実践的に学ぶことに意義がある。

### **「なぜ Worker Pool？」**

監視対象が増えても同時リクエスト数を押さえられる。相手サーバーへの負荷配慮と、自サービスのリソース消費を予測可能にするため。

### **「サイクル重複はどう防いでいるか？」**

バッチ処理の二重起動防止と同じ考え方で、sync.Mutex によるフラグ管理で前のサイクルが終わっていない場合は次の発火をスキップする。Laravel の withoutOverlapping() と同じ問題を Go で自前実装している。

### **「このツールの限界は？」**

GoWatch 自体が落ちた場合は検知できない（監視する側と監視される側が同一プロセス）。また SQLite のためスケールアウトができないが、監視先が独立した別デプロイであれば問題は発生しない。

### **Appeal points**

- 運用を意識した設計: Worker Pool による負荷制御、graceful shutdown、データ肥大化対策、サイクル重複防止
- Go の並行処理プリミティブを網羅的に使用: goroutine, channel, context, sync.RWMutex, sync.Mutex, errgroup
- 自作サービスを自分で監視するという一貫したストーリー
- Docker Compose でマルチサービスを一括管理
- 設計の限界と除外理由をトレードオフとして説明できる
