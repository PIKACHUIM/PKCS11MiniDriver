import React, { Suspense, lazy, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, theme, Spin } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { useAppStore } from './store/appStore';
import MainLayout from './layouts/MainLayout';
import ErrorBoundary from './components/ErrorBoundary';
import PrivateRoute from './components/PrivateRoute';

// 公开页面
const Home = lazy(() => import('./pages/Home'));
const Login = lazy(() => import('./pages/Login'));

// 受保护页面（懒加载）
const Dashboard = lazy(() => import('./pages/Dashboard'));
const Users = lazy(() => import('./pages/Users'));
const Cards = lazy(() => import('./pages/Cards'));
const Certs = lazy(() => import('./pages/Certs'));
const TOTP = lazy(() => import('./pages/TOTP'));
const PKI = lazy(() => import('./pages/PKI'));
const Logs = lazy(() => import('./pages/Logs'));
const Settings = lazy(() => import('./pages/Settings'));
const Profile = lazy(() => import('./pages/Profile'));
// 新增页面
const CA = lazy(() => import('./pages/CA'));
const Templates = lazy(() => import('./pages/Templates'));
const Payment = lazy(() => import('./pages/Payment'));
const Identity = lazy(() => import('./pages/Identity'));
const CertOrders = lazy(() => import('./pages/CertOrders'));
const CTRecords = lazy(() => import('./pages/CTRecords'));

const PageLoader = () => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
    <Spin size="large" />
  </div>
);

const S = ({ children }: { children: React.ReactNode }) => (
  <Suspense fallback={<PageLoader />}>{children}</Suspense>
);

const App: React.FC = () => {
  const { darkMode, themeMode, setThemeMode } = useAppStore();

  useEffect(() => {
    if (themeMode !== 'system') return;
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => setThemeMode('system');
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, [themeMode, setThemeMode]);

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: darkMode ? theme.darkAlgorithm : theme.defaultAlgorithm,
        token: {
          colorPrimary: '#1677ff',
          borderRadius: 8,
          fontFamily: "'PingFang SC', 'Microsoft YaHei', 'Segoe UI', sans-serif",
        },
        components: {
          Layout: { siderBg: darkMode ? '#0d1117' : '#001529' },
          Menu: { darkItemBg: 'transparent', darkSubMenuItemBg: 'transparent' },
        },
      }}
    >
      <ErrorBoundary>
        <BrowserRouter>
          <Routes>
            {/* 公开路由 */}
            <Route path="/" element={<S><Home /></S>} />
            <Route path="/login" element={<S><Login /></S>} />

            {/* 受保护路由：包裹在 MainLayout 内 */}
            <Route path="/" element={<PrivateRoute><MainLayout /></PrivateRoute>}>
              <Route path="dashboard" element={<S><Dashboard /></S>} />
              <Route path="users" element={<S><Users /></S>} />
              <Route path="cards" element={<S><Cards /></S>} />
              <Route path="certs" element={<S><Certs /></S>} />
              <Route path="totp" element={<S><TOTP /></S>} />
              <Route path="pki" element={<S><PKI /></S>} />
              <Route path="logs" element={<S><Logs /></S>} />
              <Route path="settings" element={<S><Settings /></S>} />
              <Route path="profile" element={<S><Profile /></S>} />
              {/* 新增页面 */}
              <Route path="ca" element={<S><CA /></S>} />
              <Route path="templates" element={<S><Templates /></S>} />
              <Route path="payment" element={<S><Payment /></S>} />
              <Route path="identity" element={<S><Identity /></S>} />
              <Route path="cert-orders" element={<S><CertOrders /></S>} />
              <Route path="ct-records" element={<S><CTRecords /></S>} />
            </Route>

            {/* 兜底重定向 */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </ErrorBoundary>
    </ConfigProvider>
  );
};

export default App;
