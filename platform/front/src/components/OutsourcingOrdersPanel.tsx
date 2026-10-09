import React, { useEffect, useRef, useState } from 'react';
import type { FlowData } from '../types/productionFlow';
import type { OutsourcingOrder, OutsourcingOrderLine, OutsourcingOrderPage } from '../types/outsourcingOrder';
import { flowDate, flowRequest, flowToday, withRequestID } from '../services/productionFlow';
import OutsourcingOrderEditor from './OutsourcingOrderEditor';
import OutsourcingRemoveDialog from './OutsourcingRemoveDialog';
import OutsourcingSuppliers from '../pages/OutsourcingSuppliers';

const labels: Record<string,string> = { ordered:'已下单待发出', partial_dispatched:'部分发出', dispatched:'已全部发出', cancelled:'已取消', deleted:'已删除（保留记录）' };
export default function OutsourcingOrdersPanel({data,onChanged}:{data:FlowData;onChanged:()=>Promise<void>}) {
 const [orders,setOrders]=useState<OutsourcingOrderPage>({items:[],page:1,has_more:false});
 const [page,setPage]=useState(1),[editing,setEditing]=useState<OutsourcingOrder|null>(null),[remove,setRemove]=useState<{order:OutsourcingOrder;line?:OutsourcingOrderLine}|null>(null);
 const [error,setError]=useState(''),[message,setMessage]=useState(''),[saving,setSaving]=useState(false),[manage,setManage]=useState(false),[refresh,setRefresh]=useState(0);
 const [dispatch,setDispatch]=useState<{order:OutsourcingOrder;line?:OutsourcingOrderLine}|null>(null);
 const [sendQty,setSendQty]=useState(''),[sendDate,setSendDate]=useState(flowToday()),[sendExpected,setSendExpected]=useState(flowToday()),[sendNotes,setSendNotes]=useState('');
 const pending=useRef({signature:'',id:''}),busy=useRef(false);
 const itemID=data.flow.item.id;
 const load=async()=>setOrders(await flowRequest<OutsourcingOrderPage>(`/outsourcing-orders?order_item_id=${itemID}&page=${page}&pageSize=20`));
 useEffect(()=>{let active=true;flowRequest<OutsourcingOrderPage>(`/outsourcing-orders?order_item_id=${itemID}&page=${page}&pageSize=20`).then(v=>{if(active)setOrders(v)}).catch(e=>{if(active)setError(e.message)});return()=>{active=false}},[itemID,page,data.flow]);
 const act=async(fn:()=>Promise<unknown>,success:string)=>{
  if(busy.current)return false;busy.current=true;setSaving(true);setError('');setMessage('');
  try{await fn();await load();await onChanged();setDispatch(null);setMessage(success);pending.current={signature:'',id:''};return true}catch(e){setError((e as Error).message);return false}finally{busy.current=false;setSaving(false)}
 };
 const openDispatch=(order:OutsourcingOrder,line?:OutsourcingOrderLine)=>{
  setDispatch({order,line});setSendQty(String(line?line.quantity-line.dispatched_quantity:order.lines[0].quantity-order.lines[0].dispatched_quantity));
  setSendDate(flowToday());setSendExpected(flowDate(order.expected_return_date)<flowToday()?flowToday():flowDate(order.expected_return_date));setSendNotes('');setError('');
 };
 const processes=data.flow.processes.filter(p=>p.name!=='送货');
 return <section className="space-y-3">
  <div className="flex flex-wrap justify-between gap-2"><div><h3 className="font-bold text-sm">外协订单 · 先下单，再发出</h3><p className="text-xs text-slate-500 mt-1">前道未开工也可下单。同一道工序可分别向多个供应商下单，合计预留不能超过投产余量。</p></div><button type="button" className="erp-btn" onClick={()=>setManage(v=>!v)}>{manage?'收起供应商资料':'管理供应商资料'}</button></div>
  {manage&&<div className="erp-card p-3"><OutsourcingSuppliers embedded/><button className="erp-btn mt-3" onClick={()=>{setManage(false);setRefresh(n=>n+1)}}>完成资料维护，刷新供应商选项</button></div>}
  {data.can_manage&&<OutsourcingOrderEditor key={editing?.id||'new'} data={data} refresh={refresh} initial={editing||undefined} defaultOpen={!orders.items.length} onClose={()=>setEditing(null)} onSaved={async()=>{await load();await onChanged();setMessage(editing?'外协工序及下单信息已修改，预留数量已同步。':'外协订单已保存。')}}/>}
  {orders.items.map(o=><article key={o.id} className="erp-card p-3 space-y-2"><div className="flex flex-wrap justify-between gap-2"><strong className="text-sm">{o.supplier_name} · {o.scope==='full'?'整套工序外协':'指定工序外协'}</strong><span className={`text-xs ${o.status==='cancelled'?'text-slate-400':'text-violet-700'}`}>{labels[o.status]||o.status}</span></div><p className="text-xs text-slate-500 break-all">{o.order_no} · 下单 {flowDate(o.order_date)} · 要求回厂 {flowDate(o.expected_return_date)}</p>
   {o.scope==='full'?<><p className="text-xs">{o.lines.map(l=>l.process_name).join(' → ')} · 下单 {o.lines[0]?.quantity} · 已发 {o.lines[0]?.dispatched_quantity}{o.unit_price!==undefined?` · 整套单件价 ${o.unit_price}`:''}</p>{data.can_manage&&!['cancelled','deleted'].includes(o.status)&&o.lines[0]&&o.lines[0].quantity>o.lines[0].dispatched_quantity&&<button className="erp-btn erp-primary" disabled={!data.flow.tracking} onClick={()=>openDispatch(o)}>整套实物发出</button>}</>:o.lines.map(l=><div key={l.id} className="flex flex-wrap justify-between items-center gap-2 border-t border-slate-100 pt-2"><p className="text-xs">{l.process_name} · 下单 {l.quantity} · 已发 {l.dispatched_quantity} · 待发 {l.quantity-l.dispatched_quantity}{l.unit_price!==undefined?` · 单价 ${l.unit_price}`:''}</p>{data.can_manage&&!['cancelled','deleted'].includes(o.status)&&l.quantity>l.dispatched_quantity&&<button className="erp-btn" disabled={!data.flow.tracking} onClick={()=>openDispatch(o,l)}>登记{l.process_name}实物发出</button>}{data.can_manage&&!['cancelled','deleted'].includes(o.status)&&o.lines.every(v=>v.dispatched_quantity===0)&&<div className="flex gap-2"><button className="erp-btn" disabled={saving} onClick={()=>{setEditing(o);setRemove(null)}}>修改工序</button><button className="erp-btn text-rose-600" disabled={saving} onClick={()=>{setRemove({order:o,line:l});setEditing(null)}}>删除工序</button></div>}</div>)}
   {!data.flow.tracking&&!['cancelled','deleted'].includes(o.status)&&<p className="text-xs text-amber-700">订单已成立；在“工序数量”核对并启用数量跟踪后，才能登记真实发出。</p>}{data.can_manage&&o.status!=='deleted'&&o.lines.every(l=>l.dispatched_quantity===0)&&<div className="flex flex-wrap gap-2">{o.status!=='cancelled'&&<button className="erp-btn" disabled={saving} onClick={()=>{setEditing(o);setRemove(null);setError('')}}>修改外协工序 / 下单</button>}<button className="erp-btn text-rose-600" disabled={saving} onClick={()=>{setRemove({order:o});setEditing(null)}}>删除外协订单</button></div>}
   {data.can_manage&&!['cancelled','deleted'].includes(o.status)&&o.lines.some(l=>l.dispatched_quantity>0)&&<p className="text-xs text-slate-500">已有实物发出，工序与报价已锁定；误录请先在下方撤销尚未回厂的发出记录。</p>}

  </article>)}
  {!orders.items.length&&<p className="text-xs text-slate-400">尚无外协订单，原有发出记录保留在下方。</p>}
  <div className="flex gap-2 items-center text-xs"><button className="erp-btn" disabled={page<=1} onClick={()=>setPage(p=>p-1)}>前一页外协订单</button><span>第 {page} 页</span><button className="erp-btn" disabled={!orders.has_more} onClick={()=>setPage(p=>p+1)}>后一页外协订单</button></div>
  {dispatch&&<form className="erp-card border-blue-200 p-3 space-y-3" onSubmit={e=>{e.preventDefault();act(()=>flowRequest(`/outsourcing-orders/${dispatch.order.id}/dispatch`,'POST',withRequestID(pending.current,{order_line_id:dispatch.line?.id,quantity:Number(sendQty),dispatch_date:sendDate,expected_return_date:sendExpected,notes:sendNotes})),'实物发出已登记；开工数量与外协待回数量已更新。')}}><h4 className="font-semibold text-sm">{dispatch.order.supplier_name} · {dispatch.line?.process_name||'整套工序'}实物发出</h4><div className="grid grid-cols-1 sm:grid-cols-2 gap-3"><label className="erp-label">本次实物数量<input className="erp-input" type="number" required min={1} step={1} value={sendQty} onChange={e=>setSendQty(e.target.value)}/></label><label className="erp-label">实际发出日期<input className="erp-input" type="date" required min={flowDate(dispatch.order.order_date)} max={flowToday()} value={sendDate} onChange={e=>setSendDate(e.target.value)}/></label><label className="erp-label">预计回厂日期<input className="erp-input" type="date" required min={sendDate} value={sendExpected} onChange={e=>setSendExpected(e.target.value)}/></label><label className="erp-label">发出说明<input className="erp-input" maxLength={500} value={sendNotes} onChange={e=>setSendNotes(e.target.value)}/></label></div><p className="text-xs text-slate-500">实物数量校验前道已完工数量。整套外协从首道开工；回厂质检时汇总整套工序进度，不需要逐道重复登记。</p><div className="flex gap-2"><button className="erp-btn erp-primary" disabled={saving}>确认实物发出</button><button type="button" className="erp-btn" disabled={saving} onClick={()=>setDispatch(null)}>取消本次登记</button></div></form>}
  {remove&&<OutsourcingRemoveDialog title={remove.line?`删除外协工序 · ${remove.line.process_name}`:'删除未发出的外协订单'} description="删除会释放该工序预留数量。最后一道工序删除时会一并删除外协订单；已有发出记录不能直接删除。系统保留操作记录。" onClose={()=>setRemove(null)} onConfirm={reason=>act(async()=>{
    const o=remove.order;
    if(!remove.line||o.lines.length===1){await flowRequest(`/outsourcing-orders/${o.id}`,'DELETE',{edit_token:o.edit_token,reason});return}
    await flowRequest(`/outsourcing-orders/${o.id}`,'PATCH',withRequestID(pending.current,{order_item_id:itemID,supplier_id:o.supplier_id,scope:o.scope,delivery_batch_id:o.delivery_batch_id,order_date:flowDate(o.order_date),expected_return_date:flowDate(o.expected_return_date),unit_price:o.unit_price||'0',notes:o.notes,change_reason:reason,edit_token:o.edit_token,lines:o.lines.filter(l=>l.id!==remove.line?.id).map(l=>({order_process_id:l.order_process_id,quantity:l.quantity,unit_price:l.unit_price||'0'}))}));
   },'已删除，外协预留数量已同步释放。')}/>}
  {error&&<p role="alert" className="text-sm text-rose-600">{error}</p>}{message&&<p role="status" className="text-sm text-emerald-600">{message}</p>}
 </section>;
}
