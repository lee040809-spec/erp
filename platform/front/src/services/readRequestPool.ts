// Share only simultaneous identical GETs; completed data is never cached.
export function createReadRequestPool(fetcher: typeof fetch = (input, init) => fetch(input, init)) {
  const pending = new Map<string, Promise<Response>>();
  return async (url: string, options: RequestInit = {}): Promise<Response> => {
    const method = (options.method || 'GET').toUpperCase();
    if (method !== 'GET' && method !== 'HEAD') {
      pending.clear();
      try { return await fetcher(url, options); } finally { pending.clear(); }
    }
    if (method !== 'GET' || !url.startsWith('/api/platform/') ||
        Object.keys(options).some(key => key !== 'method' && key !== 'headers')) {
      return fetcher(url, options);
    }
    const headers = [...new Headers(options.headers).entries()].sort(([a], [b]) => a.localeCompare(b));
    const key = JSON.stringify([url, headers]);
    let request = pending.get(key);
    if (!request) {
      request = fetcher(url, options).finally(() => {
        if (pending.get(key) === request) pending.delete(key);
      });
      pending.set(key, request);
    }
    return (await request).clone();
  };
}
export const fetchReadRequest = createReadRequestPool();
