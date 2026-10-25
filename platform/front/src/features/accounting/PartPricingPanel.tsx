import { useRef, useState } from "react";
import { flowRequest } from "../../services/productionFlow";

export default function PartPricingPanel({
  itemId,
  canPrice,
  onChanged,
  onBusyChange,
}: {
  itemId: number;
  canPrice: boolean;
  onChanged: () => Promise<void>;
  onBusyChange: (busy: boolean) => void;
}) {
  const [price, setPrice] = useState(""),
    [saving, setSaving] = useState(false),
    [error, setError] = useState("");
  const lock = useRef(false);
  const save = async () => {
    if (lock.current) return;
    if (!/^\d{1,8}(\.\d{1,2})?$/.test(price) || Number(price) <= 0) {
      setError("请填写大于0、最多2位小数且不超过99999999.99的单价");
      return;
    }
    lock.current = true;
    setSaving(true);
    onBusyChange(true);
    setError("");
    let saved = false;
    try {
      await flowRequest(`/order-items/${itemId}/price`, "POST", {
        unit_price: price,
      });
      saved = true;
      await onChanged();
    } catch (e) {
      setError(
        saved
          ? "单价已保存，但页面刷新失败。请重新打开零件核对，勿重复变更价格。"
          : (e as Error).message,
      );
    } finally {
      lock.current = false;
      setSaving(false);
      onBusyChange(false);
    }
  };
  return (
    <aside className="mx-5 my-3 rounded-xl bg-amber-50 border border-amber-200 p-3 text-sm">
      <strong>金额待补价</strong>
      <p className="mt-1">
        可先确认合格并登记交货。补价后，按原合格完工月份计入产值；备用件仍不计产值。
      </p>
      {canPrice ? (
        <div className="flex flex-wrap gap-2 mt-2 items-center">
          <label>
            单价{" "}
            <input
              aria-label="补录零件单价"
              className="erp-input !w-36"
              inputMode="decimal"
              value={price}
              disabled={saving}
              onChange={(e) => setPrice(e.target.value)}
            />
          </label>
          <button
            type="button"
            className="erp-btn"
            disabled={saving}
            onClick={save}
          >
            {saving ? "正在补价…" : "保存单价并计入产值"}
          </button>
        </div>
      ) : (
        <p className="mt-2 text-slate-500">
          请由有订单价格维护权限的人员补价。
        </p>
      )}
      {error && (
        <p role="alert" className="text-rose-700 mt-2">
          {error}；原填写内容已保留。
        </p>
      )}
    </aside>
  );
}
