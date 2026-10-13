import { AlertTriangle, Search, RefreshCw } from 'lucide-react';

export default function WorkbenchResultsState({ loading, error, empty, filtered, onRetry, onReset }: {
  loading: boolean;
  error: string;
  empty: boolean;
  filtered: boolean;
  onRetry: () => void;
  onReset: () => void;
}) {
  if (loading) return <div role="status" className="py-12 px-4 text-center text-sm text-slate-500">正在读取生产任务…</div>;
  if (error) return <div role="alert" className="py-10 px-4 text-center">
    <AlertTriangle size={30} className="mx-auto mb-3 text-amber-500"/>
    <p className="font-medium text-slate-700">生产任务暂时读取失败</p>
    <p className="text-sm text-slate-500 mt-2">{error}</p>
    <button type="button" className="erp-btn mt-4" onClick={onRetry}><RefreshCw size={14}/>重新读取</button>
  </div>;
  if (!empty) return null;
  return <div role="status" className="py-10 px-4 text-center">
    <Search size={30} className="mx-auto mb-3 text-slate-300"/>
    <p className="font-medium text-slate-700">{filtered ? '没有找到符合条件的任务' : '当前没有待办任务'}</p>
    {filtered && <button type="button" className="erp-btn mt-4" onClick={onReset}>清除筛选，查看全部任务</button>}
  </div>;
}
