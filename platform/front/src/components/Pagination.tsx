import React from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';

interface PaginationProps {
  total: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  itemName?: string;
  activeColor?: string;
}

const Pagination: React.FC<PaginationProps> = ({
  total,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
  itemName = '订单',
  activeColor = 'blue'
}) => {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  // Generate color classes based on theme color name
  const colorClasses: Record<string, { bg: string; shadow: string }> = {
    blue: { bg: 'bg-blue-600', shadow: 'shadow-blue-100' },
    rose: { bg: 'bg-rose-600', shadow: 'shadow-rose-100' },
    orange: { bg: 'bg-orange-600', shadow: 'shadow-orange-100' },
    amber: { bg: 'bg-amber-600', shadow: 'shadow-amber-100' },
    zinc: { bg: 'bg-zinc-600', shadow: 'shadow-zinc-100' },
  };

  const colors = colorClasses[activeColor] || colorClasses.blue;

  const getPageNumbers = () => {
    if (totalPages <= 1) return [1];
    return Array.from({ length: totalPages }, (_, i) => i + 1)
      .filter(p => p === 1 || p === totalPages || Math.abs(p - page) <= 1);
  };

  const pageNumbers = getPageNumbers();

  return (
    <div className="erp-pagination flex flex-col sm:flex-row items-center justify-between gap-4 px-4 md:px-8 py-3 bg-white border-t border-zinc-100 flex-shrink-0">
      <div className="flex items-center gap-4 text-sm text-zinc-500">
        <span>共<span className="font-bold text-zinc-900">{total}</span> 个{itemName}</span>
        <select
          aria-label="每页数量"
          value={pageSize}
          onChange={(e) => {
            onPageSizeChange(Number(e.target.value));
          }}
          className="bg-zinc-50 border border-zinc-200 rounded-lg px-2 py-1 outline-none focus:ring-2 focus:ring-zinc-900"
        >
          {[10, 20, 50, 100].map(size => (
            <option key={size} value={size}>每页 {size} 条</option>
          ))}
        </select>
      </div>
      <div className="erp-pagination-controls flex items-center gap-2">
        <button
          aria-label="上一页"
          disabled={page === 1}
          onClick={() => onPageChange(page - 1)}
          className="p-2 border border-zinc-200 rounded-xl hover:bg-zinc-50 disabled:opacity-30 transition-colors"
        >
          <ChevronLeft className="w-5 h-5" />
        </button>
        <span className="erp-pagination-position" aria-live="polite">第 {page} / {totalPages} 页</span>
        <div className="erp-pagination-pages flex items-center gap-1">
          {pageNumbers.map((p, i, arr) => (
            <React.Fragment key={p}>
              {i > 0 && arr[i - 1] !== p - 1 && <span className="px-2 text-zinc-400">...</span>}
              <button
                onClick={() => onPageChange(p)}
                className={`w-10 h-10 rounded-xl font-bold transition-all ${
                  page === p
                    ? `${colors.bg} text-white shadow-lg ${colors.shadow}`
                    : 'hover:bg-zinc-100 text-zinc-500'
                }`}
              >
                {p}
              </button>
            </React.Fragment>
          ))}
        </div>
        <button
          aria-label="下一页"
          disabled={page === totalPages || total === 0}
          onClick={() => onPageChange(page + 1)}
          className="p-2 border border-zinc-200 rounded-xl hover:bg-zinc-50 disabled:opacity-30 transition-colors"
        >
          <ChevronRight className="w-5 h-5" />
        </button>
      </div>
    </div>
  );
};

export default Pagination;
