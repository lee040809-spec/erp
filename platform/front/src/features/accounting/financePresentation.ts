import type { FinanceSummaryResponse } from "../../types";

export type FinanceMode = "overview" | "output";
export const financeQuantity = (value: string | number) =>
  Number(value || 0).toLocaleString("zh-CN", { maximumFractionDigits: 3 });
export function defaultFinanceMonth(year: string, today = new Date()) {
  return `${year}-${year === String(today.getFullYear()) ? String(today.getMonth() + 1).padStart(2, "0") : "12"}`;
}
export const financeAmount = (value: string | number | undefined) =>
  Number(value || 0).toLocaleString("zh-CN", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
export function financeCards(
  summary: FinanceSummaryResponse,
  mode: FinanceMode,
) {
  if (mode === "output")
    return [
      {
        label: "全年合格件数",
        value: summary.months
          .reduce((n, m) => n + Number(m.qualified_quantity), 0)
          .toLocaleString("zh-CN"),
        hint: "包括订单内零件和备用件",
      },
      {
        label: "按订单计价件数",
        value: summary.months
          .reduce((n, m) => n + Number(m.billable_quantity), 0)
          .toLocaleString("zh-CN"),
        hint: "只计算订单约定数量以内的合格件",
      },
      {
        label: "备用件数",
        value: summary.months
          .reduce((n, m) => n + Number(m.reserve_quantity), 0)
          .toLocaleString("zh-CN"),
        hint: "超出订单的合格件，不计入完成金额",
      },
      {
        label: "全年完成金额",
        value: financeAmount(summary.annual.output_value),
        hint: "按合格完工日期统计，单位：元",
      },
    ];
  return [
    {
      label: "全年完成金额",
      value: financeAmount(summary.annual.output_value),
      hint: "订单内合格数量 × 当批确认单价",
    },
    {
      label: "已计入的生产花费",
      value: financeAmount(summary.annual.production_cost),
      hint: "已分配到完工记录的材料、外协等费用",
    },
    {
      label: "其他经营花费",
      value: financeAmount(summary.annual.period_expense),
      hint: "按发生月份记录的日常经营费用",
    },
    {
      label: "预计结余",
      value: financeAmount(summary.annual.operating_profit),
      hint: "完成金额减去以上两类花费；费用未录全时为暂估",
    },
  ];
}

export const financeCostLabels: Record<string, string> = {
  material: "材料",
  outsourcing: "外协",
  tool: "刀具",
  fixture: "工装",
  labor: "人工",
  manufacturing: "水电等生产费用",
  other_direct: "其他生产费用",
  period: "其他经营费用",
};
