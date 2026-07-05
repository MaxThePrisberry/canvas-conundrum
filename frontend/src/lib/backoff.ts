// Reconnection backoff policy (websocket-events.md § Client reconnection
// backoff): exponential from 1s, doubling, capped at 30s, reset on success.
// Closes with 1000 or any 4xxx code are terminal — never auto-reconnect.

export const INITIAL_BACKOFF_MS = 1000;
export const MAX_BACKOFF_MS = 30000;

// shouldReconnect decides whether a close code permits an automatic retry.
// 1000 = deliberate (reset / supersede); 4xxx = application rejections,
// terminal per RFC 6455 §7.4.2 guidance in the spec.
export function shouldReconnect(closeCode: number): boolean {
  if (closeCode === 1000) return false;
  if (closeCode >= 4000 && closeCode < 5000) return false;
  return true;
}

// nextBackoff returns the delay for the given retry attempt (0-based).
export function nextBackoff(attempt: number): number {
  const delay = INITIAL_BACKOFF_MS * 2 ** attempt;
  return Math.min(delay, MAX_BACKOFF_MS);
}
