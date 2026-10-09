import React, { useState, useEffect } from 'react';
import { Lock, User, AlertCircle, Loader2, Building2, X, ShieldCheck, Cog, Wrench } from 'lucide-react';

interface LoginPageProps {
  onLoginSuccess: (token: string, user: {
    id: number;
    username: string;
    name: string;
    email: string;
    role_name: string;
    corp_name: string;
    expired_at?: string;
  }) => void;
}

interface UserInfo {
  id: number;
  corp_id: number;
  corp_name: string;
  username: string;
  name: string;
  role_type: string;
}

const LoginPage: React.FC<LoginPageProps> = ({ onLoginSuccess }) => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [lockedInfo, setLockedInfo] = useState<{ locked: boolean; remainingMinutes?: number }>({ locked: false });

  // 公司信息
  const [corpInfo, setCorpInfo] = useState<UserInfo | null>(null);
  const [checkingUser, setCheckingUser] = useState(false);
  const [companyMessage, setCompanyMessage] = useState('输入账号后自动查询公司');

  // 错误弹框
  const [showErrorModal, setShowErrorModal] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [remainingAttempts, setRemainingAttempts] = useState<number | null>(null);

  // 初始动画状态
  const [initialAnimation, setInitialAnimation] = useState(true);

  // Check lock status on mount
  useEffect(() => {
    checkLockStatus();
  }, []);

  // 页面加载2秒后停止初始动画
  useEffect(() => {
    const timer = setTimeout(() => {
      setInitialAnimation(false);
    }, 2000);
    return () => clearTimeout(timer);
  }, []);

  // 用户名变化时查询公司信息
  useEffect(() => {
    const controller = new AbortController();
    const timer = setTimeout(() => {
      if (username.trim()) {
        checkUser(username.trim(), controller.signal);
      } else {
        setCorpInfo(null);
        setCompanyMessage('输入账号后自动查询公司');
        setCheckingUser(false);
      }
    }, 300);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [username]);

  const checkLockStatus = async () => {
    try {
      const res = await fetch('/api/platform/auth/status');
      const data = await res.json();
      if (data.locked) {
        setLockedInfo({ locked: true, remainingMinutes: data.remainingMinutes });
      }
    } catch (e) {
      // Ignore
    }
  };

  // 查询用户/公司信息
  const checkUser = async (name: string, signal: AbortSignal) => {
    setCheckingUser(true);
    setCorpInfo(null);
    setCompanyMessage('正在查询公司…');
    try {
      const res = await fetch('/api/platform/check-user', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: name }),
        signal
      });

      const data = await res.json();

      if (data.success && data.user) {
        setCorpInfo(data.user);
        setCompanyMessage('');
      } else {
        setCorpInfo(null);
        setCompanyMessage(data.error || '未查询到所属公司');
      }
    } catch (e) {
      if (e instanceof DOMException && e.name === 'AbortError') return;
      setCorpInfo(null);
      setCompanyMessage('公司查询服务暂不可用');
    } finally {
      if (!signal.aborted) setCheckingUser(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    if (lockedInfo.locked) return;
    if (!username.trim()) {
      setErrorMessage('请输入账号');
      setRemainingAttempts(null);
      setShowErrorModal(true);
      return;
    }
    if (!password.trim()) {
      setErrorMessage('请输入密码');
      setRemainingAttempts(null);
      setShowErrorModal(true);
      return;
    }

    // 检查公司是否存在
    if (!corpInfo) {
      setErrorMessage(companyMessage || '未查询到所属公司');
      setRemainingAttempts(null);
      setShowErrorModal(true);
      return;
    }

    setLoading(true);

    try {
      const res = await fetch('/api/platform/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username,
          password,
          corp_id: corpInfo?.corp_id
        })
      });

      const data = await res.json();

      if (data.success) {
        localStorage.setItem('auth_token', data.token);
        localStorage.setItem('auth_user', JSON.stringify(data.user));
        onLoginSuccess(data.token, data.user);
      } else {
        setErrorMessage(data.error || '登录失败');
        setRemainingAttempts(data.remaining_attempts ?? null);
        setShowErrorModal(true);

        if (data.locked) {
          setLockedInfo({ locked: true, remainingMinutes: data.remainingMinutes });
        }
      }
    } catch (e) {
      setErrorMessage('网络错误，请稍后重试');
      setRemainingAttempts(null);
      setShowErrorModal(true);
    } finally {
      setLoading(false);
    }
  };

  const handleReset = () => {
    setUsername('');
    setPassword('');
    setCorpInfo(null);
    setCompanyMessage('输入账号后自动查询公司');
  };

  const closeErrorModal = () => {
    setShowErrorModal(false);
  };

  return (
    <>
      <style>{`
        @keyframes scroll-handle {
          0% { transform: translateX(-3px); opacity: 0.4; }
          50% { opacity: 1; }
          100% { transform: translateX(3px); opacity: 0.4; }
        }
        /* 初始动画：页面加载时播放2秒 */
        .initial-animate .animate-scroll {
          animation: scroll-handle 0.4s infinite alternate ease-in-out;
        }
        .screwdriver-container:hover .animate-scroll {
          animation: scroll-handle 0.4s infinite alternate ease-in-out;
        }
        @keyframes screwdriver-float {
          0%, 100% { transform: translateY(0); }
          50% { transform: translateY(-4px); }
        }
        .initial-animate.screwdriver-container {
          animation: screwdriver-float 1.5s infinite ease-in-out;
        }
        .screwdriver-container:hover {
          animation: screwdriver-float 1.5s infinite ease-in-out;
        }
        /* 极致精密：将摆动幅度压缩至 3.5度，模拟高频切削颤震 */
        @keyframes tool-vibrate-precision {
          0% { transform: rotate(-3.5deg); }
          50% { transform: rotate(3.5deg); }
          100% { transform: rotate(-3.5deg); }
        }
        /* 刀具容器：锚点精准锁定在主轴连接处 */
        .tool-part {
          transform-box: fill-box;
          transform-origin: 12px 7.5px;
          transition: all 0.3s ease;
        }
        /* 初始动画：页面加载时播放2秒 */
        .initial-animate.cnc-machine .tool-part {
          animation: tool-vibrate-precision 0.1s linear infinite;
        }
        /* 鼠标悬停机床时，触发精密微震 */
        .cnc-machine:hover .tool-part {
          animation: tool-vibrate-precision 0.1s linear infinite;
        }
        /* 齿轮初始旋转动画 */
        @keyframes cog-rotate {
          0% { transform: rotate(0deg); }
          100% { transform: rotate(180deg); }
        }
        .initial-animate .cog-icon {
          animation: cog-rotate 0.7s ease-out forwards;
        }
        /* 扳手初始旋转动画 */
        @keyframes wrench-rotate {
          0% { transform: rotate(-45deg); }
          100% { transform: rotate(0deg); }
        }
        .initial-animate .wrench-icon {
          animation: wrench-rotate 0.5s ease-out forwards;
        }
        .glow-overlay {
          background: radial-gradient(circle at 50% 50%, rgba(30, 58, 138, 0.15) 0%, transparent 70%);
        }
      `}</style>

      <div className="relative min-h-screen w-full flex items-center justify-center overflow-hidden font-sans bg-slate-950">
        {/* 背景装饰：微光层 */}
        <div className="absolute inset-0 glow-overlay z-0"></div>

        {/* 装饰线条 */}
        <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-blue-500/20 to-transparent"></div>
        <div className="absolute bottom-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-blue-500/20 to-transparent"></div>

        {/* 登录主体 */}
        <div className="relative z-10 w-full max-w-md p-6">
          <div className="w-full">
            {/* 图标和标题 */}
            <div className="flex flex-col items-center mb-8 sm:mb-10 text-center">
              <div className={`mb-4 sm:mb-6 flex items-center justify-center space-x-3 sm:space-x-6 ${initialAnimation ? 'initial-animate' : ''}`}>
                <div className="cursor-pointer group">
                  <Cog className="cog-icon w-8 h-8 sm:w-12 sm:h-12 text-blue-500 transition-transform duration-700 group-hover:rotate-180" />
                </div>
                <div className="cursor-pointer group">
                  <Wrench className="wrench-icon w-8 h-8 sm:w-12 sm:h-12 text-slate-400 transition-transform duration-500 group-hover:rotate-0" />
                </div>
                <div className={`screwdriver-container relative w-8 h-8 sm:w-12 sm:h-12 flex items-center justify-center cursor-pointer ${initialAnimation ? 'initial-animate' : ''}`}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="w-8 h-8 sm:w-12 sm:h-12 text-blue-400">
                    <path d="M9 3h6a1 1 0 0 1 1 1v4a2 2 0 0 1-2 2h-4a2 2 0 0 1-2-2V4a1 1 0 0 1 1-1z" />
                    <g className="animate-scroll">
                      <path d="M11 5v3" className="stroke-blue-200" />
                      <path d="M13 5v3" className="stroke-blue-200" />
                    </g>
                    <path d="M12 10v9" />
                    <path d="M10 22h4" />
                  </svg>
                </div>
                {/* 工业机床图标 - 刀具高频微颤效果 */}
                <div className={`cnc-machine cursor-pointer ${initialAnimation ? 'initial-animate' : ''}`}>
                  <svg viewBox="0 0 24 24" className="w-8 h-8 sm:w-12 sm:h-12 text-blue-400" fill="none" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M3 20h18M4 20V5a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v15" stroke="currentColor" />
                    <path d="M10.5 4h3v2.5h-3z" fill="currentColor" fillOpacity="0.15" stroke="none" />
                    <rect x="11.2" y="6.5" width="1.6" height="1" fill="currentColor" fillOpacity="0.3" stroke="none" />
                    {/* 刀具：微幅(3.5deg)高频震动 */}
                    <g className="tool-part">
                      <line x1="12" y1="7.5" x2="12" y2="12" stroke="#3b82f6" strokeWidth="1.8" />
                      <path d="M11.7 11.2 L12 12.3 L12.3 11.2" stroke="#60a5fa" strokeWidth="0.8" />
                    </g>
                    <rect x="6" y="7" width="8" height="9" rx="0.5" strokeOpacity="0.2" />
                    <ellipse cx="10" cy="17.5" rx="3.5" ry="0.8" strokeOpacity="0.3" />
                    <rect x="18" y="8" width="3" height="7" rx="0.5" fill="currentColor" fillOpacity="0.1" />
                    <circle cx="19.5" cy="10" r="0.4" fill="#3b82f6" />
                  </svg>
                </div>
              </div>

              <h1 className="text-2xl sm:text-3xl font-light tracking-tight text-white">
                裕合森 <span className="font-bold text-blue-500">ERP 系统</span>
              </h1>
              <div className="flex items-center mt-2 justify-center space-x-2">
                <div className="h-[1px] w-3 sm:w-4 bg-blue-500/30"></div>
                <p className="text-slate-500 text-[8px] sm:text-[10px] tracking-[0.15em] sm:tracking-[0.2em] font-medium uppercase">
                  YUHESEN Machining Resource Planning
                </p>
                <div className="h-[1px] w-3 sm:w-4 bg-blue-500/30"></div>
              </div>
            </div>

            {/* 登录表单 */}
            <form onSubmit={handleSubmit} className="space-y-4">
              {/* 公司字段 - 动态显示 */}
              <div className="relative group">
                <span className="absolute inset-y-0 left-0 pl-4 flex items-center text-slate-600">
                  <Building2 className="w-[18px] h-[18px]" />
                </span>
                <div className="w-full pl-12 pr-4 py-4 bg-slate-900/50 border border-slate-800 rounded-xl text-sm font-medium text-slate-500 min-h-[56px] flex items-center">
                  {corpInfo?.corp_name || companyMessage}
                </div>
              </div>

              {/* 账户锁定提示 */}
              {lockedInfo.locked && (
                <div className="flex items-center gap-3 p-4 bg-orange-500/20 border border-orange-500/30 rounded-xl text-orange-300">
                  <Lock className="w-5 h-5 flex-shrink-0" />
                  <span className="text-sm">
                    账户已锁定，请等待 {lockedInfo.remainingMinutes} 分钟后再试
                  </span>
                </div>
              )}

              {/* 用户名字段 */}
              <div className="relative group">
                <span className="absolute inset-y-0 left-0 pl-4 flex items-center text-slate-400 transition-colors group-focus-within:text-blue-500">
                  <User className="w-[18px] h-[18px]" />
                </span>
                <input
                  type="text"
                  className="w-full pl-12 pr-12 py-4 bg-white/5 border border-white/10 rounded-xl focus:border-blue-500/50 focus:bg-white/10 transition-all outline-none placeholder:text-slate-600 text-sm font-medium text-white"
                  placeholder="登录账户 / Username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  disabled={loading || lockedInfo.locked}
                  required
                />
                {checkingUser && (
                  <Loader2 className="absolute right-4 top-1/2 -translate-y-1/2 w-5 h-5 text-blue-400 animate-spin" />
                )}
              </div>

              {/* 密码字段 */}
              <div className="relative group">
                <span className="absolute inset-y-0 left-0 pl-4 flex items-center text-slate-400 transition-colors group-focus-within:text-blue-500">
                  <Lock className="w-[18px] h-[18px]" />
                </span>
                <input
                  type="password"
                  className="w-full pl-12 pr-4 py-4 bg-white/5 border border-white/10 rounded-xl focus:border-blue-500/50 focus:bg-white/10 transition-all outline-none placeholder:text-slate-600 text-sm font-medium text-white"
                  placeholder="登录密码 / Access Key"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  disabled={loading || lockedInfo.locked}
                  required
                />
              </div>

              {/* 登录按钮 */}
              <div className="flex gap-4 pt-2">
                <button
                  type="button"
                  onClick={handleReset}
                  disabled={loading}
                  className="flex-1 py-4 bg-white/10 hover:bg-white/20 text-white font-semibold rounded-xl transition-all disabled:opacity-50"
                >
                  重置
                </button>
                <button
                  type="submit"
                  disabled={loading || lockedInfo.locked}
                  className="flex-1 py-4 bg-blue-600 hover:bg-blue-500 active:bg-blue-700 text-white font-semibold rounded-xl transition-all shadow-lg shadow-blue-900/20 flex items-center justify-center space-x-2 disabled:opacity-50"
                >
                  {loading ? (
                    <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin"></div>
                  ) : (
                    <>
                      <ShieldCheck className="w-[18px] h-[18px]" />
                      <span className="tracking-widest font-bold">登录</span>
                    </>
                  )}
                </button>
              </div>
            </form>

            {/* 版权信息 */}
            <div className="mt-16 text-center opacity-20">
              <p className="text-[9px] tracking-widest font-bold text-slate-400 uppercase">
                © 2026 Yuhesen Precision. All Rights Reserved.
              </p>
            </div>
          </div>
        </div>

        {/* 错误弹窗 */}
        {showErrorModal && (
          <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4 z-50">
            <div className="bg-slate-800 rounded-2xl p-6 w-full max-w-sm shadow-2xl border border-slate-700">
              {/* Modal Header */}
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 bg-red-500/20 rounded-full flex items-center justify-center">
                    <AlertCircle className="w-5 h-5 text-red-400" />
                  </div>
                  <h3 className="text-lg font-semibold text-white">提示</h3>
                </div>
                <button
                  onClick={closeErrorModal}
                  className="text-slate-400 hover:text-white transition-colors"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Error Message */}
              <div className="mb-6">
                <p className="text-slate-300">{errorMessage}</p>
                {remainingAttempts !== null && remainingAttempts > 0 && (
                  <p className="text-slate-400 text-sm mt-2">
                    剩余 {remainingAttempts} 次尝试机会
                  </p>
                )}
              </div>

              {/* Close Button */}
              <button
                onClick={closeErrorModal}
                className="w-full py-2.5 px-4 bg-slate-700 hover:bg-slate-600 text-white rounded-xl font-medium transition-all"
              >
                确定
              </button>
            </div>
          </div>
        )}
      </div>
    </>
  );
};

export default LoginPage;
