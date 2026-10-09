import React, { useState } from 'react';
import ResponsiveTable from '../components/ResponsiveTable';
import { Plus, Users, Edit, Trash2, ChevronDown, ChevronUp, RefreshCw } from 'lucide-react';
import { Customer, Contact } from '../types';

interface CustomersProps {
  customers: Customer[];
  setEditingCustomer: (customer: Customer | null) => void;
  setNewCustomer: (customer: { name: string; short_name: string; contacts: Contact[] }) => void;
  setShowCustomerModal: (show: boolean) => void;
  setDeletingCustomerId: (id: number | null) => void;
  fetchData: () => void;
}

const Customers: React.FC<CustomersProps> = ({
  customers,
  setEditingCustomer,
  setNewCustomer,
  setShowCustomerModal,
  setDeletingCustomerId,
  fetchData
}) => {
  const [expandedCustomers, setExpandedCustomers] = useState<Set<number>>(new Set());
  const [isRefreshing, setIsRefreshing] = useState(false);

  const toggleExpand = (customerId: number) => {
    setExpandedCustomers(prev => {
      const newSet = new Set(prev);
      if (newSet.has(customerId)) {
        newSet.delete(customerId);
      } else {
        newSet.add(customerId);
      }
      return newSet;
    });
  };

  const handleRefresh = async () => {
    setIsRefreshing(true);
    try {
      await fetchData();
    } finally {
      setIsRefreshing(false);
    }
  };

  return (
    <div className="erp-legacy-page flex-1 flex flex-col min-h-0 space-y-8 animate-in fade-in duration-500 py-4 md:py-8 !w-full !max-w-none m-0 !p-0 relative">
      {/* 加载遮罩 */}
      {isRefreshing && (
        <div className="absolute inset-0 bg-white/80 z-50 flex items-center justify-center backdrop-blur-sm">
          <div className="flex items-center gap-3 text-zinc-600 bg-white px-6 py-3 rounded-xl shadow-lg border border-zinc-200">
            <RefreshCw className="w-5 h-5 animate-spin" />
            <span className="font-medium">数据加载中...</span>
          </div>
        </div>
      )}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 px-4 md:px-8">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-zinc-900 flex items-center gap-2">
            <Users className="w-8 h-8" />
            客户管理
          </h2>
          <p className="text-zinc-500">管理客户信息，修改客户名称不会影响历史订单</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="p-2 hover:bg-zinc-100 rounded-lg text-zinc-500 transition-colors disabled:opacity-50"
            title="刷新数据"
          >
            <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          </button>
          <button
            onClick={() => {
              setEditingCustomer(null);
              setNewCustomer({ name: '', short_name: '', contacts: [] });
              setShowCustomerModal(true);
            }}
            className="bg-zinc-900 text-white px-4 py-2 rounded-xl text-sm font-medium flex items-center gap-2 hover:bg-zinc-800 transition-colors"
          >
            <Plus className="w-4 h-4" />
            新建客户
          </button>
        </div>
      </div>

      <div className="flex-1 min-h-0 bg-white rounded-none border-y border-zinc-200 overflow-auto">
        <ResponsiveTable className="erp-customer-table w-full text-left text-sm">
          <thead className="bg-zinc-50 border-b border-zinc-200 sticky top-0 z-10">
            <tr>
              <th className="w-8 px-4 py-4"></th>
              <th className="px-6 py-4 font-semibold text-zinc-500">客户名称</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">客户简称</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">联系人</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">创建时间</th>
              <th className="px-6 py-4 font-semibold text-zinc-500 text-right">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {customers.map((customer) => {
              const isExpanded = expandedCustomers.has(customer.id);
              const contactCount = customer.contacts?.length || 0;

              return (
                <React.Fragment key={customer.id}>
                  {/* 主行 */}
                  <tr className="hover:bg-zinc-50 transition-colors">
                    <td className="px-4 py-4">
                      {contactCount > 0 && (
                        <button
                          onClick={() => toggleExpand(customer.id)}
                          aria-label={isExpanded ? "收起联系人" : "展开联系人"} className="p-1 hover:bg-zinc-100 rounded transition-colors"
                        >
                          {isExpanded ? (
                            <ChevronUp className="w-4 h-4 text-zinc-400" />
                          ) : (
                            <ChevronDown className="w-4 h-4 text-zinc-400" />
                          )}
                        </button>
                      )}
                    </td>
                    <td className="px-6 py-4">
                      <span className="font-bold text-zinc-900">{customer.name}</span>
                    </td>
                    <td className="px-6 py-4 text-zinc-500">{customer.short_name || '-'}</td>
                    <td className="px-6 py-4">
                      <span className={`text-sm ${contactCount > 0 ? 'text-zinc-600' : 'text-zinc-400'}`}>
                        {contactCount}人
                      </span>
                    </td>
                    <td className="px-6 py-4 text-zinc-500 text-sm">
                      {customer.created_at ? new Date(customer.created_at).toLocaleDateString('zh-CN') : '-'}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => {
                            setEditingCustomer(customer);
                            setNewCustomer({
                              name: customer.name,
                              short_name: customer.short_name || '',
                              contacts: customer.contacts || []
                            });
                            setShowCustomerModal(true);
                          }}
                          className="p-2 hover:bg-zinc-100 rounded-lg transition-colors"
                          title="编辑"
                        >
                          <Edit className="w-4 h-4 text-zinc-500" />
                        </button>
                        <button
                          onClick={() => setDeletingCustomerId(customer.id)}
                          className="p-2 hover:bg-red-50 rounded-lg transition-colors"
                          title="删除"
                        >
                          <Trash2 className="w-4 h-4 text-red-500" />
                        </button>
                      </div>
                    </td>
                  </tr>

                  {/* 联系人子行 */}
                  {isExpanded && contactCount > 0 && (
                    <tr className="bg-zinc-50/50">
                      <td colSpan={6} className="px-6 py-3">
                        <div className="ml-8 space-y-2">
                          <div className="grid grid-cols-2 gap-4 text-xs font-bold text-zinc-400 uppercase mb-1">
                            <span>联系人</span>
                            <span>联系方式</span>
                          </div>
                          {customer.contacts.map((contact, idx) => (
                            <div
                              key={contact.id || idx}
                              className="grid grid-cols-2 gap-4 text-sm py-2 border-b border-zinc-100 last:border-0"
                            >
                              <span className="text-zinc-700">{contact.name}</span>
                              <span className="text-zinc-500">{contact.contact}</span>
                            </div>
                          ))}
                        </div>
                      </td>
                    </tr>
                  )}
                </React.Fragment>
              );
            })}
            {customers.length === 0 && (
              <tr>
                <td colSpan={6} className="px-6 py-12 text-center text-zinc-400">
                  暂无客户数据，点击"新建客户"添加
                </td>
              </tr>
            )}
          </tbody>
        </ResponsiveTable>
      </div>
    </div>
  );
};

export default Customers;