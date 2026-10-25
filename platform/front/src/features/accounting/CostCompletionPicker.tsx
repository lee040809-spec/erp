import { useCallback, useState } from "react";
import { fetchCompletionsApi } from "../../services/api";
import { useAsyncResource } from "../../hooks/useAsyncResource";
import type { CompletionListResponse } from "../../types";
import { processDate } from "../../utils/processDates";

export default function CostCompletionPicker({
  itemId,
  value,
  onChange,
}: {
  itemId: number;
  value: string;
  onChange: (id: string, month?: string) => void;
}) {
  const [page, setPage] = useState(1);
  const request = useCallback(
    () => fetchCompletionsApi(itemId, page),
    [itemId, page],
  );
  const { data, loading, error, reload } = useAsyncResource(
    request,
    null as CompletionListResponse | null,
    true,
  );
  return (
    <div className="rounded-xl border border-blue-200 bg-blue-50 p-3 space-y-2 text-sm">
      <label className="erp-label">
        保存时同时计入该零件（可选）
        <select
          aria-label="费用计入完工批次"
          className="erp-input"
          disabled={loading || !!error}
          value={value}
          onChange={(e) => {
            const row = data?.records.find(
              (r) => String(r.id) === e.target.value,
            );
            onChange(
              e.target.value,
              row ? processDate(row.business_date).slice(0, 7) : undefined,
            );
          }}
        >
          <option value="">仅登记费用，之后分配</option>
          {data?.records
            .filter((r) => r.status === "confirmed")
            .map((r) => (
              <option key={r.id} value={r.id}>
                {processDate(r.business_date)} · 合格 {r.qualified_quantity} 件
                · {r.completion_no}
              </option>
            ))}
        </select>
      </label>
      <div className="flex gap-2 items-center">
        <button
          type="button"
          className="erp-btn"
          disabled={page === 1 || loading}
          onClick={() => {
            onChange("");
            setPage(page - 1);
          }}
        >
          较新批次
        </button>
        <span>第 {page} 页</span>
        <button
          type="button"
          className="erp-btn"
          disabled={!data?.has_more || loading}
          onClick={() => {
            onChange("");
            setPage(page + 1);
          }}
        >
          较早批次
        </button>
      </div>
      <p>
        选中后只保存一次费用，同时计入所选批次的完工月份。未确认合格或费用涉及多个零件时，可先登记再分配。
      </p>
      {error && (
        <p role="alert">
          {error}
          <button type="button" onClick={reload} className="erp-btn">
            重试
          </button>
        </p>
      )}
    </div>
  );
}
