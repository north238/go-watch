CREATE TABLE IF NOT EXISTS targets (
    id           TEXT PRIMARY KEY,
    url          TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL,
    interval_sec INTEGER NOT NULL DEFAULT 30,
    status       TEXT NOT NULL DEFAULT 'unknown',
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS check_results (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id        TEXT NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
    status           TEXT NOT NULL,
    status_code      INTEGER,
    response_time_ms INTEGER NOT NULL,
    error            TEXT,
    checked_at       DATETIME NOT NULL
);

-- レスポンスタイム推移グラフで「特定 URL の直近 N 件」を頻繁に取得するための複合インデックス
CREATE INDEX IF NOT EXISTS idx_results_target_time
    ON check_results(target_id, checked_at DESC);
