/**
 * Univer 数据结构类型定义
 * 用于解析 Univer workbook snapshot 数据
 */

/**
 * Univer 水平对齐方式枚举
 */
export enum UniverHorizontalAlignment {
  LEFT = 1,
  CENTER = 2,
  RIGHT = 3,
}

/**
 * Univer 垂直对齐方式枚举
 */
export enum UniverVerticalAlignment {
  TOP = 0,
  MIDDLE = 1,
  BOTTOM = 2,
}

/**
 * Univer 边框样式枚举
 */
export enum UniverBorderStyle {
  NONE = 0,
  THIN = 1,
  HAIR = 2,
  DOTTED = 3,
  DASHED = 4,
  DASH_DOT = 5,
  DASH_DOT_DOT = 6,
  DOUBLE = 7,
  MEDIUM = 8,
  MEDIUM_DASHED = 9,
  MEDIUM_DASH_DOT = 10,
  MEDIUM_DASH_DOT_DOT = 11,
  SLANT_DASH_DOT = 12,
  THICK = 13,
}

/**
 * Univer BooleanNumber 枚举 (0=false, 1=true)
 */
export enum UniverBooleanNumber {
  FALSE = 0,
  TRUE = 1,
}

/**
 * Univer 文字旋转
 */
export interface ITextRotation {
  /** 旋转角度 */
  a?: number;
  /** 是否垂直文字 (0/1) */
  v?: UniverBooleanNumber;
}

/**
 * Univer 颜色结构
 */
export interface IUniverColor {
  /** RGB颜色值，格式: #RRGGBB */
  rgb?: string;
  /** 主题色索引 */
  th?: number;
}

/**
 * Univer 单边边框
 */
export interface ISideBorder {
  /** 边框样式 */
  s?: UniverBorderStyle;
  /** 边框颜色 */
  cl?: IUniverColor;
}

/**
 * Univer 边框结构
 */
export interface IBorders {
  /** 上边框 */
  t?: ISideBorder;
  /** 右边框 */
  r?: ISideBorder;
  /** 下边框 */
  b?: ISideBorder;
  /** 左边框 */
  l?: ISideBorder;
}

/**
 * Univer 下划线/删除线结构
 */
export interface ITextDecoration {
  /** 是否启用 (0/1) */
  s?: UniverBooleanNumber;
}

/**
 * Univer 自动换行策略枚举 (WrapStrategy)
 */
export enum UniverWrapStrategy {
  /** 不换行 */
  NONE = 0,
  /** 溢出 - 文本延伸到相邻空白单元格 */
  OVERFLOW = 1,
  /** 裁剪 - 文本在单元格边界截断 */
  CLIP = 2,
  /** 自动换行 */
  WRAP = 3,
}

/**
 * Univer 数字格式
 */
export interface INumberFormat {
  /** 格式模式，如 "General", "0.00", "#,##0.00" 等 */
  pattern?: string;
}

/**
 * Univer 单元格样式数据 (IStyleData)
 * 用于 styles 数组或单元格内联样式
 */
export interface IUniverStyleData {
  /** 字体名称，如 "等线", "微软雅黑", "Arial" 等 */
  ff?: string;
  /** 字体大小 (pt) */
  fs?: number;
  /** 斜体 (BooleanNumber: 0/1) */
  it?: UniverBooleanNumber;
  /** 粗体 (BooleanNumber: 0/1) */
  bl?: UniverBooleanNumber;
  /** 下划线 */
  ul?: ITextDecoration;
  /** 删除线 */
  st?: ITextDecoration;
  /** 字体颜色 */
  cl?: IUniverColor;
  /** 背景色 */
  bg?: IUniverColor;
  /** 水平对齐 (UniverHorizontalAlignment) */
  ht?: UniverHorizontalAlignment;
  /** 垂直对齐 (UniverVerticalAlignment) */
  vt?: UniverVerticalAlignment;
  /** 文字旋转 */
  tr?: ITextRotation;
  /** 自动换行 (WrapStrategy: 0=不换行, 1=换行) */
  tb?: UniverBooleanNumber;
  /** 边框 */
  bd?: IBorders;
  /** 数字格式 */
  n?: INumberFormat;
}

/**
 * Univer 单元格类型
 */
export enum UniverCellType {
  /** 普通值 */
  VALUE = 0,
  /** 富文本 */
  RICH_TEXT = 1,
  /** 公式 */
  FORMULA = 2,
  /** 数组公式 */
  ARRAY_FORMULA = 3,
}

/**
 * Univer 富文本片段
 */
export interface IRichTextSegment {
  /** 文本内容 */
  v?: string;
  /** 该片段的样式（覆盖单元格样式） */
  s?: IUniverStyleData;
}

/**
 * Univer 富文本内容结构
 */
export interface IRichTextContent {
  /** 富文本片段数组 */
  s?: IRichTextSegment[];
}

/**
 * Univer 单元格数据结构 (ICellData)
 */
export interface IUniverCellData {
  /** 样式引用ID，指向 styles 数组中的索引 */
  s?: string | number;

  // ============ 内联样式属性（可覆盖 s 引用的样式） ============
  /** 字体名称 */
  ff?: string;
  /** 字体大小 */
  fs?: number;
  /** 粗体 (BooleanNumber: 0/1) */
  bl?: UniverBooleanNumber;
  /** 斜体 (BooleanNumber: 0/1) */
  it?: UniverBooleanNumber;
  /** 下划线 */
  ul?: ITextDecoration;
  /** 删除线 */
  st?: ITextDecoration;
  /** 水平对齐 */
  ht?: UniverHorizontalAlignment;
  /** 垂直对齐 */
  vt?: UniverVerticalAlignment;
  /** 字体颜色 */
  cl?: IUniverColor;
  /** 背景色 */
  bg?: IUniverColor;
  /** 边框 */
  bd?: IBorders;
  /** 文字旋转 */
  tr?: ITextRotation;
  /** 自动换行 */
  tb?: UniverBooleanNumber;

  // ============ 单元格数据属性 ============
  /** 单元格类型 */
  t?: UniverCellType;
  /** 单元格值 */
  v?: string | number | boolean | null;
  /** 公式字符串 */
  f?: string;
  /** 富文本内容 */
  ct?: IRichTextContent;
  /** 格式化后的显示文本 */
  p?: string;
  /** 格式ID */
  si?: string | number | null;
}

/**
 * Univer 合并单元格数据
 */
export interface IUniverMergeData {
  /** 起始行 (0-based) */
  startRow: number;
  /** 起始列 (0-based) */
  startColumn: number;
  /** 结束行 (0-based) */
  endRow: number;
  /** 结束列 (0-based) */
  endColumn: number;
}

/**
 * Univer 行数据
 */
export interface IUniverRowData {
  /** 行高 */
  h?: number;
  /** 是否隐藏 */
  hd?: UniverBooleanNumber;
}

/**
 * Univer 列数据
 */
export interface IUniverColumnData {
  /** 列宽 (像素) */
  w?: number;
  /** 是否隐藏 */
  hd?: UniverBooleanNumber;
}

/**
 * Univer 工作表数据 (ISheetData)
 */
export interface IUniverSheetData {
  /** 工作表ID */
  id: string;
  /** 工作表名称 */
  name: string;
  /** 单元格数据，格式: { [rowKey]: { [colKey]: ICellData } } */
  cellData?: Record<string, Record<string, IUniverCellData>>;
  /** 合并单元格数据 */
  mergeData?: IUniverMergeData[];
  /** 或存储在 config.merge */
  config?: {
    merge?: IUniverMergeData[] | Record<string, IUniverMergeData>;
  };
  /** 行数据，格式: { [rowKey]: IRowData } */
  rowData?: Record<string, IUniverRowData>;
  rows?: Record<string, IUniverRowData>;
  /** 列数据，格式: { [colKey]: IColumnData } */
  columnData?: Record<string, IUniverColumnData>;
  columns?: Record<string, IUniverColumnData>;
  /** 行数 */
  rowCount?: number;
  /** 列数 */
  columnCount?: number;
  /** 默认列宽 */
  defaultColumnWidth?: number;
  /** 默认行高 */
  defaultRowHeight?: number;
  /** 工作表级别样式数组 */
  styles?: IUniverStyleData[];
}

/**
 * Univer Workbook Snapshot 数据结构
 */
export interface IUniverWorkbookSnapshot {
  /** Workbook ID */
  id?: string;
  /** 工作表顺序 (sheetId 数组) */
  sheetOrder: string[];
  /** 工作表数据，格式: { [sheetId]: ISheetData } */
  sheets: Record<string, IUniverSheetData>;
  /** Workbook 级别样式数组 */
  styles?: IUniverStyleData[];
  /** App 版本 */
  appVersion?: string;
  /** Locale */
  locale?: string;
  /** 命名空间 */
  name?: string;
}

// ============== 工具函数 ==============

/**
 * 水平对齐数字转字符串
 */
export function univerHorizontalToString(ht: UniverHorizontalAlignment | undefined): string | undefined {
  if (ht === undefined) return undefined;
  const map: Record<UniverHorizontalAlignment, string> = {
    [UniverHorizontalAlignment.LEFT]: 'left',
    [UniverHorizontalAlignment.CENTER]: 'center',
    [UniverHorizontalAlignment.RIGHT]: 'right',
  };
  return map[ht];
}

/**
 * 垂直对齐数字转字符串
 */
export function univerVerticalToString(vt: UniverVerticalAlignment | undefined): string | undefined {
  if (vt === undefined) return undefined;
  const map: Record<UniverVerticalAlignment, string> = {
    [UniverVerticalAlignment.TOP]: 'top',
    [UniverVerticalAlignment.MIDDLE]: 'middle',
    [UniverVerticalAlignment.BOTTOM]: 'bottom',
  };
  return map[vt];
}

/**
 * 边框样式数字转字符串
 */
export function univerBorderStyleToString(s: UniverBorderStyle | undefined): string | undefined {
  if (s === undefined) return undefined;
  const map: Record<UniverBorderStyle, string> = {
    [UniverBorderStyle.NONE]: 'none',
    [UniverBorderStyle.THIN]: 'thin',
    [UniverBorderStyle.HAIR]: 'hair',
    [UniverBorderStyle.DOTTED]: 'dotted',
    [UniverBorderStyle.DASHED]: 'dashed',
    [UniverBorderStyle.DASH_DOT]: 'dashDot',
    [UniverBorderStyle.DASH_DOT_DOT]: 'dashDotDot',
    [UniverBorderStyle.DOUBLE]: 'double',
    [UniverBorderStyle.MEDIUM]: 'medium',
    [UniverBorderStyle.MEDIUM_DASHED]: 'mediumDashed',
    [UniverBorderStyle.MEDIUM_DASH_DOT]: 'mediumDashDot',
    [UniverBorderStyle.MEDIUM_DASH_DOT_DOT]: 'mediumDashDotDot',
    [UniverBorderStyle.SLANT_DASH_DOT]: 'slantDashDot',
    [UniverBorderStyle.THICK]: 'thick',
  };
  return map[s];
}

/**
 * 颜色转 ARGB 格式
 */
export function univerColorToArgb(color: IUniverColor | string | undefined): string | undefined {
  if (!color) return undefined;
  let rgb = typeof color === 'string' ? color : color.rgb;
  if (!rgb) return undefined;

  // 去掉 # 号
  rgb = rgb.replace('#', '');
  // 处理 rgb(r, g, b) 格式
  if (rgb.startsWith('rgb')) {
    const match = rgb.match(/\d+/g);
    if (match && match.length === 3) {
      rgb = match.map(p => parseInt(p).toString(16).padStart(2, '0')).join('');
    }
  }
  if (!rgb || rgb.length < 6) return undefined;
  // 添加 Alpha 通道 (FF = 255 = 完全透明度，即完全不透明)
  return 'FF' + rgb.toUpperCase();
}