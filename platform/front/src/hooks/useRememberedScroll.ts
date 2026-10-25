import { useEffect, useRef } from "react";
import { scopedKey } from "../utils/scopedStorage";
// Restore only after data is present; focus-to-order navigation can opt out.
export function useRememberedScroll(
  name: string,
  ready: boolean,
  enabled = true,
) {
  const ref = useRef<HTMLDivElement>(null),
    restored = useRef<string | null>(null),
    last = useRef(0);
  const key = scopedKey(name);
  useEffect(() => {
    const element = ref.current;
    if (!element || !ready || !enabled || !key) return;
    // Capture the owner before timers and cleanup: neither may use a later login.
    const write = () => {
      try {
        localStorage.setItem(key, JSON.stringify(last.current));
      } catch {}
    };
    if (restored.current !== key) {
      let value = 0;
      try {
        value = JSON.parse(localStorage.getItem(key) || "0");
      } catch {}
      if (Number.isFinite(value) && value >= 0) element.scrollTop = value;
      last.current = element.scrollTop;
      restored.current = key;
    }
    let timer: ReturnType<typeof setTimeout>;
    const save = () => {
      last.current = element.scrollTop;
      clearTimeout(timer);
      timer = setTimeout(write, 150);
    };
    element.addEventListener("scroll", save, { passive: true });
    return () => {
      clearTimeout(timer);
      if (restored.current === key) write();
      element.removeEventListener("scroll", save);
    };
  }, [key, ready, enabled]);
  return ref;
}
