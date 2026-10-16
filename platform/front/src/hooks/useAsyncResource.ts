import { useCallback, useEffect, useRef, useState } from "react";

// Preserve the last successful read; discard obsolete results and report refresh failures.
export function useAsyncResource<T>(
  request: () => Promise<T>,
  initial: T,
  enabled = true,
) {
  const [data, setData] = useState(initial);
  const [loading, setLoading] = useState(enabled);
  const [error, setError] = useState("");
  const active = useRef(false),
    generation = useRef(0);
  const reload = useCallback(async () => {
    if (!enabled) return true;
    const version = ++generation.current;
    setLoading(true);
    setError("");
    try {
      const result = await request();
      if (active.current && version === generation.current) {
        setData(result);
        return true;
      }
    } catch (error) {
      if (active.current && version === generation.current)
        setError(
          error instanceof Error ? error.message : "读取失败，请稍后重试",
        );
    } finally {
      if (active.current && version === generation.current) setLoading(false);
    }
    return false;
  }, [request, enabled]);
  useEffect(() => {
    active.current = true;
    if (enabled) void reload();
    else setLoading(false);
    return () => {
      active.current = false;
      generation.current++;
    };
  }, [reload, enabled]);
  return { data, loading, error, reload };
}
