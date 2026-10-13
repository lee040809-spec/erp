import { useState } from "react";
import { drawingRequest, PartDrawing, referenceDrawing } from "./api";
export default function DrawingHistoryPicker({
  customerId,
  itemId,
  reason,
  disabled,
  onBusy,
  onAdded,
  onPreview,
  onError,
}: {
  customerId?: number;
  itemId?: number;
  reason: string;
  disabled: boolean;
  onBusy: (busy: boolean) => void;
  onAdded: (drawing: PartDrawing) => Promise<void>;
  onPreview: (drawing: PartDrawing) => void;
  onError: (error: string) => void;
}) {
  const [q, setQ] = useState(""),
    [results, setResults] = useState<PartDrawing[]>([]),
    [searched, setSearched] = useState(false);
  const search = async () => {
    onError("");
    onBusy(true);
    try {
      setResults(
        await drawingRequest<PartDrawing[]>(
          `drawings/search?customer_id=${customerId}&q=${encodeURIComponent(q.trim())}`,
        ),
      );
      setSearched(true);
    } catch (e) {
      onError(e instanceof Error ? e.message : "搜索失败");
    } finally {
      onBusy(false);
    }
  };
  const select = async (d: PartDrawing) => {
    if (
      !window.confirm(
        `请核对客户、图号和版本。\n客户：${d.customer_name}\n零件：${d.part_name}\n图号：${d.part_number || d.drawing_number || "未填写"}\n版本：${d.version || "未填写"}${!d.is_current ? "（历史版本）" : ""}\n引用后作为此零件的当前生产图纸，继续吗？`,
      )
    )
      return;
    onError("");
    onBusy(true);
    try {
      await onAdded(await referenceDrawing(d.id, itemId, reason));
      setResults([]);
      setSearched(false);
    } catch (e) {
      onError(e instanceof Error ? e.message : "引用失败");
    } finally {
      onBusy(false);
    }
  };
  return (
    <details className="erp-card p-3">
      <summary className="cursor-pointer text-sm">
        引用相同客户的历史图纸
      </summary>
      <div className="flex gap-2 mt-3">
        <input
          className="erp-input"
          aria-label="搜索历史图纸"
          placeholder="零件名称、图号或文件名"
          disabled={disabled || !customerId}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              if (!disabled && customerId && q.trim().length >= 2) void search();
            }
          }}
        />
        <button
          type="button"
          className="erp-btn"
          disabled={disabled || !customerId || q.trim().length < 2}
          onClick={() => void search()}
        >
          查询
        </button>
      </div>
      {!customerId && (
        <p className="text-xs text-amber-700 mt-2">请先在订单中选择客户。</p>
      )}
      {searched && !results.length && (
        <p className="text-sm text-slate-500 mt-2">没有找到匹配的历史图纸。</p>
      )}
      {results.map((d) => (
        <div key={d.id} className="drawing-row">
          <div>
            <strong>{d.file_name}</strong>
            <small>
              {d.customer_name} · {d.part_name} · {d.order_number}
            </small>
            <small>
              图号 {d.part_number || d.drawing_number || "未填"} · 版本{" "}
              {d.version || "未填"} · {d.is_current ? "当前" : "历史"}
            </small>
          </div>
          <div className="drawing-row-actions">
            <button
              type="button"
              className="erp-btn"
              disabled={disabled}
              onClick={() => onPreview(d)}
            >
              预览
            </button>
            <button
              type="button"
              className="erp-btn"
              disabled={disabled}
              onClick={() => void select(d)}
            >
              引用此图纸
            </button>
          </div>
        </div>
      ))}
    </details>
  );
}
