type Props = {
  connected: boolean;
  onAddClick: () => void;
};

export function Header({ connected, onAddClick }: Props) {
  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '16px',
        borderBottom: '1px solid #ccc',
      }}
    >
      <h1 style={{ margin: 0 }}>GoWatch</h1>
      <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
        <span>{connected ? '🟢 Connected' : '🔴 Disconnected'}</span>
        <button onClick={onAddClick}>+ Add URL</button>
      </div>
    </div>
  );
}
