import type {
  Delivery,
  Statement,
  Detail,
} from "./deliveryReconciliationTypes";
import DeliveryStatementDetail from "./DeliveryStatementDetail";
import { useCallback, useEffect, useRef, useState } from "react";
import { useAsyncResource } from "../../hooks/useAsyncResource";
import { flowRequest, withRequestID } from "../../services/productionFlow";
import { businessToday } from "../../utils/businessDate";
import { financeAmount } from "./financePresentation";
import ResponsiveTable from "../../components/ResponsiveTable";

type Page<T> = { items: T[]; has_more: boolean };
export default function DeliveryReconciliation({
  onBusyChange,
}: {
  onBusyChange?: (busy: boolean) => void;
}) {
  const today = businessToday();
  const [filters, setFilters] = useState({
    start: today.slice(0, 7) + "-01",
    end: today,
    customer: "",
  });
  const [query, setQuery] = useState(filters),
    [page, setPage] = useState(1),
    [mode, setMode] = useState<"deliveries" | "statements">("deliveries");
  const [date, setDate] = useState(today),
    [notes, setNotes] = useState(""),
    [selected, setSelected] = useState<Map<number, Delivery>>(new Map());
  const [busy, setBusy] = useState(false),
    [message, setMessage] = useState(""),
    [detail, setDetail] = useState<Detail | null>(null),
    [reverse, setReverse] = useState<Statement | null>(null),
    [reason, setReason] = useState("");
  const lock = useRef(false),
    pending = useRef({ signature: "", id: "" });
  useEffect(() => {
    onBusyChange?.(busy);
    return () => onBusyChange?.(false);
  }, [busy, onBusyChange]);
  const request = useCallback(
    () =>
      flowRequest<Page<Delivery | Statement>>(
        `/finance/delivery-reconciliations?${new URLSearchParams({ ...query, page: String(page), mode })}`,
      ),
    [query, page, mode],
  );
  const { data, loading, error, reload } = useAsyncResource(
    request,
    { items: [], has_more: false } as Page<Delivery | Statement>,
    true,
  );
  const toggle = (row: Delivery) => {
    if (row.reconciliation_id || !row.price_ready) return;
    const next = new Map(selected);
    if (next.has(row.id)) next.delete(row.id);
    else {
      if (next.size >= 200) {
        setMessage("每次最多核对200次送货，请分次处理。");
        return;
      }
      if (next.size && [...next.values()][0].customer_id !== row.customer_id) {
        setMessage("已选清单属于另一客户，请先清空清单。");
        return;
      }
      next.set(row.id, row);
    }
    setSelected(next);
    setMessage("");
  };
  const act = async (work: () => Promise<void>) => {
    if (lock.current) return;
    lock.current = true;
    setBusy(true);
    setMessage("");
    try {
      await work();
    } catch (e) {
      setMessage((e as Error).message + "；填写内容已保留，请核对后重试。");
    } finally {
      lock.current = false;
      setBusy(false);
    }
  };
  const save = () =>
    act(async () => {
      const rows = [...selected.values()];
      if (!rows.length) return;
      const header = await flowRequest<Statement>(
        "/finance/delivery-reconciliations",
        "POST",
        withRequestID(pending.current, {
          customer_id: rows[0].customer_id,
          business_date: date,
          delivery_ids: rows.map((r) => r.id).sort((a, b) => a - b),
          notes,
        }),
      );
      setSelected(new Map());
      setNotes("");
      setMessage(
        `已保存对账单 ${header.statement_no}。此操作不登记收款，也不重复计算产值。`,
      );
      reload();
      try {
        setDetail(
          await flowRequest<Detail>(
            `/finance/delivery-reconciliations/${header.id}`,
          ),
        );
      } catch {
        setMessage(
          `对账单 ${header.statement_no} 已保存，但明细读取失败。请在“已保存对账单”中重试查看，无需再次保存。`,
        );
      }
    });
  return (
    <section className="erp-card p-4 space-y-3" aria-label="按送货记录对账">
      <p className="text-sm text-slate-500">
        按每次实际送货核对金额；保存后标记已对账。金额采用核对时订单单价，后续改单价不改历史对账单。待补价记录请先补价。
      </p>
      <div className="flex flex-wrap gap-2">
        <button
          className={`erp-btn ${mode === "deliveries" ? "erp-primary" : ""}`}
          disabled={busy}
          onClick={() => {
            setMode("deliveries");
            setPage(1);
          }}
        >
          待核对送货
        </button>
        <button
          className={`erp-btn ${mode === "statements" ? "erp-primary" : ""}`}
          disabled={busy}
          onClick={() => {
            setMode("statements");
            setPage(1);
          }}
        >
          已保存对账单
        </button>
      </div>
      {mode === "deliveries" && (
        <form
          className="flex flex-wrap items-end gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            setQuery({ ...filters });
            setPage(1);
          }}
        >
          <label className="erp-label">
            送货开始
            <input
              type="date"
              className="erp-input"
              required
              value={filters.start}
              onChange={(e) =>
                setFilters({ ...filters, start: e.target.value })
              }
            />
          </label>
          <label className="erp-label">
            送货结束
            <input
              type="date"
              className="erp-input"
              min={filters.start}
              required
              value={filters.end}
              onChange={(e) => setFilters({ ...filters, end: e.target.value })}
            />
          </label>
          <label className="erp-label">
            客户
            <input
              className="erp-input"
              placeholder="名称 / 简称"
              value={filters.customer}
              onChange={(e) =>
                setFilters({ ...filters, customer: e.target.value })
              }
            />
          </label>
          <button className="erp-btn" disabled={loading || busy}>
            查询（保留已选）
          </button>
        </form>
      )}
      {message && (
        <p role="status" className="text-sm bg-blue-50 p-3 rounded-lg">
          {message}
        </p>
      )}
      {error && (
        <p role="alert" className="text-rose-700">
          {error}
          <button className="erp-btn" onClick={reload}>
            重试读取
          </button>
        </p>
      )}
      {loading ? (
        <p role="status">正在读取对账资料…</p>
      ) : (
        <div className="overflow-auto">
          <ResponsiveTable className="w-full text-sm">
            <thead>
              <tr>
                {(mode === "deliveries"
                  ? [
                      "选择 / 状态",
                      "送货日期",
                      "客户 / 订单",
                      "零件",
                      "数量",
                      "金额",
                    ]
                  : ["对账日期", "对账单 / 客户", "金额", "状态", "操作"]
                ).map((s) => (
                  <th className="p-2 text-left" key={s}>
                    {s}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.items.map((r) =>
                mode === "deliveries"
                  ? (() => {
                      const row = r as Delivery;
                      return (
                        <tr key={row.id} className="border-t">
                          <td className="p-2">
                            {row.reconciliation_id ? (
                              "已对账"
                            ) : !row.price_ready ? (
                              "待补价"
                            ) : (
                              <input
                                type="checkbox"
                                aria-label={`选择送货 ${row.order_number} ${row.part_name}`}
                                disabled={busy || !!error}
                                checked={selected.has(row.id)}
                                onChange={() => toggle(row)}
                              />
                            )}
                          </td>
                          <td className="p-2 whitespace-nowrap">
                            {row.business_date}
                          </td>
                          <td className="p-2">
                            {row.customer_name}
                            <br />
                            {row.order_number}
                          </td>
                          <td className="p-2">{row.part_name}</td>
                          <td className="p-2">{row.quantity}</td>
                          <td className="p-2">
                            {row.price_ready
                              ? financeAmount(row.amount)
                              : "待补价"}
                          </td>
                        </tr>
                      );
                    })()
                  : ((row: Statement) => (
                      <tr key={row.id} className="border-t">
                        <td className="p-2">
                          {row.business_date.slice(0, 10)}
                        </td>
                        <td className="p-2 break-all">
                          {row.statement_no}
                          <br />
                          {row.customer_name}
                        </td>
                        <td className="p-2">{financeAmount(row.amount)}</td>
                        <td className="p-2">
                          {row.status === "confirmed" ? "已对账" : "已撤销"}
                        </td>
                        <td className="p-2">
                          <button
                            className="erp-btn"
                            disabled={busy}
                            onClick={() =>
                              act(async () =>
                                setDetail(
                                  await flowRequest<Detail>(
                                    `/finance/delivery-reconciliations/${row.id}`,
                                  ),
                                ),
                              )
                            }
                          >
                            查看
                          </button>
                          {row.status === "confirmed" && (
                            <button
                              className="erp-btn"
                              disabled={busy}
                              onClick={() => {
                                setReverse(row);
                                setReason("");
                              }}
                            >
                              撤销
                            </button>
                          )}
                        </td>
                      </tr>
                    ))(r as Statement),
              )}
            </tbody>
          </ResponsiveTable>
          {!data.items.length && !error && (
            <p className="py-6 text-center text-slate-500">当前范围没有记录</p>
          )}
        </div>
      )}
      <div className="flex gap-2 items-center">
        <button
          className="erp-btn"
          disabled={page === 1 || loading || busy}
          onClick={() => setPage(page - 1)}
        >
          上一页
        </button>
        <span>第 {page} 页</span>
        <button
          className="erp-btn"
          disabled={!data.has_more || loading || busy}
          onClick={() => setPage(page + 1)}
        >
          下一页
        </button>
      </div>
      {selected.size > 0 && (
        <div className="rounded-xl bg-blue-50 p-3 space-y-2 text-sm">
          <strong>
            {[...selected.values()][0].customer_name} · 已选 {selected.size}{" "}
            次送货 ·{" "}
            {[...selected.values()].reduce((n, r) => n + r.quantity, 0)} 件 ·
            参考金额{" "}
            {financeAmount(
              [...selected.values()].reduce((n, r) => n + Number(r.amount), 0),
            )}
          </strong>
          <p>
            已选清单跨页保留，提交时服务器会重新核对数量、单价和已对账状态。
          </p>
          <div className="max-h-32 overflow-auto">
            {[...selected.values()].map((r) => (
              <span key={r.id} className="inline-block mr-3">
                {r.part_name} {r.quantity} 件
                <button
                  className="ml-1 text-blue-700"
                  disabled={busy}
                  onClick={() => toggle(r)}
                >
                  移除
                </button>
              </span>
            ))}
          </div>
          <label>
            对账日期{" "}
            <input
              type="date"
              className="erp-input !w-auto"
              max={today}
              value={date}
              disabled={busy}
              onChange={(e) => setDate(e.target.value)}
            />
          </label>
          <label className="block">
            备注（选填）
            <input
              className="erp-input"
              maxLength={500}
              value={notes}
              disabled={busy}
              onChange={(e) => setNotes(e.target.value)}
            />
          </label>
          <button
            className="erp-btn erp-primary"
            disabled={busy || !date || !!error}
            onClick={save}
          >
            {busy ? "正在保存…" : "保存对账并标记已对账"}
          </button>
          <button
            className="erp-btn ml-2"
            disabled={busy}
            onClick={() => setSelected(new Map())}
          >
            清空已选
          </button>
        </div>
      )}
      {reverse && (
        <div className="rounded-xl border border-amber-200 bg-amber-50 p-3">
          <p>撤销 {reverse.statement_no}，保留原明细，释放送货记录重新对账。</p>
          <label className="erp-label mt-2">
            撤销原因（必填）
            <input
              className="erp-input"
              maxLength={500}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
          </label>
          <button
            className="erp-btn"
            disabled={busy || !reason.trim()}
            onClick={() =>
              act(async () => {
                await flowRequest(
                  `/finance/delivery-reconciliations/${reverse.id}/reverse`,
                  "POST",
                  { reason: reason.trim() },
                );
                setReverse(null);
                setDetail(null);
                setMessage("已撤销对账，原明细仍保留。");
                reload();
              })
            }
          >
            确认撤销
          </button>
          <button
            className="erp-btn"
            disabled={busy}
            onClick={() => setReverse(null)}
          >
            取消
          </button>
        </div>
      )}
      {detail && (
        <DeliveryStatementDetail
          detail={detail}
          onClose={() => setDetail(null)}
        />
      )}
    </section>
  );
}
