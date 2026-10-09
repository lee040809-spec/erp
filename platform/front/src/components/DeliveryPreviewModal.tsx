import { formatAmount as formatDisplayAmount } from '../utils/money';
import React, { useEffect, useState, useRef } from 'react';
import { X, Info, Plus, Download } from 'lucide-react';
import { Order } from '../types';
import { authFetch } from './shared';
import UniverSheet from './UniverSheet';
import pako from 'pako';
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
// 格式化日期
const formatDate = (date: any): string => {
    if (!date)
        return '';
    if (typeof date === 'string')
        return date.split('T')[0];
    if (date instanceof Date)
        return date.toISOString().split('T')[0];
    return String(date);
};
// 格式化金额
const formatAmount = (amount: any): string => {
    if (!amount)
        return '0';
    return Number(amount).toFixed(2);
};
// 获取字段值
const getFieldValue = (obj: any, key: string): string => {
    const parts = key.split('.');
    let value = obj;
    for (const part of parts) {
        if (value && typeof value === 'object') {
            value = value[part];
        }
        else {
            return '';
        }
    }
    if (value === null || value === undefined)
        return '';
    if (typeof value === 'object' && value instanceof Date)
        return formatDate(value);
    if (typeof value === 'number' && key.includes('amount') || key.includes('price'))
        return formatAmount(value);
    return String(value);
};
// 检查单元格是否包含零件字段占位符
const hasItemPlaceholder = (cellValue: string): boolean => {
    return !!cellValue && cellValue.includes('{{#items.');
};
// 检查单元格是否包含订单字段占位符
const hasOrderPlaceholder = (cellValue: string): boolean => {
    return !!cellValue && cellValue.includes('{{order.');
};
// 替换订单占位符（包含other字段）
const replaceOrderPlaceholders = (cellValue: string, orderData: any): string => {
    if (!cellValue)
        return cellValue;
    let result = cellValue;
    // 匹配 {{order.xxx}} 格式
    const orderRegex = /\{\{order\.([^}]+)\}\}/g;
    let match;
    while ((match = orderRegex.exec(cellValue)) !== null) {
        const fieldKey = match[1];
        const value = getFieldValue(orderData, fieldKey);
        result = result.replace(match[0], value);
    }
    // 匹配 {{other.xxx}} 格式（计算字段）
    const otherRegex = /\{\{other\.([^}]+)\}\}/g;
    while ((match = otherRegex.exec(result)) !== null) {
        const fieldKey = match[1];
        const value = getOtherFieldValue(orderData, fieldKey);
        result = result.replace(match[0], value);
    }
    return result;
};
// 获取其他字段值（从后端返回的other对象获取）
const getOtherFieldValue = (orderData: any, fieldKey: string): string => {
    const other = orderData.other || {};
    // 转换为小写处理（支持大小写不敏感）
    const key = fieldKey.toLowerCase();
    switch (key) {
        case 'small_total':
            return other.small_total || '0.00';
        case 'big_total':
            return other.big_total || '零元整';
        case 'total_quantity':
            return String(other.total_quantity || 0);
        case 'delivery_number':
            return other.delivery_number || '';
        default:
            // 直接尝试从 other 对象获取
            return other[key] || '';
    }
};
// 替换零件占位符
const replaceItemPlaceholders = (cellValue: string, itemData: any): string => {
    if (!cellValue)
        return cellValue;
    let result = cellValue;
    // 匹配 {{#items.xxx}} 格式
    const regex = /\{\{#items\.([^}]+)\}\}/g;
    let match;
    while ((match = regex.exec(cellValue)) !== null) {
        const fieldKey = match[1];
        const value = getFieldValue(itemData, fieldKey);
        result = result.replace(match[0], value);
    }
    return result;
};
// 填充模板数据（订单字段 + 零件逐行填充）
const fillTemplateData = (workbookData: any, orderData: any): any => {
    // 深拷贝 workbookData
    const newData = JSON.parse(JSON.stringify(workbookData));
    // 获取第一个工作表
    const sheetId = newData.sheetOrder?.[0];
    if (!sheetId)
        return newData;
    const sheet = newData.sheets?.[sheetId];
    if (!sheet || !sheet.cellData)
        return newData;
    const cellData = sheet.cellData;
    const items = orderData.item || orderData.items || [];
    // 1. 找到包含零件字段的模板行
    let itemTemplateRow = -1;
    for (const rowKey in cellData) {
        const row = parseInt(rowKey);
        for (const colKey in cellData[rowKey]) {
            const cell = cellData[rowKey][colKey];
            if (cell && cell.v && hasItemPlaceholder(String(cell.v))) {
                itemTemplateRow = row;
                break;
            }
        }
        if (itemTemplateRow >= 0)
            break;
    }
    // 2. 处理所有单元格
    if (itemTemplateRow >= 0 && items.length > 0) {
        // 有零件模板行：复制行并填充数据
        const templateRowData = cellData[itemTemplateRow] || {};
        // 删除原模板行（后面会插入填充后的行）
        delete cellData[itemTemplateRow];
        // 为每个零件插入一行
        items.forEach((item: any, idx: number) => {
            const newRow = itemTemplateRow + idx;
            cellData[newRow] = {};
            for (const colKey in templateRowData) {
                const templateCell = templateRowData[colKey];
                const cellValue = templateCell?.v || '';
                // 替换零件占位符
                let newValue = replaceItemPlaceholders(cellValue, item);
                // 也替换订单占位符（零件行可能也有订单字段）
                newValue = replaceOrderPlaceholders(newValue, orderData);
                cellData[newRow][colKey] = {
                    ...templateCell,
                    v: newValue
                };
            }
        });
        // 处理其他行的订单占位符
        for (const rowKey in cellData) {
            const row = parseInt(rowKey);
            // 跳过零件行
            if (row >= itemTemplateRow && row < itemTemplateRow + items.length)
                continue;
            for (const colKey in cellData[rowKey]) {
                const cell = cellData[rowKey][colKey];
                if (cell && cell.v && hasOrderPlaceholder(String(cell.v))) {
                    cellData[rowKey][colKey] = {
                        ...cell,
                        v: replaceOrderPlaceholders(String(cell.v), orderData)
                    };
                }
            }
        }
    }
    else {
        // 无零件模板行：只替换订单占位符
        for (const rowKey in cellData) {
            for (const colKey in cellData[rowKey]) {
                const cell = cellData[rowKey][colKey];
                if (cell && cell.v) {
                    cellData[rowKey][colKey] = {
                        ...cell,
                        v: replaceOrderPlaceholders(String(cell.v), orderData)
                    };
                }
            }
        }
    }
    return newData;
};
const DeliveryPreviewModal: React.FC<DeliveryPreviewModalProps> = ({ order, template, mode = 'config', title = '送货单', orderData, onClose }) => {
    const [toast, setToast] = useState<{
        message: string;
        type: 'success' | 'error';
    } | null>(null);
    const univerRef = useRef<any>(null);
    const [sheetData, setSheetData] = useState<any>(null);
    // 加载模板数据
    useEffect(() => {
        if (!template?.template)
            return;
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
            }
            else if (templateStr.startsWith('GZIP:')) {
                // 旧格式：gzip压缩数据
                const base64 = templateStr.substring(5);
                const binary = atob(base64);
                const bytes = new Uint8Array(binary.length);
                for (let i = 0; i < binary.length; i++) {
                    bytes[i] = binary.charCodeAt(i);
                }
                const decompressed = pako.inflate(bytes, { to: 'string' });
                workbookData = JSON.parse(decompressed);
            }
            else {
                // 直接JSON或其他格式
                workbookData = JSON.parse(templateStr);
            }
            // 只设置原始模板数据，不预处理
            setSheetData(workbookData);
        }
        catch (err) {
            // 解析失败，静默处理
        }
    }, [template?.template]);
    // preview 模式下，使用 Univer API 填充数据
    useEffect(() => {
        if (mode !== 'preview' || !orderData || !univerRef.current)
            return;
        // 等待 Univer ready
        const checkReady = () => {
            if (univerRef.current?.isReady?.()) {
                fillDataWithUniverAPI();
            }
            else {
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
                let newValue = cellValue;
                const itemRegex = /\{\{#items\.([^}]+)\}\}/g;
                let match;
                while ((match = itemRegex.exec(cellValue)) !== null) {
                    const fieldKey = match[1];
                    const value = getFieldValue(item, fieldKey);
                    newValue = newValue.replace(match[0], value);
                }
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
            const filename = `送货单_${order.order_number || '未知'}_${new Date().toISOString().slice(0, 10)}.xlsx`;
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
            const workbookJson = JSON.stringify(workbookData);
            const compressed = pako.gzip(workbookJson);
            // 转为 base64，使用 UNIVER: 前缀
            const base64 = btoa(String.fromCharCode(...compressed));
            const compressedData = 'UNIVER:' + base64;
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
    return (<div className="fixed inset-0 bg-black/50 z-[125] flex items-center justify-center">
      <div className="bg-white rounded-2xl shadow-xl w-[95vw] h-[90vh] overflow-hidden flex flex-col relative">
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
            <button onClick={onClose} className="text-zinc-400 hover:text-zinc-600 p-2">
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
                  {orderFields.map(field => (<button key={field.key} onClick={() => handleFieldClick(field)} className="inline-flex items-center gap-2 px-2 py-1.5 bg-white rounded-lg border border-zinc-200 hover:bg-zinc-50 hover:border-zinc-300 transition-colors whitespace-nowrap w-[280px]">
                      <Plus className="w-3 h-3 text-zinc-400"/>
                      <span className="text-xs text-zinc-700">{field.label}</span>
                      <code className="text-[10px] text-zinc-400 ml-2">{"{{" + field.key + "}}"}</code>
                    </button>))}
                </div>
              </div>

              {/* 其他字段 */}
              <div className="mb-4">
                <h4 className="text-xs font-bold text-zinc-700 mb-2 flex items-center gap-1">
                  其他字段
                  <span className="text-[10px] text-purple-600">(计算字段)</span>
                </h4>
                <div className="space-y-1">
                  {otherFields.map(field => (<button key={field.key} onClick={() => handleFieldClick(field)} className="inline-flex items-center gap-2 px-2 py-1.5 bg-white rounded-lg border border-purple-200 hover:bg-purple-50 hover:border-purple-300 transition-colors whitespace-nowrap w-[280px]">
                      <Plus className="w-3 h-3 text-purple-400"/>
                      <span className="text-xs text-zinc-700">{field.label}</span>
                      <code className="text-[10px] text-purple-500 ml-2">{"{{" + field.key + "}}"}</code>
                    </button>))}
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
                  {itemFields.map(field => (<button key={field.key} onClick={() => handleFieldClick(field)} className="inline-flex items-center gap-2 px-2 py-1.5 bg-white rounded-lg border border-amber-200 hover:bg-amber-50 hover:border-amber-300 transition-colors whitespace-nowrap w-[280px]">
                      <Plus className="w-3 h-3 text-amber-400"/>
                      <span className="text-xs text-zinc-700">{field.label}</span>
                      <code className="text-[10px] text-amber-500 ml-2">{"{{#" + field.key + "}}"}</code>
                    </button>))}
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
      {toast && (<div className={`fixed inset-0 z-[200] flex items-center justify-center bg-black/30 animate-in fade-in duration-200`}>
          <div className={`px-8 py-4 rounded-xl shadow-xl flex items-center gap-3 animate-in zoom-in duration-300 ${toast.type === 'success'
                ? 'bg-emerald-500 text-white'
                : 'bg-red-500 text-white'}`}>
            {toast.type === 'success' ? (<svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7"/>
              </svg>) : (<svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12"/>
              </svg>)}
            <span className="font-medium text-lg">{toast.message}</span>
          </div>
        </div>)}
    </div>);
};
export default DeliveryPreviewModal;
