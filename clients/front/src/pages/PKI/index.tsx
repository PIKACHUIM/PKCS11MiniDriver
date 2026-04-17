import React from 'react';
import { Tabs } from 'antd';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import { FileTextOutlined, ApartmentOutlined, SafetyCertificateOutlined } from '@ant-design/icons';

/** PKI 工具入口：三个 Tab 切换（CSR 管理 / 本地 CA 管理 / 证书管理） */
const PKIPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();

  // 根据当前路径确定激活的 Tab
  const activeKey = (() => {
    if (location.pathname.includes('/pki/ca')) return 'ca';
    if (location.pathname.includes('/pki/certs')) return 'certs';
    return 'csr'; // 默认 CSR
  })();

  const tabs = [
    {
      key: 'csr',
      label: (
        <span><FileTextOutlined />CSR 管理</span>
      ),
    },
    {
      key: 'ca',
      label: (
        <span><ApartmentOutlined />本地 CA 管理</span>
      ),
    },
    {
      key: 'certs',
      label: (
        <span><SafetyCertificateOutlined />证书管理</span>
      ),
    },
  ];

  const handleTabChange = (key: string) => {
    navigate(`/pki/${key}`);
  };

  return (
    <div>
      <Tabs
        activeKey={activeKey}
        items={tabs}
        onChange={handleTabChange}
        style={{ marginBottom: 0 }}
        tabBarStyle={{ marginBottom: 0, paddingBottom: 0 }}
      />
      <div style={{ marginTop: 16 }}>
        <Outlet />
      </div>
    </div>
  );
};

export default PKIPage;
