# GoWatchを作りながら学ぶGoの並行処理 #4 — 実装してわかったハマりどころまとめ

## はじめに

前回はcontextを学びました。

- contextはキャンセルやタイムアウトの信号を複数のgoroutineに伝播させる仕組み
- `WithCancel` は任意のタイミングで、`WithTimeout` は時間経過で自動キャンセルする
- GoWatchでは3層のcontext階層でアプリ全体・サイクル・個別リクエストのライフサイクルを管理している

最終回の今回は、GoWatchを実装する中で実際にハマったポイントを3つ紹介します。どれも「動いているように見えるが実は問題がある」種類のバグで、Goに慣れていないと気づきにくいものです。

---

## ハマりどころ① deferをループ内で使ってはいけない

### 問題のコード

```go
// ❌ 問題のある実装
func (c *Checker) worker(ctx context.Context) {
    for {
        select {
        case url := <-c.jobs:
            resp, err := http.Get(url)
            if err != nil {
                continue
            }
            defer resp.Body.Close() // ← ここが問題
        }
    }
}
```

一見問題なさそうですが、これは深刻なリソースリークを引き起こします。

### なぜ問題なのか

`defer` は**その関数が返るときに実行される**仕組みです。ループの中で `defer` を使うと、ループが回るたびに `defer` が積み重なり、関数が終了するまで一切実行されません。

`worker()` は無限ループで動き続けるため、`resp.Body.Close()` は永遠に呼ばれません。その結果、HTTPレスポンスのBodyが開きっぱなしになり、コネクションを占有し続けます。

### 正しい書き方

```go
// ✅ 正しい実装
func (c *Checker) worker(ctx context.Context) {
    for {
        select {
        case url := <-c.jobs:
            resp, err := http.Get(url)
            if err != nil {
                continue
            }
            resp.Body.Close() // deferを使わず明示的に呼ぶ
        }
    }
}
```

ループ内では `defer` を使わず、使い終わったタイミングで明示的に呼びます。もしくは処理を別関数に切り出して `defer` を使う方法もあります。

```go
// ✅ 別関数に切り出す方法
func (c *Checker) check(ctx context.Context, url string) CheckResult {
    resp, err := http.Get(url)
    if err != nil {
        return CheckResult{URL: url, IsDown: true}
    }
    defer resp.Body.Close() // 関数が終わると確実に呼ばれる
    // ...
}
```

---

## ハマりどころ② ポインタ型と値型の罠（\*time.Ticker）

### 問題のコード

```go
// ❌ 問題のある実装
type Checker struct {
    ticker time.Ticker // 値型で持っている
}

func (c *Checker) tickerLoop(ctx context.Context) {
    c.ticker = *time.NewTicker(30 * time.Second)
    defer c.ticker.Stop()
    // ...
}
```

### なぜ問題なのか

`time.NewTicker()` はポインタ（`*time.Ticker`）を返します。これを値型（`time.Ticker`）にコピーすると、内部のchannelや状態が意図しない形でコピーされます。

Goでは**内部にchannelやmutexを持つ型は値コピーしてはいけない**というルールがあります。コピーした瞬間に元のTickerと内部状態が乖離し、予期しない動作を引き起こします。

### 正しい書き方

```go
// ✅ 正しい実装
type Checker struct {
    ticker *time.Ticker // ポインタ型で持つ
}

func (c *Checker) tickerLoop(ctx context.Context) {
    c.ticker = time.NewTicker(30 * time.Second)
    defer c.ticker.Stop()
    // ...
}
```

ポインタで持つことで、内部状態を共有したまま同じTickerを参照し続けられます。

一般的な判断基準として、`New〇〇()` がポインタを返す型はポインタで持つと覚えておくと安全です。

---

## ハマりどころ③ goroutineリークを防ぐ考え方

### goroutineリークとは

goroutineリークとは、**不要になったgoroutineが終了せずにメモリを占有し続ける**状態です。

よくある原因は「受信されることのないchannelを待ち続けるgoroutine」です。

```go
// ❌ リークするコード
func leak() {
    ch := make(chan int)
    go func() {
        val := <-ch // 誰も送信しないので永遠に待ち続ける
        fmt.Println(val)
    }()
    // chに何も送らずに関数が終わる
}
```

このgoroutineはプログラムが終了するまでメモリに残り続けます。

### GoWatchでどう防いでいるか

GoWatchでは2つの方針でgoroutineリークを防いでいます。

**① すべてのgoroutineにctxを渡す**

```go
func (c *Checker) worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done(): // アプリ終了時に必ずここに入る
            return
        case url := <-c.jobs:
            // 処理
        }
    }
}
```

`ctx.Done()` を監視することで、アプリケーション終了時にすべてのgoroutineが確実に終了します。

**② channelのクローズを明示的に管理する**

```go
// tickerLoopが終了するときにjobsをクローズする
func (c *Checker) tickerLoop(ctx context.Context) {
    defer close(c.jobs)
    // ...
}
```

送信側がchannelをクローズすると、受信側の `for range ch` は自動的に終了します。これによりworkerが無限に待ち続ける状態を防げます。

---

## このシリーズを振り返って

4回にわたってGoWatchの実装を通じてGoの並行処理を学びました。

| 回  | テーマ              | 学んだこと                                      |
| --- | ------------------- | ----------------------------------------------- |
| #1  | goroutine / channel | 並行処理の基本単位とデータの渡し方              |
| #2  | Worker Pool         | goroutineの数を制御して安全に並列処理する       |
| #3  | context             | キャンセルとタイムアウトをgoroutineに伝播させる |
| #4  | ハマりどころ        | defer・ポインタ・goroutineリークの落とし穴      |

並行処理は「動いているように見えるが実は問題がある」バグが多い領域です。今回紹介した3つのハマりどころはどれも実際に踏んだものなので、同じところで詰まっている方の参考になれば嬉しいです。

### 次に学ぶとよいこと

このシリーズで扱えなかったテーマとして以下が挙げられます。

- **sync.RWMutex** — 読み取りと書き込みを分けてロック効率を上げる
- **goroutineリークのテスト** — `go.uber.org/goleak` を使った自動検出
- **sync.WaitGroup** — 複数のgoroutineの完了を待つ別のアプローチ

GoWatchのソースコードはこちら → GitHub（リンクは後ほど追記）

最後まで読んでいただきありがとうございました。

---

## 参考

- [Go by Example - Goroutines](https://gobyexample.com/goroutines)
- [Go by Example - Channels](https://gobyexample.com/channels)
- [Go by Example - Worker Pools](https://gobyexample.com/worker-pools)
- [Go by Example - Context](https://gobyexample.com/context)
- [pkg.go.dev/context](https://pkg.go.dev/context)
- [DigitalOcean - How To Use Contexts in Go](https://www.digitalocean.com/community/tutorials/how-to-use-contexts-in-go)
