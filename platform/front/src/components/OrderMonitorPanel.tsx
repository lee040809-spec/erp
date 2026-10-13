import DrawingButton from '../features/drawings/DrawingButton';
import PartWorkflowModal from "./PartWorkflowModal";
import { formatAmount, formatOrderAmount } from '../utils/money';
import React, { useEffect, useRef, useState } from 'react';
import { ChevronUp, ChevronDown, Settings, FileText, LucideIcon, Search, RefreshCw, Trash2, Plus, FileCheck, SlidersHorizontal } from 'lucide-react';
import { Order } from '../types';
import { formatDate, ProcessStatusBadge, StatusBadge, PriorityBadge } from './shared';
import Pagination from './Pagination';
import OrderBusinessModal from './OrderBusinessModal';
import type {WorkflowStep} from '../utils/partWorkflow';

export interface FilterConfig {
    key: string;
    label: string;
    type: 'text' | 'date' | 'select';
    placeholder?: string;
    options?: {
        value: string;
        label: string;
    }[];
    width?: string;
}
interface OrderMonitorPanelProps {
    title: string;
    icon: LucideIcon;
    orders: Order[];
    filters: any;
    setFilters: (filters: any) => void;
    filterConfigs?: FilterConfig[];
    localFilter?: (orders: Order[], filters: any, getOrderMaxDueDate: (order: Order) => string) => Order[];
    page: number;
    setPage: (page: number) => void;
    pageSize: number;
    setPageSize: (size: number) => void;
    total?: number;
    themeColor: string;
    editOrder: (order: Order) => void;
    deleteOrder?: (order: Order) => void;
    setShowDrawingModal: (data: string) => void;
    handleProcessClick: (orderId: number, itemId: number, processId: number, status: string, name: string) => void;
    getOrderMaxDueDate: (order: Order) => string;
    showOrderName?: boolean;
    showContactName?: boolean;
    showOrderNotes?: boolean;
    showOutsourcingFee?: boolean;
    showTotalAmount?: boolean;
    onSearch?: () => void;
    onReset?: (filters: any) => void;
    // 分页切换回调（带筛选条件的服务端分页）
    onPageChangeWithFilters?: (page: number, pageSize?: number) => void;
    isSearching?: boolean;
    queryError?: string;
    onRetryQuery?: () => void;
    onNewOrder?: () => void;
    // 送货管理模式
    deliveryMode?: boolean;
    selectedOrderId?: number | null;
    onSelectOrder?: (orderId: number | null) => void;
    onPreviewDelivery?: () => void;
    onConfigDelivery?: () => void;
    // 对账管理模式（支持多选）
    reconciliationMode?: boolean;
    selectedOrderIds?: Set<number>;
    onPreviewReconciliation?: () => void;
    onConfigReconciliation?: () => void;
    focusOrderId?: number;
    focusItemId?: number;
    sourcePartName?: string;
    onReturnToWorkbench?: () => void;
}
const OrderMonitorPanel: React.FC<OrderMonitorPanelProps> = (p) => {
    const { title, icon: Icon, orders, filters, setFilters, page, setPage, pageSize, setPageSize, total, themeColor, editOrder, deleteOrder, getOrderMaxDueDate, onSearch, onNewOrder, isSearching, onPageChangeWithFilters, deliveryMode, reconciliationMode, onSelectOrder } = p;
    const [expanded, setExpanded] = useState<Set<number>>(new Set()), [advanced, setAdvanced] = useState(false), [costs, setCosts] = useState(false);
    const [business, setBusiness] = useState<{
        order: Order;
        kind: 'records' | 'exception' | 'delivery' | 'scrap' | 'completion';
        item?: number;
    } | null>(null);
    const [workflow,setWorkflow]=useState<{order:Order;itemId:number;initialStep?:WorkflowStep;initialRecordKind?:'scrap';processId?:number}|null>(null);
    const focusedItemRef = useRef<HTMLDivElement | null>(null);
    const scrolledTarget = useRef('');
    useEffect(() => {
        if (!p.focusOrderId || !p.focusItemId) return;
        const targetOrder = orders.find(order => Number(order.id) === Number(p.focusOrderId));
        if (!targetOrder?.items?.some(item => Number(item.id) === Number(p.focusItemId))) return;
        const targetKey = `${p.focusOrderId}:${p.focusItemId}`;
        if (scrolledTarget.current === targetKey) return;
        if (!expanded.has(p.focusOrderId)) {
            setExpanded(old => new Set(old).add(p.focusOrderId!));
            return;
        }
        const frame = requestAnimationFrame(() => {
            if (focusedItemRef.current) {
                focusedItemRef.current.scrollIntoView({ block: 'center', behavior: 'smooth' });
                scrolledTarget.current = targetKey;
            }
        });
        return () => cancelAnimationFrame(frame);
    }, [orders, p.focusOrderId, p.focusItemId, expanded]);
    const configs = p.filterConfigs || [{ label: '订单号', key: 'orderNumber', type: 'text' as const, placeholder: '搜索订单号' }, { label: '客户名称', key: 'customerName', type: 'text' as const, placeholder: '搜索客户' }, { label: '零件名称', key: 'partName', type: 'text' as const, placeholder: '输入零件名称关键词' }, { label: '零件号', key: 'partNumber', type: 'text' as const, placeholder: '搜索图号' }, { label: '交期开始', key: 'dueDateStart', type: 'date' as const }, { label: '交期结束', key: 'dueDateEnd', type: 'date' as const }, { label: '优先级', key: 'priority', type: 'select' as const, options: [{ value: 'high', label: '紧急' }, { value: 'medium', label: '普通' }, { value: 'low', label: '较低' }] }];
    const safe = Array.isArray(orders) ? orders : [];
    const local = (o: Order) => { const text = (v: unknown) => String(v ?? '').toLowerCase(); return (!filters?.orderNumber || text(o.order_number ?? o.id).includes(text(filters.orderNumber))) && (!filters?.customerName || text(o.customer_name).includes(text(filters.customerName))) && ((!filters?.partNumber && !filters?.partName) || (o.items || []).some(i => (!filters?.partNumber || text(i.part_number).includes(text(filters.partNumber))) && (!filters?.partName || text(i.part_name).includes(text(filters.partName).trim())))) && (!filters?.priority || o.priority === filters.priority) && (!filters?.dueDateStart || getOrderMaxDueDate(o) >= filters.dueDateStart) && (!filters?.dueDateEnd || getOrderMaxDueDate(o) <= filters.dueDateEnd) && (!filters?.status || o.status === filters.status); };
    const filtered = total !== undefined ? safe : p.localFilter ? p.localFilter(safe, filters, getOrderMaxDueDate) : safe.filter(local);
    const visible = total !== undefined ? filtered : filtered.slice((page - 1) * pageSize, page * pageSize), count = total ?? filtered.length;
    const toggle = (id: number) => setExpanded(old => { const next = new Set(old); next.has(id) ? next.delete(id) : next.add(id); return next; });
    const allOpen = visible.length > 0 && visible.every(o => expanded.has(o.id || 0));
    const activeFilters = Object.entries(filters || {}).filter(([, v]) => Boolean(v)).length;
    const field = (f: FilterConfig) => <label key={f.key} className="erp-label flex-1 min-w-[140px]">{f.label}{f.type === 'select' ? <select className="erp-input" value={filters?.[f.key] || ''} onChange={e => { setFilters({ ...filters, [f.key]: e.target.value }); if (total === undefined) setPage(1); }}><option value="">{f.placeholder || '全部'}</option>{f.options?.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}</select> : <input className="erp-input" type={f.type} value={filters?.[f.key] || ''} placeholder={f.placeholder} onKeyDown={e => { if (e.key === 'Enter' && !isSearching) onSearch?.(); }} onChange={e => { setFilters({ ...filters, [f.key]: e.target.value }); if (total === undefined) setPage(1); }}/>}</label>;
    return <div className="erp-page erp-compact-orders flex flex-col gap-3" data-design-version="20261005-orders">
  <header className="flex items-center justify-between gap-3 flex-wrap"><div><p className="text-[11px] text-blue-600 tracking-widest font-semibold mb-2">订单协同 / {title}</p><h2 className="text-2xl font-bold flex items-center gap-2"><Icon size={24}/>{title}</h2><p className="text-sm text-slate-400 mt-2">共 {count} 个订单 · 本页 {visible.reduce((n, o) => n + (o.items?.length || 0), 0)} 项零件明细</p></div><div className="flex gap-2 flex-wrap"><button className="erp-btn" onClick={() => setExpanded(allOpen ? new Set() : new Set(visible.map(o => o.id || 0)))}>{allOpen ? <ChevronUp size={15}/> : <ChevronDown size={15}/>}全部{allOpen ? '收起' : '展开'}</button>{(p.showTotalAmount || p.showOutsourcingFee) && <button className="erp-btn" aria-pressed={costs} onClick={() => setCosts(!costs)}>{costs ? '收起费用' : '查看费用'}</button>}{onNewOrder && <button onClick={onNewOrder} className="erp-btn erp-primary"><Plus size={16}/>新建订单</button>}</div></header>
  {p.focusOrderId && p.onReturnToWorkbench && <div className="erp-card p-3 bg-blue-50 border-blue-100 flex flex-wrap items-center justify-between gap-3"><p className="text-xs text-blue-700">来自工作看板 · {p.sourcePartName || "对应零件"}<span className="block mt-1">对应订单已自动展开，来源零件以蓝色标出。</span></p><button onClick={p.onReturnToWorkbench} className="erp-btn text-blue-700">返回工作看板</button></div>}
  <section className="erp-order-filters erp-card p-4 space-y-3"><div className="flex flex-wrap items-end gap-3">{configs.filter(f => f.type === 'text').map(field)}<button className="erp-btn" aria-expanded={advanced} onClick={() => setAdvanced(!advanced)}><SlidersHorizontal size={15}/>更多筛选{activeFilters > 0 && <span className="text-blue-600">({activeFilters})</span>}</button>{onSearch && <button onClick={() => onSearch?.()} disabled={isSearching} className="erp-btn erp-primary">{isSearching ? <RefreshCw size={15} className="animate-spin"/> : <Search size={15}/>}查询</button>}<button className="erp-btn text-slate-500" onClick={() => { const empty = Object.fromEntries(Object.keys(filters || {}).map(k => [k, ''])); setFilters(empty); setPage(1); p.onReset?.(empty); }}>重置条件</button></div>{advanced && <div className="flex gap-3 flex-wrap border-t border-slate-100 pt-3">{configs.filter(f => f.type !== 'text').map(field)}</div>}</section>
  {(deliveryMode || reconciliationMode) && <section className="erp-card p-3 flex flex-wrap items-center gap-2"><span className="text-xs text-slate-500 mr-auto">{deliveryMode ? '选择一个订单生成送货单' : `已选 ${p.selectedOrderIds?.size || 0} 个订单`}</span>{deliveryMode && <><button id="btn-preview-delivery-note" className="erp-btn erp-success" disabled={!p.selectedOrderId} onClick={() => {
                    const order = safe.find(o => o.id === p.selectedOrderId);
                    if (order)
                        setBusiness({ order, kind: 'records' });
                }}><FileCheck size={15}/>已登记送货单</button>{p.onConfigDelivery && <button id="btn-config-delivery-note" className="erp-btn" disabled={!p.selectedOrderId} onClick={p.onConfigDelivery}><Settings size={15}/>配置送货单</button>}</>}{reconciliationMode && <><button id="btn-preview-reconciliation" className="erp-btn erp-primary" disabled={!p.selectedOrderIds?.size} onClick={p.onPreviewReconciliation}><FileCheck size={15}/>预览对账单</button>{p.onConfigReconciliation && <button id="btn-config-reconciliation" className="erp-btn" onClick={p.onConfigReconciliation}>配置对账单</button>}</>}</section>}
  {p.queryError && <div role="alert" className="erp-card border-amber-200 bg-amber-50 p-3 text-sm text-amber-800"><p>本次查询未完成：{p.queryError}</p><p className="mt-1">下方保留上次成功读取的结果，请重新读取后核对筛选条件。</p>{p.onRetryQuery && <button type="button" className="erp-btn mt-2" disabled={isSearching} onClick={p.onRetryQuery}>重新查询</button>}</div>}
  {isSearching && <p role="status" className="text-sm text-blue-600">正在读取订单…</p>}
  {!isSearching && !p.queryError && !visible.length && <div className="erp-card py-20 text-center text-slate-400">当前筛选下没有订单</div>}
  <div className="erp-order-list space-y-2">{visible.map(order => {
            const id = order.id || 0, open = expanded.has(id), items = order.items || [], ordered = items.reduce((n, i) => n + i.quantity, 0), production = items.reduce((n, i) => n + (i.production_quantity || i.quantity), 0);
            return <article key={id} className={`erp-card overflow-hidden ${Number(p.focusOrderId) === Number(id) ? "ring-2 ring-blue-200" : ""}`}>
   <div className="px-4 py-3 flex flex-wrap items-center justify-between gap-2"><div className="flex items-center gap-3">{(deliveryMode || reconciliationMode) && <input type="checkbox" aria-label={`选择订单${order.order_number || id}`} checked={deliveryMode ? p.selectedOrderId === id : p.selectedOrderIds?.has(id) || false} onChange={() => onSelectOrder?.(id)}/>}<button onClick={() => toggle(id)} aria-expanded={open} className="text-left flex items-center gap-3"><span className="rounded-xl bg-blue-50 text-blue-600 p-2"><FileText size={19}/></span><span><strong className="block text-sm text-slate-900">{order.order_number || id}<span className="font-normal text-slate-400 ml-3">{order.customer_short_name || order.customer_name}</span></strong><span className="block text-xs text-slate-400 mt-1">{items.length} 项零件 · 订单 {ordered.toLocaleString()} 件 · 投产 {production.toLocaleString()} 件{p.showOrderName && order.order_name && ` · ${order.order_name}`}</span></span></button></div><div className="flex items-center gap-3 flex-wrap"><PriorityBadge priority={order.priority}/><StatusBadge status={order.status}/>{p.showTotalAmount && <span className="text-sm font-semibold text-slate-800">{formatOrderAmount(order)}</span>}{!(deliveryMode || reconciliationMode) && <><button aria-label={`编辑订单${order.order_number || id}`} className="erp-btn py-1 min-h-8" onClick={() => editOrder(order)}>编辑</button>{deleteOrder && <button title="删除订单" aria-label={`删除订单${order.order_number || id}`} className="erp-btn text-rose-500 py-1 min-h-8" onClick={() => deleteOrder(order)}><Trash2 size={14}/></button>}</>}<button className="erp-btn py-1 min-h-8" onClick={() => setBusiness({ order, kind: 'records' })}>业务记录</button>{deliveryMode && <button className="erp-btn erp-primary py-1 min-h-8" onClick={() => setExpanded(old=>new Set(old).add(id))}>展开零件办理交货</button>}<button aria-label={open ? '收起零件' : '展开零件'} onClick={() => toggle(id)} className="erp-btn py-1 min-h-8">{open ? <ChevronUp size={16}/> : <ChevronDown size={16}/>}</button></div></div>
   <div className="px-4 pb-2 text-xs text-slate-400 flex gap-x-4 gap-y-1 flex-wrap"><span>订单日期 {formatDate(order.start_date) || '未填写'}</span><span>最晚交期 {formatDate(getOrderMaxDueDate(order)) || '未填写'}</span>{p.showContactName && <span>联系人 {order.contact_name || '未填写'}</span>}{p.showOrderNotes && order.notes && <span className="whitespace-pre-wrap">备注 {order.notes}</span>}</div>
   {open && <div className="border-t border-slate-100 divide-y divide-slate-100">{items.map((item, index) => <div key={item.id || index} ref={Number(id) === Number(p.focusOrderId) && Number(item.id) === Number(p.focusItemId) ? focusedItemRef : undefined} data-workbench-source={Number(id) === Number(p.focusOrderId) && Number(item.id) === Number(p.focusItemId) ? "true" : undefined} className="erp-order-item px-4 py-3 bg-slate-50/40"><div className="flex justify-between items-center flex-wrap gap-3"><div className="flex gap-3 items-center"><span className="text-xs text-slate-300 font-mono">{String(index + 1).padStart(2, '0')}</span><div><strong className="text-sm">{item.part_name}</strong>{Number(id) === Number(p.focusOrderId) && Number(item.id) === Number(p.focusItemId) && <span className="ml-2 text-[10px] rounded-md bg-blue-100 text-blue-700 px-2 py-1">看板来源</span>}<p className="text-xs text-slate-400 mt-1">{item.part_number || '无图号'}</p></div></div><div className="flex flex-wrap gap-2 items-center"><StatusBadge status={item.status}/>{item.id&&<button className="erp-btn erp-primary py-1 min-h-8" aria-label={`办理零件${item.part_name}`} onClick={()=>setWorkflow({order,itemId:item.id!})}>办理零件</button>}{(item.delivered_quantity || 0) > 0 && (item.delivered_quantity || 0) < item.quantity && <span className="text-xs text-amber-700 bg-amber-50 px-2 py-1 rounded-lg">部分交货</span>}{item.id && <DrawingButton itemId={item.id} partName={item.part_name} partNumber={item.part_number} customerId={order.customer_id} count={item.drawing_count || (item.has_drawing ? 1 : 0)}/>}</div></div>{!!item.delivery_batches?.length && <p className="text-xs text-blue-600 mt-1">{item.delivery_batches.map(b=>`${b.batch_no} ${b.quantity}件 · ${b.due_date.slice(0,10)}（已交${b.delivered_quantity}）`).join("；")}</p>}<div className="erp-part-metrics mt-2 text-xs"><div className="text-slate-400">订单 / 投产<p className="font-semibold text-sm text-slate-800 mt-1">{item.quantity} / {item.production_quantity || item.quantity}</p></div><div className="text-slate-400">合格 / 备用<p className="text-sm text-slate-700 mt-1"><span className="text-emerald-600">{item.qualified_quantity || 0}</span> / <span className="text-violet-600">{item.reserve_quantity || 0}</span></p></div><div className="text-slate-400">报废 / 已交货<p className="text-sm text-slate-700 mt-1"><span className={item.scrap_quantity ? 'text-rose-600' : ''}>{item.scrap_quantity || 0}</span> / {item.delivered_quantity || 0}</p></div><div className="text-slate-400">客户交期<p className="text-sm text-slate-700 mt-1">{item.due_date ? formatDate(item.due_date) : '未填写'}</p></div><div className="text-slate-400">订单数量完成日期<p className="text-sm text-slate-700 mt-1">{item.completion_date ? formatDate(item.completion_date) : '待报工'}</p></div><div className="text-slate-400">最近交货日期<p className="text-sm text-slate-700 mt-1">{item.delivery_date ? formatDate(item.delivery_date) : '暂无可核实日期'}</p></div>{p.showTotalAmount && <div className="text-slate-400">单价 / 总价<p className="text-sm text-slate-700 mt-1">{formatAmount(item.unit_price)} / {formatAmount(item.quantity * item.unit_price)}</p></div>}</div><div className="flex gap-1.5 flex-wrap items-center mt-2"><span className="text-[11px] text-slate-400 mr-1">生产工序</span>{!(item.processes || []).length && <span className="text-xs text-slate-400">未设置</span>}{(item.processes || []).map((process, idx) => <button key={process.id || idx} onClick={() => item.id && setWorkflow({order,itemId:item.id,initialStep:process.name==='送货'?'delivery':'process',processId:process.name==='送货'?undefined:process.id})} className={`erp-step erp-step-${process.status}`} title={`${process.name}：${process.status === 'completed' ? '已完成' : process.status === 'processing' ? '加工中' : '待加工'}`}><span>{process.name}</span><ProcessStatusBadge status={process.status}/>{process.is_outsourced && <span className="text-[10px]">外协</span>}</button>)}</div>{costs && (p.showTotalAmount || p.showOutsourcingFee) && <div className="mt-4 rounded-xl bg-white border border-slate-200 p-3 flex flex-wrap gap-x-6 gap-y-2 text-xs text-slate-500">{p.showOutsourcingFee && <span>外协 {formatAmount((item.processes || []).reduce((n, i) => n + Number(i.outsourcing_fee || 0), 0))}</span>}{p.showTotalAmount && <><span>刀具 {formatAmount(item.tool_cost)}</span><span>工装 {formatAmount(item.fixture_cost)}</span><span>材料 {formatAmount(item.material_cost)}</span><span>其它 {formatAmount(item.other_cost)}</span><span>{item.supply_mode === 'customer_supplied' ? '客户来料' : '自购材料'} · {item.tax_inclusive ? '含税' : '不含税'}</span></>}</div>}{(item.notes || item.item_notes) && <p className="mt-3 text-xs text-slate-500 whitespace-pre-wrap">{[item.notes, item.item_notes].filter(Boolean).join(' · ')}</p>}</div>)}</div>}
  </article>;
        })}</div>
  <Pagination total={count} page={page} pageSize={pageSize} onPageChange={n => { setPage(n); onPageChangeWithFilters?.(n); }} onPageSizeChange={n => { setPageSize(n); setPage(1); onPageChangeWithFilters?.(1, n); }} activeColor={themeColor}/>
 {workflow && <PartWorkflowModal {...workflow} onClose={()=>setWorkflow(null)} onEditOrder={()=>{setWorkflow(null);editOrder(workflow.order);}} onChanged={()=>{if(onPageChangeWithFilters)onPageChangeWithFilters(page,pageSize);else onSearch?.();}}/>}
 {business && <OrderBusinessModal order={business.order} initialKind={business.kind} initialItem={business.item} onOpenPart={(itemId,kind)=>{setWorkflow({order:business.order,itemId,initialStep:kind==='scrap'?'records':kind==='delivery'?'delivery':'completion',initialRecordKind:kind==='scrap'?'scrap':undefined});setBusiness(null);}} onEditOrder={() => { setBusiness(null); editOrder(business.order); }} onClose={() => setBusiness(null)} onChanged={() => { if (onPageChangeWithFilters) onPageChangeWithFilters(page, pageSize); else onSearch?.(); }}/>}
 </div>;
};
export default OrderMonitorPanel;
