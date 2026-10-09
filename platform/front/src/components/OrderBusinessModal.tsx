import { businessToday } from '../utils/businessDate';
import ProductionFlowModal from "./ProductionFlowModal";
 import { flowRequest } from "../services/productionFlow";
 import type { FlowData } from "../types/productionFlow";
import React, { useEffect, useRef, useState } from 'react';
import { authFetch, formatDate } from './shared';
import { Order } from '../types';
import { DashboardItem } from '../services/api';
import CompletionModal from './CompletionModal';
import DeliveryPreviewModal from './DeliveryPreviewModal';
import { EvidencePolicy, basicEvidence } from './BusinessEntrySettings';
type Kind = 'scrap' | 'delivery' | 'exception' | 'records' | 'completion';
interface Item {
    id: number;
    part_name: string;
    part_number?: string;
    quantity: number;
    production_quantity: number;
    scrap_quantity: number;
    qualified_quantity: number;
    delivered_quantity: number;
    available_delivery: number;
    delivery_date?: string;
    completion_date?: string;
}
interface RecordRow {
    id: number;
    order_item_id?: number;
    kind: string;
    quantity: number;
    business_date?: string;
    process_name: string;
    category: string;
    severity: string;
    description: string;
    notes: string;
    status: string;
    source_type: string;
    resolution: string;
    created_by: number;
}
interface Data {
    has_more: boolean;
    page: number;
    items: Item[];
    records: RecordRow[];
    can_complete: boolean;
    completions: {
        id: number;
        order_item_id: number;
        business_date: string;
        qualified_quantity: number;
        billable_quantity: number;
        status: string;
    }[];
    can_scrap: boolean;
    can_delivery: boolean;
    can_exception: boolean;
    can_reverse: boolean;
    evidence: EvidencePolicy;
}

async function req<T>(url: string, payload?: unknown) {
    const res = await authFetch(url, payload ? { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) } : undefined);
    const body = await res.json();
    if (!res.ok)
        throw new Error(body.message || body.error || '操作失败');
    return body.data as T;
}
export default function OrderBusinessModal({ order, initialKind = 'records', initialItem, onClose, onChanged }: {
    order: Order;
    initialKind?: Kind;
    initialItem?: number;
    onClose: () => void;
    onChanged: () => void;
}) {
    const [kind, setKind] = useState<Kind>(initialKind), [data, setData] = useState<Data | null>(null), [itemID, setItemID] = useState(initialItem ? String(initialItem) : ''), [quantity, setQuantity] = useState(''), [date, setDate] = useState(businessToday()), [process, setProcess] = useState(''), [category, setCategory] = useState('质量异常'), [severity, setSeverity] = useState('normal'), [description, setDescription] = useState(''), [notes, setNotes] = useState(''), [error, setError] = useState(''), [message, setMessage] = useState(''), [saving, setSaving] = useState(false), [loading, setLoading] = useState(true), [resolution, setResolution] = useState<{
        row: RecordRow;
        mode: 'resolve' | 'reverse';
    } | null>(null), [reason, setReason] = useState(''), [completion, setCompletion] = useState<DashboardItem | null>(null), [preview, setPreview] = useState<{
        template: any;
        orderData: any;
    } | null>(null);
    const [batch,setBatch]=useState("");
 const [itemFlow,setItemFlow]=useState<FlowData|null>(null);
 const [showFlow,setShowFlow]=useState(false);
 useEffect(()=>{let active=true;setBatch("");setItemFlow(null);if(itemID)flowRequest<FlowData>(`/production/items/${itemID}/flow`).then(v=>{if(active)setItemFlow(v);}).catch(e=>{if(active)setError(e.message);});return()=>{active=false;};},[itemID]);
 const [recordPage,setRecordPage]=useState(1);
    const busy = useRef(false), pending = useRef({ signature: '', id: '' });
    const load = async () => {
        setLoading(true);
        try {
            const result = await req<Data>(`/api/platform/orders/${order.id}/business-records?page=${recordPage}&pageSize=50`);
            setData(result);
            if (!itemID && result.items.length && initialKind !== 'exception')
                setItemID(String(result.items[0].id));
        }
        catch (e) {
            setError((e as Error).message);
        }
        finally {
            setLoading(false);
        }
    };
    useEffect(() => { load(); }, [order.id,recordPage]);
    const item = data?.items.find(i => i.id === Number(itemID));
    const policy = data?.evidence || basicEvidence;
    const changeKind = (next: Kind) => {
        setKind(next);
        if (next === 'exception' && !initialItem)
            setItemID('');
        else if (!itemID && data?.items.length)
            setItemID(String(data.items[0].id));
        setError('');
        setMessage('');
        setQuantity('');
        setDescription('');
        setNotes('');
    };
    const submit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (busy.current)
            return;
        busy.current = true;
        setSaving(true);
        setError('');
        setMessage('');
        try {
            const payload = { delivery_batch_id: kind === "delivery" && batch ? Number(batch) : undefined, kind, order_item_id: itemID ? Number(itemID) : undefined, quantity: kind === 'exception' ? 0 : Number(quantity), business_date: date, process_name: kind === 'scrap' ? process : '', category: kind === 'exception' ? category : '', severity, description: kind === 'exception' ? description.trim() : '', notes: notes.trim() };
            const signature = JSON.stringify(payload);
            if (pending.current.signature !== signature)
                pending.current = { signature, id: crypto.randomUUID() };
            await req(`/api/platform/orders/${order.id}/business-records`, { ...payload, request_id: pending.current.id });
            setQuantity('');
            setDescription('');
            setNotes('');
            pending.current = { signature: '', id: '' };
            setMessage('登记成功，数量和记录已更新');
            await load();
            onChanged();
            setKind('records');
        }
        catch (e) {
            setError((e as Error).message);
        }
        finally {
            busy.current = false;
            setSaving(false);
        }
    };
    const updateRecord = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!resolution || busy.current)
            return;
        busy.current = true;
        setSaving(true);
        setError('');
        try {
            await req(`/api/platform/business-records/${resolution.row.id}/${resolution.mode}`, resolution.mode === 'resolve' ? { resolution: reason.trim() } : { reason: reason.trim() });
            setResolution(null);
            setReason('');
            await load();
            onChanged();
            setMessage('记录已更新，原记录保留');
        }
        catch (e) {
            setError((e as Error).message);
        }
        finally {
            busy.current = false;
            setSaving(false);
        }
    };
    const print = async (row: RecordRow) => {
        setError('');
        try {
            const saved = await req<{
                order: any;
                delivery_date?: string;
            }>(`/api/platform/business-records/${row.id}/delivery-print`);
            const res = await authFetch(`/api/platform/print-templates/by-button?menu_route=production_delivery&button_key=btn-config-delivery-note&mode=preview&order_id=${order.id}`);
            const body = await res.json();
            if (!res.ok || !body.data)
                throw new Error(body.message || body.error || '请先绑定送货单模板');
            setPreview({ template: body.data, orderData: { ...saved.order, customer_due_date: saved.order.due_date, due_date: formatDate(saved.delivery_date), delivery_date: formatDate(saved.delivery_date) } });
        }
        catch (e) {
            setError((e as Error).message);
        }
    };
    return <div className="erp-overlay z-[105]"><section role="dialog" aria-label="订单业务登记与异常" className="erp-dialog erp-order-dialog flex flex-col overflow-hidden"><header className="p-5 border-b flex justify-between gap-3"><div><h3 className="text-xl font-bold">业务登记与异常</h3><p className="text-sm text-slate-500 mt-1">{order.order_number} · {order.customer_short_name || order.customer_name}</p></div><button type="button" disabled={saving} onClick={onClose} className="erp-btn">关闭</button></header><div className="p-5 overflow-y-auto space-y-5"><nav className="flex gap-2 flex-wrap">{([['records', '发生记录'], ['scrap', '登记报废'], ['completion', '确认合格完工'], ['delivery', '登记交货'], ['exception', '登记异常']] as [
            Kind,
            string
        ][]).filter(([key]) => key === 'records' || key === 'completion' && data?.can_complete || key === 'scrap' && data?.can_scrap || key === 'delivery' && data?.can_delivery || key === 'exception' && data?.can_exception).map(([key, label]) => <button type="button" key={key} disabled={saving} className={`erp-btn ${kind === key ? 'erp-primary' : ''}`} onClick={() => changeKind(key)}>{label}</button>)}</nav>{loading && <p>读取业务记录…</p>}{error && <p role="alert" className="text-rose-600 bg-rose-50 p-3 rounded-xl">{error}</p>}{message && <p role="status" className="text-emerald-700 bg-emerald-50 p-3 rounded-xl">{message}</p>}
 {itemFlow?.flow.tracking && <div className="text-xs text-blue-600"><button type="button" className="erp-btn text-blue-600" onClick={()=>setShowFlow(true)}>工序报工 / 分批完工交货</button></div>}
 {data && kind !== 'records' && <form onSubmit={submit} className="space-y-4">{kind === "delivery" && !!itemFlow?.flow.batches.length && <label className="erp-label">交货计划批次<select className="erp-input" required value={batch} onChange={e=>setBatch(e.target.value)}><option value="">请选择</option>{itemFlow.flow.batches.map(b=><option key={b.id} value={b.id}>{b.batch_no} · 要求{b.quantity} · 已交{b.delivered_quantity}</option>)}</select></label>}<div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4"><label className="erp-label">零件<select className="erp-input" value={itemID} required={kind !== 'exception'} onChange={e => setItemID(e.target.value)}>{kind === 'exception' && <option value="">整单异常</option>}{data.items.map(i => <option key={i.id} value={i.id}>{i.part_name} {i.part_number || ''}</option>)}</select></label>{kind !== 'completion' && <label className="erp-label">{kind === 'delivery' ? '实际交货日期' : '发生日期'} *<input className="erp-input" type="date" required max={businessToday()} value={date} onChange={e => setDate(e.target.value)}/></label>}{(kind === 'scrap' || kind === 'delivery') && <label className="erp-label">本次{kind === 'scrap' ? '报废' : '交货'}数量 *<input className="erp-input" type="number" required min="1" step="1" max={kind === 'delivery' ? item?.available_delivery : (item ? (item.production_quantity - item.scrap_quantity - item.qualified_quantity) : 0)} value={quantity} onChange={e => setQuantity(e.target.value)}/></label>}{kind === 'scrap' && <label className="erp-label">发生工序（选填）<input className="erp-input" maxLength={80} value={process} onChange={e => setProcess(e.target.value)}/></label>}{kind === 'exception' && <><label className="erp-label">异常类型<select className="erp-input" value={category} onChange={e => setCategory(e.target.value)}>{['质量异常', '数量异常', '交期异常', '材料异常', '外协异常', '交货异常', '数据更正申请', '其他异常'].map(x => <option key={x}>{x}</option>)}</select></label><label className="erp-label">紧急程度<select className="erp-input" value={severity} onChange={e => setSeverity(e.target.value)}><option value="normal">普通</option><option value="urgent">紧急</option></select></label></>}</div>
 {item && <div className="rounded-xl bg-slate-50 p-4 text-sm leading-7">订单 {item.quantity} · 投产 {item.production_quantity} · 合格 {Number(item.qualified_quantity)} · 报废 {item.scrap_quantity} · 已交货 {item.delivered_quantity} · 可交货 {Number(item.available_delivery)}<br />订单数量完成日期：{item.completion_date || '待完成'} · 最近交货日期：{item.delivery_date || '暂无可核实日期'}</div>}
 {kind === 'completion' ? <><p className="text-sm text-slate-500">选择零件后确认本批合格数量和日期，产值自动按月份统计，无需再到产值页审核一次。</p><button type="button" disabled={!item || loading || saving} className="erp-btn erp-success" onClick={() => item && (itemFlow?.flow.tracking ? setShowFlow(true) : setCompletion({ item_id: item.id, part_name: item.part_name, part_number: item.part_number, quantity: item.quantity } as DashboardItem))}>填写合格完工</button></> : <>{kind === 'exception' && <label className="erp-label">异常情况 *<textarea className="erp-input" required maxLength={500} rows={3} value={description} onChange={e => setDescription(e.target.value)} placeholder="说明发生了什么、影响范围及需要协调的事项"/></label>}<details open={policy.operation_notes} className="border rounded-xl p-3"><summary className="text-sm text-slate-500">{kind === 'scrap' ? '报废原因 / 凭证备注' : '凭证 / 备注'}（{policy.operation_notes ? '必填' : '选填'}）</summary><textarea aria-label="业务凭证备注" className="erp-input mt-3" required={policy.operation_notes} maxLength={500} value={notes} onChange={e => setNotes(e.target.value)}/></details><p className="text-xs text-slate-500">{kind === 'scrap' ? '报废只影响投产余量，不减少订单应交数量；已确认合格件的后续质量问题请登记异常。' : kind === 'delivery' ? '支持分批交货。交货不重复计产值，备用件不自动计入订单交货数量。' : '异常记录不会直接修改已经确认的数量、金额或月结数据，处理结果会单独保留。'}</p><button disabled={saving || loading || kind === 'scrap' && !data.can_scrap || kind === 'delivery' && !data.can_delivery || kind === 'exception' && !data.can_exception} className="erp-btn erp-primary">{saving ? '保存中…' : '保存本次登记'}</button></>}
 </form>}
 {data && kind === 'records' && <div className="space-y-3">{!data.records.length && !data.completions?.length && <p className="py-8 text-slate-500">暂无业务记录，可从上方登记报废、交货或异常。</p>}{(data.completions || []).map(batch => <article key={`completion-${batch.id}`} className="erp-card p-4 border-emerald-100"><strong className="text-emerald-700">合格完工 {batch.qualified_quantity} 件 · {formatDate(batch.business_date)}</strong><p className="text-sm text-slate-500 mt-2">{data.items.find(i => i.id === batch.order_item_id)?.part_name} · 计产值 {batch.billable_quantity} 件 · 备用 {Number(batch.qualified_quantity) - Number(batch.billable_quantity)} 件 · {batch.status === 'confirmed' ? '已确认' : '已调整'}</p></article>)}{data.records.map(row => <article key={row.id} className={`erp-card p-4 ${row.kind === 'exception' && row.status === 'open' ? 'border-rose-200' : ''}`}><div className="flex gap-3 items-center justify-between flex-wrap"><strong>{row.kind === 'scrap' ? '报废' : row.kind === 'delivery' ? '交货' : '异常'} {row.quantity > 0 ? `${row.quantity} 件` : row.category} · {formatDate(row.business_date) || '历史存量：日期未知'}</strong><span className={row.status === 'open' ? 'text-rose-600' : row.status === 'resolved' ? 'text-emerald-600' : 'text-slate-500'}>{({ open: '待处理', resolved: '已处理', confirmed: '已登记', reversed: '已冲销' } as Record<string, string>)[row.status]}</span></div><p className="text-sm mt-2 text-slate-600">{row.order_item_id ? data.items.find(i => i.id === row.order_item_id)?.part_name : '整单'} {row.process_name && `· ${row.process_name}`} {row.severity === 'urgent' && '· 紧急'}</p>{row.description && <p className="mt-2 whitespace-pre-wrap">{row.description}</p>}{row.notes && <p className="text-sm text-slate-500 mt-2 whitespace-pre-wrap">{row.notes}</p>}{row.resolution && <p className="text-sm mt-2 text-emerald-700 whitespace-pre-wrap">处理 / 冲销说明：{row.resolution}</p>}<p className="text-xs text-slate-400 mt-2">记录 #{row.id} · 经办人 ID {row.created_by}{row.source_type === 'legacy' ? ' · 原始存量保留' : ''}</p><div className="flex gap-2 mt-3 flex-wrap">{row.kind === 'exception' && row.status === 'open' && data.can_exception && <button className="erp-btn" onClick={() => { setResolution({ row, mode: 'resolve' }); setReason(''); }}>登记处理结果</button>}{row.kind === 'delivery' && row.status === 'confirmed' && data.can_delivery && <button className="erp-btn" onClick={() => print(row)}>本次送货单</button>}{row.status === 'confirmed' && row.source_type !== 'legacy' && row.source_type !== 'process_quantity' && data.can_reverse && <button className="erp-btn text-rose-600" onClick={() => { setResolution({ row, mode: 'reverse' }); setReason(''); }}>冲销错误登记</button>}</div></article>)}</div>}
 {kind === 'records' && data && <div className="erp-record-pagination flex items-center justify-end gap-3 text-sm"><button type="button" className="erp-btn" disabled={loading || recordPage===1} onClick={()=>setRecordPage(p=>p-1)}>上一页记录</button><span>记录第 {recordPage} 页 · 每页最多50条业务记录和完工批次</span><button type="button" className="erp-btn" disabled={loading || !data.has_more} onClick={()=>setRecordPage(p=>p+1)}>下一页记录</button></div>}
 </div></section>{resolution && <div className="erp-overlay z-[115]"><form role="dialog" aria-label="处理业务记录" onSubmit={updateRecord} className="erp-dialog p-6 space-y-4"><h3 className="text-xl font-bold">{resolution.mode === 'resolve' ? '登记异常处理结果' : '冲销错误登记'}</h3><p className="text-sm text-slate-500">原记录保留；冲销后累计数量会同步回退。</p><label className="erp-label">{resolution.mode === 'resolve' ? '处理结果' : '冲销原因'} *<textarea className="erp-input" required maxLength={500} rows={4} value={reason} onChange={e => setReason(e.target.value)}/></label><div className="flex justify-end gap-2"><button type="button" disabled={saving} className="erp-btn" onClick={() => setResolution(null)}>取消</button><button disabled={saving} className="erp-btn erp-primary">确认保存</button></div>{error && <p role="alert" className="text-rose-600">{error}</p>}</form></div>}{showFlow && itemFlow && <ProductionFlowModal itemID={itemFlow.flow.item.id} partName={itemFlow.flow.item.part_name} initialTab="batch" onClose={()=>setShowFlow(false)} onChanged={()=>{load();onChanged();}}/>}
 {completion && <CompletionModal item={completion} onClose={() => setCompletion(null)} onSuccess={() => { load(); onChanged(); setKind('records'); setMessage('合格完工已确认，产值自动更新'); }}/>}{preview && <DeliveryPreviewModal order={{ ...order, items: preview.orderData.items }} template={preview.template} mode="preview" orderData={preview.orderData} title="本次交货送货单" onClose={() => setPreview(null)}/>}</div>;
}
