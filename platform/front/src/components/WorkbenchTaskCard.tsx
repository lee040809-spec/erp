
import { ArrowUpRight, CalendarDays, CheckCircle2, SlidersHorizontal, UserRound } from 'lucide-react';
import { StatusBadge } from './shared';
import { processDate } from '../utils/processDates';
import type { ProductionTask } from '../types/productionWorkbench';

interface Props {
  task: ProductionTask;
  density: 'compact' | 'comfortable';
  canManage: boolean;
  saving: boolean;
  selected: boolean;
  today: string;
  onSelect: (checked: boolean) => void;
  onOpenOrder: () => void;
  onDetails: () => void;
  onQuantities: () => void;
  onUnblock: () => void;
  onDrawing: () => void;
}

// Presentation only: requests, permissions and business writes belong to the workbench page.
export default function WorkbenchTaskCard(p: Props) {
  const { task, canManage, saving, density } = p;
  const steps = task.processes.filter(step => step.name !== '送货');
  const done = steps.filter(step => step.status === 'completed').length;
  const overdue = !!task.due_date && task.due_date < p.today;
  const orderDate = processDate(task.start_date);
  const border = task.blocked_reason ? 'border-l-amber-400' : overdue ? 'border-l-rose-400' : 'border-l-transparent';
  return <article className={`erp-workbench-task border-l-[3px] ${border} ${density === 'compact' ? 'px-4 py-3' : 'p-5'} hover:bg-slate-50/40`}>
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div className="flex gap-3 items-start">
        {canManage && <input type="checkbox" className="mt-2" aria-label={`选择${task.part_name}`} checked={p.selected} onChange={e => p.onSelect(e.target.checked)}/>}
        <div>
          <div className="flex items-center gap-3 flex-wrap">
            <button className="font-semibold text-sm text-slate-900 hover:text-blue-600" onClick={p.onDetails}>{task.part_name}</button>
            <StatusBadge status={task.status}/>
            {task.priority === 'high' && <span className="text-[10px] bg-rose-50 text-rose-600 px-2 py-1 rounded-md font-semibold">紧急</span>}
            {overdue && task.delivery_gap > 0 && <span className="text-[10px] bg-rose-50 text-rose-600 px-2 py-1 rounded-md">交期逾期</span>}
          </div>
          <div className="text-xs text-slate-400 mt-1.5 flex flex-wrap gap-2">
            <span className="text-slate-500">{task.customer_name}</span>
            <button onClick={p.onOpenOrder} className="erp-workbench-order-number hover:text-blue-600">{task.order_number} ↗</button>
            <span className="erp-workbench-order-date whitespace-nowrap" title="零件下单日期；未填写时使用所属订单日期">下单 {orderDate || '未填'}</span>
            <span>{task.part_number || '无图号'}</span>
          </div>
        </div>
      </div>
      <div className="flex gap-2 flex-wrap">
        <button onClick={p.onDetails} disabled={saving} className="erp-btn erp-primary py-1 min-h-8">办理零件</button>
        <button onClick={p.onOpenOrder} aria-label={`查看订单${task.order_number}的${task.part_name}`} className="erp-btn py-1 min-h-8">订单资料<ArrowUpRight size={14}/></button>
      </div>
    </div>
    <div className={density === 'compact' ? 'erp-part-metrics my-2 text-xs' : 'grid grid-cols-2 sm:grid-cols-4 xl:grid-cols-6 gap-3 my-4 text-xs'}>
      <div><p className="text-slate-400">订单 / 投产</p><p className="font-semibold text-sm text-slate-800 mt-1">{task.quantity.toLocaleString()} <span className="text-slate-300 mx-1">/</span> {(task.production_quantity || task.quantity).toLocaleString()}{canManage && <button aria-label={`调整${task.part_name}投产数量`} title="调整投产数量" className="text-blue-500 ml-2" onClick={p.onQuantities}><SlidersHorizontal size={13}/></button>}</p></div>
      <div><p className="text-slate-400">合格 / 备用 / 报废</p><p className="mt-1 text-sm font-semibold"><span className="text-emerald-600">{task.qualified_quantity}</span><span className="text-slate-300 mx-1">/</span><span className="text-violet-600">{task.reserve_quantity}</span><span className="text-slate-300 mx-1">/</span><span className={task.scrap_quantity ? 'text-rose-600' : 'text-slate-600'}>{task.scrap_quantity}</span></p></div>
      <div><p className="text-slate-400">交付缺口 / 待报工</p><p className="mt-1 text-sm font-semibold text-slate-700">{task.delivery_gap} <span className="text-slate-300 mx-1">/</span> {task.remaining_quantity}</p></div>
      <div><p className="flex items-center gap-1 text-slate-400"><UserRound size={12}/>负责人 / 设备</p><p className="mt-1 text-slate-700 truncate">{task.assignee || '待安排'} / {task.equipment || '待指定'}</p></div>
      <div><p className="text-slate-400">计划完工 / 预计工时</p><p className="mt-1 text-slate-700">{task.planned_end || '未安排'} · {task.estimated_hours} h</p></div>
      <div><p className="flex items-center gap-1 text-slate-400"><CalendarDays size={12}/>客户交期</p><p className={`mt-1 font-medium ${overdue && task.delivery_gap > 0 ? 'text-rose-600' : 'text-slate-700'}`}>{task.due_date || '未填写'}</p></div>
    </div>
    {!!task.flow?.batches.length && <p className="text-xs text-blue-600 mb-2">{task.flow.batches.map(b=>`${b.batch_no} ${b.quantity}件 · ${b.due_date.slice(0,10)} · 已交${b.delivered_quantity}`).join("；")}</p>}
 {task.blocked_reason && <div className="text-xs text-amber-700 bg-amber-50 rounded-xl px-3 py-2 flex items-center justify-between mb-3 gap-2"><span>受阻：{task.blocked_reason}</span>{canManage && <button disabled={saving} onClick={p.onUnblock} className="underline shrink-0">解除</button>}</div>}
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div className={density === 'compact' ? 'flex flex-wrap items-center gap-1.5' : 'flex flex-wrap items-center gap-2 mb-3'}>
        <span className="text-[11px] text-slate-400 mr-1">工序 {done}/{steps.length}</span>
        {steps.map(step => <span key={step.id} className={`erp-step erp-step-${step.status}`}>{step.status === 'completed' && <CheckCircle2 size={12}/>}<span>{step.name}</span>{step.is_outsourced && <span className="text-[10px] opacity-70">外协</span>}</span>)}
      </div>
      <div className="flex items-center justify-between flex-wrap gap-3"><div className="flex gap-2 flex-wrap">
        {task.has_drawing && <button onClick={p.onDrawing} className="erp-btn text-slate-500 py-1 min-h-8">图纸</button>}
      </div></div>
    </div>
    {task.notes && <p className="mt-2 text-xs text-slate-400 whitespace-pre-wrap">{task.notes}</p>}
  </article>;
}
