package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"gowatch/internal/model"
	"gowatch/internal/store"
	"gowatch/internal/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

type Notifier interface {
	Notify(message string) error
}

type Checker struct {
	workNum       int
	jobChannel    chan model.Target
	resultChannel chan model.CheckResult
	store         *store.Store
	ticker        *time.Ticker
	mu            sync.Mutex
	running       bool
	hub           *websocket.Hub
	cycleStart    time.Time
	cycleUp       int
	cycleDown     int
	cycleSlow     int
	cycleExpected int
	cycleDone     int
	notifier      Notifier
}

func New(workNum int, store *store.Store, hub *websocket.Hub, notifier Notifier) *Checker {
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
		hub:           hub,
		notifier:      notifier,
	}
}

func (c *Checker) Start(ctx context.Context) {
	c.ticker = time.NewTicker(30 * time.Second)
	// 1. Workerをgoroutineで起動する（workerNum分）
	for i := 0; i < c.workNum; i++ {
		go c.worker(ctx)
	}

	// 2. Tickerを起動してgoroutineでループする
	go c.tickerLoop(ctx)

	// 3. resultChannelを受け取るループをgoroutineで起動する
	go c.resultLoop(ctx)
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

			cycleCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			req, err := http.NewRequestWithContext(cycleCtx, http.MethodGet, target.URL, nil)
			if err != nil {
				cancel()
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

				// 通知処理
				notifyMsg := fmt.Sprintf("DOWN: %s", target.URL)
				notifyErr := c.notifier.Notify(notifyMsg)
				if notifyErr != nil {
					log.Printf("feild to send notify: %v", notifyErr)

					// フロントエンドに情報を通知
					msg := model.WSMessage{
						Type:    "notification_error",
						Payload: "エラー通知設定のURLを確認してください",
					}
					wsMsg, jsonErr := json.Marshal(msg)
					if jsonErr != nil {
						log.Printf("marshal notification_error: %v", jsonErr)
					} else {
						c.hub.Broadcast(wsMsg)
					}
				}

				cancel()
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
			cancel()
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
			c.mu.Lock()
			if c.running {
				c.mu.Unlock()
				continue
			}
			c.running = true
			c.mu.Unlock()

			cycleCtx, cancel := context.WithTimeout(ctx, 25*time.Second)

			targets, err := c.store.ListTargets(cycleCtx)
			if err != nil {
				cancel()

				c.mu.Lock()
				c.running = false
				c.mu.Unlock()

				continue
			}

			now := time.Now()

			cycleStart := model.CycleStart{
				TargetCount: len(targets),
				StartedAt:   now,
			}

			msg := model.WSMessage{
				Type:    "cycle_start",
				Payload: cycleStart,
			}
			message, err := json.Marshal(msg)
			if err != nil {
				log.Printf("marshal cycle_start: %v", err)
			} else {
				c.hub.Broadcast(message)
			}

			for _, target := range targets {
				c.jobChannel <- target
			}

			cancel()

			c.mu.Lock()
			c.cycleStart = now
			c.cycleExpected = len(targets)
			c.cycleDone = 0
			c.cycleUp = 0
			c.cycleDown = 0
			c.cycleSlow = 0
			c.mu.Unlock()
		}
	}
}

// 返却値を元にDB更新
func (c *Checker) resultLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case result := <-c.resultChannel:
			// Hubへ送信
			msg := model.WSMessage{
				Type:    "check_result",
				Payload: result,
			}
			message, err := json.Marshal(msg)
			if err != nil {
				log.Printf("marshal result: %v", err)
				continue
			}
			c.hub.Broadcast(message)

			// 保存処理
			if err := c.store.SaveCheckResult(ctx, result); err != nil {
				log.Printf("saver check result: %v", err)
				continue
			}

			// ステータス更新
			if err := c.store.UpdateTargetStatus(ctx, result.TargetID, result.Status); err != nil {
				log.Printf("update target status: %v", err)
				continue
			}

			// 1,000件超過分削除
			if err := c.store.DeleteOldCheckResults(ctx, result.TargetID); err != nil {
				log.Printf("delete old check result: %v", err)
				continue
			}

			// サイクル集計
			c.mu.Lock()
			switch result.Status {
			case model.StatusUp:
				c.cycleUp++
			case model.StatusDown:
				c.cycleDown++
			case model.StatusSlow:
				c.cycleSlow++
			}
			c.cycleDone++
			done := c.cycleDone
			expected := c.cycleExpected
			cycleStart := c.cycleStart
			up := c.cycleUp
			down := c.cycleDown
			slow := c.cycleSlow
			c.mu.Unlock()

			// 全件揃ったらcycle_completeを送る
			if expected > 0 && done >= expected {
				c.mu.Lock()
				c.running = false
				c.mu.Unlock()

				cycleComplete := model.CycleComplete{
					Total:       expected,
					Up:          up,
					Down:        down,
					Slow:        slow,
					DurationMs:  time.Since(cycleStart).Milliseconds(),
					CompletedAt: time.Now(),
				}
				msg := model.WSMessage{
					Type:    "cycle_complete",
					Payload: cycleComplete,
				}
				message, err := json.Marshal(msg)
				if err != nil {
					log.Printf("marshal cycle_complete: %v", err)
					continue
				} else {
					c.hub.Broadcast(message)
				}
			}
		}
	}
}
