// PlayerApp wires the GameSocket to the player reducer and picks the screen
// for the current phase.

import { useEffect, useMemo, useReducer, useRef } from 'react';
import { ErrorToast } from '../../components/ErrorToast';
import { Events, type PlayerConnectionConfirmed } from '../../protocol/events';
import {
  initialPlayerState,
  playerReducer,
  type PlayerAction,
  type PlayerState,
} from '../../state/playerReducer';
import { clearPlayerToken, GameSocket } from '../../ws/client';
import { Lobby } from './Lobby';
import { ResourceRound } from './ResourceRound';
import { IndividualPuzzle } from './IndividualPuzzle';
import { CentralGrid } from './CentralGrid';
import { PlayerAnalytics } from './PlayerAnalytics';

export function PlayerApp() {
  const [state, dispatch] = useReducer(playerReducer, initialPlayerState);
  const socketRef = useRef<GameSocket | null>(null);

  const socket = useMemo(() => {
    const s = new GameSocket({
      kind: 'player',
      handlers: {
        onFrame: (frame) => {
          if (frame.event === Events.SetupToPlayerConnectionConfirmed) {
            s.adoptPlayerToken((frame.payload as PlayerConnectionConfirmed).playerId);
          }
          dispatch({ type: 'frame', frame });
        },
        onStatus: (status, closeCode) => dispatch({ type: 'status', status, closeCode }),
      },
    });
    return s;
  }, []);

  useEffect(() => {
    socketRef.current = socket;
    socket.connect();
    return () => socket.stop();
  }, [socket]);

  if (state.gameReset) {
    return (
      <Shell title="Game over">
        <p>{state.gameReset.reconnectInstructions}</p>
        <button
          onClick={() => {
            clearPlayerToken();
            window.location.reload();
          }}
        >
          Join the next game
        </button>
      </Shell>
    );
  }

  if (state.status === 'closed') {
    return (
      <Shell title="Connection closed">
        <p>{closeExplanation(state.closeCode)}</p>
        <button
          onClick={() => {
            clearPlayerToken();
            window.location.reload();
          }}
        >
          Start over
        </button>
      </Shell>
    );
  }

  return (
    <div className="app player-app">
      {state.status === 'reconnecting' && <div className="banner">Reconnecting…</div>}
      {state.paused && <div className="banner banner-pause">Game paused — waiting for the host</div>}
      <PhaseScreen state={state} socket={socket} dispatch={dispatch} />
      <ErrorToast error={state.lastError} onDismiss={() => dispatch({ type: 'clearError' })} />
    </div>
  );
}

function PhaseScreen({
  state,
  socket,
  dispatch,
}: {
  state: PlayerState;
  socket: GameSocket;
  dispatch: React.Dispatch<PlayerAction>;
}) {
  switch (state.phase) {
    case 'setup':
      return <Lobby state={state} socket={socket} dispatch={dispatch} />;
    case 'resource_gathering':
      return <ResourceRound state={state} socket={socket} dispatch={dispatch} />;
    case 'puzzle_preparation':
      return (
        <Shell title="Puzzle incoming">
          <p>Head to the puzzle room! The host starts the timer when everyone is ready.</p>
          {state.phaseLoad && <p className="muted">Your segment: {state.phaseLoad.assignedSegmentId}</p>}
        </Shell>
      );
    case 'puzzle_assembly':
      if (state.completion) {
        return (
          <Shell title={state.completion.success ? 'Puzzle solved! 🎉' : 'Time expired'}>
            <p>
              {state.completion.success
                ? 'The canvas is complete. Crunching the numbers…'
                : 'The picture stayed unfinished. Crunching the numbers…'}
            </p>
          </Shell>
        );
      }
      if (state.segmentAck) {
        return <CentralGrid state={state} socket={socket} dispatch={dispatch} />;
      }
      return <IndividualPuzzle state={state} socket={socket} />;
    case 'analytics':
      return <PlayerAnalytics state={state} />;
  }
}

export function Shell({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="shell">
      <h1>{title}</h1>
      {children}
    </div>
  );
}

function closeExplanation(code?: number): string {
  switch (code) {
    case 4001:
      return 'This session is no longer valid.';
    case 4002:
      return 'The game is full or already underway — new players cannot join right now.';
    case 4003:
      return 'The puzzle is being assembled; rejoining mid-assembly is not possible.';
    default:
      return 'The connection was closed.';
  }
}
