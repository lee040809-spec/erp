import { formatAmount } from '../utils/money';
import React, { useEffect, useState, useRef } from 'react';
import { X, Info, Plus, Download } from 'lucide-react';
import { Order } from '../types';
import { authFetch } from './shared';
import UniverSheet from './UniverSheet';
import pako from 'pako';
import { exportUniverToExcel } from '../utils/univerExport';
import {
  getFieldValue,
  replaceReconciliationOrderPlaceholders,
  replaceItemPlaceholders,
  hasItemPlaceholder,
  hasOrderPlaceholder
} from '../utils/templateReplace';

interface PrintTemplate {
  id: number;
  name: string;
  template: string; // gzip压缩的Univer workbook数据（base64编码）
  excel_filename?: string;
}

interface ReconciliationPreviewModalProps {
  orders?: Order[]; // 多选订单（预览模式）
  template: PrintTemplate;
  mode?: 'preview' | 'config';
  title?: string;
  orderData?: any;
  onClose: () => void;
}

// 订单字段列表
const orderFields = [
  { key: 'order.order_number', label: '订单编号' },
  { key: 'order.customer_name', label: '客户名称' },
  { key: 'order.customer_short_name', label: '客户简称' },
  { key: 'order.contact_name', label: '联系人' },
  { key: 'order.contact_info', label: '联系方式' },
  { key: 'order.start_date', label: '订单日期' },
  { key: 'order.due_date', label: '交货日期' },
  { key: 'order.total_amount', label: '订单总额' },
  { key: 'order.notes', label: '备注' },
  { key: 'order.priority', label: '优先级' },
];

// 其他字段列表（计算字段）- 对账单专用
const otherFields = [
  { key: 'other.small_total', label: '小写总计' },
  { key: 'other.big_total', label: '大写总计' },
  { key: 'other.total_quantity', label: '总数量' },
  { key: 'other.reconciliation_number', label: '对账单号' },
];

// 零件字段列表（带#前缀表示列表变量）
const itemFields = [
  { key: 'items.row_index', label: '序号', isList: true },
  { key: 'items.part_name', label: '零件名称', isList: true },
  { key: 'items.part_number', label: '零件号', isList: true },
  { key: 'items.quantity', label: '数量', isList: true },
  { key: 'items.unit_price', label: '单价', isList: true },
  { key: 'items.total_price', label: '总计', isList: true },
  { key: 'items.completion_date', label: '完工日期', isList: true },
  { key: 'items.delivered_quantity', label: '交货数量', isList: true },
  { key: 'items.scrap_quantity', label: '报废数量', isList: true },
  { key: 'items.notes', label: '零件备注', isList: true },
];

const ReconciliationPreviewModal: React.FC<ReconciliationPreviewModalProps> = ({
  orders,
  template,
  mode = 'config',
  title = '对账单',
  orderData,
  onClose
}) => {
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);
  const univerRef = useRef<any>(null);
  const [sheetData, setSheetData] = useState<any>(null);

  // 加载模板数据
  useEffect(() => {
    if (!template?.template) return;

    try {
      let workbookData: any;
      const templateStr = template.template;

      // 检查数据格式
      if (templateStr.startsWith('UNIVER:')) {
        // 新格式：gzip压缩的Univer workbook数据
        const base64 = templateStr.substring(7);
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
          bytes[i] = binary.charCodeAt(i);
        }
        const decompressed = pako.inflate(bytes, { to: 'string' });
        workbookData = JSON.parse(decompressed);
      } else if (templateStr.startsWith('GZIP:')) {
        // 旧格式：gzip压缩数据
        const base64 = templateStr.substring(5);
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
          bytes[i] = binary.charCodeAt(i);
        }
        const decompressed = pako.inflate(bytes, { to: 'string' });
        workbookData = JSON.parse(decompressed);
      } else {
        // 直接JSON或其他格式
        workbookData = JSON.parse(templateStr);
      }

      // 只设置原始模板数据，不预处理
      setSheetData(workbookData);
    } catch (err) {
      // 解析失败，静默处理
    }
  }, [template?.template]);

  // preview 模式下，使用 Univer API 填充数据
  useEffect(() => {
    if (mode !== 'preview' || !orderData || !univerRef.current) return;

    // 等待 Univer ready
    const checkReady = () => {
      if (univerRef.current?.isReady?.()) {
        fillDataWithUniverAPI();
      } else {
        setTimeout(checkReady, 100);
      }
    };
    checkReady();
  }, [mode, orderData, sheetData]);

  // 使用 Univer API 填充数据
  const fillDataWithUniverAPI = () => {
    if (!univerRef.current || !orderData) {
      return;
    }

    // 处理订单数据：如果是数组，合并所有订单的 items
    let mergedOrderData: any = orderData;
    if (Array.isArray(orderData)) {
      const allItems: any[] = [];
      let totalQuantity = 0;
      let totalAmount = 0;
      let rowIndex = 0;

      for (const order of orderData) {
        for (const item of (order.items || [])) {
          allItems.push({
            ...item,
            row_index: rowIndex + 1
          });
          rowIndex++;
          totalQuantity += item.quantity || 0;
          totalAmount += item.total_price || 0;
        }
      }

      mergedOrderData = {
        ...orderData[0],
        items: allItems,
        other: {
          ...orderData[0]?.other,
          total_quantity: totalQuantity,
          small_total: totalAmount.toFixed(2),
          big_total: orderData[0]?.other?.big_total || '零元整'
        }
      };
    }

    const items = mergedOrderData.items || [];

    // 1. 查找零件模板行
    const templateCell = univerRef.current.findCellWithText?.('{{#items.');

    if (!templateCell) {
      // 没有零件占位符，只替换订单字段
      replaceOrderFields(mergedOrderData);
      return;
    }

    const templateRow = templateCell.row;

    // 2. 获取模板行所有列的值
    const templateRowValues: Map<number, string> = new Map();
    for (let col = 0; col < 20; col++) {
      try {
        const val = univerRef.current.getCellValue?.(templateRow, col) || '';
        if (val) {
          templateRowValues.set(col, val);
        }
      } catch (e) {}
    }

    // 3. 插入行（如果零件数量大于1）
    if (items.length > 1) {
      univerRef.current.insertRowsBelow?.(templateRow, items.length - 1);
    }

    // 4. 遍历每个零件，替换占位符
    for (let i = 0; i < items.length; i++) {
      const currentRow = templateRow + i;
      const item = { ...items[i], row_index: i + 1 };

      for (const [col, cellValue] of templateRowValues) {
        let newValue = cellValue;

        // 替换零件占位符 {{#items.xxx}}
        newValue = replaceItemPlaceholders(newValue, item);

        // 替换订单占位符 {{order.xxx}} {{other.xxx}}
        newValue = replaceReconciliationOrderPlaceholders(newValue, mergedOrderData);

        if (newValue !== cellValue) {
          try {
            univerRef.current.setCellValue?.(currentRow, col, newValue);
          } catch (e) {}
        }
      }
    }

    // 5. 替换其他行的订单字段
    replaceOrderFields(mergedOrderData, templateRow, items.length);
  };

  // 替换订单字段占位符
  const replaceOrderFields = (orderData: any, skipStartRow?: number, skipRowCount?: number) => {
    if (!univerRef.current) return;

    const rowCount = Math.min(univerRef.current.getRowCount?.() || 100, 100);

    for (let row = 0; row < rowCount; row++) {
      // 跳过零件行
      if (skipStartRow !== undefined && row >= skipStartRow && row < skipStartRow + (skipRowCount || 0)) {
        continue;
      }

      // 遍历列查找 {{order. 或 {{other. 占位符
      for (let col = 0; col < 20; col++) {
        try {
          const cellValue = univerRef.current.getCellValue?.(row, col) || '';
          if (cellValue.includes('{{order.') || cellValue.includes('{{other.')) {
            const newValue = replaceReconciliationOrderPlaceholders(cellValue, orderData);
            if (newValue !== cellValue) {
              univerRef.current.setCellValue?.(row, col, newValue);
            }
          }
        } catch (e) {
          // 跳过错误
        }
      }
    }
  };

  const showToast = (message: string, type: 'success' | 'error') => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 1000);
  };

  // 插入字段到选中单元格或编辑光标位置
  const insertField = (field: { key: string; label: string; isList?: boolean }) => {
    if (!univerRef.current) return;

    const placeholder = field.isList ? `{{#${field.key}}}` : `{{${field.key}}}`;

    if (univerRef.current.isEditing && univerRef.current.isEditing()) {
      univerRef.current.insertTextAtCursor(placeholder);
    } else {
      const selection = univerRef.current.getSelection();
      if (!selection) {
        showToast('请先选择单元格', 'error');
        return;
      }
      univerRef.current.setCellValue(selection.row, selection.col, placeholder);
    }
  };

  // 字段点击处理
  const handleFieldClick = (field: { key: string; label: string; isList?: boolean }) => {
    if (!univerRef.current || !univerRef.current.isReady()) {
      showToast('模板正在加载，请稍候', 'error');
      return;
    }
    insertField(field);
  };

  // 导出Excel
  const handleExportExcel = async () => {
    if (!univerRef.current) {
      showToast('表格未加载完成', 'error');
      return;
    }

    const snapshot = univerRef.current.getSnapshot();
    if (!snapshot) {
      showToast('获取数据失败', 'error');
      return;
    }

    try {
      const orderNumberStr = orders && orders.length > 0
        ? (orders.length === 1 ? orders[0].order_number : `${orders.length}单汇总`)
        : '未知';
      const filename = `对账单_${orderNumberStr}_${new Date().toISOString().slice(0, 10)}.xlsx`;
      await exportUniverToExcel(snapshot, filename);
      showToast('导出成功', 'success');
    } catch (err) {
      showToast('导出失败', 'error');
    }
  };

  // 保存模板
  const handleSaveTemplate = async () => {
    if (!univerRef.current) return;

    const workbookData = univerRef.current.getRawData();
    if (!workbookData) {
      showToast('获取模板数据失败', 'error');
      return;
    }

    try {
      const workbookJson = JSON.stringify(workbookData);
      const compressed = pako.gzip(workbookJson);
      const base64 = btoa(String.fromCharCode(...compressed));
      const compressedData = 'UNIVER:' + base64;

      const res = await authFetch(`/api/platform/print-templates/${template.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ template: compressedData })
      });

      const result = await res.json();
      if (res.ok) {
        showToast('保存成功', 'success');
      } else {
        showToast(result.error || '保存失败', 'error');
      }
    } catch (err) {
      showToast('保存失败', 'error');
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 z-[100] flex items-center justify-center">
      <div className="bg-white rounded-2xl shadow-xl w-[95vw] h-[90vh] overflow-hidden flex flex-col relative">
        {/* 头部 */}
        <div className="p-4 border-b border-zinc-200 flex justify-between items-center">
          <div className="flex items-center gap-4">
            <h3 className="text-xl font-bold text-zinc-900">{title}</h3>
            {orders && orders.length > 0 && (
              <span className="text-sm text-zinc-500">
                {orders.length === 1
                  ? `订单号: ${orders[0].order_number}`
                  : `已选 ${orders.length} 个订单`}
              </span>
            )}
          </div>
          <div className="flex items-center gap-3">
            {mode === 'preview' && (
              <button
                onClick={handleExportExcel}
                className="bg-blue-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-blue-700 transition-colors flex items-center gap-2"
              >
                <Download className="w-4 h-4" />
                导出Excel
              </button>
            )}
            {mode === 'config' && (
              <button
                onClick={handleSaveTemplate}
                className="bg-emerald-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-emerald-700 transition-colors"
              >
                保存模板
              </button>
            )}
            <button
              onClick={onClose}
              className="text-zinc-400 hover:text-zinc-600 p-2"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* 内容区域 */}
        <div className="flex-1 flex overflow-hidden" style={{ position: 'relative' }}>
          {/* 左边：UniverSheet模板 */}
          <div className="flex-1 overflow-hidden" style={{ position: 'relative', minHeight: 0 }}>
            {sheetData ? (
              <UniverSheet
                ref={univerRef}
                data={sheetData}
                height="100%"
              />
            ) : (
              <div className="flex items-center justify-center h-full text-zinc-400">
                模板数据加载中...
              </div>
            )}
          </div>

          {/* 右边：字段面板（仅config模式显示） */}
          {mode === 'config' && (
          <div className="w-72 border-l border-zinc-200 bg-zinc-50 overflow-y-auto overflow-x-visible">
            <div className="p-4 min-w-[260px]">

              {/* 订单字段 */}
              <div className="mb-4">
                <h4 className="text-xs font-bold text-zinc-700 mb-2 flex items-center gap-1">
                  订单字段
                  <span className="relative group">
                    <Info className="w-3 h-3 text-blue-500 cursor-help" />
                    <span className="absolute left-0 top-4 w-[180px] p-2 bg-blue-50 text-blue-700 text-xs rounded-lg border border-blue-200 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-50 pointer-events-none shadow-sm">
                      <span className="absolute -top-1 left-0 w-2 h-2 bg-blue-50 border-l border-t border-blue-200 rotate-45"></span>
                      点击单元格后点击字段填充，或双击进入编辑后在光标位置插入
                    </span>
                  </span>
                </h4>
                <div className="space-y-1">
                  {orderFields.map(field => (
                    <button
                      key={field.key}
                      onClick={() => handleFieldClick(field)}
                      className="inline-flex items-center gap-2 px-2 py-1.5 bg-white rounded-lg border border-zinc-200 hover:bg-zinc-50 hover:border-zinc-300 transition-colors whitespace-nowrap w-[280px]"
                    >
                      <Plus className="w-3 h-3 text-zinc-400" />
                      <span className="text-xs text-zinc-700">{field.label}</span>
                      <code className="text-[10px] text-zinc-400 ml-2">{"{{" + field.key + "}}"}</code>
                    </button>
                  ))}
                </div>
              </div>

              {/* 其他字段 */}
              <div className="mb-4">
                <h4 className="text-xs font-bold text-zinc-700 mb-2 flex items-center gap-1">
                  其他字段
                  <span className="text-[10px] text-purple-600">(计算字段)</span>
                </h4>
                <div className="space-y-1">
                  {otherFields.map(field => (
                    <button
                      key={field.key}
                      onClick={() => handleFieldClick(field)}
                      className="inline-flex items-center gap-2 px-2 py-1.5 bg-white rounded-lg border border-purple-200 hover:bg-purple-50 hover:border-purple-300 transition-colors whitespace-nowrap w-[280px]"
                    >
                      <Plus className="w-3 h-3 text-purple-400" />
                      <span className="text-xs text-zinc-700">{field.label}</span>
                      <code className="text-[10px] text-purple-500 ml-2">{"{{" + field.key + "}}"}</code>
                    </button>
                  ))}
                </div>
              </div>

              {/* 零件字段 */}
              <div>
                <h4 className="text-xs font-bold text-zinc-700 mb-2 flex items-center gap-1">
                  零件字段
                  <span className="relative group">
                    <Info className="w-3 h-3 text-amber-500 cursor-help" />
                    <span className="absolute left-0 top-4 w-[180px] p-2 bg-amber-50 text-amber-700 text-xs rounded-lg border border-amber-200 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-50 pointer-events-none shadow-sm">
                      <span className="absolute -top-1 left-0 w-2 h-2 bg-amber-50 border-l border-t border-amber-200 rotate-45"></span>
                      零件字段以 {'{{#字段名}}'} 格式标记，包含零件字段的行将作为零件模板行，预览时会根据零件数量自动复制多行
                    </span>
                  </span>
                  <span className="text-[10px] text-amber-600">(列表变量)</span>
                </h4>
                <div className="space-y-1">
                  {itemFields.map(field => (
                    <button
                      key={field.key}
                      onClick={() => handleFieldClick(field)}
                      className="inline-flex items-center gap-2 px-2 py-1.5 bg-white rounded-lg border border-amber-200 hover:bg-amber-50 hover:border-amber-300 transition-colors whitespace-nowrap w-[280px]"
                    >
                      <Plus className="w-3 h-3 text-amber-400" />
                      <span className="text-xs text-zinc-700">{field.label}</span>
                      <code className="text-[10px] text-amber-500 ml-2">{"{{#" + field.key + "}}"}</code>
                    </button>
                  ))}
                </div>
              </div>

              {/* 零件数据预览 */}
              {orders && orders.length > 0 && (
              <div className="mt-6 pt-4 border-t border-zinc-200">
                <h4 className="text-sm font-bold text-zinc-700 mb-2">
                  零件汇总 ({orders.reduce((sum, o) => sum + (o.items?.length || 0), 0)} 个)
                </h4>
                <div className="space-y-2">
                  {orders.flatMap(o => o.items || []).slice(0, 5).map((item, idx) => (
                    <div key={idx} className="p-2 bg-white rounded-lg border border-zinc-200 text-xs">
                      <div className="font-medium text-zinc-900">{item.part_name}</div>
                      <div className="text-zinc-500 mt-1">
                        数量: {item.quantity} | 单价: {formatAmount(item.unit_price)}
                      </div>
                    </div>
                  ))}
                  {orders.reduce((sum, o) => sum + (o.items?.length || 0), 0) > 5 && (
                    <div className="text-xs text-zinc-400 text-center">
                      还有 {orders.reduce((sum, o) => sum + (o.items?.length || 0), 0) - 5} 个零件...
                    </div>
                  )}
                </div>
              </div>
              )}
            </div>
          </div>
          )}
        </div>
      </div>

      {/* Toast提示 */}
      {toast && (
        <div
          className={`fixed inset-0 z-[200] flex items-center justify-center bg-black/30 animate-in fade-in duration-200`}
        >
          <div
            className={`px-8 py-4 rounded-xl shadow-xl flex items-center gap-3 animate-in zoom-in duration-300 ${
              toast.type === 'success'
                ? 'bg-emerald-500 text-white'
                : 'bg-red-500 text-white'
            }`}
          >
            {toast.type === 'success' ? (
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            ) : (
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            )}
            <span className="font-medium text-lg">{toast.message}</span>
          </div>
        </div>
      )}
    </div>
  );
};

export default ReconciliationPreviewModal;