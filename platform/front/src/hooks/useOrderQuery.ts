import { useCallback, useEffect, useRef, useState } from 'react';

// Report failed reads without losing the last successful list or rejecting UI handlers.
export function useOrderQuery() {
  const [isSearching, setIsSearching] = useState(false);
  const [queryError, setQueryError] = useState('');
  const generation = useRef(0);
  const active = useRef(true);
  const lastRequest = useRef<(() => Promise<void>) | null>(null);
  useEffect(() => {
    active.current = true;
    return () => { active.current = false; generation.current++; };
  }, []);
  const run = useCallback(async (request: () => Promise<void>) => {
    lastRequest.current = request;
    const current = ++generation.current;
    setIsSearching(true);
    setQueryError('');
    try {
      await request();
    } catch (error) {
      if (active.current && current === generation.current) {
        setQueryError(error instanceof Error ? error.message : '订单读取失败，请稍后重试');
      }
    } finally {
      if (active.current && current === generation.current) setIsSearching(false);
    }
  }, []);
  const retry = useCallback(() => { if (lastRequest.current) void run(lastRequest.current); }, [run]);
  return { isSearching, queryError, run, retry };
}
