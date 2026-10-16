import { useCallback, useState } from "react";
import type { Order } from "../../types";
import { fetchOrdersApi } from "../../services/api";
import { useAsyncResource } from "../../hooks/useAsyncResource";

type SearchKey = "orderNumber" | "partName" | "customerName";
const initial = { data: [] as Order[], total: 0 };
export default function CostOrderPicker({
  onPick,
}: {
  onPick: (orderId: string, itemId: string) => void;
}) {
  const [text, setText] = useState(""),
    [by, setBy] = useState<SearchKey>("orderNumber");
  const [query, setQuery] = useState({ by, text: "" }),
    [selected, setSelected] = useState<Order | null>(null),
    [item, setItem] = useState("");
  const request = useCallback(
    () =>
      fetchOrdersApi(undefined, 1, 20, undefined, { [query.by]: query.text }),
    [query],
  );
  const { data, loading, error, reload } = useAsyncResource(
    request,
    initial,
    !!query.text,
  );
  const search = () => {
    if (text.trim()) setQuery({ by, text: text.trim() });
  };
  return (
    <section className="rounded-xl border border-slate-200 p-3 space-y-3">
      <h4 className="text-sm font-semibold">
        这笔费用用于哪个订单或零件？（选填）
      </h4>
      <p className="text-xs text-slate-500">
        选名称即可，不用查内部编号。暂时不确定可以不选，之后再分配费用。
      </p>
      {selected && (
        <div className="rounded-lg bg-blue-50 p-3 text-sm">
          <p>
            已关联：{selected.order_number} ·{" "}
            {selected.customer_short_name || selected.customer_name}
          </p>
          <label className="erp-label mt-2">
            对应零件
            <select
              className="erp-input"
              value={item}
              onChange={(e) => {
                setItem(e.target.value);
                onPick(String(selected.id), e.target.value);
              }}
            >
              <option value="">整个订单，暂不指定零件</option>
              {(selected.items || []).map((i) => (
                <option key={i.id} value={i.id}>
                  {i.part_name}
                  {i.part_number ? ` · ${i.part_number}` : ""}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            className="erp-btn mt-2"
            onClick={() => {
              setSelected(null);
              setItem("");
              onPick("", "");
            }}
          >
            取消关联
          </button>
        </div>
      )}
      <div className="flex flex-wrap gap-2">
        <select
          aria-label="费用关联查询方式"
          className="erp-input !w-auto"
          value={by}
          onChange={(e) => setBy(e.target.value as SearchKey)}
        >
          <option value="orderNumber">按订单号</option>
          <option value="partName">按零件名称</option>
          <option value="customerName">按客户名称</option>
        </select>
        <input
          aria-label="查找费用对应订单"
          className="erp-input flex-1 min-w-0"
          placeholder="输入关键词"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              search();
            }
          }}
        />
        <button
          type="button"
          className="erp-btn"
          disabled={loading || !text.trim()}
          onClick={search}
        >
          查找订单
        </button>
      </div>
      {loading && (
        <p role="status" className="text-sm">
          正在查找订单…
        </p>
      )}
      {error && (
        <p role="alert" className="text-sm text-rose-700">
          {error}
          <button type="button" className="erp-btn ml-2" onClick={reload}>
            重试查找
          </button>
        </p>
      )}
      {!!query.text && !loading && !error && (
        <div className="max-h-40 overflow-auto space-y-1">
          {data.data.map((order) => (
            <button
              key={order.id}
              type="button"
              className="block w-full text-left rounded-lg border p-2 text-sm hover:bg-blue-50"
              onClick={() => {
                setSelected(order);
                setItem("");
                onPick(String(order.id), "");
              }}
            >
              {order.order_number} ·{" "}
              {order.customer_short_name || order.customer_name} ·{" "}
              {order.order_name || `${order.items?.length || 0}项零件`}
            </button>
          ))}
          {!data.data.length && (
            <p className="text-sm text-slate-500">
              未找到订单，请换个关键词；也可以暂不关联。
            </p>
          )}
          {data.total > 20 && (
            <p className="text-xs text-slate-500">
              只显示前 20 个结果，请输入更完整的名称或订单号。
            </p>
          )}
        </div>
      )}
    </section>
  );
}
