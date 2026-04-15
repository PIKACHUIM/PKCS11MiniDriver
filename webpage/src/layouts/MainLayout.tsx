import React, { useEffect } from 'react';
import {
  Layout, Menu, Badge, Tooltip, Switch, Typography, Space, Tag,
  Avatar, Dropdown,
} from 'antd';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import {
  DashboardOutlined, UserOutlined, CreditCardOutlined, FileTextOutlined,
  SettingOutlined, SafetyCertificateOutlined, BulbOutlined, ApiOutlined,
  ClockCircleOutlined, FileProtectOutlined, ApartmentOutlined, AppstoreOutlined,
  WalletOutlined, IdcardOutlined, ShoppingOutlined, AuditOutlined,
  LogoutOutlined, ProfileOutlined, ExperimentOutlined,
} from '@ant-design/icons';
import { useAppStore } from '../store/appStore';
import { useAuthStore } from '../store/auth';
import { logout } from '../api';

const { Sider, Content, Header } = Layout;
const { Text } = Typography;

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { connected, serverVersion, darkMode, toggleDarkMode, checkConnection, loadSlots } = useAppStore();
  const { role, username, clearAuth } = useAuthStore();

  useEffect(() => {
    checkConnection();
    loadSlots();
    const timer = setInterval(() => checkConnection(), 30000);
    return () => clearInterval(timer);
  }, []);

  const selectedKey = '/' + location.pathname.split('/')[1] || '/dashboard';

  const handleLogout = async () => {
    try { await logout(); } catch {}
    clearAuth();
    navigate('/login', { replace: true });
  };

  // 构建分组菜单
  const menuItems: any[] = [
    {
      type: 'group',
      label: '概览',
      children: [
        { key: '/dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
      ],
    },
    {
      type: 'group',
      label: '我的',
      children: [
        { key: '/cards', icon: <CreditCardOutlined />, label: '智能卡' },
        { key: '/certs', icon: <SafetyCertificateOutlined />, label: '证书' },
        { key: '/totp', icon: <ClockCircleOutlined />, label: 'TOTP 验证器' },
        { key: '/identity', icon: <IdcardOutlined />, label: '身份信息' },
        { key: '/cert-orders', icon: <ShoppingOutlined />, label: '证书申请' },
        { key: '/payment', icon: <WalletOutlined />, label: '支付' },
      ],
    },
    {
      type: 'group',
      label: 'PKI 工具',
      children: [
        { key: '/pki', icon: <FileProtectOutlined />, label: '本地 PKI' },
      ],
    },
    // 平台管理分组仅 admin 可见
    ...(role === 'admin' ? [{
      type: 'group',
      label: '平台管理',
      children: [
        { key: '/ca', icon: <ApartmentOutlined />, label: 'CA 管理' },
        { key: '/templates', icon: <AppstoreOutlined />, label: '模板管理' },
        { key: '/users', icon: <UserOutlined />, label: '用户管理' },
        { key: '/ct-records', icon: <AuditOutlined />, label: 'CT 记录' },
      ],
    }] : []),
    {
      type: 'group',
      label: '系统',
      children: [
        { key: '/logs', icon: <FileTextOutlined />, label: '操作日志' },
        { key: '/settings', icon: <SettingOutlined />, label: '设置' },
      ],
    },
  ];

  const userMenuItems = [
    { key: 'profile', icon: <ProfileOutlined />, label: '个人中心', onClick: () => navigate('/profile') },
    { type: 'divider' },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', danger: true, onClick: handleLogout },
  ];

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
                OpenCert
              </div>
              <div style={{ color: 'rgba(255,255,255,0.45)', fontSize: 11 }}>
                {role === 'admin' ? 'Platform 管理端' : '证书管理平台'}
              </div>
            </div>
          </Space>
        </div>

        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
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
          {/* 连接状态 */}
          <Space>
            <ApiOutlined style={{ color: connected ? '#52c41a' : '#ff4d4f' }} />
            <Badge
              status={connected ? 'success' : 'error'}
              text={
                <Text style={{ fontSize: 13, color: darkMode ? '#c9d1d9' : undefined }}>
                  {connected ? `已连接 v${serverVersion}` : '未连接服务'}
                </Text>
              }
            />
            {connected && <Tag color="blue" style={{ fontSize: 11 }}>:1026</Tag>}
          </Space>

          {/* 右侧工具栏 */}
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
            {/* 用户头像下拉菜单 */}
            <Dropdown menu={{ items: userMenuItems as any }} placement="bottomRight" arrow>
              <Space style={{ cursor: 'pointer' }}>
                <Avatar
                  size={32}
                  style={{ background: 'linear-gradient(135deg, #1677ff, #722ed1)', cursor: 'pointer' }}
                  icon={<UserOutlined />}
                />
                <Text style={{ fontSize: 13, color: darkMode ? '#c9d1d9' : undefined }}>
                  {username}
                </Text>
                {role === 'admin' && <Tag color="gold" style={{ fontSize: 11 }}>管理员</Tag>}
              </Space>
            </Dropdown>
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
