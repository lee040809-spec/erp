import type { FinanceMode } from "./financePresentation";

export default function AccountingGuide({
  mode,
}: {
  mode: FinanceMode | "reconciliation";
}) {
  const instructions = {
    overview: [
      "看经营结果",
      "选择年份看每月完成金额、花费和预计结余；缺少费用时，先记费用再分配到零件。",
    ],
    output: [
      "看完成了多少",
      "平时在“办理零件”确认合格数量后自动统计；漏记的旧记录才需要补录，已确认的不要重复填写。",
    ],
    reconciliation: [
      "和客户核对订单金额",
      "先查询客户，再勾选同一客户的订单，最后预览或导出对账单。翻页、重新查询会清空已选。",
    ],
  }[mode];
  return (
    <aside className="rounded-xl border border-blue-100 bg-blue-50 px-4 py-3 text-sm text-blue-900">
      <strong>{instructions[0]}</strong>
      <p className="mt-1 leading-6">{instructions[1]}</p>
      <p className="text-xs mt-1 text-blue-700">
        完成金额、对账金额都不等于已收款；预计结余按已录入并计入的费用估算。
      </p>
    </aside>
  );
}
