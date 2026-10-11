import React, { useEffect, useRef, useState } from 'react';
import { flowDate, flowRequest } from '../services/productionFlow';
import type { FlowData } from '../types/productionFlow';
interface Plan {
    assignee: string;
    equipment: string;
    planned_start: string;
    planned_end: string;
    estimated_hours: string;
    blocked_reason: string;
    notes: string;
}
const empty: Plan = { assignee: '', equipment: '', planned_start: '', planned_end: '', estimated_hours: '0', blocked_reason: '', notes: '' };
export default function PartPlanningPanel({ data, onChanged, onBusyChange }: {
    data: FlowData;
    onChanged: () => Promise<void>;
    onBusyChange: (busy: boolean) => void;
}) {
    const [plan, setPlan] = useState(empty), [canManage, setCanManage] = useState(false), [loading, setLoading] = useState(true), [saving, setSaving] = useState(false), [error, setError] = useState(''), [message, setMessage] = useState('');
    const [production, setProduction] = useState(String(data.flow.item.production_quantity || data.flow.item.quantity));
    const busy = useRef(false);
    useEffect(() => { let active = true; flowRequest<{
        plan: Partial<Plan>;
        can_manage: boolean;
    }>('/production/items/' + data.flow.item.id + '/plan').then(v => { if (active) {
        setPlan({ ...empty, ...v.plan, estimated_hours: v.plan.estimated_hours || '0', planned_start: flowDate(v.plan.planned_start), planned_end: flowDate(v.plan.planned_end) });
        setCanManage(v.can_manage);
    } }).catch(e => { if (active)
        setError(e.message); }).finally(() => { if (active)
        setLoading(false); }); return () => { active = false; }; }, [data.flow.item.id]);
    useEffect(() => { onBusyChange(saving); return () => onBusyChange(false); }, [saving, onBusyChange]);
    const save = async (e: React.FormEvent) => { e.preventDefault(); if (busy.current)
        return; busy.current = true; setSaving(true); setError(''); setMessage(''); try {
        await flowRequest('/production/plans', 'PATCH', { item_ids: [data.flow.item.id], ...plan });
        await onChanged();
        setMessage('生产安排已保存。');
    }
    catch (e) {
        setError((e as Error).message);
    }
    finally {
        busy.current = false;
        setSaving(false);
    } };
    const saveProduction = async () => { if (busy.current)
        return; const value = Number(production); if (!Number.isInteger(value) || value < Number(data.flow.item.quantity)) {
        setError('投产数量必须为整数，且不能小于订单数量');
        return;
    } busy.current = true; setSaving(true); setError(''); try {
        await flowRequest('/production/items/' + data.flow.item.id + '/quantities', 'PATCH', { production_quantity: value, scrap_quantity: Number(data.flow.item.scrap_quantity || 0) });
        await onChanged();
        setMessage('投产数量已保存。');
    }
    catch (e) {
        setError((e as Error).message);
    }
    finally {
        busy.current = false;
        setSaving(false);
    } };
    return <section className="space-y-3"><h4 className="font-semibold">负责人、设备与投产安排</h4>{loading ? <p>读取生产安排…</p> : <><form onSubmit={save} className="space-y-3"><div className="grid grid-cols-2 lg:grid-cols-4 gap-3">{([['assignee', '负责人', 'text'], ['equipment', '设备 / 工位', 'text'], ['planned_start', '计划开始', 'date'], ['planned_end', '计划完工', 'date']] as const).map(([key, label, type]) => <label key={key} className="erp-label">{label}<input className="erp-input" type={type} maxLength={100} disabled={!canManage || saving} value={plan[key]} onChange={e => setPlan({ ...plan, [key]: e.target.value })}/></label>)}</div><label className="erp-label">受阻原因（清空并保存可解除受阻）<input className="erp-input" maxLength={500} disabled={!canManage || saving} value={plan.blocked_reason} onChange={e => setPlan({ ...plan, blocked_reason: e.target.value })}/></label><label className="erp-label">生产备注<input className="erp-input" maxLength={500} disabled={!canManage || saving} value={plan.notes} onChange={e => setPlan({ ...plan, notes: e.target.value })}/></label>{canManage ? <button className="erp-btn erp-primary" disabled={saving}>保存生产安排</button> : <p className="text-sm text-slate-500">负责人、设备和投产调整需要生产工作台操作权限；工序操作使用各自原有权限。</p>}</form>{canManage && <div className="flex items-end gap-3 flex-wrap border-t pt-3"><label className="erp-label">投产数量（可包含备用件）<input className="erp-input" type="number" min={data.flow.item.quantity} step={1} value={production} disabled={saving} onChange={e => setProduction(e.target.value)}/></label><button type="button" disabled={saving} onClick={saveProduction} className="erp-btn">保存投产数量</button></div>}</>}{error && <p role="alert" className="text-rose-600">{error}</p>}{message && <p role="status" className="text-emerald-700">{message}</p>}</section>;
}
