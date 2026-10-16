import PrintTemplateFieldButtons from './PrintTemplateFieldButtons';
import { printItemFields as itemFields } from '../configs/printTemplateFields';
import { replaceItemPlaceholders, replaceDeliveryOrderPlaceholders as replaceOrderPlaceholders } from '../utils/templateReplace';
import { formatAmount as formatDisplayAmount } from '../utils/money';
import React, { useState, useRef } from 'react';
import { X, Info, Download } from 'lucide-react';
import { Order } from '../types';
import { authFetch } from './shared';
import UniverSheet from './UniverSheet';
import { encodePrintTemplate } from '../utils/templateCodec';
import { usePrintWorkbook, usePrintPreview } from '../hooks/usePrintPreview';
import { exportUniverToExcel } from '../utils/univerExport';
interface PrintTemplate {
    id: number;
    name: string;
    template: string; // gzip压缩的Univer workbook数据（base64编码）
    excel_filename?: string;
}
interface DeliveryPreviewModalProps {
    order?: Order; // 可选，配置模式不需要
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
    { key: 'order.due_date', label: '实际交货日期（兼容旧模板）' },
    { key: 'order.customer_due_date', label: '客户交期' },
    { key: 'order.delivery_date', label: '实际交货日期' },
    { key: 'order.total_amount', label: '订单总额' },
    { key: 'order.notes', label: '备注' },
    { key: 'order.priority', label: '优先级' },
];
// 其他字段列表（计算字段）
const otherFields = [
    { key: 'other.small_total', label: '小写总计' },
    { key: 'other.big_total', label: '大写总计' },
    { key: 'other.total_quantity', label: '总数量' },
    { key: 'other.delivery_number', label: '送货单号' },
];
const DeliveryPreviewModal: React.FC<DeliveryPreviewModalProps> = ({ order, template, mode = 'config', title = '送货单', orderData, onClose }) => {
    const [toast, setToast] = useState<{
        message: string;
        type: 'success' | 'error';
    } | null>(null);
    const univerRef = useRef<any>(null);
  const { workbook: sheetData, error: templateError } = usePrintWorkbook(template?.template);
    // 使用 Univer API 填充数据
    const fillDataWithUniverAPI = () => {
        if (!univerRef.current || !orderData) {
            return;
        }
        const items = orderData.items || [];
        // 1. 查找包含零件占位符的模板行
        const templateCell = univerRef.current.findCellWithText?.('{{#items.');
        if (!templateCell) {
            // 没有零件占位符，只替换订单字段
            replaceOrderFields(orderData);
            return;
        }
        const templateRow = templateCell.row;
        const templateCol = templateCell.col; // 从找到的位置开始向左右扩展查找有内容的列
        // 2. 先获取模板行所有有内容的列
        const templateRowValues: Map<number, string> = new Map();
        // 从找到的列开始，向左查找有内容的列
        for (let col = templateCol; col >= 0; col--) {
            try {
                const val = univerRef.current.getCellValue?.(templateRow, col) || '';
                if (val) {
                    templateRowValues.set(col, val);
                }
            }
            catch (e) {
                break;
            }
        }
        // 从找到的列开始，向右查找有内容的列
        for (let col = templateCol + 1; col < 50; col++) {
            try {
                const val = univerRef.current.getCellValue?.(templateRow, col) || '';
                if (val) {
                    templateRowValues.set(col, val);
                }
                else if (!templateRowValues.has(col - 1)) {
                    // 如果前一列没有内容且当前列也没有，停止
                    break;
                }
            }
            catch (e) {
                break;
            }
        }
        // 3. 如果有多个零件，在模板行下方插入 N-1 行
        if (items.length > 1) {
            // 在模板行下方插入行
            univerRef.current.insertRowsBelow?.(templateRow, items.length - 1);
        }
        // 4. 填充每个零件的数据（模板行 + 下方插入的行）
        for (let i = 0; i < items.length; i++) {
            const currentRow = templateRow + i; // 模板行开始，往下填充
            const item = { ...items[i], row_index: i + 1 }; // 添加序号（1-based）
            // 遍历已找到的有内容的列
            for (const [col, cellValue] of templateRowValues) {
                // 替换零件占位符
                let newValue = replaceItemPlaceholders(cellValue, item);
                // 替换订单和其他占位符
                newValue = replaceOrderPlaceholders(newValue, orderData);
                if (newValue !== cellValue) {
                    try {
                        univerRef.current.setCellValue?.(currentRow, col, newValue);
                    }
                    catch (e) {
                        // 设置失败，静默处理
                    }
                }
            }
        }
        // 5. 替换其他行的订单字段
        replaceOrderFields(orderData, templateRow, items.length);
    };
    // 替换订单字段占位符
    const replaceOrderFields = (orderData: any, skipStartRow?: number, skipRowCount?: number) => {
        if (!univerRef.current)
            return;
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
                        const newValue = replaceOrderPlaceholders(cellValue, orderData);
                        if (newValue !== cellValue) {
                            univerRef.current.setCellValue?.(row, col, newValue);
                        }
                    }
                }
                catch (e) {
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
    const insertField = (field: {
        key: string;
        label: string;
        isList?: boolean;
    }) => {
        if (!univerRef.current)
            return;
        const placeholder = field.isList ? `{{#${field.key}}}` : `{{${field.key}}}`;
        // 检查是否在编辑模式（双击单元格后）
        if (univerRef.current.isEditing && univerRef.current.isEditing()) {
            // 在编辑模式下，在光标位置插入文本
            univerRef.current.insertTextAtCursor(placeholder);
        }
        else {
            // 非编辑模式，获取当前选中单元格并设置值
            const selection = univerRef.current.getSelection();
            if (!selection) {
                showToast('请先选择单元格', 'error');
                return;
            }
            univerRef.current.setCellValue(selection.row, selection.col, placeholder);
        }
    };
    // 订单字段点击处理（右侧面板）
    const handleFieldClick = (field: {
        key: string;
        label: string;
        isList?: boolean;
    }) => {
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
            const filename = `送货单_${order?.order_number || '未知'}_${new Date().toISOString().slice(0, 10)}.xlsx`;
            await exportUniverToExcel(snapshot, filename);
            showToast('导出成功', 'success');
        }
        catch (err) {
            showToast('导出失败', 'error');
        }
    };
    // 保存模板
    const handleSaveTemplate = async () => {
        if (!univerRef.current)
            return;
        const workbookData = univerRef.current.getRawData();
        if (!workbookData) {
            showToast('获取模板数据失败', 'error');
            return;
        }
        try {
            // gzip 压缩 workbook JSON
            const compressedData = encodePrintTemplate(workbookData);
            // 保存到数据库
            const res = await authFetch(`/api/platform/print-templates/${template.id}`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ template: compressedData })
            });
            const result = await res.json();
            if (res.ok) {
                showToast('保存成功', 'success');
            }
            else {
                showToast(result.error || '保存失败', 'error');
            }
        }
        catch (err) {
            showToast('保存失败', 'error');
        }
    };
  usePrintPreview(mode, orderData, sheetData, univerRef, fillDataWithUniverAPI);
    return (<div data-escape-dialog className="fixed inset-0 bg-black/50 z-[125] flex items-center justify-center">
      <div className="bg-white rounded-2xl shadow-xl w-[95vw] h-[90vh] overflow-hidden flex flex-col relative">
        {templateError && <p role="alert" className="bg-rose-50 p-3 text-sm text-rose-700">{templateError}</p>}
        {/* 头部 */}
        <div className="p-4 border-b border-zinc-200 flex justify-between items-center">
          <div className="flex items-center gap-4">
            <h3 className="text-xl font-bold text-zinc-900">{title}</h3>
            {order?.order_number && (<span className="text-sm text-zinc-500">订单号: {order.order_number}</span>)}
          </div>
          <div className="flex items-center gap-3">
            {/* 导出按钮 - 仅preview模式 */}
            {mode === 'preview' && (<button onClick={handleExportExcel} className="bg-blue-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-blue-700 transition-colors flex items-center gap-2">
                <Download className="w-4 h-4"/>
                导出Excel
              </button>)}
            {mode === 'config' && (<button onClick={handleSaveTemplate} className="bg-emerald-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-emerald-700 transition-colors">
                保存模板
              </button>)}
            <button data-escape-close onClick={onClose} className="text-zinc-400 hover:text-zinc-600 p-2">
              <X className="w-5 h-5"/>
            </button>
          </div>
        </div>

        {/* 内容区域 */}
        <div className="flex-1 flex overflow-hidden" style={{ position: 'relative' }}>
          {/* 左边：UniverSheet模板 */}
          <div className="flex-1 overflow-hidden" style={{ position: 'relative', minHeight: 0 }}>
            {sheetData ? (<UniverSheet ref={univerRef} data={sheetData} height="100%"/>) : (<div className="flex items-center justify-center h-full text-zinc-400">
                模板数据加载中...
              </div>)}
          </div>

          {/* 右边：字段面板（仅config模式显示） */}
          {mode === 'config' && (<div className="w-72 border-l border-zinc-200 bg-zinc-50 overflow-y-auto overflow-x-visible">
            <div className="p-4 min-w-[260px]">

              {/* 订单字段 */}
              <div className="mb-4">
                <h4 className="text-xs font-bold text-zinc-700 mb-2 flex items-center gap-1">
                  订单字段
                  <span className="relative group">
                    <Info className="w-3 h-3 text-blue-500 cursor-help"/>
                    <span className="absolute left-0 top-4 w-[180px] p-2 bg-blue-50 text-blue-700 text-xs rounded-lg border border-blue-200 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-50 pointer-events-none shadow-sm">
                      <span className="absolute -top-1 left-0 w-2 h-2 bg-blue-50 border-l border-t border-blue-200 rotate-45"></span>
                      点击单元格后点击字段填充，或双击进入编辑后在光标位置插入
                    </span>
                  </span>
                </h4>
                <div className="space-y-1">
                  <PrintTemplateFieldButtons fields={orderFields} onSelect={handleFieldClick} tone="order"/>
                </div>
              </div>

              {/* 其他字段 */}
              <div className="mb-4">
                <h4 className="text-xs font-bold text-zinc-700 mb-2 flex items-center gap-1">
                  其他字段
                  <span className="text-[10px] text-purple-600">(计算字段)</span>
                </h4>
                <div className="space-y-1">
                  <PrintTemplateFieldButtons fields={otherFields} onSelect={handleFieldClick} tone="other"/>
                </div>
              </div>

              {/* 零件字段 */}
              <div>
                <h4 className="text-xs font-bold text-zinc-700 mb-2 flex items-center gap-1">
                  零件字段
                  <span className="relative group">
                    <Info className="w-3 h-3 text-amber-500 cursor-help"/>
                    <span className="absolute left-0 top-4 w-[180px] p-2 bg-amber-50 text-amber-700 text-xs rounded-lg border border-amber-200 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-50 pointer-events-none shadow-sm">
                      <span className="absolute -top-1 left-0 w-2 h-2 bg-amber-50 border-l border-t border-amber-200 rotate-45"></span>
                      零件字段以 {'{{#字段名}}'} 格式标记，包含零件字段的行将作为零件模板行，预览时会根据零件数量自动复制多行
                    </span>
                  </span>
                  <span className="text-[10px] text-amber-600">(列表变量)</span>
                </h4>
                <div className="space-y-1">
                  <PrintTemplateFieldButtons fields={itemFields} onSelect={handleFieldClick} tone="item"/>
                </div>
              </div>

              {/* 零件数据预览 */}
              {order && (<div className="mt-6 pt-4 border-t border-zinc-200">
                <h4 className="text-sm font-bold text-zinc-700 mb-2">当前订单零件 ({order.items?.length || 0})</h4>
                <div className="space-y-2">
                  {(order.items || []).slice(0, 5).map((item, idx) => (<div key={idx} className="p-2 bg-white rounded-lg border border-zinc-200 text-xs">
                      <div className="font-medium text-zinc-900">{item.part_name}</div>
                      <div className="text-zinc-500 mt-1">
                        数量: {item.quantity} | 单价: {formatDisplayAmount(item.unit_price)}
                      </div>
                    </div>))}
                  {(order.items?.length || 0) > 5 && (<div className="text-xs text-zinc-400 text-center">
                      还有 {(order.items?.length || 0) - 5} 个零件...
                    </div>)}
                </div>
              </div>)}
            </div>
          </div>)}
        </div>
      </div>

      {/* Toast提示 */}
      {toast && (<div data-escape-dialog className={`fixed inset-0 z-[200] flex items-center justify-center bg-black/30 animate-in fade-in duration-200`}>
          <div className={`px-8 py-4 rounded-xl shadow-xl flex items-center gap-3 animate-in zoom-in duration-300 ${toast.type === 'success'
                ? 'bg-emerald-500 text-white'
                : 'bg-red-500 text-white'}`}>
            {toast.type === 'success' ? (<svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7"/>
              </svg>) : (<svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12"/>
              </svg>)}
            <span className="font-medium text-lg">{toast.message}</span><button type="button" data-escape-close aria-label="关闭提示" className="ml-2 text-xl" onClick={()=>setToast(null)}>×</button>
          </div>
        </div>)}
    </div>);
};
export default DeliveryPreviewModal;
