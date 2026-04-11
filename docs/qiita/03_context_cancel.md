# GoWatchを作りながら学ぶGoの並行処理 #3 — contextでキャンセルを伝播させる

## はじめに

前回はWorker Poolパターンを学びました。

- goroutineを無制限に起動するとリソースが枯渇するリスクがある
- Worker Poolパターンはworkerの数を固定することで並列数を制御する
- GoWatchでは `jobs` / `results` channelを介してworkerに仕事を渡している

今回は `worker()` のコードに登場した `ctx.Done()` の正体を解説します。contextを理解することでGoWatchのGraceful Shutdownがどう機能しているかが見えてきます。

---

## contextとは

contextはGoの標準パッケージで、**キャンセルやタイムアウトの信号を複数のgoroutineに伝播させる**仕組みです。

Webアプリケーションでよくあるケースを考えてみます。

- HTTPリクエストが来てDBクエリを実行中にクライアントが切断した
- 外部APIへのリクエストが5秒以上かかっている

こういったとき、処理を続けても意味がありません。contextを使うと「もう処理をやめてよい」という信号を関連するすべてのgoroutineに一斉に伝えられます。

### ctx.Done() とは

`ctx.Done()` はchannelを返します。contextがキャンセルされるとこのchannelが閉じられます。

```go
select {
case <-ctx.Done():
    // キャンセルされたので処理を終了する
    return
}
```

前回の `worker()` で `ctx.Done()` を監視していたのはこのためです。アプリケーション終了時にcontextがキャンセルされると、すべてのworkerがこのcaseに入って安全に終了します。

---

## WithCancel / WithTimeout の使い分け

contextは親から子へ派生させて使います。代表的な2つを見ていきます。

### context.WithCancel

任意のタイミングでキャンセルできるcontextを作ります。

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    // ctxを受け取って処理する
    doSomething(ctx)
}()

// 何らかの条件でキャンセル
cancel()
```

`cancel()` を呼ぶと派生したすべてのcontextに即座にキャンセルが伝播します。

### context.WithTimeout

指定した時間が経過すると自動でキャンセルされるcontextを作ります。

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// 5秒以内に完了しなければctxがキャンセルされる
result, err := http.Get("https://example.com")
```

外部へのHTTPリクエストやDBクエリなど「時間がかかりすぎたら止める」用途に使います。

---

## GoWatchでのcontext階層

GoWatchでは3層のcontext階層を組んでいます。

```
[appCtx]  signal.NotifyContext — SIGTERMで自動キャンセル
    │
    └─ [cycleCtx]  WithCancel — 1チェックサイクル全体を管理
            │
            └─ [reqCtx]  WithTimeout(5s) — 1URLへのHTTPリクエストを管理
```

### appCtx — アプリケーション全体

```go
appCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
defer stop()
```

`signal.NotifyContext` はOSからの終了シグナル（SIGTERM / SIGINT）を受け取ると自動でキャンセルするcontextを作ります。`Ctrl+C` を押したときやDockerコンテナの停止時に発火します。

### cycleCtx — 1チェックサイクル全体

```go
cycleCtx, cycleCancel := context.WithCancel(appCtx)
defer cycleCancel()
```

1回のチェックサイクル（全URLへのリクエスト一巡）を管理します。`appCtx` から派生しているため、アプリケーションが終了するとこのcontextも連動してキャンセルされます。

### reqCtx — 1URLへのリクエスト

```go
reqCtx, reqCancel := context.WithTimeout(cycleCtx, 5*time.Second)
defer reqCancel()
```

個々のHTTPリクエストに5秒のタイムアウトを設定します。レスポンスが遅いURLがあっても他のチェックを止めません。

---

## Graceful Shutdown

Graceful Shutdownとは、終了シグナルを受け取ったあと**処理中のリクエストを完了させてから安全に停止する**仕組みです。

GoWatchでは以下の流れで実現しています。

```
① Ctrl+C / SIGTERM を受信
        │
        ▼
② appCtx がキャンセルされる
        │
        ▼
③ cycleCtx → reqCtx へキャンセルが伝播
        │
        ▼
④ worker() の ctx.Done() が発火して全workerが終了
        │
        ▼
⑤ HTTPサーバーが Shutdown(ctx) で既存リクエストの完了を待って停止
```

```go
<-appCtx.Done() // SIGTERMを待つ

shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer shutdownCancel()

// 10秒以内に既存リクエストを処理し終えてから停止
if err := srv.Shutdown(shutdownCtx); err != nil {
    log.Printf("server shutdown error: %v", err)
}
```

`appCtx` がキャンセルされた瞬間に処理を強制終了するのではなく、進行中の処理が終わるのを最大10秒待ってから停止します。

---

## まとめ

この記事で学んだことは3つです。

- contextはキャンセルやタイムアウトの信号を複数のgoroutineに伝播させる仕組み
- `WithCancel` は任意のタイミングで、`WithTimeout` は時間経過で自動キャンセルする
- GoWatchでは3層のcontext階層でアプリ全体・サイクル・個別リクエストのライフサイクルを管理している

**次回** は実装を通じてハマったポイントをまとめます。`defer` のループ内での誤用、ポインタ型と値型の罠、goroutineリークを防ぐ考え方を解説します。

---

## 参考

- [pkg.go.dev/context](https://pkg.go.dev/context)
- [Go by Example - Context](https://gobyexample.com/context)
- [DigitalOcean - How To Use Contexts in Go](https://www.digitalocean.com/community/tutorials/how-to-use-contexts-in-go)
