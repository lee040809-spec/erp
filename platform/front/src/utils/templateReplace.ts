/** Shared field replacement for delivery and reconciliation print templates. */
export const formatDate = (date: unknown): string => {
  if (!date) return '';
  if (typeof date === 'string') return date.split('T')[0];
  if (date instanceof Date) return date.toISOString().split('T')[0];
  return String(date);
};

export const formatAmount = (amount: unknown): string => {
  if (!amount) return '0';
  return Number(amount).toFixed(2);
};

export const getFieldValue = (obj: any, key: string): string => {
  let value = obj;
  for (const part of key.split('.')) {
    if (!value || typeof value !== 'object') return '';
    value = value[part];
  }
  if (value === null || value === undefined) return '';
  if (value instanceof Date) return formatDate(value);
  // Database DECIMAL fields can arrive as strings; format both representations alike.
  if ((typeof value === 'number' || typeof value === 'string') &&
      (key.includes('amount') || key.includes('price')) &&
      String(value).trim() !== '' && Number.isFinite(Number(value))) return formatAmount(value);
  return String(value);
};

const getOtherFieldValue = (orderData: any, fieldKey: string): string => {
  const other = orderData?.other || {}, key = fieldKey.toLowerCase();
  if (key === 'small_total') return other.small_total || '0.00';
  if (key === 'big_total') return other.big_total || '零元整';
  if (key === 'total_quantity') return String(other.total_quantity || 0);
  return String(other[key] ?? '');
};
export const getDeliveryOtherFieldValue = getOtherFieldValue;
export const getReconciliationOtherFieldValue = getOtherFieldValue;

export const replaceItemPlaceholders = (cellValue: string, itemData: any): string =>
  cellValue ? cellValue.replace(/\{\{#items\.([^}]+)\}\}/g, (_match, key: string) => getFieldValue(itemData, key)) : cellValue;

const replaceOrderPlaceholders = (cellValue: string, orderData: any): string =>
  cellValue ? cellValue.replace(/\{\{(order|other)\.([^}]+)\}\}/g,
    (_match, namespace: string, key: string) => namespace === 'order' ? getFieldValue(orderData, key) : getOtherFieldValue(orderData, key)) : cellValue;
export const replaceDeliveryOrderPlaceholders = replaceOrderPlaceholders;
export const replaceReconciliationOrderPlaceholders = replaceOrderPlaceholders;

export const hasItemPlaceholder = (cellValue: string): boolean => !!cellValue && cellValue.includes('{{#items.');
export const hasOrderPlaceholder = (cellValue: string): boolean => !!cellValue && (cellValue.includes('{{order.') || cellValue.includes('{{other.'));
