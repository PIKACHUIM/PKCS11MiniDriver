import React, { useEffect } from 'react';
import {
  Layout, Menu, Badge, Tooltip, Switch, Typography, Space, Tag, Button,
} from 'antd';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import {
  DashboardOutlined, UserOutlined, CreditCardOutlined, FileTextOutlined,
  SettingOutlined, SafetyCertificateOutlined, BulbOutlined, ApiOutlined,
  ClockCircleOutlined, FileProtectOutlined, BankOutlined, KeyOutlined,
  LogoutOutlined, FileDoneOutlined,
} from '@ant-design/icons';
import { useAppStore } from '../store/appStore';
import { useAuthStore } from '../store/auth';


const { Sider, Content, Header } = Layout;
const { Text } = Typography;

// Manager 菜单项（只管理本地 client-card :1026 的功能）
const menuItems = [
  // 概览
  {
    type: 'group' as const,
    label: '概览',
    children: [
      { key: '/dashboard', icon: <DashboardOutlined />, label: '系统概览' },
    ],
  },
  {
    key: 'group-local',
    icon: <CreditCardOutlined />,
    label: '设备管理',
    children: [
      { key: '/cards', icon: <CreditCardOutlined />, label: '智能卡片管理' },
      { key: '/certs', icon: <SafetyCertificateOutlined />, label: '用户证书管理' },
    ],
  },
  {
    key: 'group-pki',
    icon: <FileProtectOutlined />,
    label: '证书工具',
    children: [
      { key: '/pki/csr', icon: <KeyOutlined />, label: '本地证书申请' },
      { key: '/pki/ca', icon: <BankOutlined />, label: '本地颁发机构' },
      { key: '/pki/certs', icon: <FileDoneOutlined />, label: '证书签发管理' },
    ],
  },
  {
    key: 'group-security',
    icon: <ClockCircleOutlined />,
    label: '安全凭据',
    children: [
      { key: '/totp', icon: <ClockCircleOutlined />, label: 'TOTP验证管理' },
    ],
  },
  {
    key: 'group-cloud',
    icon: <ApiOutlined />,
    label: '云端功能',
    children: [
      { key: '/users', icon: <UserOutlined />, label: '云端账号管理' },
    ],
  },
  {
    key: 'group-system',
    icon: <SettingOutlined />,
    label: '系统管理',
    children: [
      { key: '/logs', icon: <FileTextOutlined />, label: '平台操作日志' },
      { key: '/settings', icon: <SettingOutlined />, label: '平台系统设置' },
    ],
  },
];

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { connected, serverVersion, darkMode, toggleDarkMode, checkConnection, loadSlots } = useAppStore();
  const { clearAuth, username } = useAuthStore();

  const handleLogout = () => {
    clearAuth();
    navigate('/login', { replace: true });
  };

  useEffect(() => {
    checkConnection();
    loadSlots();
    const timer = setInterval(() => checkConnection(), 30000);
    return () => clearInterval(timer);
  }, []);

  const selectedKey = location.pathname;
  // 根据当前路径计算应展开的父菜单 key（概览不需要展开）
  const getOpenKey = () => {
    const p = location.pathname;
    if (p.startsWith('/pki')) return 'group-pki';
    if (['/users', '/cards', '/certs', '/totp'].some(r => p.startsWith(r))) return 'group-local';
    return 'group-system';
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {/* 侧边栏 */}
      <Sider
        width={220}
        style={{
          background: darkMode ? '#0d1117' : '#001529',
          borderRight: darkMode ? '1px solid #21262d' : 'none',
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0, top: 0, bottom: 0,
        }}
      >
        {/* Logo 区域 */}
        <div style={{
          padding: '20px 16px 16px',
          borderBottom: '1px solid rgba(255,255,255,0.08)',
          marginBottom: 8,
        }}>
          <Space align="center">
            <SafetyCertificateOutlined style={{ fontSize: 24, color: '#1677ff' }} />
            <div>
              <div style={{ color: '#fff', fontWeight: 700, fontSize: 14, lineHeight: 1.2 }}>
                OpenCert Manager
              </div>
              <div style={{ color: 'rgba(255,255,255,0.45)', fontSize: 11 }}>
                本地智能卡管理
              </div>
            </div>
          </Space>
        </div>

        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          defaultOpenKeys={[getOpenKey()]}
          items={menuItems as any}
          onClick={({ key }) => navigate(key)}
          style={{ background: 'transparent', border: 'none' }}
        />
      </Sider>

      <Layout style={{ marginLeft: 220 }}>
        {/* 顶部栏 */}
        <Header style={{
          background: darkMode ? '#161b22' : '#fff',
          padding: '0 24px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: darkMode ? '1px solid #21262d' : '1px solid #f0f0f0',
          height: 56,
          position: 'sticky',
          top: 0,
          zIndex: 10,
        }}>
          {/* 连接状态（连接 client-card :1026） */}
          <Space>
            <ApiOutlined style={{ color: connected ? '#52c41a' : '#ff4d4f' }} />
            <Badge
              status={connected ? 'success' : 'error'}
              text={
                <Text style={{ fontSize: 13, color: darkMode ? '#c9d1d9' : undefined }}>
                  {connected ? `已连接 client-card v${serverVersion}` : '未连接 client-card'}
                </Text>
              }
            />
            {connected && <Tag color="blue" style={{ fontSize: 11 }}>:1026</Tag>}
          </Space>

          {/* 右侧：用户信息 + 主题切换 */}
          <Space>
            <Tooltip title={darkMode ? '切换亮色模式' : '切换暗色模式'}>
              <Switch
                checkedChildren={<BulbOutlined />}
                unCheckedChildren={<BulbOutlined />}
                checked={darkMode}
                onChange={toggleDarkMode}
                size="small"
              />
            </Tooltip>
            {username && (
              <Text style={{ fontSize: 13, color: darkMode ? '#8b949e' : '#666' }}>
                {username}
              </Text>
            )}
            <Tooltip title="退出登录">
              <Button
                type="text"
                size="small"
                icon={<LogoutOutlined />}
                onClick={handleLogout}
                style={{ color: darkMode ? '#8b949e' : '#666' }}
              />
            </Tooltip>
          </Space>
        </Header>

        {/* 主内容区 */}
        <Content style={{
          background: darkMode ? '#0d1117' : '#f5f5f5',
          minHeight: 'calc(100vh - 56px)',
          overflow: 'auto',
          padding: 24,
        }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

export default MainLayout;
