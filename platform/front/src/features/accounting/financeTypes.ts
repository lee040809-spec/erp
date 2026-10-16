export interface Candidate {
  order_item_id: number;
  order_id: number;
  order_number: string;
  customer_name: string;
  part_name: string;
  quantity: number;
  production_quantity: number;
  scrap_quantity: number;
  business_date: string | null;
  confirmed_quantity: number;
  remaining_quantity: number;
  unit_price: string;
  supply_mode: string;
  currency: string;
  tax_inclusive: number;
}
export interface Batch {
  id: number;
  completion_no: string;
  order_number: string;
  customer_name: string;
  part_name: string;
  business_date: string;
  qualified_quantity: string;
  billable_quantity: string;
  reserve_quantity: string;
  unit_price_snapshot: string;
  output_value: string;
  notes: string;
}
export interface Cost {
  id: number;
  cost_type: string;
  business_date: string;
  amount: string;
  tax_amount: string;
  net_amount: string;
  order_id?: number;
  order_item_id?: number;
  supplier_name?: string;
  notes?: string;
}
export interface PendingCost extends Cost {
  remaining_amount: string;
}
export interface AllocationRecord {
  id: number;
  cost_entry_id: number;
  cost_type: string;
  cost_date: string;
  completion_no: string;
  allocated_amount: string;
  allocation_basis: string;
  created_by: number;
}
export interface HistoryCompletionForm {
  quantity: string;
  date: string;
  notes: string;
  sourceRef: string;
  price: string;
  reviewed: boolean;
}
