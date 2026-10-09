import React from 'react';
import { X, Plus, Trash2 } from 'lucide-react';
import { Customer, Contact } from '../types';

interface CustomerModalProps {
  show: boolean;
  onClose: () => void;
  editingCustomer: Customer | null;
  newCustomer: { name: string; short_name: string; contacts: Contact[] };
  setNewCustomer: (customer: { name: string; short_name: string; contacts: Contact[] }) => void;
  onSave: () => void;
}

const CustomerModal: React.FC<CustomerModalProps> = ({
  show,
  onClose,
  editingCustomer,
  newCustomer,
  setNewCustomer,
  onSave
}) => {
  const addContact = () => {
    setNewCustomer({
      ...newCustomer,
      contacts: [...newCustomer.contacts, { name: '', contact: '' }]
    });
  };

  const removeContact = (index: number) => {
    setNewCustomer({
      ...newCustomer,
      contacts: newCustomer.contacts.filter((_, i) => i !== index)
    });
  };

  const updateContact = (index: number, field: 'name' | 'contact', value: string) => {
    const updatedContacts = [...newCustomer.contacts];
    updatedContacts[index] = { ...updatedContacts[index], [field]: value };
    setNewCustomer({
      ...newCustomer,
      contacts: updatedContacts
    });
  };

  if (!show) return null;

  return (
    <div className="fixed inset-0 z-[120] flex items-center justify-center p-4">
      <div
        onClick={onClose}
        className="absolute inset-0 bg-zinc-900/60 backdrop-blur-sm"
      />
      <div className="relative w-full max-w-lg bg-white rounded-3xl shadow-2xl">
        <div className="p-4 border-b border-zinc-100 flex items-center justify-between bg-zinc-50/50">
          <h3 className="text-xl font-bold">{editingCustomer ? '修改客户' : '新建客户'}</h3>
          <button onClick={onClose} className="p-2 hover:bg-zinc-100 rounded-full">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          {/* 客户名称和简称一行 */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-bold text-zinc-400 uppercase mb-1">客户名称 *</label>
              <input
                type="text"
                required
                placeholder="输入客户名称..."
                value={newCustomer.name}
                onChange={e => setNewCustomer({ ...newCustomer, name: e.target.value })}
                className="w-full px-4 py-2 bg-white border border-zinc-200 rounded-xl focus:ring-2 focus:ring-zinc-900 outline-none"
              />
            </div>
            <div>
              <label className="block text-xs font-bold text-zinc-400 uppercase mb-1">客户简称 *</label>
              <input
                type="text"
                required
                placeholder="输入客户简称..."
                value={newCustomer.short_name}
                onChange={e => setNewCustomer({ ...newCustomer, short_name: e.target.value })}
                className="w-full px-4 py-2 bg-white border border-zinc-200 rounded-xl focus:ring-2 focus:ring-zinc-900 outline-none"
              />
            </div>
          </div>

          {/* 联系人区域 */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-xs font-bold text-zinc-400 uppercase">联系人</label>
              <button
                type="button"
                onClick={addContact}
                className="flex items-center gap-1 text-xs text-zinc-500 hover:text-zinc-700 transition-colors"
              >
                <Plus className="w-4 h-4" />
                添加
              </button>
            </div>

            <div className="space-y-2">
              {newCustomer.contacts.map((contact, index) => (
                <div key={index} className="flex items-center gap-2 p-3 bg-zinc-50 rounded-xl">
                  <input
                    type="text"
                    placeholder="联系人"
                    value={contact.name}
                    onChange={e => updateContact(index, 'name', e.target.value)}
                    className="flex-1 px-3 py-1.5 bg-white border border-zinc-200 rounded-lg text-sm focus:ring-2 focus:ring-zinc-900 outline-none"
                  />
                  <input
                    type="text"
                    placeholder="联系方式"
                    value={contact.contact}
                    onChange={e => updateContact(index, 'contact', e.target.value)}
                    className="flex-1 px-3 py-1.5 bg-white border border-zinc-200 rounded-lg text-sm focus:ring-2 focus:ring-zinc-900 outline-none"
                  />
                  <button
                    type="button"
                    onClick={() => removeContact(index)}
                    className="p-1.5 text-zinc-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              ))}
              {newCustomer.contacts.length === 0 && (
                <div className="text-center text-zinc-400 text-sm py-4">
                  暂无联系人，点击"添加"按钮添加
                </div>
              )}
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-zinc-500 hover:text-zinc-700 transition-colors"
            >
              取消
            </button>
            <button
              type="button"
              onClick={onSave}
              className="bg-zinc-900 text-white px-6 py-2 rounded-xl font-medium hover:bg-zinc-800 transition-colors"
            >
              保存
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default CustomerModal;