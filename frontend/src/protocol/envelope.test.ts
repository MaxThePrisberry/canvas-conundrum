import { describe, expect, it } from 'vitest';
import { buildFrame, parseFrame, wireTimestamp } from './envelope';
import { Events } from './events';

describe('envelope', () => {
  it('builds an authenticated frame', () => {
    const frame = buildFrame(Events.SystemPing, 'tok-1', { sequenceNumber: 1 });
    expect(frame.auth).toEqual({ token: 'tok-1' });
    expect(frame.event).toBe('SYSTEM_PING');
    expect(frame.timestamp).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/);
  });

  it('omits auth for a token-less join', () => {
    const frame = buildFrame(Events.SetupToServerPlayerConnect, '', {});
    expect(frame.auth).toBeUndefined();
  });

  it('parses valid server frames and rejects junk', () => {
    const ok = parseFrame(
      JSON.stringify({ event: 'SYSTEM_PONG', payload: {}, timestamp: wireTimestamp() }),
    );
    expect(ok?.event).toBe('SYSTEM_PONG');

    expect(parseFrame('{not json')).toBeNull();
    expect(parseFrame(JSON.stringify({ payload: {} }))).toBeNull();
  });
});
