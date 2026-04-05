import { api } from '../api/client';
import type { Target } from '../types';

type Props = {
  target: Target;
  onDelete: (id: string) => void;
};

const statusColor: Record<string, string> = {
  up: 'green',
  down: 'red',
  slow: 'orange',
  unknown: 'gray',
};

export function TargetRow({ target, onDelete }: Props) {
  const handleDelete = async () => {
    try {
      await api.deleteTarget(target.id);
      onDelete(target.id);
    } catch (e) {
      console.error('Failed to delete target', e);
    }
  };

  return (
    <tr>
      <td>{target.name}</td>
      <td>{target.url}</td>
      <td style={{ color: statusColor[target.status] }}>{target.status.toUpperCase()}</td>
      <td>
        <button onClick={handleDelete}>削除</button>
      </td>
    </tr>
  );
}
