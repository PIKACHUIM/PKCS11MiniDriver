import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Badge, Table, Tag, Typography, Space, Spin, Alert } from 'antd';
import {
  CreditCardOutlined,
  UserOutlined,
  SafetyCertificateOutlined,
  ApiOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import { useAppStore } from '../../store/appStore';
import { getUsers, getCards, getLogs } from '../../api';
import type { Log } from '../../types';
import dayjs from 'dayjs';

const { Title, Text } = Typography;

const Dashboard: React.FC = () => {
  const { connected, slots, slotsLoading, darkMode } = useAppStore();
  const [userCount, setUserCount] = useState(0);
  const [cardCount, setCardCount] = useState(0);
  const [recentLogs, setRecentLogs] = useState<Log[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!connected) return;
    setLoading(true);
    Promise.all([
      getUsers({ page: 1, page_size: 1 }),
      getCards({ page: 1, page_size: 1 }),
      getLogs({ page: 1, page_size: 8 }),
    ])
      .then(([users, cards, logs]) => {
        setUserCount(users?.total ?? 0);
        setCardCount(cards?.total ?? 0);
        setRecentLogs(logs?.items ?? []);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [connected]);

  const cardStyle = {
    background: darkMode ? '#161b22' : '#fff',
    border: darkMode ? '1px solid #21262d' : '1px solid #f0f0f0',
    borderRadius: 12,
  };

  const logColumns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 160,
      render: (v: string) => (
        <Text style={{ fontSize: 12, color: darkMode ? '#8b949e' : '#999' }}>
          {dayjs(v).format('MM-DD HH:mm:ss')}
        </Text>
      ),
    },
    {
      title: '级别',
      dataIndex: 'level',
      width: 70,
      render: (v: string) => (
        <Tag color={v === 'error' ? 'red' : v === 'warn' ? 'orange' : 'blue'} style={{ fontSize: 11 }}>
          {v?.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      render: (v: string) => <Text style={{ fontSize: 13 }}>{v}</Text>,
    },
    {
      title: '类型',
      dataIndex: 'slot_type',
      width: 80,
      render: (v: string) => (
        <Tag color={v === 'cloud' ? 'purple' : v === 'tpm2' ? 'cyan' : 'green'} style={{ fontSize: 11 }}>
          {v || 'local'}
        </Tag>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Title level={4} style={{ marginBottom: 24, color: darkMode ? '#c9d1d9' : undefined }}>
        系统概览
      </Title>

      {!connected && (
        <Alert
          message="未连接到 client-card 服务"
          description="请确保 client-card 服务已启动（默认端口 1026）"
          type="warning"
          showIcon
          style={{ marginBottom: 24 }}
        />
      )}

      <Spin spinning={loading}>
        {/* 统计卡片 */}
        <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
          <Col xs={24} sm={12} lg={6}>
            <Card style={cardStyle} bodyStyle={{ padding: '20px 24px' }}>
              <Statistic
                title={<Text style={{ color: darkMode ? '#8b949e' : '#666' }}>用户总数</Text>}
                value={userCount}
                prefix={<UserOutlined style={{ color: '#1677ff' }} />}
                valueStyle={{ color: darkMode ? '#c9d1d9' : undefined }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card style={cardStyle} bodyStyle={{ padding: '20px 24px' }}>
              <Statistic
                title={<Text style={{ color: darkMode ? '#8b949e' : '#666' }}>卡片总数</Text>}
                value={cardCount}
                prefix={<CreditCardOutlined style={{ color: '#52c41a' }} />}
                valueStyle={{ color: darkMode ? '#c9d1d9' : undefined }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card style={cardStyle} bodyStyle={{ padding: '20px 24px' }}>
              <Statistic
                title={<Text style={{ color: darkMode ? '#8b949e' : '#666' }}>活跃 Slot</Text>}
                value={slots.filter((s) => s.token_present).length}
                suffix={`/ ${slots.length}`}
                prefix={<SafetyCertificateOutlined style={{ color: '#722ed1' }} />}
                valueStyle={{ color: darkMode ? '#c9d1d9' : undefined }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card style={cardStyle} bodyStyle={{ padding: '20px 24px' }}>
              <Statistic
                title={<Text style={{ color: darkMode ? '#8b949e' : '#666' }}>服务状态</Text>}
                value={connected ? '在线' : '离线'}
                prefix={
                  connected
                    ? <CheckCircleOutlined style={{ color: '#52c41a' }} />
                    : <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                }
                valueStyle={{ color: connected ? '#52c41a' : '#ff4d4f' }}
              />
            </Card>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          {/* Slot 状态 */}
          <Col xs={24} lg={10}>
            <Card
              title={
                <Space>
                  <ApiOutlined />
                  <span>Slot 状态</span>
                </Space>
              }
              style={cardStyle}
              headStyle={{ borderBottom: darkMode ? '1px solid #21262d' : undefined }}
              loading={slotsLoading}
            >
              {slots.length === 0 ? (
                <Text style={{ color: darkMode ? '#8b949e' : '#999' }}>暂无 Slot</Text>
              ) : (
                <Space direction="vertical" style={{ width: '100%' }}>
                  {slots.map((slot) => (
                    <div
                      key={slot.slot_id}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        padding: '10px 12px',
                        background: darkMode ? '#0d1117' : '#fafafa',
                        borderRadius: 8,
                        border: darkMode ? '1px solid #21262d' : '1px solid #f0f0f0',
                      }}
                    >
                      <Space>
                        <Badge status={slot.token_present ? 'success' : 'default'} />
                        <Text style={{ fontSize: 13, color: darkMode ? '#c9d1d9' : undefined }}>
                          Slot #{slot.slot_id}
                        </Text>
                        <Text style={{ fontSize: 12, color: darkMode ? '#8b949e' : '#999' }}>
                          {slot.description}
                        </Text>
                      </Space>
                      <Tag color={slot.token_present ? 'success' : 'default'} style={{ fontSize: 11 }}>
                        {slot.token_present ? '已就绪' : '未就绪'}
                      </Tag>
                    </div>
                  ))}
                </Space>
              )}
            </Card>
          </Col>

          {/* 最近日志 */}
          <Col xs={24} lg={14}>
            <Card
              title={
                <Space>
                  <SafetyCertificateOutlined />
                  <span>最近操作日志</span>
                </Space>
              }
              style={cardStyle}
              headStyle={{ borderBottom: darkMode ? '1px solid #21262d' : undefined }}
            >
              <Table
                dataSource={recentLogs}
                columns={logColumns}
                rowKey="uuid"
                pagination={false}
                size="small"
                style={{ background: 'transparent' }}
              />
            </Card>
          </Col>
        </Row>
      </Spin>
    </div>
  );
};

export default Dashboard;
