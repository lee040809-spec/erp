import { useEffect, useMemo, useRef, useState } from "react";
import {
  fetchFinanceAccess,
  type FinanceAccess,
} from "../services/financeAccess";
import { fetchFinanceSummaryApi } from "../services/api";
import type { FinanceSummaryResponse, AnnualFinanceSummary } from "../types";
import BusinessEntrySettings from "../components/BusinessEntrySettings";
import CostEntryModal from "../components/CostEntryModal";
import FinanceReviewPanel from "../components/FinanceReviewPanel";
import AccountingGuide from "../features/accounting/AccountingGuide";
import FinanceMonthTable from "../features/accounting/FinanceMonthTable";
import OrderCostMonthTable from "../features/accounting/OrderCostMonthTable";
import {
  defaultFinanceMonth,
  financeAmount,
  financeCards,
  type FinanceMode,
} from "../features/accounting/financePresentation";

const emptyAnnual = (year: string): AnnualFinanceSummary => ({
  year,
  output_value: "0",
  production_cost: "0",
  production_profit: "0",
  period_expense: "0",
  operating_profit: "0",
});
export default function Finance({ mode = "overview" }: { mode?: FinanceMode }) {
  const currentYear = String(new Date().getFullYear());
  const [year, setYear] = useState(currentYear),
    [summary, setSummary] = useState<FinanceSummaryResponse>({
      year: currentYear,
      months: [],
      annual: emptyAnnual(currentYear),
    });
  const [access, setAccess] = useState<FinanceAccess | null>(null),
    [loading, setLoading] = useState(true),
    [error, setError] = useState("");
  const [showCost, setShowCost] = useState(false),
    [policyVersion, setPolicyVersion] = useState(0),
    [month, setMonth] = useState(defaultFinanceMonth(currentYear));
  const [section, setSection] = useState<"details" | "history" | "allocation">(
      "details",
    ),
    [selectionVersion, setSelectionVersion] = useState(0);
  const [panelBusy, setPanelBusy] = useState(false);
  const [settingsOpened, setSettingsOpened] = useState(false);
  const detailRef = useRef<HTMLDivElement>(null),
    generation = useRef(0);
  const years = useMemo(
    () => Array.from({ length: 6 }, (_, i) => String(Number(currentYear) - i)),
    [currentYear],
  );
  const load = async () => {
    const version = ++generation.current;
    setLoading(true);
    setError("");
    try {
      const [data, permissions] = await Promise.all([
        fetchFinanceSummaryApi(year),
        fetchFinanceAccess(),
      ]);
      if (version === generation.current) {
        setSummary(data);
        setAccess(permissions);
      }
    } catch (err) {
      if (version === generation.current) {
        setError(err instanceof Error ? err.message : "经营数据读取失败");
        setAccess(null);
      }
    } finally {
      if (version === generation.current) setLoading(false);
    }
  };
  useEffect(() => {
    setMonth(defaultFinanceMonth(year));
    setSection("details");
    void load();
    return () => {
      generation.current++;
    };
  }, [year]);
  const openSection = (next: typeof section, selectedMonth = month) => {
    if (panelBusy) return;
    setMonth(selectedMonth);
    setSection(next);
    setSelectionVersion((v) => v + 1);
    detailRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  };
  const selectedMonthSummary = summary.months.find((m) => m.month === month);
  const exportCsv = () => {
    if (loading || error) return;
    const header =
      mode === "output"
        ? [
            "月份",
            "每月订单金额",
            "完工产值金额",
            "合格件数",
            "计价件数",
            "备用件数",
          ]
        : [
            "月份",
            "完成金额",
            "材料花费",
            "外协花费",
            "其他生产花费",
            "生产花费合计",
            "其他经营花费",
            "预计结余",
            "核对状态",
            "订单填写材料费",
            "订单填写外协费",
            "订单填写刀具费",
            "订单填写工装费",
            "订单填写其他费",
          ];
    const rows = summary.months.map((m) =>
      mode === "output"
        ? [
            m.month,
            m.order_amount,
            m.output_value,
            m.qualified_quantity,
            m.billable_quantity,
            m.reserve_quantity,
          ]
        : [
            m.month,
            m.output_value,
            m.material_cost,
            m.outsourcing_cost,
            m.other_production_cost,
            m.production_cost,
            m.period_expense,
            m.operating_profit,
            m.closing_status === "closed" ? "已核对并锁定" : "可继续补充",
            m.order_costs?.material,
            m.order_costs?.outsourcing,
            m.order_costs?.tool,
            m.order_costs?.fixture,
            m.order_costs?.other,
          ],
    );
    const csv =
      "\ufeff" +
      [header, ...rows]
        .map((row) =>
          row
            .map((cell) => `"${String(cell ?? "").replace(/"/g, '""')}"`)
            .join(","),
        )
        .join("\r\n");
    const url = URL.createObjectURL(
        new Blob([csv], { type: "text/csv;charset=utf-8" }),
      ),
      link = document.createElement("a");
    link.href = url;
    link.download = `${mode === "output" ? "产值统计" : "经营结果"}-${year}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  };
  return (
    <div className="erp-page">
      <div className="erp-finance-content px-4 md:px-8 py-5 space-y-4">
        <header className="flex flex-wrap justify-between items-center gap-3">
          <div>
            <h2 className="text-2xl font-bold">
              {mode === "output" ? "产值统计" : "财务报表 · 经营结果"}
            </h2>
            <p className="text-sm text-slate-500 mt-1">
              {mode === "output"
                ? "按合格完工日期查看数量和金额"
                : "用完成金额和已计入费用，了解每月的经营情况"}{" "}
              · 金额单位：元
            </p>
          </div>
          <div className="flex gap-2 flex-wrap">
            <select
              disabled={panelBusy}
              aria-label="报表年份"
              className="erp-input !w-auto"
              value={year}
              onChange={(e) => setYear(e.target.value)}
            >
              {years.map((y) => (
                <option key={y} value={y}>
                  {y} 年
                </option>
              ))}
            </select>
            <button
              type="button"
              className="erp-btn"
              disabled={loading || panelBusy}
              onClick={load}
            >
              刷新报表
            </button>
            <button
              type="button"
              className="erp-btn"
              disabled={
                panelBusy || loading || !!error || !summary.months.length
              }
              onClick={exportCsv}
            >
              导出表格
            </button>
          </div>
        </header>
        <AccountingGuide mode={mode} />
        {mode === "output" && (
          <section className="erp-card p-4" aria-label="所选月份金额对比">
            <label className="erp-label max-w-56">
              统计月份
              <input
                type="month"
                aria-label="产值统计月份"
                className="erp-input"
                value={month}
                min={`${year}-01`}
                max={`${year}-12`}
                disabled={loading || panelBusy}
                onChange={(e) => {
                  if (e.target.value.startsWith(year + "-")) {
                    setMonth(e.target.value);
                    setSelectionVersion((v) => v + 1);
                  }
                }}
              />
            </label>
            <div className="grid grid-cols-2 gap-4 mt-3">
              {[
                ["当月订单金额", selectedMonthSummary?.order_amount],
                ["完工产值金额", selectedMonthSummary?.output_value],
              ].map(([label, value]) => (
                <div key={label}>
                  <p className="text-sm text-slate-500">{label}</p>
                  <p className="font-bold text-2xl text-blue-800 mt-1">
                    {loading
                      ? "读取中…"
                      : error
                        ? "未读取"
                        : financeAmount(value)}
                  </p>
                </div>
              ))}
            </div>
            <p className="text-xs text-slate-500 mt-3">
              下单金额按零件下单日期统计（未填时沿用订单日期）；完工产值按实际合格完工日期统计，可能来自以前月份订单，备用件不计产值。两种金额不相加，均不代表收款。
            </p>
          </section>
        )}
        <div className="flex flex-wrap gap-2">
          {mode === "overview" && access?.can_manage_costs && (
            <>
              <button
                type="button"
                className="erp-btn erp-primary"
                onClick={() => setShowCost(true)}
              >
                记一笔费用
              </button>
              <button
                type="button"
                className="erp-btn"
                onClick={() => openSection("allocation")}
              >
                把费用计入零件
              </button>
            </>
          )}
          {mode === "output" && access?.can_import_history && (
            <button
              type="button"
              className="erp-btn erp-primary"
              onClick={() => openSection("history")}
            >
              补录漏记完工
            </button>
          )}
          <details className="text-sm" onToggle={e=>{if(e.currentTarget.open)setSettingsOpened(true);}}>
            <summary className="erp-btn cursor-pointer">填写要求设置</summary>
            <div className="mt-2">
              {settingsOpened && <BusinessEntrySettings
                onSaved={() => {
                  void load();
                  setPolicyVersion((v) => v + 1);
                }}
              />}
            </div>
          </details>
        </div>
        {error && (
          <div role="alert" className="rounded-xl bg-rose-50 p-4 text-rose-700">
            {error}
            <button className="erp-btn ml-3" onClick={load}>
              重新读取
            </button>
          </div>
        )}
        <div className="erp-finance-cards grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3">
          {financeCards(summary, mode).map((card) => (
            <article key={card.label} className="erp-card p-4">
              <p className="text-sm text-slate-600">{card.label}</p>
              <p className="erp-finance-card-amount text-2xl font-bold text-blue-800 mt-2">
                {loading ? "读取中…" : error ? "未读取" : card.value}
              </p>
              <p className="text-xs text-slate-500 mt-2 leading-5">
                {card.hint}
              </p>
            </article>
          ))}
        </div>
        {mode === "overview" && !loading && !error && (
          <aside
            className={`rounded-xl p-4 text-sm ${Number(summary.pending_cost_amount || 0) > 0 ? "bg-amber-50 text-amber-900 border border-amber-200" : "bg-slate-50 text-slate-600"}`}
          >
            还有 {financeAmount(summary.pending_cost_amount)}{" "}
            元生产费用未计入零件（{summary.pending_cost_count || 0}{" "}
            笔，包含其他年份），暂未从预计结余中扣除。
            {access?.can_manage_costs && (
              <button
                type="button"
                className="erp-btn ml-2"
                onClick={() => openSection("allocation")}
              >
                去分配费用
              </button>
            )}
            <p className="text-xs mt-1">
              没有待分配费用，也不代表所有费用都已录入。
            </p>
          </aside>
        )}
        <section className="erp-card overflow-hidden">
          <header className="px-4 py-3 border-b">
            <h3 className="font-semibold">{mode === "output" ? "每月订单金额与完工产值" : "每月情况"}</h3>
            <p className="text-xs text-slate-500 mt-1">
              {mode === "output"
                ? "下单金额与完工产值按各自发生月份统计；下单日期和订单日期均未填写的零件，不计入月度下单金额。"
                : "预计结余 = 完成金额 − 已计入的生产花费 − 其他经营花费。费用未记全时，结果为暂估。"}
            </p>
          </header>
          {loading ? (
            <p role="status" className="p-8 text-center">
              正在读取经营数据…
            </p>
          ) : (
            !error &&
            (summary.months.length ? (
              <FinanceMonthTable
                months={summary.months}
                mode={mode}
                onSelect={(m) => openSection("details", m)}
              />
            ) : (
              <div className="p-8 text-center text-slate-500">
                <p>本年度暂无下单、已确认完工或费用记录。</p>
                <p className="text-sm mt-2">
                  先在“办理零件”确认合格数量，漏记的旧记录可在“产值统计”补录。
                </p>
                {mode === "output" && access?.can_import_history && (
                  <button
                    type="button"
                    className="erp-btn mt-3"
                    onClick={() => openSection("history")}
                  >
                    补录漏记完工
                  </button>
                )}
              </div>
            ))
          )}
        </section>
        {mode === "overview" &&
          !loading &&
          !error &&
          summary.months.length > 0 && (
            <OrderCostMonthTable months={summary.months} />
          )}
        <div ref={detailRef} className="scroll-mt-4">
          {!error && access && (
            <FinanceReviewPanel
              key={policyVersion}
              access={access}
              mode={mode}
              initialTab={section}
              year={year}
              month={month}
              selectionVersion={selectionVersion}
              onImported={load}
              onBusyChange={setPanelBusy}
            />
          )}
        </div>
        {showCost && access?.can_manage_costs && (
          <CostEntryModal
            onClose={() => setShowCost(false)}
            onSuccess={(costType) => {
              void load();
              openSection(costType === "period" ? "details" : "allocation");
            }}
          />
        )}
      </div>
    </div>
  );
}
