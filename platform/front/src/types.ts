export interface Contact {
    id?: number;
    corp_id?: number;
    name: string;
    contact: string;
}
export interface Customer {
    id?: number;
    corp_id?: number;
    name: string;
    short_name: string;
    contacts: Contact[];
    created_at?: string;
}
export interface OrderProcess {
    cutting_spec?: string | null;
    deleted_at?: string | null;
    id?: number;
    corp_id?: number;
    order_item_id?: number;
    name: string;
    is_outsourced: boolean;
    outsourcing_fee: number;
    status: 'pending' | 'processing' | 'completed';
    sort_order?: number;
    planned_start_at?: string | null;
    planned_end_at?: string | null;
    actual_start_at?: string | null;
    actual_end_at?: string | null;
}
export interface OrderItem {
 delivery_plan?: import("./types/productionFlow").DeliveryPlan[];
 delivery_batches?: import("./types/productionFlow").DeliveryBatch[];
 flow_tracking?: boolean;
 next_batch_due_date?: string;
 open_outsourcing?: number;
    id?: number;
    corp_id?: number;
    order_id?: number;
    part_name: string;
    part_number?: string;
    quantity: number;
    production_quantity?: number;
    qualified_quantity?: number;
    billable_quantity?: number;
    reserve_quantity?: number;
    delivery_gap?: number;
    scrap_quantity?: number;
    unit_price: number;
    total_price: number;
    status: 'pending' | 'processing' | 'completed' | 'delivered';
    drawing_count?: number;
    drawing_ids?: number[];
    drawing_drafts?: import("./features/drawings/api").PartDrawing[];
    drawing_change_reason?: string;
    has_drawing?: boolean;
    drawing_data?: string;
    notes?: string;
    processes?: OrderProcess[];
    completion_date?: string;
    start_date?: string;
    due_date?: string;
    delivered_quantity?: number;
  delivery_date?: string;
    tool_cost?: number;
    fixture_cost?: number;
    material_cost?: number;
    other_cost?: number;
    supply_mode?: 'self_purchased' | 'customer_supplied';
    currency?: string;
    tax_inclusive?: boolean;
    item_notes?: string;
}
export interface Order {
    id?: number;
    corp_id?: number;
    customer_id?: number;
    customer_name?: string;
    customer_short_name?: string;
    order_number?: string;
    order_name?: string;
    contact_id?: number;
    contact_name?: string;
    status: 'pending' | 'processing' | 'completed' | 'delivered';
    priority: 'low' | 'medium' | 'high';
    start_date?: string;
    due_date: string;
    total_amount?: number;
    notes?: string;
    created_at?: string;
    items?: OrderItem[];
}
export interface Material {
    id?: number;
    corp_id?: number;
    name: string;
    spec: string;
    quantity: number;
    unit: string;
}
export interface Remnant {
    id?: number;
    corp_id?: number;
    material_id?: number;
    material_name?: string;
    dimensions: string;
    photo_data?: string;
    notes?: string;
    created_at?: string;
}
export interface Reconciliation {
    month: string;
    total_amount: number;
    order_count: number;
    delivered_amount: number;
}
export interface MonthlyFinanceSummary {
    month: string;
    billable_quantity: string;
    reserve_quantity: string;
    qualified_quantity: string;
    completion_count: number;
    output_value: string;
    material_cost: string;
    outsourcing_cost: string;
    other_production_cost: string;
    production_cost: string;
    production_profit: string;
    period_expense: string;
    operating_profit: string;
    closing_status: 'open' | 'closed';
}
export interface AnnualFinanceSummary {
    year: string;
    output_value: string;
    production_cost: string;
    production_profit: string;
    period_expense: string;
    operating_profit: string;
}
export interface FinanceSummaryResponse {
    pending_cost_amount?: string;
    pending_cost_count?: number;
    year: string;
    months: MonthlyFinanceSummary[];
    annual: AnnualFinanceSummary;
}
export interface ProductionCompletion {
    id: number;
    order_id: number;
    order_item_id: number;
    completion_no: string;
    business_date: string;
    qualified_quantity: string;
    status: 'confirmed' | 'reversed';
    notes?: string;
    confirmed_at: string;
}
export interface CompletionListResponse {
    records: ProductionCompletion[];
    production_quantity: number;
    scrap_quantity: number;
    billable_quantity: number;
    billable_remaining: number;
    reserve_quantity: number;
    delivery_gap: number;
    ordered_quantity: number;
    confirmed_quantity: number;
    remaining_quantity: number;
}
export interface AdventRule {
    id?: number;
    corp_id?: number;
    name: string;
    description?: string;
    formula: string;
    target_status: 'pending' | 'processing' | 'completed';
    scopeType: 'general' | 'specific';
    ruleType: 'warning' | 'imminent';
}
export interface User {
    id?: number;
    corp_id?: number;
    role_id?: number;
    role_type?: 'admin' | 'user';
    role?: Role;
    username: string;
    name: string;
    email?: string;
    phone?: string;
    status: 'active' | 'inactive';
    expired_at?: string;
    created_at?: string;
    updated_at?: string;
}
export interface Role {
    id?: number;
    corp_id?: number;
    name: string;
    account_type: 'admin' | 'user';
    status?: string;
    created_at?: string;
    updated_at?: string;
}
export interface Resource {
    id?: number;
    resource_type: string;
    resource_key: string;
    name: string;
    parent_id?: number;
    path?: string;
    icon?: string;
    sort_order?: number;
    page_resources?: PageResource[];
    platform_type?: string;
    status?: string;
    created_at?: string;
}
export interface PageResource {
    key: string;
    name: string;
    type: 'button' | 'link' | 'other';
}
export interface PrintTemplate {
    id?: number;
    corp_id?: number;
    name: string;
    menu_route?: string;
    button_key?: string;
    button_name?: string;
    template?: string;
    excel_filename?: string;
    created_at?: string;
    updated_at?: string;
}
