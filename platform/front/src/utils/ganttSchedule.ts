import type { OrderProcess } from '../types';
import { processDate } from './processDates';

export const GANTT_DAY = 86400000;
export const ganttDay = (value?: string | null) => processDate(value) ? Date.parse(`${processDate(value)}T00:00:00+08:00`) : NaN;
export const beijingDayStart = (value: number) => Math.floor((value + 8 * 3600000) / GANTT_DAY) * GANTT_DAY - 8 * 3600000;

export function processScheduleRisk(process: OrderProcess, now = Date.now()) {
  const actualEnd = ganttDay(process.actual_end_at), plannedEnd = ganttDay(process.planned_end_at);
  if (Number.isFinite(actualEnd)) {
    if (actualEnd > plannedEnd) return `完工晚于计划 ${Math.round((actualEnd - plannedEnd) / GANTT_DAY)} 天`;
    return '';
  }
  // Historical completion status alone does not establish an actual completion date.
  if (process.status === 'completed') return '实际日期待补录';
  if (beijingDayStart(now) > plannedEnd) return `工序超期 ${Math.round((beijingDayStart(now) - plannedEnd) / GANTT_DAY)} 天`;
  return '';
}

export interface GanttDelivery {
  last_date: string;
  delivered_quantity: number;
  dated_quantity: number;
  undated_quantity: number;
  batches: number;
}

export function lastProcessDeliveryRisk(processes: OrderProcess[], dueDate?: string | null) {
  const due = ganttDay(dueDate), last = processes.filter(p => p.name !== '送货').at(-1);
  if (!last || !Number.isFinite(due)) return '';
  const plannedEnd = ganttDay(last.planned_end_at);
  return plannedEnd > due ? `末道计划晚于客户交期 ${Math.round((plannedEnd - due) / GANTT_DAY)} 天` : '';
}

export function timelineExtent(rows: { process?: OrderProcess; dueDate?: string | null; delivery?: GanttDelivery }[], now = Date.now()) {
  const dates = rows.flatMap(({ process: p, dueDate, delivery }) => [
    ganttDay(p?.planned_start_at), ganttDay(p?.planned_end_at), ganttDay(p?.actual_start_at), ganttDay(p?.actual_end_at), ganttDay(dueDate),
    ganttDay(delivery?.last_date),
    p?.actual_start_at && !p.actual_end_at ? beijingDayStart(now) : NaN,
  ]).filter(Number.isFinite);
  if (!dates.length) return { start: beijingDayStart(now), days: 14 };
  const start = Math.min(...dates) - GANTT_DAY;
  return { start, days: Math.max(7, Math.round((Math.max(...dates) - start) / GANTT_DAY) + 2) };
}
