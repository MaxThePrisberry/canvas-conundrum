import { Events } from '../../protocol/events';
import type { HostState } from '../../state/hostReducer';
import type { GameSocket } from '../../ws/client';

export function HostLobby({ state, socket }: { state: HostState; socket: GameSocket }) {
  const roster = state.roster;
  return (
    <div className="shell wide">
      <h1>Lobby</h1>
      {state.confirmed && (
        <p className="muted">
          {state.confirmed.gameConfig.difficultyMode} difficulty ·{' '}
          {state.confirmed.gameConfig.resourceGatheringRounds} rounds · needs{' '}
          {state.confirmed.gameConfig.minPlayers}+ players
        </p>
      )}

      {roster ? (
        <>
          <p>
            {roster.readyPlayers}/{roster.connectedPlayers} ready
          </p>
          <table>
            <thead>
              <tr>
                <th>Player</th>
                <th>Role</th>
                <th>Specialties</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(roster.playerStatuses).map(([id, p]) => (
                <tr key={id} className={p.connected ? '' : 'disconnected'}>
                  <td>{p.playerName || '—'}</td>
                  <td>{p.role ?? '—'}</td>
                  <td>{p.specialties.join(', ') || '—'}</td>
                  <td>{!p.connected ? 'offline' : p.ready ? 'ready' : 'configuring'}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <button
            className="primary"
            disabled={!roster.gameStartEligible}
            onClick={() => socket.send(Events.SetupToServerStartGame, {})}
          >
            Start game
          </button>
        </>
      ) : (
        <p className="muted">Waiting for players…</p>
      )}
    </div>
  );
}
