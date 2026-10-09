import React from 'react';
import { LayoutDashboard, Users, Shield, Settings, TrendingUp } from 'lucide-react';

interface DashboardProps {
  user: any;
}

const Dashboard: React.FC<DashboardProps> = ({ user }) => {
  const stats = [
    { icon: Users, label: '用户总数', value: '12', color: 'bg-blue-500' },
    { icon: Shield, label: '角色总数', value: '4', color: 'bg-green-500' },
    { icon: Settings, label: '资源总数', value: '25', color: 'bg-orange-500' },
    { icon: TrendingUp, label: '今日登录', value: '8', color: 'bg-purple-500' },
  ];

  return (
    <div className="p-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-zinc-900 flex items-center gap-3">
          <LayoutDashboard className="w-8 h-8 text-blue-600" />
          仪表盘
        </h1>
        <p className="text-zinc-500 mt-1">欢迎回来，{user?.username}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {stats.map((stat, idx) => (
          <div key={idx} className="bg-white p-6 rounded-2xl shadow-sm border border-zinc-100">
            <div className={`w-12 h-12 ${stat.color} rounded-xl flex items-center justify-center mb-4`}>
              <stat.icon className="w-6 h-6 text-white" />
            </div>
            <div className="text-2xl font-bold text-zinc-900">{stat.value}</div>
            <div className="text-zinc-500 text-sm">{stat.label}</div>
          </div>
        ))}
      </div>

      <div className="mt-8 bg-white p-6 rounded-2xl shadow-sm border border-zinc-100">
        <h2 className="text-lg font-bold text-zinc-900 mb-4">系统信息</h2>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div className="flex justify-between py-2 border-b border-zinc-100">
            <span className="text-zinc-500">当前版本</span>
            <span className="font-medium">Portal v1.0.0</span>
          </div>
          <div className="flex justify-between py-2 border-b border-zinc-100">
            <span className="text-zinc-500">当前角色</span>
            <span className="font-medium">{user?.role_name}</span>
          </div>
          <div className="flex justify-between py-2 border-b border-zinc-100">
            <span className="text-zinc-500">登录时间</span>
            <span className="font-medium">{new Date().toLocaleString()}</span>
          </div>
          <div className="flex justify-between py-2 border-b border-zinc-100">
            <span className="text-zinc-500">系统状态</span>
            <span className="text-green-600 font-medium">正常运行</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;