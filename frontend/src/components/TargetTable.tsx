import type { Target } from '../types';
import { TargetRow } from './TargetRow';

type Props = {
  targets: Target[];
  onDelete: (id: string) => void;
};

export function TargetTable({ targets, onDelete }: Props) {
  if (targets.length === 0) {
    return <p>監視対象がありません。URLを追加してください。</p>;
  }

  return (
    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
      <thead>
        <tr style={{ borderBottom: '1px solid #ccc' }}>
          <th style={{ textAlign: 'left', padding: '8px' }}>名前</th>
          <th style={{ textAlign: 'left', padding: '8px' }}>URL</th>
          <th style={{ textAlign: 'left', padding: '8px' }}>ステータス</th>
          <th style={{ textAlign: 'left', padding: '8px' }}>操作</th>
        </tr>
      </thead>
      <tbody>
        {targets.map((target) => (
          <TargetRow key={target.id} target={target} onDelete={onDelete} />
        ))}
      </tbody>
    </table>
  );
}
