import { useEffect, useState } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { api } from '../api/client';
import type { CheckResult, CycleComplete } from '../types';

type Props = {
  targetId: string;
  targetName: string;
  lastCycle: CycleComplete | null;
};

const statusColor: Record<string, string> = {
  up: '#22c55e',
  down: '#ef4444',
  slow: '#f97316',
  unknown: '#9ca3af',
};

export function ResponseChart({ targetId, targetName, lastCycle }: Props) {
  const [history, setHistory] = useState<CheckResult[]>([]);

  useEffect(() => {
    api.getHistory(targetId, 50).then((data) => {
      // 古い順に並び替え
      setHistory([...data].reverse());
    });
  }, [targetId, lastCycle]);

  if (history.length === 0) {
    return <p>履歴データがありません。</p>;
  }

  const chartData = history.map((r) => ({
    time: new Date(r.checked_at).toLocaleTimeString(),
    response_time_ms: r.response_time_ms,
    status: r.status,
  }));

  return (
    <div style={{ padding: '16px' }}>
      <h3>{targetName} のレスポンスタイム推移</h3>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="time" tick={{ fontSize: 10 }} />
          <YAxis unit="ms" />
          <Tooltip formatter={(value) => [`${value}ms`, 'レスポンスタイム']} />
          <Line
            type="monotone"
            dataKey="response_time_ms"
            stroke="#6366f1"
            dot={(props) => {
              const { cx, cy, payload } = props;
              return (
                <circle
                  key={payload.time}
                  cx={cx}
                  cy={cy}
                  r={4}
                  fill={statusColor[payload.status]}
                />
              );
            }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
