import React, { useState, useEffect } from 'react';
import { X, Lock, Mail, KeyRound, AlertCircle, CheckCircle, Loader2 } from 'lucide-react';

interface UserSettingsModalProps {
  isOpen: boolean;
  type: 'password' | 'reset' | 'email' | null;
  onClose: () => void;
  onLogout?: () => void;
  userId: number;
  userEmail: string;
}

const UserSettingsModal: React.FC<UserSettingsModalProps> = ({
  isOpen,
  type,
  onClose,
  onLogout,
  userId,
  userEmail: initialUserEmail
}) => {
  // 修改密码
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [passwordLoading, setPasswordLoading] = useState(false);

  // 密码重置
  const [resetLoading, setResetLoading] = useState(false);
  const [resetResult, setResetResult] = useState<{
    success: boolean;
    email_sent: boolean;
    email: string;
    message: string;
    error?: string;
  } | null>(null);

  // 用户邮箱（从API实时获取）
  const [userEmail, setUserEmail] = useState('');
  const [loadingUserInfo, setLoadingUserInfo] = useState(false);

  // 修改邮箱
  const [newEmail, setNewEmail] = useState('');
  const [emailLoading, setEmailLoading] = useState(false);

  // 错误/成功提示
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  // 获取用户信息
  const fetchUserInfo = async () => {
    setLoadingUserInfo(true);
    try {
      const token = localStorage.getItem('auth_token');
      const res = await fetch('/api/platform/user/info', {
        headers: {
          'Authorization': token || ''
        }
      });
      const data = await res.json();
      if (res.ok) {
        setUserEmail(data.email || '');
      }
    } catch (e) {
      // 忽略错误
    } finally {
      setLoadingUserInfo(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      setOldPassword('');
      setNewPassword('');
      setNewEmail('');
      setError('');
      setSuccess('');
      setResetResult(null);

      // 打开弹框时获取用户信息
      if (type === 'reset' || type === 'email') {
        fetchUserInfo();
      }
    }
  }, [isOpen, type]);

  const handleChangePassword = async () => {
    if (!oldPassword.trim()) {
      setError('请输入原密码');
      return;
    }
    if (!newPassword.trim()) {
      setError('请输入新密码');
      return;
    }
    if (newPassword.length < 8) {
      setError('新密码长度至少8位');
      return;
    }

    setPasswordLoading(true);
    setError('');

    try {
      const token = localStorage.getItem('auth_token');
      const res = await fetch('/api/platform/user/change-password', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token || ''
        },
        body: JSON.stringify({
          user_id: userId,
          old_password: oldPassword,
          new_password: newPassword
        })
      });

      const data = await res.json();

      if (data.success) {
        setSuccess('密码修改成功，请重新登录');
        setTimeout(() => {
          // 清除token并重新登录
          localStorage.removeItem('auth_token');
          localStorage.removeItem('auth_user');
          if (onLogout) {
            onLogout();
          }
          onClose();
          // 刷新页面跳转到登录
          window.location.reload();
        }, 1500);
      } else {
        setError(data.error || '密码修改失败');
      }
    } catch (e) {
      setError('网络错误，请稍后重试');
    } finally {
      setPasswordLoading(false);
    }
  };

  const handleResetPassword = async () => {
    setResetLoading(true);
    setError('');
    setResetResult(null);

    try {
      const token = localStorage.getItem('auth_token');
      const res = await fetch('/api/platform/user/reset-password', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token || ''
        },
        body: JSON.stringify({
          user_id: userId
        })
      });

      const data = await res.json();

      setResetResult({
        success: data.success,
        email_sent: data.email_sent,
        email: userEmail,
        message: data.message,
        error: data.error
      });
    } catch (e) {
      setError('网络错误，请稍后重试');
    } finally {
      setResetLoading(false);
    }
  };

  const handleChangeEmail = async () => {
    if (!newEmail.trim()) {
      setError('请输入新邮箱');
      return;
    }

    // 校验邮箱格式
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(newEmail)) {
      setError('邮箱格式不正确');
      return;
    }

    // 校验是否为QQ邮箱或163邮箱
    const allowedDomains = ['qq.com', '163.com', '126.com'];
    const emailDomain = newEmail.split('@')[1]?.toLowerCase();
    if (!allowedDomains.includes(emailDomain)) {
      setError('仅支持QQ邮箱、163邮箱、126邮箱');
      return;
    }

    setEmailLoading(true);
    setError('');

    try {
      const token = localStorage.getItem('auth_token');
      const res = await fetch('/api/platform/user/change-email', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token || ''
        },
        body: JSON.stringify({
          user_id: userId,
          new_email: newEmail
        })
      });

      const data = await res.json();

      if (data.success) {
        setSuccess('邮箱修改成功');
        setTimeout(() => onClose(), 1500);
      } else {
        setError(data.error || '邮箱修改失败');
      }
    } catch (e) {
      setError('网络错误，请稍后重试');
    } finally {
      setEmailLoading(false);
    }
  };

  if (!isOpen || !type) return null;

  const getTitle = () => {
    switch (type) {
      case 'password': return '修改密码';
      case 'reset': return '密码重置';
      case 'email': return '修改邮箱';
    }
  };

  const getIcon = () => {
    switch (type) {
      case 'password': return <Lock className="w-5 h-5" />;
      case 'reset': return <KeyRound className="w-5 h-5" />;
      case 'email': return <Mail className="w-5 h-5" />;
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-2xl p-6 w-full max-w-md shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center text-blue-600">
              {getIcon()}
            </div>
            <h3 className="text-lg font-semibold text-gray-900">{getTitle()}</h3>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="flex items-center gap-3 p-4 bg-red-50 border border-red-200 rounded-xl text-red-600 mb-4">
            <AlertCircle className="w-5 h-5 flex-shrink-0" />
            <span className="text-sm">{error}</span>
          </div>
        )}

        {/* Success Alert */}
        {success && (
          <div className="flex items-center gap-3 p-4 bg-green-50 border border-green-200 rounded-xl text-green-600 mb-4">
            <CheckCircle className="w-5 h-5 flex-shrink-0" />
            <span className="text-sm">{success}</span>
          </div>
        )}

        {/* Password Change Form */}
        {type === 'password' && (
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">原密码</label>
              <input
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="请输入原密码"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">新密码</label>
              <input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="请输入新密码（至少8位）"
              />
              <p className="text-xs text-gray-400 mt-1">密码长度至少8位</p>
            </div>
            <button
              onClick={handleChangePassword}
              disabled={passwordLoading}
              className="w-full py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
            >
              {passwordLoading && <Loader2 className="w-5 h-5 animate-spin" />}
              {passwordLoading ? '处理中...' : '确认修改'}
            </button>
          </div>
        )}

        {/* Password Reset Form */}
        {type === 'reset' && (
          <div className="space-y-4">
            {!resetResult ? (
              <>
                <p className="text-sm text-gray-600">
                  点击重置后，系统将生成8位随机密码并发送到您的邮箱：
                </p>
                {loadingUserInfo ? (
                  <div className="flex items-center justify-center py-3">
                    <Loader2 className="w-5 h-5 animate-spin text-gray-400" />
                  </div>
                ) : (
                  <p className="text-sm font-medium text-gray-900 bg-gray-100 px-4 py-2 rounded-lg">
                    {userEmail || '未设置邮箱'}
                  </p>
                )}
                <button
                  onClick={handleResetPassword}
                  disabled={resetLoading || !userEmail || loadingUserInfo}
                  className="w-full py-3 bg-orange-500 hover:bg-orange-600 text-white rounded-xl font-medium transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                >
                  {resetLoading && <Loader2 className="w-5 h-5 animate-spin" />}
                  {resetLoading ? '处理中...' : '重置密码'}
                </button>
                {!userEmail && (
                  <p className="text-xs text-red-500 text-center">请先设置邮箱</p>
                )}
              </>
            ) : (
              <div className={`p-4 rounded-xl ${resetResult.email_sent ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'}`}>
                <div className="flex items-center gap-2 mb-2">
                  {resetResult.email_sent ? (
                    <CheckCircle className="w-5 h-5 text-green-500" />
                  ) : (
                    <AlertCircle className="w-5 h-5 text-red-500" />
                  )}
                  <span className={`font-medium ${resetResult.email_sent ? 'text-green-600' : 'text-red-600'}`}>
                    {resetResult.email_sent ? '发送成功' : '发送失败'}
                  </span>
                </div>
                <p className={`text-sm ${resetResult.email_sent ? 'text-green-600' : 'text-red-600'}`}>
                  {resetResult.email_sent
                    ? `新密码已发送至 ${resetResult.email}`
                    : `发送至 ${resetResult.email} 失败：${resetResult.error || '邮件服务异常'}`
                  }
                </p>
                <button
                  onClick={onClose}
                  className="w-full mt-4 py-2 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-xl font-medium transition-colors"
                >
                  关闭
                </button>
              </div>
            )}
          </div>
        )}

        {/* Email Change Form */}
        {type === 'email' && (
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">当前邮箱</label>
              {loadingUserInfo ? (
                <div className="flex items-center justify-center py-3 bg-gray-100 rounded-xl">
                  <Loader2 className="w-5 h-5 animate-spin text-gray-400" />
                </div>
              ) : (
                <input
                  type="text"
                  value={userEmail || '未设置邮箱'}
                  disabled
                  className="w-full px-4 py-3 border border-gray-200 rounded-xl bg-gray-100 text-gray-500"
                />
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">新邮箱</label>
              <input
                type="email"
                value={newEmail}
                onChange={(e) => setNewEmail(e.target.value)}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="仅支持QQ邮箱、163邮箱、126邮箱"
              />
              <p className="text-xs text-gray-400 mt-1">支持：QQ邮箱、163邮箱、126邮箱</p>
            </div>
            <button
              onClick={handleChangeEmail}
              disabled={emailLoading}
              className="w-full py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
            >
              {emailLoading && <Loader2 className="w-5 h-5 animate-spin" />}
              {emailLoading ? '处理中...' : '确认修改'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default UserSettingsModal;