import React from 'react';
import { AlertCircle } from 'lucide-react';
import { Order } from '../types';
import OrderMonitorPanel from '../components/OrderMonitorPanel';
import { simpleFilterConfigs } from '../configs/filterConfigs';

interface OverdueProps {
  orders: Order[];
  orderTotal: number;
  overdueFilters: any;
  setOverdueFilters: (filters: any) => void;
  overduePage: number;
  setOverduePage: (page: number) => void;
  overduePageSize: number;
  setOverduePageSize: (size: number) => void;
  editOrder: (order: Order) => void;
  setShowDrawingModal: (data: string) => void;
  handleProcessClick: (orderId: number, itemId: number, processId: number, status: string, name: string) => void;
  getOrderMaxDueDate: (order: Order) => string;
  fetchData: (dateType?: string) => void;
  fetchOrdersWithFilters: (filters: any, page?: number, pageSize?: number, dateType?: string) => Promise<void>;
  hideCostFields?: boolean;
}

const Overdue: React.FC<OverdueProps> = ({ orders, orderTotal, overdueFilters, setOverdueFilters, overduePage, setOverduePage, overduePageSize, setOverduePageSize, editOrder, setShowDrawingModal, handleProcessClick, getOrderMaxDueDate, fetchOrdersWithFilters, hideCostFields = false }) => {
  const [isSearching, setIsSearching] = React.useState(false);

  const applied = React.useRef(overdueFilters);
  const handlePageChange = async (nextPage: number, nextSize = overduePageSize, nextFilters = applied.current) => {
    setIsSearching(true);
    try {
      await fetchOrdersWithFilters(nextFilters, nextPage, nextSize, 'overdue');
    } finally {
      setIsSearching(false);
    }
  };

  const handleSearch = (nextFilters = overdueFilters) => {
    applied.current = nextFilters;
    setOverduePage(1);
    return handlePageChange(1, overduePageSize, nextFilters);
  };

  const overdueOrders = orders; // 后端已通过 dateType='overdue' 筛选

  return (
    <OrderMonitorPanel
      title="逾期订单"
      icon={AlertCircle}
      orders={overdueOrders}
      filters={overdueFilters}
      setFilters={setOverdueFilters}
      filterConfigs={simpleFilterConfigs}
      total={orderTotal}
      page={overduePage}
      setPage={setOverduePage}
      pageSize={overduePageSize}
      setPageSize={setOverduePageSize}
      themeColor="rose"
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
      isSearching={isSearching}
    />
  );
};

export default Overdue;
