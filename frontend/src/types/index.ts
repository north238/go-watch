export type Status = 'up' | 'down' | 'slow' | 'unknown';

export type Target = {
  id: string;
  url: string;
  name: string;
  interval_sec: number;
  status: Status;
  created_at: string;
  updated_at: string;
};

export type CheckResult = {
  id: number;
  target_id: string;
  status: Status;
  status_code: number;
  response_time_ms: number;
  error?: string;
  checked_at: string;
};

export type WSMessage =
  | { type: 'check_result'; payload: CheckResult }
  | { type: 'cycle_start'; payload: CycleStart }
  | { type: 'cycle_complete'; payload: CycleComplete }
  | { type: 'targets_updated'; payload: Target[] }
  | { type: 'notification_error'; payload: string };

export type CycleStart = {
  target_count: number;
  started_at: string;
};

export type CycleComplete = {
  total: number;
  up: number;
  down: number;
  slow: number;
  duration_ms: number;
  completed_at: string;
};
