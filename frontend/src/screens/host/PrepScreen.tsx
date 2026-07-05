import { Events } from '../../protocol/events';
import type { HostState } from '../../state/hostReducer';
import type { GameSocket } from '../../ws/client';

export function PrepScreen({ state, socket }: { state: HostState; socket: GameSocket }) {
  return (
    <div className="shell">
      <h1>Puzzle preparation</h1>
      {state.prepStatus !== 'ready' ? (
        <p>Preparing puzzle… slicing the canvas into segments.</p>
      ) : (
        <>
          <p>Tiles are ready and players are loading their segments.</p>
          {state.phaseLoad && (
            <p className="muted">
              {state.phaseLoad.centralGridSize}×{state.phaseLoad.centralGridSize} grid ·{' '}
              {state.phaseLoad.playerCount} players · pre-solved{' '}
              {state.phaseLoad.bonusEffects.anchorPreSolved} pieces · +
              {state.phaseLoad.bonusEffects.chronosTimeBonus}s chronos
            </p>
          )}
          <p>Start the timer when everyone has gathered in the puzzle room.</p>
          <button className="primary" onClick={() => socket.send(Events.PuzzleToServerPhaseStart, {})}>
            Start puzzle timer
          </button>
        </>
      )}
    </div>
  );
}
