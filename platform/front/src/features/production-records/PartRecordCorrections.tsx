import { useEffect, useRef, useState } from 'react';
import { flowRequest, flowToday, withRequestID } from '../../services/productionFlow';
import type { RecordedQuantity, RecordHistory, RecordKind } from './recordCorrection';
import { correctionError } from './recordCorrection';

export default function PartRecordCorrections({ itemId, kind, refreshKey, onChanged, onBusyChange }: {
  itemId: number; kind: RecordKind; refreshKey: number;
  onChanged: () => void | Promise<void>; onBusyChange: (busy: boolean) => void;
}) {
  const [open, setOpen] = useState(false), [page, setPage] = useState(1);
  const [data, setData] = useState<RecordHistory | null>(null), [loading, setLoading] = useState(false);
  const [editing, setEditing] = useState<(RecordedQuantity & { revision: number }) | null>(null), [quantity, setQuantity] = useState('');
  const [date, setDate] = useState(''), [reason, setReason] = useState(''), [error, setError] = useState('');
  const [message, setMessage] = useState(''), [saving, setSaving] = useState(false), [reload, setReload] = useState(0);
  const busy = useRef(false), pending = useRef({ signature: '', id: '' });
  const label = kind === 'completion' ? '合格' : '交货';
  useEffect(() => { onBusyChange(saving); return () => onBusyChange(false); }, [saving, onBusyChange]);
  useEffect(() => {
    if (!open) return;
    let active = true; setLoading(true); setError('');
    flowRequest<RecordHistory>(`/production/items/${itemId}/records?kind=${kind}&page=${page}&pageSize=20`)
      .then(value => { if (active) setData(value); })
      .catch(e => { if (active) { setData(null); setError((e as Error).message); } })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [itemId, kind, open, page, refreshKey, reload]);
  const edit = (row: RecordedQuantity) => { setEditing({ ...row, revision: data?.revision || 0 }); setQuantity(String(Number(row.quantity))); setDate(row.business_date); setReason(''); setError(''); setMessage(''); };
  const save = async (event: React.FormEvent) => {
    event.preventDefault(); if (busy.current || !editing || !data || loading) return;
    const invalid = correctionError(quantity, date, reason, flowToday());
    if (invalid) { setError(invalid); return; }
    busy.current = true; setSaving(true); setError(''); setMessage(''); let saved = false;
    try {
      await flowRequest(`/production/items/${itemId}/records/${kind}/${editing.id}/correct`, 'POST', withRequestID(pending.current, {
        quantity: Number(quantity), business_date: date, reason: reason.trim(), expected_quantity: editing.quantity,
        expected_date: editing.business_date, expected_revision: editing.revision,
      }));
      saved = true; setEditing(null); setReason(''); setMessage('修改已保存，原值与修改原因已留存。');
      await onChanged(); setReload(value => value + 1);
    } catch (e) { setError(saved ? '修改已保存，但刷新显示失败，请刷新页面，不要重复登记。' : (e as Error).message); }
    finally { busy.current = false; setSaving(false); }
  };
  return <section className="erp-card p-4 space-y-3" aria-label={`${label}记录修改`}>
    <button type="button" className="erp-btn text-blue-700" disabled={saving || !!editing} aria-expanded={open} onClick={() => setOpen(value => !value)}>{open ? '收起' : '查看 / 修改'}已登记{label}记录（数量、日期）</button>
    {open && <>
      <p className="text-xs text-slate-500">填错时直接修改原记录，原因必填；数量、日期和统计一并更新，原记录编号与费用关联保留。已结账月份不能直接更改。</p>
      {loading && <p role="status" className="text-sm">读取已登记记录…</p>}
      {message && <p role="status" className="text-sm text-emerald-700">{message}</p>}
      {error && <p role="alert" className="text-sm text-rose-700">{error}<button type="button" className="erp-btn ml-2" disabled={saving} onClick={() => setReload(v => v + 1)}>重新读取</button></p>}
      {!loading && data && <>
        {!data.records.length && <p className="text-sm text-slate-500">暂无已登记{label}记录。</p>}
        {data.records.map(row => <article key={row.id} className="border-b border-slate-100 pb-3 space-y-2 text-sm">
          <div className="flex flex-wrap items-center justify-between gap-2"><span>#{row.id} · {row.business_date || '日期未记录'} · {Number(row.quantity)} 件{row.batch_no ? ` · ${row.batch_no}` : ''} · {row.status === 'confirmed' ? '已登记' : '已撤销'}</span>
            {data.can_edit && row.status === 'confirmed' && <button type="button" className="erp-btn" disabled={saving || !!editing} onClick={() => edit(row)}>修改数量 / 日期</button>}</div>
          {row.source_type === 'legacy' && <p className="text-xs text-amber-700">历史累计记录，请按原始交货单核对；修改不会自动拆分成多次交货。</p>}
          {!!data.changes.filter(change => change.record_id === row.id).length && <details><summary className="cursor-pointer text-xs text-slate-500">修改历史（最近100条）</summary>{data.changes.filter(change => change.record_id === row.id).map(change => <p key={change.id} className="text-xs text-slate-500 mt-2">{change.before_date || '未记录'} / {Number(change.before_quantity)} 件 → {change.after_date} / {Number(change.after_quantity)} 件；原因：{change.reason} · 经办人 #{change.created_by} · {change.created_at.slice(0, 10)}</p>)}</details>}
        </article>)}
        {!editing && <div className="flex items-center justify-end gap-2"><button type="button" className="erp-btn" disabled={saving || page <= 1} onClick={() => setPage(v => v - 1)}>上一页记录</button><span className="text-xs">第 {page} 页</span><button type="button" className="erp-btn" disabled={saving || !data.has_more} onClick={() => setPage(v => v + 1)}>下一页记录</button></div>}
      </>}
      {editing && <form className="bg-blue-50/50 rounded-xl p-4 space-y-3" onSubmit={save}>
        <h5 className="font-semibold">修改{label}记录 #{editing.id}</h5>
        <fieldset disabled={saving || loading} className="grid sm:grid-cols-2 gap-3"><label className="erp-label">正确数量 *<input className="erp-input" type="number" min={1} step={1} max={2147483647} required value={quantity} onChange={e => setQuantity(e.target.value)}/></label><label className="erp-label">正确日期 *<input className="erp-input" type="date" max={flowToday()} required value={date} onChange={e => setDate(e.target.value)}/></label><label className="erp-label sm:col-span-2">修改原因 *<textarea className="erp-input" required maxLength={500} rows={2} value={reason} onChange={e => setReason(e.target.value)} placeholder="说明原登记哪里填错，以及更正依据"/></label></fieldset>
        <div className="flex justify-end gap-2"><button type="button" disabled={saving} className="erp-btn" onClick={() => { setEditing(null); setError(''); }}>取消修改</button><button type="submit" disabled={saving || loading || !data || !reason.trim()} className="erp-btn erp-primary">{saving ? '保存中…' : '保存修改并更新统计'}</button></div>
      </form>}
    </>}
  </section>;
}
