import { lazy, Suspense, useEffect, useState } from "react";
import { FileImage } from "lucide-react";
import type { PartDrawing } from "./api";
const DrawingManager = lazy(() => import("./DrawingManager"));
export interface DrawingButtonProps {
  itemId?: number;
  partName: string;
  partNumber?: string;
  customerId?: number;
  count?: number;
  drafts?: PartDrawing[];
  onDraftChange?: (drawings: PartDrawing[]) => void;
  disabled?: boolean;
  onChanged?: () => void;
  compact?: boolean;
}
export default function DrawingButton(props: DrawingButtonProps) {
  const [open, setOpen] = useState(false),
    [count, setCount] = useState(props.count);
  useEffect(() => setCount(props.count), [props.count]);
  return (
    <>
      <button
        type="button"
        disabled={props.disabled}
        className={
          props.compact
            ? "erp-gantt-view-part inline-flex items-center gap-1 whitespace-nowrap"
            : "erp-btn py-1 min-h-8"
        }
        aria-label={`图纸${props.partName || "新零件"}`}
        onClick={() => setOpen(true)}
      >
        <FileImage size={14} className="shrink-0" />
        {props.itemId ? "图纸" : "上传图纸"}
        {(count || props.drafts?.length || 0) > 0
          ? ` ${count || props.drafts?.length}`
          : ""}
      </button>
      {open && (
        <Suspense fallback={<span role="status">读取图纸模块…</span>}>
          <DrawingManager
            {...props}
            onCountChange={setCount}
            onClose={() => setOpen(false)}
          />
        </Suspense>
      )}
    </>
  );
}
