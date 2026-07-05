import { TokenBars } from '../../components/TokenBars';
import type { HostState } from '../../state/hostReducer';

export function ResourceDashboard({ state }: { state: HostState }) {
  const dashboard = state.resourceStart?.monitoringDashboard;
  const round = state.roundAnalytics;

  return (
    <div className="shell wide">
      <h1>Resource gathering</h1>
      {dashboard && (
        <p className="muted">
          Round {round?.currentRound ?? dashboard.currentRound}/{dashboard.totalRounds} ·{' '}
          {dashboard.roundDuration}s per round
        </p>
      )}

      {round ? (
        <>
          <TokenBars tokens={round.teamTokens} thresholds={null} />
          <div className="stat-row">
            <span>
              {round.roundResults.correctAnswers}/{round.roundResults.questionsDelivered} correct
            </span>
            <span>{round.roundResults.tokensAwarded} tokens this round</span>
            <span>avg {round.roundResults.averageResponseTime.toFixed(1)}s</span>
          </div>
          <h2>Players</h2>
          <table>
            <thead>
              <tr>
                <th>Station</th>
                <th>Last answer</th>
                <th>Tokens</th>
                <th>Accuracy</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(round.playerPerformance).map(([id, p]) => (
                <tr key={id}>
                  <td>{p.location}</td>
                  <td>{p.answerCorrect ? '✓' : '✗'}</td>
                  <td>{p.tokensEarned}</td>
                  <td>{(p.runningAccuracy * 100).toFixed(0)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
          <h2>Stations</h2>
          <div className="stat-row">
            {Object.entries(round.stationDistribution).map(([station, n]) => (
              <span key={station}>
                {station}: {n}
              </span>
            ))}
          </div>
        </>
      ) : (
        <p className="muted">Round 1 begins after the players reach their first stations.</p>
      )}
    </div>
  );
}
