# 0001_worker_pool

## Status

Accepted

## Date

2026-03-22

## Context

GoWatch は time.Ticker で 30 秒ごとに登録済みの全 URL をヘルスチェックする。
監視対象の URL 数が増えたとき、並行処理の方法によっては外部サーバーへの
リクエストが一度に集中し、相手サーバーへの負荷が制御できなくなる問題がある。
また、自サービス側のリソース消費も予測できなくなる。

どのパターンで並行ヘルスチェックを実装するかを決める必要があった。

## Decision

Worker Pool パターンを採用し、同時実行する goroutine 数を N 本に固定する。

job channel に URL を投入し、あらかじめ起動した N 本の worker goroutine が
順次取り出して HTTP GET を実行する。結果は result channel 経由で受信 goroutine
に渡し、SQLite への保存と WebSocket への push を行う。

## Alternatives Considered

**fan-out / fan-in**
URL ごとに goroutine を起動する方法。実装はシンプルになるが、
監視対象が増えるほど同時 goroutine 数が増え、外部サーバーへの
リクエストが集中する。同時実行数の上限が制御できないため却下した。

## Consequences

### Good

- 監視対象が増えても同時リクエスト数が N 本に固定され、外部サーバーへの負荷が一定に保たれる
- 自サービス側の goroutine 数とリソース消費が予測可能になる
- N の値を調整するだけでスケールの調整ができる

### Bad

- fan-out/fan-in より実装がやや複雑になる（job channel / result channel の両方を管理する必要がある）

## Notes

- 面接で「なぜ Worker Pool？」と聞かれたときの答えは「負荷制御とリソースの予測可能性」
- 相手サーバーへの配慮という運用視点が設計に入っていることをアピールできる
- goroutine リーク防止（context によるキャンセル）と合わせて語ると説得力が増す
