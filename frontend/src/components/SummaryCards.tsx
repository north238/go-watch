import type { Target, CycleComplete } from '../types';

type Props = {
  targets: Target[];
  lastCycle: CycleComplete | null;
};

export function SummaryCards({ targets, lastCycle }: Props) {
  const up = targets.filter((t) => t.status === 'up').length;
  const down = targets.filter((t) => t.status === 'down').length;
  const slow = targets.filter((t) => t.status === 'slow').length;

  return (
    <div style={{ display: 'flex', gap: '16px', padding: '16px' }}>
      <div
        style={{
          padding: '16px',
          border: '1px solid green',
          borderRadius: '8px',
          minWidth: '100px',
        }}
      >
        <div style={{ color: 'green', fontSize: '24px' }}>{up}</div>
        <div>UP</div>
      </div>
      <div
        style={{ padding: '16px', border: '1px solid red', borderRadius: '8px', minWidth: '100px' }}
      >
        <div style={{ color: 'red', fontSize: '24px' }}>{down}</div>
        <div>DOWN</div>
      </div>
      <div
        style={{
          padding: '16px',
          border: '1px solid orange',
          borderRadius: '8px',
          minWidth: '100px',
        }}
      >
        <div style={{ color: 'orange', fontSize: '24px' }}>{slow}</div>
        <div>SLOW</div>
      </div>
      {lastCycle && (
        <div style={{ padding: '16px', border: '1px solid #ccc', borderRadius: '8px' }}>
          <div style={{ fontSize: '12px', color: '#666' }}>最終サイクル</div>
          <div>{new Date(lastCycle.completed_at).toLocaleTimeString()}</div>
        </div>
      )}
    </div>
  );
}
