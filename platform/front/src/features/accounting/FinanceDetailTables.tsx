import ResponsiveTable from "../../components/ResponsiveTable";
import {
  financeCostLabels as labels,
  financeAmount as amount,
  financeQuantity,
  type FinanceMode,
} from "./financePresentation";
import type { Batch, Cost, AllocationRecord } from "./financeTypes";
export default function FinanceDetailTables({
  mode,
  loading,
  batches,
  costs,
  allocationRecords,
}: {
  mode: FinanceMode;
  loading: boolean;
  batches: Batch[];
  costs: Cost[];
  allocationRecords: AllocationRecord[];
}) {
  return (
    <>
      {" "}
      <h4 className="font-semibold">已确认的合格完工记录</h4>
      <div className="erp-table-scroll overflow-x-auto">
        <ResponsiveTable className="w-full min-w-[850px] text-sm">
          <thead className="bg-zinc-50">
            <tr>
              {[
                "记录编号 / 完工日期",
                "订单 / 客户",
                "零件",
                "合格数量",
                "计价件数",
                "备用件数",
                "当批单价",
                "产值",
                "补充说明",
              ].map((x) => (
                <th className="text-left p-3" key={x}>
                  {x}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {!loading &&
              batches.map((b) => (
                <tr key={b.id} className="border-t">
                  <td className="p-3">
                    {b.completion_no}
                    <div>{b.business_date}</div>
                  </td>
                  <td className="p-3">
                    {b.order_number}
                    <div>{b.customer_name}</div>
                  </td>
                  <td className="p-3">{b.part_name}</td>
                  <td className="p-3">
                    {financeQuantity(b.qualified_quantity)}
                  </td>
                  <td className="p-3">
                    {financeQuantity(b.billable_quantity)}
                  </td>
                  <td className="p-3 text-violet-600">
                    {financeQuantity(b.reserve_quantity)}
                  </td>
                  <td className="p-3">{b.unit_price_snapshot}</td>
                  <td className="p-3">{amount(b.output_value)}</td>
                  <td className="p-3">{b.notes}</td>
                </tr>
              ))}
          </tbody>
        </ResponsiveTable>
      </div>
      {!loading && !batches.length && (
        <p className="text-zinc-500 text-sm">该月暂无已确认批次。</p>
      )}
      {mode === "overview" && (
        <details className="rounded-xl border border-slate-200 p-3 space-y-3">
          <summary className="cursor-pointer text-sm font-semibold">
            费用详细记录（需要时展开）
          </summary>
          <h4 className="font-semibold">本月已计入零件的费用</h4>
          <div className="erp-table-scroll overflow-x-auto">
            <ResponsiveTable className="w-full min-w-[850px] text-sm">
              <thead className="bg-zinc-50">
                <tr>
                  {[
                    "费用编号 / 类型",
                    "原费用日期",
                    "对应完工记录",
                    "本月计入金额",
                    "依据",
                    "登记人编号",
                  ].map((x) => (
                    <th className="text-left p-3" key={x}>
                      {x}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {!loading &&
                  allocationRecords.map((a) => (
                    <tr className="border-t" key={a.id}>
                      <td className="p-3">
                        #{a.cost_entry_id} {labels[a.cost_type] || a.cost_type}
                      </td>
                      <td className="p-3">{a.cost_date}</td>
                      <td className="p-3">{a.completion_no}</td>
                      <td className="p-3">{amount(a.allocated_amount)}</td>
                      <td className="p-3">{a.allocation_basis}</td>
                      <td className="p-3">{a.created_by}</td>
                    </tr>
                  ))}
              </tbody>
            </ResponsiveTable>
          </div>
          {!loading && !allocationRecords.length && (
            <p className="text-zinc-500 text-sm">该月暂无已分摊生产成本。</p>
          )}
          <h4 className="font-semibold">
            本月记下的费用（生产费用还需计入对应零件）
          </h4>
          <div className="erp-table-scroll overflow-x-auto">
            <ResponsiveTable className="w-full min-w-[850px] text-sm">
              <thead className="bg-zinc-50">
                <tr>
                  {[
                    "日期",
                    "类型",
                    "费用总金额",
                    "税额",
                    "计入金额",
                    "费用用途",
                    "供应商 / 备注",
                  ].map((x) => (
                    <th className="text-left p-3" key={x}>
                      {x}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {!loading &&
                  costs.map((c) => (
                    <tr className="border-t" key={c.id}>
                      <td className="p-3">{c.business_date.slice(0, 10)}</td>
                      <td className="p-3">
                        {labels[c.cost_type] || c.cost_type}
                      </td>
                      <td className="p-3">{amount(c.amount)}</td>
                      <td className="p-3">{amount(c.tax_amount)}</td>
                      <td className="p-3">{amount(c.net_amount)}</td>
                      <td className="p-3">
                        {c.order_item_id
                          ? "指定零件"
                          : c.order_id
                            ? "指定订单"
                            : "暂未指定"}
                      </td>
                      <td className="p-3">
                        {c.supplier_name}
                        <div>{c.notes}</div>
                      </td>
                    </tr>
                  ))}
              </tbody>
            </ResponsiveTable>
          </div>
          {!loading && !costs.length && (
            <p className="text-zinc-500 text-sm">
              该月暂无费用。零成本不表示成本完整。
            </p>
          )}
        </details>
      )}
    </>
  );
}
