import React, { useRef, useState } from 'react';
import type { Baseline, FlowData } from '../types/productionFlow';
import { flowDate, flowRequest, flowToday, withRequestID } from '../services/productionFlow';

export default function ProcessQuantityPanel({ data, onChanged, onOutsourcing }: { data: FlowData; onChanged: () => Promise<void>; onOutsourcing: () => void }) {
  const { flow, can_manage } = data;
  const [baseline, setBaseline] = useState<Baseline[]>(flow.processes.map(p => ({ process_id: p.id, started_quantity: 0, completed_quantity: 0, scrap_quantity: 0 })));
  const [process, setProcess] = useState(String(flow.processes.find(p => !p.is_outsourced)?.id || ''));
  const [kind, setKind] = useState('start'), [quantity, setQuantity] = useState(''), [date, setDate] = useState(flowToday()), [notes, setNotes] = useState(''), [error, setError] = useState(''), [saving, setSaving] = useState(false);
  const pending = useRef({ signature: '', id: '' }), busy = useRef(false);
  const act = async (action: () => Promise<unknown>) => { if (busy.current) return; busy.current = true; setSaving(true); setError(''); try { await action(); await onChanged(); setQuantity(''); } catch (e) { setError((e as Error).message); } finally { busy.current = false; setSaving(false); } };
  const selected = flow.processes.find(p => p.id === Number(process));
  const max = kind === 'start' ? selected?.available_start : selected?.in_progress;
  return <div className="space-y-4">
    <p className="text-xs text-slate-500">各工序共用数量进度。前道完成 500 件后，后道即可开工 500 件；无需等其余零件全部加工完。</p>
    {!flow.tracking && <section className="erp-card p-3 space-y-3 bg-amber-50/30">
      <strong className="text-sm">启用工序数量跟踪</strong><p className="text-xs text-slate-500">新零件从 0 开始；旧零件请逐道核实已开工、已完工、报废数量。历史数量仅作为基线，保留原产值、交货和实际日期。</p>
      {flow.processes.map((p, i) => <div key={p.id} className="grid grid-cols-2 sm:grid-cols-4 gap-2 items-center"><strong className="text-xs">{i + 1}. {p.name}</strong>{(['started_quantity', 'completed_quantity', 'scrap_quantity'] as const).map((key, n) => <label key={key} className="erp-label">{['历史已开工', '历史已完工', '历史报废'][n]}<input type="number" min={0} step={1} disabled={!can_manage || saving} className="erp-input" value={baseline[i]?.[key] ?? 0} onChange={e => setBaseline(old => old.map((b, j) => j === i ? { ...b, [key]: Number(e.target.value) } : b))}/></label>)}</div>)}
      {can_manage && <button type="button" className="erp-btn erp-primary" disabled={saving || !flow.processes.length} onClick={() => act(() => flowRequest(`/production/items/${flow.item.id}/flow/activate`, 'POST', { baseline }))}>核对后启用</button>}
    </section>}
    <div className="overflow-x-auto"><table className="w-full text-xs whitespace-nowrap"><thead className="bg-slate-50"><tr>{['工序', '已开工', '已完工', '报废', '在制', '可新开工'].map(v => <th key={v} className="p-2 text-left">{v}</th>)}</tr></thead><tbody>{flow.processes.map(p => <tr key={p.id} className="border-b border-slate-100"><td className="p-2 font-medium">{p.name}{p.is_outsourced && <span className="ml-2 text-violet-600">外协</span>}</td>{[p.started_quantity, p.completed_quantity, p.scrap_quantity, p.in_progress, p.available_start].map((q, i) => <td key={i} className="p-2 tabular-nums">{flow.tracking ? q : '未启用'}</td>)}</tr>)}</tbody></table></div>
    {flow.tracking && can_manage && <form className="erp-card p-3 space-y-3" onSubmit={e => { e.preventDefault(); act(() => flowRequest(`/production/items/${flow.item.id}/processes/${process}/quantities`, 'POST', withRequestID(pending.current, { kind, quantity: Number(quantity), business_date: date, notes }))); }}>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3"><label className="erp-label">工序<select className="erp-input" value={process} onChange={e => setProcess(e.target.value)} required><option value="">请选择</option>{flow.processes.filter(p => !p.is_outsourced).map(p => <option key={p.id} value={p.id}>{p.name}</option>)}</select></label><label className="erp-label">本次操作<select className="erp-input" value={kind} onChange={e => setKind(e.target.value)}><option value="start">开工</option><option value="complete">完工</option><option value="scrap">报废</option></select></label><label className="erp-label">本次数量（最多 {max || 0}）<input className="erp-input" type="number" min={1} max={max} step={1} required value={quantity} onChange={e => setQuantity(e.target.value)}/></label><label className="erp-label">发生日期<input className="erp-input" type="date" min={flowDate(flow.item.start_date)} max={flowToday()} required value={date} onChange={e => setDate(e.target.value)}/></label></div>
      <label className="erp-label">备注（选填）<input className="erp-input" maxLength={500} value={notes} onChange={e => setNotes(e.target.value)}/></label><div className="flex gap-2"><button className="erp-btn erp-primary" disabled={saving || !selected || !max}>登记工序数量</button><button className="erp-btn" type="button" onClick={onOutsourcing}>外协发出 / 回厂</button></div>
    </form>}
    {error && <p role="alert" className="text-sm text-rose-600">{error}</p>}
    <details className="text-xs text-slate-500"><summary className="cursor-pointer">最近工序报工记录（最多50条）</summary><div className="mt-2 space-y-1">{flow.events.map(e => <p key={e.id}>{flowDate(e.business_date) || '历史基线（日期未知）'} · {flow.processes.find(p => p.id === e.order_process_id)?.name} · {{ start: '开工', complete: '完工', scrap: '报废' }[e.kind] || e.kind} {e.quantity} 件 · {e.source_type === 'outsourcing_reversal' ? '撤销外协发出的数量冲回' : e.source_type === 'outsourcing' ? '外协自动报工' : ''}</p>)}</div></details>
  </div>;
}
