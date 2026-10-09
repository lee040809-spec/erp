export interface OutsourcingSupplier { id: number; name: string; contact_name: string; phone: string; address: string; capabilities: string; notes: string; active: boolean; }
export interface SupplierPage { items: OutsourcingSupplier[]; page: number; has_more: boolean; can_manage: boolean; }
export interface OutsourcingOrderLine { id: number; order_process_id: number; process_name: string; quantity: number; dispatched_quantity: number; started_quantity: number; unit_price?: string; }
export interface OutsourcingOrder { id: number; edit_token: string; order_no: string; scope: 'full'|'selected'; supplier_id: number; supplier_name: string; order_date: string; expected_return_date: string; unit_price?: string; delivery_batch_id?: number; status: string; notes: string; lines: OutsourcingOrderLine[]; }
export interface OutsourcingOrderPage { items: OutsourcingOrder[]; page: number; has_more: boolean; }
