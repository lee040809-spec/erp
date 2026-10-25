import type { FinanceMode } from "./financePresentation";

export default function AccountingGuide({
  mode,
}: {
  mode: FinanceMode | "reconciliation";
}) {
  const instructions = {
    overview: [
      "看经营结果",
      "选择年份查看每月完成金额、花费和预计结余；单零件费用可同次登记并计入，涉及多个零件时再分配。",
    ],
    output: [
      "看完成了多少",
      "平时在“办理零件”确认合格数量；已补价部分按完工月份计产值，待补价部分先保留数量。漏记旧记录才需要补录，已确认的不要重复填写。",
    ],
    reconciliation: [
      "和客户核对送货或订单金额",
      "分批交货可按送货记录保存对账单和已对账标记；整单模式可跨页选择同一客户订单。切换客户前请先清空已选清单。",
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
