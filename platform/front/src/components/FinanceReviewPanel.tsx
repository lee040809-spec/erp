import HistoryCompletionDialog from "../features/accounting/HistoryCompletionDialog";
import FinanceDetailTables from "../features/accounting/FinanceDetailTables";
import ResponsiveTable from "./ResponsiveTable";
import {
  defaultFinanceMonth,
  type FinanceMode,
} from "../features/accounting/financePresentation";
import type { FinanceAccess } from "../services/financeAccess";
import React, { useEffect, useRef, useState } from "react";
import { authFetch } from "./shared";
import { basicEvidence, readEntryPolicy } from "./BusinessEntrySettings";
import type {
  Candidate,
  Batch,
  Cost,
  PendingCost,
  AllocationRecord,
} from "../features/accounting/financeTypes";
import { financeCostLabels as labels } from "../features/accounting/financePresentation";
async function request<T>(url: string, payload?: unknown): Promise<T> {
  const res = await authFetch(
    url,
    payload
      ? {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        }
      : undefined,
  );
  const result = await res.json();
  if (!res.ok) throw new Error(result.message || result.error || "读取失败");
  return result.data;
}
export default function FinanceReviewPanel({
  year,
  month,
  selectionVersion,
  onImported,
  access,
  mode = "overview",
  initialTab = "details",
  onBusyChange,
}: {
  mode?: FinanceMode;
  initialTab?: "history" | "details" | "allocation";
  onBusyChange?: (busy: boolean) => void;
  access: FinanceAccess | null;
  year: string;
  month: string;
  selectionVersion: number;
  onImported: () => void;
}) {
  const [policy, setPolicy] = useState(basicEvidence);
  useEffect(() => {
    readEntryPolicy()
      .then((v) => setPolicy(v.effective))
      .catch((e) => setError(e.message));
  }, []);
  const historyRequest = useRef({ signature: "", id: "" });
  const [tab, setTab] = useState<"history" | "details" | "allocation">(
    initialTab,
  );
  const [page, setPage] = useState(1);
  const [candidates, setCandidates] = useState<Candidate[]>([]);
  const [total, setTotal] = useState(0);
  const [selectedMonth, setSelectedMonth] = useState(defaultFinanceMonth(year));
  const [detailPage, setDetailPage] = useState(1),
    [detailMore, setDetailMore] = useState(false),
    [hasCompletions, setHasCompletions] = useState(false),
    [pendingPage, setPendingPage] = useState(1),
    [pendingMore, setPendingMore] = useState(false);
  const [batches, setBatches] = useState<Batch[]>([]);
  const [costs, setCosts] = useState<Cost[]>([]);
  const [pending, setPending] = useState<PendingCost[]>([]);
  const [allocationRecords, setAllocationRecords] = useState<
    AllocationRecord[]
  >([]);
  const [allocation, setAllocation] = useState({
    cost: "",
    batch: "",
    amount: "",
    basis: "",
  });
  const [closeNotes, setCloseNotes] = useState("");
  const [closeReviewed, setCloseReviewed] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [selected, setSelected] = useState<Candidate | null>(null);
  const [form, setForm] = useState({
    quantity: "",
    date: "",
    notes: "",
    sourceRef: "",
    price: "",
    reviewed: false,
  });
  const [saving, setSaving] = useState(false);
  const [readFailed, setReadFailed] = useState(false);
  useEffect(() => {
    onBusyChange?.(saving);
    return () => onBusyChange?.(false);
  }, [saving, onBusyChange]);
  const busy = useRef(false);
  const generation = useRef(0);
  const load = async () => {
    const current = ++generation.current;
    setLoading(true);
    setReadFailed(false);
    setCloseReviewed(false);
    setError("");
    try {
      if (tab === "history") {
        const data = await request<{
          items: Candidate[];
          total: number;
        }>(`/api/platform/finance/history?year=${year}&page=${page}`);
        if (current === generation.current) {
          setCandidates(data.items || []);
          setTotal(data.total);
        }
      } else {
        const data = await request<{
          completions: Batch[];
          costs: Cost[];
          allocations: AllocationRecord[];
          has_more: boolean;
          has_completions: boolean;
        }>(
          `/api/platform/finance/details?month=${selectedMonth}&page=${detailPage}&pageSize=50`,
        );
        if (current === generation.current) {
          setDetailMore(data.has_more);
          setHasCompletions(data.has_completions);
          setBatches(data.completions || []);
          setCosts(data.costs || []);
          setAllocationRecords(data.allocations || []);
        }
        if (tab === "allocation") {
          const result = await request<{
            items: PendingCost[];
            has_more: boolean;
          }>(
            `/api/platform/finance/pending-costs?page=${pendingPage}&pageSize=50`,
          );
          if (current === generation.current) {
            setPending(result.items || []);
            setPendingMore(result.has_more);
          }
        }
      }
    } catch (e) {
      if (current === generation.current) {
        setReadFailed(true);
        setError(e instanceof Error ? e.message : "读取失败");
      }
    } finally {
      if (current === generation.current) setLoading(false);
    }
  };
  useEffect(() => {
    setPage(1);
    setSelected(null);
    setSelectedMonth(defaultFinanceMonth(year));
  }, [year]);
  useEffect(() => {
    if (month) {
      setSelectedMonth(month);
      setTab(initialTab);
    }
  }, [month, selectionVersion, initialTab]);
  useEffect(() => {
    load();
    return () => {
      generation.current++;
    };
  }, [year, page, tab, selectedMonth, detailPage, pendingPage]);
  useEffect(() => {
    setCloseReviewed(false);
    setCloseNotes("");
    setDetailPage(1);
    setPendingPage(1);
    setAllocation({ cost: "", batch: "", amount: "", basis: "" });
  }, [selectedMonth, year]);
  const review = (item: Candidate) => {
    setSelected(item);
    setMessage("");
    setError("");
    setForm({
      quantity: String(item.remaining_quantity),
      date: item.business_date || "",
      notes: "",
      sourceRef: "",
      price: item.unit_price || "",
      reviewed: false,
    });
  };
  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!selected || busy.current || !access?.can_import_history) return;
    busy.current = true;
    setSaving(true);
    setError("");
    try {
      const signature = JSON.stringify([selected.order_item_id, form]);
      if (historyRequest.current.signature !== signature)
        historyRequest.current = { signature, id: crypto.randomUUID() };
      await request("/api/platform/finance/history", {
        request_id: historyRequest.current.id,
        order_item_id: selected.order_item_id,
        qualified_quantity: Number(form.quantity),
        business_date: form.date,
        expected_price: selected.unit_price,
        source_ref: form.sourceRef.trim(),
        confirmed_price: form.price.trim(),
        notes: form.notes.trim(),
        reviewed: true,
      });
      setSelected(null);
      setMessage("已计入正式产值，请在对应年份、月份查看明细。");
      onImported();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "导入失败");
    } finally {
      busy.current = false;
      setSaving(false);
    }
  };
  const allocate = async (event: React.FormEvent) => {
    event.preventDefault();
    if (busy.current || readFailed || loading || !access?.can_manage_costs)
      return;
    busy.current = true;
    setSaving(true);
    setError("");
    setMessage("");
    try {
      await request("/api/platform/finance/allocations", {
        cost_entry_id: Number(allocation.cost),
        production_completion_id: Number(allocation.batch),
        allocated_amount: allocation.amount,
        allocation_basis: allocation.basis.trim(),
      });
      setAllocation({ cost: "", batch: "", amount: "", basis: "" });
      setMessage("分摊已保存，生产成本已计入该批次完工月份。");
      onImported();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "分摊失败");
    } finally {
      busy.current = false;
      setSaving(false);
    }
  };
  const closeMonth = async (event: React.FormEvent) => {
    event.preventDefault();
    if (
      busy.current ||
      readFailed ||
      loading ||
      !closeReviewed ||
      selectedMonth >= defaultFinanceMonth(String(new Date().getFullYear())) ||
      !access?.can_manage_costs
    )
      return;
    busy.current = true;
    setSaving(true);
    setError("");
    setMessage("");
    try {
      await request(`/api/platform/finance/months/${selectedMonth}/close`, {
        reviewed_cost_complete: closeReviewed,
        notes: closeNotes.trim(),
      });
      setMessage(`${selectedMonth} 已结账，新增完工、费用和分摊将被锁定。`);
      setCloseNotes("");
      setCloseReviewed(false);
      onImported();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "月结失败");
    } finally {
      busy.current = false;
      setSaving(false);
    }
  };
  return (
    <section className="bg-white border border-zinc-200 p-5 space-y-4">
      <div className="flex flex-wrap gap-2 items-center">
        {mode === "output" && (
          <button
            disabled={saving}
            onClick={() => setTab("history")}
            className={`px-4 py-2 border ${tab === "history" ? "bg-zinc-900 text-white" : ""}`}
          >
            漏记待补列表
          </button>
        )}
        <button
          disabled={saving}
          onClick={() => setTab("details")}
          className={`px-4 py-2 border ${tab === "details" ? "bg-zinc-900 text-white" : ""}`}
        >
          {mode === "output" ? "本月完成记录" : "本月费用与完工记录"}
        </button>
        {mode === "overview" && access?.can_manage_costs && (
          <button
            disabled={saving}
            onClick={() => setTab("allocation")}
            className={`px-4 py-2 border ${tab === "allocation" ? "bg-zinc-900 text-white" : ""}`}
          >
            把费用计入零件
          </button>
        )}
        <button
          onClick={load}
          disabled={loading || saving}
          className="px-3 py-2 border disabled:opacity-50"
        >
          刷新明细
        </button>
      </div>
      {error && (
        <p role="alert" className="p-3 bg-rose-50 text-rose-700">
          {error}
        </p>
      )}
      {message && (
        <p role="status" className="p-3 bg-emerald-50 text-emerald-700">
          {message}
        </p>
      )}
      {tab === "history" ? (
        <>
          <p className="text-sm text-zinc-600">
            {year} 年待审核 {total}{" "}
            个零件（含缺少完工日期的记录）。旧订单数量不等于合格数量，确认后才进入正式统计。支持分多个月补录，系统自动生成批次编号，累计不能超过剩余数量。历史成本仍需另外审核录入。
          </p>
          <div className="erp-table-scroll overflow-auto max-h-[500px]">
            <ResponsiveTable className="w-full min-w-[850px] text-sm">
              <thead className="bg-zinc-50 sticky top-0">
                <tr>
                  {[
                    "订单 / 客户",
                    "零件",
                    "已记数量 / 可补数量",
                    "原完工日期",
                    "当前单价",
                    "供料",
                    "操作",
                  ].map((x) => (
                    <th key={x} className="text-left p-3">
                      {x}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {!loading &&
                  candidates.map((item) => (
                    <tr key={item.order_item_id} className="border-t">
                      <td className="p-3">
                        {item.order_number}
                        <div className="text-zinc-500">
                          {item.customer_name}
                        </div>
                      </td>
                      <td className="p-3">{item.part_name}</td>
                      <td className="p-3">
                        已记 {item.confirmed_quantity} 件 · 可补{" "}
                        {item.remaining_quantity} 件
                      </td>
                      <td className="p-3">
                        {item.business_date || "缺失，需补录"}
                      </td>
                      <td className="p-3">
                        {Number(item.unit_price) > 0
                          ? item.unit_price
                          : "未填单价"}{" "}
                        {item.currency === "CNY" ? "" : item.currency}
                        {item.tax_inclusive ? "（含税）" : ""}
                      </td>
                      <td className="p-3">
                        {item.supply_mode === "customer_supplied"
                          ? "客户供料"
                          : "自购材料"}
                      </td>
                      <td className="p-3">
                        {access?.can_import_history &&
                        Number(item.unit_price) > 0 &&
                        item.currency === "CNY" &&
                        item.remaining_quantity > 0 ? (
                          <button
                            onClick={() => review(item)}
                            disabled={saving}
                            className="text-blue-700 underline"
                          >
                            补录完工
                          </button>
                        ) : (
                          <span className="text-amber-700">
                            {!access
                              ? "正在核对权限…"
                              : !access.can_import_history
                                ? "仅查看"
                                : item.currency !== "CNY"
                                  ? "该记录不是人民币，请核对订单"
                                  : Number(item.unit_price) <= 0
                                    ? "先在订单中填写单价"
                                    : "数量已补齐"}
                          </span>
                        )}
                      </td>
                    </tr>
                  ))}
              </tbody>
            </ResponsiveTable>
          </div>
          {!loading && !candidates.length && (
            <p className="py-6 text-center text-zinc-500">
              当前年份没有待审核的历史完工记录。
            </p>
          )}
          <div className="flex items-center gap-3 text-sm">
            <button
              disabled={page <= 1 || loading || saving}
              onClick={() => setPage(page - 1)}
              className="border px-3 py-2 disabled:opacity-40"
            >
              上一页
            </button>
            <span>
              第 {page} 页 / {Math.max(1, Math.ceil(total / 50))} 页
            </span>
            <button
              disabled={page * 50 >= total || loading || saving}
              onClick={() => setPage(page + 1)}
              className="border px-3 py-2 disabled:opacity-40"
            >
              下一页
            </button>
          </div>
        </>
      ) : tab === "allocation" ? (
        <>
          <p className="text-sm text-zinc-600">
            先选一笔费用，再选合格完工记录，确认要计入的金额。一次可计入一部分，剩余金额以后再分配；没有完工记录时，先到“办理零件”确认合格数量。
          </p>
          <label className="text-sm">
            费用计入哪个完工月份{" "}
            <input
              aria-label="费用计入哪个完工月份"
              type="month"
              required
              min={`${year}-01`}
              max={`${year}-12`}
              disabled={saving}
              value={selectedMonth}
              onChange={(e) => {
                setSelectedMonth(e.target.value);
                setAllocation({ ...allocation, batch: "" });
              }}
              className="border px-3 py-2 ml-2"
            />
          </label>
          <form onSubmit={allocate}>
            <fieldset
              disabled={saving || loading || readFailed}
              className="grid md:grid-cols-2 gap-4 text-sm"
            >
              <label>
                尚未计入零件的费用
                <select
                  required
                  value={allocation.cost}
                  onChange={(e) =>
                    setAllocation({
                      ...allocation,
                      cost: e.target.value,
                      amount:
                        pending.find((c) => String(c.id) === e.target.value)
                          ?.remaining_amount || "",
                      batch: "",
                    })
                  }
                  className="block border w-full p-2 mt-1"
                >
                  <option value="">请选择费用</option>
                  {pending.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.business_date} · {labels[c.cost_type]} ·{" "}
                      {c.supplier_name || "未填收款方"} · 剩余{" "}
                      {c.remaining_amount} 元 {c.notes || ""}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                计入哪个零件的完工记录
                <select
                  required
                  value={allocation.batch}
                  onChange={(e) =>
                    setAllocation({ ...allocation, batch: e.target.value })
                  }
                  className="block border w-full p-2 mt-1"
                >
                  <option value="">请选择完工记录</option>
                  {batches.map((b) => (
                    <option key={b.id} value={b.id}>
                      {b.business_date} · {b.order_number} / {b.part_name} /{" "}
                      {b.qualified_quantity} 件
                    </option>
                  ))}
                </select>
              </label>
              <label>
                本次计入金额
                <input
                  required
                  inputMode="decimal"
                  pattern="\d+(\.\d{1,2})?"
                  value={allocation.amount}
                  onChange={(e) =>
                    setAllocation({ ...allocation, amount: e.target.value })
                  }
                  className="block border w-full p-2 mt-1"
                />
              </label>
              <label>
                说明 / 备注（{policy.allocation_basis ? "必填" : "选填"}）
                <input
                  required={policy.allocation_basis}
                  maxLength={100}
                  placeholder="例如：材料领用单、外协结算单、工时比例"
                  value={allocation.basis}
                  onChange={(e) =>
                    setAllocation({ ...allocation, basis: e.target.value })
                  }
                  className="block border w-full p-2 mt-1"
                />
              </label>
              <button
                disabled={saving || loading || readFailed}
                className="bg-zinc-900 text-white p-3 disabled:opacity-40 md:col-span-2"
              >
                {saving ? "正在保存…" : "确认计入费用"}
              </button>
              <p className="text-xs text-slate-500 md:col-span-2">
                金额默认填入这笔费用的全部剩余金额，可以改成其中一部分。
              </p>
            </fieldset>
          </form>
          {!loading && !pending.length && (
            <p className="text-zinc-500 text-sm">
              暂无待分摊生产费用，请先录入实际耗用及费用。
            </p>
          )}
          {!loading && !batches.length && (
            <p className="text-amber-700 text-sm">
              目标月份没有合格完工批次，请先确认或审核导入。
            </p>
          )}
        </>
      ) : (
        <>
          <label className="text-sm">
            明细月份{" "}
            <input
              aria-label="明细月份"
              type="month"
              required
              min={`${year}-01`}
              max={`${year}-12`}
              disabled={saving}
              value={selectedMonth}
              onChange={(e) => setSelectedMonth(e.target.value)}
              className="border px-3 py-2 ml-2"
            />
          </label>
          <FinanceDetailTables
            mode={mode}
            loading={loading}
            batches={batches}
            costs={costs}
            allocationRecords={allocationRecords}
          />
          {mode === "overview" && access?.can_manage_costs && (
            <details className="rounded-xl border border-slate-200 p-3">
              <summary className="cursor-pointer text-sm">
                核对完成后锁定月份（可选）
              </summary>
              <form onSubmit={closeMonth} className="border-t pt-4 space-y-3">
                <h4 className="font-semibold">核对并锁定本月</h4>
                <p className="text-sm text-zinc-600">
                  日常使用不需要锁定。只有确认某个已结束月份的数量、单价和全部费用都正确时再操作；锁定后不能再补记该月的完工和费用。
                </p>
                <label className="flex gap-2 text-sm">
                  <input
                    required
                    type="checkbox"
                    checked={closeReviewed}
                    onChange={(e) => setCloseReviewed(e.target.checked)}
                  />
                  已核对完工数量和单价，材料、外协、人工等费用均已记全，确认锁定本月。
                </label>
                <label className="block text-sm">
                  核对说明（{policy.closing_notes ? "必填" : "选填"}）
                  <input
                    required={policy.closing_notes}
                    maxLength={500}
                    value={closeNotes}
                    onChange={(e) => setCloseNotes(e.target.value)}
                    className="block border w-full p-2 mt-1"
                  />
                </label>
                <button
                  disabled={
                    saving ||
                    loading ||
                    readFailed ||
                    !hasCompletions ||
                    !closeReviewed ||
                    selectedMonth >=
                      defaultFinanceMonth(String(new Date().getFullYear()))
                  }
                  className="border border-zinc-800 px-4 py-2 text-sm disabled:opacity-40"
                >
                  {saving ? "正在处理…" : `确认锁定 ${selectedMonth}`}
                </button>
              </form>
            </details>
          )}
        </>
      )}
      {loading && (
        <p role="status" className="py-6 text-center text-zinc-500">
          正在读取…
        </p>
      )}
      {tab !== "history" && (
        <div className="erp-history-pagination flex flex-wrap items-center justify-end gap-3 text-sm">
          <button
            className="erp-btn"
            disabled={saving || loading || detailPage === 1}
            onClick={() => {
              setDetailPage((p) => p - 1);
              setAllocation((a) => ({ ...a, batch: "" }));
            }}
          >
            上一页明细
          </button>
          <span>明细第 {detailPage} 页 · 每类50条</span>
          <button
            className="erp-btn"
            disabled={saving || loading || !detailMore}
            onClick={() => {
              setDetailPage((p) => p + 1);
              setAllocation((a) => ({ ...a, batch: "" }));
            }}
          >
            下一页明细
          </button>
          {tab === "allocation" && (
            <>
              <button
                className="erp-btn"
                disabled={saving || loading || pendingPage === 1}
                onClick={() => {
                  setPendingPage((p) => p - 1);
                  setAllocation((a) => ({ ...a, cost: "" }));
                }}
              >
                上一页尚未计入零件的费用
              </button>
              <span>费用第 {pendingPage} 页</span>
              <button
                className="erp-btn"
                disabled={saving || loading || !pendingMore}
                onClick={() => {
                  setPendingPage((p) => p + 1);
                  setAllocation((a) => ({ ...a, cost: "" }));
                }}
              >
                下一页尚未计入零件的费用
              </button>
            </>
          )}
        </div>
      )}
      {selected && (
        <HistoryCompletionDialog
          selected={selected}
          form={form}
          setForm={setForm}
          saving={saving}
          error={error}
          policy={policy}
          onSubmit={submit}
          onClose={() => setSelected(null)}
          canSave={!!access?.can_import_history}
        />
      )}
    </section>
  );
}
