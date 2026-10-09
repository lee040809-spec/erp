import React, { useState, useEffect } from 'react';
import { authFetch, getToken, setToken, removeToken } from './utils/auth';
import { User, Resource } from './types';
import LoginPage from './components/LoginPage';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import UsersPage from './pages/UsersPage';
import RolesPage from './pages/RolesPage';
import ResourcesPage from './pages/ResourcesPage';
import CompaniesPage from './pages/CompaniesPage';

function App() {
  const [user, setUser] = useState<User | null>(null);
  const [menus, setMenus] = useState<Resource[]>([]);
  const [currentPage, setCurrentPage] = useState('/portal/dashboard');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    checkAuth();
  }, []);

  const checkAuth = async () => {
    const token = getToken();
    if (!token) {
      setLoading(false);
      return;
    }

    try {
      const res = await authFetch('/api/portal/auth/status');
      const data = await res.json();

      if (data.authenticated && data.user) {
        setUser(data.user);
        fetchMenus();
      } else {
        removeToken();
      }
    } catch (err) {
      console.error('Auth check failed:', err);
      removeToken();
    } finally {
      setLoading(false);
    }
  };

  const fetchMenus = async () => {
    try {
      const res = await authFetch('/api/portal/resources/menus');
      const data = await res.json();
      setMenus(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch menus:', err);
    }
  };

  const handleLogin = (userData: User) => {
    setUser(userData);
    fetchMenus();
  };

  const handleLogout = async () => {
    try {
      await authFetch('/api/portal/logout', { method: 'POST' });
    } catch (err) {
      console.error('Logout failed:', err);
    }
    removeToken();
    setUser(null);
    setMenus([]);
  };

  const renderPage = () => {
    switch (currentPage) {
      case '/portal/dashboard':
        return <Dashboard user={user} />;
      case '/portal/users':
        return <UsersPage />;
      case '/portal/roles':
        return <RolesPage />;
      case '/portal/resources':
        return <ResourcesPage />;
      case '/portal/companies':
        return <CompaniesPage />;
      default:
        return <Dashboard user={user} />;
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-zinc-50">
        <div className="text-zinc-500">加载中...</div>
      </div>
    );
  }

  if (!user) {
    return <LoginPage onLogin={handleLogin} />;
  }

  return (
    <Layout
      user={user}
      menus={menus}
      onLogout={handleLogout}
      onPageChange={setCurrentPage}
      currentPage={currentPage}
    >
      {renderPage()}
    </Layout>
  );
}

export default App;