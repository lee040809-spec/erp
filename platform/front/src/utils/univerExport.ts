import ExcelJS from 'exceljs';
import {
  pixelColumnWidthToExcel,
  pixelRowHeightToExcel,
  univerHorizontalAlignToExcel,
  univerVerticalAlignToExcel,
  univerWrapStrategyToExcel,
  univerBorderStyleToExcel,
  univerColorToArgb,
} from './excelConvert';

// 动态导入，兼容 Node.js 和浏览器环境
let saveAs: ((blob: Blob, filename: string) => void) | null = null;
let fsModule: typeof import('fs') | null = null;

// 检测环境并加载对应的保存模块
async function initSaveModule() {
  if (typeof window !== 'undefined') {
    const fileSaver = await import('file-saver');
    saveAs = fileSaver.saveAs;
  } else {
    fsModule = await import('fs');
  }
}

/**
 * 转换 Univer 单元格样式为 ExcelJS 样式
 */
function convertUniverStyle(styleData: any): any {
  if (!styleData) return {};

  const style: any = {};

  // ============ 字体样式 ============
  const font: any = {};

  if (styleData.ff) font.name = styleData.ff;
  if (styleData.fs) font.size = styleData.fs;
  if (styleData.bl === 1) font.bold = true;
  if (styleData.it === 1) font.italic = true;
  if (styleData.ul?.s === 1) font.underline = true;
  if (styleData.st?.s === 1) font.strike = true;

  const fontColor = univerColorToArgb(styleData.cl);
  if (fontColor) font.color = { argb: fontColor };

  if (Object.keys(font).length > 0) style.font = font;

  // ============ 背景色 ============
  const bgColor = univerColorToArgb(styleData.bg);
  if (bgColor) {
    style.fill = {
      type: 'pattern',
      pattern: 'solid',
      fgColor: { argb: bgColor }
    };
  }

  // ============ 对齐样式 ============
  const alignment: any = {};

  const horizontal = univerHorizontalAlignToExcel(styleData.ht);
  if (horizontal) alignment.horizontal = horizontal;

  const vertical = univerVerticalAlignToExcel(styleData.vt);
  if (vertical) alignment.vertical = vertical;

  if (styleData.tr) {
    if (styleData.tr.v === 1) {
      alignment.textRotation = 'vertical';
    } else if (styleData.tr.a) {
      alignment.textRotation = styleData.tr.a;
    }
  }

  const wrapText = univerWrapStrategyToExcel(styleData.tb);
  if (wrapText !== undefined) alignment.wrapText = wrapText;

  if (Object.keys(alignment).length > 0) style.alignment = alignment;

  // ============ 边框 ============
  if (styleData.bd) {
    const border: any = {};
    const sides = ['t', 'r', 'b', 'l'];
    const excelSides = ['top', 'right', 'bottom', 'left'];

    sides.forEach((side, index) => {
      const sideData = styleData.bd[side];
      if (sideData) {
        const borderColor = univerColorToArgb(sideData.cl) || 'FF000000';
        const borderStyle = univerBorderStyleToExcel(sideData.s) || 'thin';
        border[excelSides[index]] = {
          style: borderStyle,
          color: { argb: borderColor }
        };
      }
    });

    if (Object.keys(border).length > 0) style.border = border;
  }

  return style;
}

/**
 * 设置单元格样式和值
 */
function setStyleAndValue(
  cellData: Record<string, Record<string, any>>,
  worksheet: ExcelJS.Worksheet,
  stylesArray?: any
) {
  if (!cellData) return;

  for (const rowKey in cellData) {
    const row = parseInt(rowKey);
    const rowData = cellData[rowKey];

    for (const colKey in rowData) {
      const col = parseInt(colKey);
      const cell = rowData[colKey];

      if (!cell) continue;

      const excelCell = worksheet.getCell(row + 1, col + 1);

      // 设置值
      let value: any = '';
      if (cell.f) {
        value = { formula: cell.f, result: cell.v };
      } else if (cell.v !== undefined && cell.v !== null) {
        value = cell.v;
        if (!cell.v && cell.ct?.s) {
          cell.ct.s.forEach((segment: any) => {
            value += segment.v || '';
          });
        }
      }

      excelCell.value = value;

      // 设置样式
      let style: any = {};

      // 样式引用 (cell.s 指向 stylesArray 中的样式，格式如 "style_3")
      if (cell.s !== undefined && stylesArray) {
        const styleRef = typeof stylesArray === 'object' && !Array.isArray(stylesArray)
          ? stylesArray[cell.s]  // 对象格式: stylesArray["style_3"]
          : Array.isArray(stylesArray) ? stylesArray[cell.s] : null;
        if (styleRef) {
          style = convertUniverStyle(styleRef);
        }
      }

      // 单元格内联样式 (优先级更高)
      const cellInlineStyle = convertUniverStyle(cell);
      if (Object.keys(cellInlineStyle).length > 0) {
        style = { ...style, ...cellInlineStyle };
      }

      // 应用样式
      if (style.fill) excelCell.fill = style.fill;
      if (style.font) excelCell.font = style.font;
      if (style.alignment) excelCell.alignment = style.alignment;
      if (style.border) excelCell.border = style.border;
    }
  }
}

/**
 * 设置合并单元格
 */
function setMerges(mergeData: any[], worksheet: ExcelJS.Worksheet) {
  if (!mergeData || !Array.isArray(mergeData)) return;

  mergeData.forEach((merge) => {
    try {
      worksheet.mergeCells(
        merge.startRow + 1,
        merge.startColumn + 1,
        merge.endRow + 1,
        merge.endColumn + 1
      );
    } catch (e) {
      console.warn('Merge cell failed:', e);
    }
  });
}

/**
 * 设置列宽
 */
function setColumnWidths(columnData: any, worksheet: ExcelJS.Worksheet) {
  if (!columnData) return;

  for (const colKey in columnData) {
    const col = parseInt(colKey);
    const colInfo = columnData[colKey];
    if (colInfo?.w) {
      worksheet.getColumn(col + 1).width = pixelColumnWidthToExcel(colInfo.w);
    }
  }
}

/**
 * 设置行高
 */
function setRowHeights(rowData: any, worksheet: ExcelJS.Worksheet) {
  if (!rowData) return;

  for (const rowKey in rowData) {
    const row = parseInt(rowKey);
    const rowInfo = rowData[rowKey];
    if (rowInfo?.h) {
      worksheet.getRow(row + 1).height = pixelRowHeightToExcel(rowInfo.h);
    }
  }
}

/**
 * 导出 Univer Workbook 数据为 Excel 文件
 */
export async function exportUniverToExcel(
  snapshot: any,
  filename: string = 'export.xlsx'
): Promise<void> {
  const workbook = new ExcelJS.Workbook();

  const sheetOrder = snapshot.sheetOrder || [];
  const sheets = snapshot.sheets || {};

  for (const sheetId of sheetOrder) {
    const sheetData = sheets[sheetId];
    if (!sheetData) continue;

    const sheetName = sheetData.name || 'Sheet';
    const worksheet = workbook.addWorksheet(sheetName);

    const cellData = sheetData.cellData || {};
    const stylesArray = snapshot.styles || sheetData.styles || [];
    const mergeData = sheetData.mergeData || sheetData.config?.merge || [];
    const columnData = sheetData.columnData || sheetData.columns || {};
    const rowData = sheetData.rowData || sheetData.rows || {};

    setStyleAndValue(cellData, worksheet, stylesArray);

    if (Array.isArray(mergeData)) {
      setMerges(mergeData, worksheet);
    } else if (typeof mergeData === 'object') {
      const merges: any[] = [];
      for (const key in mergeData) {
        merges.push(mergeData[key]);
      }
      setMerges(merges, worksheet);
    }

    setColumnWidths(columnData, worksheet);
    setRowHeights(rowData, worksheet);
  }

  const buffer = await workbook.xlsx.writeBuffer();
  await initSaveModule();

  if (typeof window !== 'undefined' && saveAs) {
    const blob = new Blob([buffer], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    });
    saveAs(blob, filename);
  } else if (fsModule) {
    fsModule.writeFileSync(filename, Buffer.from(buffer));
  } else {
    throw new Error('无法保存文件：未找到可用的保存模块');
  }
}

export default exportUniverToExcel;