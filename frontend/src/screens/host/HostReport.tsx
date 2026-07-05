import { Events } from '../../protocol/events';
import type { HostState } from '../../state/hostReducer';
import type { GameSocket } from '../../ws/client';

export function HostReport({ state, socket }: { state: HostState; socket: GameSocket }) {
  const report = state.completeReport;
  const completion = state.completionAnalytics;

  return (
    <div className="shell wide">
      <h1>{report?.gameSuccess ? 'Victory! 🎉' : 'Time expired'}</h1>

      {report && (
        <>
          <div className="stat-row">
            <span>{report.totalPlayers} players</span>
            <span>{Math.round(report.totalGameTime)}s total</span>
            <span>{report.overallPerformance.totalScore} team points</span>
            <span>avg {report.overallPerformance.averageScore.toFixed(1)}</span>
          </div>

          <h2>Resource gathering</h2>
          <p className="muted">
            {report.resourceGatheringAnalytics.questionsAnswered} questions ·{' '}
            {(report.resourceGatheringAnalytics.overallAccuracy * 100).toFixed(0)}% accuracy
          </p>

          <h2>Puzzle assembly</h2>
          <p className="muted">
            {report.puzzleAssemblyAnalytics.completionTime.toFixed(0)}s of{' '}
            {report.puzzleAssemblyAnalytics.totalTime.toFixed(0)}s (
            {(report.puzzleAssemblyAnalytics.timeUtilization * 100).toFixed(0)}% used) ·{' '}
            {report.puzzleAssemblyAnalytics.collaborativePhaseMetrics.totalMoves} moves ·{' '}
            {report.puzzleAssemblyAnalytics.collaborativePhaseMetrics.totalRecommendations}{' '}
            recommendations
          </p>

          <h2>Categories</h2>
          <table>
            <thead>
              <tr>
                <th>Category</th>
                <th>Asked</th>
                <th>Correct</th>
                <th>Accuracy</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(report.categoryPerformance).map(([cat, c]) => (
                <tr key={cat}>
                  <td>{cat.replace('_', ' ')}</td>
                  <td>{c.questionsAsked}</td>
                  <td>{c.correctAnswers}</td>
                  <td>{(c.accuracy * 100).toFixed(0)}%</td>
                </tr>
              ))}
            </tbody>
          </table>

          <h2>Timeline</h2>
          <div className="stat-row">
            <span>setup {Math.round(report.timelineAnalysis.setupPhase)}s</span>
            <span>gathering {Math.round(report.timelineAnalysis.resourcePhase)}s</span>
            <span>prep {Math.round(report.timelineAnalysis.preparationPhase)}s</span>
            <span>puzzle {Math.round(report.timelineAnalysis.puzzlePhase)}s</span>
          </div>
        </>
      )}

      {!report && completion && <p className="muted">Compiling the final report…</p>}

      <button
        className="primary danger"
        onClick={() => {
          if (window.confirm('Reset the game for a new group? All players must rejoin.')) {
            socket.send(Events.AnalyticsToServerResetGame, { confirmReset: true });
          }
        }}
      >
        Reset for a new game
      </button>
    </div>
  );
}
