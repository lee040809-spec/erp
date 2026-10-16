import { useEffect, useRef } from "react";

export function useDraftGuard(
  active: boolean,
  session: string | number,
  saving: boolean,
) {
  const dirty = useRef(false);
  useEffect(() => {
    dirty.current = false;
  }, [active, session]);
  return {
    markChanged: () => {
      dirty.current = true;
    },
    reset: () => {
      dirty.current = false;
    },
    confirmDiscard: () =>
      !saving &&
      (!dirty.current ||
        window.confirm("还有未保存的填写内容，确定放弃这些修改吗？")),
  };
}
