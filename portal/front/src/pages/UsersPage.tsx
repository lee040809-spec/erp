import React, { useState, useEffect } from 'react';
import { Users, Plus, Edit, Trash2, Key, Unlock, Search, X } from 'lucide-react';
import { User, Role, Company } from '../types';
import { authFetch } from '../utils/auth';

const UsersPage: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [companies, setCompanies] = useState<Company[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    email: '',
    company_id: 0,
    role_id: 1,
  });

  useEffect(() => {
    fetchUsers();
    fetchRoles();
    fetchCompanies();
  }, []);

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const res = await authFetch('/api/portal/users');
      const data = await res.json();
      setUsers(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch users:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchRoles = async () => {
    try {
      const res = await authFetch('/api/portal/roles');
      const data = await res.json();
      setRoles(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch roles:', err);
    }
  };

  const fetchCompanies = async () => {
    try {
      const res = await authFetch('/api/portal/companies');
      const data = await res.json();
      setCompanies(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch companies:', err);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const url = editingUser
        ? `/api/portal/users/${editingUser.id}`
        : '/api/portal/users';
      const method = editingUser ? 'PATCH' : 'POST';

      const body: any = {
        email: formData.email,
        company_id: formData.company_id || null,
        role_id: formData.role_id,
      };

      if (!editingUser) {
        body.username = formData.username;
        body.password = formData.password;
      }

      const res = await authFetch(url, {
        method,
        body: JSON.stringify(body),
      });

      if (res.ok) {
        setShowModal(false);
        setEditingUser(null);
        setFormData({ username: '', password: '', email: '', company_id: 0, role_id: 1 });
        fetchUsers();
      }
    } catch (err) {
      console.error('Failed to save user:', err);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个用户吗？')) return;
    try {
      await authFetch(`/api/portal/users/${id}`, { method: 'DELETE' });
      fetchUsers();
    } catch (err) {
      console.error('Failed to delete user:', err);
    }
  };

  const handleResetPassword = async (id: number) => {
    if (!confirm('确定要重置该用户的密码吗？')) return;
    try {
      const res = await authFetch(`/api/portal/users/${id}/reset-password`, { method: 'POST' });
      const data = await res.json();
      if (data.success) {
        alert(`密码已重置为: ${data.new_password}\n请通过邮件通知用户`);
      }
    } catch (err) {
      console.error('Failed to reset password:', err);
    }
  };

  const handleUnlock = async (id: number) => {
    try {
      await authFetch(`/api/portal/users/${id}/unlock`, { method: 'POST' });
      fetchUsers();
    } catch (err) {
      console.error('Failed to unlock user:', err);
    }
  };

  const openEditModal = (user: User) => {
    setEditingUser(user);
    setFormData({
      username: user.username,
      password: '',
      email: user.email || '',
      company_id: user.company_id || 0,
      role_id: user.role_id,
    });
    setShowModal(true);
  };

  const filteredUsers = users.filter(u =>
    u.username.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="p-8">
      <div className="mb-8 flex items-center justify-between">
        <h1 className="text-3xl font-bold text-zinc-900 flex items-center gap-3">
          <Users className="w-8 h-8 text-blue-600" />
          用户管理
        </h1>
        <button
          onClick={() => {
            setEditingUser(null);
            setFormData({ username: '', password: '', email: '', company_id: 0, role_id: 1 });
            setShowModal(true);
          }}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition-colors"
        >
          <Plus className="w-5 h-5" />
          添加用户
        </button>
      </div>

      <div className="bg-white rounded-2xl shadow-sm border border-zinc-100">
        <div className="p-4 border-b border-zinc-100">
          <div className="relative">
            <Search className="w-5 h-5 text-zinc-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="搜索用户..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        {loading ? (
          <div className="p-8 text-center text-zinc-500">加载中...</div>
        ) : (
          <table className="w-full">
            <thead className="bg-zinc-50">
              <tr>
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">用户名</th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">邮箱</th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">公司</th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">角色</th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">状态</th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">创建时间</th>
                <th className="px-6 py-4 text-right text-sm font-semibold text-zinc-900">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100">
              {filteredUsers.map((user) => (
                <tr key={user.id} className="hover:bg-zinc-50">
                  <td className="px-6 py-4 font-medium">{user.username}</td>
                  <td className="px-6 py-4 text-zinc-500">{user.email || '-'}</td>
                  <td className="px-6 py-4">
                    {user.company_name ? (
                      <span className="px-2 py-1 bg-purple-100 text-purple-700 rounded-lg text-sm">
                        {user.company_name}
                      </span>
                    ) : '-'}
                  </td>
                  <td className="px-6 py-4">
                    <span className="px-2 py-1 bg-blue-100 text-blue-700 rounded-lg text-sm">
                      {user.role_name}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    {user.lock_until ? (
                      <span className="px-2 py-1 bg-red-100 text-red-700 rounded-lg text-sm">
                        已锁定
                      </span>
                    ) : (
                      <span className="px-2 py-1 bg-green-100 text-green-700 rounded-lg text-sm">
                        正常
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-zinc-500 text-sm">
                    {new Date(user.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center justify-end gap-2">
                      {user.lock_until && (
                        <button
                          onClick={() => handleUnlock(user.id)}
                          className="p-2 text-green-600 hover:bg-green-50 rounded-lg"
                          title="解锁"
                        >
                          <Unlock className="w-4 h-4" />
                        </button>
                      )}
                      <button
                        onClick={() => handleResetPassword(user.id)}
                        className="p-2 text-orange-600 hover:bg-orange-50 rounded-lg"
                        title="重置密码"
                      >
                        <Key className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => openEditModal(user)}
                        className="p-2 text-blue-600 hover:bg-blue-50 rounded-lg"
                        title="编辑"
                      >
                        <Edit className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleDelete(user.id)}
                        className="p-2 text-red-600 hover:bg-red-50 rounded-lg"
                        title="删除"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold">{editingUser ? '编辑用户' : '添加用户'}</h2>
              <button onClick={() => setShowModal(false)} className="p-2 hover:bg-zinc-100 rounded-lg">
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">用户名</label>
                <input
                  type="text"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  disabled={!!editingUser}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-zinc-100"
                  required
                />
              </div>

              {!editingUser && (
                <div>
                  <label className="block text-sm font-medium text-zinc-700 mb-1">密码</label>
                  <input
                    type="password"
                    value={formData.password}
                    onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                    className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                    required={!editingUser}
                  />
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">邮箱</label>
                <input
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">公司</label>
                <select
                  value={formData.company_id}
                  onChange={(e) => setFormData({ ...formData, company_id: Number(e.target.value) })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value={0}>无</option>
                  {companies.map((company) => (
                    <option key={company.id} value={company.id}>
                      {company.name}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">角色</label>
                <select
                  value={formData.role_id}
                  onChange={(e) => setFormData({ ...formData, role_id: Number(e.target.value) })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {roles.map((role) => (
                    <option key={role.id} value={role.id}>
                      {role.name}
                    </option>
                  ))}
                </select>
              </div>

              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="flex-1 px-4 py-2 border border-zinc-200 rounded-xl hover:bg-zinc-50"
                >
                  取消
                </button>
                <button
                  type="submit"
                  className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-xl hover:bg-blue-700"
                >
                  {editingUser ? '保存' : '创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default UsersPage;