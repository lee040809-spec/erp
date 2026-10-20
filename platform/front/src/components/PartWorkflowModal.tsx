import DrawingButton from '../features/drawings/DrawingButton';
import PartRecordCorrections from '../features/production-records/PartRecordCorrections';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { X, ArrowRight, RefreshCw } from 'lucide-react';
import type { Order } from '../types';
import type { FlowData } from '../types/productionFlow';
import type { DashboardItem } from '../services/api';
import { flowDate, flowRequest } from '../services/productionFlow';
import { nextPartAction, withBusinessQuantities, WorkflowStep } from '../utils/partWorkflow';
import ProcessTimelineModal from './ProcessTimelineModal';
import ProcessQuantityPanel from './ProcessQuantityPanel';
import BatchDeliveryPanel from './BatchDeliveryPanel';
import OutsourcingPanel from './OutsourcingPanel';
import CompletionModal from './CompletionModal';
import OrderBusinessModal, { BusinessData, Kind } from './OrderBusinessModal';
import PartPlanningPanel from './PartPlanningPanel';
import './part-workflow.css';
const tabs: [
    WorkflowStep,
    string
][] = [['process', '1 安排与加工'], ['completion', '2 确认合格'], ['delivery', '3 登记交货'], ['batches', '分批计划'], ['outsourcing', '外协'], ['records', '报废 / 异常 / 记录']];
interface Props {
    order: Order;
    itemId: number;
    initialStep?: WorkflowStep;
    initialRecordKind?: Kind;
    processId?: number;
    initialMode?: 'timeline' | 'route';
    onClose: () => void;
    onChanged: () => void | Promise<void>;
    onEditOrder?: () => void;
}
export default function PartWorkflowModal(p: Props) {
    const [data, setData] = useState<FlowData | null>(null), [business, setBusiness] = useState<BusinessData | null>(null), [step, setStep] = useState<WorkflowStep>(p.initialStep || 'process'), [visited, setVisited] = useState<Set<WorkflowStep>>(new Set(p.initialStep ? [p.initialStep] : [])), [error, setError] = useState(''), [loading, setLoading] = useState(true), [revision, setRevision] = useState(0), [notice, setNotice] = useState(''), [timelineRevision, setTimelineRevision] = useState(0), [busyMap, setBusyMap] = useState<Record<string, boolean>>({});
    const firstLoad = useRef(true), dirty = useRef(new Set<WorkflowStep>()), generation = useRef(0), scroll = useRef<HTMLDivElement>(null), busy = Object.values(busyMap).some(Boolean);
    const busyHandlers = useMemo(() => Object.fromEntries(['timeline', 'quantity', 'completion', 'delivery', 'records', 'batches', 'outsourcing', 'planning', 'completionCorrection', 'deliveryCorrection'].map(k => [k, (v: boolean) => setBusyMap(old => old[k] === v ? old : { ...old, [k]: v })])), []);
    const navigate = useCallback((next: WorkflowStep) => { setStep(next); setVisited(old => new Set(old).add(next)); scroll.current?.scrollTo({ top: 0 }); }, []);
    const load = useCallback(async () => { const token = ++generation.current; try {
        const [flow, records] = await Promise.all([flowRequest<FlowData>('/production/items/' + p.itemId + '/flow'), flowRequest<BusinessData>('/orders/' + p.order.id + '/business-records?page=1&pageSize=50')]);
        if (token === generation.current) {
            const totals = records.items.find(i => Number(i.id) === p.itemId), current = withBusinessQuantities(flow, totals);
            setData(current);
            setBusiness(records);
            setError('');
            if (firstLoad.current) {
                firstLoad.current = false;
                if (!p.initialStep)
                    navigate(nextPartAction(current, totals?.price_ready === undefined ? undefined : !!Number(totals.price_ready)).step);
            }
        }
    }
    catch (e) {
        if (token === generation.current)
            setError((e as Error).message);
        throw e;
    }
    finally {
        if (token === generation.current)
            setLoading(false);
    } }, [p.itemId, p.order.id, p.initialStep, navigate]);
    useEffect(() => { load().catch(() => { }); return () => { generation.current++; }; }, [load]);
    const changed = async (origin: WorkflowStep, timelineLoaded = false) => { await Promise.all([load(), p.onChanged()]); dirty.current.delete(origin); if (origin === 'outsourcing' || origin === 'process' && !timelineLoaded)
        setTimelineRevision(v => v + 1); setRevision(v => v + 1); setNotice('已保存，数量、工序状态和可交货数量已更新。'); };
    const close = () => { if (busy)
        return; if (dirty.current.size && !window.confirm('还有未保存的填写内容，确定关闭零件办理窗口吗？'))
        return; p.onClose(); };
    const item = data?.flow.item, businessItem = business?.items.find(i => Number(i.id) === p.itemId), next = data ? nextPartAction(data, businessItem?.price_ready === undefined ? undefined : !!Number(businessItem.price_ready)) : null;
    const completeReady = data && (data.flow.tracking ? Number(data.flow.available_completion) > 0 : data.flow.processes.filter(q => q.name !== '送货').length > 0 && data.flow.processes.filter(q => q.name !== '送货').every(q => q.status === 'completed'));
    const panel = (key: WorkflowStep, content: React.ReactNode) => visited.has(key) && <section key={key} hidden={step !== key} className="erp-workflow-pane" onChangeCapture={() => dirty.current.add(key)}>{content}</section>;
    return <div data-escape-dialog data-escape-blocked={busy} className="erp-overlay z-[110] erp-workflow-overlay"><section role="dialog" aria-modal="true" aria-label="零件办理" className="erp-dialog erp-workflow-dialog"><header className="erp-workflow-header"><div><p className="text-xs text-slate-500">零件办理 · {p.order.order_number} · {p.order.customer_short_name || p.order.customer_name}</p><h3 className="text-xl font-bold mt-1">{item?.part_name || '读取零件…'}<span className="text-sm font-normal text-slate-400 ml-3">{item?.part_number || ''}</span></h3></div><div className="erp-workflow-header-actions"><DrawingButton itemId={p.itemId} partName={item?.part_name || "当前零件"} partNumber={item?.part_number} customerId={p.order.customer_id} disabled={busy}/><button data-escape-close type="button" aria-label="关闭零件办理" className="erp-btn" disabled={busy} onClick={close}><X size={18}/></button></div></header>
 <div className="erp-workflow-scroll" ref={scroll}>{loading && !data && <p role="status" className="p-5">正在读取零件生产与交货记录…</p>}{error && <div role="alert" className="m-4 bg-rose-50 p-3 rounded-xl text-rose-700">{error}<button className="erp-btn ml-3" disabled={busy} onClick={() => load().catch(() => { })}><RefreshCw size={15}/>重试</button></div>}
 {item && data && <><div className="erp-workflow-summary">{[['订单', Number(item.quantity)], ['投产', Number(item.production_quantity || item.quantity)], ['合格', Number(item.qualified_quantity || 0)], ['报废', Number(item.scrap_quantity || 0)], ['已交货', Number(item.delivered_quantity || 0)]].map(([label, value]) => <span key={label}><small>{label}</small><strong>{value}</strong></span>)}<span><small>客户交期</small><strong>{flowDate(item.due_date) || '未填写'}</strong></span></div>
 {next && <div className="erp-workflow-next"><div><strong>{next.title}</strong><p className="text-xs mt-1 leading-5 text-blue-700">{next.detail}</p></div><button type="button" disabled={busy} className="erp-btn erp-primary" onClick={() => navigate(next.step)}>办理下一步<ArrowRight size={15}/></button></div>}
 <nav aria-label="零件操作流程" className="erp-workflow-tabs">{tabs.map(([key, label]) => <button key={key} type="button" aria-current={step === key ? 'step' : undefined} disabled={busy} className={`erp-btn ${step === key ? 'erp-primary' : ''}`} onClick={() => navigate(key)}>{label}</button>)}</nav>
 {notice && <p role="status" className="px-5 py-2 text-sm text-emerald-700">{notice}</p>}
 {panel('process', <><h4 className="font-semibold px-5 pt-4">选择工序，安排日期或登记实际加工</h4>{data.flow.tracking && <div className="p-5"><ProcessQuantityPanel data={data} initialProcess={p.processId} onBusyChange={busyHandlers.quantity} onChanged={() => changed('process')} onOutsourcing={() => navigate('outsourcing')}/></div>}<ProcessTimelineModal embedded itemId={p.itemId} partName={item.part_name} orderNumber={p.order.order_number} dueDate={item.due_date} processId={p.processId} initialMode={p.initialMode} quantityTracking={data.flow.tracking} refreshKey={timelineRevision} onBusyChange={busyHandlers.timeline} onClose={close} onChanged={() => changed('process', true)}/><details className="erp-card m-5 p-4"><summary className="cursor-pointer font-semibold text-sm">负责人 / 设备 / 投产数量 / 受阻原因</summary><div className="mt-4"><PartPlanningPanel data={data} onBusyChange={busyHandlers.planning} onChanged={() => changed('process')}/></div></details>{!data.flow.tracking && <details className="erp-card m-5 p-4"><summary className="cursor-pointer font-semibold text-sm">需要按工序数量报工或分批生产？核对历史数量后启用</summary><div className="mt-4"><ProcessQuantityPanel data={data} onBusyChange={busyHandlers.quantity} onChanged={() => changed('process')} onOutsourcing={() => navigate('outsourcing')}/></div></details>}</>)}
 {panel('completion', <div className="p-5 space-y-3"><PartRecordCorrections itemId={p.itemId} kind="completion" refreshKey={revision} onBusyChange={busyHandlers.completionCorrection} onChanged={() => changed('completion')}/><h4 className="font-semibold">填写本次实际合格数量和完工日期</h4>{!data.can_complete ? <p className="text-amber-700">当前账号无合格确认权限，请由有生产报工或产值统计权限的人员办理。</p> : !completeReady ? <div className="bg-amber-50 p-4 rounded-xl text-sm">{data.flow.tracking ? '末道工序尚无可确认的完工数量，请先登记末道本次完工数量。' : '生产工序尚未全部完成，请先登记或更正实际加工记录。'}<button type="button" className="erp-btn ml-2" onClick={() => navigate('process')}>办理加工</button></div> : businessItem?.price_ready !== undefined && !Number(businessItem.price_ready) ? <div className="bg-amber-50 p-4 rounded-xl text-sm">尚未录入有效单价，请补录真实单价后再确认合格。{p.onEditOrder && <button type="button" className="erp-btn ml-2" disabled={busy} onClick={() => { if (!dirty.current.size || window.confirm('补录单价需要打开订单资料，请先确认本窗口的填写内容已保存。继续吗？'))
            p.onEditOrder?.(); }}>编辑订单单价</button>}</div> : data.flow.tracking ? <BatchDeliveryPanel data={data} mode="completion" onBusyChange={busyHandlers.completion} onChanged={() => changed('completion')}/> : <CompletionModal embedded item={{ item_id: item.id, part_name: item.part_name, part_number: item.part_number, quantity: item.quantity } as DashboardItem} refreshKey={revision} onBusyChange={busyHandlers.completion} onClose={close} onSuccess={() => changed('completion')}/>}<button type="button" className="erp-btn text-blue-600" disabled={busy} onClick={() => navigate('delivery')}>继续登记交货<ArrowRight size={15}/></button></div>)}
 {panel('delivery', <><div className="p-5 pb-0"><PartRecordCorrections itemId={p.itemId} kind="delivery" refreshKey={revision} onBusyChange={busyHandlers.deliveryCorrection} onChanged={() => changed('delivery')}/></div><OrderBusinessModal embedded order={p.order} initialItem={p.itemId} initialKind="delivery" refreshKey={revision} onBusyChange={busyHandlers.delivery} onNavigate={navigate} onClose={close} onChanged={() => changed('delivery')} onEditOrder={p.onEditOrder}/></>)}
 {panel('batches', <div className="p-5 space-y-3"><h4 className="font-semibold">设置各批应交数量和要求日期</h4><p className="text-sm text-slate-500">这里只安排批次，实际完工和交货分别在步骤 2、3 登记，各批共用工序进度。</p><BatchDeliveryPanel data={data} mode="plan" onBusyChange={busyHandlers.batches} onChanged={() => changed('batches')}/></div>)}
 {panel('outsourcing', <div className="p-5"><p className="text-sm text-slate-500 mb-4">下单 → 发出 → 回厂 / 不合格处理 → 对账入账 → 费用分摊。可以在前道未开工时下单。</p>{!data.flow.tracking && <button type="button" className="erp-btn mb-3" onClick={() => navigate('process')}>核对工序历史数量并启用实物流转跟踪</button>}<OutsourcingPanel data={data} onBusyChange={busyHandlers.outsourcing} onChanged={() => changed('outsourcing')}/></div>)}
 {panel('records', <OrderBusinessModal embedded showActions order={p.order} initialItem={p.itemId} initialKind={p.initialRecordKind || 'records'} refreshKey={revision} onBusyChange={busyHandlers.records} onNavigate={navigate} onClose={close} onChanged={() => changed('records')}/>)}
 </>}
 </div><footer className="erp-workflow-footer"><p className="text-xs text-slate-500">{busy ? '正在保存，请稍候…' : '切换页签会保留未提交内容；实际数量和日期以保存的记录为准。'}</p><button data-escape-close type="button" className="erp-btn" disabled={busy} onClick={close}>关闭办理窗口</button></footer></section></div>;
}
