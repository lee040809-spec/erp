import DeliveryReconciliation from "../features/accounting/DeliveryReconciliation";
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
  const [mode, setMode] = React.useState<"orders" | "deliveries">("deliveries");
  const [deliveryBusy, setDeliveryBusy] = React.useState(false);
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

  const selectedSnapshots = React.useRef(new Map<number, Order>());
  const applied = React.useRef(reconciliationFilters);
  const handlePageChange = async (
    nextPage: number,
    nextSize = reconciliationPageSize,
    nextFilters = applied.current,
  ) => {
    if (busy.current) return;
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
    if (
      selectedOrderIds.size &&
      nextFilters.customerName &&
      nextFilters.customerName !== applied.current.customerName &&
      !selectedCustomer.includes(nextFilters.customerName)
    ) {
      if (!window.confirm("切换客户将清空当前已选订单，是否继续？")) return;
      setSelectedOrderIds(new Set());
      selectedSnapshots.current.clear();
    }
    applied.current = nextFilters;
    setReconciliationPage(1);
    return handlePageChange(1, reconciliationPageSize, nextFilters);
  };
  React.useEffect(() => {
    if (mode === "orders") void handlePageChange(1);
  }, [mode]);
  // 多选订单处理
  const handleSelectOrder = (orderId: number | null) => {
    if (orderId === null || busy.current || isSearching || queryError) return;
    const next = new Set(selectedOrderIds);
    if (next.has(orderId)) next.delete(orderId);
    else next.add(orderId);
    const candidate = orders.find((o) => o.id === orderId);
    if (candidate) selectedSnapshots.current.set(orderId, candidate);
    if (next.size > 200) {
      setNotice("一次最多选择200个订单，请分次核对。");
      return;
    }
    const problem = next.size
      ? reconciliationSelectionError(
          [...selectedSnapshots.current.values()].filter((o) =>
            next.has(o.id || 0),
          ),
        )
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
    return [...selectedSnapshots.current.values()].filter((o) =>
      selectedOrderIds.has(o.id || 0),
    );
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
      <div className="flex gap-2">
        <button
          className={`erp-btn ${mode === "deliveries" ? "erp-primary" : ""}`}
          disabled={previewBusy || deliveryBusy}
          onClick={() => setMode("deliveries")}
        >
          按实际送货对账
        </button>
        <button
          className={`erp-btn ${mode === "orders" ? "erp-primary" : ""}`}
          disabled={previewBusy || deliveryBusy}
          onClick={() => setMode("orders")}
        >
          按整单约定金额核对
        </button>
      </div>
      <section hidden={mode !== "deliveries"}>
        <DeliveryReconciliation onBusyChange={setDeliveryBusy} />
      </section>
      {mode === "orders" && (
        <>
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
          {!!selectedOrders.length && (
            <aside className="erp-card p-3 bg-blue-50 text-sm">
              <strong>
                {selectedCustomer} · 已选 {selectedOrders.length} 单 ·{" "}
                {selectedOrders.reduce(
                  (n, o) =>
                    n +
                    (o.items || []).reduce(
                      (q, i) => q + Number(i.quantity || 0),
                      0,
                    ),
                  0,
                )}{" "}
                件
                {!hideCostFields &&
                  ` · 参考金额 ${selectedOrders.reduce((n, o) => n + Number(o.total_amount || 0), 0).toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`}
              </strong>
              <p className="text-xs text-slate-500 mt-1">
                跨页保留；生成前由服务器重新校验当前资料。
              </p>
              <div className="max-h-24 overflow-auto mt-2">
                {selectedOrders.map((o) => (
                  <button
                    key={o.id}
                    className="erp-btn mr-2 mb-1"
                    disabled={previewBusy || isSearching}
                    onClick={() => handleSelectOrder(o.id!)}
                  >
                    {o.order_number} × 移除
                  </button>
                ))}
              </div>
            </aside>
          )}
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
                ? `${selectedCustomer} · 已选 ${selectedOrders.length} 个订单（跨页保留）`
                : "先勾选同一客户的订单"
            }
            onClearReconciliationSelection={() => {
              setSelectedOrderIds(new Set());
              selectedSnapshots.current.clear();
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
        </>
      )}
    </div>
  );
};

export default Reconciliation;
