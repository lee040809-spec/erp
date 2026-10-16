import React from 'react';
import { Shield, Plus, Edit, Trash2, RefreshCw, Link2, Unlink } from 'lucide-react';
import { Role } from '../types';
import Pagination from '../components/Pagination';
import { authFetch } from '../components/shared';

interface Permission {
  id: number;
  role_id: number;
  resource_id: number;
  permission: string;
  resource_key?: string;
  name?: string;
  resource_type?: string;
  icon?: string;
  path?: string;
  sort_order?: number;
}

interface Resource {
  id: number;
  resource_key: string;
  name: string;
  resource_type: string;
  icon?: string;
  path?: string;
  sort_order?: number;
}

const RoleManagement: React.FC = () => {
  const [roles, setRoles] = React.useState<Role[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(20);
  const [loading, setLoading] = React.useState(false);

  // Modal states
  const [showModal, setShowModal] = React.useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = React.useState(false);
  const [showPermissionModal, setShowPermissionModal] = React.useState(false);
  const [editingRole, setEditingRole] = React.useState<Role | null>(null);
  const [deletingRole, setDeletingRole] = React.useState<Role | null>(null);
  const [bindingRole, setBindingRole] = React.useState<Role | null>(null);

  // Permission state
  const [permissions, setPermissions] = React.useState<Permission[]>([]);
  const [allResources, setAllResources] = React.useState<Resource[]>([]);
  const [permissionLoading, setPermissionLoading] = React.useState(false);

  // Form state
  const [formData, setFormData] = React.useState({
    name: '',
    account_type: 'user' as 'admin' | 'user'
  });

  const fetchRoles = React.useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('page', String(page));
      params.append('pageSize', String(pageSize));

      const res = await authFetch(`/api/platform/roles?${params.toString()}`);
      const result = await res.json();
      if (result.data) {
        setRoles(result.data);
        setTotal(result.total);
      }
    } catch (error) {
      console.error('Failed to fetch roles:', error);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  React.useEffect(() => {
    fetchRoles();
  }, [fetchRoles]);

  const handleCreate = () => {
    setEditingRole(null);
    setFormData({ name: '', account_type: 'user' });
    setShowModal(true);
  };

  const handleEdit = (role: Role) => {
    setEditingRole(role);
    setFormData({
      name: role.name,
      account_type: role.account_type
    });
    setShowModal(true);
  };

  const handleDelete = (role: Role) => {
    setDeletingRole(role);
    setShowDeleteConfirm(true);
  };

  const handleBindResource = async (role: Role) => {
    setBindingRole(role);
    setPermissionLoading(true);
    setShowPermissionModal(true);

    try {
      // 获取角色权限和所有资源
      const [permRes, resourceRes] = await Promise.all([
        authFetch(`/api/platform/roles/${role.id}/permissions`),
        authFetch(`/api/platform/roles/${role.id}/resources`)
      ]);

      const permResult = await permRes.json();
      const resourceResult = await resourceRes.json();

      if (permResult.data) {
        setPermissions(permResult.data);
      }
      if (resourceResult.data) {
        setAllResources(resourceResult.data);
      }
    } catch (error) {
      console.error('Failed to fetch permissions:', error);
    } finally {
      setPermissionLoading(false);
    }
  };

  const handleBindPermission = async (resourceId: number) => {
    if (!bindingRole) return;

    try {
      const res = await authFetch(`/api/platform/roles/${bindingRole.id}/permissions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ resource_id: resourceId, permission: 'read' })
      });
      const result = await res.json();
      if (result.success) {
        // 刷新权限列表
        const permRes = await authFetch(`/api/platform/roles/${bindingRole.id}/permissions`);
        const permResult = await permRes.json();
        if (permResult.data) {
          setPermissions(permResult.data);
        }
      } else {
        alert(result.error || '绑定失败');
      }
    } catch (error) {
      console.error('Failed to bind permission:', error);
      alert('绑定失败');
    }
  };

  const handleUnbindPermission = async (permissionId: number) => {
    if (!bindingRole) return;

    try {
      const res = await authFetch(`/api/platform/roles/${bindingRole.id}/permissions/${permissionId}`, {
        method: 'DELETE'
      });
      const result = await res.json();
      if (result.success) {
        // 刷新权限列表
        const permRes = await authFetch(`/api/platform/roles/${bindingRole.id}/permissions`);
        const permResult = await permRes.json();
        if (permResult.data) {
          setPermissions(permResult.data);
        }
      } else {
        alert(result.error || '解绑失败');
      }
    } catch (error) {
      console.error('Failed to unbind permission:', error);
      alert('解绑失败');
    }
  };

  const confirmDelete = async () => {
    if (!deletingRole) return;
    try {
      const res = await authFetch(`/api/platform/roles/${deletingRole.id}`, {
        method: 'DELETE'
      });
      const result = await res.json();
      if (result.success) {
        fetchRoles();
        setShowDeleteConfirm(false);
        setDeletingRole(null);
      } else {
        alert(result.error || '删除失败');
      }
    } catch (error) {
      console.error('Failed to delete role:', error);
      alert('删除失败');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (editingRole) {
        const res = await authFetch(`/api/platform/roles/${editingRole.id}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(formData)
        });
        const result = await res.json();
        if (result.success) {
          setShowModal(false);
          fetchRoles();
        } else {
          alert(result.error || '更新失败');
        }
      } else {
        const res = await authFetch('/api/platform/roles', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(formData)
        });
        const result = await res.json();
        if (result.success) {
          setShowModal(false);
          fetchRoles();
        } else {
          alert(result.error || '创建失败');
        }
      }
    } catch (error) {
      console.error('Failed to save role:', error);
      alert('操作失败');
    }
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleDateString('zh-CN');
  };

  // 获取已绑定的资源ID列表
  const boundResourceIds = permissions.map(p => p.resource_id);

  // 过滤出未绑定的资源
  const unboundResources = allResources.filter(r => !boundResourceIds.includes(r.id));

  return (
    <div className="erp-legacy-page flex-1 !w-full flex flex-col min-h-0 bg-white">
      {/* Header */}
      <div className="hidden md:flex items-center justify-between gap-4 px-4 md:px-8 py-4 border-b border-zinc-200">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-zinc-900 flex items-center gap-2">
            <Shield className="w-8 h-8" />
            角色管理
            <span className="text-base font-normal text-zinc-500">共 {total} 个角色</span>
          </h2>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={fetchRoles}
            disabled={loading}
            className="p-2 border border-zinc-200 rounded-xl hover:bg-zinc-50 disabled:opacity-50"
            title="刷新"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
          <button
            onClick={handleCreate}
            className="bg-zinc-900 text-white px-4 py-2 rounded-xl font-medium flex items-center gap-2 hover:bg-zinc-800 transition-colors"
          >
            <Plus className="w-4 h-4" />
            新建角色
          </button>
        </div>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="min-w-[800px] w-full text-left text-sm">
          <thead className="bg-zinc-50 sticky top-0 z-10">
            <tr className="border-b border-zinc-200">
              <th className="px-6 py-4 font-bold text-zinc-900">角色名称</th>
              <th className="px-6 py-4 font-bold text-zinc-900">账号类型</th>
              <th className="px-6 py-4 font-bold text-zinc-900">状态</th>
              <th className="px-6 py-4 font-bold text-zinc-900">创建时间</th>
              <th className="px-6 py-4 font-bold text-zinc-900">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {loading ? (
              <tr>
                <td colSpan={5} className="px-6 py-8 text-center text-zinc-400">加载中...</td>
              </tr>
            ) : roles.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-6 py-8 text-center text-zinc-400">暂无角色数据</td>
              </tr>
            ) : (
              roles.map((role) => (
                <tr key={role.id} className="hover:bg-zinc-50 transition-colors">
                  <td className="px-6 py-4 font-medium">{role.name}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-lg text-xs font-bold ${
                      role.account_type === 'admin'
                        ? 'bg-purple-100 text-purple-700'
                        : 'bg-zinc-100 text-zinc-600'
                    }`}>
                      {role.account_type === 'admin' ? '管理员' : '普通用户'}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-lg text-xs font-bold ${
                      role.status === 'active'
                        ? 'bg-emerald-100 text-emerald-700'
                        : 'bg-red-100 text-red-700'
                    }`}>
                      {role.status === 'active' ? '启用' : '禁用'}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-zinc-500">{formatDate(role.created_at)}</td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => handleEdit(role)}
                        className="p-2 hover:bg-zinc-100 rounded-lg text-zinc-400 hover:text-zinc-900 transition-colors"
                        title="编辑"
                      >
                        <Edit className="w-4 h-4" />
                      </button>
                      {role.account_type === 'user' && (
                        <button
                          onClick={() => handleBindResource(role)}
                          className="p-2 hover:bg-blue-50 rounded-lg text-zinc-400 hover:text-blue-600 transition-colors"
                          title="绑定资源"
                        >
                          <Link2 className="w-4 h-4" />
                        </button>
                      )}
                      {role.account_type !== 'admin' && (
                        <button
                          onClick={() => handleDelete(role)}
                          className="p-2 hover:bg-red-50 rounded-lg text-zinc-400 hover:text-red-600 transition-colors"
                          title="删除"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      )}
                    </div>
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

      {/* Role Modal */}
      {showModal && (
        <div data-escape-dialog className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-md mx-4">
            <div className="px-6 py-4 border-b border-zinc-200">
              <h3 className="text-lg font-bold">{editingRole ? '编辑角色' : '新建角色'}</h3>
            </div>
            <form onSubmit={handleSubmit} className="p-6 space-y-4">
              <div>
                <label className="text-xs font-bold text-zinc-500 uppercase">角色名称</label>
                <input
                  type="text"
                  required
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900"
                />
              </div>
              <div>
                <label className="text-xs font-bold text-zinc-500 uppercase">账号类型</label>
                <select
                  value={formData.account_type}
                  onChange={(e) => setFormData({ ...formData, account_type: e.target.value as 'admin' | 'user' })}
                  disabled={true}
                  className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900 bg-zinc-100 cursor-not-allowed"
                >
                  <option value="user">普通用户</option>
                  <option value="admin">管理员</option>
                </select>
              </div>
              <div className="flex gap-3 pt-4">
                <button data-escape-close
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="flex-1 px-4 py-2 border border-zinc-200 rounded-xl text-sm font-medium hover:bg-zinc-50 transition-colors"
                >
                  取消
                </button>
                <button
                  type="submit"
                  className="flex-1 px-4 py-2 bg-zinc-900 text-white rounded-xl text-sm font-medium hover:bg-zinc-800 transition-colors"
                >
                  {editingRole ? '更新' : '创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Confirm Modal */}
      {showDeleteConfirm && deletingRole && (
        <div data-escape-dialog className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-sm mx-4 p-6">
            <h3 className="text-lg font-bold mb-4">确认删除</h3>
            <p className="text-zinc-600 mb-6">确定要删除角色 "{deletingRole.name}" 吗？此操作不可恢复。</p>
            <div className="flex gap-3">
              <button data-escape-close
                onClick={() => { setShowDeleteConfirm(false); setDeletingRole(null); }}
                className="flex-1 px-4 py-2 border border-zinc-200 rounded-xl text-sm font-medium hover:bg-zinc-50 transition-colors"
              >
                取消
              </button>
              <button
                onClick={confirmDelete}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded-xl text-sm font-medium hover:bg-red-700 transition-colors"
              >
                删除
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Permission Binding Modal */}
      {showPermissionModal && bindingRole && (
        <div data-escape-dialog className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-3xl mx-4 max-h-[80vh] flex flex-col">
            <div className="px-6 py-4 border-b border-zinc-200 flex items-center justify-between">
              <h3 className="text-lg font-bold">绑定资源 - {bindingRole.name}</h3>
              <button data-escape-close
                onClick={() => { setShowPermissionModal(false); setBindingRole(null); setPermissions([]); setAllResources([]); }}
                className="text-zinc-400 hover:text-zinc-600"
              >
                ✕
              </button>
            </div>

            {permissionLoading ? (
              <div className="flex-1 flex items-center justify-center">
                <RefreshCw className="w-6 h-6 animate-spin text-zinc-400" />
              </div>
            ) : (
              <div className="flex-1 overflow-auto p-6">
                <div className="grid grid-cols-2 gap-6 h-full min-h-[400px]">
                  {/* 左侧：已绑定资源 */}
                  <div className="flex flex-col border border-zinc-200 rounded-xl overflow-hidden">
                    <div className="px-4 py-3 bg-zinc-50 border-b border-zinc-200">
                      <h4 className="text-sm font-bold text-zinc-700">已绑定资源 ({permissions.length})</h4>
                    </div>
                    <div className="flex-1 overflow-auto p-3">
                      {permissions.length === 0 ? (
                        <p className="text-sm text-zinc-400 text-center py-8">暂无绑定资源</p>
                      ) : (
                        <div className="space-y-2">
                          {permissions.map((perm) => (
                            <div key={perm.id} className="flex items-center justify-between p-3 bg-zinc-50 rounded-lg hover:bg-zinc-100 transition-colors">
                              <div className="flex items-center gap-2">
                                <span className={`px-2 py-0.5 rounded text-xs font-bold ${
                                  perm.resource_type === 'menu' ? 'bg-blue-100 text-blue-700' :
                                  perm.resource_type === 'api' ? 'bg-amber-100 text-amber-700' :
                                  'bg-emerald-100 text-emerald-700'
                                }`}>
                                  {perm.resource_type === 'menu' ? '菜单' : perm.resource_type === 'api' ? 'API' : '按钮'}
                                </span>
                                <span className="font-medium text-sm">{perm.name}</span>
                              </div>
                              <button
                                onClick={() => handleUnbindPermission(perm.id)}
                                className="p-1.5 hover:bg-red-100 rounded text-red-500 hover:text-red-700 transition-colors"
                                title="解绑"
                              >
                                <Unlink className="w-4 h-4" />
                              </button>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>

                  {/* 右侧：可绑定资源 */}
                  <div className="flex flex-col border border-zinc-200 rounded-xl overflow-hidden">
                    <div className="px-4 py-3 bg-emerald-50 border-b border-zinc-200">
                      <h4 className="text-sm font-bold text-emerald-700">可绑定资源 ({unboundResources.length})</h4>
                    </div>
                    <div className="flex-1 overflow-auto p-3">
                      {unboundResources.length === 0 ? (
                        <p className="text-sm text-zinc-400 text-center py-8">所有资源已绑定</p>
                      ) : (
                        <div className="space-y-2">
                          {unboundResources.map((resource) => (
                            <div key={resource.id} className="flex items-center justify-between p-3 bg-emerald-50 rounded-lg hover:bg-emerald-100 transition-colors">
                              <div className="flex items-center gap-2">
                                <span className={`px-2 py-0.5 rounded text-xs font-bold ${
                                  resource.resource_type === 'menu' ? 'bg-blue-100 text-blue-700' :
                                  resource.resource_type === 'api' ? 'bg-amber-100 text-amber-700' :
                                  'bg-emerald-100 text-emerald-700'
                                }`}>
                                  {resource.resource_type === 'menu' ? '菜单' : resource.resource_type === 'api' ? 'API' : '按钮'}
                                </span>
                                <span className="font-medium text-sm">{resource.name}</span>
                              </div>
                              <button
                                onClick={() => handleBindPermission(resource.id)}
                                className="p-1.5 hover:bg-emerald-200 rounded text-emerald-600 hover:text-emerald-800 transition-colors"
                                title="绑定"
                              >
                                <Link2 className="w-4 h-4" />
                              </button>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div className="px-6 py-4 border-t border-zinc-200">
              <button data-escape-close
                onClick={() => { setShowPermissionModal(false); setBindingRole(null); setPermissions([]); setAllResources([]); }}
                className="w-full px-4 py-2 border border-zinc-200 rounded-xl text-sm font-medium hover:bg-zinc-50 transition-colors"
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default RoleManagement;