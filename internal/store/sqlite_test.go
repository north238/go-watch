package store_test

import (
	"context"
	"database/sql"
	"gowatch/internal/model"
	"gowatch/internal/store"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		dbPath        string
		migrationPath string
		want          *store.Store
		wantErr       bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := store.New(tt.dbPath, tt.migrationPath)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("New() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("New() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_Close(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		dbPath        string
		migrationPath string
		wantErr       bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := store.New(tt.dbPath, tt.migrationPath)
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			gotErr := s.Close()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Close() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Close() succeeded unexpectedly")
			}
		})
	}
}

func TestStore_DB(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		dbPath        string
		migrationPath string
		want          *sql.DB
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := store.New(tt.dbPath, tt.migrationPath)
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			got := s.DB()
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("DB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_AddTarget(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		tarName string
		preload bool
		wantErr bool
	}{
		{
			name:    "正常登録",
			url:     "https://example.com",
			tarName: "Example",
			preload: false,
			wantErr: false,
		},
		{
			name:    "重複URL",
			url:     "https://example.com",
			tarName: "Example2",
			preload: true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := store.New(":memory:", "../../migrations/001_init.sql")
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			defer s.Close()

			// preloadがtrueのときだけ事前登録する
			if tt.preload {
				_, _ = s.AddTarget(context.Background(), "https://example.com", "Example")
			}

			_, err = s.AddTarget(context.Background(), tt.url, tt.tarName)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddTarget() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_ListTargets(t *testing.T) {
	tests := []struct {
		name      string
		preloads  []string // 事前登録するURLのリスト
		wantCount int      // 期待する件数
		wantErr   bool
	}{
		{
			name:      "0件のとき空スライスを返す",
			preloads:  []string{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "登録済みのURLを全件返す",
			preloads:  []string{"https://example.com", "https://example2.com"},
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := store.New(":memory:", "../../migrations/001_init.sql")
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			defer s.Close()

			// 事前登録
			for _, url := range tt.preloads {
				_, err := s.AddTarget(context.Background(), url, "test")
				if err != nil {
					t.Fatalf("preload failed: %v", err)
				}
			}

			got, err := s.ListTargets(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ListTargets() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("ListTargets() count = %v, want %v", len(got), tt.wantCount)
			}
		})
	}
}

func TestStore_DeleteTarget(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		preload    bool
		useValidId bool
		wantErr    bool
	}{
		{
			name:       "削除成功",
			preload:    true,
			useValidId: true,
			wantErr:    false,
		},
		{
			name:       "削除対象なし",
			preload:    false,
			useValidId: false,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := store.New(":memory:", "../../migrations/001_init.sql")
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			defer s.Close()

			id := "non-existent-id"
			if tt.preload {
				target := mustAddTarget(t, s, "https://exapmle.com")
				if tt.useValidId {
					id = target.ID
				}
			}

			err = s.DeleteTarget(context.Background(), id)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteTarget() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// テスト用のヘルパー関数
func mustAddTarget(t *testing.T, s *store.Store, url string) model.Target {
	t.Helper()
	target, err := s.AddTarget(context.Background(), url, "test")
	if err != nil {
		t.Fatalf("failed to add target: %v", err)
	}
	return target
}
