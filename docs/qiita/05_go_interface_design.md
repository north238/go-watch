## はじめに

前回までの記事でGoWatchのコア機能（Worker Pool、context、graceful shutdown）を実装しました。

今回はMVPの実装対象外だったSlack通知機能を追加実装した際に体験した、**Goのinterface設計の考え方**を番外編として紹介します。

---

## Interfaceは「消費側が定義する」

GoWatchにSlack通知を追加しようとしたとき、最初に悩んだのは「interfaceをどこに定義するか」でした。

GoWatchでいうと「通知を送りたい」のは `checker` パッケージです。DOWN検知という出来事を知っているのは `checker` であり、通知が必要と判断するのも `checker` です。

そのため `Notifier` interfaceは `checker` パッケージの中に定義します。`notifier` パッケージ側には置きません。

### なぜ提供側に置いてはいけないのか

`notifier` パッケージにinterfaceを定義すると、`checker` は `notifier` をimportしなければなりません。これがテストに影響します。

```text
❌ 提供側にinterfaceを定義した場合

[checker] → import → [notifier]
                         ↑
                   interfaceがここにある

checkerのテストを書くとき
→ notifierパッケージをimportしなければならない
→ テスト中に本物のSlack APIが呼ばれてしまう
→ ネットワークなしでテストが動かない
```

```text
✅ 消費側にinterfaceを定義した場合

[checker] ← interfaceがここにある
    ↑
[notifier] checkerを知らない

checkerのテストを書くとき
→ notifierパッケージをimportしなくていい
→ Notifyメソッドを持つ偽物を自由に作れる
→ ネットワークなしでテストが動く
```

### Goの「暗黙的な実装」

Goのinterfaceは `implements` のような宣言が不要です。メソッドの形が合えば自動的に満たされます。

```go
// checkerパッケージ（消費側）にinterfaceを定義する
type Notifier interface {
    Notify(message string) error
}
```

`notifier` パッケージの `SlackNotifier` は `checker` パッケージのことを何も知らなくても、`Notify(message string) error` というメソッドを持っているだけで自動的に `Notifier` を満たします。

---

## 設計の判断

### 外部ライブラリは使わない

Slack専用の外部ライブラリは使いませんでした。GoWatchがやりたいのは「通知を送る」だけです。

Slack Incoming WebhookはHTTP POSTを1回叩くだけで通知が送れます。標準ライブラリの `net/http` で賄えるため、外部ライブラリを追加する理由がありません。

```text
外部ライブラリが向いているケース  → Slackを操作したい（チャンネル管理、メッセージ検索など）
標準ライブラリで十分なケース      → 通知を送るだけ ← 今回はこちら
```

「必要になったときに複雑にする」というGoの設計思想に沿った判断です。

### NopNotifier（何もしない実装）を用意する

`SLACK_WEBHOOK_URL` が未設定のときの対応として、nilを渡す代わりに「何もしない通知」を用意しました。

```go
// ❌ nilチェックを呼び出し箇所に書いた場合
if c.notifier != nil {
    c.notifier.Notify(msg)
}
// 通知箇所が増えるたびにnilチェックが必要になる

// ✅ NopNotifierを用意した場合
type NopNotifier struct{}
func (n *NopNotifier) Notify(message string) error { return nil }

// main.goで一度だけ切り替える → 呼び出し側はnilを気にしなくていい
```

### main.goで組み立てる

どの通知先を使うかの決定は `checker` の責務ではありません。`main.go` が環境変数を見て切り替え、`Checker` に注入します。

```go
// main.goでの組み立て
var n checker.Notifier
if webhookURL == "" {
    n = &notifier.NopNotifier{}
} else {
    n = notifier.NewSlackNotifier(webhookURL)
}
checker := checker.New(..., n)
```

`checker` は「通知できる何かが渡されている」という事実だけ知っていれば動作します。Slackなのかどうかは知りません。

### 通知失敗はシステムを止めない

GoWatchの目的は「監視すること」です。Slack APIが落ちていてもURLの死活監視は継続できます。

```text
致命的なエラー    → 処理を止める（例: DB接続失敗）
致命的でないエラー → ログ + フィードバックして処理を続ける（例: 通知失敗）
```

通知失敗時はログ出力とWebSocket経由のToast表示に留め、ヘルスチェック処理を継続しました。

---

## 拡張したときにわかる設計の価値

Discord通知を追加したい場合を考えてみます。

```text
✅ この設計の場合

internal/notifier/discord.go → 新規作成するだけ
cmd/server/main.go           → 切り替え箇所を変更するだけ
internal/checker/checker.go  → 変更不要

❌ checkerがSlackNotifierを直接知っていた場合

internal/notifier/discord.go → 新規作成
cmd/server/main.go           → 切り替え箇所を変更
internal/checker/checker.go  → 直接参照している箇所の修正が必要
```

`checker` がinterfaceを通じて通知先を知らない設計にしたことで、通知先の追加・変更が `checker` に影響しない構造となっています。

---

## まとめ

この記事で学んだことは3つです。

- interfaceは消費側が定義する — 使う側に必要な形を定義することで依存の方向が整い、テストがしやすくなる
- Goの暗黙的な実装 — `implements` 宣言が不要なため、パッケージ間の結合を最小限にできる
- 必要になったとき複雑化する — interfacとNopNotifierは、使う場面が出てきて初めて追加した

GoのinterfaceはMVP段階では定義せず、テストや拡張が必要になったタイミングで導入するのがGoらしい設計です。今回のSlack通知追加がまさにそのタイミングでした。

GoWatchのソースコードはこちら →[GitHub](https://github.com/north238/go-watch)

---

## 参考

- [Effective Go - Interfaces](https://go.dev/doc/effective_go#interfaces)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [pkg.go.dev/net/http](https://pkg.go.dev/net/http)

---

## 関連記事リンク

- [GoWatchを作りながら学ぶGoの並行処理 #1 — goroutineとchannelの基本](https://qiita.com/north238/items/b495549b326bb7a1a033)
- [GoWatchを作りながら学ぶGoの並行処理 #2 — Worker Poolパターンでgoroutineを制御する](https://qiita.com/north238/items/b04935ef461432c04e5a)
- [GoWatchを作りながら学ぶGoの並行処理 #3 — contextでキャンセルを伝播させる](https://qiita.com/north238/items/310531fd0d8052bd137c)
- [GoWatchを作りながら学ぶGoの並行処理 #4 — 実装してわかったハマりどころまとめ](https://qiita.com/north238/items/3149d00f1b3612725f41)
