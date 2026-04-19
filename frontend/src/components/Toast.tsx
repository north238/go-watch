import { useEffect } from 'react';

type Props = {
  message: string;
  index: number;
  onClose: () => void;
};

export function Toast({ message, index, onClose }: Props) {
  useEffect(() => {
    const timer = setTimeout(onClose, 5000);
    return () => clearTimeout(timer);
  }, [onClose]);

  return (
    <div
      style={{
        position: 'fixed',
        top: `${16 + index * 70}px`,
        right: '16px',
        background: '#ef4444',
        color: '#fff',
        padding: '12px 24px',
        borderRadius: '8px',
        boxShadow: '0 4px 6px rgba(0,0,0,0.1)',
        zIndex: 1000,
      }}
    >
      🔴 {message}
    </div>
  );
}
