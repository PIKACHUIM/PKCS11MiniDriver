import React, { useState, useEffect } from 'react';
import {
  Tabs, Table, Button, Modal, Form, Input, InputNumber, Select,
  Typography, Space, Tag, Statistic, Row, Col, Card, message,
} from 'antd';
import { WalletOutlined, PlusOutlined } from '@ant-design/icons';
import {
  getBalance, createRecharge, listPaymentOrders, createRefund, listPaymentPlugins,
} from '../../api';
import type { UserBalance, PaymentOrder, PaymentPlugin } from '../../types';

const { Title } = Typography;
const { Option } = Select;

const statusColor: Record<string, string> = {
  pending: 'orange', paid: 'green', failed: 'red', refunded: 'blue',
};
const statusText: Record<string, string> = {
  pending: '待支付', paid: '已支付', failed: '失败', refunded: '已退款',
};

const Payment: React.FC = () => {
  const [balance, setBalance] = useState<UserBalance | null>(null);
  const [orders, setOrders] = useState<PaymentOrder[]>([]);
  const [ordersTotal, setOrdersTotal] = useState(0);
  const [ordersPage, setOrdersPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [plugins, setPlugins] = useState<PaymentPlugin[]>([]);

  // 充值弹窗
  const [rechargeOpen, setRechargeOpen] = useState(false);
  const [rechargeLoading, setRechargeLoading] = useState(false);
  const [rechargeForm] = Form.useForm();

  // 退款弹窗
  const [refundOpen, setRefundOpen] = useState(false);
  const [refundLoading, setRefundLoading] = useState(false);
  const [refundForm] = Form.useForm();

  const loadBalance = async () => {
    try { setBalance(await getBalance()); } catch {}
  };

  const loadOrders = async (p = 1) => {
    setLoading(true);
    try {
      const res = await listPaymentOrders({ page: p, page_size: 20 });
      setOrders(res.items || []);
      setOrdersTotal(res.total);
    } catch (e: any) { message.error(e.message); }
    finally { setLoading(false); }
  };

  const loadPlugins = async () => {
    try { setPlugins(await listPaymentPlugins()); } catch {}
  };

  useEffect(() => {
    loadBalance();
    loadOrders();
    loadPlugins();
  }, []);

  const handleRecharge = async (values: { amount: number; channel: string }) => {
    setRechargeLoading(true);
    try {
      const order = await createRecharge({ amount: Math.round(values.amount * 100), channel: values.channel });
      message.success('充值订单已创建');
      if (order.pay_url) window.open(order.pay_url, '_blank');
      setRechargeOpen(false);
      rechargeForm.resetFields();
      loadOrders();
      loadBalance();
    } catch (e: any) { message.error(e.message); }
    finally { setRechargeLoading(false); }
  };

  const handleRefund = async (values: { order_uuid: string; reason: string }) => {
    setRefundLoading(true);
    try {
      await createRefund(values);
      message.success('退款申请已提交，等待管理员审批');
      setRefundOpen(false);
      refundForm.resetFields();
    } catch (e: any) { message.error(e.message); }
    finally { setRefundLoading(false); }
  };

  const orderColumns = [
    { title: '订单号', dataIndex: 'uuid', key: 'uuid', render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v.slice(0, 16)}...</span> },
    { title: '金额', dataIndex: 'amount', key: 'amount', render: (v: number) => `¥${(v / 100).toFixed(2)}` },
    { title: '支付渠道', dataIndex: 'channel', key: 'channel' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (v: string) => <Tag color={statusColor[v]}>{statusText[v] || v}</Tag> },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: (v: string) => v?.slice(0, 19) },
    {
      title: '操作', key: 'action', render: (_: any, r: PaymentOrder) => (
        r.status === 'paid' ? (
          <Button size="small" onClick={() => { refundForm.setFieldValue('order_uuid', r.uuid); setRefundOpen(true); }}>
            申请退款
          </Button>
        ) : null
      ),
    },
  ];

  return (
    <div>
      <Title level={4} style={{ margin: '0 0 16px' }}>支付与订单</Title>
      <Tabs items={[
        {
          key: 'balance',
          label: '余额概览',
          children: (
            <div>
              <Row gutter={24} style={{ marginBottom: 24 }}>
                <Col xs={24} sm={8}>
                  <Card>
                    <Statistic
                      title="可用余额"
                      value={balance ? (balance.available / 100).toFixed(2) : '—'}
                      prefix={<WalletOutlined />}
                      suffix="元"
                      valueStyle={{ color: '#1677ff' }}
                    />
                  </Card>
                </Col>
                <Col xs={24} sm={8}>
                  <Card>
                    <Statistic
                      title="累计充值"
                      value={balance ? (balance.total_recharged / 100).toFixed(2) : '—'}
                      suffix="元"
                      valueStyle={{ color: '#52c41a' }}
                    />
                  </Card>
                </Col>
                <Col xs={24} sm={8}>
                  <Card>
                    <Statistic
                      title="累计消费"
                      value={balance ? (balance.total_consumed / 100).toFixed(2) : '—'}
                      suffix="元"
                      valueStyle={{ color: '#fa8c16' }}
                    />
                  </Card>
                </Col>
              </Row>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setRechargeOpen(true)}>
                立即充值
              </Button>
            </div>
          ),
        },
        {
          key: 'orders',
          label: '充值记录',
          children: (
            <Table
              rowKey="uuid"
              columns={orderColumns}
              dataSource={orders}
              loading={loading}
              pagination={{ current: ordersPage, total: ordersTotal, pageSize: 20, onChange: (p) => { setOrdersPage(p); loadOrders(p); } }}
            />
          ),
        },
        {
          key: 'refund',
          label: '退款申请',
          children: (
            <div style={{ maxWidth: 480 }}>
              <Form layout="vertical" form={refundForm} onFinish={handleRefund}>
                <Form.Item label="订单号" name="order_uuid" rules={[{ required: true, message: '请输入订单号' }]}>
                  <Input placeholder="请输入需要退款的订单号" />
                </Form.Item>
                <Form.Item label="退款原因" name="reason" rules={[{ required: true, message: '请输入退款原因' }]}>
                  <Input.TextArea rows={3} placeholder="请描述退款原因" />
                </Form.Item>
                <Form.Item>
                  <Button type="primary" htmlType="submit" loading={refundLoading}>提交退款申请</Button>
                </Form.Item>
              </Form>
            </div>
          ),
        },
      ]} />

      {/* 充值弹窗 */}
      <Modal title="立即充值" open={rechargeOpen} onCancel={() => setRechargeOpen(false)} footer={null}>
        <Form form={rechargeForm} layout="vertical" onFinish={handleRecharge} style={{ marginTop: 16 }}>
          <Form.Item label="充值金额（元）" name="amount" rules={[{ required: true, message: '请输入充值金额' }, { type: 'number', min: 1, message: '最低充值 1 元' }]}>
            <InputNumber min={1} precision={2} style={{ width: '100%' }} placeholder="请输入充值金额" addonAfter="元" />
          </Form.Item>
          <Form.Item label="支付渠道" name="channel" rules={[{ required: true, message: '请选择支付渠道' }]}>
            <Select placeholder="请选择支付渠道">
              {plugins.filter(p => p.enabled).map(p => (
                <Option key={p.uuid} value={p.plugin_type}>{p.name}</Option>
              ))}
              {plugins.length === 0 && <Option value="default">默认渠道</Option>}
            </Select>
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setRechargeOpen(false)}>取消</Button>
              <Button type="primary" htmlType="submit" loading={rechargeLoading}>确认充值</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 退款弹窗（从订单列表触发） */}
      <Modal title="申请退款" open={refundOpen} onCancel={() => setRefundOpen(false)} footer={null}>
        <Form form={refundForm} layout="vertical" onFinish={handleRefund} style={{ marginTop: 16 }}>
          <Form.Item label="订单号" name="order_uuid" rules={[{ required: true }]}>
            <Input readOnly />
          </Form.Item>
          <Form.Item label="退款原因" name="reason" rules={[{ required: true, message: '请输入退款原因' }]}>
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setRefundOpen(false)}>取消</Button>
              <Button type="primary" danger htmlType="submit" loading={refundLoading}>提交申请</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Payment;
