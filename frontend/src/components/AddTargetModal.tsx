import { useState } from 'react';
import { api } from '../api/client';
import type { Target } from '../types';

type Props = {
  onAdd: (target: Target) => void;
  onClose: () => void;
};

export function AddTargetModal({ onAdd, onClose }: Props) {
  const [url, setUrl] = useState('');
  const [name, setName] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async () => {
    if (!url || !name) {
      setError('URLと名前を入力してください');
      return;
    }
    try {
      const target = await api.addTarget(url, name);
      onAdd(target);
      onClose();
    } catch (e) {
      setError('登録に失敗しました');
    }
  };

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <div style={{ background: '#fff', padding: '24px', borderRadius: '8px', width: '400px' }}>
        <h2>Add URL</h2>
        {error && <p style={{ color: 'red' }}>{error}</p>}
        <div style={{ marginBottom: '12px' }}>
          <label>URL</label>
          <input
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://example.com"
            style={{ display: 'block', width: '100%', padding: '8px', marginTop: '4px' }}
          />
        </div>
        <div style={{ marginBottom: '12px' }}>
          <label>名前</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="My Site"
            style={{ display: 'block', width: '100%', padding: '8px', marginTop: '4px' }}
          />
        </div>
        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
          <button onClick={onClose}>キャンセル</button>
          <button onClick={handleSubmit}>追加</button>
        </div>
      </div>
    </div>
  );
}
