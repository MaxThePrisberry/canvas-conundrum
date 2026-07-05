import type { TeamTokens, ThresholdSet } from '../protocol/events';

interface Props {
  tokens: TeamTokens;
  thresholds: ThresholdSet | null; // achieved threshold counts
}

const TOKEN_TYPES = [
  { key: 'anchorTokens', label: 'Anchor', threshold: 'anchor' },
  { key: 'chronosTokens', label: 'Chronos', threshold: 'chronos' },
  { key: 'guideTokens', label: 'Guide', threshold: 'guide' },
  { key: 'clarityTokens', label: 'Clarity', threshold: 'clarity' },
] as const;

export function TokenBars({ tokens, thresholds }: Props) {
  return (
    <div className="token-bars">
      {TOKEN_TYPES.map(({ key, label, threshold }) => (
        <div key={key} className={`token-bar token-${threshold}`}>
          <span className="token-label">{label}</span>
          <span className="token-value">{tokens[key]}</span>
          {thresholds && <span className="token-thresholds">{'★'.repeat(thresholds[threshold])}</span>}
        </div>
      ))}
    </div>
  );
}
