import React, { useState, useEffect } from 'react';
import { Building2, Plus, Edit, Trash2, Search, X } from 'lucide-react';
import { Company } from '../types';
import { authFetch } from '../utils/auth';

const CompaniesPage: React.FC = () => {
  const [companies, setCompanies] = useState<Company[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [editingCompany, setEditingCompany] = useState<Company | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    code: '',
    description: '',
    status: 1,
  });

  useEffect(() => {
    fetchCompanies();
  }, []);

  const fetchCompanies = async () => {
    setLoading(true);
    try {
      const res = await authFetch('/api/portal/companies');
      const data = await res.json();
      setCompanies(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch companies:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const url = editingCompany
        ? `/api/portal/companies/${editingCompany.id}`
        : '/api/portal/companies';
      const method = editingCompany ? 'PATCH' : 'POST';

      const res = await authFetch(url, {
        method,
        body: JSON.stringify(formData),
      });

      if (res.ok) {
        setShowModal(false);
        setEditingCompany(null);
        setFormData({ name: '', code: '', description: '', status: 1 });
        fetchCompanies();
      }
    } catch (err) {
      console.error('Failed to save company:', err);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个公司吗？')) return;
    try {
      await authFetch(`/api/portal/companies/${id}`, { method: 'DELETE' });
      fetchCompanies();
    } catch (err) {
      console.error('Failed to delete company:', err);
    }
  };

  const openEditModal = (company: Company) => {
    setEditingCompany(company);
    setFormData({
      name: company.name,
      code: company.code,
      description: company.description || '',
      status: company.status,
    });
    setShowModal(true);
  };

  const filteredCompanies = companies.filter(c =>
    c.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    c.code.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="p-8">
      <div className="mb-8 flex items-center justify-between">
        <h1 className="text-3xl font-bold text-zinc-900 flex items-center gap-3">
          <Building2 className="w-8 h-8 text-blue-600" />
          公司管理
        </h1>
        <button
          onClick={() => {
            setEditingCompany(null);
            setFormData({ name: '', code: '', description: '', status: 1 });
            setShowModal(true);
          }}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition-colors"
        >
          <Plus className="w-5 h-5" />
          添加公司
        </button>
      </div>

      <div className="bg-white rounded-2xl shadow-sm border border-zinc-100">
        <div className="p-4 border-b border-zinc-100">
          <div className="relative">
            <Search className="w-5 h-5 text-zinc-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="搜索公司名称或代码..."
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
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">公司名称</th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">公司代码</th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">描述</th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">状态</th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-zinc-900">创建时间</th>
                <th className="px-6 py-4 text-right text-sm font-semibold text-zinc-900">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100">
              {filteredCompanies.map((company) => (
                <tr key={company.id} className="hover:bg-zinc-50">
                  <td className="px-6 py-4 font-medium">{company.name}</td>
                  <td className="px-6 py-4">
                    <span className="px-2 py-1 bg-zinc-100 text-zinc-700 rounded text-sm font-mono">
                      {company.code}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-zinc-500">{company.description || '-'}</td>
                  <td className="px-6 py-4">
                    {company.status === 1 ? (
                      <span className="px-2 py-1 bg-green-100 text-green-700 rounded-lg text-sm">
                        启用
                      </span>
                    ) : (
                      <span className="px-2 py-1 bg-red-100 text-red-700 rounded-lg text-sm">
                        禁用
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-zinc-500 text-sm">
                    {new Date(company.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => openEditModal(company)}
                        className="p-2 text-blue-600 hover:bg-blue-50 rounded-lg"
                        title="编辑"
                      >
                        <Edit className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleDelete(company.id)}
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
              <h2 className="text-xl font-bold">{editingCompany ? '编辑公司' : '添加公司'}</h2>
              <button onClick={() => setShowModal(false)} className="p-2 hover:bg-zinc-100 rounded-lg">
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">公司名称</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">公司代码</label>
                <input
                  type="text"
                  value={formData.code}
                  onChange={(e) => setFormData({ ...formData, code: e.target.value })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="如：YHS、BJKJ"
                  required
                />
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
                <label className="block text-sm font-medium text-zinc-700 mb-1">状态</label>
                <select
                  value={formData.status}
                  onChange={(e) => setFormData({ ...formData, status: Number(e.target.value) })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value={1}>启用</option>
                  <option value={0}>禁用</option>
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
                  {editingCompany ? '保存' : '创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default CompaniesPage;