export interface DeliveryQuantities {
  quantity: number;
  qualified_quantity: number;
  delivered_quantity: number;
  production_quantity?: number;
  scrap_quantity?: number;
  available_delivery?: number;
}
export function deliveryEntryState(item?: DeliveryQuantities, batch?: DeliveryQuantities, requiresBatch = false) {
  if (!item) return { limit: 0, needsCompletion: false, message: '请先选择零件。' };
  if (requiresBatch && !batch) return { limit: 0, needsCompletion: false, message: '请先选择本次交货所属的计划批次。' };
  const scope = batch || item;
  const ordered = Number(scope.quantity) || 0, delivered = Number(scope.delivered_quantity) || 0;
  const qualified = Number(scope.qualified_quantity) || 0;
  const remaining = Math.max(0, ordered - delivered);
  const itemLimit = Math.max(0, Math.min(Number(item.quantity), Number(item.qualified_quantity)) - Number(item.delivered_quantity));
  const limit = Math.max(0, Math.min(remaining, qualified - delivered, itemLimit, item.available_delivery ?? Infinity));
  const possibleQualified = Math.max(0, Number(scope.production_quantity ?? scope.quantity) - Number(scope.scrap_quantity || 0));
  const needsCompletion = remaining > 0 && qualified < Math.min(ordered, possibleQualified);
  const subject = batch ? '本批' : '该零件';
  const message = remaining === 0 ? subject + '已全部登记交货，无需重复登记；日期或数量有误时，可在“办理零件 → 登记交货”的已登记记录中修改。'
    : limit === 0 && needsCompletion ? subject + '尚无未交的已确认合格数量。生产工序完成后，还需确认合格完工数量，才能登记交货。'
    : limit === 0 ? subject + '当前可交数量为 0，请核对合格完工及交货记录。'
    : '本次最多可交 ' + limit + ' 件，超过的部分请先确认合格完工，或核对已有交货记录。';
  return { limit, needsCompletion, message };
}
