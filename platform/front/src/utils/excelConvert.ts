/**
 * Excel 与 Univer 单位转换工具
 * 参考导入库 @mertdeveci55/univer-import-export 的转换公式
 *
 * 转换公式说明：
 * - 列宽：Univer 使用像素，Excel 使用字符宽度
 * - 行高：Univer 使用像素，Excel 使用磅(pt)
 */

// ============ 列宽转换 ============

/**
 * 列宽转换系数
 * 导入公式: Excel列宽 * 7.5 = 像素列宽
 * 导出公式: 像素列宽 / 7.5 = Excel列宽
 */
export const COLUMN_WIDTH_FACTOR = 7.5;

/**
 * Excel 列宽转像素列宽 (导入时使用)
 * @param excelWidth Excel 列宽（字符宽度单位）
 * @returns 像素列宽
 */
export function excelColumnWidthToPixel(excelWidth: number): number {
  return Math.round(excelWidth * COLUMN_WIDTH_FACTOR);
}

/**
 * 像素列宽转 Excel 列宽 (导出时使用)
 * @param pixelWidth 像素列宽
 * @returns Excel 列宽（字符宽度单位）
 */
export function pixelColumnWidthToExcel(pixelWidth: number): number {
  return Math.max(pixelWidth / COLUMN_WIDTH_FACTOR, 1);
}

// ============ 行高转换 ============

/**
 * 行高转换系数
 *
 * 导入库源码分析:
 * - 导入公式: function M(e){return Math.round(e/.75)}
 *   即: Univer height = Math.round(Excel height / 0.75)
 * - 导出公式: a.height = .75 * n.h
 *   即: Excel height = Univer height * 0.75
 *
 * 结论: 导入和导出使用相同的系数 0.75，只是运算方向不同
 */
export const ROW_HEIGHT_FACTOR = 0.75;

/**
 * Excel 行高转 Univer 行高 (导入时使用)
 * 公式: Univer height = Math.round(Excel height / 0.75)
 * @param excelHeight Excel 行高（磅单位 pt）
 * @returns Univer rowData.h 值
 */
export function excelRowHeightToUniver(excelHeight: number): number {
  return Math.round(excelHeight / ROW_HEIGHT_FACTOR);
}

/**
 * Univer 行高转 Excel 行高 (导出时使用)
 * 公式: Excel height = Univer height * 0.75
 * @param univerHeight Univer rowData.h 值
 * @returns Excel 行高（磅单位 pt）
 */
export function univerRowHeightToExcel(univerHeight: number): number {
  return univerHeight * ROW_HEIGHT_FACTOR;
}

// 别名，保持向后兼容
export const pixelRowHeightToExcel = univerRowHeightToExcel;

// ============ 默认值 ============

/**
 * Univer 默认列宽（像素）
 */
export const DEFAULT_COLUMN_WIDTH_PIXEL = 93;

/**
 * Univer 默认行高（像素）
 */
export const DEFAULT_ROW_HEIGHT_PIXEL = 27;

/**
 * Excel 默认列宽（字符宽度）
 * 计算: 93 / 7.5 = 12.4
 */
export const DEFAULT_COLUMN_WIDTH_EXCEL = DEFAULT_COLUMN_WIDTH_PIXEL / COLUMN_WIDTH_FACTOR;

/**
 * Excel 默认行高（磅）
 * 计算: 27 / 0.75 = 36
 */
export const DEFAULT_ROW_HEIGHT_EXCEL = DEFAULT_ROW_HEIGHT_PIXEL / ROW_HEIGHT_FACTOR;

// ============ 对齐转换 ============

/**
 * 水平对齐枚举映射
 * Univer: 0=UNSPECIFIED, 1=LEFT, 2=CENTER, 3=RIGHT, 4=JUSTIFIED
 * Excel: left, center, right, justify
 */
export const HORIZONTAL_ALIGN_MAP: Record<number, string> = {
  0: undefined as any, // UNSPECIFIED - 不设置
  1: 'left',
  2: 'center',
  3: 'right',
  4: 'justify',
};

/**
 * Excel 水平对齐转 Univer 数字 (导入时使用)
 * @param align Excel 水平对齐字符串
 * @returns Univer 水平对齐数字
 */
export function excelHorizontalAlignToUniver(align: string): number {
  const map: Record<string, number> = {
    'left': 1,
    'center': 2,
    'right': 3,
    'justify': 4,
  };
  return map[align] || 0;
}

/**
 * Univer 水平对齐数字转 Excel 字符串 (导出时使用)
 * @param ht Univer 水平对齐数字
 * @returns Excel 水平对齐字符串
 */
export function univerHorizontalAlignToExcel(ht: number): string | undefined {
  return HORIZONTAL_ALIGN_MAP[ht];
}

/**
 * 垂直对齐枚举映射
 * Univer: 0=UNSPECIFIED, 1=TOP, 2=MIDDLE, 3=BOTTOM
 * Excel: top, middle, bottom
 */
export const VERTICAL_ALIGN_MAP: Record<number, string> = {
  0: undefined as any, // UNSPECIFIED - 不设置
  1: 'top',
  2: 'middle',
  3: 'bottom',
};

/**
 * Excel 垂直对齐转 Univer 数字 (导入时使用)
 * @param align Excel 垂直对齐字符串
 * @returns Univer 垂直对齐数字
 */
export function excelVerticalAlignToUniver(align: string): number {
  const map: Record<string, number> = {
    'top': 1,
    'middle': 2,
    'bottom': 3,
  };
  return map[align] || 0;
}

/**
 * Univer 垂直对齐数字转 Excel 字符串 (导出时使用)
 * @param vt Univer 垂直对齐数字
 * @returns Excel 垂直对齐字符串
 */
export function univerVerticalAlignToExcel(vt: number): string | undefined {
  return VERTICAL_ALIGN_MAP[vt];
}

// ============ 换行策略转换 ============

/**
 * 换行策略枚举 (WrapStrategy)
 * Univer: 0=UNSPECIFIED, 1=OVERFLOW, 2=CLIP, 3=WRAP
 * Excel wrapText: true=WRAP, false=OVERFLOW/CLIP
 */
export const WRAP_STRATEGY_MAP: Record<number, boolean> = {
  0: undefined as any, // UNSPECIFIED - 不设置
  1: false, // OVERFLOW - 溢出
  2: false, // CLIP - 裁剪
  3: true,  // WRAP - 自动换行
};

/**
 * Excel wrapText 转 Univer 换行策略 (导入时使用)
 * @param wrapText Excel 是否自动换行
 * @returns Univer 换行策略数字
 */
export function excelWrapTextToUniver(wrapText: boolean): number {
  return wrapText ? 3 : 1; // wrapText=true → WRAP(3), wrapText=false → OVERFLOW(1)
}

/**
 * Univer 换行策略转 Excel wrapText (导出时使用)
 * @param tb Univer 换行策略数字
 * @returns Excel wrapText 值
 */
export function univerWrapStrategyToExcel(tb: number): boolean | undefined {
  return WRAP_STRATEGY_MAP[tb];
}

// ============ 边框样式转换 ============

/**
 * 边框样式枚举映射
 * Univer: 0=NONE, 1=THIN, 2=HAIR, 3=DOTTED, 4=DASHED, ...
 * Excel: thin, hair, dotted, dashed, ...
 */
export const BORDER_STYLE_MAP: Record<number, string> = {
  0: undefined as any, // NONE
  1: 'thin',
  2: 'hair',
  3: 'dotted',
  4: 'dashed',
  5: 'dashDot',
  6: 'dashDotDot',
  7: 'double',
  8: 'medium',
  9: 'mediumDashed',
  10: 'mediumDashDot',
  11: 'mediumDashDotDot',
  12: 'slantDashDot',
  13: 'thick',
};

/**
 * Excel 边框样式转 Univer 数字 (导入时使用)
 * @param style Excel 边框样式字符串
 * @returns Univer 边框样式数字
 */
export function excelBorderStyleToUniver(style: string): number {
  const map: Record<string, number> = {
    'thin': 1,
    'hair': 2,
    'dotted': 3,
    'dashed': 4,
    'dashDot': 5,
    'dashDotDot': 6,
    'double': 7,
    'medium': 8,
    'mediumDashed': 9,
    'mediumDashDot': 10,
    'mediumDashDotDot': 11,
    'slantDashDot': 12,
    'thick': 13,
  };
  return map[style] || 0;
}

/**
 * Univer 边框样式数字转 Excel 字符串 (导出时使用)
 * @param s Univer 边框样式数字
 * @returns Excel 边框样式字符串
 */
export function univerBorderStyleToExcel(s: number): string | undefined {
  return BORDER_STYLE_MAP[s];
}

// ============ 颜色转换 ============

/**
 * Univer 颜色转 Excel ARGB 格式 (导出时使用)
 * Univer 颜色格式: { rgb: '#RRGGBB' } 或字符串 '#RRGGBB'
 * Excel ARGB 格式: 'FFRRGGBB' (FF 是 Alpha 通道，表示完全不透明)
 * @param color Univer 颜色对象或字符串
 * @returns Excel ARGB 颜色字符串
 */
export function univerColorToArgb(color: { rgb?: string } | string | undefined): string | undefined {
  if (!color) return undefined;

  let rgb = typeof color === 'string' ? color : color.rgb;
  if (!rgb) return undefined;

  // 去掉 # 号
  rgb = rgb.replace('#', '');

  // 处理 rgb(r, g, b) 格式
  if (rgb.startsWith('rgb')) {
    const match = rgb.match(/\d+/g);
    if (match && match.length === 3) {
      rgb = match.map((p: string) => parseInt(p).toString(16).padStart(2, '0')).join('');
    }
  }

  // 验证颜色格式
  if (!rgb || rgb.length < 6) return undefined;

  // 添加 Alpha 通道 (FF = 255 = 完全不透明)
  return 'FF' + rgb.toUpperCase();
}

/**
 * Excel ARGB 颜色转 Univer 格式 (导入时使用)
 * Excel ARGB 格式: 'FFRRGGBB' 或 'RRGGBB'
 * Univer 颜色格式: { rgb: '#RRGGBB' }
 * @param argb Excel ARGB 颜色字符串
 * @returns Univer 颜色对象
 */
export function argbColorToUniver(argb: string | undefined): { rgb: string } | undefined {
  if (!argb) return undefined;

  // 去掉 Alpha 通道 (前两位)
  let rgb = argb.length === 8 ? argb.substring(2) : argb;

  // 验证颜色格式
  if (!rgb || rgb.length !== 6) return undefined;

  // 添加 # 号
  return { rgb: '#' + rgb.toUpperCase() };
}