import { LayoutDashboard, ClipboardList, AlertCircle, AlertTriangle, Clock, Users, Package, CircleDollarSign, Settings2, LucideIcon, FileText } from 'lucide-react';

export interface NavItem {
  id: string;
  label: string;
  icon: LucideIcon;
}

export const NAV_ITEMS: NavItem[] = [
  { id: 'dashboard', label: '工作看板', icon: LayoutDashboard },
  { id: 'orders', label: '订单管理', icon: ClipboardList },
  { id: 'overdue', label: '逾期订单', icon: AlertCircle },
  { id: 'warning_orders', label: '告警订单', icon: AlertTriangle },
  { id: 'imminent_orders', label: '临期订单', icon: Clock },
  { id: 'customers', label: '客户管理', icon: Users },
  { id: 'outsourcing_suppliers', label: '外协供应商', icon: Users },
  { id: 'reconciliation', label: '对账管理', icon: FileText },
  { id: 'inventory', label: '仓库余料', icon: Package },
  { id: 'finance', label: '财务对账', icon: CircleDollarSign },
  { id: 'advent_rules', label: '规则管理', icon: Settings2 },
];

// Supplier maintenance belongs to the existing production/order capability, without altering roles.
export function withOperationalMenus(items: NavItem[]): NavItem[] {
 if (items.some(i => i.id === 'outsourcing_suppliers') || !items.some(i => ['orders','production_orders','production_dashboard','dashboard','production_delivery','monthly_output'].includes(i.id))) return items;
 return [...items, { id:'outsourcing_suppliers',label:'外协供应商',icon:Users }];
}
