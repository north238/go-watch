## はじめに

前回はgoroutineとchannelの基本を学びました。
▶ [前回記事](https://qiita.com/north238/items/b495549b326bb7a1a033)

- goroutineは `go` キーワードで起動できる軽量な並行実行の単位
- channelはgoroutine間で値を安全に渡すための通路

今回はその知識を活かして、GoWatchの中核となる **Worker Poolパターン** を解説します。

---

## goroutineを無制限に起動する問題

前回学んだようにgoroutineは非常に軽量です。では、監視対象のURLが増えるたびにgoroutineを1つずつ起動すればよいのでしょうか。

```go
// ❌ URLごとにgoroutineを起動する素朴な実装
for _, url := range urls {
    go check(url)
}
```

少数のURLであれば問題ありません。しかし監視対象が100件、1000件と増えたとき、同じ数だけgoroutineが同時に起動します。goroutineは軽量とはいえ無制限に起動すれば以下の問題が起きます。

- メモリを大量に消費する
- 外部サービスへのリクエストが集中してサーバーに負荷をかける
- リソースの枯渇によりシステム全体が不安定になる

並行処理の数を**意図的に制御する**仕組みが必要です。

---

## Worker Poolパターンとは

Worker Poolパターンは、あらかじめ決まった数のworker（goroutine）を起動しておき、jobsをchannelで渡すことで並列数を制御するパターンです。

```text
[送信側]
    │ jobsをchannelに送信
    ▼
  jobs channel
    │
    ├─ [worker 1]
    ├─ [worker 2]  ← 常にN個のworkerだけが動いている
    └─ [worker 3]
    │
    ▼
  results channel
    │
    ▼
[受信側]
    │ 結果を処理
```

workerの数を固定することで「同時に処理できるjobの上限」を制御できます。jobsがworkerの数より多くても、channelがバッファとして機能するため処理が詰まることなく順番にさばいていきます。

---

## GoWatchでの実装

### Checkerの構造体

GoWatchでは `Checker` 構造体がWorker Poolを管理しています。

```go
type Checker struct {
    db          *store.Store
    hub         *websocket.Hub
    jobs        chan string
    results     chan CheckResult
    workerCount int
}

func New(db *store.Store, hub *websocket.Hub, workerCount int) *Checker {
    return &Checker{
        db:          db,
        hub:         hub,
        jobs:        make(chan string, workerCount),
        results:     make(chan CheckResult, workerCount),
        workerCount: workerCount,
    }
}
```

`jobs` と `results` はどちらも `workerCount` をバッファサイズとしています。workerが処理できる数だけ先行して受け付けられるようにするためです。

### Start() — workerを起動する

```go
func (c *Checker) Start(ctx context.Context) {
    for i := 0; i < c.workerCount; i++ {
        go c.worker(ctx)
    }

    go c.resultLoop(ctx)
    go c.tickerLoop(ctx)
}
```

`Start()` が呼ばれると `workerCount` の数だけ `worker` goroutineを起動します。以降、workerの数は増えも減りもしません。

### worker() — jobsを受け取って処理する

```go
func (c *Checker) worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case url, ok := <-c.jobs:
            if !ok {
                return
            }
            result := c.check(ctx, url)
            c.results <- result
        }
    }
}
```

各workerは `jobs` channelを監視し続け、URLが送られてきたらHTTPリクエストを送信して結果を `results` channelに流します。`ctx.Done()` を監視しているのはアプリケーション終了時に安全にworkerを止めるためです。これはシリーズ#3で詳しく解説します。

### tickerLoop() — 定期的にjobsを送信する

```go
func (c *Checker) tickerLoop(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            targets, err := c.db.ListTargets(ctx)
            if err != nil {
                continue
            }
            for _, t := range targets {
                c.jobs <- t.URL
            }
        }
    }
}
```

30秒ごとにDBから監視対象URLを取得し、1件ずつ `jobs` channelに送信します。workerは受け取った順に処理するため、一度に大量のHTTPリクエストが走ることはありません。

---

## まとめ

この記事で学んだことは3つです。

- goroutineを無制限に起動するとリソースが枯渇するリスクがある
- Worker Poolパターンはworkerの数を固定することで並列数を制御する
- GoWatchでは `jobs` / `results` channelを介してworkerに仕事を渡している

**次回** はcontextを取り上げます。`ctx.Done()` が何者なのか、GoWatchのGraceful Shutdownでcontextがどう機能しているかを実装コードを交えて解説します。

---

## 参考

- [Go by Example - Worker Pools](https://gobyexample.com/worker-pools)
- [Go by Example - Channels](https://gobyexample.com/channels)
- [pkg.go.dev/context](https://pkg.go.dev/context)

---

## 関連記事リンク

- [GoWatchを作りながら学ぶGoの並行処理 #1 — goroutineとchannelの基本](https://qiita.com/north238/items/b495549b326bb7a1a033)
- [GoWatchを作りながら学ぶGoの並行処理 #2 — Worker Poolパターンでgoroutineを制御する](https://qiita.com/north238/items/b04935ef461432c04e5a)
- [GoWatchを作りながら学ぶGoの並行処理 #3 — contextでキャンセルを伝播させる](https://qiita.com/north238/items/310531fd0d8052bd137c)
- [GoWatchを作りながら学ぶGoの並行処理 #4 — 実装してわかったハマりどころまとめ](https://qiita.com/north238/items/3149d00f1b3612725f41)
