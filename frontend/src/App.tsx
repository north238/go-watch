import { useReducer, useEffect, useCallback, useState, useRef } from 'react';
import type { Target, CheckResult, WSMessage, CycleComplete } from './types';
import { api } from './api/client';
import { useWebSocket } from './hooks/useWebSocket';
import { Header } from './components/Header';
import { SummaryCards } from './components/SummaryCards';
import { TargetTable } from './components/TargetTable';
import { AddTargetModal } from './components/AddTargetModal';
import { ResponseChart } from './components/ResponseChart';
import { Toast } from './components/Toast';

type State = {
  targets: Map<string, Target>;
  connected: boolean;
  lastCycle: CycleComplete | null;
  selectedTargetId: string | null;
};

const initialState: State = {
  targets: new Map(),
  connected: false,
  lastCycle: null,
  selectedTargetId: null,
};

type Action =
  | { type: 'SET_TARGETS'; payload: Target[] }
  | { type: 'UPDATE_TARGET'; payload: CheckResult }
  | { type: 'ADD_TARGET'; payload: Target }
  | { type: 'DELETE_TARGET'; payload: string }
  | { type: 'SET_CONNECTED'; payload: boolean }
  | { type: 'SET_LAST_CYCLE'; payload: CycleComplete }
  | { type: 'SELECT_TARGET'; payload: string | null };

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'SET_TARGETS': {
      const map = new Map(action.payload.map((t) => [t.id, t]));
      return { ...state, targets: map };
    }
    case 'UPDATE_TARGET': {
      const result = action.payload;
      const target = state.targets.get(result.target_id);
      if (!target) return state;
      const updated = new Map(state.targets);
      updated.set(result.target_id, { ...target, status: result.status });
      return { ...state, targets: updated };
    }
    case 'ADD_TARGET': {
      const updated = new Map(state.targets);
      updated.set(action.payload.id, action.payload);
      return { ...state, targets: updated };
    }
    case 'DELETE_TARGET': {
      const updated = new Map(state.targets);
      updated.delete(action.payload);
      return { ...state, targets: updated };
    }
    case 'SET_CONNECTED':
      return { ...state, connected: action.payload };
    case 'SET_LAST_CYCLE':
      return { ...state, lastCycle: action.payload };
    case 'SELECT_TARGET':
      return { ...state, selectedTargetId: action.payload };
    default:
      return state;
  }
}

function App() {
  const [state, dispatch] = useReducer(reducer, initialState);
  const [showModal, setShowModal] = useState(false);
  const [toasts, setToasts] = useState<string[]>([]);
  const targets: Target[] = Array.from(state.targets.values());

  useEffect(() => {
    api.getTargets().then((targets) => {
      dispatch({ type: 'SET_TARGETS', payload: targets });
    });
  }, []);

  const targetsRef = useRef(state.targets);

  useEffect(() => {
    targetsRef.current = state.targets;
  }, [state.targets]);

  const handleMessage = useCallback((msg: WSMessage) => {
    switch (msg.type) {
      case 'check_result':
        // DOWN検知時にトースト表示
        if (msg.payload.status === 'down') {
          const target = targetsRef.current.get(msg.payload.target_id);
          setToasts(prev => [...prev, `${target?.name ?? '不明'} (${target?.url ?? msg.payload.target_id}) がDOWNしました`]);
        }
        dispatch({ type: 'UPDATE_TARGET', payload: msg.payload });
        break;
      case 'cycle_complete':
        dispatch({ type: 'SET_LAST_CYCLE', payload: msg.payload });
        break;
      case 'notification_error': {
        setToasts(prev => [...prev, `エラー： ${msg.payload}`]);
        break;
      }
    }
  }, []);

  useWebSocket({
    onMessage: handleMessage,
    onConnect: () => dispatch({ type: 'SET_CONNECTED', payload: true }),
    onDisconnect: () => dispatch({ type: 'SET_CONNECTED', payload: false }),
  });

  return (
    <div style={{ maxWidth: '1200px', margin: '0 auto' }}>
      <Header connected={state.connected} onAddClick={() => setShowModal(true)} />
      <SummaryCards targets={targets} lastCycle={state.lastCycle} />
      <div style={{ padding: '16px' }}>
        <TargetTable
          targets={targets}
          onDelete={(id) => dispatch({ type: 'DELETE_TARGET', payload: id })}
          onSelect={(id) => dispatch({ type: 'SELECT_TARGET', payload: id })}
          selectedTargetId={state.selectedTargetId}
        />
      </div>
      {toasts && toasts.map((message, index) => (
        <div key={index}>
          <Toast message={message} index={index} onClose={() => setToasts(prev => prev.filter((_, i) => i !== index))} />
        </div>
      ))}
      {state.selectedTargetId && (
        <ResponseChart
          targetId={state.selectedTargetId}
          targetName={state.targets.get(state.selectedTargetId)?.name ?? ''}
          lastCycle={state.lastCycle}
        />
      )}
      {showModal && (
        <AddTargetModal
          onAdd={(target) => dispatch({ type: 'ADD_TARGET', payload: target })}
          onClose={() => setShowModal(false)}
        />
      )}
    </div>
  );
}

export default App;
