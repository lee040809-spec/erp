import type { Order, OrderItem } from "../../types";
import { businessToday } from "../../utils/businessDate";
import { processDate } from "../../utils/processDates";
import { calculateOrderAmounts } from "../../utils/orderAmounts";

export function changeOrderDate(
  order: Partial<Order>,
  date: string,
): Partial<Order> {
  const previous = processDate(order.start_date);
  return {
    ...order,
    start_date: date,
    items: (order.items || []).map((item) =>
      !processDate(item.start_date) || processDate(item.start_date) === previous
        ? { ...item, start_date: date }
        : item,
    ),
  };
}

// Whitelist template fields: never copy IDs, production facts, files or ledger links.
export function copyPart(
  item: OrderItem,
  startDate: string,
  dueDate = "",
): OrderItem {
  return {
    part_name: item.part_name,
    part_number: item.part_number,
    quantity: item.quantity,
    production_quantity: Math.max(
      item.quantity,
      item.production_quantity || item.quantity,
    ),
    unit_price: item.unit_price,
    total_price: calculateOrderAmounts([item]).itemTotals[0],
    supply_mode: item.supply_mode || "self_purchased",
    currency: "CNY",
    tax_inclusive: item.tax_inclusive,
    status: "pending",
    start_date: startDate,
    due_date: dueDate,
    item_notes: item.item_notes,
    processes: (item.processes || [])
      .filter((p) => !p.deleted_at)
      .map((p, index) => ({
        name: p.name,
        is_outsourced: p.is_outsourced,
        cutting_spec: p.cutting_spec,
        status: "pending",
        sort_order: index,
        outsourcing_fee: 0,
        planned_start_at: "",
        planned_end_at: "",
      })),
  };
}

export function copyOrder(order: Partial<Order>): Partial<Order> {
  const today = businessToday();
  return {
    customer_id: order.customer_id,
    customer_name: order.customer_name,
    customer_short_name: order.customer_short_name,
    contact_name: order.contact_name,
    priority: order.priority || "medium",
    status: "pending",
    start_date: today,
    due_date: "",
    order_number: "",
    order_name: order.order_name,
    notes: order.notes,
    items: (order.items || []).map((item) => copyPart(item, today)),
  };
}
