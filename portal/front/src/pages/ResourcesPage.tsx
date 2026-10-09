import React, { useState, useEffect } from 'react';
import { Settings, Plus, Edit, Trash2, X } from 'lucide-react';
import { Resource } from '../types';
import { authFetch } from '../utils/auth';

const ResourcesPage: React.FC = () => {
  const [resources, setResources] = useState<Resource[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingResource, setEditingResource] = useState<Resource | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    type: 'menu' as 'menu' | 'button' | 'api',
    path: '',
    parent_id: 0,
    sort_order: 0,
  });

  useEffect(() => {
    fetchResources();
  }, []);

  const fetchResources = async () => {
    setLoading(true);
    try {
      const res = await authFetch('/api/portal/resources');
      const data = await res.json();
      setResources(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch resources:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const url = editingResource
        ? `/api/portal/resources/${editingResource.id}`
        : '/api/portal/resources';
      const method = editingResource ? 'PATCH' : 'POST';

      const res = await authFetch(url, {
        method,
        body: JSON.stringify(formData),
      });

      if (res.ok) {
        setShowModal(false);
        setEditingResource(null);
        setFormData({ name: '', type: 'menu', path: '', parent_id: 0, sort_order: 0 });
        fetchResources();
      }
    } catch (err) {
      console.error('Failed to save resource:', err);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个资源吗？')) return;
    try {
      await authFetch(`/api/portal/resources/${id}`, { method: 'DELETE' });
      fetchResources();
    } catch (err) {
      console.error('Failed to delete resource:', err);
    }
  };

  const openEditModal = (resource: Resource) => {
    setEditingResource(resource);
    setFormData({
      name: resource.name,
      type: resource.type,
      path: resource.path,
      parent_id: resource.parent_id,
      sort_order: resource.sort_order,
    });
    setShowModal(true);
  };

  const typeColors = {
    menu: 'bg-blue-100 text-blue-700',
    button: 'bg-green-100 text-green-700',
    api: 'bg-orange-100 text-orange-700',
  };

  const menuResources = resources.filter(r => r.type === 'menu');

  return (
    <div className="p-8">
      <div className="mb-8 flex items-center justify-between">
        <h1 className="text-3xl font-bold text-zinc-900 flex items-center gap-3">
          <Settings className="w-8 h-8 text-blue-600" />
          资源管理
        </h1>
        <button
          onClick={() => {
            setEditingResource(null);
            setFormData({ name: '', type: 'menu', path: '', parent_id: 0, sort_order: 0 });
            setShowModal(true);
          }}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition-colors"
        >
          <Plus className="w-5 h-5" />
          添加资源
        </button>
      </div>

      <div className="bg-white rounded-2xl shadow-sm border border-zinc-100">
        <table className="w-full">
          <thead className="bg-zinc-50">
            <tr>
              <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">名称</th>
              <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">类型</th>
              <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">路径</th>
              <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">排序</th>
              <th className="px-6 py-4 text-right text-sm font-semibold text-zinc-900">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {resources.map((resource) => (
              <tr key={resource.id} className="hover:bg-zinc-50">
                <td className="px-6 py-4 font-medium">
                  {resource.parent_id > 0 && '└ '}
                  {resource.name}
                </td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 rounded-lg text-sm ${typeColors[resource.type]}`}>
                    {resource.type === 'menu' ? '菜单' : resource.type === 'button' ? '按钮' : 'API'}
                  </span>
                </td>
                <td className="px-6 py-4 text-zinc-500 text-sm font-mono">{resource.path}</td>
                <td className="px-6 py-4 text-zinc-500">{resource.sort_order}</td>
                <td className="px-6 py-4">
                  <div className="flex items-center justify-end gap-2">
                    <button
                      onClick={() => openEditModal(resource)}
                      className="p-2 text-blue-600 hover:bg-blue-50 rounded-lg"
                      title="编辑"
                    >
                      <Edit className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(resource.id)}
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
      </div>

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold">{editingResource ? '编辑资源' : '添加资源'}</h2>
              <button onClick={() => setShowModal(false)} className="p-2 hover:bg-zinc-100 rounded-lg">
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">名称</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">类型</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value as any })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="menu">菜单</option>
                  <option value="button">按钮</option>
                  <option value="api">API</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">路径</label>
                <input
                  type="text"
                  value={formData.path}
                  onChange={(e) => setFormData({ ...formData, path: e.target.value })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="/portal/xxx"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">父菜单</label>
                <select
                  value={formData.parent_id}
                  onChange={(e) => setFormData({ ...formData, parent_id: Number(e.target.value) })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value={0}>无（顶级菜单）</option>
                  {menuResources.map((menu) => (
                    <option key={menu.id} value={menu.id}>
                      {menu.name}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">排序</label>
                <input
                  type="number"
                  value={formData.sort_order}
                  onChange={(e) => setFormData({ ...formData, sort_order: Number(e.target.value) })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
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
                  {editingResource ? '保存' : '创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default ResourcesPage;