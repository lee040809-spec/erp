import React from 'react';
import type { DeliveryPlan } from '../types/productionFlow';
import { flowDate } from '../services/productionFlow';

export function invalidDeliveryPlan(plans: DeliveryPlan[], quantity: number, production: number, due: string, start?: string) {
  if (!plans.length) return false;
  return plans.length > 50 || plans.reduce((n, b) => n + b.quantity, 0) !== quantity || plans.reduce((n, b) => n + b.production_quantity, 0) !== production ||
    new Set(plans.map(b => b.batch_no.trim())).size !== plans.length || plans.some(b => !b.batch_no.trim() || !Number.isInteger(b.quantity) || b.quantity <= 0 || !Number.isInteger(b.production_quantity) || b.production_quantity < b.quantity || !b.due_date || b.due_date > flowDate(due) || !!start && b.due_date < flowDate(start));
}
export default function DeliveryBatchEditor({ plans, quantity, production, dueDate, startDate, onChange, disabled = false }: { plans: DeliveryPlan[]; quantity: number; production: number; dueDate: string; startDate?: string; onChange: (plans: DeliveryPlan[]) => void; disabled?: boolean }) {
  const update = (index: number, values: Partial<DeliveryPlan>) => onChange(plans.map((b, i) => i === index ? { ...b, ...values } : b));
  const add = () => {
    if (!plans.length) { const first = Math.max(1, Math.floor(quantity / 2)); onChange(quantity > 1 ? [{ batch_no: '首批', quantity: first, production_quantity: first, due_date: flowDate(dueDate), notes: '' }, { batch_no: '余量', quantity: quantity - first, production_quantity: production - first, due_date: flowDate(dueDate), notes: '' }] : [{ batch_no: '首批', quantity, production_quantity: production, due_date: flowDate(dueDate), notes: '' }]); }
    else onChange([...plans, { batch_no: `第${plans.length + 1}批`, quantity: 1, production_quantity: 1, due_date: flowDate(dueDate), notes: '' }]);
  };
  return <section className="rounded-xl border border-blue-100 bg-blue-50/30 p-3 space-y-3">
    <div className="flex flex-wrap gap-2 items-center justify-between"><strong className="text-sm">分批交货计划（选填）</strong><button type="button" disabled={disabled || plans.length >= 50} className="erp-btn text-blue-600 min-h-8 py-1" onClick={add}>{plans.length ? '增加批次' : '设置分批交货'}</button></div>
    <p className="text-xs text-slate-500">共用工序；前道部分完工即可安排后道。各批交货合计 {plans.reduce((n, b) => n + b.quantity, 0)} / {quantity}，投产合计 {plans.reduce((n, b) => n + b.production_quantity, 0)} / {production}。</p>
    {plans.map((b, i) => <div key={b.id || i} className="grid grid-cols-2 md:grid-cols-5 gap-2 items-end">
      <label className="erp-label">批次名称<input className="erp-input" maxLength={40} disabled={disabled} value={b.batch_no} onChange={e => update(i, { batch_no: e.target.value })}/></label>
      <label className="erp-label">要求交货数量<input className="erp-input" type="number" min={1} step={1} disabled={disabled} value={b.quantity} onChange={e => update(i, { quantity: Number(e.target.value), production_quantity: Math.max(Number(e.target.value), b.production_quantity) })}/></label>
      <label className="erp-label">安排投产数量<input className="erp-input" type="number" min={b.quantity} step={1} disabled={disabled} value={b.production_quantity} onChange={e => update(i, { production_quantity: Number(e.target.value) })}/></label>
      <label className="erp-label">本批交期<input className="erp-input" type="date" min={flowDate(startDate)} max={flowDate(dueDate)} disabled={disabled} value={flowDate(b.due_date)} onChange={e => update(i, { due_date: e.target.value })}/></label>
      <button type="button" disabled={disabled} className="erp-btn text-rose-600" onClick={() => onChange(plans.filter((_, n) => n !== i))}>移除此批</button>
    </div>)}
    {!!plans.length && invalidDeliveryPlan(plans, quantity, production, dueDate, startDate) && <p role="alert" className="text-xs text-rose-600">请核对：批次数量合计须与零件数量一致，批次交期不能超过总交期。</p>}
  </section>;
}
