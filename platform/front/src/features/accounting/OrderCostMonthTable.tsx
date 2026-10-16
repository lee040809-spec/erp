import type { MonthlyFinanceSummary } from "../../types";
import ResponsiveTable from "../../components/ResponsiveTable";
import { financeAmount } from "./financePresentation";

export default function OrderCostMonthTable({
  months,
}: {
  months: MonthlyFinanceSummary[];
}) {
  const headers = [
    "下单月份",
    "材料费",
    "外协费",
    "刀具费",
    "工装费",
    "其他费",
    "费用合计",
  ];
  return (
    <section className="erp-card overflow-hidden">
      <header className="px-4 py-3 border-b">
        <h3 className="font-semibold">订单填写费用</h3>
        <p className="text-xs text-slate-500 mt-1">
          按零件下单月份汇总；未填日期时沿用订单日期。这里只展示订单中的费用，不重复扣减预计结余。费用为零也可能是尚未填写。
        </p>
      </header>
      <div className="erp-table-scroll overflow-auto">
        <ResponsiveTable className="w-full text-sm min-w-[650px]">
          <thead>
            <tr>
              {headers.map((h) => (
                <th key={h} className="p-3 text-left">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {months.map((m) => {
              const c = m.order_costs,
                values = [
                  c?.material,
                  c?.outsourcing,
                  c?.tool,
                  c?.fixture,
                  c?.other,
                ];
              return (
                <tr key={m.month} className="border-t">
                  <td className="p-3">{m.month}</td>
                  {values.map((v, i) => (
                    <td key={i} className="p-3">
                      {financeAmount(v)}
                    </td>
                  ))}
                  <td className="p-3 font-semibold text-blue-700">
                    {financeAmount(
                      values.reduce<number>((n, v) => n + Number(v || 0), 0),
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </ResponsiveTable>
      </div>
    </section>
  );
}
