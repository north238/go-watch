import type { Target } from '../types';

const BASE_URL = 'http://localhost:8080';

export const api = {
  // URL一覧取得
  getTargets: async (): Promise<Target[]> => {
    const res = await fetch(`${BASE_URL}/api/targets`);
    if (!res.ok) throw new Error('Failed to fetch targets');
    return res.json();
  },

  // URL登録
  addTarget: async (url: string, name: string): Promise<Target> => {
    const res = await fetch(`${BASE_URL}/api/targets`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url, name }),
    });
    if (!res.ok) throw new Error('Failed to add target');
    return res.json();
  },

  // URL削除
  deleteTarget: async (id: string): Promise<void> => {
    const res = await fetch(`${BASE_URL}/api/targets/${id}`, {
      method: 'DELETE',
    });
    if (!res.ok) throw new Error('Failed to delete target');
  },

  // 履歴取得
  getHistory: async (id: string, limit = 100) => {
    const res = await fetch(`${BASE_URL}/api/targets/${id}/history?limit=${limit}`);
    if (!res.ok) throw new Error('Failed to fetch history');
    return res.json();
  },
};
