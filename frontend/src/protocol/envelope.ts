// Frame envelope (websocket-events.md § Authentication Wrapper Format).

import type { EventName } from './events';

export interface ServerFrame {
  event: EventName;
  payload: unknown;
  timestamp: string;
}

export interface ClientFrame {
  event: EventName;
  auth?: { token: string };
  payload: unknown;
  timestamp: string;
}

// Wire timestamps are ISO 8601 UTC with millisecond precision.
export function wireTimestamp(date: Date = new Date()): string {
  return date.toISOString().replace(/(\.\d{3})\d*Z$/, '$1Z');
}

// buildFrame assembles a client frame; token '' omits auth (only valid for
// a first-connect join).
export function buildFrame(event: EventName, token: string, payload: unknown): ClientFrame {
  const frame: ClientFrame = { event, payload, timestamp: wireTimestamp() };
  if (token !== '') {
    frame.auth = { token };
  }
  return frame;
}

// parseFrame decodes a server frame, returning null for anything malformed
// (a malformed server frame is a bug, but the client must not crash on it).
export function parseFrame(data: string): ServerFrame | null {
  try {
    const parsed: unknown = JSON.parse(data);
    if (
      typeof parsed === 'object' &&
      parsed !== null &&
      typeof (parsed as ServerFrame).event === 'string' &&
      typeof (parsed as ServerFrame).timestamp === 'string'
    ) {
      return parsed as ServerFrame;
    }
  } catch {
    // fall through
  }
  return null;
}
