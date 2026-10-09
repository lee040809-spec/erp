import React, { useState, useEffect, useMemo } from 'react';
import { motion } from 'motion/react';
import { parseProgressRule, progressRuleMatches } from '../utils/progressRule';
import { businessToday } from '../utils/businessDate';
import { X, Eye, Clock, ChevronLeft, Trash2, AlertCircle } from 'lucide-react';

export interface RuleForm {
  name: string;
  description: string;
  formula: string;
  target_status: 'pending' | 'processing' | 'completed';
  scopeType: 'general' | 'specific';
  ruleType: 'warning' | 'imminent';
}

export interface AdventRule {
  id: number;
  name: string;
  description?: string;
  formula: string;
  target_status: string;
  scopeType: string;
  ruleType: string;
}

interface RuleModalProps {
  show: boolean;
  onClose: () => void;
  editingRuleId: number | null;
  ruleForm: RuleForm;
  setRuleForm: (form: RuleForm) => void;
  adventRules: AdventRule[];
  onSubmit: () => void;
  onValidationError: (message: string) => void;
}

const RuleModal: React.FC<RuleModalProps> = ({
  show,
  onClose,
  editingRuleId,
  ruleForm,
  setRuleForm,
  adventRules,
  onSubmit,
  onValidationError
}) => {
  // Preview state
  const [previewValues, setPreviewValues] = useState<{
    deliveryDate: string;
    orderDate: string;
    partStatus: 'pending' | 'processing' | 'completed';
  }>({
    deliveryDate: new Date(Date.parse(businessToday()+'T00:00:00Z') + 86400000 * 7).toISOString().split('T')[0],
    orderDate: businessToday(),
    partStatus: 'pending'
  });

  // Check for duplicate general rules
  const ruleError = useMemo(() => {
    if (ruleForm.scopeType === 'general' && Array.isArray(adventRules)) {
      const isDuplicate = adventRules.some(rule =>
        rule.scopeType === 'general' &&
        rule.ruleType === ruleForm.ruleType &&
        rule.id !== editingRuleId
      );
      if (isDuplicate) {
        return `系统中已存在通用${ruleForm.ruleType === 'warning' ? '告警' : '临期'}规则`;
      }
    }
    return null;
  }, [ruleForm.scopeType, ruleForm.ruleType, adventRules, editingRuleId]);

  // Evaluate a parsed range without executing user input as JavaScript.
  const calculatePreviewResult = (formula: string, target_status: string = 'pending'): string => {
    const rule=parseProgressRule(formula);
    if(!rule)return '计算错误 (请使用进度范围公式，例如 0<=(({当天}-{订单日期})/({交货日期}-{订单日期}))<=1)';
    if(previewValues.partStatus!==target_status)return '不触发(状态不匹配)';
    const day=(date:string)=>Date.parse(date+'T00:00:00Z')/86400000;
    const start=day(previewValues.orderDate),end=day(previewValues.deliveryDate);
    if(!Number.isFinite(start)||!Number.isFinite(end)||end===start)return '计算错误 (日期无效或交期等于下单日期)';
    return progressRuleMatches(rule,(day(businessToday())-start)/(end-start))?'成立 (True)':'不成立 (False)';
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    // Formula validation
    if (!ruleForm.formula.trim()) {
      onValidationError("规则公式不能为空，请组合公式");
      return;
    }

    const previewResult = calculatePreviewResult(ruleForm.formula, ruleForm.target_status);
    if (typeof previewResult === 'string' && previewResult.includes('计算错误')) {
      onValidationError("当前公式存在语法错误或计算异常，请调整后再保存");
      return;
    }

    onSubmit();
  };

  if (!show) return null;

  return (
    <div className="fixed inset-0 z-[130] flex items-center justify-center p-4">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
        className="absolute inset-0 bg-zinc-900/60 backdrop-blur-sm"
      />
      <motion.div
        initial={{ opacity: 0, scale: 0.95, y: 20 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.95, y: 20 }}
        className="relative w-full max-w-4xl bg-white rounded-3xl shadow-2xl flex flex-col overflow-hidden h-auto"
      >
        <div className="p-4 border-b border-zinc-100 flex items-center justify-between bg-zinc-50/50">
          <h3 className="text-xl font-bold">{editingRuleId ? '修改规则' : '新建规则'}</h3>
          <button onClick={onClose} className="p-2 hover:bg-zinc-100 rounded-full">
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4 flex-1 flex flex-col min-h-0">
          {/* Top Row: 4 Columns */}
          <div className="grid grid-cols-4 gap-4 items-start">
            <div className="space-y-1.5 flex-1 min-w-0">
              <label className="block text-xs font-bold text-zinc-400 uppercase">规则名称 *</label>
              <input
                type="text"
                required
                placeholder="例如：标准交期提醒"
                value={ruleForm.name}
                onChange={e => setRuleForm({ ...ruleForm, name: e.target.value })}
                className="w-full px-4 py-2 bg-zinc-50 border border-zinc-200 rounded-2xl focus:ring-2 focus:ring-zinc-900 outline-none"
              />
            </div>
            <div className="space-y-1.5 flex-1 min-w-0">
              <label className="block text-xs font-bold text-zinc-400 uppercase">规则说明</label>
              <input
                type="text"
                placeholder="简短描述规则用途..."
                value={ruleForm.description}
                onChange={e => setRuleForm({ ...ruleForm, description: e.target.value })}
                className="w-full px-4 py-2 bg-zinc-50 border border-zinc-200 rounded-2xl focus:ring-2 focus:ring-zinc-900 outline-none"
              />
            </div>
            <div className="space-y-1.5 flex-1 min-w-0">
              <label className="block text-xs font-bold text-zinc-400 uppercase">规则类型</label>
              <div className="flex bg-zinc-100 p-1 rounded-2xl border border-zinc-200">
                {[
                  { label: '告警', value: 'warning' },
                  { label: '临期', value: 'imminent' }
                ].map(opt => (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => setRuleForm({ ...ruleForm, ruleType: opt.value as 'warning' | 'imminent' })}
                    className={`flex-1 py-1.5 text-xs font-bold rounded-xl transition-all ${
                      ruleForm.ruleType === opt.value
                        ? 'bg-white text-zinc-900 shadow-sm'
                        : 'text-zinc-400 hover:text-zinc-600'
                    }`}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
              {ruleError && (
                <p className="text-[10px] font-bold text-rose-500 mt-1 animate-pulse">{ruleError}</p>
              )}
            </div>
            <div className="space-y-1.5 flex-1 min-w-0">
              <label className="block text-xs font-bold text-zinc-400 uppercase">适用范围</label>
              <div className="flex bg-zinc-100 p-1 rounded-2xl border border-zinc-200">
                {[
                  { label: '通用', value: 'general' },
                  { label: '特定（待绑定）', value: 'specific' }
                ].map(opt => (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => setRuleForm({ ...ruleForm, scopeType: opt.value as 'general' | 'specific' })}
                    className={`flex-1 py-1.5 text-xs font-bold rounded-xl transition-all ${
                      ruleForm.scopeType === opt.value
                        ? 'bg-white text-zinc-900 shadow-sm'
                        : 'text-zinc-400 hover:text-zinc-600'
                    }`}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {ruleForm.scopeType === 'specific' && <p className="text-xs text-amber-700">订单绑定尚未接入，特定规则可保存为草稿，当前不会触发订单告警。</p>}
          <p className="text-xs text-zinc-500">支持进度上下限比较，交货日期与订单交期含义相同；其他公式暂不支持。</p>
          {/* Preview Effect Section */}
          <div className="bg-zinc-50 border border-zinc-200 rounded-2xl p-4 space-y-3 shadow-inner">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold text-zinc-500 uppercase tracking-widest flex items-center gap-1.5">
                <Eye className="w-3.5 h-3.5" />
                预览效果
              </span>
              <div className="text-[10px] text-zinc-400 font-medium bg-zinc-100 px-2 py-0.5 rounded-full flex items-center gap-1">
                <Clock className="w-3 h-3" /> 当天: {businessToday()}
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 items-center">
              <div className="space-y-1">
                <span className="text-[10px] font-bold text-zinc-400 ml-1">模拟交货日期</span>
                <input
                  type="date"
                  value={previewValues.deliveryDate}
                  onChange={e => setPreviewValues({ ...previewValues, deliveryDate: e.target.value })}
                  className="w-full bg-white border border-zinc-200 rounded-xl px-3 py-1.5 text-xs outline-none focus:ring-2 focus:ring-zinc-900 transition-all font-medium"
                />
              </div>
              <div className="space-y-1">
                <span className="text-[10px] font-bold text-zinc-400 ml-1">模拟订单日期</span>
                <input
                  type="date"
                  value={previewValues.orderDate}
                  onChange={e => setPreviewValues({ ...previewValues, orderDate: e.target.value })}
                  className="w-full bg-white border border-zinc-200 rounded-xl px-3 py-1.5 text-xs outline-none focus:ring-2 focus:ring-zinc-900 transition-all font-medium"
                />
              </div>
              <div className="space-y-1">
                <span className="text-[10px] font-bold text-zinc-400 ml-1">模拟零件状态</span>
                <select
                  value={previewValues.partStatus}
                  onChange={e => setPreviewValues({ ...previewValues, partStatus: e.target.value as 'pending' | 'processing' | 'completed' })}
                  className="w-full bg-white border border-zinc-200 rounded-xl px-3 py-1.5 text-xs outline-none focus:ring-2 focus:ring-zinc-900 transition-all font-medium appearance-none"
                >
                  <option value="pending">待加工</option>
                  <option value="processing">加工中</option>
                  <option value="completed">已完成</option>
                </select>
              </div>
              <div className="bg-zinc-900 rounded-xl p-3 flex flex-col justify-center">
                <span className="text-[10px] font-bold text-zinc-300 uppercase leading-none mb-1 text-center">计算结果</span>
                <div className="text-lg font-mono font-bold text-white text-center truncate">
                  {calculatePreviewResult(ruleForm.formula, ruleForm.target_status)}
                </div>
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <label className="block text-xs font-bold text-zinc-400 uppercase">自定义公式编辑器</label>

            {/* Formula Output */}
            <div className="relative">
              <textarea
                readOnly
                value={ruleForm.formula}
                placeholder="点击下方按钮组合公式..."
                className="w-full flex-1 min-h-[60px] max-h-[120px] px-4 py-2 bg-zinc-100 border border-zinc-200 rounded-2xl font-mono text-lg text-zinc-700 resize-none outline-none transition-all pr-20"
              />
              <div className="absolute right-2 top-2 flex items-center gap-1.5">
                <button
                  type="button"
                  onClick={() => setRuleForm({ ...ruleForm, formula: ruleForm.formula.slice(0, -1) })}
                  className="p-1 px-2 bg-white text-zinc-500 rounded-lg shadow-sm border border-zinc-200 hover:bg-zinc-100 transition-colors flex items-center gap-1"
                  title="退格"
                >
                  <ChevronLeft className="w-3.5 h-3.5" />
                  <span className="text-[10px] font-bold">退格</span>
                </button>
                <button
                  type="button"
                  onClick={() => setRuleForm({ ...ruleForm, formula: '' })}
                  className="p-1.5 bg-white text-rose-500 rounded-lg shadow-sm border border-zinc-200 hover:bg-rose-50 transition-colors"
                  title="清空"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>

            {/* Logic Combination Row */}
            <div className="flex items-center gap-3 bg-zinc-50 p-3 rounded-2xl border border-zinc-100 shadow-inner">
              <div className="bg-zinc-900 px-4 py-2 rounded-xl text-white font-bold shadow-sm shadow-zinc-200">
                无              </div>
              <div className="flex-1">
                <div className="flex items-center gap-3">
                  <span className="text-[10px] font-bold text-zinc-400 uppercase tracking-widest whitespace-nowrap">零件状态为</span>
                  <select
                    value={ruleForm.target_status}
                    onChange={e => setRuleForm({ ...ruleForm, target_status: e.target.value as 'pending' | 'processing' | 'completed' })}
                    className="flex-1 bg-white border border-zinc-200 rounded-xl px-4 py-2 text-sm outline-none focus:ring-2 focus:ring-zinc-900 transition-all font-medium appearance-none shadow-sm cursor-pointer"
                  >
                    <option value="pending">待加工</option>
                    <option value="processing">加工中</option>
                    <option value="completed">已完成</option>
                  </select>
                </div>
              </div>
            </div>

            {/* Editor Controls */}
            <div className="bg-zinc-50 p-4 rounded-2xl border border-zinc-100 shadow-inner space-y-5 flex-1 flex flex-col min-h-0">
              {/* Top Row: Variables and Numbers */}
              <div className="grid grid-cols-2 gap-6 items-start flex-1 min-h-0">
                {/* Left: Factors */}
                <div className="space-y-2 flex flex-col h-full">
                  <span className="text-[10px] font-bold text-zinc-400 uppercase tracking-widest block">变量因子</span>
                  <div className="flex flex-wrap gap-2 content-start flex-1">
                    {['交货日期', '订单日期', '当天'].map(factor => (
                      <button
                        key={factor}
                        type="button"
                        onClick={() => setRuleForm({ ...ruleForm, formula: ruleForm.formula + `{${factor}}` })}
                        className="px-3 py-1.5 bg-indigo-600 text-white text-xs font-bold rounded-xl hover:bg-indigo-700 transition-colors shadow-sm flex-1 min-w-[80px]"
                      >
                        {factor}
                      </button>
                    ))}
                  </div>
                </div>

                {/* Right: Numbers */}
                <div className="space-y-2 flex flex-col h-full">
                  <div className="flex items-center justify-between">
                    <span className="text-[10px] font-bold text-zinc-400 uppercase tracking-widest">数字键盘</span>
                  </div>
                  <div className="bg-white p-2.5 rounded-2xl border border-zinc-200/50 grid grid-cols-4 gap-2 flex-1">
                    {['1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '.'].map((btn) => (
                      <button
                        key={btn}
                        type="button"
                        onClick={() => setRuleForm({ ...ruleForm, formula: ruleForm.formula + btn })}
                        className={`flex items-center justify-center rounded-lg font-bold transition-all text-sm h-full min-h-[36px] ${
                          btn === '.' ? 'bg-zinc-100 text-zinc-600' : 'text-zinc-600 hover:bg-zinc-900 hover:text-white'
                        }`}
                      >
                        {btn}
                      </button>
                    ))}
                  </div>
                </div>
              </div>

              {/* Bottom Row: Arithmetic and Logic (Aligned) */}
              <div className="grid grid-cols-2 gap-6 items-start flex-1 min-h-0">
                {/* Left: Arithmetic */}
                <div className="space-y-2 flex flex-col h-full">
                  <span className="text-[10px] font-bold text-zinc-400 uppercase tracking-widest block">基础运算符号</span>
                  <div className="grid grid-cols-3 gap-2 flex-1">
                    {['+', '-', '*', '/', '(', ')'].map((btn) => (
                      <button
                        key={btn}
                        type="button"
                        onClick={() => setRuleForm({ ...ruleForm, formula: ruleForm.formula + btn })}
                        className="flex items-center justify-center rounded-xl font-bold bg-zinc-900 text-white border border-zinc-900 hover:bg-zinc-800 transition-all shadow-sm text-sm h-full min-h-[40px]"
                      >
                        {btn}
                      </button>
                    ))}
                  </div>
                </div>

                {/* Right: Logic */}
                <div className="space-y-2 flex flex-col h-full">
                  <span className="text-[10px] font-bold text-zinc-400 uppercase tracking-widest block">数学逻辑判断</span>
                  <div className="grid grid-cols-3 gap-2 flex-1">
                    {['<', '<=', '>', '>=', '=', '!='].map((btn) => (
                      <button
                        key={btn}
                        type="button"
                        onClick={() => setRuleForm({ ...ruleForm, formula: ruleForm.formula + btn })}
                        className="flex items-center justify-center rounded-xl font-bold bg-zinc-900 text-white border border-zinc-900 hover:bg-zinc-800 transition-all shadow-sm text-sm h-full min-h-[40px]"
                      >
                        {btn}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>

          {ruleError && (
            <motion.div
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              className="p-3 bg-rose-50 border border-rose-100 rounded-xl flex items-center gap-2 text-rose-600 mb-2"
            >
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span className="text-xs font-bold leading-tight">{ruleError}</span>
            </motion.div>
          )}

          <div className="pt-2 flex gap-3">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-6 py-3 border border-zinc-200 text-zinc-600 rounded-2xl font-bold hover:bg-zinc-50 transition-colors"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={!!ruleError}
              className={`flex-[2] py-3 rounded-2xl font-bold transition-colors shadow-lg shadow-zinc-200 ${
                ruleError
                  ? 'bg-zinc-200 text-zinc-400 cursor-not-allowed'
                  : 'bg-zinc-900 text-white hover:bg-zinc-800'
              }`}
            >
              {editingRuleId ? '保存修改' : '创建规则'}
            </button>
          </div>
        </form>
      </motion.div>
    </div>
  );
};

export default RuleModal;
