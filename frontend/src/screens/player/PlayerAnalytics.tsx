// Post-game personal report and team leaderboard.

import type { PlayerState } from '../../state/playerReducer';
import { Shell } from './PlayerApp';

export function PlayerAnalytics({ state }: { state: PlayerState }) {
  const report = state.personalReport;
  const summary = state.teamSummary;

  return (
    <Shell title={report?.gameSuccess ? 'Victory!' : 'Game over'}>
      {report && (
        <div className="report-card">
          <h2>
            {report.playerName} — #{report.rank} of {report.totalPlayers}
          </h2>
          <p className="score-big">{report.personalScore} points</p>
          <table>
            <tbody>
              <tr>
                <td>Trivia</td>
                <td>{report.scoreBreakdown.triviaPoints}</td>
              </tr>
              <tr>
                <td>Specialty</td>
                <td>{report.scoreBreakdown.specialtyPoints}</td>
              </tr>
              <tr>
                <td>Completion bonus</td>
                <td>{report.scoreBreakdown.completionBonus}</td>
              </tr>
              <tr>
                <td>Moves</td>
                <td>{report.scoreBreakdown.movePoints}</td>
              </tr>
              <tr>
                <td>Recommendations</td>
                <td>{report.scoreBreakdown.recommendationPoints}</td>
              </tr>
            </tbody>
          </table>
          <p className="muted">
            Trivia {report.triviaPerformance.correctAnswers}/{report.triviaPerformance.totalQuestions} ·{' '}
            {report.tokenCollection.totalTokens} tokens ·{' '}
            {report.puzzleSolvingMetrics.successfulMoves} successful moves
          </p>
        </div>
      )}

      {summary && (
        <div className="leaderboard">
          <h2>Team — {summary.totalScore} points</h2>
          <table>
            <thead>
              <tr>
                <th>#</th>
                <th>Player</th>
                <th>Role</th>
                <th>Score</th>
              </tr>
            </thead>
            <tbody>
              {summary.leaderboard.map((entry) => (
                <tr
                  key={entry.playerId}
                  className={entry.playerId === state.playerId ? 'me' : ''}
                >
                  <td>{entry.rank}</td>
                  <td>{entry.playerName}</td>
                  <td>{entry.role.replace('_', ' ')}</td>
                  <td>{entry.totalScore}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="muted">Waiting for the host to start the next game…</p>
    </Shell>
  );
}
