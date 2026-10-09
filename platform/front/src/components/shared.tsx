import React from 'react';
import { Order } from '../types';
// Process options and colors
export const PROCESS_OPTIONS = ['下料', '车削', '铣削', '磨削', '线切割', '电火花', '热处理', '表面处理', '送货'];
export const PROCESS_COLORS: Record<string, string> = {
    '下料': 'bg-orange-50 border-orange-200 hover:bg-orange-100',
    '车削': 'bg-blue-50 border-blue-200 hover:bg-blue-100',
    '铣削': 'bg-indigo-50 border-indigo-200 hover:bg-indigo-100',
    '磨削': 'bg-cyan-50 border-cyan-200 hover:bg-cyan-100',
    '线切割': 'bg-purple-50 border-purple-200 hover:bg-purple-100',
    '电火花': 'bg-fuchsia-50 border-fuchsia-200 hover:bg-fuchsia-100',
    '热处理': 'bg-rose-50 border-rose-200 hover:bg-rose-100',
    '表面处理': 'bg-emerald-50 border-emerald-200 hover:bg-emerald-100',
    '送货': 'bg-zinc-50 border-zinc-200 hover:bg-zinc-100',
};
export const ProcessStatusBadge = ({ status }: {
    status: 'pending' | 'processing' | 'completed';
}) => {
    const styles = {
        pending: 'bg-zinc-100 text-zinc-500 border-zinc-200',
        processing: 'bg-blue-50 text-blue-600 border-blue-100',
        completed: 'bg-emerald-50 text-emerald-600 border-emerald-100',
    };
    const labels = {
        pending: '待加工',
        processing: '加工中',
        completed: '已完成',
    };
    return (<span className={`px-2 py-0.5 rounded text-[10px] font-bold border ${styles[status]}`}>
      {labels[status]}
    </span>);
};
export const StatusBadge = ({ status, onUpdate }: {
    status: Order['status'];
    onUpdate?: (newStatus: Order['status']) => void;
}) => {
    const styles = {
        pending: 'bg-amber-50 text-amber-700 border-amber-200',
        processing: 'bg-blue-100 text-blue-700 border-blue-200',
        completed: 'bg-emerald-100 text-emerald-700 border-emerald-200',
        delivered: 'bg-violet-100 text-violet-700 border-violet-200',
    };
    const labels = {
        pending: '待加工',
        processing: '加工中',
        completed: '已完成',
        delivered: '已送货',
    };
    const handleClick = (e: React.MouseEvent) => {
        if (!onUpdate)
            return;
        e.stopPropagation();
        // Cycle logic: pending -> processing -> completed -> pending
        const cycle: Order['status'][] = ['pending', 'processing', 'completed'];
        const currentIndex = cycle.indexOf(status);
        const nextStatus = currentIndex === -1 ? 'pending' : cycle[(currentIndex + 1) % cycle.length];
        onUpdate(nextStatus);
    };
    return (<button type="button" disabled={!onUpdate} onClick={handleClick} className={`px-3 py-1 rounded-full text-xs font-bold border transition-all whitespace-nowrap ${onUpdate ? 'active:scale-95 hover:brightness-95' : 'cursor-default'} ${styles[status] || styles.pending}`}>
      {labels[status] || labels.pending}
    </button>);
};
export const PriorityBadge = ({ priority }: {
    priority: string;
}) => {
    const styles: Record<string, string> = {
        high: 'bg-rose-100 text-rose-700 border-rose-200',
        medium: 'bg-amber-100 text-amber-700 border-amber-200',
        low: 'bg-zinc-100 text-zinc-500 border-zinc-200',
    };
    const labels: Record<string, string> = {
        high: '紧急',
        medium: '普通',
        low: '较低',
    };
    const style = styles[priority] || styles.medium;
    return (<span className={`px-2 py-0.5 rounded-full text-[10px] font-bold border ${style}`}>
      {labels[priority] || labels.medium}
    </span>);
};
// Helper to format date to YYYY-MM-DD
export const formatDate = (date: string | null | undefined): string => {
    if (!date)
        return '-';
    // Already in correct format
    if (/^\d{4}-\d{2}-\d{2}$/.test(date))
        return date;
    // ISO string format with timezone (e.g. 2026-03-10T00:00:00+08:00)
    if (/^\d{4}-\d{2}-\d{2}T/.test(date)) {
        // 直接提取日期部分，避免时区转换问题
        return date.substring(0, 10);
    }
    // Other formats
    const d = new Date(date);
    if (isNaN(d.getTime()))
        return '-';
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
};
// Helper to format date for form input (returns empty string instead of '-')
// Business DATE values preserve their written calendar day across time zones.
export const formatDateForInput = (date: string | null | undefined): string => {
    if (!date)
        return '';
    const businessDay = date.match(/^(\d{4}-\d{2}-\d{2})(?:$|T| )/);
    if (businessDay)
        return businessDay[1];
    const value = new Date(date);
    if (isNaN(value.getTime()))
        return '';
    return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`;
};
// Helper for authorized fetch
export const authFetch = async (url: string, options: RequestInit = {}): Promise<Response> => {
    const token = localStorage.getItem('auth_token');
    const headers: HeadersInit = {
        ...options.headers,
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
    };
    const response = await fetch(url, { ...options, headers });
    // 处理token失效
    if (response.status === 401) {
        const data = await response.clone().json().catch(() => ({}));
        // 错误码 4003 或 4004 表示token无效/过期
        if (data.code === 4003 || data.code === 4004) {
            localStorage.removeItem('auth_token');
            window.location.href = '/';
        }
    }
    return response;
};
