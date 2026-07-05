import { useEffect, useRef } from 'react';
import { Countdown } from '../../components/Countdown';
import { GridView } from '../../components/GridView';
import type { HostState } from '../../state/hostReducer';
import type { GameSocket } from '../../ws/client';

export function GridMonitor({ state, socket }: { state: HostState; socket: GameSocket }) {
  const grid = state.gridState;
  const anchoredAt = useRef(Date.now());
  useEffect(() => {
    anchoredAt.current = Date.now();
  }, [grid]);

  const gridSize = state.phaseLoad?.centralGridSize ?? 3;
  const fragments = (grid?.fragments ?? []).map((f) => ({
    segmentId: f.segmentId,
    playerId: f.playerId,
    playerName: f.playerName,
    position: f.position,
  }));

  return (
    <div className="shell wide">
      <div className="grid-header">
        <h1>Puzzle assembly</h1>
        {grid && <Countdown secondsRemaining={grid.timeRemaining} anchoredAt={anchoredAt.current} />}
      </div>

      {state.puzzleStart && (
        <p className="muted">
          {state.puzzleStart.totalTime}s total ({state.puzzleStart.baseTime}s +{' '}
          {state.puzzleStart.chronosBonus}s chronos) · 2A: {state.puzzleStart.playersInPhase2a} · 2B:{' '}
          {state.puzzleStart.playersInPhase2b}
          {grid && ` · ${grid.activeRecommendations} active recommendations`}
        </p>
      )}

      <GridView gridSize={gridSize} fragments={fragments} token={socket.authToken} />

      {grid && (
        <table>
          <thead>
            <tr>
              <th>Phase</th>
              <th>Moves</th>
              <th>Successful</th>
            </tr>
          </thead>
          <tbody>
            {Object.entries(grid.playerMetrics).map(([id, m]) => (
              <tr key={id}>
                <td>{m.phase}</td>
                <td>{m.movesContributed}</td>
                <td>{m.successfulMoves}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {state.lastCompletion && (
        <p className="muted">
          {state.lastCompletion.playerName} finished their segment (
          {state.lastCompletion.completionStats.totalCompleted}/
          {state.lastCompletion.completionStats.totalRequired})
        </p>
      )}
    </div>
  );
}
