import { useEffect, useRef } from "react";
// A stale form is hidden immediately when the account/company changes, before effects reset it.
export function useAccountFormBoundary(
  scope: string | null,
  reset: () => void,
) {
  const owner = useRef(scope),
    resetRef = useRef(reset);
  resetRef.current = reset;
  useEffect(() => {
    if (owner.current !== scope) {
      owner.current = scope;
      resetRef.current();
    }
  }, [scope]);
  return scope !== null && owner.current === scope;
}
