package checker

import (
	"context"
	"gowatch/internal/model"
	"gowatch/internal/store"
	"net/http"
	"time"
)

type Checker struct {
	workNum       int
	jobChannel    chan model.Target
	resultChannel chan model.CheckResult
	store         *store.Store
	ctx           context.Context
	ticker        *time.Ticker
}

// func (c *Checker) runCycle() {
// 	cycleCtx, cancel := context.WithTimeout(c.ctx, 25*time.Second)
// 	defer cancel()

// 	urlCtx, cancel := context.WithTimeout(cycleCtx, 5*time.Second)
// 	defer cancel()
// }

func New(workNum int, store *store.Store) *Checker {
	// 1. jobの初期化
	job := make(chan model.Target, workNum)

	// 2. 返却値の初期化
	result := make(chan model.CheckResult, workNum)

	// 3. 構造そのものを初期化
	return &Checker{
		workNum:       workNum,
		jobChannel:    job,
		resultChannel: result,
		store:         store,
	}
}

func (c *Checker) Start(ctx context.Context) {
	// 1. Workerをgoroutineで起動する（workerNum分）
	for i := 0; i < c.workNum; i++ {
		go c.worker(ctx)
	}

	// 2. Tickerを起動してgoroutineでループする
	// 別メソッドで記述
	go c.tickerLoop(ctx)

	// 3. resultChannelを受け取るループをgoroutineで起動する
	// 別メソッドで記述
	// go getResultChan()
}

func (c *Checker) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// ctxがキャンセルされたら終了
			return
		case target := <-c.jobChannel:
			// targetを処理する
			// 確認したいURLを検証する
			start := time.Now()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
			if err != nil {
				continue
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				c.resultChannel <- model.CheckResult{
					TargetID:  target.ID,
					Status:    model.StatusDown,
					Error:     err.Error(),
					CheckedAt: time.Now(),
				}
				continue
			}
			resp.Body.Close()

			elapsed := time.Since(start).Milliseconds()

			status := c.judgeStatus(resp.StatusCode, elapsed)

			var result = model.CheckResult{
				TargetID:       target.ID,
				Status:         status,
				StatusCode:     resp.StatusCode,
				ResponseTimeMs: elapsed,
				Error:          "",
				CheckedAt:      time.Now(),
			}

			// 結果を送る
			c.resultChannel <- result
		}
	}
}

// ステータスを判定
func (c *Checker) judgeStatus(statusCode int, elapsed int64) model.Status {
	if statusCode >= 200 && statusCode < 300 {
		if elapsed > 2000 {
			return model.StatusSlow
		}
		return model.StatusUp
	}
	return model.StatusDown
}

func (c *Checker) tickerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.ticker.C:
			targets, err := c.store.ListTargets(ctx)
			if err != nil {
				continue
			}

			for _, target := range targets {
				c.jobChannel <- target
			}
		}
	}
}
