import { businessToday } from "../utils/businessDate";
import CostOrderPicker from "../features/accounting/CostOrderPicker";
import React, { useRef, useState } from "react";
import { AlertCircle, Loader2, X } from "lucide-react";
import { createCostEntryApi } from "../services/api";

interface CostEntryModalProps {
  onClose: () => void;
  onSuccess: (costType: string) => void;
}

const costTypes = [
  ["material", "生产使用的材料"],
  ["outsourcing", "外协费用"],
  ["tool", "刀具费用"],
  ["fixture", "工装费用"],
  ["labor", "生产人工费用"],
  ["manufacturing", "水电等生产费用"],
  ["other_direct", "其他生产费用"],
  ["period", "其他经营费用"],
];

const CostEntryModal: React.FC<CostEntryModalProps> = ({
  onClose,
  onSuccess,
}) => {
  const [form, setForm] = useState({
    cost_type: "material",
    amount: "",
    tax_amount: "0.00",
    net_amount: "",
    business_date: businessToday(),
    order_id: "",
    order_item_id: "",
    supplier_name: "",
    notes: "",
  });
  const [saving, setSaving] = useState(false);
  const busy = useRef(false);
  const [error, setError] = useState("");
  const [taxInCost, setTaxInCost] = useState(true);
  const accountingAmount = taxInCost
    ? form.amount
    : (
        (Math.round(Number(form.amount) * 100) -
          Math.round(Number(form.tax_amount || 0) * 100)) /
        100
      ).toFixed(2);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (busy.current) return;
    if (
      !Number.isFinite(Number(accountingAmount)) ||
      Number(accountingAmount) <= 0 ||
      Number(form.tax_amount || 0) > Number(form.amount)
    ) {
      setError("请填写大于零的费用金额，税额不能超过费用总金额。");
      return;
    }
    busy.current = true;
    setSaving(true);
    setError("");
    try {
      await createCostEntryApi({
        cost_type: form.cost_type,
        amount: form.amount,
        tax_amount: form.tax_amount || "0.00",
        net_amount: accountingAmount,
        business_date: form.business_date,
        order_id: form.order_id ? Number(form.order_id) : undefined,
        order_item_id: form.order_item_id
          ? Number(form.order_item_id)
          : undefined,
        supplier_name: form.supplier_name.trim() || undefined,
        notes: form.notes.trim() || undefined,
      });
      onSuccess(form.cost_type);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "成本费用保存失败");
    } finally {
      busy.current = false;
      setSaving(false);
    }
  };

  const set = (key: keyof typeof form, value: string) =>
    setForm((current) => ({ ...current, [key]: value }));

  return (
    <div
      data-escape-dialog
      data-escape-blocked={saving}
      className="fixed inset-0 z-50 bg-black/40 flex items-center justify-center p-4"
      onMouseDown={(event) =>
        !saving && event.target === event.currentTarget && onClose()
      }
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="记一笔费用"
        className="bg-white w-full max-w-2xl max-h-[92vh] overflow-auto rounded-xl border border-zinc-200 shadow-2xl"
      >
        <div className="px-5 py-4 border-b border-zinc-200 flex justify-between items-start">
          <div>
            <h3 className="font-bold text-lg">记一笔费用</h3>
            <p className="text-sm text-zinc-500 mt-1">
              记录本次实际用于生产或经营的费用；仅采购入库、还没有使用的材料不要重复计入。
            </p>
          </div>
          <button
            data-escape-close
            type="button"
            aria-label="关闭费用登记"
            disabled={saving}
            onClick={onClose}
            className="p-1 text-zinc-400 hover:text-zinc-900"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={submit} className="p-5">
          <fieldset disabled={saving} className="space-y-4">
            {error && (
              <div className="bg-rose-50 border border-rose-200 text-rose-700 px-3 py-2 text-sm flex gap-2">
                <AlertCircle className="w-4 h-4 mt-0.5" />
                {error}
              </div>
            )}
            <div className="grid sm:grid-cols-2 gap-4">
              <label>
                <span className="text-sm font-medium">费用类型</span>
                <select
                  value={form.cost_type}
                  onChange={(e) => set("cost_type", e.target.value)}
                  className="mt-1 w-full h-10 border border-zinc-300 bg-white px-3"
                >
                  {costTypes.map(([value, label]) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                <span className="text-sm font-medium">费用发生日期</span>
                <input
                  type="date"
                  required
                  value={form.business_date}
                  onChange={(e) => set("business_date", e.target.value)}
                  className="mt-1 w-full h-10 border border-zinc-300 px-3"
                />
              </label>
              <label>
                <span className="text-sm font-medium">费用总金额</span>
                <input
                  required
                  inputMode="decimal"
                  pattern="\d+(\.\d{1,2})?"
                  value={form.amount}
                  onChange={(e) => set("amount", e.target.value)}
                  placeholder="0.00"
                  className="mt-1 w-full h-10 border border-zinc-300 px-3"
                />
              </label>
              <label>
                <span className="text-sm font-medium">供应商／收款方</span>
                <input
                  value={form.supplier_name}
                  onChange={(e) => set("supplier_name", e.target.value)}
                  className="mt-1 w-full h-10 border border-zinc-300 px-3"
                />
              </label>
            </div>
            <CostOrderPicker
              onPick={(order_id, order_item_id) =>
                setForm((current) => ({ ...current, order_id, order_item_id }))
              }
            />
            <details className="rounded-xl border p-3">
              <summary className="cursor-pointer text-sm">
                税额处理（需要单独处理税额时展开）
              </summary>
              <p className="text-xs text-slate-500 my-2">
                默认将费用总金额全部计入。只有明确确认税额不计入成本时才调整。
              </p>
              <div className="grid sm:grid-cols-2 gap-3">
                {" "}
                <label>
                  <span className="text-sm font-medium">税额</span>
                  <input
                    inputMode="decimal"
                    pattern="\d+(\.\d{1,2})?"
                    value={form.tax_amount}
                    onChange={(e) => set("tax_amount", e.target.value)}
                    className="mt-1 w-full h-10 border border-zinc-300 px-3"
                  />
                </label>
                <label>
                  <span className="text-sm font-medium">税额核算方式</span>
                  <select
                    value={taxInCost ? "included" : "excluded"}
                    onChange={(e) =>
                      setTaxInCost(e.target.value === "included")
                    }
                    className="mt-1 w-full h-10 border border-zinc-300 px-3"
                  >
                    <option value="included">
                      税额计入成本（含不可抵扣税）
                    </option>
                    <option value="excluded">
                      税额不计入成本（需确认可抵扣）
                    </option>
                  </select>
                  <span className="block text-xs text-zinc-500 mt-1">
                    核算金额：{accountingAmount || "0.00"}{" "}
                    元，请按实际税务口径选择。
                  </span>
                </label>
              </div>
            </details>
            <p className="text-sm text-blue-800">
              {form.cost_type === "period"
                ? "这笔其他经营费用按发生月份直接汇总，无需分配到零件。"
                : "保存后，请把这笔生产费用计入对应零件的完工记录，预计结余才会扣除。"}
            </p>
            <label className="block">
              <span className="text-sm font-medium">备注</span>
              <textarea
                rows={3}
                maxLength={500}
                value={form.notes}
                onChange={(e) => set("notes", e.target.value)}
                className="mt-1 w-full border border-zinc-300 px-3 py-2"
              />
            </label>
            <div className="flex justify-end gap-2">
              <button
                data-escape-close
                type="button"
                onClick={onClose}
                className="h-10 px-4 border border-zinc-300 text-sm"
              >
                取消
              </button>
              <button
                type="submit"
                disabled={saving}
                className="h-10 px-4 bg-zinc-900 text-white text-sm flex items-center gap-2 disabled:opacity-50"
              >
                {saving && <Loader2 className="w-4 h-4 animate-spin" />}
                保存成本费用
              </button>
            </div>
          </fieldset>
        </form>
      </div>
    </div>
  );
};

export default CostEntryModal;
