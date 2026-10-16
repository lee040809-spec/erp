import type { DashboardItem } from '../services/api';

export interface ProductionTask extends DashboardItem {
  order_costs?: import('../types').OrderCostTotals;
 flow?: import("./productionFlow").FlowSummary;
  customer_name: string;
  assignee: string;
  equipment: string;
  planned_start: string;
  planned_end: string;
  estimated_hours: string;
  blocked_reason: string;
  notes: string;
  qualified_quantity: number;
  remaining_quantity: number;
  scrap_quantity: number;
  production_quantity: number;
  reserve_quantity: number;
  delivery_gap: number;
  billable_quantity: number;
  has_drawing: boolean;
  drawing_count?: number;
}
