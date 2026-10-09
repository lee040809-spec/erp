import { businessToday } from '../utils/businessDate';
import React, { useState } from 'react';
import { AlertCircle, Loader2, X } from 'lucide-react';
import { createCostEntryApi } from '../services/api';

interface CostEntryModalProps {
  onClose: () => void;
  onSuccess: () => void;
}



const costTypes = [
  ['material', '自购材料耗用'], ['outsourcing', '外协费用'], ['tool', '刀具费用'],
  ['fixture', '工装费用'], ['labor', '直接人工'], ['manufacturing', '制造费用'],
  ['other_direct', '其他直接费用'], ['period', '期间费用']
];

const CostEntryModal: React.FC<CostEntryModalProps> = ({ onClose, onSuccess }) => {
  const [form, setForm] = useState({ cost_type: 'material', amount: '', tax_amount: '0.00', net_amount: '', business_date: businessToday(), order_id: '', order_item_id: '', supplier_name: '', notes: '' });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [taxInCost, setTaxInCost] = useState(true);
  const accountingAmount = taxInCost ? form.amount : ((Math.round(Number(form.amount) * 100) - Math.round(Number(form.tax_amount || 0) * 100)) / 100).toFixed(2);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSaving(true);
    setError('');
    try {
      await createCostEntryApi({
        cost_type: form.cost_type,
        amount: form.amount,
        tax_amount: form.tax_amount || '0.00',
        net_amount: accountingAmount,
        business_date: form.business_date,
        order_id: form.order_id ? Number(form.order_id) : undefined,
        order_item_id: form.order_item_id ? Number(form.order_item_id) : undefined,
        supplier_name: form.supplier_name.trim() || undefined,
        notes: form.notes.trim() || undefined
      });
      onSuccess();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : '成本费用保存失败');
    } finally {
      setSaving(false);
    }
  };

  const set = (key: keyof typeof form, value: string) => setForm(current => ({ ...current, [key]: value }));

  return (
    <div className="fixed inset-0 z-50 bg-black/40 flex items-center justify-center p-4" onMouseDown={event => event.target === event.currentTarget && onClose()}>
      <div className="bg-white w-full max-w-2xl border border-zinc-200 shadow-2xl">
        <div className="px-5 py-4 border-b border-zinc-200 flex justify-between items-start">
          <div><h3 className="font-bold text-lg">录入成本费用</h3><p className="text-sm text-zinc-500 mt-1">采购付款与实际耗用应分别记录；这里录入归属于经营核算的成本。</p></div>
          <button onClick={onClose} className="p-1 text-zinc-400 hover:text-zinc-900"><X className="w-5 h-5" /></button>
        </div>
        <form onSubmit={submit} className="p-5 space-y-4">
          {error && <div className="bg-rose-50 border border-rose-200 text-rose-700 px-3 py-2 text-sm flex gap-2"><AlertCircle className="w-4 h-4 mt-0.5" />{error}</div>}
          <div className="grid sm:grid-cols-2 gap-4">
            <label><span className="text-sm font-medium">费用类型</span><select value={form.cost_type} onChange={e => set('cost_type', e.target.value)} className="mt-1 w-full h-10 border border-zinc-300 bg-white px-3">{costTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
            <label><span className="text-sm font-medium">业务日期</span><input type="date" required value={form.business_date} onChange={e => set('business_date', e.target.value)} className="mt-1 w-full h-10 border border-zinc-300 px-3" /></label>
            <label><span className="text-sm font-medium">含税金额</span><input required inputMode="decimal" pattern="\d+(\.\d{1,2})?" value={form.amount} onChange={e => set('amount', e.target.value)} placeholder="0.00" className="mt-1 w-full h-10 border border-zinc-300 px-3" /></label>
            <label><span className="text-sm font-medium">税额</span><input inputMode="decimal" pattern="\d+(\.\d{1,2})?" value={form.tax_amount} onChange={e => set('tax_amount', e.target.value)} className="mt-1 w-full h-10 border border-zinc-300 px-3" /></label>
            <label><span className="text-sm font-medium">税额核算方式</span><select value={taxInCost ? 'included' : 'excluded'} onChange={e => setTaxInCost(e.target.value === 'included')} className="mt-1 w-full h-10 border border-zinc-300 px-3"><option value="included">税额计入成本（含不可抵扣税）</option><option value="excluded">税额不计入成本（需确认可抵扣）</option></select><span className="block text-xs text-zinc-500 mt-1">核算金额：{accountingAmount || '0.00'} 元，请按实际税务口径选择。</span></label>
            <label><span className="text-sm font-medium">供应商／收款方</span><input value={form.supplier_name} onChange={e => set('supplier_name', e.target.value)} className="mt-1 w-full h-10 border border-zinc-300 px-3" /></label>
            <label><span className="text-sm font-medium">订单 ID（可选）</span><input type="number" min="1" value={form.order_id} onChange={e => set('order_id', e.target.value)} className="mt-1 w-full h-10 border border-zinc-300 px-3" /></label>
            <label><span className="text-sm font-medium">零件 ID（可选）</span><input type="number" min="1" value={form.order_item_id} onChange={e => set('order_item_id', e.target.value)} className="mt-1 w-full h-10 border border-zinc-300 px-3" /></label>
          </div>
          <label className="block"><span className="text-sm font-medium">备注</span><textarea rows={3} maxLength={500} value={form.notes} onChange={e => set('notes', e.target.value)} className="mt-1 w-full border border-zinc-300 px-3 py-2" /></label>
          <div className="flex justify-end gap-2"><button type="button" onClick={onClose} className="h-10 px-4 border border-zinc-300 text-sm">取消</button><button type="submit" disabled={saving} className="h-10 px-4 bg-zinc-900 text-white text-sm flex items-center gap-2 disabled:opacity-50">{saving && <Loader2 className="w-4 h-4 animate-spin" />}保存成本费用</button></div>
        </form>
      </div>
    </div>
  );
};

export default CostEntryModal;
