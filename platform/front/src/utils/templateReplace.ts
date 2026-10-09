/**
 * 模板数据替换工具
 * 用于送货单和对账单的模板占位符替换
 */

// 格式化日期
export const formatDate = (date: any): string => {
  if (!date) return '';
  if (typeof date === 'string') return date.split('T')[0];
  if (date instanceof Date) return date.toISOString().split('T')[0];
  return String(date);
};

// 格式化金额
export const formatAmount = (amount: any): string => {
  if (!amount) return '0';
  return Number(amount).toFixed(2);
};

// 获取字段值（通用）
export const getFieldValue = (obj: any, key: string): string => {
  const parts = key.split('.');
  let value = obj;
  for (const part of parts) {
    if (value && typeof value === 'object') {
      value = value[part];
    } else {
      return '';
    }
  }
  if (value === null || value === undefined) return '';
  if (typeof value === 'object' && value instanceof Date) return formatDate(value);
  if (typeof value === 'number' && (key.includes('amount') || key.includes('price'))) return formatAmount(value);
  return String(value);
};

// 送货单其他字段处理
export const getDeliveryOtherFieldValue = (orderData: any, fieldKey: string): string => {
  const other = orderData.other || {};
  const key = fieldKey.toLowerCase();
  switch (key) {
    case 'small_total':
      return other.small_total || '0.00';
    case 'big_total':
      return other.big_total || '零元整';
    case 'total_quantity':
      return String(other.total_quantity || 0);
    case 'delivery_number':
      return other.delivery_number || '';
    default:
      return other[key] || '';
  }
};

// 对账单其他字段处理
export const getReconciliationOtherFieldValue = (orderData: any, fieldKey: string): string => {
  const other = orderData.other || {};
  const key = fieldKey.toLowerCase();
  switch (key) {
    case 'small_total':
      return other.small_total || '0.00';
    case 'big_total':
      return other.big_total || '零元整';
    case 'total_quantity':
      return String(other.total_quantity || 0);
    case 'reconciliation_number':
      return other.reconciliation_number || '';
    default:
      return other[key] || '';
  }
};

// 替换零件占位符
export const replaceItemPlaceholders = (cellValue: string, itemData: any): string => {
  if (!cellValue) return cellValue;
  let result = cellValue;
  const regex = /\{\{#items\.([^}]+)\}\}/g;
  let match;
  while ((match = regex.exec(cellValue)) !== null) {
    const fieldKey = match[1];
    const value = getFieldValue(itemData, fieldKey);
    result = result.replace(match[0], value);
  }
  return result;
};

// 送货单占位符替换（订单字段 + 其他字段）
export const replaceDeliveryOrderPlaceholders = (cellValue: string, orderData: any): string => {
  if (!cellValue) return cellValue;
  let result = cellValue;

  // 替换 {{order.xxx}} 格式
  const orderRegex = /\{\{order\.([^}]+)\}\}/g;
  let match;
  while ((match = orderRegex.exec(cellValue)) !== null) {
    const fieldKey = match[1];
    const value = getFieldValue(orderData, fieldKey);
    result = result.replace(match[0], value);
  }

  // 替换 {{other.xxx}} 格式（送货单其他字段）
  const otherRegex = /\{\{other\.([^}]+)\}\}/g;
  while ((match = otherRegex.exec(result)) !== null) {
    const fieldKey = match[1];
    const value = getDeliveryOtherFieldValue(orderData, fieldKey);
    result = result.replace(match[0], value);
  }

  return result;
};

// 对账单占位符替换（订单字段 + 其他字段）
export const replaceReconciliationOrderPlaceholders = (cellValue: string, orderData: any): string => {
  if (!cellValue) return cellValue;
  let result = cellValue;

  // 替换 {{order.xxx}} 格式
  const orderRegex = /\{\{order\.([^}]+)\}\}/g;
  let match;
  while ((match = orderRegex.exec(cellValue)) !== null) {
    const fieldKey = match[1];
    const value = getFieldValue(orderData, fieldKey);
    result = result.replace(match[0], value);
  }

  // 替换 {{other.xxx}} 格式（对账单其他字段）
  const otherRegex = /\{\{other\.([^}]+)\}\}/g;
  while ((match = otherRegex.exec(result)) !== null) {
    const fieldKey = match[1];
    const value = getReconciliationOtherFieldValue(orderData, fieldKey);
    result = result.replace(match[0], value);
  }

  return result;
};

// 检查是否包含零件占位符
export const hasItemPlaceholder = (cellValue: string): boolean => {
  return !!cellValue && cellValue.includes('{{#items.');
};

// 检查是否包含订单占位符
export const hasOrderPlaceholder = (cellValue: string): boolean => {
  return !!cellValue && (cellValue.includes('{{order.') || cellValue.includes('{{other.'));
};