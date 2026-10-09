import BusinessEntrySettings from '../components/BusinessEntrySettings';
import React, { useState } from 'react';
import { Plus, Settings2, Search, Settings, Trash2, Link as LinkIcon, RefreshCw } from 'lucide-react';
import { AdventRule } from '../types';
import type { RuleForm } from '../components/RuleModal';
interface RuleFilters {
    name: string;
}
interface RulesProps {
    adventRules: AdventRule[];
    ruleFilters: RuleFilters;
    setRuleFilters: (filters: RuleFilters) => void;
    editingRuleId: number | null;
    setEditingRuleId: (id: number | null) => void;
    ruleForm: RuleForm;
    setRuleForm: (form: RuleForm) => void;
    setShowRuleModal: (show: boolean) => void;
    setDeletingRuleId: (id: number | null) => void;
    fetchData: () => void;
}
const Rules: React.FC<RulesProps> = ({ adventRules, ruleFilters, setRuleFilters, editingRuleId, setEditingRuleId, ruleForm, setRuleForm, setShowRuleModal, setDeletingRuleId, fetchData }) => {
    const [isRefreshing, setIsRefreshing] = useState(false);
    const handleRefresh = async () => {
        setIsRefreshing(true);
        try {
            await fetchData();
        }
        finally {
            setIsRefreshing(false);
        }
    };
    return (<div className="erp-legacy-page flex-1 flex flex-col min-h-0 space-y-8 animate-in fade-in duration-500 py-4 md:py-8 !w-full !max-w-none m-0 !p-0 relative">
      {/* 加载遮罩 */}
      {isRefreshing && (<div className="absolute inset-0 bg-white/80 z-50 flex items-center justify-center backdrop-blur-sm">
          <div className="flex items-center gap-3 text-zinc-600 bg-white px-6 py-3 rounded-xl shadow-lg border border-zinc-200">
            <RefreshCw className="w-5 h-5 animate-spin"/>
            <span className="font-medium">数据加载中...</span>
          </div>
        </div>)}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 px-4 md:px-8">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-zinc-900 flex items-center gap-2">
            <Settings2 className="w-8 h-8"/>
            规则管理
          </h2><BusinessEntrySettings />
          <p className="text-zinc-500">管理交货日期提醒的计算规则</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"/>
            <input type="text" placeholder="搜索规则名称..." value={ruleFilters.name} onChange={(e) => setRuleFilters({ ...ruleFilters, name: e.target.value })} className="pl-9 pr-4 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900 w-64"/>
          </div>
          <button onClick={handleRefresh} disabled={isRefreshing} className="p-2 hover:bg-zinc-100 rounded-lg text-zinc-500 transition-colors disabled:opacity-50" title="刷新数据">
            <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`}/>
          </button>
          <button onClick={() => {
            setEditingRuleId(null);
            setRuleForm({
                name: '',
                description: '',
                formula: '',
                target_status: 'pending',
                scopeType: 'general',
                ruleType: 'imminent'
            });
            setShowRuleModal(true);
        }} className="bg-zinc-900 text-white px-4 py-2 rounded-xl text-sm font-medium flex items-center gap-2 hover:bg-zinc-800 transition-colors">
            <Plus className="w-4 h-4"/>
            新建规则
          </button>
        </div>
      </div>

      <div className="flex-1 min-h-0 bg-white rounded-none border-y border-zinc-200 overflow-auto">
        <table className="w-full text-left text-sm">
          <thead className="bg-zinc-50 border-b border-zinc-200">
            <tr>
              <th className="px-6 py-4 font-semibold text-zinc-500">规则名称</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">规则类型</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">适用范围</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">规则逻辑</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">规则说明</th>
              <th className="px-6 py-4 font-semibold text-zinc-500 text-right">绑定订单</th>
              <th className="px-6 py-4 font-semibold text-zinc-500 text-left">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {adventRules.map((rule) => (<tr key={rule.id} className="hover:bg-zinc-50 transition-colors">
                <td className="px-6 py-4">
                  <span className="font-bold text-zinc-900">{rule.name}</span>
                </td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 rounded-lg text-[10px] font-bold border ${rule.ruleType === 'warning'
                ? 'bg-rose-50 text-rose-600 border-rose-100'
                : 'bg-indigo-50 text-indigo-600 border-indigo-100'}`}>
                    {rule.ruleType === 'warning' ? '告警' : '临期'}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 rounded-lg text-[10px] font-bold border ${rule.scopeType === 'general'
                ? 'bg-amber-50 text-amber-600 border-amber-100'
                : 'bg-zinc-100 text-zinc-500 border-zinc-200'}`}>
                    {rule.scopeType === 'general' ? '通用' : '特定（未绑定）'}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <div className="flex flex-col gap-2 py-1">
                    <div className="flex items-center gap-1">
                      <code className="bg-zinc-100 px-2 py-1 rounded text-xs font-mono text-zinc-600">
                        {rule.formula}
                      </code>
                      <span className="bg-zinc-900 text-white px-1.5 py-0.5 rounded text-[10px] font-bold ml-1">终</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-[10px] text-zinc-400 font-bold uppercase tracking-wider">零件状态为</span>
                      <span className={`px-2 py-0.5 rounded-full text-[10px] font-bold border ${rule.target_status === 'pending' ? 'bg-amber-100 text-amber-700 border-amber-200' :
                rule.target_status === 'processing' ? 'bg-blue-100 text-blue-700 border-blue-200' :
                    'bg-emerald-100 text-emerald-700 border-emerald-200'}`}>
                        {rule.target_status === 'pending' ? '待加工' : rule.target_status === 'processing' ? '加工中' : '已完成'}
                      </span>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 text-zinc-500">{rule.description || '-'}</td>
                <td className="px-6 py-4 text-right">
                  <button type="button" disabled title="订单绑定尚未接入；特定规则目前为草稿，不参与告警" className="opacity-50 cursor-not-allowed inline-flex items-center gap-1.5 px-3 py-1.5 bg-zinc-50 border border-zinc-200 text-zinc-600 rounded-lg font-bold text-xs hover:bg-zinc-100 transition-all shadow-sm">
                    <LinkIcon className="w-3.5 h-3.5"/>
                    绑定（待接入）
                  </button>
                </td>
                <td className="px-6 py-4 text-left">
                  <div className="flex justify-start gap-2">
                    <button onClick={() => {
                setEditingRuleId(rule.id);
                setRuleForm({
                    name: rule.name,
                    description: rule.description || '',
                    formula: rule.formula,
                    target_status: rule.target_status || 'pending',
                    scopeType: rule.scopeType || 'specific',
                    ruleType: rule.ruleType || 'imminent'
                });
                setShowRuleModal(true);
            }} className="p-2 hover:bg-white border hover:border-zinc-200 rounded-lg text-zinc-600 hover:text-zinc-900 transition-all font-bold text-xs flex items-center gap-1.5 shadow-sm">
                      <Settings className="w-3.5 h-3.5"/>
                      修改
                    </button>
                    <button onClick={() => setDeletingRuleId(rule.id)} className="p-2 hover:bg-rose-50 border border-transparent hover:border-rose-100 rounded-lg text-zinc-400 hover:text-rose-600 transition-all">
                      <Trash2 className="w-4 h-4"/>
                    </button>
                  </div>
                </td>
              </tr>))}
            {adventRules.length === 0 && (<tr>
                <td colSpan={7} className="px-6 py-12 text-center text-zinc-400">
                  暂无规则，点击右上角新建
                </td>
              </tr>)}
          </tbody>
        </table>
      </div>
    </div>);
};
export default Rules;
