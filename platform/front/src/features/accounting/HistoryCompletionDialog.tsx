import type { FormEvent } from "react";
import type { Candidate, HistoryCompletionForm } from "./financeTypes";
export default function HistoryCompletionDialog({
  selected,
  form,
  setForm,
  saving,
  error,
  policy,
  onSubmit,
  onClose,
  canSave,
}: {
  selected: Candidate;
  form: HistoryCompletionForm;
  setForm: (form: HistoryCompletionForm) => void;
  saving: boolean;
  error: string;
  policy: { source_ref: boolean; review_notes: boolean };
  onSubmit: (event: FormEvent) => void | Promise<void>;
  onClose: () => void;
  canSave: boolean;
}) {
  return (
    <>
      {" "}
      {selected && (
        <div
          data-escape-dialog
          data-escape-blocked={saving}
          className="erp-overlay z-[110]"
        >
          <form
            role="dialog"
            aria-label="补录历史完工"
            onSubmit={onSubmit}
            className="erp-dialog p-6 w-full max-w-xl space-y-4 max-h-[90vh] overflow-y-auto"
          >
            <h3 className="text-lg font-bold">
              补录历史完工：{selected.part_name}
            </h3>
            <p className="text-sm text-zinc-600">
              {selected.order_number}；订单 {selected.quantity} 件，投产{" "}
              {selected.production_quantity} 件，报废 {selected.scrap_quantity}{" "}
              件；当前订单单价 {selected.unit_price} 元。已审核{" "}
              {selected.confirmed_quantity} 件，剩余{" "}
              {selected.remaining_quantity}{" "}
              件。每次补录一批，核对实际数量、日期与单价即可；凭证信息选填，订单当前单价保持不变。
            </p>
            <div className="grid sm:grid-cols-2 gap-4">
              <label className="erp-label">
                实际合格数量 *
                <input
                  required
                  type="number"
                  min="1"
                  max={selected.remaining_quantity}
                  step="1"
                  value={form.quantity}
                  onChange={(e) =>
                    setForm({ ...form, quantity: e.target.value })
                  }
                  className="erp-input"
                />
              </label>
              <label className="erp-label">
                实际合格完工日期 *
                <input
                  required
                  type="date"
                  value={form.date}
                  onChange={(e) => setForm({ ...form, date: e.target.value })}
                  className="erp-input"
                />
              </label>
            </div>
            <label className="erp-label">
              本批确认单价 *
              <input
                aria-label="本批确认单价"
                required
                inputMode="decimal"
                pattern="\d{1,11}(\.\d{1,4})?"
                value={form.price}
                onChange={(e) => setForm({ ...form, price: e.target.value })}
                className="erp-input"
              />
              <span className="text-xs text-slate-500">
                默认使用订单当前单价，仅用于本次完工，不修改订单单价。
              </span>
            </label>
            <details
              open={policy.source_ref || policy.review_notes}
              className="border rounded-xl p-3 space-y-3"
            >
              <summary className="text-sm text-slate-500">
                凭证与补充信息（默认选填，按角色要求）
              </summary>
              <label className="erp-label">
                原始批次依据编号（{policy.source_ref ? "必填" : "选填"}）
                <input
                  required={policy.source_ref}
                  maxLength={80}
                  placeholder="留空由系统生成批次编号"
                  value={form.sourceRef}
                  onChange={(e) =>
                    setForm({ ...form, sourceRef: e.target.value })
                  }
                  className="erp-input"
                />
              </label>
              <label className="erp-label">
                审核依据（{policy.review_notes ? "必填" : "选填"}）
                <textarea
                  required={policy.review_notes}
                  maxLength={400}
                  value={form.notes}
                  onChange={(e) => setForm({ ...form, notes: e.target.value })}
                  className="erp-input"
                />
              </label>
            </details>
            <p className="text-xs text-slate-500">
              请确认数量、日期与单价准确。超出订单数量的合格件作为备用件，不计产值；点击确认即可完成补录。
            </p>
            {error && (
              <p role="alert" className="text-rose-700">
                {error}
              </p>
            )}
            <div className="flex justify-end gap-3">
              <button
                data-escape-close
                type="button"
                disabled={saving}
                onClick={onClose}
                className="border px-4 py-2"
              >
                取消
              </button>
              <button
                disabled={saving || !canSave}
                className="bg-zinc-900 text-white px-4 py-2 disabled:opacity-50"
              >
                {saving ? "正在导入…" : "确认计入产值"}
              </button>
            </div>
          </form>
        </div>
      )}
    </>
  );
}
