import type { Order } from "../../types";

export function reconciliationSelectionError(orders: Order[]) {
  if (!orders.length) return "请先勾选需要核对的订单。";
  const customerKey = (order: Order) =>
    Number(order.customer_id) > 0
      ? `id:${order.customer_id}`
      : `name:${(order.customer_name || "").trim()}`;
  if (orders.some((o) => !Number(o.customer_id) && !o.customer_name?.trim()))
    return "所选订单缺少客户，请先补齐客户信息。";
  if (new Set(orders.map(customerKey)).size !== 1)
    return "一份对账单只能包含同一客户。请先取消其他客户的订单。";
  return "";
}
