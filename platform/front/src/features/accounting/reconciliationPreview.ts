import type { Order } from "../../types";

export function reconciliationSources(orders: Order[] = []) {
  return new Map(
    orders.flatMap((order) =>
      (order.items || []).map((item) => [Number(item.id), order] as const),
    ),
  );
}
export function reconciliationRowContext(
  header: Record<string, unknown>,
  itemId: number,
  sources: Map<number, Order>,
  multiple: boolean,
) {
  const source = sources.get(Number(itemId));
  if (!source)
    return multiple ? { ...header, order_number: "待核对来源订单" } : header;
  return {
    ...header,
    ...source,
    start_date: source.start_date?.split("T")[0],
    due_date: source.due_date?.split("T")[0],
    other: header.other,
  };
}

// Update only the explicitly labelled reconciliation date in a preview copy.
// The stored workbook, invoice dates, formulas and all other formatting stay intact.
export function reconciliationWorkbook(workbook: any, date?: string) {
  if (!workbook || !date || !/^\d{4}-\d{2}-\d{2}$/.test(date)) return workbook;
  let copy: any;
  const fixedDate = /^\s*\d{4}[-/]\d{1,2}[-/]\d{1,2}/;
  const set = (sheet: string, row: string, col: string, value: string) => {
    copy ||= structuredClone(workbook);
    copy.sheets[sheet].cellData[row][col].v = value;
  };
  for (const [sheet, data] of Object.entries<any>(workbook.sheets || {})) {
    for (const [row, cells] of Object.entries<Record<string, any>>(
      data.cellData || {},
    )) {
      for (const [col, cell] of Object.entries(cells)) {
        if (
          typeof cell?.v !== "string" ||
          cell.f ||
          !cell.v.includes("对账日期")
        )
          continue;
        const inline = /^(\s*对账日期\s*[:：]?\s*)\d{4}[-/]\d{1,2}[-/]\d{1,2}/;
        if (inline.test(cell.v)) {
          set(sheet, row, col, cell.v.replace(inline, (_match: string, prefix: string) => prefix + date));
          continue;
        }
        for (const next of Object.keys(cells)
          .filter((key) => Number(key) > Number(col))
          .sort((a, b) => Number(a) - Number(b))) {
          const candidate = cells[next];
          if (candidate?.v === "" || candidate?.v == null) continue;
          if (
            !candidate.f &&
            typeof candidate.v === "string" &&
            fixedDate.test(candidate.v)
          )
            set(sheet, row, next, candidate.v.replace(fixedDate, date));
          break;
        }
      }
    }
  }
  return copy || workbook;
}
