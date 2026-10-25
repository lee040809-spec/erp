import type { Detail } from "./deliveryReconciliationTypes";
import ResponsiveTable from "../../components/ResponsiveTable";
import { financeAmount } from "./financePresentation";
function exportDetail(detail: Detail) {
  // Treat all text as spreadsheet text, including names starting with formula characters.
  const text = (v: unknown) =>
    '"' +
    String(v ?? "")
      .replace(/^[=+@-]/, "'$&")
      .replace(/"/g, '""') +
    '"';
  const rows = [
    [
      "对账单",
      detail.header.statement_no,
      "对账日期",
      detail.header.business_date.slice(0, 10),
    ],
    ["送货日期", "订单号", "零件", "图号", "本次交货数量", "核对单价", "金额"],
    ...detail.lines.map((r) => [
      r.business_date.slice(0, 10),
      r.order_number,
      r.part_name,
      r.part_number,
      r.quantity,
      r.unit_price,
      r.amount,
    ]),
  ];
  const url = URL.createObjectURL(
    new Blob(["\ufeff" + rows.map((r) => r.map(text).join(",")).join("\r\n")], {
      type: "text/csv;charset=utf-8",
    }),
  );
  const link = document.createElement("a");
  link.href = url;
  link.download = detail.header.statement_no + ".csv";
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export default function DeliveryStatementDetail({
  detail,
  onClose,
}: {
  detail: Detail;
  onClose: () => void;
}) {
  return (
    <section
      className="border rounded-xl p-3 space-y-2"
      aria-label="已保存对账明细"
    >
      <strong>
        {detail.header.statement_no} ·{" "}
        {detail.header.status === "confirmed" ? "已对账" : "已撤销"} ·{" "}
        {financeAmount(detail.header.amount)}
      </strong>
      <button className="erp-btn ml-2" onClick={() => exportDetail(detail)}>
        导出明细
      </button>
      <button className="erp-btn" onClick={onClose}>
        收起
      </button>
      <div className="overflow-auto">
        <ResponsiveTable className="w-full text-sm">
          <thead>
            <tr>
              {["送货日期", "订单", "零件", "数量", "核对单价", "金额"].map(
                (s) => (
                  <th key={s} className="p-2 text-left">
                    {s}
                  </th>
                ),
              )}
            </tr>
          </thead>
          <tbody>
            {detail.lines.map((r, i) => (
              <tr key={i} className="border-t">
                <td className="p-2">{r.business_date.slice(0, 10)}</td>
                <td className="p-2">{r.order_number}</td>
                <td className="p-2">{r.part_name}</td>
                <td className="p-2">{r.quantity}</td>
                <td className="p-2">{r.unit_price}</td>
                <td className="p-2">{financeAmount(r.amount)}</td>
              </tr>
            ))}
          </tbody>
        </ResponsiveTable>
      </div>
      {detail.header.reversal_reason && (
        <p>撤销原因：{detail.header.reversal_reason}</p>
      )}
    </section>
  );
}
