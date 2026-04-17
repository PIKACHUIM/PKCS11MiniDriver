import React, { useEffect, useState } from 'react';
import {
  Card, Table, Button, Space, Tag, Modal, message,
  Descriptions, Typography, Tooltip, Drawer,
} from 'antd';
import {
  DeleteOutlined, EyeOutlined, ImportOutlined,
  ExportOutlined, SafetyCertificateOutlined, CopyOutlined,
  KeyOutlined, LockOutlined,
} from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import type { Certificate } from '../../types';
import { getCerts, deleteCert, exportCert } from '../../api';
import dayjs from 'dayjs';

const { Text, Paragraph } = Typography;

const certTypeColors: Record<string, string> = {
  x509: 'blue', ssh: 'green', gpg: 'purple', totp: 'orange',
  fido: 'cyan', login: 'gold', text: 'default', note: 'lime', payment: 'red',
};
const certTypeLabels: Record<string, string> = {
  x509: 'X.509 证书', ssh: 'SSH 密钥', gpg: 'GPG 证书', totp: 'TOTP 认证',
  fido: 'FIDO 认证', login: '登录信息', text: '密钥文本', note: '安全笔记', payment: '支付信息',
};

const CertsPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const cardUUID = searchParams.get('card') || '';

  const [certs, setCerts] = useState<Certificate[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedCert, setSelectedCert] = useState<Certificate | null>(null);

  const loadCerts = async () => {
    if (!cardUUID) return;
    setLoading(true);
    try {
      const data = await getCerts(cardUUID);
      setCerts(data || []);
    } catch (err: any) {
      message.error(err.message || '加载证书列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCerts();
  }, [cardUUID]);

  const handleDelete = (cert: Certificate) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除证书 ${cert.uuid.slice(0, 8)}... 吗？此操作不可恢复。`,
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteCert(cardUUID, cert.uuid);
          message.success('证书已删除');
          loadCerts();
        } catch (err: any) {
          message.error(err.message || '删除失败');
        }
      },
    });
  };

  const handleExport = async (cert: Certificate, format: string) => {
    try {
      const blob = await exportCert(cardUUID, cert.uuid, format);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${cert.uuid.slice(0, 8)}.${format}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err: any) {
      message.error(err.message || '导出失败');
    }
  };

  const showDetail = (cert: Certificate) => {
    setSelectedCert(cert);
    setDetailVisible(true);
  };

  const columns = [
    {
      title: '类型',
      dataIndex: 'cert_type',
      width: 120,
      render: (type: string) => (
        <Tag color={certTypeColors[type] || 'default'}>
          {certTypeLabels[type] || type}
        </Tag>
      ),
    },
    {
      title: '密钥类型',
      dataIndex: 'key_type',
      width: 120,
      render: (v: string) => <Tag icon={<KeyOutlined />}>{v?.toUpperCase() || '-'}</Tag>,
    },
    {
      title: '备注',
      dataIndex: 'remark',
      ellipsis: true,
      render: (v: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 180,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      width: 160,
      render: (_: unknown, record: Certificate) => (
        <Space>
          <Tooltip title="查看详情">
            <Button type="text" icon={<EyeOutlined />} onClick={() => showDetail(record)} />
          </Tooltip>
          <Tooltip title="删除">
            <Button type="text" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)} />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Card
        title={
          <Space>
            <SafetyCertificateOutlined />
            <span>证书管理</span>
            {cardUUID && <Tag>{`卡片: ${cardUUID.slice(0, 8)}...`}</Tag>}
          </Space>
        }
        extra={
          <Space>
            <Button icon={<ImportOutlined />}>导入证书</Button>
          </Space>
        }
      >
        {!cardUUID ? (
          <div style={{ textAlign: 'center', padding: 48 }}>
            <LockOutlined style={{ fontSize: 48, color: '#bbb', marginBottom: 16 }} />
            <div>
              <Text type="secondary">请从卡片管理页面选择一张卡片查看证书</Text>
            </div>
          </div>
        ) : (
          <Table
            rowKey="uuid"
            columns={columns}
            dataSource={certs}
            loading={loading}
            pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
          />
        )}
      </Card>

      {/* 证书详情抽屉 */}
      <Drawer
        title="证书详情"
        width={560}
        open={detailVisible}
        onClose={() => setDetailVisible(false)}
        extra={
          <Space>
            <Button icon={<ExportOutlined />} onClick={() => selectedCert && handleExport(selectedCert, 'pem')}>
              导出 PEM
            </Button>
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
              <Tag color={certTypeColors[selectedCert.cert_type]}>
                {certTypeLabels[selectedCert.cert_type] || selectedCert.cert_type}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="密钥类型">{selectedCert.key_type}</Descriptions.Item>
            <Descriptions.Item label="Slot 类型">{selectedCert.slot_type}</Descriptions.Item>
            <Descriptions.Item label="备注">{selectedCert.remark || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {dayjs(selectedCert.created_at).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
            {selectedCert.cert_content && (
              <Descriptions.Item label="证书内容">
                <Paragraph
                  copyable
                  ellipsis={{ rows: 4, expandable: true }}
                  style={{ fontFamily: 'monospace', fontSize: 12, marginBottom: 0 }}
                >
                  {atob(selectedCert.cert_content)}
                </Paragraph>
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Drawer>
    </div>
  );
};

export default CertsPage;
