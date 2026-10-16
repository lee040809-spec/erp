import { useRef, useState } from "react";
import type { OutsourcingSupplier } from "../types/outsourcingOrder";
import { flowRequest } from "../services/productionFlow";

const empty = {
  name: "",
  contact_name: "",
  phone: "",
  address: "",
  capabilities: "",
  notes: "",
  active: true,
};
export default function SupplierEditor({
  supplier,
  onDirty,
  onBusy,
  onCancel,
  onSaved,
}: {
  supplier?: OutsourcingSupplier;
  onDirty: () => void;
  onBusy: (value: boolean) => void;
  onCancel: () => void;
  onSaved: () => Promise<void>;
}) {
  const [form, setForm] = useState(supplier ? { ...supplier } : empty);
  const [saving, setSaving] = useState(false),
    [error, setError] = useState("");
  const busy = useRef(false);
  return (
    <form
      className="erp-card p-3 space-y-3"
      onSubmit={async (e) => {
        e.preventDefault();
        if (busy.current) return;
        if (!form.name.trim()) {
          setError("请填写供应商名称");
          return;
        }
        busy.current = true;
        setSaving(true);
        onBusy(true);
        setError("");
        try {
          await flowRequest(
            `/outsourcing-suppliers${supplier ? `/${supplier.id}` : ""}`,
            supplier ? "PATCH" : "POST",
            { ...form, name: form.name.trim() },
          );
          await onSaved();
        } catch (error) {
          setError(error instanceof Error ? error.message : "保存失败");
        } finally {
          busy.current = false;
          setSaving(false);
          onBusy(false);
        }
      }}
    >
      <h3 className="font-semibold">
        {supplier ? "编辑供应商" : "新增供应商"}
      </h3>
      <fieldset disabled={saving} className="space-y-3">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {(
            [
              ["name", "供应商名称", 100],
              ["contact_name", "联系人", 100],
              ["phone", "联系电话", 60],
              ["capabilities", "加工能力 / 可承接工序", 300],
              ["address", "地址", 300],
              ["notes", "备注", 500],
            ] as const
          ).map(([key, label, max]) => (
            <label className="erp-label" key={key}>
              {label}
              {key === "name" ? " *" : ""}
              <input
                className="erp-input"
                required={key === "name"}
                maxLength={max}
                value={form[key]}
                onChange={(e) => {
                  onDirty();
                  setForm({ ...form, [key]: e.target.value });
                }}
              />
            </label>
          ))}
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={form.active}
            onChange={(e) => {
              onDirty();
              setForm({ ...form, active: e.target.checked });
            }}
          />
          启用（停用后不能新增外协下单或发出，已有回厂和对账继续处理）
        </label>
        {error && (
          <p role="alert" className="text-rose-600 text-sm">
            {error}
          </p>
        )}
        <div className="flex gap-2">
          <button className="erp-btn erp-primary">
            {saving ? "保存中…" : "保存供应商"}
          </button>
          <button type="button" className="erp-btn" onClick={onCancel}>
            取消编辑
          </button>
        </div>
      </fieldset>
    </form>
  );
}
