import React, { useState, useEffect } from 'react';
import { Shield, Plus, Edit, Trash2, X, Check } from 'lucide-react';
import { Role, Resource, Company } from '../types';
import { authFetch } from '../utils/auth';

const RolesPage: React.FC = () => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [resources, setResources] = useState<Resource[]>([]);
  const [companies, setCompanies] = useState<Company[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    company_id: 0,
    description: '',
    resources: [] as number[],
  });

  useEffect(() => {
    fetchRoles();
    fetchResources();
    fetchCompanies();
  }, []);

  const fetchRoles = async () => {
    setLoading(true);
    try {
      const res = await authFetch('/api/portal/roles');
      const data = await res.json();
      setRoles(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch roles:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchResources = async () => {
    try {
      const res = await authFetch('/api/portal/resources');
      const data = await res.json();
      setResources(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch resources:', err);
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
      const url = editingRole
        ? `/api/portal/roles/${editingRole.id}`
        : '/api/portal/roles';
      const method = editingRole ? 'PATCH' : 'POST';

      const res = await authFetch(url, {
        method,
        body: JSON.stringify({
          ...formData,
          company_id: formData.company_id || null,
        }),
      });

      if (res.ok) {
        setShowModal(false);
        setEditingRole(null);
        setFormData({ name: '', company_id: 0, description: '', resources: [] });
        fetchRoles();
      }
    } catch (err) {
      console.error('Failed to save role:', err);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个角色吗？')) return;
    try {
      await authFetch(`/api/portal/roles/${id}`, { method: 'DELETE' });
      fetchRoles();
    } catch (err) {
      console.error('Failed to delete role:', err);
    }
  };

  const openEditModal = (role: Role) => {
    setEditingRole(role);
    setFormData({
      name: role.name,
      company_id: role.company_id || 0,
      description: role.description || '',
      resources: role.resources?.map(r => r.id) || [],
    });
    setShowModal(true);
  };

  const toggleResource = (resourceId: number) => {
    setFormData(prev => ({
      ...prev,
      resources: prev.resources.includes(resourceId)
        ? prev.resources.filter(id => id !== resourceId)
        : [...prev.resources, resourceId],
    }));
  };

  // Group resources by parent
  const groupedResources = resources.reduce((acc, res) => {
    if (res.type === 'menu') {
      acc.push({ menu: res, buttons: resources.filter(r => r.parent_id === res.id) });
    }
    return acc;
  }, [] as { menu: Resource; buttons: Resource[] }[]);

  return (
    <div className="p-8">
      <div className="mb-8 flex items-center justify-between">
        <h1 className="text-3xl font-bold text-zinc-900 flex items-center gap-3">
          <Shield className="w-8 h-8 text-blue-600" />
          角色管理
        </h1>
        <button
          onClick={() => {
            setEditingRole(null);
            setFormData({ name: '', company_id: 0, description: '', resources: [] });
            setShowModal(true);
          }}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition-colors"
        >
          <Plus className="w-5 h-5" />
          添加角色
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {roles.map((role) => (
          <div key={role.id} className="bg-white rounded-2xl shadow-sm border border-zinc-100 overflow-hidden">
            <div className="p-6">
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h3 className="text-lg font-bold text-zinc-900">{role.name}</h3>
                  {role.company_name && (
                    <span className="text-xs text-purple-600 bg-purple-50 px-2 py-0.5 rounded mt-1 inline-block">
                      {role.company_name}
                    </span>
                  )}
                  <p className="text-sm text-zinc-500 mt-1">{role.description || '暂无描述'}</p>
                </div>
                <div className="flex gap-1">
                  <button
                    onClick={() => openEditModal(role)}
                    className="p-2 text-blue-600 hover:bg-blue-50 rounded-lg"
                  >
                    <Edit className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => handleDelete(role.id)}
                    className="p-2 text-red-600 hover:bg-red-50 rounded-lg"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>

              <div className="space-y-2">
                <div className="text-sm text-zinc-500">权限资源:</div>
                <div className="flex flex-wrap gap-2">
                  {role.resources?.slice(0, 5).map((res) => (
                    <span key={res.id} className="px-2 py-1 bg-zinc-100 text-zinc-600 rounded text-xs">
                      {res.name}
                    </span>
                  ))}
                  {(role.resources?.length || 0) > 5 && (
                    <span className="px-2 py-1 bg-zinc-100 text-zinc-600 rounded text-xs">
                      +{(role.resources?.length || 0) - 5} 更多
                    </span>
                  )}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold">{editingRole ? '编辑角色' : '添加角色'}</h2>
              <button onClick={() => setShowModal(false)} className="p-2 hover:bg-zinc-100 rounded-lg">
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">角色名称</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">所属公司</label>
                <select
                  value={formData.company_id}
                  onChange={(e) => setFormData({ ...formData, company_id: Number(e.target.value) })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value={0}>全局角色（不限公司）</option>
                  {companies.map((company) => (
                    <option key={company.id} value={company.id}>
                      {company.name}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">描述</label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                  rows={3}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-2">权限资源</label>
                <div className="space-y-3 max-h-60 overflow-y-auto border border-zinc-200 rounded-xl p-4">
                  {groupedResources.map(({ menu, buttons }) => (
                    <div key={menu.id}>
                      <label className="flex items-center gap-2 font-medium text-zinc-900 mb-2">
                        <input
                          type="checkbox"
                          checked={formData.resources.includes(menu.id)}
                          onChange={() => toggleResource(menu.id)}
                          className="w-4 h-4 rounded border-zinc-300 text-blue-600 focus:ring-blue-500"
                        />
                        {menu.name}
                      </label>
                      <div className="ml-6 flex flex-wrap gap-2">
                        {buttons.map((btn) => (
                          <label
                            key={btn.id}
                            className={`flex items-center gap-1 px-2 py-1 rounded-lg cursor-pointer text-xs ${
                              formData.resources.includes(btn.id)
                                ? 'bg-blue-100 text-blue-700'
                                : 'bg-zinc-100 text-zinc-600'
                            }`}
                          >
                            <input
                              type="checkbox"
                              checked={formData.resources.includes(btn.id)}
                              onChange={() => toggleResource(btn.id)}
                              className="sr-only"
                            />
                            {formData.resources.includes(btn.id) && <Check className="w-3 h-3" />}
                            {btn.name}
                          </label>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
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
                  {editingRole ? '保存' : '创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default RolesPage;