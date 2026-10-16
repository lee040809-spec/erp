import ResponsiveTable from "../../components/ResponsiveTable";
import type { MonthlyFinanceSummary } from "../../types";
import {
  financeAmount,
  financeQuantity,
  type FinanceMode,
} from "./financePresentation";

export default function FinanceMonthTable({
  months,
  mode,
  onSelect,
}: {
  months: MonthlyFinanceSummary[];
  mode: FinanceMode;
  onSelect: (month: string) => void;
}) {
  return (
    <div className="erp-table-scroll overflow-auto">
      <ResponsiveTable className="w-full min-w-[700px] text-sm">
        <thead className="bg-slate-50 text-slate-600">
          <tr>
            {(mode === "output"
              ? [
                  "月份",
                  "每月订单金额",
                  "完工产值金额",
                  "合格件数",
                  "计价件数",
                  "备用件数",
                  "操作",
                ]
              : [
                  "月份",
                  "完成金额",
                  "生产花费",
                  "其他经营花费",
                  "预计结余",
                  "核对状态",
                  "操作",
                ]
            ).map((label) => (
              <th key={label} className="p-3 text-left whitespace-nowrap">
                {label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {months.map((item) => (
            <tr key={item.month} className="border-t hover:bg-slate-50">
              <td className="p-3 font-semibold">{item.month}</td>
              {mode === "output" ? (
                <>
                  <td className="p-3 font-semibold">
                    {financeAmount(item.order_amount)}
                  </td>
                  <td className="p-3 font-semibold text-blue-700">
                    {financeAmount(item.output_value)}
                  </td>
                  <td className="p-3">
                    {financeQuantity(item.qualified_quantity)}
                  </td>
                  <td className="p-3">
                    {financeQuantity(item.billable_quantity)}
                  </td>
                  <td className="p-3 text-violet-700">
                    {financeQuantity(item.reserve_quantity)}
                  </td>
                </>
              ) : (
                <>
                  <td className="p-3 font-semibold">
                    {financeAmount(item.output_value)}
                  </td>
                  <td className="p-3">{financeAmount(item.production_cost)}</td>
                  <td className="p-3">{financeAmount(item.period_expense)}</td>
                  <td
                    className={`p-3 font-bold ${Number(item.operating_profit) < 0 ? "text-rose-700" : "text-emerald-700"}`}
                  >
                    {financeAmount(item.operating_profit)}
                  </td>
                  <td className="p-3">
                    {item.closing_status === "closed"
                      ? "已核对并锁定"
                      : "可继续补充"}
                  </td>
                </>
              )}
              <td className="p-3">
                <button
                  type="button"
                  className="erp-btn"
                  onClick={() => onSelect(item.month)}
                >
                  查看本月
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </ResponsiveTable>
    </div>
  );
}
