// GameSocket: one resilient WebSocket session for a player or the host.
//
// - Player sockets open /ws and send SETUP_TO_SERVER_PLAYER_CONNECT as the
//   first frame (token attached when reconnecting).
// - Host sockets open /ws/host/{uuid}; the URL carries the token.
// - SYSTEM_PING every 30s once the handshake completes.
// - Reconnect with exponential backoff (1s → 30s); closes with 1000 or any
//   4xxx code are terminal and reported instead.

import { buildFrame, parseFrame, wireTimestamp, type ServerFrame } from '../protocol/envelope';
import { Events, type EventName } from '../protocol/events';
import { nextBackoff, shouldReconnect } from '../lib/backoff';

const PING_INTERVAL_MS = 30000;
const TOKEN_STORAGE_KEY = 'canvas-conundrum-token';

export type SocketStatus = 'connecting' | 'open' | 'reconnecting' | 'closed';

export interface GameSocketHandlers {
  onFrame: (frame: ServerFrame) => void;
  onStatus: (status: SocketStatus, closeCode?: number) => void;
}

interface GameSocketOptions {
  kind: 'player' | 'host';
  hostUuid?: string; // required for kind: 'host'
  handlers: GameSocketHandlers;
}

export class GameSocket {
  private ws: WebSocket | null = null;
  private readonly kind: 'player' | 'host';
  private readonly hostUuid: string;
  private readonly handlers: GameSocketHandlers;
  private token: string;
  private attempt = 0;
  private pingTimer: number | null = null;
  private pingSeq = 0;
  private reconnectTimer: number | null = null;
  private stopped = false;

  constructor(opts: GameSocketOptions) {
    this.kind = opts.kind;
    this.hostUuid = opts.hostUuid ?? '';
    this.handlers = opts.handlers;
    this.token = this.kind === 'host' ? this.hostUuid : loadPlayerToken();
  }

  connect(): void {
    this.stopped = false;
    this.open();
  }

  // stop tears the socket down without reconnecting (component unmount).
  stop(): void {
    this.stopped = true;
    this.clearTimers();
    this.ws?.close(1000);
    this.ws = null;
  }

  // send wraps a payload in the auth envelope and ships it.
  send(event: EventName, payload: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(buildFrame(event, this.token, payload)));
    }
  }

  // adoptPlayerToken records the server-issued identity after the player
  // handshake so later frames, pings, and reconnections carry it.
  adoptPlayerToken(token: string): void {
    this.token = token;
    sessionStorage.setItem(TOKEN_STORAGE_KEY, token);
  }

  get authToken(): string {
    return this.token;
  }

  private url(): string {
    const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
    const path = this.kind === 'host' ? `/ws/host/${this.hostUuid}` : '/ws';
    return `${scheme}://${location.host}${path}`;
  }

  private open(): void {
    this.handlers.onStatus(this.attempt === 0 ? 'connecting' : 'reconnecting');
    const ws = new WebSocket(this.url());
    this.ws = ws;

    ws.onopen = () => {
      this.attempt = 0;
      if (this.kind === 'player') {
        // First frame: connect (token = reconnect, none = new join).
        ws.send(JSON.stringify(buildFrame(Events.SetupToServerPlayerConnect, this.token, {})));
      }
      this.startPinger();
      this.handlers.onStatus('open');
    };

    ws.onmessage = (msg) => {
      const frame = parseFrame(String(msg.data));
      if (frame) {
        this.handlers.onFrame(frame);
      }
    };

    ws.onclose = (ev) => {
      this.clearTimers();
      if (this.ws !== ws) return; // superseded by a newer socket
      this.ws = null;
      if (this.stopped) return;

      if (!shouldReconnect(ev.code)) {
        this.handlers.onStatus('closed', ev.code);
        return;
      }
      const delay = nextBackoff(this.attempt++);
      this.handlers.onStatus('reconnecting', ev.code);
      this.reconnectTimer = window.setTimeout(() => this.open(), delay);
    };
  }

  private startPinger(): void {
    this.stopPinger();
    this.pingTimer = window.setInterval(() => {
      this.pingSeq++;
      this.send(Events.SystemPing, {
        clientTimestamp: wireTimestamp(),
        sequenceNumber: this.pingSeq,
      });
    }, PING_INTERVAL_MS);
  }

  private stopPinger(): void {
    if (this.pingTimer !== null) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
  }

  private clearTimers(): void {
    this.stopPinger();
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}

export function loadPlayerToken(): string {
  return sessionStorage.getItem(TOKEN_STORAGE_KEY) ?? '';
}

export function clearPlayerToken(): void {
  sessionStorage.removeItem(TOKEN_STORAGE_KEY);
}
