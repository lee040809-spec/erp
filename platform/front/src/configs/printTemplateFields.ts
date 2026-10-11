/** Shared part fields; document-specific order and voucher fields stay in each preview. */
export const printItemFields = [
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
