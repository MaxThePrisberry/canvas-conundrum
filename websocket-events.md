# Canvas Conundrum - Complete WebSocket Event Specifications

## Authentication Wrapper Format

All client-to-server messages use this format. The `auth` field is optional only on `SETUP_TO_SERVER_PLAYER_CONNECT`, where its absence means "new player joining" (see that event); everywhere else it is required.

```json
{
  "event": "EVENT_NAME",
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    // Event-specific data shown below
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

All server-to-client messages use this format:

```json
{
  "event": "EVENT_NAME",
  "payload": {
    // Event-specific data shown below
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

---

## Conventions

### Required vs optional fields

Every payload field shown in this spec is **required** unless explicitly marked **(optional)** in prose following the JSON example. Optional fields may be omitted entirely or sent as JSON `null`; their absence has the meaning described per-event. A missing required field returns `SYSTEM_TO_CLIENT_ERROR` with code `MALFORMED_PAYLOAD`.

### Message limits

**Client-to-server** WebSocket messages are limited to **8 KB**; violations are rejected with `SYSTEM_TO_CLIENT_ERROR` / `MALFORMED_PAYLOAD`. Server-to-client messages have no fixed cap — host rosters, grid states, and analytics reports legitimately exceed 8 KB at high player counts.

All text fields must be valid UTF-8. Free-text fields have these length limits (everything else is fixed-format — UUIDs, enums, ISO timestamps, numbers):

| Field | Limit |
|---|---|
| `playerName` | 1–32 characters |
| `reasoning`, `responseReason` | ≤ 200 characters |
| `stationHash` | ≤ 128 characters |

### Timestamps

All timestamp fields are ISO 8601 UTC strings — the envelope `timestamp` and payload fields like `startTimestamp`, `completionTimestamp`, `nextMoveAvailable`, and `timerPausedAt` alike. Durations and countdowns are plain numeric seconds (milliseconds only where a config value says so, e.g. `fragmentMoveCooldown`).

### Canonical example game

All example payloads in this document describe the same game: **5 players** — Alice (detective, science), Bob (art_enthusiast, music), Charlie (tourist, geography), Diana (janitor, history), Eve (art_enthusiast, general) — on **medium** difficulty with **5 rounds**, producing a **3×3 grid** (segments `a1`–`c3`; Alice→`a1`, Bob→`a2`, Charlie→`a3`, Diana→`b1`, Eve→`b2`). Final team tokens: anchor 90, chronos 70, guide 50, clarity 70 (280 total) — hence final thresholds anchor 3, chronos 3, guide 3, clarity 2, giving 6 pre-solved pieces, +60s chronos bonus (360s total), 5 guide highlights, and a 5s clarity preview. Example values are internally consistent with this game and the formulas in `game-design.md`; setup-phase examples are snapshots taken before all five players finished configuring.

### Game phases

`currentPhase` / `phase` fields take exactly these values: `setup`, `resource_gathering`, `puzzle_preparation`, `puzzle_assembly`, `analytics`. `puzzle_preparation` is a first-class phase: it begins when the final resource-gathering round completes and ends when the host sends `PUZZLE_TO_SERVER_PHASE_START` — it covers both tile generation and the wait for the host to start the timer. `puzzle_assembly` begins at that start signal.

### WebSocket close codes

When the server closes a WebSocket connection it uses these codes:

| Code | Meaning |
|---|---|
| `1000` | Normal closure (e.g. game reset, graceful client disconnect) |
| `1001` | Server going away (planned shutdown) |
| `4001` | Unauthorized (malformed/unknown token in the host URL or player connect frame, or no `SETUP_TO_SERVER_PLAYER_CONNECT` frame within 10 seconds of the upgrade) |
| `4002` | Join rejected (new player while the game is past setup or already at `maxPlayers`) |
| `4003` | Forbidden (player tried to connect — new or reconnecting — during puzzle assembly phase) |
| `4004` | Token invalid or expired (e.g. game reset since the token was issued) |

Codes in the `4xxx` range are application-specific per RFC 6455 §7.4.2; clients should treat any unrecognized `4xxx` close as a terminal failure.

Handshake-time rejections (`4001`–`4004`) are delivered by completing the WebSocket upgrade and closing with the code, without sending any application frames. For hosts the token is judged from the URL immediately; for players it is judged from the first frame (`SETUP_TO_SERVER_PLAYER_CONNECT`). Rejecting at the HTTP layer instead would leave browser clients — which cannot read the HTTP status of a failed upgrade — unable to distinguish a permanent rejection from a transient network failure (both surface as a `1006` close).

### Client reconnection backoff

After an unexpected close (any code other than `1000`), clients should attempt to reconnect using exponential backoff: start at 1 second, double each retry, cap at 30 seconds. Reset to 1 second on a successful connection. Connection attempts that close immediately with code `4001`, `4002`, `4003`, or `4004` should not be retried — the failure is permanent for this game.

### Error code registry

Error codes form a single project-wide namespace shared by:
- HTTP error responses (`error` field in the JSON body — see *Asset Delivery (HTTP)*)
- `SYSTEM_TO_CLIENT_ERROR` and `SYSTEM_TO_HOST_ERROR` events (`errorCode` field)

Codes are stable identifiers; new codes may be added over time, but existing codes will never be repurposed.

| Code | Used in | Meaning |
|---|---|---|
| `UNAUTHORIZED` | HTTP, WS | Missing/malformed/unknown bearer token (HTTP) or unauthenticated WS action |
| `MALFORMED_REQUEST` | HTTP | Path, header, or body unparsable |
| `MALFORMED_PAYLOAD` | WS | Required WebSocket field missing or wrong type |
| `INVALID_ROLE_SELECTION` | WS | Selected role is unknown |
| `INVALID_SPECIALTY_SELECTION` | WS | Specialty list is empty, has duplicates, exceeds `maxSpecialtiesPerPlayer`, or names an unknown category |
| `ROLE_FULL` | WS | All slots for the requested role are taken |
| `INVALID_STATION_HASH` | WS | QR-code hash did not match any configured station |
| `INSUFFICIENT_PLAYERS` | WS | Host tried to start the game before every connected player was ready and at least `minPlayers` were ready |
| `COOLDOWN_ACTIVE` | WS | Recommendation created or accepted while an involved fragment is on its move cooldown |
| `RECOMMENDATION_PENDING` | WS | Sender already has an outgoing recommendation pending |
| `FORBIDDEN_PHASE` | HTTP, WS | Action not permitted in the current game phase or phase state (e.g. puzzle start before tile generation completes) |
| `FORBIDDEN_NOT_OWNER` | HTTP, WS | Caller is not the assigned owner of the resource |
| `FORBIDDEN_PREVIEW_WINDOW_CLOSED` | HTTP | Full-image preview requested outside its active time window |
| `NOT_FOUND` | HTTP | Resource does not currently exist (e.g. tiles before generation, or after game reset) |

HTTP responses map status codes as follows: `400` for `MALFORMED_REQUEST`; `401` for `UNAUTHORIZED`; `403` for any `FORBIDDEN_*` code; `404` for `NOT_FOUND`.

---

## Event Index

Every event in this spec, in protocol order. Directions: H = host, P = player, S = server; "all P" = broadcast to all players; "all" = every connected client (all players **and** the host).

| Event | Direction | When |
|---|---|---|
| `SETUP_TO_HOST_CONNECTION_CONFIRMED` | S → H | Host connects or reconnects (handshake) |
| `SETUP_TO_SERVER_PLAYER_CONNECT` | P → S | First frame after opening `/ws` (token = reconnect; no token = new join) |
| `SETUP_TO_PLAYER_CONNECTION_CONFIRMED` | S → P | Player connects or reconnects (handshake) |
| `SETUP_TO_PLAYER_ROLES_AVAILABLE` | S → P | After player handshake in setup; on any role availability change (unready players only) |
| `SETUP_TO_SERVER_PLAYER_CONFIGURATION` | P → S | Player submits role, specialty, and name |
| `SETUP_TO_CLIENT_LOBBY_STATUS` | S → all P | Any player joins/leaves or changes configuration |
| `SETUP_TO_HOST_PLAYER_ROSTER` | S → H | Any change to player status |
| `SETUP_TO_SERVER_START_GAME` | H → S | Host starts the game |
| `SETUP_TO_HOST_GAME_STARTED` | S → H | Game successfully started |
| `RESOURCE_TO_CLIENT_PHASE_START` | S → all P | Resource gathering begins |
| `RESOURCE_TO_HOST_PHASE_START` | S → H | Resource gathering begins |
| `RESOURCE_TO_SERVER_LOCATION_VERIFIED` | P → S | Player scans a station QR code |
| `RESOURCE_TO_PLAYER_LOCATION_CONFIRMED` | S → P | Server applies a station change |
| `RESOURCE_TO_PLAYER_TRIVIA_QUESTION` | S → P | Start of each gathering round |
| `RESOURCE_TO_SERVER_TRIVIA_ANSWER` | P → S | Player selects an answer |
| `RESOURCE_TO_PLAYER_ANSWER_RESULT` | S → P | Answer window ends |
| `RESOURCE_TO_CLIENT_TEAM_PROGRESS` | S → all P | End of each round's answer window, after marking and token awards |
| `RESOURCE_TO_HOST_ROUND_ANALYTICS` | S → H | After each round |
| `RESOURCE_TO_CLIENT_PHASE_COMPLETE` | S → all P | All gathering rounds completed |
| `RESOURCE_TO_HOST_PHASE_COMPLETE` | S → H | All gathering rounds completed |
| `PUZZLE_TO_HOST_PREPARING` | S → H | Tile generation begins (`puzzle_preparation` starts) |
| `PUZZLE_TO_HOST_READY` | S → H | Tile generation complete; host may start the timer |
| `PUZZLE_TO_CLIENT_PHASE_LOAD` | S → all P | Tiles ready; players fetch segments, UI stays hidden |
| `PUZZLE_TO_HOST_PHASE_LOAD` | S → H | Tiles ready (immediately after `PUZZLE_TO_HOST_READY`) |
| `PUZZLE_TO_SERVER_PHASE_START` | H → S | Host starts the puzzle timer |
| `PUZZLE_TO_CLIENT_PHASE_START` | S → all P | Puzzle timer started (`puzzle_assembly` begins) |
| `PUZZLE_TO_HOST_PHASE_START` | S → H | Puzzle timer started |
| `PUZZLE_TO_CLIENT_PREVIEW_EXPIRED` | S → all P | Clarity preview window elapses (only if a window opened) |
| `PUZZLE_TO_SERVER_SEGMENT_COMPLETED` | P → S | Player completes their individual puzzle |
| `PUZZLE_TO_PLAYER_SEGMENT_ACKNOWLEDGED` | S → P | Server accepts a segment completion |
| `PUZZLE_TO_HOST_SEGMENT_COMPLETED` | S → H | A player completes their segment |
| `PUZZLE_TO_SERVER_FRAGMENT_MOVE` | P → S | Player moves or swaps a fragment |
| `PUZZLE_TO_PLAYER_MOVE_RESULT` | S → P | Server processes a move request |
| `PUZZLE_TO_CLIENT_GRID_STATE` | S → all P | Every `gridUpdateInterval` seconds |
| `PUZZLE_TO_HOST_GRID_STATE` | S → H | Immediately on any grid change |
| `PUZZLE_TO_PLAYER_PERSONAL_STATE` | S → P | Guide highlights, per grid tick (Phase 2B players only) |
| `PUZZLE_TO_SERVER_RECOMMEND_MOVE` | P → S | Player proposes a swap to a fragment owner |
| `PUZZLE_TO_PLAYER_MOVE_RECOMMENDATION` | S → P | Recommendation delivered to the target player |
| `PUZZLE_TO_SERVER_RECOMMENDATION_RESPONSE` | P → S | Target accepts or rejects a recommendation |
| `PUZZLE_TO_PLAYER_RECOMMENDATION_RESULT` | S → P | Outcome delivered to the recommender |
| `PUZZLE_TO_PLAYER_RECOMMENDATION_EXPIRED` | S → P (both parties) | Recommendation times out or a party disconnects |
| `PUZZLE_TO_CLIENT_COMPLETED_SUCCESS` | S → all | Both victory conditions met |
| `PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT` | S → all | Timer expires before completion |
| `PUZZLE_TO_HOST_COMPLETION_ANALYTICS` | S → H | Puzzle phase ends (either outcome) |
| `ANALYTICS_TO_PLAYER_PERSONAL_REPORT` | S → P | Game completion |
| `ANALYTICS_TO_CLIENT_TEAM_SUMMARY` | S → all P | Game completion |
| `ANALYTICS_TO_HOST_COMPLETE_REPORT` | S → H | Game completion |
| `ANALYTICS_TO_SERVER_RESET_GAME` | H → S | Host resets for a new game |
| `ANALYTICS_TO_CLIENT_GAME_RESET` | S → all | Server has reset; clients must reconnect |
| `SYSTEM_TO_CLIENT_ERROR` | S → P | Error conditions (see error code registry) |
| `SYSTEM_TO_HOST_ERROR` | S → H | Host-specific errors |
| `SYSTEM_TO_CLIENT_HOST_DISCONNECTED` | S → all P | Host disconnects |
| `SYSTEM_TO_CLIENT_HOST_RECONNECTED` | S → all P | Host reconnects |
| `SYSTEM_TO_HOST_PLAYER_DISCONNECTED` | S → H | A player disconnects |
| `SYSTEM_PING` | C → S | Client heartbeat (every 30 seconds) |
| `SYSTEM_PONG` | S → C | Heartbeat response |

---

## Phase Transition Sequences

The events fired in order at each phase boundary. See each event's own subsection for trigger conditions and payload details.

**Setup → Resource Gathering**
1. Host sends `SETUP_TO_SERVER_START_GAME`
2. Host receives `SETUP_TO_HOST_GAME_STARTED`, then `RESOURCE_TO_HOST_PHASE_START`
3. All players receive `RESOURCE_TO_CLIENT_PHASE_START`
4. Server waits one round duration before the first `RESOURCE_TO_PLAYER_TRIVIA_QUESTION`

**Resource Gathering → Puzzle Assembly**
1. Final round completes automatically; the game enters `puzzle_preparation`
2. Host receives `RESOURCE_TO_HOST_PHASE_COMPLETE`; players receive `RESOURCE_TO_CLIENT_PHASE_COMPLETE`
3. Host receives `PUZZLE_TO_HOST_PREPARING` while the server generates tiles in memory
4. Host receives `PUZZLE_TO_HOST_READY`, then `PUZZLE_TO_HOST_PHASE_LOAD`
5. Players receive `PUZZLE_TO_CLIENT_PHASE_LOAD` and begin fetching their segment from `GET /api/segments/{segmentId}` (puzzle UI stays hidden)
6. Host sends `PUZZLE_TO_SERVER_PHASE_START` when ready (rejected before step 4)
7. Host receives `PUZZLE_TO_HOST_PHASE_START`; players receive `PUZZLE_TO_CLIENT_PHASE_START` and reveal the puzzle UI; the game enters `puzzle_assembly`

**Puzzle Assembly → Analytics**
1. Puzzle completes successfully or timer expires
2. Players receive `PUZZLE_TO_CLIENT_COMPLETED_SUCCESS` or `PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT`
3. Host receives `PUZZLE_TO_HOST_COMPLETION_ANALYTICS`, then `ANALYTICS_TO_HOST_COMPLETE_REPORT`
4. Each player receives `ANALYTICS_TO_PLAYER_PERSONAL_REPORT`, then `ANALYTICS_TO_CLIENT_TEAM_SUMMARY`

---

## Reconnection Behavior

Both host and players reconnect using the same WebSocket endpoint and token they used originally — the host presents its UUID in the connection URL; a player presents theirs in the `auth` field of `SETUP_TO_SERVER_PLAYER_CONNECT`, the first frame after opening the socket. The handshake event (`SETUP_TO_HOST_CONNECTION_CONFIRMED` for the host, `SETUP_TO_PLAYER_CONNECTION_CONFIRMED` for a player) carries `isReconnection: true` and `currentPhase`. The server then replays a phase-appropriate sequence of state-restoration events (defined under each event's own subsection — only the sequence is listed here).

### Host

| Phase | Reconnect allowed? | State-restoration events sent after handshake |
|---|---|---|
| Setup | Yes | `SETUP_TO_HOST_PLAYER_ROSTER` |
| Resource Gathering | Yes | `RESOURCE_TO_HOST_PHASE_START`; `RESOURCE_TO_HOST_ROUND_ANALYTICS` if a round is in progress |
| Puzzle Preparation | Yes | `PUZZLE_TO_HOST_PREPARING` if tile generation is still running, otherwise `PUZZLE_TO_HOST_READY` then `PUZZLE_TO_HOST_PHASE_LOAD` |
| Puzzle Assembly | Yes | `PUZZLE_TO_HOST_PHASE_LOAD`; `PUZZLE_TO_HOST_GRID_STATE`; `PUZZLE_TO_HOST_PHASE_START` re-anchored to the resume (`startTimestamp` = resume time, `totalTime` = seconds remaining after the pause extension) |
| Analytics | Yes | `ANALYTICS_TO_HOST_COMPLETE_REPORT` |

All currently-connected players additionally receive `SYSTEM_TO_CLIENT_HOST_RECONNECTED`.

If a host socket is already open, a new connection presenting the valid host UUID supersedes it: the older socket is closed with code `1000` and the new connection receives the normal reconnection handshake.

### Player

| Phase | Reconnect allowed? | State-restoration events sent after handshake | Disconnect impact |
|---|---|---|---|
| Setup | Yes | `SETUP_TO_PLAYER_ROLES_AVAILABLE` (unless the player is ready). The handshake's `existingConfiguration` restores specialty and name; the role is restored only if a slot is still available, otherwise the player must reselect (see *Race Resolution* in `game-design.md`). All players receive an updated `SETUP_TO_CLIENT_LOBBY_STATUS`; host receives `SETUP_TO_HOST_PLAYER_ROSTER`. | Player removed from connected/ready counts and role distribution while disconnected; re-added on reconnect. |
| Resource Gathering | Yes | `RESOURCE_TO_CLIENT_PHASE_START`; `RESOURCE_TO_CLIENT_TEAM_PROGRESS`; the current round's `RESOURCE_TO_PLAYER_TRIVIA_QUESTION` if mid-round. | Player remains in counts; tokens stay in team total. |
| Puzzle Preparation | Yes | `PUZZLE_TO_CLIENT_PHASE_LOAD` if tile generation has already finished; otherwise nothing further (the load event arrives with the normal broadcast). | Player remains in counts; if still disconnected when the host starts the puzzle timer, their segment is auto-solved into an unassigned fragment at that moment. |
| Puzzle Assembly | **No.** | The server completes the WebSocket upgrade and immediately closes it with code `4003`; no application frames are exchanged. | Disconnected player's individual puzzle is auto-solved into an unassigned fragment; remaining players control it as any unassigned fragment. |
| Analytics | Yes | `ANALYTICS_TO_PLAYER_PERSONAL_REPORT`; `ANALYTICS_TO_CLIENT_TEAM_SUMMARY`. | Player remains in counts; analytics preserved. |

---

## Phase 0: Connection and Setup

### Initial Connection Events

#### `SETUP_TO_HOST_CONNECTION_CONFIRMED`
**Direction**: Server → Host
**Trigger**: Host connects to `/ws/host/{uuid}` (initial connection or reconnection)

```json
{
  "event": "SETUP_TO_HOST_CONNECTION_CONFIRMED",
  "payload": {
    "hostId": "host-uuid",
    "currentPhase": "setup",
    "isReconnection": false,
    "gameConfig": {
      "minPlayers": 4,
      "maxPlayers": 64,
      "resourceGatheringRounds": 5,
      "triviaAnswerTime": 30,
      "triviaGraceTime": 30,
      "puzzleBaseTime": 300,
      "difficultyMode": "medium"
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`hostId` echoes the host UUID from the connection URL — it is the token the host uses for all subsequent authenticated messages and HTTP asset fetches.

#### `SETUP_TO_SERVER_PLAYER_CONNECT`
**Direction**: Player → Server
**Trigger**: The first frame a player client sends after the WebSocket to `/ws` opens. With `auth` present, it is a reconnection attempt using the previously issued token; with `auth` omitted (or `null`), it is a new player joining. The server replies with `SETUP_TO_PLAYER_CONNECTION_CONFIRMED`, or closes the socket with `4001`/`4002`/`4003`/`4004` (see *WebSocket close codes*). If no connect frame arrives within 10 seconds of the upgrade, the server closes with `4001`. Host connections do not send this event — the host token travels in the URL.

```json
{
  "event": "SETUP_TO_SERVER_PLAYER_CONNECT",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {},
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `SETUP_TO_PLAYER_CONNECTION_CONFIRMED`
**Direction**: Server → Player
**Trigger**: Sent in response to a valid `SETUP_TO_SERVER_PLAYER_CONNECT` (initial connection or reconnection). This is the player handshake event, mirroring `SETUP_TO_HOST_CONNECTION_CONFIRMED`, and is sent in every phase that permits player connections.

```json
{
  "event": "SETUP_TO_PLAYER_CONNECTION_CONFIRMED",
  "payload": {
    "playerId": "uuid-generated-by-server",
    "currentPhase": "setup",
    "isReconnection": false,
    "existingConfiguration": null
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`existingConfiguration` is `null` on first connection. On reconnection it carries the player's preserved state:

```json
"existingConfiguration": {
  "selectedRole": "detective",
  "selectedSpecialties": ["science"],
  "playerName": "Alice",
  "ready": true
}
```

`selectedRole` is `null` if the previously held role lost its slot while the player was disconnected (setup phase only); the player must reselect a role and resubmit. After the handshake, the server sends the phase-appropriate state-restoration events listed under *Reconnection Behavior*.

#### `SETUP_TO_PLAYER_ROLES_AVAILABLE`
**Direction**: Server → Player
**Trigger**: Sent immediately after `SETUP_TO_PLAYER_CONNECTION_CONFIRMED` during setup, and again whenever role availability changes (players joining/leaving, role slots filling or freeing).

**Important**: This event is only sent to players who have not yet marked themselves as ready — a ready player's configuration is locked, so availability updates are irrelevant to them. (A ready player who reconnects still receives the connection handshake above; they simply don't receive this event.)

Example snapshot: early in setup, 4 players connected (role capacity `max(1, ceil(4/4)) = 1`) and only Alice has configured — so detective is at capacity and unavailable, the other three roles are open.

```json
{
  "event": "SETUP_TO_PLAYER_ROLES_AVAILABLE",
  "payload": {
    "roles": [
      {
        "roleType": "art_enthusiast",
        "displayName": "Art Enthusiast",
        "resourceBonus": 1.5,
        "bonusTokenType": "clarity",
        "description": "Excels at clarity token collection",
        "available": true
      },
      {
        "roleType": "detective",
        "displayName": "Detective",
        "resourceBonus": 1.5,
        "bonusTokenType": "guide",
        "description": "Excels at guide token collection",
        "available": false
      },
      {
        "roleType": "tourist",
        "displayName": "Tourist",
        "resourceBonus": 1.5,
        "bonusTokenType": "chronos",
        "description": "Excels at chronos token collection",
        "available": true
      },
      {
        "roleType": "janitor",
        "displayName": "Janitor",
        "resourceBonus": 1.5,
        "bonusTokenType": "anchor",
        "description": "Excels at anchor token collection",
        "available": true
      }
    ],
    "triviaCategories": [
      "general",
      "geography",
      "history",
      "music",
      "science",
      "video_games"
    ],
    "maxSpecialties": 1
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`maxSpecialties` echoes `gameConfig.maxSpecialtiesPerPlayer` — the upper bound on how many `selectedSpecialties` a configuration may contain.

### Role and Specialty Selection

#### `SETUP_TO_SERVER_PLAYER_CONFIGURATION`
**Direction**: Player → Server
**Trigger**: Player completes role and specialty selection

```json
{
  "event": "SETUP_TO_SERVER_PLAYER_CONFIGURATION",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "selectedRole": "art_enthusiast",
    "selectedSpecialties": ["science"],
    "playerName": "Player Display Name"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`selectedSpecialties` must contain 1 to `gameConfig.maxSpecialtiesPerPlayer` distinct known categories, else the message is rejected with `INVALID_SPECIALTY_SELECTION`.

The server processes these messages serially; if two players race for the last slot of a role, the first to land wins and the loser receives `SYSTEM_TO_CLIENT_ERROR` with `errorCode: "ROLE_FULL"` (see *Race Resolution* in `game-design.md`). A rejected configuration is discarded in full; the loser resubmits a complete configuration with a different role.

#### `SETUP_TO_CLIENT_LOBBY_STATUS`
**Direction**: Server → All Players
**Trigger**: Any player joins/leaves or changes configuration

```json
{
  "event": "SETUP_TO_CLIENT_LOBBY_STATUS",
  "payload": {
    "currentPlayers": 5,
    "minPlayers": 4,
    "maxPlayers": 64,
    "playerRoles": {
      "art_enthusiast": 1,
      "detective": 1,
      "tourist": 1,
      "janitor": 1
    },
    "hasHost": true,
    "allPlayersReady": false,
    "readyPlayers": 4,
    "gameStartEligible": false,
    "waitingMessage": "Waiting for 1 more player to be ready"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

Because configuration is atomic (role + specialty in one message, which also marks the player ready), `playerRoles` always sums to `readyPlayers` — here Eve has not yet configured, so her art-enthusiast slot isn't counted.

#### `SETUP_TO_HOST_PLAYER_ROSTER`
**Direction**: Server → Host
**Trigger**: Any change to player status

```json
{
  "event": "SETUP_TO_HOST_PLAYER_ROSTER",
  "payload": {
    "phase": "setup",
    "connectedPlayers": 5,
    "readyPlayers": 4,
    "gameStartEligible": false,
    "playerStatuses": {
      "player1-uuid": {
        "playerName": "Alice",
        "role": "detective",
        "specialties": ["science"],
        "connected": true,
        "ready": true,
        "lastActivity": "2025-06-15T14:23:05.000Z"
      },
      "player5-uuid": {
        "playerName": "Eve",
        "role": null,
        "specialties": [],
        "connected": true,
        "ready": false,
        "lastActivity": "2025-06-15T14:23:05.000Z"
      }
    },
    "roleDistribution": {
      "art_enthusiast": 1,
      "detective": 1,
      "tourist": 1,
      "janitor": 1
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

Same snapshot as the lobby-status example above: Eve is connected but hasn't configured yet, so her `role` is `null` and she isn't in `roleDistribution`. (Alice through Diana's entries are elided.)

### Game Start

#### `SETUP_TO_SERVER_START_GAME`
**Direction**: Host → Server
**Trigger**: Host clicks start game button

```json
{
  "event": "SETUP_TO_SERVER_START_GAME",
  "auth": {
    "token": "host-uuid"
  },
  "payload": {},
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `SETUP_TO_HOST_GAME_STARTED`
**Direction**: Server → Host
**Trigger**: Game successfully started

```json
{
  "event": "SETUP_TO_HOST_GAME_STARTED",
  "payload": {
    "phase": "resource_gathering",
    "totalPlayers": 5,
    "initialTeamTokens": {
      "anchorTokens": 0,
      "chronosTokens": 0,
      "guideTokens": 0,
      "clarityTokens": 0
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

---

## Phase 1: Resource Gathering

### Phase Initialization

#### `RESOURCE_TO_CLIENT_PHASE_START`
**Direction**: Server → All Players
**Trigger**: Resource gathering phase begins (marks transition from Setup phase)

**Important**: After sending this event, the server waits one full round duration (`gameConfig.triviaAnswerTime + gameConfig.triviaGraceTime` seconds) before starting Round 1 and sending the first `RESOURCE_TO_PLAYER_TRIVIA_QUESTION`. The `roundDuration` payload field is that derived sum, echoed for display convenience.

```json
{
  "event": "RESOURCE_TO_CLIENT_PHASE_START",
  "payload": {
    "phase": "resource_gathering",
    "totalRounds": 5,
    "roundDuration": 60,
    "answerTime": 30,
    "graceTime": 30,
    "tokenThresholds": {
      "anchor": 25,
      "chronos": 20,
      "guide": 15,
      "clarity": 30
    },
    "difficultySettings": {
      "mode": "medium",
      "specialtyProbability": 0.3,
      "timeMultiplier": 1.0,
      "thresholdMultiplier": 1.0
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

Station QR hashes are deliberately **never sent to clients**. A client only relays the raw text of a QR code it physically scanned (`RESOURCE_TO_SERVER_LOCATION_VERIFIED`); the server alone maps hashes to stations. Broadcasting the hashes would let players change stations without moving.

#### `RESOURCE_TO_HOST_PHASE_START`
**Direction**: Server → Host
**Trigger**: Resource gathering phase begins

```json
{
  "event": "RESOURCE_TO_HOST_PHASE_START",
  "payload": {
    "phase": "resource_gathering",
    "monitoringDashboard": {
      "totalRounds": 5,
      "currentRound": 0,
      "roundDuration": 60,
      "playerDistribution": {
        "anchor": 0,
        "chronos": 0,
        "guide": 0,
        "clarity": 0,
        "unknown": 5
      }
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

### Location Verification

#### `RESOURCE_TO_SERVER_LOCATION_VERIFIED`
**Direction**: Player → Server
**Trigger**: Player scans QR code at new station

```json
{
  "event": "RESOURCE_TO_SERVER_LOCATION_VERIFIED",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "stationHash": "hash_anchor_station_constant",
    "previousLocation": "clarity",
    "scanTimestamp": "2025-06-15T14:23:05.000Z"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`previousLocation` is **(optional)** — omitted on a player's first scan, when there is no prior station to report.

On a recognized hash, the server replies with `RESOURCE_TO_PLAYER_LOCATION_CONFIRMED` (below). On an unrecognized hash, the server returns `SYSTEM_TO_CLIENT_ERROR` with `errorCode: "INVALID_STATION_HASH"` and the player's station is unchanged.

#### `RESOURCE_TO_PLAYER_LOCATION_CONFIRMED`
**Direction**: Server → Individual Player
**Trigger**: Server validates and applies a station change in response to `RESOURCE_TO_SERVER_LOCATION_VERIFIED`

```json
{
  "event": "RESOURCE_TO_PLAYER_LOCATION_CONFIRMED",
  "payload": {
    "newLocation": "anchor"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`newLocation` is one of `anchor`, `chronos`, `guide`, `clarity`. The same value will subsequently appear in the next `RESOURCE_TO_CLIENT_TEAM_PROGRESS` broadcast, but this event lets the player's UI confirm the scan immediately rather than wait for the next progress tick.

### Trivia Questions and Answers

#### `RESOURCE_TO_PLAYER_TRIVIA_QUESTION`
**Direction**: Server → Individual Player
**Trigger**: Start of each resource gathering round

```json
{
  "event": "RESOURCE_TO_PLAYER_TRIVIA_QUESTION",
  "payload": {
    "questionId": "geography_medium_042",
    "questionText": "What is the capital of France?",
    "category": "geography",
    "difficulty": "medium",
    "isSpecialty": false,
    "options": [
      "Paris",
      "London",
      "Berlin",
      "Madrid"
    ],
    "roundNumber": 3,
    "totalRounds": 5,
    "answerDeadline": "2025-06-15T14:23:05.000Z"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`difficulty` is `gameConfig.difficultyMode` for regular questions and one level higher (capped at hard) for specialty questions. `questionId` follows `{category}_{difficulty}_{index}`.

#### `RESOURCE_TO_SERVER_TRIVIA_ANSWER`
**Direction**: Player → Server
**Trigger**: Player selects an answer. A player may resubmit before `answerDeadline`; the last answer received counts. If the player never selects one, the client sends nothing; at the answer deadline the server marks the question incorrect for that player. Answers arriving after the deadline are silently ignored.

```json
{
  "event": "RESOURCE_TO_SERVER_TRIVIA_ANSWER",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "questionId": "geography_medium_042",
    "answerIndex": 0,
    "timeElapsed": 18.5
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

The server determines which token type an answer earns from its own station tracking (QR scans), never from the client — specifically the station on record when the answer window closes. Players whose station is `unknown` at that moment (no successful scan yet this game) earn no tokens.

#### `RESOURCE_TO_PLAYER_ANSWER_RESULT`
**Direction**: Server → Individual Player
**Trigger**: Answer period ends for question

```json
{
  "event": "RESOURCE_TO_PLAYER_ANSWER_RESULT",
  "payload": {
    "questionId": "geography_medium_042",
    "correct": true,
    "selectedAnswer": "Paris",
    "correctAnswer": "Paris",
    "tokensEarned": 30,
    "baseTokens": 20,
    "bonuses": {
      "roleBonus": true,
      "roleBonusTokens": 10,
      "specialtyBonus": false,
      "specialtyBonusTokens": 0
    },
    "currentLocation": "clarity",
    "nextTriviaTimestamp": "2025-06-15T14:23:05.000Z"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`tokensEarned` follows the multiplicative award formula in `game-design.md` § *Token Scoring*. The bonus fields decompose it for display: `roleBonusTokens = baseTokens × (roleResourceMultiplier − 1)` when at the matching station, and `specialtyBonusTokens = baseTokens × roleFactor × (specialtyPointMultiplier − 1)` for specialty questions (each is `0` when its flag is `false`).

`currentLocation` is the server-tracked station — `"unknown"` if the player has never scanned one, in which case `tokensEarned` is `0` even for a correct answer.

### Progress Updates

#### `RESOURCE_TO_CLIENT_TEAM_PROGRESS`
**Direction**: Server → All Players
**Trigger**: End of each round's answer window, after answers are marked and tokens awarded

```json
{
  "event": "RESOURCE_TO_CLIENT_TEAM_PROGRESS",
  "payload": {
    "currentRound": 3,
    "totalRounds": 5,
    "questionsAnswered": 14,
    "totalQuestions": 25,
    "teamTokens": {
      "anchorTokens": 40,
      "chronosTokens": 30,
      "guideTokens": 30,
      "clarityTokens": 40
    },
    "currentThresholds": {
      "anchor": 1,
      "chronos": 1,
      "guide": 2,
      "clarity": 1
    },
    "teamPerformance": {
      "averageAccuracy": 0.79,
      "roundTimeRemaining": 28
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `RESOURCE_TO_HOST_ROUND_ANALYTICS`
**Direction**: Server → Host
**Trigger**: After each question round

```json
{
  "event": "RESOURCE_TO_HOST_ROUND_ANALYTICS",
  "payload": {
    "currentRound": 3,
    "totalRounds": 5,
    "roundResults": {
      "questionsDelivered": 5,
      "answersReceived": 5,
      "correctAnswers": 4,
      "averageResponseTime": 22.3,
      "tokensAwarded": 90
    },
    "playerPerformance": {
      "player1-uuid": {
        "location": "guide",
        "answerCorrect": true,
        "responseTime": 18.2,
        "tokensEarned": 30,
        "runningAccuracy": 1.0
      },
      "player2-uuid": {
        "location": "clarity",
        "answerCorrect": false,
        "responseTime": 28.7,
        "tokensEarned": 0,
        "runningAccuracy": 0.67
      }
    },
    "stationDistribution": {
      "anchor": 1,
      "chronos": 1,
      "guide": 2,
      "clarity": 1
    },
    "teamTokens": {
      "anchorTokens": 40,
      "chronosTokens": 30,
      "guideTokens": 30,
      "clarityTokens": 40
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

### Phase Completion

#### `RESOURCE_TO_CLIENT_PHASE_COMPLETE`
**Direction**: Server → All Players
**Trigger**: All resource gathering rounds completed

```json
{
  "event": "RESOURCE_TO_CLIENT_PHASE_COMPLETE",
  "payload": {
    "phase": "resource_gathering",
    "nextPhase": "puzzle_preparation",
    "finalTokenTotals": {
      "anchorTokens": 90,
      "chronosTokens": 70,
      "guideTokens": 50,
      "clarityTokens": 70
    },
    "thresholdAchievements": {
      "anchor": 3,
      "chronos": 3,
      "guide": 3,
      "clarity": 2
    },
    "bonusEffects": {
      "anchorPreSolved": 6,
      "chronosTimeBonus": 60,
      "guideHighlightCount": 5,
      "clarityPreviewDuration": 5
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `RESOURCE_TO_HOST_PHASE_COMPLETE`
**Direction**: Server → Host
**Trigger**: Resource gathering phase ends

```json
{
  "event": "RESOURCE_TO_HOST_PHASE_COMPLETE",
  "payload": {
    "phase": "resource_gathering",
    "totalQuestionsAnswered": 25,
    "teamPerformance": {
      "overallAccuracy": 0.76,
      "totalTokensEarned": 280,
      "averageResponseTime": 23.5
    },
    "finalTokenDistribution": {
      "anchorTokens": 90,
      "chronosTokens": 70,
      "guideTokens": 50,
      "clarityTokens": 70
    },
    "playerAnalytics": {
      "player1-uuid": {
        "questionsAnswered": 5,
        "correctAnswers": 4,
        "accuracy": 0.8,
        "tokensEarned": 130,
        "specialtyPerformance": {
          "questionsReceived": 2,
          "correctAnswers": 2,
          "bonusTokens": 40
        }
      }
    },
    "readyForPuzzlePhase": true
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

---

## Phase 2: Puzzle Assembly

### Segment ID Convention

Segment IDs are deterministic given the central grid size and follow the
pattern `segment_{row}{col}` where `row` is a lowercase letter starting at
`a` and `col` is a 1-based number. For a `centralGridSize` of N, valid IDs
are `segment_a1` through `segment_{Nth letter}{N}`.

For example, a 4×4 grid has segments `segment_a1`, `segment_a2`, …,
`segment_a4`, `segment_b1`, …, `segment_d4`. An 8×8 grid has segments
`segment_a1` through `segment_h8`.

Clients and the host derive the full set of valid segment IDs from
`centralGridSize` (delivered in `PUZZLE_TO_CLIENT_PHASE_LOAD` and
`PUZZLE_TO_HOST_PHASE_LOAD`); the server never sends the enumeration.

Fragments are identified by the `segmentId` of the segment they contain —
there is no separate fragment ID namespace.

**Coordinate mapping**: grid positions are `{x, y}` with `x` = column
(`col − 1`) and `y` = row (letter index, `a` = 0), both 0-based from the
top-left. A segment's correct cell follows directly: `segment_a1` belongs at
`{x: 0, y: 0}`, `segment_c3` at `{x: 2, y: 2}`. The server uses this mapping
for victory validation and guide highlights.

### Puzzle Preparation

When resource gathering ends, the game enters the `puzzle_preparation` phase, during which the server crops the configured source image into per-segment tiles in memory. The host's `PUZZLE_TO_SERVER_PHASE_START` is rejected until tile generation completes. (For the gameplay rationale of this pause, see `game-design.md` § *Puzzle Preparation Phase*.)

#### `PUZZLE_TO_HOST_PREPARING`
**Direction**: Server → Host
**Trigger**: Sent immediately after the final resource-gathering round completes, signalling that the server has begun tile generation. The host UI should display a "preparing puzzle…" indicator and disable any "start puzzle" controls until `PUZZLE_TO_HOST_READY` arrives.

```json
{
  "event": "PUZZLE_TO_HOST_PREPARING",
  "payload": {},
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `PUZZLE_TO_HOST_READY`
**Direction**: Server → Host
**Trigger**: All segment tiles are cached in memory and the server is ready to deliver them through `/api/segments/{segmentId}`. Sent immediately before `PUZZLE_TO_HOST_PHASE_LOAD` and the per-client `PUZZLE_TO_CLIENT_PHASE_LOAD` broadcast. Once received, the host may send `PUZZLE_TO_SERVER_PHASE_START` to begin the phase timer.

```json
{
  "event": "PUZZLE_TO_HOST_READY",
  "payload": {},
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

### Phase Initialization

#### `PUZZLE_TO_CLIENT_PHASE_LOAD`
**Direction**: Server → All Players
**Trigger**: Tile preparation completes (still within the `puzzle_preparation` phase, emitted immediately after `PUZZLE_TO_HOST_READY`).

**Important**: Players can load their puzzle segments via `GET /api/segments/{segmentId}` (see *Asset Delivery (HTTP)* below) and prepare the puzzle UI on receipt, but the puzzle phase timer doesn't start until the host sends `PUZZLE_TO_SERVER_PHASE_START`. Pieces should remain hidden until the host triggers the start. `imageId` is the configured `gameConfig.puzzleImage` filename.

```json
{
  "event": "PUZZLE_TO_CLIENT_PHASE_LOAD",
  "payload": {
    "phase": "puzzle_preparation",
    "imageId": "nature_image.png",
    "assignedSegmentId": "segment_a1",
    "individualPuzzleSize": 16,
    "anchorPreSolvedPieces": 6,
    "centralGridSize": 3,
    "totalFragments": 9,
    "clarityPreviewDuration": 5,
    "guideHighlightCount": 5
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `PUZZLE_TO_HOST_PHASE_LOAD`
**Direction**: Server → Host
**Trigger**: Tile preparation completes (sent immediately after `PUZZLE_TO_HOST_READY`)

```json
{
  "event": "PUZZLE_TO_HOST_PHASE_LOAD",
  "payload": {
    "phase": "puzzle_preparation",
    "imageId": "nature_image.png",
    "centralGridSize": 3,
    "totalFragments": 9,
    "playerCount": 5,
    "playerSegmentAssignments": {
      "player1-uuid": "segment_a1",
      "player2-uuid": "segment_a2",
      "player3-uuid": "segment_a3",
      "player4-uuid": "segment_b1",
      "player5-uuid": "segment_b2"
    },
    "bonusEffects": {
      "anchorPreSolved": 6,
      "chronosTimeBonus": 60,
      "guideHighlightCount": 5,
      "clarityPreviewDuration": 5
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

### Asset Delivery (HTTP)

Puzzle imagery is delivered over authenticated HTTP requests, **not** through WebSocket payloads. The `PUZZLE_TO_*_PHASE_LOAD` events tell each client which `segmentId` they own; the bytes themselves are fetched from the endpoints below. This keeps WebSocket frames small and centralizes server-side authorization on tile access.

**Authentication**: All asset endpoints require an `Authorization: Bearer {token}` header. The token is the same session UUID used for WebSocket auth — for players this is the UUID issued at first connection; for the host it is the server-startup UUID embedded in the host WebSocket URL. Tokens in query strings are not accepted (they leak into proxy logs). Missing/malformed/unrecognized tokens receive `401 Unauthorized`.

**Response content type**: Successful responses set `Content-Type: image/png`.

**Error response shape**: Non-2xx responses return JSON with this shape:

```json
{
  "error": "FORBIDDEN_NOT_OWNER",
  "message": "Segment segment_a3 is not assigned to this player."
}
```

The `error` field is a fixed machine-readable code from the *Error code registry* (Conventions section); `message` is human-readable and intended for logs and developer-facing UI, not end-user messaging.

#### `GET /api/segments/{segmentId}`
Returns PNG bytes of a single puzzle segment. The server enforces, in order:

1. The token must correspond to a known player or the host of the current game.
2. The current phase must be `puzzle_preparation` with tile generation complete (see `PUZZLE_TO_HOST_READY`), `puzzle_assembly`, or `analytics` (the final grid stays viewable on result screens until the game resets).
3. The requested `segmentId` must match either:
   - the requesting player's own assigned segment, OR
   - a fragment that has already been completed and is now visible to all players on the central grid, OR
   - any segment if the request comes from the host (read-only access for monitoring).

Authorization failures return `FORBIDDEN_NOT_OWNER` or `FORBIDDEN_PHASE` as appropriate.

#### `GET /api/preview/full`
Returns PNG bytes of the complete (un-cropped) puzzle source image, used for the clarity-token preview. Server enforces:

1. The token must correspond to a known player or the host.
2. The current phase must be `puzzle_assembly`.
3. The team must have earned at least one clarity-token threshold during resource gathering (see `game-design.md` § *Clarity Tokens*).
4. The current server time must fall inside the active clarity-preview window (`gameConfig.clarityBasePreviewTime + (N × gameConfig.previewTimePerThreshold)` seconds, starting from `PUZZLE_TO_CLIENT_PHASE_START`, where `N` is the number of clarity thresholds earned).

If the team earned zero clarity thresholds, the endpoint returns `FORBIDDEN_PREVIEW_WINDOW_CLOSED` for the entire puzzle phase. Otherwise, requests outside the preview window return `FORBIDDEN_PREVIEW_WINDOW_CLOSED` regardless of token validity. The window is server-clock authoritative; clients cannot extend it.

**Caching**: Tile responses include `Cache-Control: no-cache` (clients revalidate on each request). Tiles are small enough that revalidation is cheap, and `no-cache` avoids the risk of a stale response surviving past `ANALYTICS_TO_CLIENT_GAME_RESET` within the cache's max-age window.

**Lifecycle**: Tiles exist only between `PUZZLE_TO_HOST_READY` (server has finished generation) and `ANALYTICS_TO_CLIENT_GAME_RESET` (server clears the in-memory cache). Requests outside that window return `NOT_FOUND`.

### Phase Start Management

#### `PUZZLE_TO_SERVER_PHASE_START`
**Direction**: Host → Server
**Trigger**: Host starts puzzle phase. Rejected with `SYSTEM_TO_HOST_ERROR` / `FORBIDDEN_PHASE` if tile preparation has not yet completed (i.e. before `PUZZLE_TO_HOST_READY`).

```json
{
  "event": "PUZZLE_TO_SERVER_PHASE_START",
  "auth": {
    "token": "host-uuid"
  },
  "payload": {},
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `PUZZLE_TO_CLIENT_PHASE_START`
**Direction**: Server → All Players
**Trigger**: Puzzle phase started by host

```json
{
  "event": "PUZZLE_TO_CLIENT_PHASE_START",
  "payload": {
    "startTimestamp": "2025-06-15T14:23:05.000Z",
    "totalTime": 360,
    "baseTime": 300,
    "chronosBonus": 60,
    "clarityPreviewActive": true,
    "clarityPreviewDuration": 5,
    "playerPhases": {
      "phase2a": ["player1-uuid", "player2-uuid", "player3-uuid", "player4-uuid", "player5-uuid"],
      "phase2b": []
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`playerPhases` partitions every connected player by current sub-phase: `phase2a` for those still solving their individual puzzle privately, `phase2b` for those who have completed it and are now collaborating on the central grid. At phase start every player is in `phase2a`. Subsequent transitions are reflected in `PUZZLE_TO_HOST_GRID_STATE.playerMetrics[*].phase`.

`clarityPreviewActive` and `clarityPreviewDuration` are gated on whether the team earned at least one clarity-token threshold during resource gathering (see `game-design.md` § *Clarity Tokens*). If zero clarity thresholds were earned, `clarityPreviewActive: false` and `clarityPreviewDuration: 0`; clients render no preview overlay and the corresponding `PUZZLE_TO_CLIENT_PREVIEW_EXPIRED` event is not emitted.

#### `PUZZLE_TO_HOST_PHASE_START`
**Direction**: Server → Host
**Trigger**: Puzzle phase started

```json
{
  "event": "PUZZLE_TO_HOST_PHASE_START",
  "payload": {
    "timerActive": true,
    "startTimestamp": "2025-06-15T14:23:05.000Z",
    "totalTime": 360,
    "baseTime": 300,
    "chronosBonus": 60,
    "playersInPhase2a": 5,
    "playersInPhase2b": 0
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `PUZZLE_TO_CLIENT_PREVIEW_EXPIRED`
**Direction**: Server → All Players
**Trigger**: The clarity-token full-image preview window has elapsed. Sent at most once per game, `clarityPreviewDuration` seconds after `PUZZLE_TO_CLIENT_PHASE_START` (the preview clock pauses during a host-disconnect pause, like every other timer), and **only** if the team earned at least one clarity-token threshold (i.e. `PUZZLE_TO_CLIENT_PHASE_START.clarityPreviewActive` was `true`). Games where no clarity thresholds were earned never see a preview window open and never receive this event. Clients should dismiss the preview overlay on receipt rather than relying on a locally-computed deadline (which can drift).

After this event fires, `GET /api/preview/full` returns `403 FORBIDDEN_PREVIEW_WINDOW_CLOSED` for the rest of the game.

```json
{
  "event": "PUZZLE_TO_CLIENT_PREVIEW_EXPIRED",
  "payload": {},
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

### Individual Puzzle Solving (Phase 2A)

#### `PUZZLE_TO_SERVER_SEGMENT_COMPLETED`
**Direction**: Player → Server
**Trigger**: Player completes their individual puzzle

```json
{
  "event": "PUZZLE_TO_SERVER_SEGMENT_COMPLETED",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "segmentId": "segment_a1",
    "completionTimestamp": "2025-06-15T14:23:05.000Z",
    "solveTime": 180,
    "manualPiecesSolved": 10,
    "preSolvedPieces": 6
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `PUZZLE_TO_PLAYER_SEGMENT_ACKNOWLEDGED`
**Direction**: Server → Individual Player
**Trigger**: Server processes segment completion

```json
{
  "event": "PUZZLE_TO_PLAYER_SEGMENT_ACKNOWLEDGED",
  "payload": {
    "segmentId": "segment_a1",
    "position": {"x": 2, "y": 1}
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

The completing player learns about other players' progress through subsequent
`PUZZLE_TO_CLIENT_GRID_STATE` updates — every completed individual puzzle
appears as a new fragment on the central grid. There is no separate
team-completion-status payload.

#### `PUZZLE_TO_HOST_SEGMENT_COMPLETED`
**Direction**: Server → Host
**Trigger**: Player completes individual segment

```json
{
  "event": "PUZZLE_TO_HOST_SEGMENT_COMPLETED",
  "payload": {
    "playerId": "player1-uuid",
    "playerName": "Alice",
    "segmentId": "segment_a1",
    "completionTime": 180,
    "position": {"x": 2, "y": 1},
    "phaseTransition": {
      "playersInPhase2a": 4,
      "playersInPhase2b": 1
    },
    "completionStats": {
      "totalCompleted": 1,
      "totalRequired": 5,
      "unassignedFragments": 1
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`unassignedFragments` counts the unassigned fragments currently visible on the grid (per the *Proportional Unassigned Fragment Reveal* rule in `game-design.md`), not the game's total.

### Collaborative Fragment Movement (Phase 2B)

#### `PUZZLE_TO_SERVER_FRAGMENT_MOVE`
**Direction**: Player → Server
**Trigger**: Player attempts to move or swap a fragment on the central grid

```json
{
  "event": "PUZZLE_TO_SERVER_FRAGMENT_MOVE",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "segmentId": "segment_a1",
    "targetPosition": {"x": 0, "y": 0},
    "swapWithSegmentId": "segment_b3"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

The event covers two move modes:

- **Swap**: `swapWithSegmentId` is set. The fragment at `targetPosition` (which must be `swapWithSegmentId`) and the moving fragment exchange positions. Both fragments must be controllable by the caller (their own fragment or unassigned ones — in the example, `segment_b3` is unassigned); a swap that would displace another player's owned fragment is rejected with reason `not_owner` — propose it via `PUZZLE_TO_SERVER_RECOMMEND_MOVE` instead.
- **Move to empty cell**: `swapWithSegmentId` is `null` or omitted **(optional)**. `targetPosition` must reference an unoccupied cell; the moving fragment relocates into it.

The moving fragment is identified by `segmentId` alone; the server always uses its own authoritative position for it. All game-rule violations are reported through `PUZZLE_TO_PLAYER_MOVE_RESULT` with `status: "rejected"` and the matching `reason` (`not_owner`, `target_invalid`, `cooldown`, `phase_invalid`); `SYSTEM_TO_CLIENT_ERROR` / `MALFORMED_PAYLOAD` is reserved for structurally invalid payloads (missing fields, wrong types).

#### `PUZZLE_TO_PLAYER_MOVE_RESULT`
**Direction**: Server → Individual Player
**Trigger**: Server processes a movement request

`status` is exactly `"success"` or `"rejected"`. On success the payload describes the resulting board state and the moved fragment's cooldown (cooldowns are per fragment; on a swap, both fragments' cooldowns restart — `cooldownInfo` describes the moving fragment). On rejection the payload carries a `reason` enum and no state-change fields.

**Success example:**

```json
{
  "event": "PUZZLE_TO_PLAYER_MOVE_RESULT",
  "payload": {
    "moveId": "move-uuid-12345",
    "status": "success",
    "segmentId": "segment_a1",
    "newPosition": {"x": 0, "y": 0},
    "swappedSegmentId": "segment_b3",
    "swappedSegmentNewPosition": {"x": 2, "y": 1},
    "cooldownInfo": {
      "nextMoveAvailable": "2025-06-15T14:23:07.500Z",
      "cooldownRemaining": 2.5
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`swappedSegmentId` and `swappedSegmentNewPosition` are **(optional)** — present only when the move was a swap; absent when the move was into an empty cell.

**Rejection example:**

```json
{
  "event": "PUZZLE_TO_PLAYER_MOVE_RESULT",
  "payload": {
    "moveId": "move-uuid-12345",
    "status": "rejected",
    "segmentId": "segment_a1",
    "reason": "cooldown",
    "cooldownInfo": {
      "nextMoveAvailable": "2025-06-15T14:23:06.200Z",
      "cooldownRemaining": 1.2
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`reason` is one of:

| Value | Meaning |
|---|---|
| `cooldown` | Target fragment is still in its `gameConfig.fragmentMoveCooldown` window from a prior move. `cooldownInfo` describes when it will be free. |
| `not_owner` | Caller does not control the moving fragment (not theirs and not unassigned), or the swap would displace another player's owned fragment. |
| `target_invalid` | `targetPosition` is out of bounds, or the swap/empty-cell mode declared in the request does not match the actual occupant of `targetPosition`. |
| `phase_invalid` | Move arrived outside the puzzle assembly phase, before `PUZZLE_TO_CLIENT_PHASE_START`, while the game is paused by a host disconnect, or from a player still in Phase 2A. |

`cooldownInfo` is **(optional)** on rejection — present only when `reason` is `cooldown`.

### Grid State Updates

#### `PUZZLE_TO_CLIENT_GRID_STATE`
**Direction**: Server → All Players
**Trigger**: Every `gameConfig.gridUpdateInterval` seconds

The `fragments` array grows over time as players complete their individual puzzles. Per the *Proportional Unassigned Fragment Reveal* rule in `game-design.md`, after *k* of *N* players have completed their individual puzzles the array contains `ceil((k / N) × gridSize²)` entries — *k* player-owned fragments plus the rest randomly-selected unassigned fragments at random unoccupied positions.

`playerId` and `playerName` identify each fragment's owner; both are `null` for unassigned fragments. Clients need this to apply the movement rules locally (own + unassigned fragments are movable) and to address recommendations (`targetPlayerId`).

```json
{
  "event": "PUZZLE_TO_CLIENT_GRID_STATE",
  "payload": {
    "fragments": [
      {
        "segmentId": "segment_a1",
        "playerId": "player1-uuid",
        "playerName": "Alice",
        "position": {"x": 0, "y": 0}
      },
      {
        "segmentId": "segment_b3",
        "playerId": null,
        "playerName": null,
        "position": {"x": 2, "y": 1}
      }
    ],
    "timeRemaining": 165
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `PUZZLE_TO_HOST_GRID_STATE`
**Direction**: Server → Host
**Trigger**: Immediate updates on any change

The `fragments` array follows the same proportional-reveal rule as `PUZZLE_TO_CLIENT_GRID_STATE` — only the fragments currently visible on the central grid are included.

```json
{
  "event": "PUZZLE_TO_HOST_GRID_STATE",
  "payload": {
    "fragments": [
      {
        "playerId": "player1-uuid",
        "playerName": "Alice",
        "segmentId": "segment_a1",
        "position": {"x": 0, "y": 0},
        "lastMoved": "2025-06-15T14:23:05.000Z",
        "moveCount": 3
      }
    ],
    "playerMetrics": {
      "player1-uuid": {
        "phase": "phase2b",
        "fragmentsOwned": 1,
        "movesContributed": 3,
        "successfulMoves": 2,
        "lastActivity": "2025-06-15T14:23:05.000Z"
      }
    },
    "activeRecommendations": 2,
    "timeRemaining": 165
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `PUZZLE_TO_PLAYER_PERSONAL_STATE`
**Direction**: Server → Individual Player
**Trigger**: Same cadence as `PUZZLE_TO_CLIENT_GRID_STATE` (every `gameConfig.gridUpdateInterval` seconds), and additionally one snapshot sent immediately after the player's individual puzzle completes so the player's first set of guide highlights arrives without a delay. Sent only to players who have completed their individual puzzle (Phase 2B); Phase 2A players do not receive this event.

The `guideHighlights` array carries the cells on the central grid currently highlighted as possible positions for *this specific player's* fragment. Highlight count follows the formula in `game-design.md` § *Guide Tokens*. The list is private — different players receive different `guideHighlights` arrays in the same tick.

```json
{
  "event": "PUZZLE_TO_PLAYER_PERSONAL_STATE",
  "payload": {
    "guideHighlights": [
      {"x": 0, "y": 0},
      {"x": 0, "y": 2},
      {"x": 1, "y": 1},
      {"x": 2, "y": 0},
      {"x": 2, "y": 2}
    ]
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`guideHighlights` is an empty array until the team earns its first guide-token threshold. The set always contains the fragment's correct cell and is stable between ticks — it changes only if a new guide threshold is earned (see `game-design.md` § *Guide Tokens*). This is the only client-visible delivery channel for guide highlights — they intentionally do not appear in the broadcast `PUZZLE_TO_CLIENT_GRID_STATE`.

### Strategic Collaboration

#### `PUZZLE_TO_SERVER_RECOMMEND_MOVE`
**Direction**: Player → Server
**Trigger**: Player sends recommendation to another player

Validation: the sender must be in Phase 2B and control `fromSegmentId` (their own or unassigned); `toSegmentId` must be owned by `targetPlayerId` (recommendations exist precisely for the swaps a player cannot execute directly); both fragments must be off cooldown, otherwise the request is rejected with `SYSTEM_TO_CLIENT_ERROR` / `COOLDOWN_ACTIVE`; and the sender must have no other outgoing recommendation pending, otherwise `RECOMMENDATION_PENDING`.

```json
{
  "event": "PUZZLE_TO_SERVER_RECOMMEND_MOVE",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "targetPlayerId": "player2-uuid",
    "fromSegmentId": "segment_a1",
    "toSegmentId": "segment_a2",
    "reasoning": "This swap would place both fragments closer to correct positions"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `PUZZLE_TO_PLAYER_MOVE_RECOMMENDATION`
**Direction**: Server → Target Player
**Trigger**: Another player sends recommendation

```json
{
  "event": "PUZZLE_TO_PLAYER_MOVE_RECOMMENDATION",
  "payload": {
    "moveId": "rec-uuid-67890",
    "fromPlayerId": "player1-uuid",
    "fromPlayerName": "Alice",
    "targetPlayerId": "player2-uuid",
    "fromSegmentId": "segment_a1",
    "toSegmentId": "segment_a2",
    "reasoning": "This swap would place both fragments closer to correct positions",
    "expiresAt": "2025-06-15T14:23:35.000Z"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`expiresAt` is creation time plus `gameConfig.recommendationTimeout` seconds. Fragment positions are intentionally not included — a recommendation names fragments, not positions, and survives fragment moves; the target reads current positions from grid state.

#### `PUZZLE_TO_SERVER_RECOMMENDATION_RESPONSE`
**Direction**: Player → Server
**Trigger**: Player responds to recommendation

```json
{
  "event": "PUZZLE_TO_SERVER_RECOMMENDATION_RESPONSE",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "moveId": "rec-uuid-67890",
    "response": "accept",
    "responseReason": "Good strategic move"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`response` is exactly `"accept"` or `"reject"`. `responseReason` is **(optional)**; clients may send a short human-readable string for analytics or omit the field.

An `accept` executes only if both fragments are still off cooldown; otherwise the responder receives `SYSTEM_TO_CLIENT_ERROR` / `COOLDOWN_ACTIVE` and the recommendation stays pending — it can be accepted again once the cooldown passes (or it times out). Reject example payload:

```json
{
  "moveId": "rec-uuid-67890",
  "response": "reject"
}
```

On reject, the swap is not executed and the recommender receives `PUZZLE_TO_PLAYER_RECOMMENDATION_RESULT` with `response: "reject"` and no `swapExecuted` field.

#### `PUZZLE_TO_PLAYER_RECOMMENDATION_RESULT`
**Direction**: Server → Original Player
**Trigger**: Target player responds to recommendation

```json
{
  "event": "PUZZLE_TO_PLAYER_RECOMMENDATION_RESULT",
  "payload": {
    "moveId": "rec-uuid-67890",
    "targetPlayerId": "player2-uuid",
    "targetPlayerName": "Bob",
    "response": "accept",
    "responseReason": "Good strategic move",
    "swapExecuted": {
      "segment1Id": "segment_a1",
      "segment1OldPosition": {"x": 0, "y": 0},
      "segment1NewPosition": {"x": 1, "y": 1},
      "segment2Id": "segment_a2",
      "segment2OldPosition": {"x": 1, "y": 1},
      "segment2NewPosition": {"x": 0, "y": 0}
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

The swap exchanges the fragments' positions at execution time, and both fragments' cooldowns restart.

#### `PUZZLE_TO_PLAYER_RECOMMENDATION_EXPIRED`
**Direction**: Server → Both the original recommender and the target player
**Trigger**: The recommendation timed out (`expiresAt` passed without a response) or either involved player disconnected. Fragment moves do **not** expire pending recommendations. Sent to both parties so the target can clear the prompt UI and the recommender knows their suggestion is no longer pending. Timeout clocks pause while the game is paused by a host disconnect.

```json
{
  "event": "PUZZLE_TO_PLAYER_RECOMMENDATION_EXPIRED",
  "payload": {
    "moveId": "rec-uuid-67890",
    "reason": "timeout"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`reason` is `"timeout"` or `"player_disconnected"`.

### Puzzle Completion

#### `PUZZLE_TO_CLIENT_COMPLETED_SUCCESS`
**Direction**: Server → All Participants
**Trigger**: Puzzle completed successfully

```json
{
  "event": "PUZZLE_TO_CLIENT_COMPLETED_SUCCESS",
  "payload": {
    "success": true,
    "completionTime": 285,
    "totalTime": 360,
    "timeRemaining": 75,
    "finalGridState": {
      "allFragmentsCorrect": true,
      "totalFragments": 9,
      "correctFragments": 9
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT`
**Direction**: Server → All Participants
**Trigger**: Time expires before completion

```json
{
  "event": "PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT",
  "payload": {
    "success": false,
    "reason": "time_expired",
    "totalTime": 360,
    "timeExpired": true,
    "finalStats": {
      "fragmentsPlaced": 8,
      "totalFragments": 9,
      "correctlyPlaced": 6,
      "completionPercentage": 66.7
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `PUZZLE_TO_HOST_COMPLETION_ANALYTICS`
**Direction**: Server → Host
**Trigger**: Puzzle phase ends (success or timeout)

```json
{
  "event": "PUZZLE_TO_HOST_COMPLETION_ANALYTICS",
  "payload": {
    "puzzleSuccess": true,
    "completionTime": 285,
    "totalTime": 360,
    "playerContributions": {
      "player1-uuid": {
        "individualSolveTime": 180,
        "fragmentMoves": 8,
        "successfulMoves": 7,
        "recommendationsSent": 3,
        "recommendationsReceived": 2,
        "recommendationsAccepted": 1,
        "finalFragmentCorrect": true
      }
    },
    "collaborationMetrics": {
      "totalMoves": 32,
      "successfulMoves": 28,
      "totalRecommendations": 12,
      "acceptedRecommendations": 7,
      "averageResponseTime": 15.2
    },
    "phaseTransitions": {
      "playersCompletedIndividual": 5,
      "averageIndividualTime": 165,
      "fastestIndividual": 142,
      "slowestIndividual": 198
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

---

## Phase 3: Post-Game Analytics

### Analytics Distribution

#### `ANALYTICS_TO_PLAYER_PERSONAL_REPORT`
**Direction**: Server → Individual Player
**Trigger**: Game completion

```json
{
  "event": "ANALYTICS_TO_PLAYER_PERSONAL_REPORT",
  "payload": {
    "playerId": "player1-uuid",
    "playerName": "Alice",
    "gameSuccess": true,
    "personalScore": 196,
    "rank": 1,
    "totalPlayers": 5,
    "tokenCollection": {
      "anchorTokens": 40,
      "chronosTokens": 20,
      "guideTokens": 30,
      "clarityTokens": 40,
      "totalTokens": 130
    },
    "triviaPerformance": {
      "totalQuestions": 5,
      "correctAnswers": 4,
      "accuracy": 0.8,
      "accuracyByCategory": {
        "science": 1.0,
        "geography": 1.0,
        "general": 0.5
      },
      "specialtyPerformance": {
        "specialtyQuestions": 2,
        "specialtyCorrect": 2,
        "specialtyAccuracy": 1.0,
        "specialtyBonus": 40
      },
      "averageResponseTime": 18.5
    },
    "puzzleSolvingMetrics": {
      "individualSolveTime": 180,
      "individualRank": 2,
      "fragmentMoves": 8,
      "successfulMoves": 7,
      "moveAccuracy": 0.875,
      "recommendationsSent": 3,
      "recommendationsReceived": 2,
      "recommendationsAccepted": 1
    },
    "scoreBreakdown": {
      "triviaPoints": 40,
      "specialtyBonus": 4,
      "completionBonus": 100,
      "movePoints": 35,
      "recommendationPoints": 17,
      "totalScore": 196
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`scoreBreakdown` terms mirror the *Scoring Algorithm* in `game-design.md`; `recommendationPoints` combines the recommendations-sent and recommendations-accepted terms.

#### `ANALYTICS_TO_CLIENT_TEAM_SUMMARY`
**Direction**: Server → All Players
**Trigger**: Game completion

```json
{
  "event": "ANALYTICS_TO_CLIENT_TEAM_SUMMARY",
  "payload": {
    "gameSuccess": true,
    "totalScore": 830,
    "totalPlayers": 5,
    "totalGameTime": 795,
    "teamPerformance": {
      "overallAccuracy": 0.76,
      "totalTokensCollected": 280,
      "thresholdAchievements": {
        "anchor": 3,
        "chronos": 3,
        "guide": 3,
        "clarity": 2
      },
      "puzzleCompletionTime": 285
    },
    "leaderboard": [
      {
        "playerId": "player1-uuid",
        "playerName": "Alice",
        "totalScore": 196,
        "rank": 1,
        "role": "detective"
      },
      {
        "playerId": "player2-uuid",
        "playerName": "Bob",
        "totalScore": 178,
        "rank": 2,
        "role": "art_enthusiast"
      },
      {
        "playerId": "player3-uuid",
        "playerName": "Charlie",
        "totalScore": 165,
        "rank": 3,
        "role": "tourist"
      },
      {
        "playerId": "player4-uuid",
        "playerName": "Diana",
        "totalScore": 152,
        "rank": 4,
        "role": "janitor"
      },
      {
        "playerId": "player5-uuid",
        "playerName": "Eve",
        "totalScore": 139,
        "rank": 5,
        "role": "art_enthusiast"
      }
    ]
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

The leaderboard is sorted by `totalScore` descending. Players with equal scores share the same rank (competition ranking: 1, 2, 2, 4) and are ordered alphabetically by `playerName` within the tie; the same rule produces `rank` in `ANALYTICS_TO_PLAYER_PERSONAL_REPORT`.

#### `ANALYTICS_TO_HOST_COMPLETE_REPORT`
**Direction**: Server → Host
**Trigger**: Game completion

```json
{
  "event": "ANALYTICS_TO_HOST_COMPLETE_REPORT",
  "payload": {
    "gameId": "game-uuid-12345",
    "gameSuccess": true,
    "totalGameTime": 795,
    "totalPlayers": 5,
    "difficultyMode": "medium",
    "overallPerformance": {
      "totalScore": 830,
      "averageScore": 166,
      "completionRate": 1.0
    },
    "resourceGatheringAnalytics": {
      "totalRounds": 5,
      "questionsAnswered": 25,
      "overallAccuracy": 0.76,
      "tokenDistribution": {
        "anchorTokens": 90,
        "chronosTokens": 70,
        "guideTokens": 50,
        "clarityTokens": 70
      },
      "playerPerformance": {
        "player1-uuid": {
          "questionsAnswered": 5,
          "correctAnswers": 4,
          "accuracy": 0.8,
          "tokensEarned": 130,
          "averageResponseTime": 18.5,
          "specialtyPerformance": {
            "questionsReceived": 2,
            "correctAnswers": 2,
            "bonusTokens": 40
          },
          "stationPreferences": {
            "anchor": 1,
            "chronos": 1,
            "guide": 2,
            "clarity": 1
          }
        }
      }
    },
    "puzzleAssemblyAnalytics": {
      "totalTime": 360,
      "completionTime": 285,
      "timeUtilization": 0.79,
      "individualPhaseMetrics": {
        "averageSolveTime": 165,
        "fastestCompletion": 142,
        "slowestCompletion": 198,
        "preSolvedPiecesUsed": 30
      },
      "collaborativePhaseMetrics": {
        "totalMoves": 32,
        "successfulMoves": 28,
        "moveAccuracy": 0.875,
        "totalRecommendations": 12,
        "acceptedRecommendations": 7,
        "recommendationAcceptanceRate": 0.583
      },
      "playerContributions": {
        "player1-uuid": {
          "individualSolveTime": 180,
          "fragmentMoves": 8,
          "successfulMoves": 7,
          "recommendationsSent": 3,
          "recommendationsReceived": 2,
          "recommendationsAccepted": 1
        }
      }
    },
    "categoryPerformance": {
      "general": {
        "questionsAsked": 6,
        "correctAnswers": 4,
        "accuracy": 0.667
      },
      "science": {
        "questionsAsked": 5,
        "correctAnswers": 4,
        "accuracy": 0.8
      },
      "history": {
        "questionsAsked": 4,
        "correctAnswers": 3,
        "accuracy": 0.75
      },
      "geography": {
        "questionsAsked": 5,
        "correctAnswers": 3,
        "accuracy": 0.6
      },
      "music": {
        "questionsAsked": 3,
        "correctAnswers": 3,
        "accuracy": 1.0
      },
      "video_games": {
        "questionsAsked": 2,
        "correctAnswers": 2,
        "accuracy": 1.0
      }
    },
    "timelineAnalysis": {
      "setupPhase": 120,
      "resourcePhase": 360,
      "preparationPhase": 30,
      "puzzlePhase": 285
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`timelineAnalysis` values are phase durations in seconds; they sum to `totalGameTime`. `resourcePhase` includes the one-round wait before Round 1 (here 6 × 60s).

### Game Reset

#### `ANALYTICS_TO_SERVER_RESET_GAME`
**Direction**: Host → Server
**Trigger**: Host initiates new game

```json
{
  "event": "ANALYTICS_TO_SERVER_RESET_GAME",
  "auth": {
    "token": "host-uuid"
  },
  "payload": {
    "confirmReset": true
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `ANALYTICS_TO_CLIENT_GAME_RESET`
**Direction**: Server → All Participants
**Trigger**: Host resets game

```json
{
  "event": "ANALYTICS_TO_CLIENT_GAME_RESET",
  "payload": {
    "reason": "host_initiated_reset",
    "reconnectRequired": true,
    "reconnectInstructions": "Refresh your browser and reconnect to join the next game",
    "newGameAvailable": true
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

---

## System-Wide Events (Cross-Phase)

### Error Handling

The `errorCode` field in both error events draws from the *Error code registry* (Conventions section, near the top of this document). The `errorType` field is a coarser category for log filtering — currently `auth_error`, `validation_error`, or `game_state_error`. The `details`, `context`, and `suggestedActions` fields are **(optional)**.

#### `SYSTEM_TO_CLIENT_ERROR`
**Direction**: Server → Client
**Trigger**: Various error conditions

```json
{
  "event": "SYSTEM_TO_CLIENT_ERROR",
  "payload": {
    "errorType": "validation_error",
    "errorCode": "INVALID_ROLE_SELECTION",
    "message": "Role selection validation failed",
    "details": "Selected role 'invalid_role' is not available",
    "context": {
      "requestedAction": "role_selection",
      "currentPhase": "setup",
      "playerId": "player-uuid"
    },
    "suggestedActions": [
      "Select from available roles: art_enthusiast, detective, tourist, janitor",
      "Refresh the page if role list appears outdated"
    ]
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

#### `SYSTEM_TO_HOST_ERROR`
**Direction**: Server → Host
**Trigger**: Host-specific errors

```json
{
  "event": "SYSTEM_TO_HOST_ERROR",
  "payload": {
    "errorType": "game_state_error",
    "errorCode": "INSUFFICIENT_PLAYERS",
    "message": "Cannot start game with insufficient players",
    "details": "Need at least 4 players, currently have 3",
    "context": {
      "requestedAction": "start_game",
      "currentPlayers": 3,
      "requiredPlayers": 4,
      "connectedPlayers": 3,
      "readyPlayers": 3
    },
    "suggestedActions": [
      "Wait for more players to join",
      "Verify all players are properly connected"
    ]
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

### Connection Management

#### `SYSTEM_TO_CLIENT_HOST_DISCONNECTED`
**Direction**: Server → All Players
**Trigger**: Host disconnects

```json
{
  "event": "SYSTEM_TO_CLIENT_HOST_DISCONNECTED",
  "payload": {
    "hostStatus": "disconnected",
    "currentPhase": "puzzle_assembly",
    "gameImpact": {
      "canContinue": false,
      "affectedFeatures": ["host_monitoring", "phase_transitions", "puzzle_timer"]
    },
    "timerPausedAt": "2025-06-15T14:25:30.000Z"
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`gameImpact.canContinue` is phase-aware — see the host-disconnect rules in `game-design.md` § *Disconnections and Reconnection*. It is `false` only during `puzzle_assembly`, where the timer pauses and the server rejects all puzzle actions (segment completions, fragment moves, recommendation creation and responses) until the host reconnects. In every other phase it is `true`: setup activity and trivia rounds proceed, but host-gated transitions (game start, puzzle start) remain blocked.

`gameImpact.affectedFeatures` includes `"puzzle_timer"` only during Puzzle Assembly. `timerPausedAt` is **(optional)** — present only when the puzzle timer pauses (Puzzle Assembly host disconnect). The value is the server timestamp at which the timer was frozen; clients use it to display "paused at N seconds remaining" without drift.

#### `SYSTEM_TO_CLIENT_HOST_RECONNECTED`
**Direction**: Server → All Players
**Trigger**: Host reconnects

```json
{
  "event": "SYSTEM_TO_CLIENT_HOST_RECONNECTED",
  "payload": {
    "hostStatus": "reconnected",
    "currentPhase": "puzzle_assembly",
    "restoredFeatures": ["host_monitoring", "phase_transitions", "puzzle_timer"],
    "timeRemaining": 215
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`timeRemaining` is **(optional)** — present only when the puzzle timer resumes after a Puzzle Assembly host disconnect. It carries the new authoritative seconds-remaining value (deadline pushed forward by the disconnect duration). Clients should re-anchor their countdown to this value rather than to the original `PUZZLE_TO_CLIENT_PHASE_START.totalTime`.

#### `SYSTEM_TO_HOST_PLAYER_DISCONNECTED`
**Direction**: Server → Host
**Trigger**: Player disconnects

**Setup Phase Example:**
```json
{
  "event": "SYSTEM_TO_HOST_PLAYER_DISCONNECTED",
  "payload": {
    "playerId": "player3-uuid",
    "playerName": "Charlie",
    "disconnectionTime": "2025-06-15T14:23:05.000Z",
    "currentPhase": "setup",
    "updatedCounts": {
      "connectedPlayers": 4,
      "readyPlayers": 3,
      "roleDistribution": {
        "art_enthusiast": 1,
        "detective": 1,
        "tourist": 0,
        "janitor": 1
      }
    }
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

**Puzzle Assembly Phase Example:**
```json
{
  "event": "SYSTEM_TO_HOST_PLAYER_DISCONNECTED",
  "payload": {
    "playerId": "player3-uuid",
    "playerName": "Charlie",
    "disconnectionTime": "2025-06-15T14:23:05.000Z",
    "currentPhase": "puzzle_assembly",
    "fragmentHandling": {
      "segmentId": "segment_a3",
      "newPosition": {"x": 1, "y": 2},
      "nowUnassigned": true
    },
    "updatedPlayerCount": 4
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

**Resource Gathering/Analytics Phase Example:**
```json
{
  "event": "SYSTEM_TO_HOST_PLAYER_DISCONNECTED",
  "payload": {
    "playerId": "player3-uuid",
    "playerName": "Charlie",
    "disconnectionTime": "2025-06-15T14:23:05.000Z",
    "currentPhase": "resource_gathering",
    "updatedPlayerCount": 4
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

### Heartbeat and Health

#### `SYSTEM_PING`
**Direction**: Client → Server
**Trigger**: Periodic heartbeat (every 30 seconds)

```json
{
  "event": "SYSTEM_PING",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "clientTimestamp": "2025-06-15T14:23:05.000Z",
    "sequenceNumber": 42
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

Heartbeats use the standard client wrapper, `auth` included (the host sends its own UUID). Clients start pinging only after their handshake completes.

#### `SYSTEM_PONG`
**Direction**: Server → Client
**Trigger**: Response to ping

```json
{
  "event": "SYSTEM_PONG",
  "payload": {
    "serverTimestamp": "2025-06-15T14:23:05.000Z",
    "clientTimestamp": "2025-06-15T14:23:05.000Z",
    "sequenceNumber": 42
  },
  "timestamp": "2025-06-15T14:23:05.000Z"
}
```

`clientTimestamp` echoes the ping's value so the client can compute round-trip time itself.

The server treats a client as disconnected when its WebSocket closes or when no `SYSTEM_PING` has arrived for 90 seconds (three missed heartbeats); disconnect handling then follows `game-design.md` § *Disconnections and Reconnection*.
