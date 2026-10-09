import React, { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import type { FlowData } from '../types/productionFlow';
import { flowRequest } from '../services/productionFlow';
import ProcessQuantityPanel from './ProcessQuantityPanel';
import BatchDeliveryPanel from './BatchDeliveryPanel';
import OutsourcingPanel from './OutsourcingPanel';

export default function ProductionFlowModal({ itemID, partName, initialTab = 'process', onClose, onChanged }: { itemID: number; partName: string; initialTab?: 'process' | 'batch' | 'outsourcing'; onClose: () => void; onChanged: () => void }) {
  const [tab, setTab] = useState(initialTab), [data, setData] = useState<FlowData | null>(null), [error, setError] = useState('');
  const refresh = async () => { setData(await flowRequest<FlowData>(`/production/items/${itemID}/flow`)); };
  useEffect(() => { let active = true; flowRequest<FlowData>(`/production/items/${itemID}/flow`).then(v => { if (active) setData(v); }).catch(e => { if (active) setError(e.message); }); return () => { active = false; }; }, [itemID]);
  const changed = async () => { await refresh(); onChanged(); };
  return <div className="erp-overlay z-[120]"><section role="dialog" aria-modal="true" aria-label={`分批生产 ${partName}`} className="erp-dialog erp-production-flow-dialog w-full max-w-5xl flex flex-col overflow-hidden"><header className="border-b border-slate-100 px-4 sm:px-6 py-4 flex justify-between items-center gap-2"><div><h3 className="text-lg font-bold">分批生产 · {partName}</h3><p className="text-xs text-slate-400 mt-1">共用工序数量流转 · 分批完工交货 · 外协费用闭环</p></div><button type="button" className="erp-btn" aria-label="关闭分批生产" onClick={onClose}><X size={18}/></button></header>
    <nav className="flex gap-2 px-4 pt-3 flex-wrap">{([['process', '工序数量'], ['batch', '分批交货'], ['outsourcing', '外协流转']] as const).map(([key, label]) => <button type="button" className={`erp-btn ${tab === key ? 'erp-primary' : ''}`} key={key} onClick={() => setTab(key)}>{label}</button>)}</nav>
    <div className="overflow-y-auto p-4 sm:p-6 space-y-3">{error && <p role="alert" className="text-rose-600">{error}</p>}{!data && !error && <p className="text-slate-400">正在读取生产台账…</p>}{data && <>{tab === 'process' && <ProcessQuantityPanel data={data} onChanged={changed} onOutsourcing={() => setTab('outsourcing')}/>} {tab === 'batch' && <BatchDeliveryPanel data={data} onChanged={changed}/>} {tab === 'outsourcing' && <OutsourcingPanel data={data} onChanged={changed}/>}</>}</div>
  </section></div>;
}
