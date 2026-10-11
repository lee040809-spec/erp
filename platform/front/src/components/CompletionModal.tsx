import { businessToday } from '../utils/businessDate';
import { basicEvidence, readEntryPolicy } from './BusinessEntrySettings';
import React, { useEffect, useRef, useState } from 'react';
import { AlertCircle, CheckCircle2, Loader2, X } from 'lucide-react';
import { CompletionListResponse } from '../types';
import { createCompletionApi, DashboardItem, fetchCompletionsApi } from '../services/api';
interface CompletionModalProps {
    item: DashboardItem;
    onClose: () => void;
    onSuccess: () => void | Promise<void>;
    suggestedQuantity?: number;
    latestBusinessDate?: string;
    embedded?: boolean;
    refreshKey?: number;
    onBusyChange?: (busy: boolean) => void;
}

const CompletionModal: React.FC<CompletionModalProps> = ({ item, onClose, onSuccess, suggestedQuantity, latestBusinessDate, embedded = false, refreshKey = 0, onBusyChange }) => {
    const [policy, setPolicy] = useState(basicEvidence);
    useEffect(() => { readEntryPolicy().then(v => setPolicy(v.effective)).catch(e => setError(e.message)); }, []);
    const [info, setInfo] = useState<CompletionListResponse | null>(null);
    const [quantity, setQuantity] = useState('');
    const maxDate = latestBusinessDate && latestBusinessDate < businessToday() ? latestBusinessDate : businessToday();
    const [businessDate, setBusinessDate] = useState(maxDate);
    const [notes, setNotes] = useState('');
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    const pendingRequest = useRef({ signature: '', id: '' });
    const submitting = useRef(false);
    useEffect(() => { onBusyChange?.(saving); return()=>onBusyChange?.(false); }, [saving, onBusyChange]);
    useEffect(() => {
        let active = true; setLoading(true);
        fetchCompletionsApi(item.item_id)
            .then(data => {
            if (!active) return;
            setInfo(data);
            setQuantity(old => embedded && old ? old : String(Math.max(0, Math.min(Math.floor(data.remaining_quantity), suggestedQuantity && suggestedQuantity > 0 ? Math.floor(suggestedQuantity) : Infinity))));
        })
            .catch(err => active && setError(err instanceof Error ? err.message : '完工记录加载失败'))
            .finally(() => active && setLoading(false));
        return () => { active = false; };
    }, [item.item_id, refreshKey]);
    const submit = async (event: React.FormEvent) => {
        event.preventDefault();
        if (submitting.current)
            return;
        const value = Number(quantity);
        if (!Number.isInteger(value) || value <= 0) {
            setError('本次合格数量必须是大于 0 的整数');
            return;
        }
        if (info && value > info.remaining_quantity) {
            setError(`本次数量不能超过剩余数量 ${info.remaining_quantity}`);
            return;
        }
        if (!businessDate || businessDate > maxDate) {
            setError('实际完工日期不能晚于交货日期或今天');
            return;
        }
        setSaving(true);
        submitting.current = true;
        setError('');
        try {
            const signature = JSON.stringify([item.item_id, value, businessDate, notes.trim()]);
            if (pendingRequest.current.signature !== signature)
                pendingRequest.current = { signature, id: crypto.randomUUID() };
            await createCompletionApi(item.item_id, {
                request_id: pendingRequest.current.id,
                qualified_quantity: value,
                business_date: businessDate,
                notes: notes.trim() || undefined
            });
            setQuantity('');
            await onSuccess();
            if (!embedded) onClose();
        }
        catch (err) {
            setError(err instanceof Error ? err.message : '合格完工确认失败');
        }
        finally {
            submitting.current = false;
            setSaving(false);
        }
    };
    return (<div className={embedded ? "erp-workflow-embedded" : "fixed inset-0 z-[110] bg-black/40 flex items-center justify-center p-4"} onMouseDown={event => !embedded && !saving && event.target === event.currentTarget && onClose()}>
      <div className={embedded ? "bg-white w-full" : "bg-white w-full max-w-lg rounded-2xl overflow-hidden shadow-2xl border border-zinc-200"}>
        {!embedded && <div className="px-5 py-4 border-b border-zinc-200 flex items-start justify-between">
          <div>
            <h3 className="font-bold text-lg">确认合格完工</h3>
            <p className="text-sm text-zinc-500 mt-1">{item.part_name} · {item.part_number || '无图号'}</p>
          </div>
          <button onClick={onClose} className="p-1 text-zinc-400 hover:text-zinc-900"><X className="w-5 h-5"/></button>
        </div>}

        {loading ? (<div className="h-56 flex items-center justify-center text-zinc-400"><Loader2 className="w-5 h-5 animate-spin mr-2"/>正在读取完工记录</div>) : (<form onSubmit={submit} className="p-5 space-y-5">
            <div className="grid grid-cols-3 gap-3">
              <div className="bg-zinc-50 border border-zinc-200 p-3"><p className="text-xs text-zinc-500">订单数量</p><p className="text-xl font-bold mt-1">{info?.ordered_quantity ?? item.quantity}</p></div>
              <div className="bg-emerald-50 border border-emerald-200 p-3"><p className="text-xs text-emerald-700">已确认合格</p><p className="text-xl font-bold text-emerald-700 mt-1">{info?.confirmed_quantity ?? 0}</p></div>
              <div className="bg-amber-50 border border-amber-200 p-3"><p className="text-xs text-amber-700">待报工数量</p><p className="text-xl font-bold text-amber-700 mt-1">{info?.remaining_quantity ?? item.quantity}</p></div>
            </div>

            <div className="bg-slate-50 rounded-xl p-3 text-xs text-slate-500 leading-6">投产 {info?.production_quantity} · 报废 {info?.scrap_quantity} · 备用 {info?.reserve_quantity}<br />交付缺口 {info?.delivery_gap} · 剩余可计产值数量 {info?.billable_remaining}</div>
            {error && <div className="bg-rose-50 border border-rose-200 text-rose-700 px-3 py-2 text-sm flex items-start gap-2"><AlertCircle className="w-4 h-4 mt-0.5 shrink-0"/>{error}</div>}

            <div className="grid sm:grid-cols-2 gap-4">
              <label className="block"><span className="text-sm font-medium">本次合格数量</span><input type="number" min="1" step="1" max={info?.remaining_quantity} value={quantity} onChange={event => setQuantity(event.target.value)} className="mt-1 w-full h-10 border border-zinc-300 px-3 outline-none focus:ring-2 focus:ring-zinc-900" required/></label>
              <label className="block"><span className="text-sm font-medium">实际完工日期</span><input type="date" max={maxDate} value={businessDate} onChange={event => setBusinessDate(event.target.value)} className="mt-1 w-full h-10 border border-zinc-300 px-3 outline-none focus:ring-2 focus:ring-zinc-900" required/></label>
            </div>
            {latestBusinessDate && <p className="text-xs text-blue-700">完工日期须不晚于本次交货日期 {latestBusinessDate}，请按实际完工日期填写。</p>}
            <label className="block"><span className="text-sm font-medium">备注 / 完工依据（{policy.review_notes ? '必填' : '选填'}）</span><textarea required={policy.review_notes} value={notes} onChange={event => setNotes(event.target.value)} rows={3} maxLength={500} className="mt-1 w-full border border-zinc-300 px-3 py-2 outline-none focus:ring-2 focus:ring-zinc-900" placeholder="例如：质检单号、分批说明"/></label>

            <div className="bg-blue-50 border border-blue-200 text-blue-800 p-3 text-xs leading-5">
              本次计产值数量：{Math.max(0, Math.min(Number(quantity) || 0, info?.billable_remaining || 0))}；本次备用数量：{Math.max(0, (Number(quantity) || 0) - (info?.billable_remaining || 0))}。超出订单的合格件不计产值，已结账月份不能补录。
            </div>

            <div className="flex justify-end gap-2 pt-1">
              {!embedded && <button type="button" onClick={onClose} disabled={saving} className="h-10 px-4 border border-zinc-300 text-sm font-medium hover:bg-zinc-50">取消</button>}
              <button type="submit" disabled={saving || !info || info.remaining_quantity <= 0} className="h-10 px-4 bg-zinc-900 text-white text-sm font-medium flex items-center gap-2 hover:bg-zinc-800 disabled:opacity-40">
                {saving ? <Loader2 className="w-4 h-4 animate-spin"/> : <CheckCircle2 className="w-4 h-4"/>}确认合格完工
              </button>
            </div>
          </form>)}
      </div>
    </div>);
};
export default CompletionModal;
