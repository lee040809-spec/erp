import type { OrderItem, OrderCostTotals } from "../../types";
import { financeAmount } from "./financePresentation";

export function orderItemCosts(item: OrderItem): OrderCostTotals {
  return {
    material: item.material_cost,
    outsourcing:
      item.processes
        ?.filter((p) => p.is_outsourced && !p.deleted_at && p.name !== "送货")
        .reduce((n, p) => n + Number(p.outsourcing_fee || 0), 0) ?? 0,
    tool: item.tool_cost,
    fixture: item.fixture_cost,
    other: item.other_cost,
  };
}
export default function PartCostSummary({
  costs,
  showDirect = true,
  showOutsourcing = true,
}: {
  costs: OrderCostTotals;
  showDirect?: boolean;
  showOutsourcing?: boolean;
}) {
  const values = [
    ...(showDirect
      ? [
          ["材料费", costs.material],
          ["刀具费", costs.tool],
          ["工装费", costs.fixture],
          ["其他费", costs.other],
        ]
      : []),
    ...(showOutsourcing ? [["外协费", costs.outsourcing]] : []),
  ];
  const total = values.reduce((n, [, value]) => n + Number(value || 0), 0);
  return (
    <details
      aria-label="订单填写费用"
      className="mt-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs"
    >
      <summary className="cursor-pointer text-slate-500">
        订单填写费用 · {showDirect && showOutsourcing ? "合计" : "可见小计"}{" "}
        {financeAmount(total)}{" "}
        <span className="text-slate-400">展开明细（参考）</span>
      </summary>
      <div className="flex flex-wrap items-center gap-x-5 gap-y-2 mt-2">
        {values.map(([label, value]) => (
          <span key={String(label)} className="text-slate-500">
            {label}{" "}
            <strong className="font-medium text-slate-800">
              {financeAmount(value)}
            </strong>
          </span>
        ))}
        <span className="text-blue-700 font-semibold">
          {showDirect && showOutsourcing ? "费用合计" : "可见费用小计"}{" "}
          {financeAmount(total)}
        </span>
      </div>
    </details>
  );
}
