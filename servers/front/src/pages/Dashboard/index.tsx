import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Table, Tag, Typography, Space, Spin, Alert } from 'antd';
import {
  UserOutlined, ApartmentOutlined, SafetyCertificateOutlined,
  AuditOutlined, WalletOutlined, ClockCircleOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../../store/auth';
import { useThemeStore } from '../../store/theme';
import { listUsers, listCAs, listAllCerts, listCertApplications, getBalance, getLogs } from '../../api';
import type { Log } from '../../types';
import dayjs from 'dayjs';

const { Title, Text } = Typography;

const Dashboard: React.FC = () => {
  const { role } = useAuthStore();
  const { darkMode } = useThemeStore();
  const [loading, setLoading] = useState(true);
  const [userCount, setUserCount] = useState(0);
  const [caCount, setCACount] = useState(0);
  const [certCount, setCertCount] = useState(0);
  const [pendingApps, setPendingApps] = useState(0);
  const [balance, setBalance] = useState<number | null>(null);
  const [recentLogs, setRecentLogs] = useState<Log[]>([]);

  useEffect(() => {
    setLoading(true);
    const tasks: Promise<any>[] = [
      listCAs({ page: 1, page_size: 1 }).catch(() => null),
      listAllCerts({ page: 1, page_size: 1 }).catch(() => null),
      getLogs({ page: 1, page_size: 8 }).catch(() => null),
      getBalance().catch(() => null),
    ];
    if (role === 'admin') {
      tasks.push(listUsers({ page: 1, page_size: 1 }).catch(() => null));
      tasks.push(listCertApplications({ page: 1, page_size: 1, status: 'pending' }).catch(() => null));
    }
    Promise.all(tasks)
      .then(([cas, certs, logs, bal, users, apps]) => {
        setCACount(cas?.total ?? 0);
        setCertCount(certs?.total ?? 0);
        setRecentLogs(logs?.items ?? []);
        if (bal) setBalance(bal.available);
        if (users) setUserCount(users.total ?? 0);
        if (apps) setPendingApps(apps.total ?? 0);
      })
      .finally(() => setLoading(false));
  }, [role]);

  const cardStyle = {
    background: darkMode ? '#161b22' : '#fff',
    border: darkMode ? '1px solid #21262d' : '1px solid #f0f0f0',
    borderRadius: 12,
  };

  const logColumns = [
    { title: '时间', dataIndex: 'created_at', width: 160, render: (v: string) => <Text style={{ fontSize: 12, color: '#8b949e' }}>{dayjs(v).format('MM-DD HH:mm:ss')}</Text> },
    { title: '级别', dataIndex: 'level', width: 70, render: (v: string) => <Tag color={v === 'error' ? 'red' : v === 'warn' ? 'orange' : 'blue'} style={{ fontSize: 11 }}>{v?.toUpperCase()}</Tag> },
    { title: '标题', dataIndex: 'title', render: (v: string) => <Text style={{ fontSize: 13 }}>{v}</Text> },
  ];

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>平台概览</Title>

      <Alert
        message="已连接 OpenCert Platform (server-card :1027)"
        type="success" showIcon style={{ marginBottom: 16 }}
      />

      <Spin spinning={loading}>
        <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
          <Col xs={24} sm={12} lg={6}>
            <Card style={cardStyle} bodyStyle={{ padding: '20px 24px' }}>
              <Statistic title="CA 数量" value={caCount}
                prefix={<ApartmentOutlined style={{ color: '#1677ff' }} />} />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card style={cardStyle} bodyStyle={{ padding: '20px 24px' }}>
              <Statistic title="证书总数" value={certCount}
                prefix={<SafetyCertificateOutlined style={{ color: '#52c41a' }} />} />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card style={cardStyle} bodyStyle={{ padding: '20px 24px' }}>
              <Statistic title="账户余额" value={balance !== null ? (balance / 100).toFixed(2) : '—'}
                suffix="元" prefix={<WalletOutlined style={{ color: '#fa8c16' }} />} />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card style={cardStyle} bodyStyle={{ padding: '20px 24px' }}>
              <Statistic title="待审批申请" value={pendingApps}
                prefix={<AuditOutlined style={{ color: pendingApps > 0 ? '#ff4d4f' : '#722ed1' }} />}
                valueStyle={{ color: pendingApps > 0 ? '#ff4d4f' : undefined }} />
            </Card>
          </Col>
        </Row>

        {role === 'admin' && (
          <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
            <Col xs={24} sm={12} lg={6}>
              <Card style={{ ...cardStyle, borderColor: '#1677ff33' }} bodyStyle={{ padding: '20px 24px' }}>
                <Statistic title="平台用户总数" value={userCount}
                  prefix={<UserOutlined style={{ color: '#1677ff' }} />} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card style={{ ...cardStyle, borderColor: '#722ed133' }} bodyStyle={{ padding: '20px 24px' }}>
                <Statistic title="TOTP 条目" value="—"
                  prefix={<ClockCircleOutlined style={{ color: '#722ed1' }} />} />
              </Card>
            </Col>
          </Row>
        )}

        <Card title={<Space><SafetyCertificateOutlined /><span>最近操作日志</span></Space>} style={cardStyle}>
          <Table dataSource={recentLogs} columns={logColumns} rowKey="uuid"
            pagination={false} size="small" />
        </Card>
      </Spin>
    </div>
  );
};

export default Dashboard;
