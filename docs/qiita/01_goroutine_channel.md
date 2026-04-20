## はじめに

この記事は、URLヘルスモニタリングツール **GoWatch** を実装しながらGoの並行処理を学んだ記録です。

このシリーズを読み終えると、以下のようなツールを自分で実装できるようになります。

- 複数のURLを並列でヘルスチェックする
- チェック結果をリアルタイムでブラウザに通知する
- サーバーを安全に停止する（Graceful Shutdown）

GoWatchのソースコードはこちら → [GitHub](https://github.com/north238/go-watch)

---

GoWatchを作ったきっかけは2つあります。1つ目は以前ポートフォリオとして自作したWebアプリを自分で監視したかったこと、2つ目はGoを使った実践的な開発を通じて並行処理を深く学びたかったことです。

複数のURLを**同時に**チェックする必要があるため、並行処理が不可欠なプロジェクトです。GoWatchはそのユースケースとGoの並行処理を学ぶ題材として非常に相性がよいと感じています。

このシリーズでは以下を学びます。

- goroutineとchannel（本記事）
- Worker Poolパターンで並列数を制御する
- contextでキャンセルを伝播させる
- 実装してわかったハマりどころ

対象読者はGoの基本文法は知っているが並行処理はまだ自信がない方です。

---

## goroutineとは

goroutineはGoが提供する**軽量な並行実行の単位**です。

OSのスレッドと比べると非常に軽量で、数千〜数万のgoroutineを同時に起動しても問題ありません。起動方法はシンプルで、関数呼び出しの前に `go` をつけるだけです。

```go
func main() {
    go hello() // goroutineとして起動
    hello()    // 通常の呼び出し
}

func hello() {
    fmt.Println("hello")
}
```

ただしこのコードには問題があります。`main` 関数が終了するとすべてのgoroutineも強制終了するため、`go hello()` が実行される前にプログラムが終わってしまうことがあります。

goroutineの終了を待つ方法はいくつかありますが、その一つが次に紹介するchannelです。

---

## channelとは

channelはgoroutine間で**値を安全にやり取りするための通路**です。

Goには「メモリを共有して通信するのではなく、通信によってメモリを共有する」という設計思想があります。channelはその思想を体現した仕組みです。

### 基本的な使い方

```go
func main() {
    ch := make(chan string) // stringを渡すchannelを作成

    go func() {
        ch <- "hello" // channelに送信
    }()

    msg := <-ch // channelから受信（goroutineが送信するまでここで待つ）
    fmt.Println(msg) // "hello"
}
```

ポイントは受信側（`<-ch`）がブロッキングする点です。goroutineが値を送信するまでメイン処理はここで待機します。これによって先ほどの「mainが先に終わってしまう」問題を解決できます。

### buffered channel

`make(chan string)` はバッファなしのchannelです。送信側は受信側が受け取るまでブロックされます。

バッファを持たせると、受信側が準備できていなくても指定した数まで送信できます。

```go
ch := make(chan string, 3) // バッファサイズ3のchannel

ch <- "a" // 受信側がいなくてもブロックされない
ch <- "b"
ch <- "c"
// ch <- "d" // バッファが埋まるのでここでブロック
```

---

## GoWatchではどう使っているか

GoWatchのチェック処理を担う `checker` パッケージでは、goroutineとchannelを次のような形で使っています。

```text
[tickerLoop goroutine]
    │ 一定間隔でチェック対象URLを送信
    ▼
  jobs channel
    │
    ▼
[worker goroutine × N]
    │ URLにHTTPリクエストを送信
    ▼
  results channel
    │
    ▼
[resultLoop goroutine]
    │ 結果をDBに保存 & WebSocketで通知
```

複数のURLを並列でチェックするために `worker` goroutineを複数起動し、`jobs` channelを通じて仕事を渡しています。これをWorker Poolパターンと呼びます。

詳細は次回の記事で解説します。

---

## まとめ

この記事で学んだことは3つです。

- goroutineは `go` キーワードで起動できる軽量な並行実行の単位
- channelはgoroutine間で値を安全に渡すための仕組みで、受信側はブロッキングする
- GoWatchでは複数URLを並列チェックするためにgoroutineとchannelを組み合わせている

**次回** はWorker Poolパターンを取り上げます。goroutineを無制限に起動することの問題点と、GoWatchでどう制御しているかを実装コードを交えて解説します。

---

## 参考

- [A Tour of Go](https://go.dev/tour/)
- [Go by Example - Goroutines](https://gobyexample.com/goroutines)
- [Go by Example - Channels](https://gobyexample.com/channels)
- [pkg.go.dev/context](https://pkg.go.dev/context)

---

## 関連記事リンク

- [GoWatchを作りながら学ぶGoの並行処理 #1 — goroutineとchannelの基本](https://qiita.com/north238/items/b495549b326bb7a1a033)
- [GoWatchを作りながら学ぶGoの並行処理 #2 — Worker Poolパターンでgoroutineを制御する](https://qiita.com/north238/items/b04935ef461432c04e5a)
- [GoWatchを作りながら学ぶGoの並行処理 #3 — contextでキャンセルを伝播させる](https://qiita.com/north238/items/310531fd0d8052bd137c)
- [GoWatchを作りながら学ぶGoの並行処理 #4 — 実装してわかったハマりどころまとめ](https://qiita.com/north238/items/3149d00f1b3612725f41)
