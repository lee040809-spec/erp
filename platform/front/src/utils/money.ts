// Orders are settled in RMB; display amounts without a currency unit.
export function formatAmount(value: number | string | null | undefined): string {
  const amount = Number(value ?? 0);
  if (!Number.isFinite(amount)) return '金额待核对';
  return amount.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

export function formatOrderAmount(order: {total_amount?: number; items?: {currency?: string}[]}): string {
  const currencies = [...new Set((order.items || []).map(item => item.currency || 'CNY'))];
  if (currencies.length > 1) return '多币种：请查看零件明细';
  return formatAmount(order.total_amount);
}
