import { useEffect, useRef, useState } from "react";
import type { Order } from "../../types";
import { scopedKey } from "../../utils/scopedStorage";

interface Draft {
  savedAt: string;
  order: Partial<Order>;
}
export function useOrderDraft(
  active: boolean,
  order: Partial<Order>,
) {
  const key = scopedKey(`order-draft:${order.id || "new"}`);
  const [available, setAvailable] = useState<Draft | null>(null),
    [message, setMessage] = useState("");
  const dirty = useRef(false),
    changeKey = useRef<string | null>(null),
    session = useRef<{
      key: string;
      order: Partial<Order>;
      dirty: boolean;
    } | null>(null);
  const read = () => {
    try {
      const draft = key
        ? (JSON.parse(localStorage.getItem(key) || "null") as Draft | null)
        : null;
      return draft?.order &&
        Date.now() - Date.parse(draft.savedAt) >= 0 &&
        Date.now() - Date.parse(draft.savedAt) < 7 * 86400000
        ? draft
        : null;
    } catch {
      return null;
    }
  };
  const forget = () => {
    if (key)
      try {
        localStorage.removeItem(key);
      } catch {}
    setAvailable(null);
    setMessage("");
    dirty.current = false;
    changeKey.current = null;
    if (session.current) session.current.dirty = false;
  };
  const flush = (captured = session.current) => {
    if (!captured?.dirty) return;
    try {
      const value = JSON.stringify(
        { savedAt: new Date().toISOString(), order: captured.order },
        (name, value) => (name === "drawing_data" ? undefined : value),
      );
      if (value.length > 1500000) throw Error("草稿过大");
      localStorage.setItem(captured.key, value);
      if (session.current === captured)
        setMessage(
          "草稿已保存到当前浏览器，尚未提交业务记录（附件需重新选择）",
        );
    } catch {
      setMessage("浏览器无法保存草稿，请及时保存订单");
    }
  };
  useEffect(() => {
    const previous = session.current;
    if (!active || previous?.key !== key) {
      if (previous) {
        flush(previous);
      }
      session.current =
        active && key
          ? {
              key,
              order,
              dirty: !!previous && dirty.current && changeKey.current === key,
            }
          : null;
      dirty.current = false;
      changeKey.current = null;
      setAvailable(active ? read() : null);
      setMessage("");
    } else if (previous) {
      previous.order = order;
      previous.dirty ||= dirty.current && changeKey.current === key;
    }
    const captured = session.current;
    const timer = setTimeout(() => flush(captured), 400);
    return () => clearTimeout(timer);
  }, [active, key, order]);
  useEffect(() => {
    if (typeof window === "undefined") return;
    const leave = () => flush();
    window.addEventListener?.("beforeunload", leave);
    return () => {
      flush();
      window.removeEventListener?.("beforeunload", leave);
    };
  }, []);
  return {
    available,
    message,
    markChanged: (nextOrder: Partial<Order> = order) => {
      dirty.current = true;
      changeKey.current = scopedKey(`order-draft:${nextOrder.id || "new"}`);
    },
    forget,
    // The parent invokes this callback only after the server accepts the order,
    // before conditionally unmounting the modal. Failed requests retain the draft.
    captureSubmission: () => {
      const captured = session.current;
      const submittedKey = key;
      return () => {
        if (captured) captured.dirty = false;
        if (session.current === captured) {
          dirty.current = false;
          changeKey.current = null;
        }
        if (submittedKey) {
          try {
            localStorage.removeItem(submittedKey);
          } catch {}
        }
      };
    },
    restore: () => {
      const draft = read();
      setAvailable(null);
      dirty.current = true;
      changeKey.current = key;
      return draft?.order;
    },
  };
}
