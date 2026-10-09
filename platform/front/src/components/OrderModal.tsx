import DeliveryBatchEditor, { invalidDeliveryPlan } from "./DeliveryBatchEditor";
import { formatAmount } from '../utils/money';
import { calculateOrderAmounts } from '../utils/orderAmounts';
import React, { useMemo, useEffect } from 'react';
import { motion } from 'motion/react';
import { X, Calendar, AlertCircle, Plus, Trash2 } from 'lucide-react';
import { Order, Customer, OrderItem } from '../types';
import ProcessCell from './ProcessCell';
import OrderProcessDates, { invalidOrderProcessDates } from './OrderProcessDates';
import { formatDate, StatusBadge } from './shared';
export interface FormErrors {
    items?: string;
    customer?: string;
    orderDate?: string;
    deliveryDate?: string;
}
interface OrderModalProps {
    show: boolean;
    onClose: () => void;
    newOrder: Partial<Order>;
    setNewOrder: (order: Partial<Order>) => void;
    customers: Customer[];
    customerSearch: string;
    setCustomerSearch: (search: string) => void;
    showCustomerDropdown: boolean;
    setShowCustomerDropdown: (show: boolean) => void;
    customerDropdownRef: React.RefObject<HTMLDivElement>;
    formErrors: FormErrors;
    setFormErrors: (errors: FormErrors) => void;
    onSubmit: (e: React.FormEvent) => void;
    isSaving: boolean;
    hideCostFields?: boolean; // 生产成本和外协费用
    hidePriceFields?: boolean; // 订单价格按订单管理权限独立控制
}
const OrderModal: React.FC<OrderModalProps> = (p) => {
    const { show, onClose, newOrder, setNewOrder, customers, customerSearch, setCustomerSearch, showCustomerDropdown, setShowCustomerDropdown, customerDropdownRef, formErrors, setFormErrors, onSubmit, isSaving, hideCostFields = false, hidePriceFields = false } = p;
    const items = newOrder.items || [];
    const amounts = useMemo(() => calculateOrderAmounts(items), [items]);
    const total = amounts.total;
    useEffect(() => {
        if (show && !hidePriceFields && Number.isFinite(total) && total !== newOrder.total_amount)
            setNewOrder({ ...newOrder, total_amount: total });
    }, [show, hidePriceFields, total, newOrder, setNewOrder]);
    const itemUpdate = (index: number, changes: Partial<OrderItem>) => {
        const updated = items.map((item, n) => n === index ? { ...item, ...changes } : item);
        if (hidePriceFields) { setNewOrder({ ...newOrder, items: updated }); return; }
        const calculated = calculateOrderAmounts(updated);
        setNewOrder({ ...newOrder, total_amount: Number.isFinite(calculated.total) ? calculated.total : undefined,
            items: updated.map((item,n) => ({ ...item, total_price: Number.isFinite(calculated.itemTotals[n]) ? calculated.itemTotals[n] : undefined })) });
    };
    const selectedCustomer = customers.find(c => Number(c.id) === Number(newOrder.customer_id));
    const matches = customers.filter(c => String(c.short_name || c.name || '').toLowerCase().includes(customerSearch.toLowerCase()));
    const add = () => setNewOrder({ ...newOrder, items: [...items, { part_name: '', quantity: 1, production_quantity: 1, unit_price: 0, total_price: 0, status: 'pending', start_date: newOrder.start_date, due_date: newOrder.due_date, supply_mode: 'self_purchased', currency: 'CNY', tax_inclusive: false, processes: [] }] });
    if (!show)
        return null;
    return <div className="erp-overlay z-[100]"><div role="dialog" aria-modal="true" aria-labelledby="order-dialog-title" className="erp-dialog erp-order-dialog flex flex-col overflow-hidden"><header className="p-5 sm:px-7 border-b border-slate-100 flex items-center justify-between"><div><h3 id="order-dialog-title" className="text-xl font-bold">{newOrder.id ? '编辑订单' : '新建订单'}</h3><p className="text-xs text-slate-400 mt-1">{newOrder.order_number || '填写基本信息，再添加零件与生产安排'}</p></div><button aria-label="关闭订单编辑" type="button" disabled={isSaving} onClick={onClose} className="erp-btn"><X size={18}/></button></header>
 <form onSubmit={e => {
            e.preventDefault();
            if (items.some(i => !Number.isInteger(i.quantity) || i.quantity <= 0 || !Number.isInteger(i.production_quantity ?? i.quantity) || (i.production_quantity ?? i.quantity) < i.quantity)) {
                setFormErrors({ ...formErrors, items: '订单数量须为正整数，投产数量须为不小于订单数量的整数' });
                return;
            }
            if (!hidePriceFields && amounts.error) {
                setFormErrors({ ...formErrors, items: amounts.error });
                return;
            }
            if (items.some(i => invalidDeliveryPlan(i.delivery_plan || [], i.quantity, i.production_quantity || i.quantity, i.due_date || "", i.start_date || newOrder.start_date))) { setFormErrors({ ...formErrors, items: "请核对各批交货/投产数量合计和交期" }); return; }
 if (items.some(item => (item.processes || []).some(step => !step.name.trim() || step.name.trim().length > 100))) {
                setFormErrors({ ...formErrors, items: '工序名称须填写，最多100个字' });
                return;
            }
            if (items.some(i => invalidOrderProcessDates(i.processes || []))) {
                setFormErrors({ ...formErrors, items: '工序计划开始和结束日期须一起填写或一起清空，结束日期不能早于开始日期' });
                return;
            }
            onSubmit(e);
        }} className="min-h-0 flex flex-col">
 <div className="overflow-y-auto p-5 sm:p-7 space-y-5">
  {Object.values(formErrors).some(Boolean) && <div role="alert" className="text-sm text-rose-600 bg-rose-50 rounded-xl p-3">{Object.values(formErrors).filter(Boolean).join('；')}</div>}
  <section className="erp-card p-4 bg-slate-50/50"><h4 className="font-semibold text-sm mb-4">订单信息</h4><div className="erp-order-form-fields grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4"><div className="relative" ref={customerDropdownRef}><label className="erp-label">客户 *<input aria-label="选择客户" className="erp-input" placeholder="搜索并选择客户" value={customerSearch || (selectedCustomer?.short_name || selectedCustomer?.name || '')} onFocus={() => setShowCustomerDropdown(true)} onChange={e => {
            setCustomerSearch(e.target.value);
            setShowCustomerDropdown(true);
            if (!e.target.value)
                setNewOrder({ ...newOrder, customer_id: undefined, customer_name: undefined, customer_short_name: undefined, contact_name: undefined });
        }}/></label>{showCustomerDropdown && <div className="absolute top-full w-full z-20 bg-white shadow-xl border border-slate-200 rounded-xl max-h-48 overflow-auto mt-1">{matches.map(c => <button key={c.id} type="button" className="block w-full text-left px-3 py-2.5 text-sm hover:bg-blue-50" onClick={() => { setNewOrder({ ...newOrder, customer_id: c.id, customer_name: c.name, customer_short_name: c.short_name, contact_name: undefined }); setCustomerSearch(''); setShowCustomerDropdown(false); setFormErrors({ ...formErrors, customer: '' }); }}>{c.short_name || c.name}</button>)}{!matches.length && <p className="text-xs p-3 text-slate-400">没有匹配的客户</p>}</div>}</div><label className="erp-label">联系人<select className="erp-input" disabled={!newOrder.customer_id} value={newOrder.contact_name || ''} onChange={e => setNewOrder({ ...newOrder, contact_name: e.target.value || undefined })}><option value="">选择联系人</option>{selectedCustomer?.contacts?.map(c => <option key={c.id || c.name} value={c.name}>{c.name}</option>)}</select></label><label className="erp-label">订单日期 *<input type="date" required className="erp-input" value={newOrder.start_date || ''} onChange={e => { setNewOrder({ ...newOrder, start_date: e.target.value, items: items.map(i => ({ ...i, start_date: e.target.value })) }); setFormErrors({ ...formErrors, orderDate: '' }); }}/></label><label className="erp-label">优先级<select className="erp-input" value={newOrder.priority || 'medium'} onChange={e => setNewOrder({ ...newOrder, priority: e.target.value as Order['priority'] })}><option value="low">较低</option><option value="medium">普通</option><option value="high">紧急</option></select></label><label className="erp-label">订单号<input className="erp-input" placeholder="留空自动生成" value={newOrder.order_number || ''} onChange={e => setNewOrder({ ...newOrder, order_number: e.target.value })}/></label><label className="erp-label">订单名称<input className="erp-input" value={newOrder.order_name || ''} onChange={e => setNewOrder({ ...newOrder, order_name: e.target.value })}/></label><label className="erp-label sm:col-span-2">订单备注<input className="erp-input" value={newOrder.notes || ''} onChange={e => setNewOrder({ ...newOrder, notes: e.target.value })}/></label></div></section>
  {hidePriceFields && <p className="text-xs text-slate-500">当前账号无订单价格维护权限，单价与金额需由有订单管理权限的人员填写。</p>}
  <div className="flex justify-between items-center"><div><h4 className="font-semibold">零件与生产</h4><p className="text-xs text-slate-400 mt-1">投产可多于订单，超出合格件作为备用件、不计产值</p></div><button type="button" onClick={add} className="erp-btn erp-primary"><Plus size={15}/>添加零件</button></div>
  {!newOrder.id && <p className="text-xs text-slate-500 bg-blue-50 rounded-xl px-4 py-3">新建时填写订单数量、投产数量与客户交期；报废、已交货数量和实际完工日期在后续生产、完工与交货时记录。</p>}
  {!items.length && <button type="button" onClick={add} className="erp-card w-full py-10 text-sm text-blue-600 border-dashed">＋ 添加第一个零件</button>}
  {items.map((item, index) => <section key={item.id || index} className="erp-card p-4 space-y-4"><div className="flex justify-between items-center"><span className="text-sm font-semibold text-slate-600">零件 {index + 1} {item.part_name && `· ${item.part_name}`}</span><div className="flex gap-2 items-center"><StatusBadge status={item.status}/><button type="button" aria-label={`删除零件${index + 1}`} className="erp-btn text-rose-500 py-1 min-h-8" onClick={() => setNewOrder({ ...newOrder, items: items.filter((_, n) => n !== index) })}><Trash2 size={14}/></button></div></div><div className="erp-order-form-fields grid grid-cols-2 lg:grid-cols-4 gap-4"><label className="erp-label">零件名称 *<input required className="erp-input" value={item.part_name || ''} onChange={e => itemUpdate(index, { part_name: e.target.value })}/></label><label className="erp-label">零件号 / 图号<input className="erp-input" value={item.part_number || ''} onChange={e => itemUpdate(index, { part_number: e.target.value })}/></label><label className="erp-label">订单数量 *<input required type="number" min="1" max={2147483647} step="1" className="erp-input" value={item.quantity || ''} onChange={e => { const qty = Number(e.target.value); itemUpdate(index, { quantity: qty, production_quantity: Math.max(qty, item.production_quantity || item.quantity || qty) }); }}/></label><label className="erp-label">投产数量 *<input required type="number" min={item.quantity || 1} max={2147483647} step="1" className="erp-input" value={item.production_quantity ?? item.quantity ?? ''} onChange={e => itemUpdate(index, { production_quantity: Number(e.target.value) })}/></label><label className="erp-label">客户交期 *<input required type="date" className="erp-input" value={item.due_date || ''} onChange={e => { itemUpdate(index, { due_date: e.target.value }); setFormErrors({ ...formErrors, deliveryDate: '' }); }}/></label><div className="lg:col-span-3"><span className="erp-label mb-2">工序流程</span><ProcessCell processes={item.processes || []} onUpdate={processes => itemUpdate(index, { processes })} hideCostFields={hideCostFields} routeOnly={!!item.id}/></div></div>
  {!hidePriceFields && <div className="erp-order-form-fields grid grid-cols-2 gap-4 rounded-xl bg-blue-50/50 p-3"><label className="erp-label">零件单价<input aria-label={'零件' + (index + 1) + '单价'} type="number" min="0" max="99999999.99" step="0.01" required className="erp-input" value={item.unit_price ?? 0} onChange={e => itemUpdate(index, { unit_price: Number(e.target.value) })}/></label><label className="erp-label">零件总额<input aria-label={'零件' + (index + 1) + '总额'} readOnly className="erp-input bg-slate-50" value={formatAmount(amounts.itemTotals[index])}/></label></div>}
  {!item.id && <DeliveryBatchEditor plans={item.delivery_plan || []} quantity={item.quantity} production={item.production_quantity || item.quantity} dueDate={item.due_date || ""} startDate={item.start_date || newOrder.start_date} disabled={isSaving} onChange={delivery_plan => itemUpdate(index, { delivery_plan })}/>}
 {item.id && item.flow_tracking && <p className="text-xs text-blue-700">已启用数量跟踪，工序数量和交货批次请在订单列表的“分批生产”维护；此处可编辑工序计划日期。</p>}
 <OrderProcessDates processes={item.processes || []} itemIndex={index} disabled={isSaving} onUpdate={processes => itemUpdate(index, { processes })}/>
  {!hideCostFields && <div className="erp-order-form-fields grid grid-cols-2 lg:grid-cols-4 gap-4"><label className="erp-label">材料来源<select className="erp-input" value={item.supply_mode || 'self_purchased'} onChange={e => itemUpdate(index, { supply_mode: e.target.value as OrderItem['supply_mode'] })}><option value="self_purchased">自购材料</option><option value="customer_supplied">客户来料</option></select></label><label className="flex items-center gap-2 text-sm text-slate-500 pt-5"><input type="checkbox" checked={item.tax_inclusive || false} onChange={e => itemUpdate(index, { tax_inclusive: e.target.checked })}/>单价含税</label></div>}
  <details className="rounded-xl bg-slate-50/70 border border-slate-100"><summary className="text-xs text-slate-500 p-3 cursor-pointer">{newOrder.id ? '报废、交货与补充信息' : '零件备注与补充信息'}{!hideCostFields ? ' / 费用' : ''}</summary><div className="p-3 pt-0 grid grid-cols-2 lg:grid-cols-4 gap-4">{newOrder.id && <><div className="erp-label">累计报废<p className="erp-input bg-slate-50">{item.scrap_quantity || 0}</p></div><div className="erp-label">累计已交货<p className="erp-input bg-slate-50">{item.delivered_quantity || 0}</p></div><div className="erp-label">订单数量完成日期<p className="erp-input bg-slate-50">{formatDate(item.completion_date) || '待报工'}</p></div><p className="col-span-2 lg:col-span-4 text-xs text-slate-500">以上信息由业务记录汇总；报废、完工、交货或纠错请从订单的“业务记录 / 登记异常”入口操作。</p></>}<label className="erp-label">零件备注<input className="erp-input" value={item.item_notes || ''} onChange={e => itemUpdate(index, { item_notes: e.target.value })}/></label>{!hideCostFields && <>{(['tool_cost', 'fixture_cost', 'material_cost', 'other_cost'] as const).map((key, i) => <label key={key} className="erp-label">{['刀具费用', '工装费用', '材料费用', '其它费用'][i]}<input type="number" min="0" step="0.01" className="erp-input" value={item[key] ?? 0} onChange={e => itemUpdate(index, { [key]: Number(e.target.value) })}/></label>)}<p className="col-span-2 text-xs text-slate-400">外协共计 {formatAmount((item.processes || []).reduce((n, i) => n + Number(i.outsourcing_fee || 0), 0))} · 在工序管理中维护</p></>}</div></details>
  </section>)}
 </div><footer className="border-t border-slate-100 p-4 sm:px-7 flex justify-between items-center gap-3 bg-white"><p className="text-xs text-slate-400">{items.length} 项零件{!hidePriceFields && <span className="ml-3 text-slate-700 text-sm font-semibold">订单总金额 {formatAmount(total)}</span>}</p><div className="flex gap-2"><button type="button" onClick={onClose} disabled={isSaving} className="erp-btn">取消</button><button type="submit" disabled={isSaving} className="erp-btn erp-primary">{isSaving ? '保存中…' : '保存订单'}</button></div></footer>
 </form></div></div>;
};
export default OrderModal;
