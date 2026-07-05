// Resource-gathering round: station scanning, the trivia question with its
// deadline, the marking result, and team progress.

import { useRef } from 'react';
import { Countdown } from '../../components/Countdown';
import { QrScanner } from '../../components/QrScanner';
import { TokenBars } from '../../components/TokenBars';
import { wireTimestamp } from '../../protocol/envelope';
import { Events } from '../../protocol/events';
import type { PlayerAction, PlayerState } from '../../state/playerReducer';
import type { GameSocket } from '../../ws/client';
import { Shell } from './PlayerApp';

interface Props {
  state: PlayerState;
  socket: GameSocket;
  dispatch: React.Dispatch<PlayerAction>;
}

export function ResourceRound({ state, socket, dispatch }: Props) {
  const questionReceivedAt = useRef<number>(Date.now());
  const lastQuestionId = useRef<string>('');

  if (state.question && state.question.questionId !== lastQuestionId.current) {
    lastQuestionId.current = state.question.questionId;
    questionReceivedAt.current = Date.now();
  }

  const scan = (hash: string) => {
    socket.send(Events.ResourceToServerLocationVerified, {
      stationHash: hash,
      previousLocation: state.location === 'unknown' ? undefined : state.location,
      scanTimestamp: wireTimestamp(),
    });
  };

  const answer = (index: number) => {
    if (!state.question) return;
    socket.send(Events.ResourceToServerTriviaAnswer, {
      questionId: state.question.questionId,
      answerIndex: index,
      timeElapsed: (Date.now() - questionReceivedAt.current) / 1000,
    });
    dispatch({ type: 'answered', index });
  };

  const q = state.question;
  const deadlineRemaining = q ? (new Date(q.answerDeadline).getTime() - Date.now()) / 1000 : 0;

  return (
    <Shell title="Resource gathering">
      <p className="station-line">
        Station: <strong className={`station-${state.location}`}>{state.location}</strong>
        {state.location === 'unknown' && (
          <span className="muted"> — scan a station to start earning tokens!</span>
        )}
      </p>

      {q ? (
        <div className="question-card">
          <header>
            <span>
              Round {q.roundNumber}/{q.totalRounds}
              {q.isSpecialty && <em className="specialty-tag"> ★ specialty</em>}
            </span>
            <Countdown secondsRemaining={deadlineRemaining} anchoredAt={Date.now()} />
          </header>
          <p className="question-text">{q.questionText}</p>
          <div className="options">
            {q.options.map((option, i) => (
              <button
                key={i}
                className={state.answeredIndex === i ? 'selected' : ''}
                onClick={() => answer(i)}
              >
                {option}
              </button>
            ))}
          </div>
          {state.answeredIndex !== null && (
            <p className="muted">Answer locked in — you can change it until the deadline.</p>
          )}
        </div>
      ) : state.answerResult ? (
        <div className={`result-card ${state.answerResult.correct ? 'correct' : 'incorrect'}`}>
          <h2>{state.answerResult.correct ? 'Correct!' : 'Not quite.'}</h2>
          <p>
            Answer: <strong>{state.answerResult.correctAnswer}</strong>
          </p>
          <p>
            +{state.answerResult.tokensEarned} tokens
            {state.answerResult.bonuses.roleBonus && ' (role bonus!)'}
            {state.answerResult.bonuses.specialtyBonus && ' (specialty!)'}
          </p>
          <p className="muted">Move to another station or stay put — next question soon.</p>
        </div>
      ) : (
        <p className="muted">Waiting for the first question — get to a station!</p>
      )}

      <QrScanner onScan={scan} />

      {state.teamProgress && (
        <div className="team-progress">
          <h2>
            Team progress — round {state.teamProgress.currentRound}/{state.teamProgress.totalRounds}
          </h2>
          <TokenBars
            tokens={state.teamProgress.teamTokens}
            thresholds={state.teamProgress.currentThresholds}
          />
          <p className="muted">
            Team accuracy {(state.teamProgress.teamPerformance.averageAccuracy * 100).toFixed(0)}%
          </p>
        </div>
      )}
    </Shell>
  );
}
