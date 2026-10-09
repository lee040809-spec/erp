import { authFetch } from '../components/shared';
import type { OrderItem, Order } from '../types';

export interface ProcessSaveNotice { kind: 'saving' | 'success' | 'error'; message: string }
export async function saveProcessStatus(itemId: number, processId: number, status: string): Promise<{ item_status: OrderItem['status']; order_status: Order['status'] }> {
  const response = await authFetch(`/api/platform/order-items/${itemId}/processes/${processId}`, {
    method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ status })
  });
  const result = await response.json().catch(() => ({}));
  if (!response.ok || !result.success) throw new Error(result.error || result.message || '工序状态保存失败');
  return result;
}
