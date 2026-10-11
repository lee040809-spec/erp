import { useEffect, useState } from 'react';
import type {OutsourcingDispatch} from '../types/productionFlow';
import {flowDate,flowRequest} from '../services/productionFlow';
export default function OutsourcingDispatchEditor({row,onSave,onClose,onBusyChange}:{row:OutsourcingDispatch;onSave:()=>Promise<void>;onClose:()=>void;onBusyChange?:(busy:boolean)=>void}){
 const [expected,setExpected]=useState(flowDate(row.expected_return_date)),[notes,setNotes]=useState(row.notes||''),[busy,setBusy]=useState(false),[error,setError]=useState('');
 useEffect(()=>{onBusyChange?.(busy);return()=>onBusyChange?.(false)},[busy,onBusyChange]);
 return <form className="erp-card p-3 border-blue-200 space-y-3" onSubmit={async e=>{e.preventDefault();if(busy)return;setBusy(true);setError('');try{await flowRequest(`/outsourcing/${row.id}`,'PATCH',{edit_token:row.edit_token,expected_return_date:expected,notes});await onSave();onClose()}catch(e){setError((e as Error).message)}finally{setBusy(false)}}}>
  <h4 className="font-semibold text-sm">修改外协发出记录 · {row.supplier_name}</h4>
  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3"><label className="erp-label">预计回厂日期<input className="erp-input" autoFocus type="date" required min={flowDate(row.dispatch_date)} value={expected} onChange={e=>setExpected(e.target.value)}/></label><label className="erp-label">说明<input className="erp-input" maxLength={500} value={notes} onChange={e=>setNotes(e.target.value)}/></label></div>
  <p className="text-xs text-slate-500">发出数量、工序、供应商及单价属于实物与费用依据。误录时先“删除 / 撤销发出”，再按正确信息登记；已有回厂或入账的记录保留原依据。</p>
  <div className="flex gap-2"><button className="erp-btn erp-primary" disabled={busy}>保存修改</button><button type="button" className="erp-btn" disabled={busy} onClick={onClose}>取消修改</button></div>{error&&<p role="alert" className="text-rose-600 text-sm">{error}</p>}
 </form>;
}
