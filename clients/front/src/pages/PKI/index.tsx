import React from 'react';
import { Outlet } from 'react-router-dom';

/** PKI 工具入口：作为嵌套路由容器，渲染子页面 */
const PKIPage: React.FC = () => {
  return <Outlet />;
};

export default PKIPage;
