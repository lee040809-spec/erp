import ResponsiveTable from '../components/ResponsiveTable';
import { fetchFinanceAccess, FinanceAccess } from '../services/financeAccess';
import BusinessEntrySettings from '../components/BusinessEntrySettings';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { AlertCircle, Download, RefreshCw, TrendingDown, TrendingUp, WalletCards } from 'lucide-react';
import { AnnualFinanceSummary, FinanceSummaryResponse, MonthlyFinanceSummary, Reconciliation } from '../types';
import { fetchFinanceSummaryApi } from '../services/api';
import CostEntryModal from '../components/CostEntryModal';
import FinanceReviewPanel from '../components/FinanceReviewPanel';
interface FinanceProps {
    reconciliation?: Reconciliation[];
}
const emptyAnnual = (year: string): AnnualFinanceSummary => ({
    year,
    output_value: '0.00',
    production_cost: '0.00',
    production_profit: '0.00',
    period_expense: '0.00',
    operating_profit: '0.00'
});
const money = (value: string | number) => new Intl.NumberFormat('zh-CN', { style: 'currency', currency: 'CNY', minimumFractionDigits: 2 }).format(Number(value || 0));
const Finance: React.FC<FinanceProps> = () => {
    const currentYear = String(new Date().getFullYear());
    const [policyVersion, setPolicyVersion] = useState(0);
    const [access, setAccess] = useState<FinanceAccess | null>(null);
    const [year, setYear] = useState(currentYear);
    const [summary, setSummary] = useState<FinanceSummaryResponse>({ year: currentYear, months: [], annual: emptyAnnual(currentYear) });
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [showCostModal, setShowCostModal] = useState(false);
    const [detailsMonth, setDetailsMonth] = useState('');
    const [selectionVersion, setSelectionVersion] = useState(0);
    const loadVersion = useRef(0);
    const years = useMemo(() => {
        const y = Number(currentYear);
        return Array.from({ length: 6 }, (_, index) => String(y - index));
    }, [currentYear]);
    const load = async () => {
        const version = ++loadVersion.current;
        setLoading(true);
        setError('');
        try {
            const [data, permissions] = await Promise.all([fetchFinanceSummaryApi(year), fetchFinanceAccess()]);
            if (version === loadVersion.current) {
                setSummary(data);
                setAccess(permissions);
            }
        }
        catch (err) {
            if (version !== loadVersion.current)
                return;
            setAccess(null);
            setError(err instanceof Error ? err.message : '经营报表加载失败');
            setSummary({ year, months: [], annual: emptyAnnual(year) });
        }
        finally {
            if (version === loadVersion.current)
                setLoading(false);
        }
    };
    useEffect(() => {
        setDetailsMonth('');
        load();
        return () => { loadVersion.current++; };
    }, [year]);
    const exportCsv = () => {
        const header = ['月份', '合格数量', '计产值数量', '备用数量', '完工产值', '材料成本', '外协成本', '其他生产成本', '生产成本', '完工贡献', '期间费用', '经营贡献', '月结状态'];
        const rows = summary.months.map(item => [
            item.month, item.qualified_quantity, item.billable_quantity, item.reserve_quantity, item.output_value, item.material_cost,
            item.outsourcing_cost, item.other_production_cost, item.production_cost,
            item.production_profit, item.period_expense, item.operating_profit,
            item.closing_status === 'closed' ? '已结账' : '未结账'
        ]);
        const csv = '\ufeff' + [header, ...rows].map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(',')).join('\r\n');
        const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
        const link = document.createElement('a');
        link.href = url;
        link.download = `经营报表-${year}.csv`;
        link.click();
        URL.revokeObjectURL(url);
    };
    const cards = [
        { label: '年度合格完工产值', value: summary.annual.output_value, icon: WalletCards, color: 'text-blue-600', bg: 'bg-blue-50' },
        { label: '年度已分摊生产成本', value: summary.annual.production_cost, icon: TrendingDown, color: 'text-amber-600', bg: 'bg-amber-50' },
        { label: '年度完工贡献', value: summary.annual.production_profit, icon: TrendingUp, color: 'text-emerald-600', bg: 'bg-emerald-50' },
        { label: '年度经营贡献', value: summary.annual.operating_profit, icon: TrendingUp, color: Number(summary.annual.operating_profit) >= 0 ? 'text-emerald-600' : 'text-rose-600', bg: Number(summary.annual.operating_profit) >= 0 ? 'bg-emerald-50' : 'bg-rose-50' }
    ];
    return (<div className="erp-page">
      <div className="erp-finance-content px-4 md:px-8 py-6 space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 className="text-2xl font-bold text-zinc-900">产值与经营报表</h2>
            <p className="text-sm text-zinc-500 mt-1">产值按订单内合格数量 × 单价快照计算；超出的合格件作为备用件，不计产值</p>
          </div>
          <div className="flex items-center gap-2 flex-wrap">
            <BusinessEntrySettings onSaved={() => { load(); setPolicyVersion(v => v + 1); }}/>
            {access?.can_manage_costs && <button onClick={() => setShowCostModal(true)} className="h-10 border border-zinc-300 bg-white px-4 text-sm font-medium hover:bg-zinc-100">录入成本费用</button>}
            <select aria-label="报表年份" value={year} onChange={event => setYear(event.target.value)} className="h-10 border border-zinc-200 bg-white px-3 text-sm outline-none focus:ring-2 focus:ring-zinc-900">
              {years.map(item => <option value={item} key={item}>{item} 年</option>)}
            </select>
            <button onClick={load} disabled={loading} className="h-10 border border-zinc-200 bg-white px-3 text-sm font-medium hover:bg-zinc-100 disabled:opacity-50" title="刷新">
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`}/>
            </button>
            <button onClick={exportCsv} disabled={loading || !!error || summary.months.length === 0} className="h-10 bg-zinc-900 text-white px-4 text-sm font-medium flex items-center gap-2 hover:bg-zinc-800 disabled:opacity-40">
              <Download className="w-4 h-4"/>导出 CSV
            </button>
          </div>
        </div>

        {error && (<div role="alert" className="border border-rose-200 bg-rose-50 text-rose-700 px-4 py-3 flex items-center gap-2 text-sm">
            <AlertCircle className="w-4 h-4"/>{error}
          </div>)}

        <div className="erp-finance-cards grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
          {cards.map(card => (<div key={card.label} className="erp-card p-5">
              <div className={`w-10 h-10 ${card.bg} ${card.color} flex items-center justify-center mb-4`}><card.icon className="w-5 h-5"/></div>
              <p className="text-sm text-zinc-500">{card.label}</p>
              <p className={`text-2xl font-bold mt-1 ${card.color}`}>{loading ? '读取中…' : error ? '未读取' : money(card.value)}</p>
            </div>))}
        </div>

        <div className="erp-card overflow-hidden">
          <div className="px-5 py-4 border-b border-zinc-200">
            <h3 className="font-bold">月度明细</h3>
            <p className="text-xs text-zinc-500 mt-1">生产成本按分摊的完工批次月份归集，期间费用按业务月份计入。费用未录全或未分摊时，贡献金额为暂估，不能当作财务利润。点击月份查看来源明细。</p>
          </div>
          <div className="erp-table-scroll overflow-x-auto">
            <ResponsiveTable className="w-full min-w-[1180px] text-sm">
              <thead className="bg-zinc-50 text-zinc-500">
                <tr>
                  {['月份', '合格数量', '计产值数量', '备用数量', '完工产值', '材料成本', '外协成本', '其他生产成本', '生产成本', '完工贡献', '期间费用', '经营贡献', '状态'].map(label => (<th key={label} className="px-4 py-3 text-right first:text-left font-medium">{label}</th>))}
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100">
                {!loading && !error && summary.months.map((item: MonthlyFinanceSummary) => (<tr key={item.month} className="hover:bg-zinc-50">
                    <td className="px-4 py-3 font-semibold"><button className="text-blue-700 underline" onClick={() => { setDetailsMonth(item.month); setSelectionVersion(v => v + 1); }}>{item.month}</button></td>
                    <td className="px-4 py-3 text-right">{item.qualified_quantity}</td><td className="px-4 py-3 text-right">{item.billable_quantity}</td><td className="px-4 py-3 text-right text-violet-600">{item.reserve_quantity}</td>
                    <td className="px-4 py-3 text-right font-semibold">{money(item.output_value)}</td>
                    <td className="px-4 py-3 text-right">{money(item.material_cost)}</td>
                    <td className="px-4 py-3 text-right">{money(item.outsourcing_cost)}</td>
                    <td className="px-4 py-3 text-right">{money(item.other_production_cost)}</td>
                    <td className="px-4 py-3 text-right">{money(item.production_cost)}</td>
                    <td className={`px-4 py-3 text-right font-semibold ${Number(item.production_profit) >= 0 ? 'text-emerald-600' : 'text-rose-600'}`}>{money(item.production_profit)}</td>
                    <td className="px-4 py-3 text-right">{money(item.period_expense)}</td>
                    <td className={`px-4 py-3 text-right font-bold ${Number(item.operating_profit) >= 0 ? 'text-emerald-600' : 'text-rose-600'}`}>{money(item.operating_profit)}</td>
                    <td className="px-4 py-3 text-right"><span className={`px-2 py-1 text-xs ${item.closing_status === 'closed' ? 'bg-zinc-900 text-white' : 'bg-amber-50 text-amber-700 border border-amber-200'}`}>{item.closing_status === 'closed' ? '已结账' : '未结账'}</span></td>
                  </tr>))}
              </tbody>
            </ResponsiveTable>
          </div>
          {!loading && !error && summary.months.length === 0 && (<div className="py-16 text-center text-zinc-500">
              <p className="font-medium">本年度暂无已确认的合格完工批次</p>
              <p className="text-xs mt-2">历史“已完成”状态不会自动计入精确产值，需要审核后再迁移。</p>
            </div>)}
          {loading && <div role="status" className="py-16 text-center text-zinc-400">正在读取经营数据…</div>}
        </div>
        {!loading && !error && <div className="border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">全公司未分摊生产费用：{money(summary.pending_cost_amount || '0')}（{summary.pending_cost_count || 0} 笔，跨所有年份）。未分摊费用暂不扣减完工贡献，请先在下方完成成本分摊。</div>}
        <p className="rounded-xl bg-blue-50 p-4 text-sm text-blue-800">日常产值：在生产工作台确认合格数量和日期，系统自动汇总；历史数据只需补录合格数量和日期，默认使用订单单价。凭证类信息默认选填。</p>
        {!error && access && <FinanceReviewPanel access={access} key={policyVersion} year={year} month={detailsMonth} selectionVersion={selectionVersion} onImported={load}/>}
        {showCostModal && access?.can_manage_costs && <CostEntryModal onClose={() => setShowCostModal(false)} onSuccess={load}/>}
      </div>
    </div>);
};
export default Finance;
