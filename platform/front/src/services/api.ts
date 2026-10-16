import { orderQueryParams, type OrderFilters } from './orderService';
import { authFetch } from '../components/shared';
import { Order, Customer, AdventRule, CompletionListResponse, FinanceSummaryResponse } from '../types';

// 获取订单数据
export const fetchOrdersApi = async (dateType?: string, page?: number, pageSize?: number, status?: string, filters: OrderFilters = {}): Promise<{data: Order[], total: number}> => {
  const params = orderQueryParams({...filters, status: status || filters.status}, page, pageSize, dateType);
  const res = await authFetch(`/api/platform/orders?${params}`);
  const result = await res.json();
  if (!res.ok) throw new Error(result.message || result.error || '查询订单失败');
  return {
    data: Array.isArray(result.data) ? result.data : (Array.isArray(result) ? result : []),
    total: typeof result.total === 'number' ? result.total : 0,
  };
};

// 获取客户数据
export const fetchCustomersApi = async (): Promise<Customer[]> => {
  const res = await authFetch('/api/platform/customers');
  const result = await res.json();
  return Array.isArray(result.data) ? result.data : (Array.isArray(result) ? result : []);
};

export const fetchFinanceSummaryApi = async (year: string): Promise<FinanceSummaryResponse> => {
  const res = await authFetch(`/api/platform/finance/summary?year=${encodeURIComponent(year)}`);
  const result = await res.json();
  if (!res.ok) {
    throw new Error(result.message || result.error || '经营报表加载失败');
  }
  return result.data;
};

export const fetchCompletionsApi = async (itemId: number): Promise<CompletionListResponse> => {
  const res = await authFetch(`/api/platform/order-items/${itemId}/completions`);
  const result = await res.json();
  if (!res.ok) {
    throw new Error(result.message || result.error || '完工记录加载失败');
  }
  return result.data;
};

export const createCompletionApi = async (itemId: number, payload: {
  request_id?: string;
  qualified_quantity: number;
  business_date: string;
  notes?: string;
}): Promise<void> => {
  const res = await authFetch(`/api/platform/order-items/${itemId}/completions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  const result = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(result.message || result.error || '合格完工确认失败');
  }
};

export const createCostEntryApi = async (payload: {
  cost_type: string;
  amount: string;
  tax_amount?: string;
  net_amount?: string;
  business_date: string;
  order_id?: number;
  order_item_id?: number;
  supplier_name?: string;
  notes?: string;
}): Promise<void> => {
  const res = await authFetch('/api/platform/finance/cost-entries', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  const result = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(result.message || result.error || '成本费用保存失败');
  }
};

// 获取规则数据
export const fetchRulesApi = async (filters?: { name?: string }): Promise<AdventRule[]> => {
  const qs = filters ? new URLSearchParams(filters as Record<string, string>).toString() : '';
  const url = qs ? `/api/platform/advent-rules?${qs}` : '/api/platform/advent-rules';
  const res = await authFetch(url);
  const result = await res.json();
  return Array.isArray(result.data) ? result.data : (Array.isArray(result) ? result : []);
};

// 获取工作看板数据（零件 + 订单 + 统计）
export const fetchDashboardDataApiNew = async (page: number = 1, pageSize: number = 20): Promise<{
  items: DashboardItem[];
  orders: OrderInfo[];
  total: number;
  stats: {
    pending_count: number;
    processing_count: number;
    completed_count: number;
    overdue_count: number;
    warning_count: number;
    near_due_count: number;
  }
}> => {
  const [itemsRes, statsRes] = await Promise.all([
    authFetch(`/api/platform/dashboard/items?page=${page}&pageSize=${pageSize}`),
    authFetch('/api/platform/dashboard/stats')
  ]);

  const itemsResult = await itemsRes.json();
  const statsResult = await statsRes.json();

  return {
    items: itemsResult.data?.items || [],
    orders: itemsResult.data?.orders || [],
    total: itemsResult.data?.total || 0,
    stats: statsResult.data || { pending_count: 0, processing_count: 0, completed_count: 0, overdue_count: 0, warning_count: 0, near_due_count: 0 }
  };
};

// 工作看板零件数据类型
export interface DashboardItem {
  delivery?: import('../utils/ganttSchedule').GanttDelivery;
  order_id: number;
  order_number: string;
  customer_short_name: string;
  priority: string;
  item_id: number;
  part_name: string;
  part_number: string;
  quantity: number;
  status: 'pending' | 'processing' | 'completed' | 'delivered';
  start_date: string | null;
  due_date: string | null;
  drawing_data: string;
  processes: Array<{
    id: number;
    order_item_id: number;
    name: string;
    is_outsourced: boolean;
    outsourcing_fee: number;
    status: 'pending' | 'processing' | 'completed';
    sort_order: number;
    planned_start_at?: string | null;
    planned_end_at?: string | null;
    actual_start_at?: string | null;
    actual_end_at?: string | null;
  }>;
}

// 订单简要信息类型
export interface OrderInfo {
  id: number;
  order_number: string;
  customer_name: string;
  customer_short_name: string;
  status: string;
  priority: string;
}

// 获取工作看板零件数据
export const fetchDashboardItemsApi = async (page: number = 1, pageSize: number = 20): Promise<{ items: DashboardItem[]; total: number; page: number; pageSize: number }> => {
  const res = await authFetch(`/api/platform/dashboard/items?page=${page}&pageSize=${pageSize}`);
  const result = await res.json();
  // 返回格式: { code: 0, data: { items: [...], total: X, page: X, pageSize: X } }
  if (result.data) {
    return {
      items: result.data.items || [],
      total: result.data.total || 0,
      page: result.data.page || page,
      pageSize: result.data.pageSize || pageSize
    };
  }
  return { items: [], total: 0, page, pageSize };
};

// 获取看板卡片统计数据
export const fetchDashboardStatsApi = async (): Promise<{
  pending_count: number;
  processing_count: number;
  completed_count: number;
  overdue_count: number;
  warning_count: number;
  near_due_count: number;
}> => {
  const res = await authFetch('/api/platform/dashboard/stats');
  const result = await res.json();
  return result.data || { pending_count: 0, processing_count: 0, completed_count: 0, overdue_count: 0, warning_count: 0, near_due_count: 0 };
};

// 获取库存数据（物料 + 余料）
export const fetchInventoryDataApi = async (): Promise<{ materials: any[]; remnants: any[] }> => {
  const [materialsRes, remnantsRes] = await Promise.all([
    authFetch('/api/platform/materials'),
    authFetch('/api/platform/remnants')
  ]);

  const materialsResult = await materialsRes.json();
  const remnantsResult = await remnantsRes.json();

  const getData = (result: any) => Array.isArray(result.data) ? result.data : (Array.isArray(result) ? result : []);

  return {
    materials: getData(materialsResult),
    remnants: getData(remnantsResult)
  };
};

// 删除订单
export const deleteOrderApi = async (id: number): Promise<void> => {
  const response = await authFetch(`/api/platform/orders/${id}`, {
    method: 'DELETE'
  });
  if (!response.ok) {
    const result = await response.json().catch(() => ({}));
    throw new Error(result.error || result.message || '删除订单失败');
  }
};

// 更新工序状态
export const updateProcessStatusApi = async (itemId: number, processId: number, status: string): Promise<void> => {
  const response = await authFetch(`/api/platform/order-items/${itemId}/processes/${processId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status })
  });
  if (!response.ok) {
    throw new Error(`Server returned ${response.status}`);
  }
};

// 创建客户
export const createCustomerApi = async (customer: { name: string; short_name: string; contacts: { name: string; contact: string }[] }): Promise<void> => {
  const response = await authFetch('/api/platform/customers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(customer)
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || '创建失败');
  }
};

// 更新客户
export const updateCustomerApi = async (id: number, customer: { name: string; short_name: string; contacts: { name: string; contact: string }[] }): Promise<void> => {
  const response = await authFetch(`/api/platform/customers/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(customer)
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || '更新失败');
  }
};

// 删除客户
export const deleteCustomerApi = async (id: number): Promise<void> => {
  const response = await authFetch(`/api/platform/customers/${id}`, { method: 'DELETE' });
  if (!response.ok) {
    throw new Error('删除客户失败');
  }
};

// 创建规则
export const createRuleApi = async (rule: any): Promise<void> => {
  const response = await authFetch('/api/platform/advent-rules', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rule)
  });
  if (!response.ok) {
    throw new Error('创建规则失败');
  }
};

// 更新规则
export const updateRuleApi = async (id: number, rule: any): Promise<void> => {
  const response = await authFetch(`/api/platform/advent-rules/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rule)
  });
  if (!response.ok) {
    throw new Error('更新规则失败');
  }
};

// 删除规则
export const deleteRuleApi = async (id: number): Promise<void> => {
  const response = await authFetch(`/api/platform/advent-rules/${id}`, { method: 'DELETE' });
  if (!response.ok) {
    throw new Error('删除规则失败');
  }
};

// 获取打印模板
export const fetchPrintTemplatesApi = async (name?: string): Promise<any[]> => {
  const params = name ? `?name=${encodeURIComponent(name)}` : '';
  const res = await authFetch(`/api/platform/print-templates${params}`);
  const result = await res.json();
  return result.data || [];
};

// 创建打印模板
export const createPrintTemplateApi = async (template: any): Promise<void> => {
  const response = await authFetch('/api/platform/print-templates', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(template)
  });
  if (!response.ok) {
    throw new Error('创建模板失败');
  }
};

// 更新打印模板
export const updatePrintTemplateApi = async (id: number, template: any): Promise<void> => {
  const response = await authFetch(`/api/platform/print-templates/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(template)
  });
  if (!response.ok) {
    throw new Error('更新模板失败');
  }
};

// 删除打印模板
export const deletePrintTemplateApi = async (id: number): Promise<void> => {
  const response = await authFetch(`/api/platform/print-templates/${id}`, { method: 'DELETE' });
  if (!response.ok) {
    throw new Error('删除模板失败');
  }
};
