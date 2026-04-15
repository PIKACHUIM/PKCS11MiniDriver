import React, { useEffect, useState } from 'react';
import {
  Card, Table, Button, Space, Tag, Modal, message,
  Descriptions, Typography, Tooltip, Select, Drawer, Form, InputNumber, Row, Col,
} from 'antd';
import {
  DeleteOutlined, EyeOutlined, ImportOutlined,
  ExportOutlined, SafetyCertificateOutlined, CopyOutlined,
  KeyOutlined, StopOutlined, SwapOutlined, ReloadOutlined,
} from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import type { Certificate, Card as CardType } from '../../types';
import { getCerts, deleteCert, listAllCerts, revokeCert, assignCert, renewCert, getCards } from '../../api';
import { useAuthStore } from '../../store/auth';
import dayjs from 'dayjs';

const { Text, Paragraph } = Typography;
const { Option } = Select;

const certTypeColors: Record<string, string> = {
  x509: 'blue', ssh: 'green', gpg: 'purple', totp: 'orange',
  fido: 'cyan', login: 'gold', text: 'default', note: 'lime', payment: 'red',
};
const certTypeLabels: Record<string, string> = {
  x509: 'X.509 证书', ssh: 'SSH 密钥', gpg: 'GPG 证书', totp: 'TOTP 认证',
  fido: 'FIDO 认证', login: '登录信息', text: '密钥文本', note: '安全笔记', payment: '支付信息',
};
const certStatusColor: Record<string, string> = { valid: 'green', revoked: 'red', expired: 'orange' };
const certStatusText: Record<string, string> = { valid: '有效', revoked: '已吊销', expired: '已过期' };

const CertsPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const cardUUID = searchParams.get('card') || '';
  const { role } = useAuthStore();

  const [certs, setCerts] = useState<Certificate[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedCert, setSelectedCert] = useState<Certificate | null>(null);
  const [allCards, setAllCards] = useState<CardType[]>([]);

  // 筛选条件
  const [filterCertType, setFilterCertType] = useState<string | undefined>();

  // 分配弹窗
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignTarget, setAssignTarget] = useState<string>('');
  const [assignCard, setAssignCard] = useState<string>('');

  // 续期弹窗
  const [renewOpen, setRenewOpen] = useState(false);
  const [renewTarget, setRenewTarget] = useState<string>('');
  const [renewDays, setRenewDays] = useState(365);

  const loadCerts = async (p = 1) => {
    setLoading(true);
    try {
      if (cardUUID) {
        // 卡片维度查询
        const data = await getCerts(cardUUID);
        setCerts(data || []);
        setTotal((data || []).length);
      } else {
        // 全局查询（Platform 接口）
        const res = await listAllCerts({
          card_uuid: cardUUID || undefined,
          cert_type: filterCertType,
          page: p, page_size: 20,
        });
        setCerts(res.items || []);
        setTotal(res.total);
      }
    } catch (err: any) {
      message.error(err.message || '加载证书列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCerts();
    getCards({ page: 1, page_size: 100 }).then(r => setAllCards(r.items || [])).catch(() => {});
  }, [cardUUID, filterCertType]);

  const handleDelete = (cert: Certificate) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除证书 ${cert.uuid.slice(0, 8)}... 吗？此操作不可恢复。`,
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteCert(cardUUID || cert.card_uuid, cert.uuid);
          message.success('证书已删除');
          loadCerts();
        } catch (err: any) { message.error(err.message || '删除失败'); }
      },
    });
  };

  const handleRevoke = (cert: Certificate) => {
    Modal.confirm({
      title: '确认吊销证书',
      content: `确定要吊销证书 ${cert.uuid.slice(0, 8)}... 吗？吊销后无法恢复。`,
      okType: 'danger',
      okText: '确认吊销',
      onOk: async () => {
        try {
          await revokeCert(cert.uuid);
          message.success('证书已吊销');
          loadCerts();
        } catch (err: any) { message.error(err.message || '吊销失败'); }
      },
    });
  };

  const handleAssign = async () => {
    if (!assignTarget || !assignCard) return;
    try {
      await assignCert(assignTarget, assignCard);
      message.success('证书已分配到智能卡');
      setAssignOpen(false);
      loadCerts();
    } catch (err: any) { message.error(err.message || '分配失败'); }
  };

  const handleRenew = async () => {
    if (!renewTarget) return;
    try {
      await renewCert(renewTarget, renewDays);
      message.success('证书续期成功');
      setRenewOpen(false);
      loadCerts();
    } catch (err: any) { message.error(err.message || '续期失败'); }
  };

  const columns = [
    {
      title: '类型', dataIndex: 'cert_type', width: 120,
      render: (type: string) => <Tag color={certTypeColors[type] || 'default'}>{certTypeLabels[type] || type}</Tag>,
    },
    {
      title: '密钥类型', dataIndex: 'key_type', width: 110,
      render: (v: string) => <Tag icon={<KeyOutlined />}>{v?.toUpperCase() || '-'}</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', width: 90,
      render: (v: string) => v ? <Tag color={certStatusColor[v]}>{certStatusText[v] || v}</Tag> : <Tag color="green">有效</Tag>,
    },
    {
      title: '备注', dataIndex: 'remark', ellipsis: true,
      render: (v: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: '关联卡片', dataIndex: 'card_uuid', width: 140,
      render: (v: string) => v ? <Text code style={{ fontSize: 11 }}>{v.slice(0, 12)}...</Text> : '-',
    },
    {
      title: '创建时间', dataIndex: 'created_at', width: 160,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作', width: 200,
      render: (_: unknown, record: Certificate) => (
        <Space size={4} wrap>
          <Tooltip title="查看详情">
            <Button type="text" size="small" icon={<EyeOutlined />} onClick={() => { setSelectedCert(record); setDetailVisible(true); }} />
          </Tooltip>
          {role === 'admin' && record.cert_type === 'x509' && (
            <>
              <Tooltip title="吊销证书">
                <Button type="text" size="small" danger icon={<StopOutlined />} onClick={() => handleRevoke(record)} />
              </Tooltip>
              <Tooltip title="分配到智能卡">
                <Button type="text" size="small" icon={<SwapOutlined />} onClick={() => { setAssignTarget(record.uuid); setAssignOpen(true); }} />
              </Tooltip>
              <Tooltip title="续期">
                <Button type="text" size="small" icon={<ReloadOutlined />} onClick={() => { setRenewTarget(record.uuid); setRenewOpen(true); }} />
              </Tooltip>
            </>
          )}
          <Tooltip title="删除">
            <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)} />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        title={
          <Space>
            <SafetyCertificateOutlined />
            <span>证书管理</span>
            {cardUUID && <Tag>{`卡片: ${cardUUID.slice(0, 8)}...`}</Tag>}
          </Space>
        }
        extra={<Button icon={<ImportOutlined />}>导入证书</Button>}
      >
        {/* 筛选栏 */}
        <Row gutter={12} style={{ marginBottom: 16 }}>
          <Col xs={24} sm={8} md={6}>
            <Select
              allowClear placeholder="证书类型" style={{ width: '100%' }}
              value={filterCertType} onChange={setFilterCertType}
            >
              {Object.entries(certTypeLabels).map(([k, v]) => <Option key={k} value={k}>{v}</Option>)}
            </Select>
          </Col>
        </Row>

        <Table
          rowKey="uuid"
          columns={columns}
          dataSource={certs}
          loading={loading}
          pagination={{
            current: page, total, pageSize: 20,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p) => { setPage(p); loadCerts(p); },
          }}
        />
      </Card>

      {/* 证书详情抽屉 */}
      <Drawer
        title="证书详情" width={560}
        open={detailVisible} onClose={() => setDetailVisible(false)}
        extra={
          <Space>
            <Button icon={<ExportOutlined />}>导出</Button>
            <Button icon={<CopyOutlined />} onClick={() => {
              if (selectedCert?.cert_content) {
                navigator.clipboard.writeText(atob(selectedCert.cert_content));
                message.success('已复制到剪贴板');
              }
            }}>复制</Button>
          </Space>
        }
      >
        {selectedCert && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="UUID">{selectedCert.uuid}</Descriptions.Item>
            <Descriptions.Item label="证书类型">
              <Tag color={certTypeColors[selectedCert.cert_type]}>{certTypeLabels[selectedCert.cert_type] || selectedCert.cert_type}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="密钥类型">{selectedCert.key_type}</Descriptions.Item>
            <Descriptions.Item label="Slot 类型">{selectedCert.slot_type}</Descriptions.Item>
            <Descriptions.Item label="备注">{selectedCert.remark || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{dayjs(selectedCert.created_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
            {selectedCert.cert_content && (
              <Descriptions.Item label="证书内容">
                <Paragraph copyable ellipsis={{ rows: 4, expandable: true }}
                  style={{ fontFamily: 'monospace', fontSize: 12, marginBottom: 0 }}>
                  {atob(selectedCert.cert_content)}
                </Paragraph>
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Drawer>

      {/* 分配到智能卡 */}
      <Modal title="分配证书到智能卡" open={assignOpen} onCancel={() => setAssignOpen(false)}
        onOk={handleAssign} okText="确认分配">
        <div style={{ marginTop: 16 }}>
          <Text>选择目标智能卡：</Text>
          <Select style={{ width: '100%', marginTop: 8 }} value={assignCard} onChange={setAssignCard} placeholder="请选择智能卡">
            {allCards.map(c => <Option key={c.uuid} value={c.uuid}>{c.card_name} ({c.slot_type})</Option>)}
          </Select>
        </div>
      </Modal>

      {/* 续期 */}
      <Modal title="证书续期" open={renewOpen} onCancel={() => setRenewOpen(false)}
        onOk={handleRenew} okText="确认续期">
        <div style={{ marginTop: 16 }}>
          <Text>新增有效天数：</Text>
          <InputNumber min={1} max={3650} value={renewDays} onChange={v => setRenewDays(v || 365)}
            style={{ width: '100%', marginTop: 8 }} addonAfter="天" />
        </div>
      </Modal>
    </div>
  );
};

export default CertsPage;
