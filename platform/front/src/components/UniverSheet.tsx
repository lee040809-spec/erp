import React, { useEffect, useRef, forwardRef, useImperativeHandle, useState } from 'react';
import { UniverSheetsCorePreset } from '@univerjs/preset-sheets-core';
import sheetsCoreZhCN from '@univerjs/preset-sheets-core/locales/zh-CN';
import { createUniver, LocaleType } from '@univerjs/presets';

// 导入Univer样式
import '@univerjs/preset-sheets-core/lib/index.css';

interface UniverSheetProps {
  data?: any;
  height?: string;
}

const UniverSheet = forwardRef((props: UniverSheetProps, ref: React.Ref<any>) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const univerAPIRef = useRef<any>(null);
  const univerRef = useRef<any>(null);
  const workbookRef = useRef<any>(null);
  const injectorRef = useRef<any>(null);
  const [isReady, setIsReady] = useState(false);
  const [mounted, setMounted] = useState(false);
  const dataRef = useRef<any>(props.data);

  useEffect(() => {
    dataRef.current = props.data;
  }, [props.data]);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (!mounted || !containerRef.current) return;

    try {
      const { univer, univerAPI } = createUniver({
        locale: LocaleType.ZH_CN,
        locales: {
          [LocaleType.ZH_CN]: sheetsCoreZhCN,
        },
        presets: [
          UniverSheetsCorePreset({
            container: containerRef.current,
          }),
        ],
      });

      univerAPIRef.current = univerAPI;
      univerRef.current = univer;
      injectorRef.current = univer.__getInjector?.();

      const workbookData = dataRef.current || {
        id: 'workbook-01',
        sheetOrder: ['sheet-01'],
        sheets: {
          'sheet-01': {
            id: 'sheet-01',
            name: 'Sheet1',
            cellData: {},
            rowCount: 1000,
            columnCount: 26,
            defaultColumnWidth: 93,
            defaultRowHeight: 27,
          },
        },
        appVersion: '1.0.0',
      };

      workbookRef.current = univerAPI.createWorkbook(workbookData);
      setIsReady(true);

    } catch (err) {
      // Univer初始化失败
    }

    return () => {
      if (univerAPIRef.current) {
        try {
          univerAPIRef.current.dispose();
        } catch (e) {
          // Dispose失败
        }
      }
      univerAPIRef.current = null;
      univerRef.current = null;
      workbookRef.current = null;
      injectorRef.current = null;
      setIsReady(false);
    };
  }, [mounted]);

  useImperativeHandle(ref, () => ({
    getData: () => workbookRef.current?.save?.() || null,
    getRawData: () => workbookRef.current?.save?.() || null,
    setCellValue: (row: number, col: number, value: string) => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      if (sheet) {
        sheet.getRange(row, col).setValue(value);
      }
    },
    getCellValue: (row: number, col: number): string => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      if (!sheet) return '';
      const range = sheet.getRange(row, col);
      return range.getValue?.() || '';
    },
    getSelection: () => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      if (!sheet) return null;

      // 获取当前激活的范围
      const activeRange = sheet.getActiveRange?.();
      if (!activeRange) return null;

      // 返回合并单元格的起始位置
      return { row: activeRange.getRow?.(), col: activeRange.getColumn?.() };
    },
    isReady: () => isReady,
    isEditing: () => univerAPIRef.current?.getActiveWorkbook?.()?.isCellEditing?.() || false,
    insertTextAtCursor: (text: string): boolean => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      if (!sheet) return false;

      // 获取当前激活的范围（合并单元格也适用）
      const activeRange = sheet.getActiveRange?.();
      if (!activeRange) return false;

      // 获取当前值并追加
      const currentValue = activeRange.getValue?.() || '';
      activeRange.setValue(currentValue + text);
      return true;
    },
    // 插入行（在指定行上方插入 count 行）
    insertRowsAbove: (row: number, count: number): boolean => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      if (!sheet) return false;
      try {
        sheet.insertRows?.(row, count);
        return true;
      } catch (e) {
        return false;
      }
    },
    // 插入行（在指定行下方插入 count 行）
    insertRowsBelow: (row: number, count: number): boolean => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      if (!sheet) return false;
      try {
        // 在 row+1 位置插入 count 行
        sheet.insertRows?.(row + 1, count);
        return true;
      } catch (e) {
        return false;
      }
    },
    // 复制行数据（从 sourceRow 复制到 targetRow）
    copyRow: (sourceRow: number, targetRow: number): boolean => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      if (!sheet) return false;
      try {
        // 复制整行
        const sourceRange = sheet.getRange(sourceRow, 0, 1, sheet.getColumnCount?.() || 26);
        const targetRange = sheet.getRange(targetRow, 0, 1, sheet.getColumnCount?.() || 26);
        sourceRange.copyTo?.(targetRange);
        return true;
      } catch (e) {
        return false;
      }
    },
    // 获取行数
    getRowCount: (): number => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      return sheet?.getRowCount?.() || 1000;
    },
    // 获取列数
    getColumnCount: (): number => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      return sheet?.getColumnCount?.() || 26;
    },
    // 查找包含特定文本的单元格
    findCellWithText: (text: string): { row: number; col: number } | null => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      if (!sheet) return null;
      const rowCount = sheet.getRowCount?.() || 1000;
      const colCount = sheet.getColumnCount?.() || 26;

      // Univer API 使用 0-based 索引
      for (let r = 0; r < rowCount; r++) {
        for (let c = 0; c < colCount; c++) {
          try {
            const value = sheet.getRange(r, c).getValue?.() || '';
            if (value.includes(text)) {
              return { row: r, col: c };
            }
          } catch (e) {
            // 跳过超出范围的单元格
          }
        }
      }
      return null;
    },
    // 替换单元格中的占位符
    replacePlaceholders: (row: number, col: number, replacements: Record<string, string>): boolean => {
      const sheet = univerAPIRef.current?.getActiveWorkbook?.()?.getActiveSheet?.();
      if (!sheet) return false;
      try {
        const range = sheet.getRange(row, col);
        let value = range.getValue?.() || '';
        for (const [key, val] of Object.entries(replacements)) {
          value = value.replace(key, val);
        }
        range.setValue(value);
        return true;
      } catch (e) {
        return false;
      }
    },
    // 导出Excel快照数据
    getSnapshot: () => {
      const workbook = univerAPIRef.current?.getActiveWorkbook?.();
      if (!workbook) return null;
      return workbook.save?.() || null;
    },
  }));

  if (!mounted) {
    return <div style={{ width: '100%', height: props.height || '100%', minHeight: '400px', display: 'flex', alignItems: 'center', justifyContent: 'center', backgroundColor: '#f5f5f5' }}>加载中...</div>;
  }

  return <div ref={containerRef} style={{ width: '100%', height: props.height || '100%', minHeight: '400px' }} />;
});

UniverSheet.displayName = 'UniverSheet';
export default UniverSheet;