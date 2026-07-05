// Setup lobby: name, role (live availability), specialties — one atomic
// submission that also marks the player ready.

import { useState } from 'react';
import { Events } from '../../protocol/events';
import type { PlayerAction, PlayerState } from '../../state/playerReducer';
import type { GameSocket } from '../../ws/client';
import { Shell } from './PlayerApp';

interface Props {
  state: PlayerState;
  socket: GameSocket;
  dispatch: React.Dispatch<PlayerAction>;
}

export function Lobby({ state, socket, dispatch }: Props) {
  const [name, setName] = useState(state.myName);
  const [role, setRole] = useState<string | null>(state.myRole);
  const [specialties, setSpecialties] = useState<string[]>(state.mySpecialties);

  if (state.ready) {
    return (
      <Shell title="You're in!">
        <p>
          Playing as <strong>{state.myName}</strong> ({state.myRole}), specialties:{' '}
          {state.mySpecialties.join(', ')}
        </p>
        {state.lobby && (
          <>
            <p>
              {state.lobby.readyPlayers}/{state.lobby.currentPlayers} players ready
              {state.lobby.hasHost ? '' : ' — no host yet'}
            </p>
            <p className="muted">{state.lobby.waitingMessage}</p>
          </>
        )}
      </Shell>
    );
  }

  const roles = state.rolesAvailable;
  const maxSpecialties = roles?.maxSpecialties ?? 1;

  const toggleSpecialty = (cat: string) => {
    setSpecialties((prev) =>
      prev.includes(cat)
        ? prev.filter((c) => c !== cat)
        : prev.length < maxSpecialties
          ? [...prev, cat]
          : prev,
    );
  };

  const canSubmit = name.trim().length > 0 && role !== null && specialties.length > 0;

  const submit = () => {
    if (!canSubmit || !role) return;
    socket.send(Events.SetupToServerPlayerConfiguration, {
      selectedRole: role,
      selectedSpecialties: specialties,
      playerName: name.trim(),
    });
    dispatch({ type: 'configured', role, specialties, name: name.trim() });
  };

  return (
    <Shell title="Canvas Conundrum">
      <label>
        Your name
        <input value={name} onChange={(e) => setName(e.target.value)} maxLength={32} />
      </label>

      <h2>Choose a role</h2>
      <div className="role-list">
        {roles?.roles.map((r) => (
          <button
            key={r.roleType}
            className={`role-card${role === r.roleType ? ' selected' : ''}`}
            disabled={!r.available}
            onClick={() => setRole(r.roleType)}
          >
            <strong>{r.displayName}</strong>
            <span>{r.description}</span>
            {!r.available && <em>full</em>}
          </button>
        ))}
      </div>

      <h2>
        Trivia specialties <span className="muted">(up to {maxSpecialties})</span>
      </h2>
      <div className="specialty-list">
        {roles?.triviaCategories.map((cat) => (
          <label key={cat} className="specialty-item">
            <input
              type="checkbox"
              checked={specialties.includes(cat)}
              onChange={() => toggleSpecialty(cat)}
            />
            {cat.replace('_', ' ')}
          </label>
        ))}
      </div>

      <button className="primary" disabled={!canSubmit} onClick={submit}>
        Ready up
      </button>
      {state.lobby && <p className="muted">{state.lobby.waitingMessage}</p>}
    </Shell>
  );
}
