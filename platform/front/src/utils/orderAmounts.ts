interface PriceItem { quantity: number; unit_price?: number; }
// Existing order_items DECIMAL(10,2), orders DECIMAL(12,2); calculate in cents.
export function calculateOrderAmounts(items: PriceItem[]): { itemTotals: number[]; total: number; error: string } {
 let totalCents=0,error='';
 const itemTotals=items.map((item,index)=>{
  const price=item.unit_price ?? 0,cents=Math.round(price*100);
  if(!Number.isFinite(price)||price<0||price>99999999.99||Math.abs(price*100-cents)>0.000001){error ||= '第'+(index+1)+'项零件单价须为非负金额，最多两位小数';return NaN;}
  const lineCents=item.quantity*cents;
  if(!Number.isSafeInteger(item.quantity)||item.quantity<0||!Number.isSafeInteger(lineCents)||lineCents>9999999999){error ||= '第'+(index+1)+'项零件总额超出允许范围，请核对数量与单价';return NaN;}
  totalCents+=lineCents;return lineCents/100;
 });
 if(totalCents>999999999999)error ||= '订单总金额超出允许范围，请核对数量与单价';
 return {itemTotals,total:error?NaN:totalCents/100,error};
}
export function canEditOrderPrices(user: {role_type?:string;resources?:{resource_key:string}[]} | null | undefined): boolean {
 return user?.role_type==='admin'||Boolean(user?.resources?.some(resource=>resource.resource_key==='orders'));
}
