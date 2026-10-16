import { businessToday } from "../utils/businessDate";
import React from "react";
import { useOrderQuery } from "../hooks/useOrderQuery";
import AccountingGuide from "../features/accounting/AccountingGuide";
import { reconciliationSelectionError } from "../features/accounting/reconciliationSelection";
import { FileText } from "lucide-react";
import { Order } from "../types";
import OrderMonitorPanel from "../components/OrderMonitorPanel";
import ReconciliationPreviewModal from "../components/ReconciliationPreviewModal";
import { simpleFilterConfigs } from "../configs/filterConfigs";
import { authFetch } from "../components/shared";

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
  handleProcessClick: (
    orderId: number,
    itemId: number,
    processId: number,
    status: string,
    name: string,
  ) => void;
  getOrderMaxDueDate: (order: Order) => string;
  fetchData: () => void;
  fetchOrdersWithFilters: (
    filters: any,
    page?: number,
    pageSize?: number,
    dateType?: string,
  ) => Promise<void>;
  hideCostFields?: boolean;
  showToast?: (message: string, type: "success" | "error") => void;
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
}) => {
  const { isSearching, queryError, run, retry } = useOrderQuery();
  const [previewBusy, setPreviewBusy] = React.useState(false),
    [notice, setNotice] = React.useState("");
  const [reconciliationDate, setReconciliationDate] =
    React.useState(businessToday());
  const busy = React.useRef(false),
    active = React.useRef(true);
  React.useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  // 多选订单ID
  const [selectedOrderIds, setSelectedOrderIds] = React.useState<Set<number>>(
    new Set(),
  );
  const [showPreviewModal, setShowPreviewModal] = React.useState(false);
  const [previewOrders, setPreviewOrders] = React.useState<Order[]>([]);
  const [previewTemplate, setPreviewTemplate] =
    React.useState<PrintTemplate | null>(null);
  const [previewOrderData, setPreviewOrderData] = React.useState<any>(null);
  const [showConfigModal, setShowConfigModal] = React.useState(false);
  const [configTemplate, setConfigTemplate] =
    React.useState<PrintTemplate | null>(null);

  const applied = React.useRef(reconciliationFilters);
  const handlePageChange = async (
    nextPage: number,
    nextSize = reconciliationPageSize,
    nextFilters = applied.current,
  ) => {
    if (busy.current) return;
    setSelectedOrderIds(new Set());
    setNotice("");
    await run(() =>
      fetchOrdersWithFilters(
        { ...nextFilters, status: "completed" },
        nextPage,
        nextSize,
      ),
    );
  };

  const handleSearch = (nextFilters = reconciliationFilters) => {
    applied.current = nextFilters;
    setReconciliationPage(1);
    return handlePageChange(1, reconciliationPageSize, nextFilters);
  };
  React.useEffect(() => {
    void handlePageChange(1);
  }, []);
  // 多选订单处理
  const handleSelectOrder = (orderId: number | null) => {
    if (orderId === null || busy.current || isSearching || queryError) return;
    const next = new Set(selectedOrderIds);
    if (next.has(orderId)) next.delete(orderId);
    else next.add(orderId);
    const problem = next.size
      ? reconciliationSelectionError(orders.filter((o) => next.has(o.id || 0)))
      : "";
    if (problem) {
      setNotice(problem);
      return;
    }
    setNotice("");
    setSelectedOrderIds(next);
  };

  // 获取选中的订单列表
  const getSelectedOrders = () => {
    return orders.filter((o) => selectedOrderIds.has(o.id || 0));
  };

  const selectedOrders = getSelectedOrders();
  const selectedCustomer =
    selectedOrders[0]?.customer_short_name ||
    selectedOrders[0]?.customer_name ||
    "";
  const readTemplate = async (config: boolean) => {
    if (busy.current || isSearching || queryError) return;
    if (!config && !/^\d{4}-\d{2}-\d{2}$/.test(reconciliationDate)) {
      setNotice("请选择本次对账日期。");
      return;
    }
    const snapshot = [...selectedOrders];
    const problem = config ? "" : reconciliationSelectionError(snapshot);
    if (problem) {
      setNotice(problem);
      return;
    }
    busy.current = true;
    setPreviewBusy(true);
    setNotice("");
    try {
      const ids = snapshot.map((o) => o.id).join(",");
      const res = await authFetch(
        "/api/platform/print-templates/by-button?menu_route=reconciliation&button_key=btn-config-reconciliation&mode=" +
          (config ? "config" : "preview&order_ids=" + encodeURIComponent(ids)),
      );
      const result = await res.json();
      if (!res.ok || !result.data)
        throw Error(
          result.message ||
            result.error ||
            "暂时没有可用的对账单格式，请联系管理员设置。",
        );
      if (!active.current) return;
      if (config) {
        setConfigTemplate(result.data);
        setShowConfigModal(true);
      } else {
        setPreviewOrders(snapshot);
        setPreviewTemplate(result.data);
        setPreviewOrderData(result.order || null);
        setShowPreviewModal(true);
      }
    } catch (err) {
      if (active.current)
        setNotice(
          err instanceof Error ? err.message : "对账单读取失败，请重试。",
        );
    } finally {
      busy.current = false;
      if (active.current) setPreviewBusy(false);
    }
  };
  const handlePreviewReconciliation = () => readTemplate(false);
  const handleConfigReconciliation = () => readTemplate(true);

  const completedOrders = orders;

  return (
    <div className="erp-page space-y-3">
      <AccountingGuide mode="reconciliation" />
      <label className="erp-label flex flex-wrap gap-2 items-center">
        本次对账日期
        <input
          aria-label="本次对账日期"
          type="date"
          required
          className="erp-input !w-auto"
          disabled={previewBusy}
          value={reconciliationDate}
          onChange={(e) => setReconciliationDate(e.target.value)}
        />
        <span className="text-xs text-slate-500">
          只用于本次对账单，不修改订单日期。
        </span>
      </label>
      <p className="text-sm text-slate-500">
        这里展示已完工订单，按订单约定金额核对；不代表已交货、客户已确认或已经付款。
      </p>
      {notice && (
        <p
          role="alert"
          className="rounded-xl bg-amber-50 p-3 text-sm text-amber-900"
        >
          {notice}
        </p>
      )}
      <OrderMonitorPanel
        title="对账管理"
        queryError={queryError}
        onRetryQuery={retry}
        reconciliationBusy={previewBusy}
        reconciliationLabel={
          selectedOrders.length
            ? `${selectedCustomer} · 已选 ${selectedOrders.length} 个订单（本页）`
            : "先勾选同一客户的订单"
        }
        onClearReconciliationSelection={() => {
          setSelectedOrderIds(new Set());
          setNotice("");
        }}
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
        showOutsourcingFee={false}
        showTotalAmount={!hideCostFields}
        onSearch={() => handleSearch()}
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
          reconciliationDate={reconciliationDate}
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
    </div>
  );
};

export default Reconciliation;
