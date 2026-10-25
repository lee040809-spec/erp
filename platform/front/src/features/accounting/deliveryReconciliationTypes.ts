export interface Delivery {
  id: number;
  customer_id: number;
  customer_name: string;
  order_number: string;
  part_name: string;
  part_number: string;
  business_date: string;
  quantity: number;
  unit_price: string;
  amount: string;
  price_ready: number;
  reconciliation_id: number | null;
}
export interface Statement {
  id: number;
  statement_no: string;
  customer_name?: string;
  business_date: string;
  amount: string;
  status: string;
  notes: string;
  reversal_reason?: string;
}
export interface Detail {
  header: Statement;
  lines: Delivery[];
}
