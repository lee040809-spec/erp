import ResponsiveTable from './ResponsiveTable';
import type { FinanceAccess } from '../services/financeAccess';
import React, { useEffect, useRef, useState } from 'react';
import { authFetch } from './shared';
import { basicEvidence, readEntryPolicy } from './BusinessEntrySettings';
interface Candidate {
    order_item_id: number;
    order_id: number;
    order_number: string;
    customer_name: string;
    part_name: string;
    quantity: number;
    production_quantity: number;
    scrap_quantity: number;
    business_date: string | null;
    confirmed_quantity: number;
    remaining_quantity: number;
    unit_price: string;
    supply_mode: string;
    currency: string;
    tax_inclusive: number;
}
interface Batch {
    id: number;
    completion_no: string;
    order_number: string;
    customer_name: string;
    part_name: string;
    business_date: string;
    qualified_quantity: string;
    billable_quantity: string;
    reserve_quantity: string;
    unit_price_snapshot: string;
    output_value: string;
    notes: string;
}
interface Cost {
    id: number;
    cost_type: string;
    business_date: string;
    amount: string;
    tax_amount: string;
    net_amount: string;
    order_id?: number;
    order_item_id?: number;
    supplier_name?: string;
    notes?: string;
}
interface PendingCost extends Cost {
    remaining_amount: string;
}
interface AllocationRecord {
    id: number;
    cost_entry_id: number;
    cost_type: string;
    cost_date: string;
    completion_no: string;
    allocated_amount: string;
    allocation_basis: string;
    created_by: number;
}
const labels: Record<string, string> = { material: '材料', outsourcing: '外协', tool: '刀具', fixture: '工装', labor: '人工', manufacturing: '制造费用', other_direct: '其他直接费用', period: '期间费用' };
const amount = (v: string) => Number(v || 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
async function request<T>(url: string, payload?: unknown): Promise<T> {
    const res = await authFetch(url, payload ? { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) } : undefined);
    const result = await res.json();
    if (!res.ok)
        throw new Error(result.message || result.error || '读取失败');
    return result.data;
}
export default function FinanceReviewPanel({ year, month, selectionVersion, onImported, access }: {
    access: FinanceAccess | null;
    year: string;
    month: string;
    selectionVersion: number;
    onImported: () => void;
}) {
    const [policy, setPolicy] = useState(basicEvidence);
    useEffect(() => { readEntryPolicy().then(v => setPolicy(v.effective)).catch(e => setError(e.message)); }, []);
    const historyRequest = useRef({ signature: '', id: '' });
    const [tab, setTab] = useState<'history' | 'details' | 'allocation'>('history');
    const [page, setPage] = useState(1);
    const [candidates, setCandidates] = useState<Candidate[]>([]);
    const [total, setTotal] = useState(0);
    const [selectedMonth, setSelectedMonth] = useState(`${year}-01`);
    const [detailPage,setDetailPage]=useState(1),[detailMore,setDetailMore]=useState(false),[hasCompletions,setHasCompletions]=useState(false),[pendingPage,setPendingPage]=useState(1),[pendingMore,setPendingMore]=useState(false);
    const [batches, setBatches] = useState<Batch[]>([]);
    const [costs, setCosts] = useState<Cost[]>([]);
    const [pending, setPending] = useState<PendingCost[]>([]);
    const [allocationRecords, setAllocationRecords] = useState<AllocationRecord[]>([]);
    const [allocation, setAllocation] = useState({ cost: '', batch: '', amount: '', basis: '' });
    const [closeNotes, setCloseNotes] = useState('');
    const [closeReviewed, setCloseReviewed] = useState(false);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [message, setMessage] = useState('');
    const [selected, setSelected] = useState<Candidate | null>(null);
    const [form, setForm] = useState({ quantity: '', date: '', notes: '', sourceRef: '', price: '', reviewed: false });
    const [saving, setSaving] = useState(false);
    const busy = useRef(false);
    const generation = useRef(0);
    const load = async () => {
        const current = ++generation.current;
        setLoading(true);
        setError('');
        try {
            if (tab === 'history') {
                const data = await request<{
                    items: Candidate[];
                    total: number;
                }>(`/api/platform/finance/history?year=${year}&page=${page}`);
                if (current === generation.current) {
                    setCandidates(data.items || []);
                    setTotal(data.total);
                }
            }
            else {
                const data = await request<{
                    completions: Batch[];
                    costs: Cost[];
                    allocations: AllocationRecord[];
                    has_more: boolean;
                    has_completions: boolean;
                }>(`/api/platform/finance/details?month=${selectedMonth}&page=${detailPage}&pageSize=50`);
                if (current === generation.current) {
                    setDetailMore(data.has_more);setHasCompletions(data.has_completions);
                    setBatches(data.completions || []);
                    setCosts(data.costs || []);
                    setAllocationRecords(data.allocations || []);
                }
                if (tab === 'allocation') {
                    const result = await request<{items:PendingCost[];has_more:boolean}>(`/api/platform/finance/pending-costs?page=${pendingPage}&pageSize=50`);
                    if (current === generation.current) {setPending(result.items || []);setPendingMore(result.has_more);}
                }
            }
        }
        catch (e) {
            if (current === generation.current)
                setError(e instanceof Error ? e.message : '读取失败');
        }
        finally {
            if (current === generation.current)
                setLoading(false);
        }
    };
    useEffect(() => { setPage(1); setSelected(null); setSelectedMonth(`${year}-01`); }, [year]);
    useEffect(() => {
        if (month) {
            setSelectedMonth(month);
            setTab('details');
        }
    }, [month, selectionVersion]);
    useEffect(() => { load(); return () => { generation.current++; }; }, [year, page, tab, selectedMonth,detailPage,pendingPage]);
    useEffect(()=>{setDetailPage(1);setPendingPage(1);setAllocation({cost:'',batch:'',amount:'',basis:''});},[selectedMonth,year]);
    const review = (item: Candidate) => {
        setSelected(item);
        setMessage('');
        setError('');
        setForm({ quantity: String(item.remaining_quantity), date: item.business_date || '', notes: '', sourceRef: '', price: item.unit_price || '', reviewed: false });
    };
    const submit = async (event: React.FormEvent) => {
        event.preventDefault();
        if (!selected || busy.current || !access?.can_import_history)
            return;
        busy.current = true;
        setSaving(true);
        setError('');
        try {
            const signature = JSON.stringify([selected.order_item_id, form]);
            if (historyRequest.current.signature !== signature)
                historyRequest.current = { signature, id: crypto.randomUUID() };
            await request('/api/platform/finance/history', { request_id: historyRequest.current.id, order_item_id: selected.order_item_id, qualified_quantity: Number(form.quantity), business_date: form.date, expected_price: selected.unit_price, source_ref: form.sourceRef.trim(), confirmed_price: form.price.trim(), notes: form.notes.trim(), reviewed: true });
            setSelected(null);
            setMessage('已计入正式产值，请在对应年份、月份查看明细。');
            onImported();
            await load();
        }
        catch (e) {
            setError(e instanceof Error ? e.message : '导入失败');
        }
        finally {
            busy.current = false;
            setSaving(false);
        }
    };
    const allocate = async (event: React.FormEvent) => {
        event.preventDefault();
        if (busy.current || !access?.can_manage_costs)
            return;
        busy.current = true;
        setSaving(true);
        setError('');
        setMessage('');
        try {
            await request('/api/platform/finance/allocations', { cost_entry_id: Number(allocation.cost), production_completion_id: Number(allocation.batch), allocated_amount: allocation.amount, allocation_basis: allocation.basis.trim() });
            setAllocation({ cost: '', batch: '', amount: '', basis: '' });
            setMessage('分摊已保存，生产成本已计入该批次完工月份。');
            onImported();
            await load();
        }
        catch (e) {
            setError(e instanceof Error ? e.message : '分摊失败');
        }
        finally {
            busy.current = false;
            setSaving(false);
        }
    };
    const closeMonth = async (event: React.FormEvent) => {
        event.preventDefault();
        if (busy.current || !access?.can_manage_costs)
            return;
        busy.current = true;
        setSaving(true);
        setError('');
        setMessage('');
        try {
            await request(`/api/platform/finance/months/${selectedMonth}/close`, { reviewed_cost_complete: closeReviewed, notes: closeNotes.trim() });
            setMessage(`${selectedMonth} 已结账，新增完工、费用和分摊将被锁定。`);
            setCloseNotes('');
            setCloseReviewed(false);
            onImported();
            await load();
        }
        catch (e) {
            setError(e instanceof Error ? e.message : '月结失败');
        }
        finally {
            busy.current = false;
            setSaving(false);
        }
    };
    return <section className="bg-white border border-zinc-200 p-5 space-y-4">
    <div className="flex flex-wrap gap-2 items-center">
      <button onClick={() => setTab('history')} className={`px-4 py-2 border ${tab === 'history' ? 'bg-zinc-900 text-white' : ''}`}>历史完工补录</button>
      <button onClick={() => setTab('details')} className={`px-4 py-2 border ${tab === 'details' ? 'bg-zinc-900 text-white' : ''}`}>产值与费用明细</button>
      {access?.can_manage_costs && <button onClick={() => setTab('allocation')} className={`px-4 py-2 border ${tab === 'allocation' ? 'bg-zinc-900 text-white' : ''}`}>成本分摊</button>}
      <button onClick={load} disabled={loading} className="px-3 py-2 border disabled:opacity-50">刷新明细</button>
    </div>
    {error && <p role="alert" className="p-3 bg-rose-50 text-rose-700">{error}</p>}
    {message && <p role="status" className="p-3 bg-emerald-50 text-emerald-700">{message}</p>}
    {tab === 'history' ? <>
      <p className="text-sm text-zinc-600">{year} 年待审核 {total} 个零件（含缺少完工日期的记录）。旧订单数量不等于合格数量，确认后才进入正式统计。支持分多个月补录，系统自动生成批次编号，累计不能超过剩余数量。历史成本仍需另外审核录入。</p>
      <div className="erp-table-scroll overflow-auto max-h-[500px]"><ResponsiveTable className="w-full min-w-[850px] text-sm"><thead className="bg-zinc-50 sticky top-0"><tr>{['订单 / 客户', '零件', '订单 / 报废 / 已审核 / 剩余', '原完工日期', '当前单价', '供料', '操作'].map(x => <th key={x} className="text-left p-3">{x}</th>)}</tr></thead><tbody>
        {!loading && candidates.map(item => <tr key={item.order_item_id} className="border-t"><td className="p-3">{item.order_number}<div className="text-zinc-500">{item.customer_name}</div></td><td className="p-3">{item.part_name}</td><td className="p-3">{item.quantity} / {item.scrap_quantity} / {item.confirmed_quantity} / {item.remaining_quantity}</td><td className="p-3">{item.business_date || '缺失，需补录'}</td><td className="p-3">{item.unit_price || '缺失'} {item.currency}{item.tax_inclusive ? '（含税）' : ''}</td><td className="p-3">{item.supply_mode === 'customer_supplied' ? '客户供料' : '自购材料'}</td><td className="p-3">{access?.can_import_history && Number(item.unit_price) > 0 && item.currency === 'CNY' && item.remaining_quantity > 0 ? <button onClick={() => review(item)} disabled={saving} className="text-blue-700 underline">补录完工</button> : <span className="text-amber-700">{!access ? '正在核对权限…' : !access.can_import_history ? '仅查看' : '先核对数量、单价和币种'}</span>}</td></tr>)}
      </tbody></ResponsiveTable></div>
      {!loading && !candidates.length && <p className="py-6 text-center text-zinc-500">当前年份没有待审核的历史完工记录。</p>}
      <div className="flex items-center gap-3 text-sm"><button disabled={page <= 1 || loading} onClick={() => setPage(page - 1)} className="border px-3 py-2 disabled:opacity-40">上一页</button><span>第 {page} 页 / {Math.max(1, Math.ceil(total / 50))} 页</span><button disabled={page * 50 >= total || loading} onClick={() => setPage(page + 1)} className="border px-3 py-2 disabled:opacity-40">下一页</button></div>
    </> : tab === 'allocation' ? <>
      <p className="text-sm text-zinc-600">选择待分摊费用，再选择实际完工月份与批次。金额可以分配给不同月份的批次，累计不能超过原费用。请先审核历史完工或在生产页面确认合格完工。</p>
      <label className="text-sm">目标完工月份 <input aria-label="目标完工月份" type="month" value={selectedMonth} onChange={e => { setSelectedMonth(e.target.value); setAllocation({ ...allocation, batch: '' }); }} className="border px-3 py-2 ml-2"/></label>
      <form onSubmit={allocate} className="grid md:grid-cols-2 gap-4 text-sm">
        <label>待分摊费用<select required value={allocation.cost} onChange={e => setAllocation({ ...allocation, cost: e.target.value })} className="block border w-full p-2 mt-1"><option value="">请选择费用</option>{pending.map(c => <option key={c.id} value={c.id}>#{c.id} {labels[c.cost_type]} {c.business_date} {c.supplier_name || ''} 待分摊 {c.remaining_amount} 元 {c.order_item_id ? `零件#${c.order_item_id}` : ''}</option>)}</select></label>
        <label>目标合格完工批次<select required value={allocation.batch} onChange={e => setAllocation({ ...allocation, batch: e.target.value })} className="block border w-full p-2 mt-1"><option value="">请选择批次</option>{batches.map(b => <option key={b.id} value={b.id}>#{b.id} {b.order_number} / {b.part_name} / {b.qualified_quantity} 件</option>)}</select></label>
        <label>本次分摊金额<input required inputMode="decimal" pattern="\d+(\.\d{1,2})?" value={allocation.amount} onChange={e => setAllocation({ ...allocation, amount: e.target.value })} className="block border w-full p-2 mt-1"/></label>
        <label>分摊依据（{policy.allocation_basis ? '必填' : '选填'}）<input required={policy.allocation_basis} maxLength={100} placeholder="例如：材料领用单、外协结算单、工时比例" value={allocation.basis} onChange={e => setAllocation({ ...allocation, basis: e.target.value })} className="block border w-full p-2 mt-1"/></label>
        <button disabled={saving || loading} className="bg-zinc-900 text-white p-3 disabled:opacity-40 md:col-span-2">{saving ? '正在保存…' : '确认成本分摊'}</button>
      </form>
      {!loading && !pending.length && <p className="text-zinc-500 text-sm">暂无待分摊生产费用，请先录入实际耗用及费用。</p>}
      {!loading && !batches.length && <p className="text-amber-700 text-sm">目标月份没有合格完工批次，请先确认或审核导入。</p>}
    </> : <>
      <label className="text-sm">明细月份 <input aria-label="明细月份" type="month" value={selectedMonth} onChange={e => setSelectedMonth(e.target.value)} className="border px-3 py-2 ml-2"/></label>
      <h4 className="font-semibold">合格完工批次</h4>
      <div className="erp-table-scroll overflow-x-auto"><ResponsiveTable className="w-full min-w-[850px] text-sm"><thead className="bg-zinc-50"><tr>{['批次 / 日期', '订单 / 客户', '零件', '合格数量', '计产值数量', '备用数量', '单价快照', '产值', '审核依据 / 备注'].map(x => <th className="text-left p-3" key={x}>{x}</th>)}</tr></thead><tbody>{!loading && batches.map(b => <tr key={b.id} className="border-t"><td className="p-3">{b.completion_no}<div>{b.business_date}</div></td><td className="p-3">{b.order_number}<div>{b.customer_name}</div></td><td className="p-3">{b.part_name}</td><td className="p-3">{b.qualified_quantity}</td><td className="p-3">{b.billable_quantity}</td><td className="p-3 text-violet-600">{b.reserve_quantity}</td><td className="p-3">{b.unit_price_snapshot}</td><td className="p-3">{amount(b.output_value)}</td><td className="p-3">{b.notes}</td></tr>)}</tbody></ResponsiveTable></div>
      {!loading && !batches.length && <p className="text-zinc-500 text-sm">该月暂无已确认批次。</p>}
      <h4 className="font-semibold">计入本月完工的成本分摊明细</h4>
      <div className="erp-table-scroll overflow-x-auto"><ResponsiveTable className="w-full min-w-[850px] text-sm"><thead className="bg-zinc-50"><tr>{['费用编号 / 类型', '原费用日期', '目标批次', '本月分摊金额', '依据', '操作人 ID'].map(x => <th className="text-left p-3" key={x}>{x}</th>)}</tr></thead><tbody>{!loading && allocationRecords.map(a => <tr className="border-t" key={a.id}><td className="p-3">#{a.cost_entry_id} {labels[a.cost_type] || a.cost_type}</td><td className="p-3">{a.cost_date}</td><td className="p-3">{a.completion_no}</td><td className="p-3">{amount(a.allocated_amount)}</td><td className="p-3">{a.allocation_basis}</td><td className="p-3">{a.created_by}</td></tr>)}</tbody></ResponsiveTable></div>
      {!loading && !allocationRecords.length && <p className="text-zinc-500 text-sm">该月暂无已分摊生产成本。</p>}
      <h4 className="font-semibold">本月发生费用台账（生产费用需在成本分摊中归集到批次）</h4>
      <div className="erp-table-scroll overflow-x-auto"><ResponsiveTable className="w-full min-w-[850px] text-sm"><thead className="bg-zinc-50"><tr>{['日期', '类型', '含税金额', '税额', '核算金额', '订单 / 零件 ID', '供应商 / 备注'].map(x => <th className="text-left p-3" key={x}>{x}</th>)}</tr></thead><tbody>{!loading && costs.map(c => <tr className="border-t" key={c.id}><td className="p-3">{c.business_date.slice(0, 10)}</td><td className="p-3">{labels[c.cost_type] || c.cost_type}</td><td className="p-3">{amount(c.amount)}</td><td className="p-3">{amount(c.tax_amount)}</td><td className="p-3">{amount(c.net_amount)}</td><td className="p-3">{c.order_id || '—'} / {c.order_item_id || '—'}</td><td className="p-3">{c.supplier_name}<div>{c.notes}</div></td></tr>)}</tbody></ResponsiveTable></div>
      {!loading && !costs.length && <p className="text-zinc-500 text-sm">该月暂无费用。零成本不表示成本完整。</p>}
      {access?.can_manage_costs && <form onSubmit={closeMonth} className="border-t pt-4 space-y-3">
        <h4 className="font-semibold">月结审核</h4>
        <p className="text-sm text-zinc-600">只能结算已结束月份。结账后该月不能新增完工、费用及成本分摊；请核对实际费用已完整录入。</p>
        <label className="flex gap-2 text-sm"><input required type="checkbox" checked={closeReviewed} onChange={e => setCloseReviewed(e.target.checked)}/>已核对完工数量、价格、材料、外协、人工、制造及期间费用，确认本月核算完整。</label>
        <label className="block text-sm">月结审核依据（{policy.closing_notes ? '必填' : '选填'}）<input required={policy.closing_notes} maxLength={500} value={closeNotes} onChange={e => setCloseNotes(e.target.value)} className="block border w-full p-2 mt-1"/></label>
        <button disabled={saving || loading || !hasCompletions} className="border border-zinc-800 px-4 py-2 text-sm disabled:opacity-40">{saving ? '正在处理…' : `确认 ${selectedMonth} 月结并锁定`}</button>
      </form>}
    </>}
    {loading && <p role="status" className="py-6 text-center text-zinc-500">正在读取…</p>}
    {tab !== 'history' && <div className="erp-history-pagination flex flex-wrap items-center justify-end gap-3 text-sm"><button className="erp-btn" disabled={loading || detailPage===1} onClick={()=>{setDetailPage(p=>p-1);setAllocation(a=>({...a,batch:''}));}}>上一页明细</button><span>明细第 {detailPage} 页 · 每类50条</span><button className="erp-btn" disabled={loading || !detailMore} onClick={()=>{setDetailPage(p=>p+1);setAllocation(a=>({...a,batch:''}));}}>下一页明细</button>{tab==='allocation' && <><button className="erp-btn" disabled={loading || pendingPage===1} onClick={()=>{setPendingPage(p=>p-1);setAllocation(a=>({...a,cost:''}));}}>上一页待分摊费用</button><span>费用第 {pendingPage} 页</span><button className="erp-btn" disabled={loading || !pendingMore} onClick={()=>{setPendingPage(p=>p+1);setAllocation(a=>({...a,cost:''}));}}>下一页待分摊费用</button></>}</div>}
  {selected && <div className="erp-overlay z-[110]"><form role="dialog" aria-label="补录历史完工" onSubmit={submit} className="erp-dialog p-6 w-full max-w-xl space-y-4 max-h-[90vh] overflow-y-auto">
      <h3 className="text-lg font-bold">补录历史完工：{selected.part_name}</h3>
      <p className="text-sm text-zinc-600">{selected.order_number}；订单 {selected.quantity} 件，投产 {selected.production_quantity} 件，报废 {selected.scrap_quantity} 件；当前订单单价 {selected.unit_price} 元。已审核 {selected.confirmed_quantity} 件，剩余 {selected.remaining_quantity} 件。每次补录一批，核对实际数量、日期与单价即可；凭证信息选填，订单当前单价保持不变。</p>
      <div className="grid sm:grid-cols-2 gap-4"><label className="erp-label">实际合格数量 *<input required type="number" min="1" max={selected.remaining_quantity} step="1" value={form.quantity} onChange={e => setForm({ ...form, quantity: e.target.value })} className="erp-input"/></label><label className="erp-label">实际合格完工日期 *<input required type="date" value={form.date} onChange={e => setForm({ ...form, date: e.target.value })} className="erp-input"/></label></div>
      <p className="text-sm text-slate-600">本批单价 {form.price}，默认使用当前订单单价；确认后自动计入对应月份。</p>
      <details open={policy.source_ref || policy.review_notes} className="border rounded-xl p-3 space-y-3"><summary className="text-sm text-slate-500">凭证与补充信息（默认选填，按角色要求）</summary><label className="erp-label">原始批次依据编号（{policy.source_ref ? '必填' : '选填'}）<input required={policy.source_ref} maxLength={80} placeholder="留空由系统生成批次编号" value={form.sourceRef} onChange={e => setForm({ ...form, sourceRef: e.target.value })} className="erp-input"/></label><label className="erp-label">审核依据（{policy.review_notes ? '必填' : '选填'}）<textarea required={policy.review_notes} maxLength={400} value={form.notes} onChange={e => setForm({ ...form, notes: e.target.value })} className="erp-input"/></label><label className="erp-label">本批历史单价<input required inputMode="decimal" pattern="\d{1,11}(\.\d{1,4})?" value={form.price} onChange={e => setForm({ ...form, price: e.target.value })} className="erp-input"/></label></details>
      <p className="text-xs text-slate-500">请确认数量、日期与单价准确。超出订单数量的合格件作为备用件，不计产值；点击确认即可完成补录。</p>
      {error && <p role="alert" className="text-rose-700">{error}</p>}
      <div className="flex justify-end gap-3"><button type="button" disabled={saving} onClick={() => setSelected(null)} className="border px-4 py-2">取消</button><button disabled={saving || !access?.can_import_history} className="bg-zinc-900 text-white px-4 py-2 disabled:opacity-50">{saving ? '正在导入…' : '确认计入产值'}</button></div>
    </form></div>}
  </section>;
}
