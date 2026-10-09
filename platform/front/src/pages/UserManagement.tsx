import React from 'react';
import { Users, Plus, Edit, Trash2, Key, RefreshCw, Link2 } from 'lucide-react';
import { User, Role } from '../types';
import Pagination from '../components/Pagination';
import { authFetch } from '../components/shared';

interface UserManagementProps {
  onNewUser?: () => void;
}

const UserManagement: React.FC<UserManagementProps> = ({ onNewUser }) => {
  const [users, setUsers] = React.useState<User[]>([]);
  const [roles, setRoles] = React.useState<Role[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(20);
  const [loading, setLoading] = React.useState(false);
  const [filters, setFilters] = React.useState({
    username: '',
    name: '',
    status: '',
    role_type: ''
  });

  // Modal states
  const [showModal, setShowModal] = React.useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = React.useState(false);
  const [showResetPassword, setShowResetPassword] = React.useState(false);
  const [showBindRoleModal, setShowBindRoleModal] = React.useState(false);
  const [editingUser, setEditingUser] = React.useState<User | null>(null);
  const [deletingUser, setDeletingUser] = React.useState<User | null>(null);
  const [resettingUser, setResettingUser] = React.useState<User | null>(null);
  const [bindingUser, setBindingUser] = React.useState<User | null>(null);
  const [newPassword, setNewPassword] = React.useState('');

  // Form state
  const [formData, setFormData] = React.useState({
    username: '',
    password: '',
    name: '',
    email: '',
    phone: '',
    role_type: 'user' as 'admin' | 'user',
    status: 'active'
  });

  const fetchUsers = React.useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('page', String(page));
      params.append('pageSize', String(pageSize));
      if (filters.username) params.append('username', filters.username);
      if (filters.name) params.append('name', filters.name);
      if (filters.status) params.append('status', filters.status);
      if (filters.role_type) params.append('role_type', filters.role_type);

      const res = await authFetch(`/api/platform/users?${params.toString()}`);
      const result = await res.json();
      if (result.data) {
        setUsers(result.data);
        setTotal(result.total);
      }
    } catch (error) {
      console.error('Failed to fetch users:', error);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, filters]);

  const fetchRoles = React.useCallback(async () => {
    try {
      const res = await authFetch('/api/platform/roles?pageSize=100');
      const result = await res.json();
      if (result.data) {
        // 只显示普通用户类型的角色
        setRoles(result.data.filter((r: Role) => r.account_type === 'user'));
      }
    } catch (error) {
      console.error('Failed to fetch roles:', error);
    }
  }, []);

  React.useEffect(() => {
    fetchUsers();
    fetchRoles();
  }, [fetchUsers, fetchRoles]);

  const handleCreate = () => {
    setEditingUser(null);
    setFormData({
      username: '',
      password: '',
      name: '',
      email: '',
      phone: '',
      role_type: 'user',
      status: 'active'
    });
    setShowModal(true);
  };

  const handleEdit = (user: User) => {
    setEditingUser(user);
    setFormData({
      username: user.username,
      password: '',
      name: user.name,
      email: user.email || '',
      phone: user.phone || '',
      role_type: 'user',
      status: user.status
    });
    setShowModal(true);
  };

  const handleDelete = (user: User) => {
    setDeletingUser(user);
    setShowDeleteConfirm(true);
  };

  const handleResetPassword = (user: User) => {
    setResettingUser(user);
    setNewPassword('');
    setShowResetPassword(true);
  };

  const handleBindRole = (user: User) => {
    setBindingUser(user);
    setShowBindRoleModal(true);
  };

  const handleSelectRole = async (roleId: number) => {
    if (!bindingUser) return;
    try {
      const res = await authFetch(`/api/platform/users/${bindingUser.id}/bind-role`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role_id: roleId })
      });
      const result = await res.json();
      if (result.success) {
        fetchUsers();
        // 更新当前绑定用户的数据
        setBindingUser(result.data);
      } else {
        alert(result.error || '绑定失败');
      }
    } catch (error) {
      console.error('Failed to bind role:', error);
      alert('绑定失败');
    }
  };

  const handleUnbindRole = async () => {
    if (!bindingUser) return;
    try {
      const res = await authFetch(`/api/platform/users/${bindingUser.id}/bind-role`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role_id: null })
      });
      const result = await res.json();
      if (result.success) {
        fetchUsers();
        setBindingUser(result.data);
      } else {
        alert(result.error || '解绑失败');
      }
    } catch (error) {
      console.error('Failed to unbind role:', error);
      alert('解绑失败');
    }
  };

  const confirmDelete = async () => {
    if (!deletingUser) return;
    try {
      const res = await authFetch(`/api/platform/users/${deletingUser.id}`, {
        method: 'DELETE'
      });
      const result = await res.json();
      if (result.success) {
        fetchUsers();
        setShowDeleteConfirm(false);
        setDeletingUser(null);
      } else {
        alert(result.error || '删除失败');
      }
    } catch (error) {
      console.error('Failed to delete user:', error);
      alert('删除失败');
    }
  };

  const confirmResetPassword = async () => {
    if (!resettingUser || !newPassword) return;
    try {
      const res = await authFetch(`/api/platform/users/${resettingUser.id}/reset-password`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ new_password: newPassword })
      });
      const result = await res.json();
      if (result.success) {
        setShowResetPassword(false);
        setResettingUser(null);
        setNewPassword('');
        alert('密码重置成功');
      } else {
        alert(result.error || '重置失败');
      }
    } catch (error) {
      console.error('Failed to reset password:', error);
      alert('重置失败');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (editingUser) {
        const res = await authFetch(`/api/platform/users/${editingUser.id}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: formData.name,
            email: formData.email,
            phone: formData.phone,
            status: formData.status
          })
        });
        const result = await res.json();
        if (result.success) {
          setShowModal(false);
          fetchUsers();
        } else {
          alert(result.error || '更新失败');
        }
      } else {
        const res = await authFetch('/api/platform/users', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(formData)
        });
        const result = await res.json();
        if (result.success) {
          setShowModal(false);
          fetchUsers();
        } else {
          alert(result.error || '创建失败');
        }
      }
    } catch (error) {
      console.error('Failed to save user:', error);
      alert('操作失败');
    }
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '';
    return new Date(dateStr).toLocaleDateString('zh-CN');
  };

  const getRoleName = (user: User) => {
    if (user.role_type === 'admin') return '管理员';
    if (user.role) {
      return user.role.name;
    }
    if (user.role_id) {
      const role = roles.find(r => r.id === user.role_id);
      return role ? role.name : '普通用户';
    }
    return '普通用户';
  };

  return (
    <div className="erp-legacy-page flex-1 !w-full flex flex-col min-h-0 bg-white">
      {/* Header */}
      <div className="hidden md:flex items-center justify-between gap-4 px-4 md:px-8 py-4 border-b border-zinc-200">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-zinc-900 flex items-center gap-2">
            <Users className="w-8 h-8" />
            用户管理
            <span className="text-base font-normal text-zinc-500">共 {total} 个用户</span>
          </h2>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={fetchUsers}
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
            新建用户
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-4 px-4 md:px-8 py-4 bg-zinc-50 border-b border-zinc-200">
        <div className="min-w-[140px] flex-1">
          <label className="text-[10px] font-bold text-zinc-400 uppercase tracking-wider ml-1">账号</label>
          <input
            type="text"
            placeholder="搜索账号..."
            value={filters.username}
            onChange={(e) => { setFilters({ ...filters, username: e.target.value }); setPage(1); }}
            className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900"
          />
        </div>
        <div className="min-w-[140px] flex-1">
          <label className="text-[10px] font-bold text-zinc-400 uppercase tracking-wider ml-1">姓名</label>
          <input
            type="text"
            placeholder="搜索姓名..."
            value={filters.name}
            onChange={(e) => { setFilters({ ...filters, name: e.target.value }); setPage(1); }}
            className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900"
          />
        </div>
        <div className="w-32">
          <label className="text-[10px] font-bold text-zinc-400 uppercase tracking-wider ml-1">状态</label>
          <select
            value={filters.status}
            onChange={(e) => { setFilters({ ...filters, status: e.target.value }); setPage(1); }}
            className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900 bg-white"
          >
            <option value="">全部</option>
            <option value="active">启用</option>
            <option value="inactive">禁用</option>
          </select>
        </div>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="min-w-[1000px] w-full text-left text-sm">
          <thead className="bg-zinc-50 sticky top-0 z-10">
            <tr className="border-b border-zinc-200">
              <th className="px-6 py-4 font-bold text-zinc-900">账号</th>
              <th className="px-6 py-4 font-bold text-zinc-900">姓名</th>
              <th className="px-6 py-4 font-bold text-zinc-900">邮箱</th>
              <th className="px-6 py-4 font-bold text-zinc-900">电话</th>
              <th className="px-6 py-4 font-bold text-zinc-900">角色</th>
              <th className="px-6 py-4 font-bold text-zinc-900">状态</th>
              <th className="px-6 py-4 font-bold text-zinc-900">创建时间</th>
              <th className="px-6 py-4 font-bold text-zinc-900">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {loading ? (
              <tr>
                <td colSpan={8} className="px-6 py-8 text-center text-zinc-400">加载中...</td>
              </tr>
            ) : users.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-6 py-8 text-center text-zinc-400">暂无用户数据</td>
              </tr>
            ) : (
              users.map((user) => (
                <tr key={user.id} className="hover:bg-zinc-50 transition-colors">
                  <td className="px-6 py-4 font-medium">{user.username}</td>
                  <td className="px-6 py-4">{user.name}</td>
                  <td className="px-6 py-4 text-zinc-500">{user.email || '-'}</td>
                  <td className="px-6 py-4 text-zinc-500">{user.phone || '-'}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-lg text-xs font-bold ${
                      user.role_type === 'admin'
                        ? 'bg-purple-100 text-purple-700'
                        : 'bg-zinc-100 text-zinc-600'
                    }`}>
                      {getRoleName(user)}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-lg text-xs font-bold ${
                      user.status === 'active'
                        ? 'bg-emerald-100 text-emerald-700'
                        : 'bg-red-100 text-red-700'
                    }`}>
                      {user.status === 'active' ? '启用' : '禁用'}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-zinc-500">{formatDate(user.created_at)}</td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => handleEdit(user)}
                        className="p-2 hover:bg-zinc-100 rounded-lg text-zinc-400 hover:text-zinc-900 transition-colors"
                        title="编辑"
                      >
                        <Edit className="w-4 h-4" />
                      </button>
                      {user.role_type !== 'admin' && (
                        <button
                          onClick={() => handleBindRole(user)}
                          className="p-2 hover:bg-blue-50 rounded-lg text-zinc-400 hover:text-blue-600 transition-colors"
                          title="绑定角色"
                        >
                          <Link2 className="w-4 h-4" />
                        </button>
                      )}
                      <button
                        onClick={() => handleResetPassword(user)}
                        className="p-2 hover:bg-zinc-100 rounded-lg text-zinc-400 hover:text-zinc-900 transition-colors"
                        title="重置密码"
                      >
                        <Key className="w-4 h-4" />
                      </button>
                      {user.role_type !== 'admin' && (
                        <button
                          onClick={() => handleDelete(user)}
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

      {/* User Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-md mx-4">
            <div className="px-6 py-4 border-b border-zinc-200">
              <h3 className="text-lg font-bold">{editingUser ? '编辑用户' : '新建用户'}</h3>
            </div>
            <form onSubmit={handleSubmit} className="p-6 space-y-4">
              <div>
                <label className="text-xs font-bold text-zinc-500 uppercase">账号</label>
                <input
                  type="text"
                  required
                  disabled={!!editingUser}
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900 disabled:bg-zinc-100"
                />
              </div>
              {!editingUser && (
                <div>
                  <label className="text-xs font-bold text-zinc-500 uppercase">密码</label>
                  <input
                    type="password"
                    required={!editingUser}
                    value={formData.password}
                    onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                    className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900"
                  />
                </div>
              )}
              <div>
                <label className="text-xs font-bold text-zinc-500 uppercase">姓名</label>
                <input
                  type="text"
                  required
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900"
                />
              </div>
              <div>
                <label className="text-xs font-bold text-zinc-500 uppercase">邮箱</label>
                <input
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900"
                />
              </div>
              <div>
                <label className="text-xs font-bold text-zinc-500 uppercase">电话</label>
                <input
                  type="text"
                  value={formData.phone}
                  onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                  className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900"
                />
              </div>
              <div>
                <label className="text-xs font-bold text-zinc-500 uppercase">状态</label>
                <select
                  value={formData.status}
                  onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                  className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900 bg-white"
                >
                  <option value="active">启用</option>
                  <option value="inactive">禁用</option>
                </select>
              </div>
              <div className="flex gap-3 pt-4">
                <button
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
                  {editingUser ? '更新' : '创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Confirm Modal */}
      {showDeleteConfirm && deletingUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-sm mx-4 p-6">
            <h3 className="text-lg font-bold mb-4">确认删除</h3>
            <p className="text-zinc-600 mb-6">确定要删除用户 "{deletingUser.name}" 吗？此操作不可恢复。</p>
            <div className="flex gap-3">
              <button
                onClick={() => { setShowDeleteConfirm(false); setDeletingUser(null); }}
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

      {/* Reset Password Modal */}
      {showResetPassword && resettingUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-sm mx-4 p-6">
            <h3 className="text-lg font-bold mb-4">重置密码</h3>
            <p className="text-zinc-600 mb-4">为用户 "{resettingUser.name}" 设置新密码</p>
            <input
              type="password"
              placeholder="请输入新密码"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className="w-full px-3 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900 mb-4"
            />
            <div className="flex gap-3">
              <button
                onClick={() => { setShowResetPassword(false); setResettingUser(null); setNewPassword(''); }}
                className="flex-1 px-4 py-2 border border-zinc-200 rounded-xl text-sm font-medium hover:bg-zinc-50 transition-colors"
              >
                取消
              </button>
              <button
                onClick={confirmResetPassword}
                disabled={!newPassword}
                className="flex-1 px-4 py-2 bg-zinc-900 text-white rounded-xl text-sm font-medium hover:bg-zinc-800 transition-colors disabled:opacity-50"
              >
                确认重置
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Bind Role Modal */}
      {showBindRoleModal && bindingUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-2xl mx-4">
            <div className="px-6 py-4 border-b border-zinc-200">
              <h3 className="text-lg font-bold">绑定角色 - {bindingUser.name}</h3>
            </div>
            <div className="p-6">
              <div className="grid grid-cols-2 gap-6 min-h-[300px]">
                {/* 左侧：已绑定角色 */}
                <div className="flex flex-col border border-zinc-200 rounded-xl overflow-hidden">
                  <div className="px-4 py-3 bg-zinc-50 border-b border-zinc-200">
                    <h4 className="text-sm font-bold text-zinc-700">已绑定角色</h4>
                  </div>
                  <div className="flex-1 overflow-auto p-3">
                    {bindingUser.role ? (
                      <div className="flex items-center justify-between p-3 bg-zinc-50 rounded-lg">
                        <span className="font-medium">{bindingUser.role.name}</span>
                        <button
                          onClick={handleUnbindRole}
                          className="p-1.5 hover:bg-red-100 rounded text-red-500 hover:text-red-700 transition-colors"
                          title="解绑"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    ) : (
                      <p className="text-sm text-zinc-400 text-center py-8">暂未绑定角色</p>
                    )}
                  </div>
                </div>

                {/* 右侧：可绑定角色 */}
                <div className="flex flex-col border border-zinc-200 rounded-xl overflow-hidden">
                  <div className="px-4 py-3 bg-emerald-50 border-b border-zinc-200">
                    <h4 className="text-sm font-bold text-emerald-700">可绑定角色 ({roles.length})</h4>
                  </div>
                  <div className="flex-1 overflow-auto p-3">
                    {roles.length === 0 ? (
                      <p className="text-sm text-zinc-400 text-center py-8">暂无可绑定角色，请先创建角色</p>
                    ) : (
                      <div className="space-y-2">
                        {roles.map((role) => (
                          <div
                            key={role.id}
                            onClick={bindingUser.role ? undefined : () => handleSelectRole(role.id)}
                            className={`flex items-center justify-between p-3 rounded-lg transition-colors ${
                              bindingUser.role
                                ? 'bg-zinc-100 opacity-50 cursor-not-allowed'
                                : 'bg-emerald-50 cursor-pointer hover:bg-emerald-100'
                            }`}
                          >
                            <span className="font-medium">{role.name}</span>
                            <button
                              className={`p-1.5 rounded transition-colors ${
                                bindingUser.role
                                  ? 'text-zinc-400 cursor-not-allowed'
                                  : 'hover:bg-emerald-200 text-emerald-600'
                              }`}
                              title={bindingUser.role ? '已绑定角色' : '绑定'}
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
            <div className="px-6 py-4 border-t border-zinc-200">
              <button
                onClick={() => { setShowBindRoleModal(false); setBindingUser(null); }}
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

export default UserManagement;