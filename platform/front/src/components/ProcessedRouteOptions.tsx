
import type { OrderProcess } from '../types';
import { processDate } from '../utils/processDates';
export default function ProcessedRouteOptions({enabled,reason,disabled,onEnabled,onReason,archived=[],historyOnly=false}:{enabled:boolean;reason:string;disabled:boolean;onEnabled:(value:boolean)=>void;onReason:(value:string)=>void;archived?:OrderProcess[];historyOnly?:boolean}) {
 return <div className="space-y-3">{!historyOnly && <label className="flex items-center gap-2 text-sm font-semibold text-blue-700"><input type="checkbox" aria-label="允许修改删除已加工工序" disabled={disabled} checked={enabled} onChange={e=>onEnabled(e.target.checked)}/>允许修改 / 删除加工中、已完成工序</label>}
 {enabled && <><p className="text-xs text-slate-500">删除后从当前工艺与甘特图移除，原开工、完工、报废、外协及费用记录保留。未发出的外协单、待回厂或返工须先处理；保存时会校验后续数量及已确认产值。</p><label className="erp-label">工艺调整原因 *<textarea aria-label="已加工工序调整原因" required maxLength={500} className="erp-input" disabled={disabled} value={reason} onChange={e=>onReason(e.target.value)} placeholder="例如：工艺变更，取消此道加工；或更正原工序录入"/></label></>}
 {!!archived.length && <details className="text-xs text-slate-500"><summary className="cursor-pointer">已移除工序（最近100道，共显示{archived.length}道）</summary><div className="space-y-2 mt-2">{archived.map(p=><p key={p.id}><strong>{p.name}</strong> · {{pending:'待加工',processing:'加工中',completed:'已完成'}[p.status]} · 开工 {processDate(p.actual_start_at)||'未记录'} · 完工 {processDate(p.actual_end_at)||'未记录'} · 移除 {processDate(p.deleted_at)}</p>)}</div></details>}
 </div>;
}
