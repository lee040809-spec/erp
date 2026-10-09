import React from 'react';
import { Layers, RefreshCw } from 'lucide-react';
import { Resource } from '../types';
import Pagination from '../components/Pagination';
import { authFetch } from '../components/shared';

const ResourceManagement: React.FC = () => {
  const [resources, setResources] = React.useState<Resource[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(20);
  const [loading, setLoading] = React.useState(false);
  const [filters, setFilters] = React.useState({
    resource_type: '',
    platform_type: ''
  });

  const fetchResources = React.useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('page', String(page));
      params.append('pageSize', String(pageSize));
      if (filters.resource_type) params.append('resource_type', filters.resource_type);
      if (filters.platform_type) params.append('platform_type', filters.platform_type);

      const res = await authFetch(`/api/platform/resources?${params.toString()}`);
      const result = await res.json();
      if (result.data) {
        setResources(result.data);
        setTotal(result.total);
      }
    } catch (error) {
      console.error('Failed to fetch resources:', error);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, filters]);

  React.useEffect(() => {
    fetchResources();
  }, [fetchResources]);

  const getResourceTypeLabel = (type: string) => {
    switch (type) {
      case 'menu': return '菜单';
      case 'api': return 'API';
      case 'button': return '按钮';
      default: return type;
    }
  };

  const getPlatformTypeLabel = (type: string) => {
    switch (type) {
      case 'business': return '业务平台';
      case 'manage': return '管理平台';
      default: return type;
    }
  };

  return (
    <div className="erp-legacy-page flex-1 !w-full flex flex-col min-h-0 bg-white">
      {/* Header */}
      <div className="hidden md:flex items-center justify-between gap-4 px-4 md:px-8 py-4 border-b border-zinc-200">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-zinc-900 flex items-center gap-2">
            <Layers className="w-8 h-8" />
            资源管理
            <span className="text-base font-normal text-zinc-500">共 {total} 个资源</span>
          </h2>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={fetchResources}
            disabled={loading}
            className="p-2 border border-zinc-200 rounded-xl hover:bg-zinc-50 disabled:opacity-50"
            title="刷新"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-4 px-4 md:px-8 py-4 bg-zinc-50 border-b border-zinc-200">
        <div className="w-40">
          <label className="text-[10px] font-bold text-zinc-400 uppercase tracking-wider ml-1">资源类型</label>
          <select
            value={filters.resource_type}
            onChange={(e) => { setFilters({ ...filters, resource_type: e.target.value }); setPage(1); }}
            className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900 bg-white"
          >
            <option value="">全部</option>
            <option value="menu">菜单</option>
            <option value="api">API</option>
            <option value="button">按钮</option>
          </select>
        </div>
        <div className="w-40">
          <label className="text-[10px] font-bold text-zinc-400 uppercase tracking-wider ml-1">平台类型</label>
          <select
            value={filters.platform_type}
            onChange={(e) => { setFilters({ ...filters, platform_type: e.target.value }); setPage(1); }}
            className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900 bg-white"
          >
            <option value="">全部</option>
            <option value="business">业务平台</option>
            <option value="manage">管理平台</option>
          </select>
        </div>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="min-w-[1200px] w-full text-left text-sm">
          <thead className="bg-zinc-50 sticky top-0 z-10">
            <tr className="border-b border-zinc-200">
              <th className="px-6 py-4 font-bold text-zinc-900">资源名称</th>
              <th className="px-6 py-4 font-bold text-zinc-900">资源标识</th>
              <th className="px-6 py-4 font-bold text-zinc-900">资源类型</th>
              <th className="px-6 py-4 font-bold text-zinc-900">平台类型</th>
              <th className="px-6 py-4 font-bold text-zinc-900">路径</th>
              <th className="px-6 py-4 font-bold text-zinc-900">图标</th>
              <th className="px-6 py-4 font-bold text-zinc-900">排序</th>
              <th className="px-6 py-4 font-bold text-zinc-900">状态</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {loading ? (
              <tr>
                <td colSpan={8} className="px-6 py-8 text-center text-zinc-400">加载中...</td>
              </tr>
            ) : resources.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-6 py-8 text-center text-zinc-400">暂无资源数据</td>
              </tr>
            ) : (
              resources.map((resource) => (
                <tr key={resource.id} className="hover:bg-zinc-50 transition-colors">
                  <td className="px-6 py-4 font-medium">{resource.name}</td>
                  <td className="px-6 py-4 text-zinc-500 font-mono text-xs">{resource.resource_key}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-lg text-xs font-bold ${
                      resource.resource_type === 'menu'
                        ? 'bg-blue-100 text-blue-700'
                        : resource.resource_type === 'api'
                        ? 'bg-amber-100 text-amber-700'
                        : 'bg-emerald-100 text-emerald-700'
                    }`}>
                      {getResourceTypeLabel(resource.resource_type)}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-lg text-xs font-bold ${
                      resource.platform_type === 'business'
                        ? 'bg-purple-100 text-purple-700'
                        : 'bg-zinc-100 text-zinc-600'
                    }`}>
                      {getPlatformTypeLabel(resource.platform_type || '')}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-zinc-500">{resource.path || '-'}</td>
                  <td className="px-6 py-4 text-zinc-500">{resource.icon || '-'}</td>
                  <td className="px-6 py-4 text-zinc-500">{resource.sort_order}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-lg text-xs font-bold ${
                      resource.status === 'active'
                        ? 'bg-emerald-100 text-emerald-700'
                        : 'bg-red-100 text-red-700'
                    }`}>
                      {resource.status === 'active' ? '启用' : '禁用'}
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <Pagination
        total={total}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={(size) => { setPageSize(size); setPage(1); }}
        activeColor="zinc"
      />
    </div>
  );
};

export default ResourceManagement;