import type { OrderProcess } from '../types';
import CuttingSpecification, { isCuttingProcess } from './CuttingSpecification';
import { processDate } from '../utils/processDates';

export function invalidOrderProcessDates(processes: OrderProcess[]): boolean {
  return processes.some(p => {
    const start = processDate(p.planned_start_at), end = processDate(p.planned_end_at);
    return p.name !== '送货' && (!!start !== !!end || !!start && end < start);
  });
}

export default function OrderProcessDates({ processes, itemIndex, disabled, onUpdate }: {
  processes: OrderProcess[]; itemIndex: number; disabled: boolean;
  onUpdate: (processes: OrderProcess[]) => void;
}) {
  const production = processes.map((process, index) => ({ process, index })).filter(p => p.process.name !== '送货');
  if (!production.length) return null;
  const update = (index: number, field: 'planned_start_at' | 'planned_end_at', value: string) => {
    // Include both nodes so clearing a pair is explicit; existing actual nodes are retained.
    onUpdate(processes.map((p, i) => i !== index ? p : {
      ...p, planned_start_at: p.planned_start_at || '', planned_end_at: p.planned_end_at || '', [field]: value,
    }));
  };
  return <details className="rounded-xl bg-blue-50/40 border border-blue-100">
    <summary className="text-sm font-semibold text-blue-700 p-3 cursor-pointer">工序计划日期（选填）</summary>
    <div className="px-3 pb-3 space-y-3">
      <p className="text-xs text-slate-500">按天安排，每道工序的开始与结束一起填写，可为同一天。实际开工／完工在生产时记录。</p>
      {production.map(({ process: p, index }) => <div key={p.id || index} className="grid grid-cols-1 sm:grid-cols-3 gap-3 items-end rounded-lg bg-white p-3 border border-blue-100">
        <div className="text-sm font-semibold pb-2">{index + 1}. {p.name}{p.is_outsourced && <span className="ml-2 text-xs text-violet-600">外协</span>}</div>
        <label className="erp-label">计划开始日期<input type="date" className="erp-input" disabled={disabled} value={processDate(p.planned_start_at)} aria-label={`零件${itemIndex + 1}工序${index + 1}${p.name}计划开始日期`} onChange={e => update(index, 'planned_start_at', e.target.value)}/></label>
        <label className="erp-label">计划结束日期<input type="date" className="erp-input" disabled={disabled} value={processDate(p.planned_end_at)} min={processDate(p.planned_start_at) || undefined} aria-label={`零件${itemIndex + 1}工序${index + 1}${p.name}计划结束日期`} onChange={e => update(index, 'planned_end_at', e.target.value)}/></label>
      {isCuttingProcess(p.name) && <CuttingSpecification value={p.cutting_spec} disabled={disabled} label={"零件"+(itemIndex+1)+"下料尺寸规格"} onChange={value=>onUpdate(processes.map((step,i)=>i===index?{...step,cutting_spec:value}:step))}/>}
      </div>)}
    </div>
  </details>;
}
