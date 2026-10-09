import React, {useState} from 'react';
export default function OutsourcingRemoveDialog({title,description,onConfirm,onClose}:{title:string;description:string;onConfirm:(reason:string)=>Promise<boolean>;onClose:()=>void}) {
 const [reason,setReason]=useState(''),[busy,setBusy]=useState(false);
 return <section role="alertdialog" aria-label={title} className="erp-card border-rose-200 bg-rose-50/40 p-3 space-y-3">
  <h4 className="font-semibold text-sm text-rose-700">{title}</h4><p className="text-xs text-slate-600">{description}</p>
  <form className="space-y-3" onSubmit={async e=>{e.preventDefault();if(busy)return;setBusy(true);try{if(await onConfirm(reason))onClose()}finally{setBusy(false)}}}>
   <label className="erp-label">删除 / 撤销原因<input className="erp-input" autoFocus required maxLength={500} value={reason} onChange={e=>setReason(e.target.value)}/></label>
   <div className="flex flex-wrap gap-2"><button className="erp-btn erp-danger" disabled={busy||!reason.trim()}>确认删除 / 撤销</button><button className="erp-btn" type="button" disabled={busy} onClick={onClose}>保留记录</button></div>
  </form>
 </section>;
}
