export type RecordKind = 'completion' | 'delivery';
export interface RecordedQuantity {
  id: number; quantity: string; business_date: string; status: string; source_type: string; batch_no: string;
}
export interface RecordChange {
  id: number; record_id: number; reason: string; before_quantity: string; after_quantity: string;
  before_date: string; after_date: string; created_by: number; created_at: string;
}
export interface RecordHistory {
  records: RecordedQuantity[]; changes: RecordChange[]; revision: number; can_edit: boolean; has_more: boolean;
}
export function correctionError(quantity: string, date: string, reason: string, today: string) {
  if (!Number.isInteger(Number(quantity)) || Number(quantity) <= 0 || Number(quantity) > 2147483647) return '数量须为大于0的整数';
  if (!date || date > today) return '请填写不晚于今天的实际日期';
  if (!reason.trim()) return '请填写修改原因';
  return '';
}
