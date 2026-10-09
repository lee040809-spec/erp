import React from 'react';
import { Clock } from 'lucide-react';
import { Order } from '../types';
import OrderMonitorPanel from '../components/OrderMonitorPanel';
import { simpleFilterConfigs } from '../configs/filterConfigs';

interface ImminentProps {
  orders: Order[];
  orderTotal: number;
  imminentFilters: any;
  setImminentFilters: (filters: any) => void;
  imminentPage: number;
  setImminentPage: (page: number) => void;
  imminentPageSize: number;
  setImminentPageSize: (size: number) => void;
  editOrder: (order: Order) => void;
  setShowDrawingModal: (data: string) => void;
  handleProcessClick: (orderId: number, itemId: number, processId: number, status: string, name: string) => void;
  getOrderMaxDueDate: (order: Order) => string;
  fetchData: (dateType?: string) => void;
  fetchOrdersWithFilters: (filters: any, page?: number, pageSize?: number, dateType?: string) => Promise<void>;
  hideCostFields?: boolean;
}

const Imminent: React.FC<ImminentProps> = ({
  orders,
  orderTotal,
  imminentFilters,
  setImminentFilters,
  imminentPage,
  setImminentPage,
  imminentPageSize,
  setImminentPageSize,
  editOrder,
  setShowDrawingModal,
  handleProcessClick,
  getOrderMaxDueDate,
  fetchData,
  fetchOrdersWithFilters,
  hideCostFields = false
}) => {
  const [isSearching, setIsSearching] = React.useState(false);

  const applied = React.useRef(imminentFilters);
  const handlePageChange = async (nextPage: number, nextSize = imminentPageSize, nextFilters = applied.current) => {
    setIsSearching(true);
    try {
      await fetchOrdersWithFilters(nextFilters, nextPage, nextSize, 'near_due');
    } finally {
      setIsSearching(false);
    }
  };

  const handleSearch = (nextFilters = imminentFilters) => {
    applied.current = nextFilters;
    setImminentPage(1);
    return handlePageChange(1, imminentPageSize, nextFilters);
  };

  const imminentOrders = orders; // 后端已通过 dateType='near_due' 筛选

  return (
    <OrderMonitorPanel
      title="临期订单"
      icon={Clock}
      orders={imminentOrders}
      filters={imminentFilters}
      setFilters={setImminentFilters}
      filterConfigs={simpleFilterConfigs}
      total={orderTotal}
      page={imminentPage}
      setPage={setImminentPage}
      pageSize={imminentPageSize}
      setPageSize={setImminentPageSize}
      themeColor="amber"
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

export default Imminent;
