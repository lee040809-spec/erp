import React from 'react';
import { Truck } from 'lucide-react';
import { Order } from '../types';
import OrderMonitorPanel from '../components/OrderMonitorPanel';
import DeliveryPreviewModal from '../components/DeliveryPreviewModal';
import { simpleFilterConfigs } from '../configs/filterConfigs';
import { authFetch } from '../components/shared';
interface PrintTemplate {
    id: number;
    name: string;
    template: string;
    excel_filename?: string;
}
interface DeliveryProps {
    orders: Order[];
    orderTotal: number;
    deliveryFilters: any;
    setDeliveryFilters: (filters: any) => void;
    deliveryPage: number;
    setDeliveryPage: (page: number) => void;
    deliveryPageSize: number;
    setDeliveryPageSize: (size: number) => void;
    editOrder: (order: Order) => void;
    setShowDrawingModal: (data: string) => void;
    handleProcessClick: (orderId: number, itemId: number, processId: number, status: string, name: string) => void;
    getOrderMaxDueDate: (order: Order) => string;
    fetchData: () => void;
    fetchOrdersWithFilters: (filters: any, page?: number, pageSize?: number, dateType?: string) => Promise<void>;
    hideCostFields?: boolean;
    showToast?: (message: string, type: 'success' | 'error') => void;
}
const Delivery: React.FC<DeliveryProps> = ({ orders, orderTotal, deliveryFilters, setDeliveryFilters, deliveryPage, setDeliveryPage, deliveryPageSize, setDeliveryPageSize, editOrder, setShowDrawingModal, handleProcessClick, getOrderMaxDueDate, fetchOrdersWithFilters, hideCostFields = false, showToast }) => {
    const [isSearching, setIsSearching] = React.useState(false);
    const [selectedOrderId, setSelectedOrderId] = React.useState<number | null>(null);
    const [showPreviewModal, setShowPreviewModal] = React.useState(false);
    const [previewOrder, setPreviewOrder] = React.useState<Order | null>(null);
    const [previewTemplate, setPreviewTemplate] = React.useState<PrintTemplate | null>(null);
    const [previewOrderData, setPreviewOrderData] = React.useState<any>(null);
    const [showConfigModal, setShowConfigModal] = React.useState(false);
    const [configTemplate, setConfigTemplate] = React.useState<PrintTemplate | null>(null);
    const applied=React.useRef(deliveryFilters);
    const handlePageChange = async (nextPage:number,nextSize=deliveryPageSize,nextFilters=applied.current) => {
        setIsSearching(true);
    setSelectedOrderId(null);
        try {
            const filtersWithStatus = { ...nextFilters, status: 'delivery_ready' };
            await fetchOrdersWithFilters(filtersWithStatus, nextPage, nextSize);
        }
        finally {
            setIsSearching(false);
        }
    };
    const handleSearch=(nextFilters=deliveryFilters)=>{applied.current=nextFilters;setDeliveryPage(1);return handlePageChange(1,deliveryPageSize,nextFilters);};
    const handleSelectOrder = (orderId: number | null) => {
        setSelectedOrderId(prev => prev === orderId ? null : orderId);
    };
    const handlePreviewDelivery = async () => {
        if (!selectedOrderId)
            return;
        const order = orders.find(o => o.id === selectedOrderId);
        if (!order)
            return;
        try {
            const res = await authFetch(`/api/platform/print-templates/by-button?menu_route=production_delivery&button_key=btn-config-delivery-note&mode=preview&order_id=${selectedOrderId}`);
            const result = await res.json();
            if (result.data) {
                setPreviewOrder(order);
                setPreviewTemplate(result.data);
                setPreviewOrderData(result.order || null);
                setShowPreviewModal(true);
            }
            else {
                showToast?.(result.error || '未找到绑定的送货单模板', 'error');
            }
        }
        catch (err) {
            showToast?.('查询模板失败', 'error');
        }
    };
    const handleConfigDelivery = async () => {
        try {
            const res = await authFetch(`/api/platform/print-templates/by-button?menu_route=production_delivery&button_key=btn-config-delivery-note&mode=config`);
            const result = await res.json();
            if (result.data) {
                setConfigTemplate(result.data);
                setShowConfigModal(true);
            }
            else {
                showToast?.(result.error || '未找到绑定的送货单配置模板', 'error');
            }
        }
        catch (err) {
            showToast?.('查询模板失败', 'error');
        }
    };
    const completedOrders = orders;
    return (<>
      <OrderMonitorPanel title="送货管理" icon={Truck} orders={completedOrders} filters={deliveryFilters} setFilters={setDeliveryFilters} filterConfigs={simpleFilterConfigs} total={orderTotal}
      page={deliveryPage} setPage={setDeliveryPage} pageSize={deliveryPageSize} setPageSize={setDeliveryPageSize}  themeColor="emerald" editOrder={editOrder} setShowDrawingModal={setShowDrawingModal} handleProcessClick={handleProcessClick} getOrderMaxDueDate={getOrderMaxDueDate} showOrderName={true} showContactName={true} showOutsourcingFee={!hideCostFields} showTotalAmount={!hideCostFields} onSearch={()=>handleSearch()}
      onReset={handleSearch}
      onPageChangeWithFilters={handlePageChange} isSearching={isSearching} deliveryMode={true} selectedOrderId={selectedOrderId} onSelectOrder={handleSelectOrder} onPreviewDelivery={handlePreviewDelivery} onConfigDelivery={handleConfigDelivery}/>

      {/* 送货单预览弹窗 */}
      {showPreviewModal && previewOrder && previewTemplate && (<DeliveryPreviewModal order={previewOrder} template={previewTemplate} mode="preview" title="预览送货单" orderData={previewOrderData} onClose={() => setShowPreviewModal(false)}/>)}

      {/* 送货单配置弹窗 - 不传订单 */}
      {showConfigModal && configTemplate && (<DeliveryPreviewModal template={configTemplate} mode="config" title="配置送货单" onClose={() => setShowConfigModal(false)}/>)}
    </>);
};
export default Delivery;
