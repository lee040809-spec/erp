import { Order } from '../types';

// Order filters interface
export interface OrderFilters {
  dueDateStart?: string;
  dueDateEnd?: string;
  orderNumber?: string;
  partNumber?: string;
  partName?: string;
  customerName?: string;
  priority?: string;
  status?: string;
}

export function orderQueryParams(filters: OrderFilters = {}, page?: number, pageSize?: number, dateType?: string): URLSearchParams {
  const params = new URLSearchParams();
  for (const key of ['dueDateStart', 'dueDateEnd', 'orderNumber', 'partNumber', 'customerName', 'priority', 'status'] as const) {
    if (filters[key]) params.set(key, filters[key]!);
  }
  if (filters.partName?.trim()) params.set('partName', filters.partName.trim());
  if (page) params.set('page', String(page));
  if (pageSize) params.set('pageSize', String(pageSize));
  if (dateType) params.set('dateType', dateType);
  return params;
}

// Filter orders locally (client-side filtering)
export const filterOrdersLocal = (
  orders: Order[],
  filters: OrderFilters,
  getOrderMaxDueDate: (order: Order) => string
): Order[] => {
  const orderNumberFilter = String(filters?.orderNumber ?? '').trim().toLowerCase();
  const customerFilter = String(filters?.customerName ?? '').trim().toLowerCase();
  const partNameFilter = String(filters?.partName ?? '').trim().toLowerCase();
  const partNumberFilter = String(filters?.partNumber ?? '').trim().toLowerCase();

  return (Array.isArray(orders) ? orders : []).filter(o => {
    const maxDueDate = getOrderMaxDueDate(o);
    const matchDueDateStart = !filters.dueDateStart || maxDueDate >= filters.dueDateStart;
    const matchDueDateEnd = !filters.dueDateEnd || maxDueDate <= filters.dueDateEnd;
    const matchOrderNumber = !orderNumberFilter || String(o.order_number ?? o.id ?? '').toLowerCase().includes(orderNumberFilter);
    const matchCustomer = !customerFilter || String(o.customer_name ?? '').toLowerCase().includes(customerFilter);
    const matchPriority = !filters.priority || filters.priority === '' || o.priority === filters.priority;
    const matchPartNumber = (!partNumberFilter && !partNameFilter) || (o.items || []).some(item => (!partNumberFilter || String(item.part_number ?? '').toLowerCase().includes(partNumberFilter)) && (!partNameFilter || String(item.part_name ?? '').toLowerCase().includes(partNameFilter)));
    const matchStatus = !filters.status || filters.status === '' || o.status === filters.status;

    return matchDueDateStart && matchDueDateEnd && matchOrderNumber && matchCustomer && matchPriority && matchPartNumber && matchStatus;
  });
};
