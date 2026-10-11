import { useEffect, useRef, useState } from 'react';
import { Menu, X, ChevronLeft, ChevronRight, ChevronUp, Lock, KeyRound, Mail, LogOut, Factory } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
interface NavItem {
    id: string;
    label: string;
    icon: LucideIcon;
}
interface SidebarProps {
    navItems: NavItem[];
    activeTab: string;
    setActiveTab: (tab: string) => void;
    isSidebarOpen: boolean;
    setIsSidebarOpen: (open: boolean) => void;
    isSidebarCollapsed: boolean;
    setIsSidebarCollapsed: (collapsed: boolean) => void;
    authUser: {
        id?: number;
        username: string;
        name: string;
        role_name: string;
        corp_name: string;
        email?: string;
        expired_at?: string;
    } | null;
    onLogout: () => void;
    onOpenSettings: (type: 'password' | 'reset' | 'email') => void;
}
const groupFor = (id: string) => /dashboard|production|overdue|warning|near_due|imminent/.test(id) ? '生产协同' : /order|delivery/.test(id) ? '订单与交付' : /finance|output|reconciliation|cost/.test(id) ? '经营核算' : /customer|material|remnant|inventory|outsourcing_supplier/.test(id) ? '基础资料' : '系统管理';
export default function Sidebar(p: SidebarProps) {
    const [dropdown, setDropdown] = useState(false), ref = useRef<HTMLDivElement>(null);
    useEffect(() => {
        const close = (e: MouseEvent) => {
            if (ref.current && !ref.current.contains(e.target as Node))
                setDropdown(false);
        };
        document.addEventListener('mousedown', close);
        return () => document.removeEventListener('mousedown', close);
    }, []);
    const [isPhone, setIsPhone] = useState(() => window.matchMedia('(max-width: 767px)').matches);
    useEffect(() => {
        const media = window.matchMedia('(max-width: 767px)');
        const update = () => setIsPhone(media.matches);
        media.addEventListener('change', update);
        return () => media.removeEventListener('change', update);
    }, []);
    const collapsed = p.isSidebarCollapsed && !isPhone;
    useEffect(() => {
        if (!p.isSidebarOpen) return;
        const closeWithEscape = (event: KeyboardEvent) => {
            if (event.key === 'Escape') p.setIsSidebarOpen(false);
        };
        document.addEventListener('keydown', closeWithEscape);
        return () => document.removeEventListener('keydown', closeWithEscape);
    }, [p.isSidebarOpen, p.setIsSidebarOpen]);
    return <>
  <header className="erp-mobile-header md:hidden fixed top-0 inset-x-0 h-14 bg-white border-b border-slate-200 flex items-center justify-between px-4 z-40"><span className="flex items-center gap-2 font-bold text-slate-900"><Factory size={22} className="text-blue-600"/>机加 ERP</span><button aria-label="打开导航" onClick={() => p.setIsSidebarOpen(true)}><Menu /></button></header>
  {p.isSidebarOpen && <button aria-label="关闭导航遮罩" className="fixed inset-0 bg-slate-950/40 z-40 md:hidden" onClick={() => p.setIsSidebarOpen(false)}/>}
  <aside className={`erp-sidebar ${collapsed ? 'is-collapsed' : ''} ${p.isSidebarOpen ? 'is-open' : ''}`}>
   <div className="flex items-center gap-3 px-5 py-6 border-b border-white/10 relative"><div className="w-10 h-10 rounded-2xl bg-blue-500 flex items-center justify-center shrink-0 shadow-lg shadow-blue-950/40"><Factory className="text-white" size={23}/></div>{!collapsed && <div><strong className="block text-lg text-white tracking-wide">机加 ERP</strong><span className="text-[11px] text-slate-400">生产协同 · 经营可见</span></div>}<button aria-label="关闭导航" className="md:hidden ml-auto text-slate-300" onClick={() => p.setIsSidebarOpen(false)}><X size={20}/></button><button aria-label={collapsed ? '展开导航' : '收起导航'} onClick={() => p.setIsSidebarCollapsed(!p.isSidebarCollapsed)} className="hidden md:flex absolute -right-3 top-9 h-6 w-6 items-center justify-center rounded-full bg-white border border-slate-200 text-slate-600 shadow">{collapsed ? <ChevronRight size={14}/> : <ChevronLeft size={14}/>}</button></div>
   {!collapsed && <div className="mx-4 mt-4 mb-1 rounded-xl bg-white/5 px-3 py-2 text-xs text-slate-300 truncate" title={p.authUser?.corp_name}>{p.authUser?.corp_name || '企业工作空间'}</div>}
   <nav aria-label="主导航" className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden px-3 py-3 space-y-4">{['生产协同', '订单与交付', '经营核算', '基础资料', '系统管理'].map(group => { const items = p.navItems.filter(i => groupFor(i.id) === group); return items.length ? <div key={group}>{!collapsed && <p className="text-[10px] font-semibold text-slate-500 tracking-widest px-3 mb-2">{group}</p>}<div className="space-y-1">{items.map(i => <button key={i.id} title={i.label} aria-current={p.activeTab === i.id ? 'page' : undefined} onClick={() => { p.setActiveTab(i.id); p.setIsSidebarOpen(false); }} className={`erp-nav ${p.activeTab === i.id ? 'is-active' : ''} ${collapsed ? 'justify-center' : ''}`}><i.icon size={18} className="shrink-0"/>{!collapsed && <span className="truncate">{i.label}</span>}</button>)}</div></div> : null; })}</nav>
   <div ref={ref} className="relative px-3 py-4 border-t border-white/10">{dropdown && <div className="erp-sidebar-account-menu absolute bottom-full left-3 w-60 mb-2 p-2 rounded-2xl shadow-2xl bg-white text-slate-700 border border-slate-200 z-50"><div className="px-3 py-2 text-xs leading-6 border-b mb-1 border-slate-100"><strong className="block text-slate-900">{p.authUser?.username}</strong>姓名：{p.authUser?.name || '未填写'}<br />角色：{p.authUser?.role_name || '未设置'}{p.authUser?.expired_at && <p>有效期：{new Date(p.authUser.expired_at).toLocaleDateString('zh-CN')}</p>}</div>{([{ type: 'password', label: '修改密码', icon: Lock }, { type: 'reset', label: '密码重置', icon: KeyRound }, { type: 'email', label: '修改邮箱', icon: Mail }] as const).map(i => <button key={i.type} className="w-full flex items-center gap-3 px-3 py-2 hover:bg-slate-50 rounded-xl text-sm" onClick={() => { setDropdown(false); p.onOpenSettings(i.type); }}><i.icon size={16}/>{i.label}</button>)}<button className="w-full flex items-center gap-3 px-3 py-2 hover:bg-rose-50 rounded-xl text-sm text-rose-600" onClick={p.onLogout}><LogOut size={16}/>退出登录</button></div>}<button aria-label="账户设置" aria-expanded={dropdown} onClick={() => setDropdown(!dropdown)} className="w-full flex items-center gap-3 rounded-xl p-2 hover:bg-white/5 text-left"><div className="h-9 w-9 rounded-full bg-slate-700 text-blue-200 flex items-center justify-center shrink-0 font-semibold">{(p.authUser?.username || 'A').slice(0, 1).toUpperCase()}</div>{!collapsed && <><div className="min-w-0 flex-1"><p className="text-sm text-white font-medium truncate">{p.authUser?.username || '账户'}</p><p className="text-[11px] text-slate-400 truncate">{p.authUser?.role_name || '账户设置'}</p></div><ChevronUp size={16} className="text-slate-400"/></>}</button></div>
  </aside>
 </>;
}
