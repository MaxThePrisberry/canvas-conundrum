// Authenticated asset fetching: tiles and the clarity preview come from
// /api with a Bearer token (websocket-events.md § Asset Delivery). Responses
// are cached as object URLs per (token, path); the cache resets with the
// page, matching the tiles' per-game lifecycle.

const cache = new Map<string, Promise<string>>();

export function fetchAssetURL(path: string, token: string): Promise<string> {
  const key = `${token}:${path}`;
  const hit = cache.get(key);
  if (hit) return hit;

  const promise = fetch(path, { headers: { Authorization: `Bearer ${token}` } }).then(
    async (resp) => {
      if (!resp.ok) {
        cache.delete(key);
        const body = (await resp.json().catch(() => null)) as { error?: string } | null;
        throw new Error(body?.error ?? `HTTP ${resp.status}`);
      }
      return URL.createObjectURL(await resp.blob());
    },
  );
  cache.set(key, promise);
  return promise;
}

export function segmentURL(segmentId: string, token: string): Promise<string> {
  return fetchAssetURL(`/api/segments/${segmentId}`, token);
}

export function previewURL(token: string): Promise<string> {
  // Deliberately uncached: the window is short and server-gated.
  return fetch('/api/preview/full', {
    headers: { Authorization: `Bearer ${token}` },
  }).then(async (resp) => {
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    return URL.createObjectURL(await resp.blob());
  });
}
