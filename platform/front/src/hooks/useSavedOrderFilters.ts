import { useCallback, useEffect, useRef, useState } from "react";
import type { Dispatch, SetStateAction } from "react";
import type { OrderFilters } from "../services/orderService";
import { scopedKey } from "../utils/scopedStorage";

// Keep draft search inputs and applied queries separate, and capture the storage key
// so an account change cannot write the previous account's filters into the new one.
export function useSavedOrderFilters<T extends OrderFilters>(
  name: string,
  initial: () => T,
): [T, Dispatch<SetStateAction<T>>] {
  const defaults = useRef<T | null>(null);
  if (!defaults.current) defaults.current = initial();
  const key = scopedKey("filters:" + name);
  const read = () => {
    const value = { ...defaults.current! };
    try {
      const saved = key
        ? JSON.parse(localStorage.getItem(key) || "null")
        : null;
      if (saved && typeof saved === "object")
        for (const field of Object.keys(value) as (keyof T)[]) {
          const text = saved[field];
          if (typeof text === "string" && text.length <= 200)
            value[field] = text as T[keyof T];
        }
    } catch {
      /* Browser preferences are optional. */
    }
    return value;
  };
  const [state, setState] = useState(() => ({ key, value: read() }));
  const value = state.key === key ? state.value : read();
  useEffect(() => {
    if (state.key !== key) setState({ key, value: read() });
  }, [key, state.key]);
  useEffect(() => {
    if (key && state.key === key)
      try {
        localStorage.setItem(key, JSON.stringify(state.value));
      } catch {}
  }, [key, state]);
  const setValue = useCallback<Dispatch<SetStateAction<T>>>(
    (next) =>
      setState((previous) => {
        const before = previous.key === key ? previous.value : read();
        return {
          key,
          value:
            typeof next === "function"
              ? (next as (value: T) => T)(before)
              : next,
        };
      }),
    [key],
  );
  return [value, setValue];
}
