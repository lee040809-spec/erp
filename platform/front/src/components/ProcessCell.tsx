import React, { useState } from 'react';
import { Settings, X, Trash2, ArrowUp, ArrowDown } from 'lucide-react';
import { PROCESS_COLORS } from './shared';
const PROCESS_OPTIONS = ['下料', '车削', '铣削', '磨削', '线切割', '电火花', '热处理', '表面处理', '送货'];
import type { OrderProcess as Process } from '../types';
import { processDate } from '../utils/processDates';
interface ProcessCellProps {
    processes: Process[];
    onUpdate: (processes: Process[]) => void;
    hideCostFields?: boolean;
    routeOnly?: boolean;
    inline?: boolean;
    allowProcessedChanges?: boolean;
    onManageProcessed?: () => void;
}
const ProcessCell: React.FC<ProcessCellProps> = ({ processes, onUpdate, hideCostFields = false, routeOnly = false, inline = false, allowProcessedChanges = false, onManageProcessed }) => {
    const [isOpen, setIsOpen] = useState(false);
    const [manualInput, setManualInput] = useState('');
    const [editingIndex, setEditingIndex] = useState<number | null>(null);
    // 检查是否有送货工序
    const hasDelivery = processes.some(p => p.name === '送货');
    const addProcess = (name: string) => {
        if (!name.trim())
            return;
        // 送货工序只能有一个
        if (name === '送货' && processes.some(p => p.name === '送货')) {
            return;
        }
        // 新增工序时不传 id，让后端自动生成
        const newProcess: Process = {
            name: name.trim(),
            is_outsourced: false,
            outsourcing_fee: 0,
            status: 'pending'
        };
        const newProcesses = [...processes, newProcess];
        onUpdate(newProcesses);
        setManualInput('');
    };
    const move = (index: number, direction: number) => {
        const next = [...processes], target = index + direction;
        if (target < 0 || target >= next.length) return;
        [next[index], next[target]] = [next[target], next[index]];
        onUpdate(next);
    };
    return (<div className="relative">
      {!inline && <button type="button" aria-label="增删工序" onClick={() => setIsOpen(!isOpen)} className={`w-full px-2 py-1.5 text-left rounded-lg transition-colors flex items-center justify-between gap-1 min-w-[100px] ${hasDelivery
            ? 'bg-zinc-50 border border-zinc-100 hover:bg-zinc-100'
            : 'bg-red-50 border border-red-300 hover:bg-red-100'}`}>
        <span className={`flex-1 truncate text-xs font-medium ${hasDelivery ? 'text-zinc-600' : 'text-red-600'}`}>
          {processes.length > 0 ? processes.map(p => p.name).join('、') : '点击添加工序'}
          {!hasDelivery && processes.length > 0 && <span className="ml-1">（缺少送货）</span>}
        </span>
        <span className="shrink-0 text-xs font-semibold text-blue-600">增删工序</span>
      </button>}

      {(isOpen || inline) && (<>
          {!inline && <div className="fixed inset-0 z-[200] bg-zinc-900/20 backdrop-blur-[2px]" onClick={() => setIsOpen(false)}/>}
          <div className={inline ? "space-y-4" : "fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-[201] bg-white border border-zinc-200 rounded-3xl shadow-2xl p-5 sm:p-8 w-[800px] max-w-[95vw] max-h-[90vh] overflow-auto space-y-6"}>
            <div className="flex items-center justify-between border-b border-zinc-100 pb-4">
              <div className="flex items-center gap-2">
                <Settings className="w-5 h-5 text-zinc-900"/>
                <h5 className="text-xl font-bold text-zinc-900 uppercase tracking-tight">修改 / 添加 / 删除工序</h5>
              </div>
              {!inline && <button type="button" aria-label="关闭工序流程管理" onClick={() => setIsOpen(false)} className="p-2 hover:bg-zinc-100 rounded-full transition-colors">
                <X className="w-6 h-6 text-zinc-400"/>
              </button>}
            </div>

            <div className="flex flex-wrap gap-2">
              {PROCESS_OPTIONS.map(opt => {
                const isDisabled = opt === '送货' && hasDelivery;
                return (<button key={opt} type="button" onClick={() => addProcess(opt)} disabled={isDisabled} className={`px-3 py-1.5 rounded text-sm font-bold transition-all hover:scale-105 ${isDisabled ? 'opacity-50 cursor-not-allowed' : ''} ${PROCESS_COLORS[opt] || 'bg-zinc-100 text-zinc-600'}`}>
                    + {opt} {opt === '送货' && hasDelivery ? '(已添加)' : ''}
                  </button>);
            })}
              <div className="flex items-center gap-2 ml-4 border-l border-zinc-200 pl-4">
                <input type="text" aria-label="自定义新增工序名称" maxLength={100} placeholder="输入新增工序名称..." value={manualInput} onChange={(e) => setManualInput(e.target.value)} onKeyDown={(e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    addProcess(manualInput);
                }
            }} className="px-3 py-1.5 bg-zinc-50 border border-zinc-200 rounded text-sm outline-none focus:ring-2 focus:ring-zinc-900 w-48"/>
                <button type="button" onClick={() => addProcess(manualInput)} className="px-3 py-1.5 bg-zinc-900 text-white rounded text-sm font-bold hover:bg-zinc-800 transition-colors">
                  新增工序
                </button>
              </div>
            </div>

            <p className="text-xs text-slate-500 leading-5">选择常用工序或自定义名称可新增；“修改工序”可调整名称、计划与内制/外协安排。{allowProcessedChanges ? "加工中、已完成工序也可删除，保存后归档并保留历史；状态仍由实际报工计算。" : "已有加工记录的工序请通过下方专项入口，或在工艺窗口勾选允许修改 / 删除已加工工序。"}修改后请点击保存。</p>
            {onManageProcessed && <button type="button" className="erp-btn text-blue-600" onClick={()=>{setIsOpen(false);onManageProcessed();}}>修改 / 删除已加工工序</button>}
            <div className="space-y-2 pr-1">
              {processes.map((p, idx) => (<div key={p.id || `new-${idx}`} className="flex flex-wrap items-center gap-2 bg-zinc-50 p-3 rounded-lg border border-zinc-100">
                  <span className="text-xs text-slate-400">{idx + 1}</span><input aria-label={`工序${idx + 1}名称`} value={p.name} maxLength={100} onChange={e => onUpdate(processes.map((step,n) => n === idx ? {...step,name:e.target.value} : step))} className="erp-input w-36 font-semibold"/>
                  <label className="flex items-center gap-2 cursor-pointer bg-white px-2 py-1 rounded border border-zinc-200">
                    <input type="checkbox" checked={p.is_outsourced} disabled={!allowProcessedChanges && (!!p.actual_start_at || !!p.actual_end_at || p.status !== "pending")} onChange={e => {
                    const newProcesses = [...processes];
                    newProcesses[idx] = { ...newProcesses[idx], is_outsourced: e.target.checked };
                    onUpdate(newProcesses);
                }} className="w-4 h-4 rounded border-zinc-300 text-zinc-900 focus:ring-zinc-900"/>
                    <span className="text-sm font-medium text-zinc-700">是否外协</span>
                  </label>
                  {p.is_outsourced && !hideCostFields && (<div className="flex items-center gap-1">

                      <input type="number" placeholder="外协费用" aria-label={`${p.name}外协费用`} value={p.outsourcing_fee || ''} onChange={e => {
                        const newProcesses = [...processes];
                        newProcesses[idx] = { ...newProcesses[idx], outsourcing_fee: parseFloat(e.target.value) };
                        onUpdate(newProcesses);
                    }} className="w-24 px-2 py-1 bg-white border border-zinc-200 rounded text-sm outline-none font-medium"/>
                    </div>)}
                  <div className="flex-1"/>
                  <select aria-label={`工序${idx + 1}生产状态`} disabled={routeOnly} value={p.status || 'pending'} onChange={e => {
                    const newProcesses = [...processes];
                    newProcesses[idx] = { ...newProcesses[idx], status: e.target.value as Process['status'] };
                    onUpdate(newProcesses);
                }} className="px-3 py-1.5 bg-white border border-zinc-200 rounded-lg text-sm outline-none font-medium">
                    <option value="pending">待加工</option>
                    <option value="processing">加工中</option>
                    <option value="completed">已完成</option>
                  </select>
                  <button type="button" className="erp-btn py-1 min-h-8" aria-label={`上移工序${idx + 1}`} disabled={idx === 0} onClick={() => move(idx,-1)}><ArrowUp size={14}/></button>
                  <button type="button" className="erp-btn py-1 min-h-8" aria-label={`下移工序${idx + 1}`} disabled={idx === processes.length - 1} onClick={() => move(idx,1)}><ArrowDown size={14}/></button>
                  <button type="button" aria-label={`修改工序${idx + 1}`} className="erp-btn text-blue-600 py-1 min-h-8" onClick={()=>setEditingIndex(editingIndex===idx?null:idx)}>修改工序</button>
                  <button type="button" aria-label={`删除工序${idx + 1}`} title={allowProcessedChanges ? "从当前工艺移除并保留历史记录，保存后生效" : p.actual_start_at || p.actual_end_at || p.status !== "pending" ? "请先勾选允许修改 / 删除已加工工序" : "删除此工序及其计划安排，保存后生效"} disabled={!allowProcessedChanges && (!!p.actual_start_at || !!p.actual_end_at || p.status !== "pending")} onClick={() => {
                    setEditingIndex(null);
                    onUpdate(processes.filter((_,n) => n !== idx));
                }} className="erp-btn text-rose-600 border-rose-200 py-1 min-h-8 disabled:opacity-50">
                    <Trash2 size={14}/>删除工序
                  </button>
                  {editingIndex===idx && <div className="w-full grid grid-cols-2 gap-3 border-t pt-3"><p className="col-span-2 text-xs text-slate-500">名称和是否外协在本行修改；实际开工 / 完工时间请在工序时间窗口中更正，已有报工数量不会被清空。</p>{(['planned_start_at','planned_end_at'] as const).map((field,n)=><label key={field} className="erp-label">{n===0?'计划开始日期':'计划结束日期'}<input type="date" aria-label={'工序'+(idx+1)+(n===0?'计划开始日期':'计划结束日期')} className="erp-input" value={processDate(p[field])} onChange={e=>onUpdate(processes.map((step,i)=>i===idx?{...step,[field]:e.target.value}:step))}/></label>)}</div>}
                </div>))}
              {processes.length === 0 && (<div className="text-center py-12 text-sm text-zinc-400 italic bg-zinc-50 rounded-xl border border-dashed border-zinc-200">
                  暂无工序，请从上方选择并添加
                </div>)}
            </div>
          </div>
        </>)}
    </div>);
};
export default ProcessCell;
