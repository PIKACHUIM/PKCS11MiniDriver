import React from 'react';
import { Navigate } from 'react-router-dom';

// 自签名证书功能已合并到「证书管理」页面，此处直接重定向
const SelfSignPage: React.FC = () => <Navigate to="/pki/certs" replace />;

export default SelfSignPage;
