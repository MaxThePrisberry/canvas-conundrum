// HostApp: the operator pastes the host UUID from the server log (or passes
// it as /host#uuid); the socket then connects to /ws/host/{uuid}.

import { useEffect, useMemo, useReducer, useState } from 'react';
import { ErrorToast } from '../../components/ErrorToast';
import { hostReducer, initialHostState } from '../../state/hostReducer';
import { GameSocket } from '../../ws/client';
import { HostLobby } from './HostLobby';
import { ResourceDashboard } from './ResourceDashboard';
import { PrepScreen } from './PrepScreen';
import { GridMonitor } from './GridMonitor';
import { HostReport } from './HostReport';

const HOST_UUID_KEY = 'canvas-conundrum-host-uuid';

export function HostApp() {
  const [uuid, setUuid] = useState<string>(() => {
    const fromHash = window.location.hash.slice(1);
    return fromHash || sessionStorage.getItem(HOST_UUID_KEY) || '';
  });

  if (!uuid) {
    return <HostLogin onSubmit={setUuid} />;
  }
  sessionStorage.setItem(HOST_UUID_KEY, uuid);
  return <HostSession uuid={uuid} onInvalid={() => setUuid('')} />;
}

function HostLogin({ onSubmit }: { onSubmit: (uuid: string) => void }) {
  const [value, setValue] = useState('');
  return (
    <div className="shell">
      <h1>Host console</h1>
      <p className="muted">
        Paste the host UUID printed in the server log (the path segment of{' '}
        <code>/ws/host/&lt;uuid&gt;</code>).
      </p>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (value.trim()) onSubmit(value.trim());
        }}
      >
        <input value={value} onChange={(e) => setValue(e.target.value)} placeholder="host uuid" />
        <button type="submit" className="primary">
          Connect
        </button>
      </form>
    </div>
  );
}

function HostSession({ uuid, onInvalid }: { uuid: string; onInvalid: () => void }) {
  const [state, dispatch] = useReducer(hostReducer, initialHostState);

  const socket = useMemo(
    () =>
      new GameSocket({
        kind: 'host',
        hostUuid: uuid,
        handlers: {
          onFrame: (frame) => dispatch({ type: 'frame', frame }),
          onStatus: (status, closeCode) => dispatch({ type: 'status', status, closeCode }),
        },
      }),
    [uuid],
  );

  useEffect(() => {
    socket.connect();
    return () => socket.stop();
  }, [socket]);

  if (state.status === 'closed') {
    const unauthorized = state.closeCode === 4001;
    return (
      <div className="shell">
        <h1>{unauthorized ? 'Invalid host UUID' : 'Disconnected'}</h1>
        <p className="muted">
          {unauthorized
            ? 'The server rejected this UUID. Check the current server log — a fresh UUID is generated on every server start.'
            : state.closeCode === 1000
              ? 'This console was superseded by a newer host connection.'
              : 'The connection closed.'}
        </p>
        <button
          onClick={() => {
            sessionStorage.removeItem(HOST_UUID_KEY);
            onInvalid();
          }}
        >
          Enter a different UUID
        </button>
      </div>
    );
  }

  return (
    <div className="app host-app">
      {state.status === 'reconnecting' && <div className="banner">Reconnecting…</div>}
      <HostPhaseScreen state={state} socket={socket} />
      <ErrorToast error={state.lastError} onDismiss={() => dispatch({ type: 'clearError' })} />
    </div>
  );
}

function HostPhaseScreen({
  state,
  socket,
}: {
  state: typeof initialHostState;
  socket: GameSocket;
}) {
  switch (state.phase) {
    case 'setup':
      return <HostLobby state={state} socket={socket} />;
    case 'resource_gathering':
      return <ResourceDashboard state={state} />;
    case 'puzzle_preparation':
      return <PrepScreen state={state} socket={socket} />;
    case 'puzzle_assembly':
      return <GridMonitor state={state} socket={socket} />;
    case 'analytics':
      return <HostReport state={state} socket={socket} />;
  }
}
