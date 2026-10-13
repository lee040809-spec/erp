import React from 'react';
import { useOrderQuery } from '../hooks/useOrderQuery';
import { AlertTriangle } from 'lucide-react';
import { Order } from '../types';
import OrderMonitorPanel from '../components/OrderMonitorPanel';
import { simpleFilterConfigs } from '../configs/filterConfigs';

interface WarningProps {
  orders: Order[];
  orderTotal: number;
  warningFilters: any;
  setWarningFilters: (filters: any) => void;
  warningPage: number;
  setWarningPage: (page: number) => void;
  warningPageSize: number;
  setWarningPageSize: (size: number) => void;
  editOrder: (order: Order) => void;
  setShowDrawingModal: (data: string) => void;
  handleProcessClick: (orderId: number, itemId: number, processId: number, status: string, name: string) => void;
  getOrderMaxDueDate: (order: Order) => string;
  fetchData: (dateType?: string) => void;
  fetchOrdersWithFilters: (filters: any, page?: number, pageSize?: number, dateType?: string) => Promise<void>;
  hideCostFields?: boolean;
}

const Warning: React.FC<WarningProps> = ({ orders, orderTotal, warningFilters, setWarningFilters, warningPage, setWarningPage, warningPageSize, setWarningPageSize, editOrder, setShowDrawingModal, handleProcessClick, getOrderMaxDueDate, fetchOrdersWithFilters, hideCostFields = false }) => {
  const { isSearching, queryError, run, retry } = useOrderQuery();

  const applied = React.useRef(warningFilters);
  const handlePageChange = async (nextPage: number, nextSize = warningPageSize, nextFilters = applied.current) => {
    await run(() => fetchOrdersWithFilters(nextFilters, nextPage, nextSize, 'warning'));
  };

  const handleSearch = (nextFilters = warningFilters) => {
    applied.current = nextFilters;
    setWarningPage(1);
    return handlePageChange(1, warningPageSize, nextFilters);
  };

  const warningOrders = orders; // 后端已通过 dateType='warning' 筛选

  return (
    <OrderMonitorPanel
      title="告警订单"
      icon={AlertTriangle}
      orders={warningOrders}
      filters={warningFilters}
      setFilters={setWarningFilters}
      filterConfigs={simpleFilterConfigs}
      total={orderTotal}
      page={warningPage}
      setPage={setWarningPage}
      pageSize={warningPageSize}
      setPageSize={setWarningPageSize}
      themeColor="orange"
      editOrder={editOrder}
      setShowDrawingModal={setShowDrawingModal}
      handleProcessClick={handleProcessClick}
      getOrderMaxDueDate={getOrderMaxDueDate}
      showOrderName={true}
      showContactName={true}
      showOutsourcingFee={!hideCostFields}
      showTotalAmount={!hideCostFields}
      onSearch={() => handleSearch()}
      onReset={handleSearch}
      onPageChangeWithFilters={handlePageChange}
      queryError={queryError}
      onRetryQuery={retry}
      isSearching={isSearching}
    />
  );
};

export default Warning;
