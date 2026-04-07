import { api } from '../api/client';
import type { Target } from '../types';

type Props = {
  target: Target;
  onDelete: (id: string) => void;
  onSelect: (id: string) => void;
  isSelected: boolean;
};

const statusColor: Record<string, string> = {
  up: 'green',
  down: 'red',
  slow: 'orange',
  unknown: 'gray',
};

export function TargetRow({ target, onDelete, onSelect, isSelected }: Props) {
  const handleDelete = async () => {
    try {
      await api.deleteTarget(target.id);
      onDelete(target.id);
    } catch (e) {
      console.error('Failed to delete target', e);
    }
  };

  return (
    <tr
      onClick={() => onSelect(target.id)}
      style={{
        cursor: 'pointer',
        backgroundColor: isSelected
          ? '#f0f0ff'
          : target.status === 'down'
            ? '#fff0f0'
            : 'transparent',
      }}
    >
      <td style={{ padding: '8px' }}>{target.name}</td>
      <td style={{ padding: '8px' }}>{target.url}</td>
      <td style={{ padding: '8px', color: statusColor[target.status] }}>
        {target.status.toUpperCase()}
      </td>
      <td style={{ padding: '8px' }}>
        <button
          onClick={(e) => {
            e.stopPropagation();
            handleDelete();
          }}
        >
          削除
        </button>
      </td>
    </tr>
  );
}
