import { FilterConfig } from '../components/OrderMonitorPanel';

// 通用筛选初始状态
export const createEmptyFilters = () => ({
  dueDateStart: '',
  dueDateEnd: '',
  orderNumber: '',
  partNumber: '',
  partName: '',
  customerName: '',
  priority: ''
});

// 订单管理筛选初始状态（包含状态筛选）
export const createOrderFilters = () => ({
  ...createEmptyFilters(),
  status: ''
});

// 订单管理页面的筛选配置
export const orderFilterConfigs: FilterConfig[] = [
  { key: 'dueDateStart', label: '订单交期(起)', type: 'date', width: 'w-32' },
  { key: 'dueDateEnd', label: '订单交期(止)', type: 'date', width: 'w-32' },
  { key: 'orderNumber', label: '订单号', type: 'text', placeholder: '搜索订单号...' },
  { key: 'partName', label: '零件名称', type: 'text', placeholder: '输入零件名称关键词' },
  { key: 'partNumber', label: '零件号', type: 'text', placeholder: '搜索零件号...' },
  { key: 'customerName', label: '客户名称', type: 'text', placeholder: '搜索客户...' },
  {
    key: 'priority',
    label: '优先级',
    type: 'select',
    width: 'w-28',
    placeholder: '全部',
    options: [
      { value: 'high', label: '高优先级' },
      { value: 'medium', label: '普通' },
      { value: 'low', label: '较低' }
    ]
  },
  {
    key: 'status',
    label: '订单状态',
    type: 'select',
    width: 'w-28',
    placeholder: '全部',
    options: [
      { value: 'pending', label: '待加工' },
      { value: 'processing', label: '加工中' },
      { value: 'completed', label: '已完成' }
    ]
  }
];

// 逾期/告警/临期订单页面的筛选配置（无订单状态筛选）
export const simpleFilterConfigs: FilterConfig[] = [
  { key: 'dueDateStart', label: '订单交期(起)', type: 'date', width: 'w-32' },
  { key: 'dueDateEnd', label: '订单交期(止)', type: 'date', width: 'w-32' },
  { key: 'orderNumber', label: '订单号', type: 'text', placeholder: '搜索订单号...' },
  { key: 'partName', label: '零件名称', type: 'text', placeholder: '输入零件名称关键词' },
  { key: 'partNumber', label: '零件号', type: 'text', placeholder: '搜索零件号...' },
  { key: 'customerName', label: '客户名称', type: 'text', placeholder: '搜索客户...' },
  {
    key: 'priority',
    label: '优先级',
    type: 'select',
    width: 'w-28',
    placeholder: '全部',
    options: [
      { value: 'high', label: '高优先级' },
      { value: 'medium', label: '普通' },
      { value: 'low', label: '较低' }
    ]
  }
];