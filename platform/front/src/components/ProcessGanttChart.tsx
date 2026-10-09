import React, { useEffect, useId, useMemo, useRef, useState } from 'react';
import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react';
import type { OrderProcess } from '../types';
import { processDate } from '../utils/processDates';
import { GANTT_DAY as DAY, ganttDay, beijingDayStart as dayStart, processScheduleRisk, lastProcessDeliveryRisk, timelineExtent, GanttDelivery } from '../utils/ganttSchedule';

export interface GanttPart { batches?: import("../types/productionFlow").DeliveryBatch[]; batchLabel?: string; itemId: number; partName: string; partNumber?: string; orderNumber?: string; startDate?: string | null; dueDate?: string | null; quantity?: number; delivery?: GanttDelivery; }
export interface ProcessGanttRow extends GanttPart { process: OrderProcess; }
interface PartGroup extends GanttPart { rows: ProcessGanttRow[]; }
export const nodeTime = ganttDay;
export const displayNode = (value?: string | null) => processDate(value) || '未记录';
const dateLabel = (value: number) => new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', month: '2-digit', day: '2-digit' }).format(new Date(value));
const shortNode = (value?: string | null) => processDate(value) || '未排期';
const statusName = (status: OrderProcess['status']) => status === 'completed' ? '已完成' : status === 'processing' ? '加工中' : '待加工';

export default function ProcessGanttChart({ rows, parts: partMetadata = [], onSelect, onDeliverySelect, onEditRoute }: { rows: ProcessGanttRow[]; parts?: GanttPart[]; onSelect?: (row: ProcessGanttRow) => void; onDeliverySelect?: (part: GanttPart) => void; onEditRoute?: (part: GanttPart) => void }) {
  const chartId = useId(), chartRef = useRef<HTMLElement>(null);
  const [days, setDays] = useState(14), [offset, setOffset] = useState<number | null>(null);
  // Undefined is a fixed day range; null fits this page; an item ID fits that part.
  const [overview, setOverview] = useState<number | null | undefined>(undefined), [viewportWidth, setViewportWidth] = useState(760);
  const [labelWidth, setLabelWidth] = useState(180);
  const [collapsed, setCollapsed] = useState<Set<number>>(() => new Set());
  useEffect(() => {
    const element = chartRef.current;
    if (!element) return;
    const observer = new ResizeObserver(() => {
      setViewportWidth(element.clientWidth);
      setLabelWidth(parseFloat(getComputedStyle(element).getPropertyValue('--erp-gantt-label-width')) || 180);
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, []);
  const groups = useMemo(() => {
    // Group by the actual item ID, never by names shared by different orders/items.
    const parts = new Map<number, PartGroup>();
    // Explicit part metadata keeps a shipping node even when production steps are not set.
    partMetadata.forEach(part => parts.set(part.itemId, { ...part, rows: [] }));
    rows.forEach(row => {
      let group = parts.get(row.itemId);
      if (!group) {
        group = { ...row, rows: [] };
        parts.set(row.itemId, group);
      }
      group.rows.push(row);
    });
    return [...parts.values()];
  }, [rows, partMetadata]);
  const initial = useMemo(() => {
    const nodes = rows.flatMap(r => [nodeTime(r.process.planned_start_at), nodeTime(r.process.actual_start_at), nodeTime(r.process.actual_end_at)]).filter(Number.isFinite);
    return dayStart(nodes.length ? Math.min(...nodes) : Date.now());
  }, [rows]);
  const extent = useMemo(() => timelineExtent([...rows, ...groups, ...groups.flatMap(g=>(g.batches || []).map(b=>({...g,dueDate:b.due_date})))].filter(r => overview == null || r.itemId === overview)), [rows, groups, overview]);
  const fullView = overview !== undefined;
  const start = fullView ? extent.start : offset ?? initial, visibleDays = fullView ? extent.days : days;
  const end = start + visibleDays * DAY, now = dayStart(Date.now());
  const dayWidth = fullView ? Math.max(3, Math.min(64, (viewportWidth - labelWidth - 18) / visibleDays)) : 64;
  const width = visibleDays * dayWidth, tickStep = Math.max(1, Math.ceil(48 / dayWidth));
  const intersects = (from: number, until: number) => Number.isFinite(from) && Number.isFinite(until) && from < end && until > start;
  const togglePart = (itemId: number) => setCollapsed(previous => {
    const next = new Set(previous);
    if (next.has(itemId)) next.delete(itemId); else next.add(itemId);
    return next;
  });
  const viewPart = (group: PartGroup) => {
    setOverview(group.itemId);
    setCollapsed(previous => { const next = new Set(previous); next.delete(group.itemId); return next; });
  };
  const bar = (from: number, until: number, actual: boolean, row: ProcessGanttRow) => {
    if (!intersects(from, until)) return null;
    const left = Math.max(0, (from - start) / (visibleDays * DAY) * 100), right = Math.min(100, (until - start) / (visibleDays * DAY) * 100);
    const p = row.process, continued = until > end, before = from < start;
    const title = `${actual ? '实际' : '计划'}：${displayNode(new Date(from).toISOString())} → ${displayNode(new Date(until - DAY).toISOString())}${actual && !p.actual_end_at ? '（进行中）' : ''}`;
    const endText = actual && !p.actual_end_at ? '进行中' : `${actual ? '实' : continued ? '续至' : '至'} ${displayNode(new Date(until - DAY).toISOString())}`;
    const expectedEnds = [actual ? nodeTime(p.planned_end_at) + DAY : NaN].filter(Number.isFinite);
    const limit = expectedEnds.length ? Math.min(...expectedEnds) : Infinity;
    const late = until > limit || !actual && processScheduleRisk(p).startsWith('工序超期');
    const clippedStart = Math.max(from, start), clippedEnd = Math.min(until, end);
    const lateWidth = late ? Math.max(0, (clippedEnd - Math.max(limit, clippedStart)) / (clippedEnd - clippedStart) * 100) : 0;
    return <span title={`${title}${late ? '；存在超期风险，详见行内提示' : ''}`} className={`erp-gantt-bar ${actual ? p.actual_end_at ? 'erp-gantt-completed' : 'erp-gantt-running' : 'erp-gantt-planned'} ${late ? 'is-late' : ''} ${continued ? 'is-continued' : ''}`} style={{ left: `${left}%`, width: `${Math.max(0.15, right - left)}%`, top: actual ? 25 : 5 }}>
      {lateWidth > 0 && <span className="erp-gantt-overdue-segment" style={{ width: `${lateWidth}%` }} />}
      {before && <span className="erp-gantt-bar-before">‹</span>}
      <span className="erp-gantt-bar-date">{endText}{continued ? ' ›' : ''}</span>
    </span>;
  };
  const renderProcess = (row: ProcessGanttRow, index: number) => {
    const p = row.process, actualStart = nodeTime(p.actual_start_at), actualEnd = nodeTime(p.actual_end_at);
    const plannedStart = nodeTime(p.planned_start_at), plannedEnd = nodeTime(p.planned_end_at) + DAY, due = nodeTime(row.dueDate);
    const hasNodes = [actualStart, actualEnd, plannedStart].some(Number.isFinite);
    const milestoneVisible = !Number.isFinite(actualStart) && actualEnd >= start && actualEnd < end;
    const hasVisible = intersects(plannedStart, plannedEnd) || intersects(actualStart, (Number.isFinite(actualEnd) ? actualEnd : now) + DAY) || milestoneVisible;
    const risk = processScheduleRisk(p);
    const dates = `计划：${displayNode(p.planned_start_at)} → ${displayNode(p.planned_end_at)}；实际：${displayNode(p.actual_start_at)} → ${displayNode(p.actual_end_at)}${risk ? `；${risk}` : ''}`;
    return <div key={`${row.itemId}-${p.id ?? index}`} className={`erp-gantt-row ${risk && risk !== '实际日期待补录' ? 'has-risk' : ''}`}>
      <button type="button" title={dates} className="erp-gantt-label erp-gantt-process-label" onClick={() => onSelect?.(row)} aria-label={`工序时间${row.partName}的${p.name}`}>
        <span className="erp-gantt-process-index">{String(index + 1).padStart(2, '0')}</span>
        <span className="erp-gantt-process-info"><span className="erp-gantt-process-title"><strong>{p.name}</strong>{p.is_outsourced && <span className="text-violet-600">外</span>}<span className={`erp-gantt-state erp-gantt-state-${p.status}`}>{statusName(p.status)}</span></span><span className="erp-gantt-process-dates">{p.planned_start_at || p.planned_end_at ? `${shortNode(p.planned_start_at)} → ${shortNode(p.planned_end_at)}` : '计划未排期'}</span></span>
      </button>
      <button type="button" title={dates} onClick={() => onSelect?.(row)} className="erp-gantt-track" style={{ width, backgroundSize: `${dayWidth * tickStep}px 100%` }} aria-label={`${row.partName}的${p.name}时间条`}>
        {Number.isFinite(due) && due + DAY >= start && due + DAY < end && <span title={`客户交期：${displayNode(row.dueDate)}（当天结束）`} className="erp-gantt-due-line" style={{ left: `${(due + DAY - start) / (visibleDays * DAY) * 100}%` }} />}
        {bar(plannedStart, plannedEnd, false, row)}
        {bar(actualStart, (Number.isFinite(actualEnd) ? actualEnd : now) + DAY, true, row)}
        {milestoneVisible && <span title={`实际完工 ${displayNode(p.actual_end_at)}，开工时间未记录`} className={`erp-gantt-milestone ${risk && risk !== '实际日期待补录' ? 'is-late' : ''}`} style={{ left: `${(actualEnd - start + DAY / 2) / (visibleDays * DAY) * 100}%` }} />}
        {!hasNodes && <span className="erp-gantt-empty-node">未排期{p.status !== 'pending' ? ' · 实际待补录' : ''}</span>}
        {hasNodes && !hasVisible && <span className="erp-gantt-empty-node">日期在当前时段外 · 查看全程</span>}
        {risk && <span className={`erp-gantt-risk ${risk === '实际日期待补录' ? 'is-unknown' : ''}`}>{risk}</span>}
        {now >= start && now < end && <span className="erp-gantt-today" style={{ left: `${(now - start) / (visibleDays * DAY) * 100}%` }} />}
      </button>
    </div>;
  };
  const renderDelivery = (group: PartGroup) => {
    const due = nodeTime(group.dueDate), delivery = group.delivery, actual = nodeTime(delivery?.last_date);
    const delivered = delivery?.delivered_quantity || 0, required = group.quantity || 0;
    const status = delivered <= 0 ? '待送货' : required > 0 && delivered >= required ? '已交足' : '部分交货';
    const lastProcess = group.rows.at(-1)?.process;
    const planRisk = group.batchLabel ? "" : lastProcessDeliveryRisk(group.rows.map(r => r.process), group.dueDate);
    const deliveryLate = actual > due ? `最近送货晚于交期 ${Math.round((actual - due) / DAY)} 天` : '';
    const duePassed = required > delivered && now > due ? `客户交期已过 ${Math.round((now - due) / DAY)} 天` : '';
    const risk = delivered >= required && required > 0 ? deliveryLate : deliveryLate || planRisk || duePassed;
    const actualText = processDate(delivery?.last_date) ? `最近实际送货 ${processDate(delivery?.last_date)}` : delivered > 0 ? '历史送货日期待补录' : '暂无实际送货记录';
    const note = `${actualText} · 已交 ${delivered}${required > 0 ? ` / ${required}` : ''} 件${delivery?.undated_quantity ? ` · ${delivery.undated_quantity} 件日期未知` : ''}`;
    const title = `客户要求送货：${displayNode(group.dueDate)}；末道计划完工：${displayNode(lastProcess?.planned_end_at)}；${note}${risk ? `；${risk}（客户交期 ${displayNode(group.dueDate)}）` : ''}。实际日期来源于有效交货记录，不取送货工序状态。`;
    const marker = (date: number, isActual: boolean) => date >= start && date < end && <span title={`${isActual ? '最近实际送货' : '客户要求送货'}：${displayNode(new Date(date).toISOString())}`} className={`erp-gantt-delivery-marker ${isActual ? actual > due ? 'is-late' : 'is-actual' : 'is-required'}`} style={{ left: `${(date - start + DAY / 2) / (visibleDays * DAY) * 100}%`, top: isActual ? 27 : 5 }} />;
    return <div className={`erp-gantt-row erp-gantt-delivery-row ${risk ? 'has-delivery-risk' : ''}`}>
      <button type="button" disabled={!onDeliverySelect} title={title} className="erp-gantt-label erp-gantt-process-label" aria-label={`查看送货记录${group.partName} · ${group.orderNumber || ''}`} onClick={() => onDeliverySelect?.(group)}>
        <span className="erp-gantt-process-index">◇</span><span className="erp-gantt-process-info"><span className="erp-gantt-process-title"><strong>{group.batchLabel || "送货"}</strong><span className={`erp-gantt-state ${status === '已交足' ? 'erp-gantt-state-completed' : 'erp-gantt-state-pending'}`}>{status}</span></span><span className="erp-gantt-process-dates">要求 {displayNode(group.dueDate)}</span><span className="erp-gantt-process-dates">已交 {delivered}{required ? ` / ${required}` : ''} 件</span></span>
      </button>
      <button type="button" disabled={!onDeliverySelect} title={title} className="erp-gantt-track erp-gantt-delivery-track" style={{ width, backgroundSize: `${dayWidth * tickStep}px 100%` }} aria-label={`${group.partName}的送货时间节点`} onClick={() => onDeliverySelect?.(group)}>
        {marker(due, false)}{marker(actual, true)}
        <span className="erp-gantt-delivery-note">{note}</span>
        <span className={`erp-gantt-delivery-plan ${risk ? 'text-rose-700' : ''}`}>{group.batchLabel ? '本批可交数量由末道工序分批完工确认' : planRisk || (lastProcess?.planned_end_at ? `末道计划完工 ${displayNode(lastProcess.planned_end_at)}` : '末道计划完工日期未设置')}{deliveryLate ? ` · ${deliveryLate}` : !planRisk && duePassed ? ` · ${duePassed}` : ''}</span>
      </button>
    </div>;
  };
  return <section ref={chartRef} className="erp-gantt" aria-label="工序甘特图">
    <header className="erp-gantt-toolbar">
      <div className="erp-gantt-legends"><span className="erp-gantt-legend"><i className="bg-blue-400" />计划</span><span className="erp-gantt-legend"><i className="bg-amber-400" />加工中</span><span className="erp-gantt-legend"><i className="bg-emerald-500" />实际</span><span className="erp-gantt-legend"><i className="erp-gantt-due-legend" />交期</span><span className="erp-gantt-legend"><i className="erp-gantt-late-legend" />超期风险</span></div>
      <div className="erp-gantt-controls">
        {groups.length > 1 && <><button type="button" className="erp-btn" disabled={groups.every(group => !collapsed.has(group.itemId))} onClick={() => setCollapsed(new Set())}>全部展开</button><button type="button" className="erp-btn" disabled={groups.every(group => collapsed.has(group.itemId))} onClick={() => setCollapsed(new Set(groups.map(group => group.itemId)))}>全部折叠</button></>}
        <select className="erp-input" aria-label="甘特图时间范围" value={fullView ? 'full' : days} onChange={e => { if (e.target.value === 'full') setOverview(null); else { setOffset(start); setDays(Number(e.target.value)); setOverview(undefined); } }}><option value="7">7天</option><option value="14">14天</option><option value="30">30天</option><option value="60">60天</option><option value="full">本页全程</option></select>
        <button type="button" className="erp-btn" aria-label="甘特图上一时段" onClick={() => { setOverview(undefined); setOffset(start - days * DAY); }}><ChevronLeft size={14} /></button><button type="button" className="erp-btn" onClick={() => { setOverview(undefined); setOffset(dayStart(Date.now())); }}>今天</button><button type="button" className="erp-btn" aria-label="甘特图下一时段" onClick={() => { setOverview(undefined); setOffset(start + days * DAY); }}><ChevronRight size={14} /></button>
      </div>
      <p className="erp-gantt-range-note">{fullView ? overview == null ? '本页全程' : `${groups.find(g => g.itemId === overview)?.partName || '零件'}全程` : '当前时段'} · {displayNode(new Date(start).toISOString())} — {displayNode(new Date(end - DAY).toISOString())}{fullView ? ' · 可选 7/14 天细看' : ' · 箭头表示工序仍延续至所标日期'}</p>
    </header>
    <div className="erp-gantt-scroll"><div style={{ minWidth: `calc(${width}px + var(--erp-gantt-label-width))` }}>
      <div className="erp-gantt-axis"><div className="erp-gantt-label font-semibold">零件 / 工序 · 按天</div><div className="flex" style={{ width }}>{Array.from({ length: Math.ceil(visibleDays / tickStep) }, (_, i) => <div key={i} style={{ width: Math.min(tickStep, visibleDays - i * tickStep) * dayWidth, flexShrink: 0 }} className="erp-gantt-day-tick">{dateLabel(start + i * tickStep * DAY)}</div>)}</div></div>
      {groups.map(group => {
        const expanded = !collapsed.has(group.itemId), contentId = `${chartId}-part-${group.itemId}`;
        const completed = group.rows.filter(row => row.process.status === 'completed').length;
        const missing = group.rows.filter(row => row.process.status !== 'completed' && (!row.process.planned_start_at || !row.process.planned_end_at)).length;
        return <div key={group.itemId} className="erp-gantt-group">
          <div className="erp-gantt-group-heading">
            <button type="button" className="erp-gantt-group-toggle" style={onEditRoute ? {width:Math.max(140,Math.min(720,viewportWidth - 170))} : undefined} aria-expanded={expanded} aria-controls={contentId}
              aria-label={`零件${group.partName}${group.orderNumber ? ` · ${group.orderNumber}` : ''}的工序`} onClick={() => togglePart(group.itemId)}>
              <ChevronDown size={15} className={`erp-gantt-group-chevron ${expanded ? '' : 'is-collapsed'}`} />
              <span className="min-w-0">
                <strong className="erp-gantt-part-name">{group.partName}</strong>
                <span className="erp-gantt-part-summary">
                  {group.orderNumber && <span className="erp-gantt-order-number">{group.orderNumber} · </span>}工序 {completed}/{group.rows.length}
                  {missing > 0 && <span className="ml-2 text-amber-700">{missing} 道未排期</span>}
                </span>
                <span className="erp-gantt-part-due">
                  <span className="text-slate-500 whitespace-nowrap" title="零件下单日期；未填写时使用所属订单日期">下单 {processDate(group.startDate) || '未填'}</span>
                  <span className="mx-1">·</span>
                  <span className="whitespace-nowrap">{processDate(group.dueDate) ? `交期 ${processDate(group.dueDate)}` : '交期未设置'}</span>
                </span>
              </span>
            </button>
            <div className="erp-gantt-group-actions">{onEditRoute && <button type="button" className="erp-gantt-view-part" aria-label={`增删工序${group.partName} · ${group.orderNumber || ''}`} onClick={()=>onEditRoute(group)}>增删工序</button>}<button type="button" className="erp-gantt-view-part" aria-label={`查看${group.partName} · ${group.orderNumber || ''}的全程`} onClick={() => viewPart(group)}>查看全程</button></div>
          </div>
          <div id={contentId} hidden={!expanded}>{expanded && <>{group.rows.map(renderProcess)}{group.batches?.length ? group.batches.map(b=><React.Fragment key={b.id}>{renderDelivery({...group,batchLabel:`送货 · ${b.batch_no}`,dueDate:b.due_date,quantity:b.quantity,delivery:{last_date:b.last_delivery_date || "",delivered_quantity:b.delivered_quantity,dated_quantity:b.delivered_quantity,undated_quantity:0,batches:1}})}</React.Fragment>) : renderDelivery(group)}</>}</div>
        </div>;
      })}
    </div></div>
    {!groups.length && <p className="p-6 text-sm text-slate-500">暂无零件。</p>}
  </section>;
}
