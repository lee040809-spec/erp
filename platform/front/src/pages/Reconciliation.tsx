import React from 'react';
import { FileText } from 'lucide-react';
import { Order } from '../types';
import OrderMonitorPanel from '../components/OrderMonitorPanel';
import ReconciliationPreviewModal from '../components/ReconciliationPreviewModal';
import { simpleFilterConfigs } from '../configs/filterConfigs';
import { authFetch } from '../components/shared';

interface PrintTemplate {
  id: number;
  name: string;
  template: string;
  excel_filename?: string;
}

interface ReconciliationProps {
  orders: Order[];
  orderTotal: number;
  reconciliationFilters: any;
  setReconciliationFilters: (filters: any) => void;
  reconciliationPage: number;
  setReconciliationPage: (page: number) => void;
  reconciliationPageSize: number;
  setReconciliationPageSize: (size: number) => void;
  editOrder: (order: Order) => void;
  setShowDrawingModal: (data: string) => void;
  handleProcessClick: (orderId: number, itemId: number, processId: number, status: string, name: string) => void;
  getOrderMaxDueDate: (order: Order) => string;
  fetchData: () => void;
  fetchOrdersWithFilters: (filters: any, page?: number, pageSize?: number, dateType?: string) => Promise<void>;
  hideCostFields?: boolean;
  showToast?: (message: string, type: 'success' | 'error') => void;
}

const Reconciliation: React.FC<ReconciliationProps> = ({
  orders,
  orderTotal,
  reconciliationFilters,
  setReconciliationFilters,
  reconciliationPage,
  setReconciliationPage,
  reconciliationPageSize,
  setReconciliationPageSize,
  editOrder,
  setShowDrawingModal,
  handleProcessClick,
  getOrderMaxDueDate,
  fetchOrdersWithFilters,
  hideCostFields = false,
  showToast
}) => {
  const [isSearching, setIsSearching] = React.useState(false);
  // 多选订单ID
  const [selectedOrderIds, setSelectedOrderIds] = React.useState<Set<number>>(new Set());
  const [showPreviewModal, setShowPreviewModal] = React.useState(false);
  const [previewOrders, setPreviewOrders] = React.useState<Order[]>([]);
  const [previewTemplate, setPreviewTemplate] = React.useState<PrintTemplate | null>(null);
  const [previewOrderData, setPreviewOrderData] = React.useState<any>(null);
  const [showConfigModal, setShowConfigModal] = React.useState(false);
  const [configTemplate, setConfigTemplate] = React.useState<PrintTemplate | null>(null);

  const applied=React.useRef(reconciliationFilters);
  const handlePageChange = async (nextPage:number,nextSize=reconciliationPageSize,nextFilters=applied.current) => {
    setIsSearching(true);
  setSelectedOrderIds(new Set());
    try {
      const filtersWithStatus = { ...nextFilters, status: 'completed' };
      await fetchOrdersWithFilters(filtersWithStatus, nextPage, nextSize);
    } finally {
      setIsSearching(false);
    }
  };

  const handleSearch=(nextFilters=reconciliationFilters)=>{applied.current=nextFilters;setReconciliationPage(1);return handlePageChange(1,reconciliationPageSize,nextFilters);};
  // 多选订单处理
  const handleSelectOrder = (orderId: number | null) => {
    if (orderId === null) return;
    setSelectedOrderIds(prev => {
      const newSet = new Set(prev);
      if (newSet.has(orderId)) {
        newSet.delete(orderId);
      } else {
        newSet.add(orderId);
      }
      return newSet;
    });
  };

  // 获取选中的订单列表
  const getSelectedOrders = () => {
    return orders.filter(o => selectedOrderIds.has(o.id || 0));
  };

  const handlePreviewReconciliation = async () => {
    if (selectedOrderIds.size === 0) {
      showToast?.('请选择订单', 'error');
      return;
    }

    const selectedOrders = getSelectedOrders();
    const orderIdsParam = Array.from(selectedOrderIds).join(',');

    try {
      const res = await authFetch(`/api/platform/print-templates/by-button?menu_route=reconciliation&button_key=btn-config-reconciliation&mode=preview&order_ids=${orderIdsParam}`);
      const result = await res.json();

      if (result.data) {
        setPreviewOrders(selectedOrders);
        setPreviewTemplate(result.data);
        setPreviewOrderData(result.order || null);
        setShowPreviewModal(true);
      } else {
        showToast?.(result.error || '未找到绑定的对账单模板', 'error');
      }
    } catch (err) {
      showToast?.('查询模板失败', 'error');
    }
  };

  const handleConfigReconciliation = async () => {
    try {
      const res = await authFetch(`/api/platform/print-templates/by-button?menu_route=reconciliation&button_key=btn-config-reconciliation&mode=config`);
      const result = await res.json();

      if (result.data) {
        setConfigTemplate(result.data);
        setShowConfigModal(true);
      } else {
        showToast?.(result.error || '未找到绑定的对账单配置模板', 'error');
      }
    } catch (err) {
      showToast?.('查询模板失败', 'error');
    }
  };

  const completedOrders = orders;

  return (
    <>
      <OrderMonitorPanel
        title="对账管理"
        icon={FileText}
        orders={completedOrders}
        filters={reconciliationFilters}
        setFilters={setReconciliationFilters}
        filterConfigs={simpleFilterConfigs}
        total={orderTotal}
      page={reconciliationPage}
        setPage={setReconciliationPage}
        pageSize={reconciliationPageSize}
        setPageSize={setReconciliationPageSize}

        themeColor="blue"
        editOrder={editOrder}
        setShowDrawingModal={setShowDrawingModal}
        handleProcessClick={handleProcessClick}
        getOrderMaxDueDate={getOrderMaxDueDate}
        showOrderName={true}
        showContactName={true}
        showOutsourcingFee={!hideCostFields}
        showTotalAmount={!hideCostFields}
        onSearch={()=>handleSearch()}
      onReset={handleSearch}
      onPageChangeWithFilters={handlePageChange}
        isSearching={isSearching}
        reconciliationMode={true}
        selectedOrderIds={selectedOrderIds}
        onSelectOrder={handleSelectOrder}
        onPreviewReconciliation={handlePreviewReconciliation}
        onConfigReconciliation={handleConfigReconciliation}
      />

      {/* 对账单预览弹窗 */}
      {showPreviewModal && previewOrders.length > 0 && previewTemplate && (
        <ReconciliationPreviewModal
          orders={previewOrders}
          template={previewTemplate}
          mode="preview"
          title="预览对账单"
          orderData={previewOrderData}
          onClose={() => setShowPreviewModal(false)}
        />
      )}

      {/* 对账单配置弹窗 - 不传订单 */}
      {showConfigModal && configTemplate && (
        <ReconciliationPreviewModal
          template={configTemplate}
          mode="config"
          title="配置对账单"
          onClose={() => setShowConfigModal(false)}
        />
      )}
    </>
  );
};

export default Reconciliation;