// Countdown re-anchored by server updates: parent passes the latest
// authoritative secondsRemaining plus when it arrived; the component ticks
// locally between updates. paused freezes the display.

import { useEffect, useState } from 'react';
import { formatSeconds } from '../lib/formulas';

interface Props {
  secondsRemaining: number;
  anchoredAt: number; // Date.now() when secondsRemaining was received
  paused?: boolean;
}

export function Countdown({ secondsRemaining, anchoredAt, paused }: Props) {
  const [, tick] = useState(0);

  useEffect(() => {
    if (paused) return;
    const timer = setInterval(() => tick((n) => n + 1), 250);
    return () => clearInterval(timer);
  }, [paused]);

  const elapsed = paused ? 0 : (Date.now() - anchoredAt) / 1000;
  const remaining = secondsRemaining - elapsed;

  return (
    <span className={`countdown${remaining < 30 ? ' countdown-low' : ''}`}>
      {formatSeconds(remaining)}
    </span>
  );
}
