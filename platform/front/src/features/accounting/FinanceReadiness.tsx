import type { FinanceSummaryResponse } from "../../types";
export default function FinanceReadiness({
  summary,
  month,
}: {
  summary: FinanceSummaryResponse;
  month: string;
}) {
  const row = summary.months.find((r) => r.month === month),
    closed = row?.closing_status === "closed";
  return (
    <aside
      className={`erp-card p-4 text-sm ${closed ? "border-emerald-200 bg-emerald-50" : "border-amber-200 bg-amber-50"}`}
      aria-label="本月数据核对状态"
    >
      <strong>
        {month} · {closed ? "费用已核对并锁定" : "费用未核对，结余为暂估"}
      </strong>
      <p className="mt-1">
        {closed
          ? "该月已确认费用完整，后续更正请由管理员处理。"
          : "请核对材料、外协、刀具及其他实际费用是否记全，再核对并锁定月份。未填费用不能视为没有费用。"}
      </p>
      <ul className="mt-2 flex flex-wrap gap-x-6 gap-y-1">
        <li>
          本月待补价：{row?.pending_pricing_count || 0} 批
          {row?.pending_pricing_count
            ? `（${row.pending_pricing_quantity} 件，尚未计产值）`
            : ""}
        </li>
        <li>
          全公司待分配费用：{summary.pending_cost_count || 0} 笔（含其他年份）
        </li>
        <li>订单填写费用仅作参考，结余沿用实际费用记录</li>
      </ul>
    </aside>
  );
}
