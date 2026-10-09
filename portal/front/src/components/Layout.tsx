import React, { useState, useEffect } from 'react';
import { LayoutDashboard, Users, Shield, Settings, Building2, Menu, X, LogOut, ChevronDown } from 'lucide-react';
import { Resource } from '../types';

interface LayoutProps {
  children: React.ReactNode;
  user: any;
  menus: Resource[];
  onLogout: () => void;
  onPageChange: (page: string) => void;
  currentPage: string;
}

const Layout: React.FC<LayoutProps> = ({ children, user, menus, onLogout, onPageChange, currentPage }) => {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  const iconMap: Record<string, React.FC<{ className?: string }>> = {
    '仪表盘': LayoutDashboard,
    '用户管理': Users,
    '角色管理': Shield,
    '资源管理': Settings,
    '公司管理': Building2,
  };

  return (
    <div className="min-h-screen bg-zinc-50 flex">
      {/* Sidebar */}
      <aside className={`${sidebarOpen ? 'w-64' : 'w-20'} bg-zinc-900 text-white transition-all duration-300 flex flex-col`}>
        <div className="p-4 border-b border-zinc-800 flex items-center justify-between">
          {sidebarOpen && <h1 className="text-xl font-bold">Portal</h1>}
          <button onClick={() => setSidebarOpen(!sidebarOpen)} className="p-2 hover:bg-zinc-800 rounded-lg">
            {sidebarOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
        </div>

        <nav className="flex-1 p-2">
          {menus.map((menu) => {
            const Icon = iconMap[menu.name] || LayoutDashboard;
            return (
              <button
                key={menu.id}
                onClick={() => onPageChange(menu.path)}
                className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl mb-1 transition-colors ${
                  currentPage === menu.path ? 'bg-blue-600 text-white' : 'hover:bg-zinc-800 text-zinc-300'
                }`}
              >
                <Icon className="w-5 h-5" />
                {sidebarOpen && <span>{menu.name}</span>}
              </button>
            );
          })}
        </nav>

        <div className="p-4 border-t border-zinc-800">
          <div className="relative">
            <button
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="w-full flex items-center gap-3 px-3 py-2 rounded-xl hover:bg-zinc-800 transition-colors"
            >
              <div className="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center">
                {user?.username?.charAt(0).toUpperCase()}
              </div>
              {sidebarOpen && (
                <>
                  <span className="flex-1 text-left text-sm">{user?.username}</span>
                  <ChevronDown className="w-4 h-4" />
                </>
              )}
            </button>

            {userMenuOpen && sidebarOpen && (
              <div className="absolute bottom-full left-0 w-full mb-2 bg-zinc-800 rounded-xl overflow-hidden shadow-lg">
                <button
                  onClick={() => {
                    onLogout();
                    setUserMenuOpen(false);
                  }}
                  className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-zinc-700"
                >
                  <LogOut className="w-4 h-4" />
                  退出登录
                </button>
              </div>
            )}
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        {children}
      </main>
    </div>
  );
};

export default Layout;