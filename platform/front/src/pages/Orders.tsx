import React from 'react';
import { useOrderQuery } from '../hooks/useOrderQuery';
import { ClipboardList } from 'lucide-react';
import { Order } from '../types';
import OrderMonitorPanel from '../components/OrderMonitorPanel';
import DeleteConfirmModal from '../components/DeleteConfirmModal';
import { OrderFilters, filterOrdersLocal } from '../services/orderService';
import { orderFilterConfigs } from '../configs/filterConfigs';
import { WorkbenchOrderTarget } from '../services/workbenchNavigation';

interface OrdersProps {
  orders: Order[];
  setOrders: (orders: Order[]) => void;
  orderFilters: OrderFilters;
  setOrderFilters: (filters: OrderFilters) => void;
  appliedOrderFilters?: OrderFilters;
  setAppliedOrderFilters?: (filters: OrderFilters) => void;
  showOrderFilters?: boolean;
  setShowOrderFilters?: (show: boolean) => void;
  currentPage: number;
  setCurrentPage: (page: number) => void;
  pageSize: number;
  setPageSize: (size: number) => void;
  orderTotal: number;
  resetAndOpenModal: () => void;
  editOrder: (order: Order) => void;
  deleteOrder: (orderId: number) => void;
  setShowDrawingModal: (data: string) => void;
  handleProcessClick: (orderId: number, itemId: number, processId: number, status: string, name: string) => void;
  getOrderMaxDueDate: (order: Order) => string;
  fetchData: () => void;
  fetchOrdersWithFilters: (filters: OrderFilters, page?: number, pageSize?: number) => Promise<void>;
  hideCostFields?: boolean;
  navigationTarget?: WorkbenchOrderTarget | null;
  onReturnToWorkbench?: () => void;
}

const Orders: React.FC<OrdersProps> = ({ orders, orderFilters, setOrderFilters, appliedOrderFilters, setAppliedOrderFilters, currentPage, setCurrentPage, pageSize, setPageSize, orderTotal, resetAndOpenModal, editOrder, deleteOrder, setShowDrawingModal, handleProcessClick, getOrderMaxDueDate, fetchOrdersWithFilters, hideCostFields = false, navigationTarget, onReturnToWorkbench }) => {
  const { isSearching, queryError, run, retry } = useOrderQuery();
  const [showDeleteConfirm, setShowDeleteConfirm] = React.useState(false);
  const [deletingOrder, setDeletingOrder] = React.useState<Order | null>(null);

  // Handle search with API call
  const handleSearch = async (nextFilters = orderFilters) => {
    setCurrentPage(1);
    setAppliedOrderFilters?.(nextFilters);
    await run(() => fetchOrdersWithFilters(nextFilters, 1, pageSize));
  };

  // 分页切换时带筛选条件查询
  const handlePageChangeWithFilters = async (newPage: number, newPageSize?: number) => {
    await run(() => fetchOrdersWithFilters(appliedOrderFilters || orderFilters, newPage, newPageSize || pageSize));
  };

  // Handle delete click
  const handleDeleteClick = (order: Order) => {
    setDeletingOrder(order);
    setShowDeleteConfirm(true);
  };

  return (
    <div className="flex-1 !w-full flex flex-col min-h-0">
      {/* Shared Panel */}
      <OrderMonitorPanel
        title="订单管理"
        queryError={queryError}
        onRetryQuery={retry}
        icon={ClipboardList}
        orders={orders}
        filters={orderFilters}
        setFilters={setOrderFilters}
        filterConfigs={orderFilterConfigs}
        localFilter={(orders, filters, getMaxDueDate) => filterOrdersLocal(orders, filters as OrderFilters, getMaxDueDate)}
        page={currentPage}
        setPage={setCurrentPage}
        pageSize={pageSize}
        setPageSize={setPageSize}
        total={orderTotal}
        themeColor="blue"
        editOrder={editOrder}
        deleteOrder={handleDeleteClick}
        setShowDrawingModal={setShowDrawingModal}
        handleProcessClick={handleProcessClick}
        getOrderMaxDueDate={getOrderMaxDueDate}
        showOrderName={true}
        showContactName={true}
        showOrderNotes={true}
        showOutsourcingFee={!hideCostFields}
        showTotalAmount={!hideCostFields}
        onSearch={() => handleSearch()}
        onReset={handleSearch}
        onPageChangeWithFilters={handlePageChangeWithFilters}
        isSearching={isSearching}
        onNewOrder={resetAndOpenModal}
        focusOrderId={navigationTarget?.orderId}
        focusItemId={navigationTarget?.itemId}
        sourcePartName={navigationTarget?.partName}
        onReturnToWorkbench={onReturnToWorkbench}
      />

      {/* Delete Confirm Modal */}
      <DeleteConfirmModal
        show={showDeleteConfirm}
        title="确认删除订单"
        message={`确定要删除订单 ${deletingOrder?.order_number || deletingOrder?.id} 吗？此操作不可撤销。`}
        onClose={() => {
          setShowDeleteConfirm(false);
          setDeletingOrder(null);
        }}
        onConfirm={() => {
          if (deletingOrder && deletingOrder.id) {
            deleteOrder(deletingOrder.id);
          }
          setShowDeleteConfirm(false);
          setDeletingOrder(null);
        }}
      />
    </div>
  );
};

export default Orders;
