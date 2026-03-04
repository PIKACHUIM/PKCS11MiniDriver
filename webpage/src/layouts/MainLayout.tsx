import React, { useEffect } from 'react';
import { Layout, Menu, Badge, Tooltip, Switch, Typography, Space, Tag } from 'antd';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import {
  DashboardOutlined,
  UserOutlined,
  CreditCardOutlined,
  FileTextOutlined,
  SettingOutlined,
  SafetyCertificateOutlined,
  BulbOutlined,
  ApiOutlined,
} from '@ant-design/icons';
import { useAppStore } from '../store/appStore';

const { Sider, Content, Header } = Layout;
const { Text } = Typography;

const menuItems = [
  { key: '/dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
  { key: '/users', icon: <UserOutlined />, label: '用户管理' },
  { key: '/cards', icon: <CreditCardOutlined />, label: '卡片管理' },
  { key: '/logs', icon: <FileTextOutlined />, label: '操作日志' },
  { key: '/settings', icon: <SettingOutlined />, label: '系统设置' },
];

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { connected, serverVersion, darkMode, toggleDarkMode, checkConnection, loadSlots } = useAppStore();

  useEffect(() => {
    checkConnection();
    loadSlots();
    // 每 30 秒刷新一次连接状态
    const timer = setInterval(() => checkConnection(), 30000);
    return () => clearInterval(timer);
  }, []);

  const selectedKey = '/' + location.pathname.split('/')[1] || '/dashboard';

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {/* 侧边栏 */}
      <Sider
        width={220}
        style={{
          background: darkMode ? '#0d1117' : '#001529',
          borderRight: darkMode ? '1px solid #21262d' : 'none',
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
                PKCS11 Driver
              </div>
              <div style={{ color: 'rgba(255,255,255,0.45)', fontSize: 11 }}>
                智能卡管理平台
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

      <Layout>
        {/* 顶部栏 */}
        <Header style={{
          background: darkMode ? '#161b22' : '#fff',
          padding: '0 24px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: darkMode ? '1px solid #21262d' : '1px solid #f0f0f0',
          height: 56,
        }}>
          {/* 连接状态 */}
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
            {connected && (
              <Tag color="blue" style={{ fontSize: 11 }}>:1026</Tag>
            )}
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
          </Space>
        </Header>

        {/* 主内容区 */}
        <Content style={{
          background: darkMode ? '#0d1117' : '#f5f5f5',
          minHeight: 'calc(100vh - 56px)',
          overflow: 'auto',
        }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

export default MainLayout;
