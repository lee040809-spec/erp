import { businessToday } from '../utils/businessDate';
import PartWorkflowModal from '../components/PartWorkflowModal';
import type {WorkflowStep} from '../utils/partWorkflow';
import React, { useEffect, useRef, useState } from 'react';
import { Search, RefreshCw, Plus, ArrowUpRight, CalendarDays, UserRound, CheckCircle2, X, SlidersHorizontal, Boxes, AlertTriangle, Clock3 } from 'lucide-react';
import { authFetch } from '../components/shared';
import ProcessGanttChart from '../components/ProcessGanttChart';
import { Order } from '../types';
import type { ProductionTask as Task } from '../types/productionWorkbench';
import WorkbenchTaskCard from '../components/WorkbenchTaskCard';
import { WorkbenchOrderTarget, WorkbenchState, WorkbenchView, workbenchViewDescriptions } from '../services/workbenchNavigation';
interface Counts {
    orders: number;
    items: number;
    pieces: number;
}
interface Data {
    can_manage: boolean;
    summary: Counts;
    metrics: Record<string, Counts>;
    items: Task[];
    total: number;
    stats: {
        total: number;
        unassigned: number;
        today: number;
        overdue: number;
        blocked: number;
        outsourced: number;
    };
}
interface Props {
    setShowDrawingModal: (data: string) => void;
    resetAndOpenModal: () => void;
    setActiveTab: (tab: string) => void;
    setOrderFilters: (value: any) => void;
    setAppliedOrderFilters: (value: any) => void;
    setCurrentPage: (page: number) => void;
    orderView?: 'orders' | 'production_orders';
    onOpenOrder?: (target: WorkbenchOrderTarget) => void;
    initialState?: WorkbenchState;
}
const initial: Data = { can_manage: false, summary: { orders: 0, items: 0, pieces: 0 }, metrics: {}, items: [], total: 0, stats: { total: 0, unassigned: 0, today: 0, overdue: 0, blocked: 0, outsourced: 0 } };
const emptyPlan = { assignee: '', equipment: '', planned_start: '', planned_end: '', estimated_hours: '', blocked_reason: '', notes: '' };
async function request(url: string, body?: unknown) {
    const res = await authFetch(url, body ? { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) } : {});
    const value = await res.json();
    if (!res.ok || value.code !== 0)
        throw new Error(value.message || value.error || '操作失败');
    return value.data;
}

const views = [['all', '全部任务', 'total', Boxes, 'blue'], ['unassigned', '待安排', 'unassigned', UserRound, 'slate'], ['today', '今日应完工', 'today', CalendarDays, 'indigo'], ['overdue', '交期逾期', 'overdue', Clock3, 'rose'], ['blocked', '生产受阻', 'blocked', AlertTriangle, 'amber'], ['outsourced', '外协待回', 'outsourced', ArrowUpRight, 'violet']] as const;
export default function ProductionWorkbench(props: Props) {
    const [data, setData] = useState(initial), [view, setView] = useState<WorkbenchView>(props.initialState?.view || 'all'), [search, setSearch] = useState(props.initialState?.search || ''), [query, setQuery] = useState(props.initialState?.query || ''), [page, setPage] = useState(props.initialState?.page || 1), [size, setSize] = useState(props.initialState?.size || 50), [loading, setLoading] = useState(false), [notice, setNotice] = useState(''), [failure, setFailure] = useState(false), [selected, setSelected] = useState<number[]>([]), [editing, setEditing] = useState<number[]>([]), [form, setForm] = useState(emptyPlan), [saving, setSaving] = useState(false), [batchMode, setBatchMode] = useState(false), [density, setDensity] = useState<'comfortable' | 'compact'>(props.initialState?.density || 'compact');
    const [workflow,setWorkflow]=useState<{task:Task;initialStep?:WorkflowStep;initialRecordKind?:'scrap'|'exception'|'records';processId?:number;initialMode?:'timeline'|'route'}|null>(null);
    const openWorkflow=(task:Task,initialStep?:WorkflowStep,processId?:number,initialMode?:'timeline'|'route')=>setWorkflow({task,initialStep,processId,initialMode});
 const [chartMode, setChartMode] = useState(props.initialState?.chartMode || false);
    const version = useRef(0), inFlight = useRef(false);
    const load = async () => {
        const v = ++version.current;
        setLoading(true);
        try {
            const result = await request(`/api/platform/production/workbench?view=${view}&search=${encodeURIComponent(query)}&page=${page}&pageSize=${size}`);
            if (v === version.current) {
                if (failure) { setFailure(false); setNotice(''); }
                setData({ ...initial, ...result, metrics: result.metrics || {}, summary: result.summary || { orders: 0, items: result.total || 0, pieces: 0 } });
                setSelected([]);
            }
        }
        catch (e) {
            if (v === version.current) {
                setFailure(true);
                setNotice(e instanceof Error ? e.message : '读取失败');
                setData(initial);
            }
        }
        finally {
            if (v === version.current)
                setLoading(false);
        }
    };
    useEffect(() => { load(); return () => { version.current++; }; }, [view, query, page, size]);
    const act = async (action: () => Promise<unknown>, message: string) => {
        if (inFlight.current)
            return;
        inFlight.current = true;
        setSaving(true);
        setNotice('正在保存…');
        setFailure(false);
        try {
            await action();
            setNotice(message);
            await load();
            return true;
        }
        catch (e) {
            setFailure(true);
            setNotice(e instanceof Error ? e.message : '保存失败');
            return false;
        }
        finally {
            setSaving(false);
            inFlight.current = false;
        }
    };
    const arrange = (task?: Task) => { setBatchMode(!task); setNotice(''); setFailure(false); setEditing(task ? [task.item_id] : selected); setForm(task ? { assignee: task.assignee, equipment: task.equipment, planned_start: task.planned_start, planned_end: task.planned_end, estimated_hours: task.estimated_hours, blocked_reason: task.blocked_reason, notes: task.notes } : emptyPlan); };
    const savePlan = async (e: React.FormEvent) => {
        e.preventDefault();
        const fields = !batchMode ? { ...form, estimated_hours: form.estimated_hours || '0' } : Object.fromEntries(Object.entries(form).filter(([, v]) => v !== ''));
        if (!Object.keys(fields).length) {
            setFailure(true);
            setNotice('请填写至少一项批量安排内容');
            return;
        }
        if (await act(() => request('/api/platform/production/plans', { item_ids: editing, ...fields }), '生产安排已保存'))
            setEditing([]);
    };
    const showDrawing = async (task: Task) => {
        try {
            const d = await request(`/api/platform/production/items/${task.item_id}/drawing`);
            if (d.drawing_data)
                props.setShowDrawingModal(d.drawing_data);
            else {
                setFailure(true);
                setNotice('该零件暂无图纸');
            }
        }
        catch (e) {
            setFailure(true);
            setNotice(e instanceof Error ? e.message : '图纸读取失败');
        }
    };
    const openOrder = (task: Task) => {
        if (props.onOpenOrder) {
            props.onOpenOrder({ orderId: task.order_id, itemId: task.item_id, orderNumber: task.order_number, partName: task.part_name, state: { view, search, query, page, size, density, chartMode } });
            return;
        }
        const filters = { status: '', dueDateStart: '', dueDateEnd: '', orderNumber: task.order_number, partNumber: '', partName: '', customerName: '', priority: '' };
        props.setOrderFilters(filters); props.setAppliedOrderFilters(filters); props.setCurrentPage(1); props.setActiveTab(props.orderView || 'production_orders');
    };
    const summary = data.metrics.all || initial.summary;
    useEffect(() => {
        const esc = (e: KeyboardEvent) => {
            if (e.key === 'Escape' && !saving && !workflow) {
                setEditing([]);
            }
        };
        document.addEventListener('keydown', esc);
        return () => document.removeEventListener('keydown', esc);
    }, [saving, workflow]);
    const noticeBlock = notice && <div role={failure ? 'alert' : 'status'} className={`rounded-xl px-4 py-3 text-sm border ${failure ? 'bg-rose-50 border-rose-200 text-rose-700' : 'bg-blue-50 border-blue-200 text-blue-800'}`}>{notice}</div>;
    return <div className={`erp-page space-y-3 ${chartMode ? 'erp-workbench-gantt' : ''} ${density === 'compact' ? 'erp-workbench-compact' : ''}`} data-design-version="20261007-order-date-card">
  <header className="flex flex-wrap items-center justify-between gap-4"><div><p className="text-[11px] tracking-[.15em] text-blue-600 font-semibold mb-2">生产协同 / 工作台</p><h2 className="text-[26px] font-bold text-slate-900">生产工作台</h2><p className="text-sm text-slate-500 mt-2">{summary.orders} 个订单 · {summary.items} 项零件明细 · {summary.pieces.toLocaleString()} 件订单零件</p></div><div className="flex gap-2"><button onClick={load} disabled={loading || saving} aria-label="刷新工作台" className="erp-btn"><RefreshCw size={16} className={loading ? 'animate-spin' : ''}/>{loading ? '刷新中' : '刷新'}</button><button onClick={props.resetAndOpenModal} className="erp-btn erp-primary"><Plus size={16}/>新建订单</button></div></header>
  {noticeBlock}
  <div className="erp-workbench-stats grid grid-cols-2 lg:grid-cols-3 2xl:grid-cols-6 gap-3">{views.map(([key, label, count, Icon, tone]) => { const color = { blue: 'text-blue-600 bg-blue-50', slate: 'text-slate-600 bg-slate-100', indigo: 'text-indigo-600 bg-indigo-50', rose: 'text-rose-600 bg-rose-50', amber: 'text-amber-600 bg-amber-50', violet: 'text-violet-600 bg-violet-50' }[tone]; return <button key={key} onClick={() => { setView(key); setPage(1); }} aria-pressed={view === key} title={workbenchViewDescriptions[key]} className={`erp-card p-4 text-left transition-all hover:-translate-y-0.5 ${view === key ? 'ring-2 ring-blue-500 border-transparent' : 'hover:shadow-md'}`}><div className="flex items-center justify-between"><span className="text-sm text-slate-500">{label}</span><span className={`rounded-lg p-2 ${color}`}><Icon size={17}/></span></div><p className="text-[28px] font-bold text-slate-900 mt-2">{data.stats[count]}<span className="ml-1 text-xs font-normal text-slate-400">项</span></p><p className="text-xs text-slate-400 mt-1"><span>{data.metrics[key]?.orders || 0} 个订单</span><span className="erp-metric-separator"> · </span><span>{(data.metrics[key]?.pieces || 0).toLocaleString()} 件</span></p></button>; })}</div>
  <section className={`erp-card ${chartMode ? "erp-workbench-gantt-card" : "overflow-hidden"}`} aria-busy={loading}>
    <div className="erp-workbench-toolbar p-4 border-b border-slate-100">
      <div className="erp-workbench-toolbar-heading">
        <h3 className="font-semibold text-slate-800">{views.find(v => v[0] === view)?.[1]}<span className="ml-2 text-xs rounded-full bg-slate-100 text-slate-500 px-2 py-1">{loading ? '查询中' : data.total}</span></h3>
        <p className="text-[11px] text-slate-400 mt-1">{workbenchViewDescriptions[view]}</p>
      </div>
      <form onSubmit={e => { e.preventDefault(); if (search === query && page === 1) { void load(); } else { setLoading(true); setQuery(search); setPage(1); } }} className="erp-workbench-search" role="search" aria-label="生产任务查询">
        <div className="erp-workbench-search-field">
          <Search size={16} aria-hidden="true"/>
          <input type="search" value={search} onChange={e => setSearch(e.target.value)} placeholder="订单、客户、零件或图号" aria-label="搜索生产任务" className="erp-input"/>
        </div>
        <button type="submit" className="erp-btn">搜索</button>
        {query && <button type="button" className="erp-btn" onClick={() => { setLoading(true); setSearch(''); setQuery(''); setPage(1); }}>清除</button>}
        {!chartMode && <button type="button" aria-label="切换任务密度" className="erp-btn" onClick={() => setDensity(density === 'comfortable' ? 'compact' : 'comfortable')}><SlidersHorizontal size={15}/>{density === 'comfortable' ? '紧凑' : '舒适'}</button>}
      </form>
    </div>
  <div className="erp-workbench-view-switch p-3 border-b border-slate-100 flex flex-wrap items-center gap-2"><button className={`erp-btn ${!chartMode ? "erp-primary" : ""}`} aria-pressed={!chartMode} onClick={()=>setChartMode(false)}>任务列表</button><button className={`erp-btn ${chartMode ? "erp-primary" : ""}`} aria-pressed={chartMode} onClick={()=>{setChartMode(true);setSize(100);setPage(1);}}>工序甘特图</button>{chartMode&&<span className="text-xs text-slate-400">显示当前筛选本页零件的生产工序，翻页查看其余任务</span>}</div>
  {chartMode && !loading && <ProcessGanttChart stickyAxis parts={data.items.map(task=>({itemId:task.item_id,partName:task.part_name,partNumber:task.part_number,orderNumber:task.order_number,startDate:task.start_date,dueDate:task.due_date,quantity:task.quantity,delivery:task.delivery,batches:task.flow?.batches}))} onEditRoute={data.can_manage ? part=>{const task=data.items.find(t=>t.item_id===part.itemId);if(task)openWorkflow(task,'process',undefined,'route');} : undefined} onDeliverySelect={part=>{const task=data.items.find(t=>t.item_id===part.itemId);if(task)openWorkflow(task,'delivery');}} rows={data.items.flatMap(task=>task.processes.filter(p=>p.name!=="送货").map(process=>({itemId:task.item_id,partName:task.part_name,orderNumber:task.order_number,startDate:task.start_date,dueDate:task.due_date,process})))} onSelect={row=>{const task=data.items.find(t=>t.item_id===row.itemId);if(task)openWorkflow(task,'process',row.process.id);}}/>}
  {data.can_manage && !chartMode && !loading && <div className="px-5 py-3 bg-slate-50/70 flex items-center justify-between gap-3 text-xs"><label className="flex items-center gap-2 text-slate-500"><input type="checkbox" checked={data.items.length > 0 && selected.length === data.items.length} onChange={e => setSelected(e.target.checked ? data.items.map(t => t.item_id) : [])}/>选择本页<span className="text-slate-400">{selected.length ? `已选 ${selected.length} 项` : ''}</span></label><button disabled={!selected.length || saving} onClick={() => arrange()} className="erp-btn text-blue-600 min-h-8 py-1">批量安排</button></div>}
  {!loading && !data.items.length && <div className="py-20 text-center"><CheckCircle2 size={40} className="mx-auto mb-3 text-slate-300"/><p className="text-slate-600 font-medium">当前筛选下没有待办任务</p><p className="text-sm text-slate-400 mt-2">可切换分类或清除搜索条件</p></div>}
  {loading && <div role="status" className="py-12 px-4 text-center text-sm text-slate-500">正在读取生产任务{query ? `：${query}` : '…'}</div>}
  {!chartMode && !loading && <div className="divide-y divide-slate-100">{data.items.map(task => <WorkbenchTaskCard
    key={task.item_id} task={task} density={density} canManage={data.can_manage} saving={saving}
    selected={selected.includes(task.item_id)} today={businessToday()}
    onSelect={checked => setSelected(checked ? [...selected, task.item_id] : selected.filter(id => id !== task.item_id))}
    onOpenOrder={() => openOrder(task)} onDetails={() => openWorkflow(task)}
    onQuantities={() => openWorkflow(task)}
    onUnblock={() => { void act(() => request('/api/platform/production/plans', {item_ids:[task.item_id], blocked_reason:''}), '阻塞已解除'); }}
    onDrawing={() => { void showDrawing(task); }}
  />)}</div>}</section>
  <div className="erp-workbench-pagination flex flex-wrap justify-between items-center gap-3 text-xs text-slate-500"><span>{loading ? '正在更新当前筛选结果…' : `当前筛选：${data.summary.orders} 个订单 · ${data.total} 项明细 · ${data.summary.pieces.toLocaleString()} 件订单零件`}</span><div className="flex gap-2 items-center"><select aria-label="每页数量" value={size} onChange={e => { setSize(Number(e.target.value)); setPage(1); }} className="erp-input w-auto py-1">{[20, 50, 100].map(n => <option key={n} value={n}>每页 {n} 条</option>)}</select><button disabled={page === 1 || loading} onClick={() => setPage(page - 1)} className="erp-btn">上一页</button><span>{page} / {Math.max(1, Math.ceil(data.total / size))}</span><button disabled={page * size >= data.total || loading} onClick={() => setPage(page + 1)} className="erp-btn">下一页</button></div></div>
  {!!editing.length && <div className="erp-overlay"><form onSubmit={savePlan} className="erp-dialog p-6 space-y-5"><div className="flex justify-between items-start"><div><h3 className="text-xl font-bold">{batchMode ? `批量安排 ${editing.length} 项零件` : '安排生产'}</h3><p className="text-xs text-slate-400 mt-2">{batchMode ? '只更新填写的字段，留空保留原安排' : '安排负责人、设备和时间，随时调整计划'}</p></div><button aria-label="关闭生产安排" type="button" disabled={saving} onClick={() => setEditing([])}><X size={20}/></button></div>{failure && noticeBlock}<div className="grid grid-cols-1 sm:grid-cols-2 gap-4">{[['assignee', '负责人', 'text'], ['equipment', '设备 / 工位', 'text'], ['planned_start', '计划开始', 'date'], ['planned_end', '计划完工', 'date'], ['estimated_hours', '预计工时（小时）', 'number']].map(([key, label, type]) => <label key={key} className="erp-label">{label}<input type={type} min={type === 'number' ? 0 : undefined} step={type === 'number' ? '0.01' : undefined} maxLength={100} value={form[key as keyof typeof form]} onChange={e => setForm({ ...form, [key]: e.target.value })} className="erp-input"/></label>)}</div><label className="erp-label">受阻原因<textarea maxLength={500} value={form.blocked_reason} onChange={e => setForm({ ...form, blocked_reason: e.target.value })} placeholder="缺料、外协未回、等待图纸确认…" className="erp-input"/></label><label className="erp-label">生产备注<textarea maxLength={500} value={form.notes} onChange={e => setForm({ ...form, notes: e.target.value })} className="erp-input"/></label><div className="flex justify-end gap-2"><button type="button" disabled={saving} onClick={() => { setEditing([]); setNotice(''); setFailure(false); }} className="erp-btn">取消</button><button disabled={saving} className="erp-btn erp-primary">{saving ? '保存中…' : '保存安排'}</button></div></form></div>}
 {workflow && <PartWorkflowModal order={{id:workflow.task.order_id,order_number:workflow.task.order_number,customer_name:workflow.task.customer_name} as Order} itemId={workflow.task.item_id} initialStep={workflow.initialStep} initialRecordKind={workflow.initialRecordKind} processId={workflow.processId} initialMode={workflow.initialMode} onClose={()=>setWorkflow(null)} onChanged={load} onEditOrder={()=>{const task=workflow.task;setWorkflow(null);openOrder(task);}}/>}</div>;
}
