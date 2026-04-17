import React, { useEffect, useState } from 'react';
import {
  Card, Form, Select, Button, Space, Input, message,
  Row, Col, Descriptions, Tag, Alert, Divider,
} from 'antd';
import { SafetyCertificateOutlined, ReloadOutlined } from '@ant-design/icons';
import { getCSRList, selfSignFromCSR } from '../../api';
import type { CSRRecord } from '../../types';
import { useAppStore } from '../../store/appStore';
import dayjs from 'dayjs';

const VALIDITY_OPTIONS = [
  { label: '30 天', value: 30 },
  { label: '90 天', value: 90 },
  { label: '180 天', value: 180 },
  { label: '1 年（365 天）', value: 365 },
  { label: '2 年（730 天）', value: 730 },
  { label: '3 年（1095 天）', value: 1095 },
  { label: '5 年（1825 天）', value: 1825 },
];

const SelfSignPage: React.FC = () => {
  const { darkMode } = useAppStore();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [csrList, setCSRList] = useState<CSRRecord[]>([]);
  const [csrLoading, setCSRLoading] = useState(false);
  const [selectedCSR, setSelectedCSR] = useState<CSRRecord | null>(null);

  const loadCSRs = async () => {
    setCSRLoading(true);
    try {
      const res = await getCSRList({ page: 1, page_size: 100 });
      // 只显示有私钥的 CSR（database 模式），才能自签名
      setCSRList(res.items.filter((c) => c.has_private_key && c.key_storage === 'database'));
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setCSRLoading(false);
    }
  };

  useEffect(() => { loadCSRs(); }, []);

  const handleCSRChange = (uuid: string) => {
    const csr = csrList.find((c) => c.uuid === uuid) ?? null;
    setSelectedCSR(csr);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await selfSignFromCSR(values.csr_uuid, values.validity_days, values.remark);
      message.success('自签名证书已生成，可在「证书管理」中查看');
      form.resetFields();
      setSelectedCSR(null);
    } catch (err: any) {
      if (err.message) message.error(err.message);
    } finally {
      setLoading(false);
    }
  };

  const cardStyle = {
    background: darkMode ? '#161b22' : '#fff',
    border: darkMode ? '1px solid #21262d' : '1px solid #f0f0f0',
    borderRadius: 12,
  };

  return (
    <div style={{ padding: 24 }}>
      <Card
        style={cardStyle}
        title={
          <Space>
            <SafetyCertificateOutlined />
            <span>自签名证书</span>
          </Space>
        }
        extra={
          <Button size="small" icon={<ReloadOutlined />} onClick={loadCSRs} loading={csrLoading}>
            刷新 CSR 列表
          </Button>
        }
      >
        <Alert
          type="info"
          showIcon
          message="自签名证书将直接使用 CSR 中的主体信息和密钥对进行签名，无需 CA 参与。生成的证书会保存到「证书管理」中。"
          style={{ marginBottom: 20 }}
        />

        <Form form={form} layout="vertical" initialValues={{ validity_days: 365 }}>
          <Form.Item
            name="csr_uuid"
            label="选择 CSR（仅显示含私钥的数据库 CSR）"
            rules={[{ required: true, message: '请选择一个 CSR' }]}
          >
            <Select
              placeholder="选择要签名的 CSR"
              loading={csrLoading}
              onChange={handleCSRChange}
              options={csrList.map((c) => ({
                value: c.uuid,
                label: `${c.common_name}${c.organization ? ' — ' + c.organization : ''} (${c.key_type})`,
              }))}
              notFoundContent={
                csrLoading ? '加载中...' : '暂无含私钥的 CSR，请先在「CSR 管理」中生成一个存储到数据库的 CSR'
              }
            />
          </Form.Item>

          {/* 选中 CSR 后展示主体信息预览 */}
          {selectedCSR && (
            <>
              <Divider orientation="left" style={{ fontSize: 13 }}>CSR 主体信息预览</Divider>
              <Descriptions size="small" column={3} bordered style={{ marginBottom: 16 }}>
                <Descriptions.Item label="通用名称 (CN)">{selectedCSR.common_name}</Descriptions.Item>
                <Descriptions.Item label="组织 (O)">{selectedCSR.organization || '-'}</Descriptions.Item>
                <Descriptions.Item label="部门 (OU)">{selectedCSR.org_unit || '-'}</Descriptions.Item>
                <Descriptions.Item label="国家 (C)">{selectedCSR.country || '-'}</Descriptions.Item>
                <Descriptions.Item label="省份 (ST)">{selectedCSR.state || '-'}</Descriptions.Item>
                <Descriptions.Item label="城市 (L)">{selectedCSR.locality || '-'}</Descriptions.Item>
                <Descriptions.Item label="密钥类型">
                  <Tag color="blue">{selectedCSR.key_type?.toUpperCase()}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="密钥存储">
                  <Tag color="green">数据库</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="创建时间">
                  {dayjs(selectedCSR.created_at).format('YYYY-MM-DD HH:mm')}
                </Descriptions.Item>
                {selectedCSR.san_dns && (
                  <Descriptions.Item label="DNS SAN" span={3}>{selectedCSR.san_dns}</Descriptions.Item>
                )}
                {selectedCSR.san_ip && (
                  <Descriptions.Item label="IP SAN" span={3}>{selectedCSR.san_ip}</Descriptions.Item>
                )}
              </Descriptions>
            </>
          )}

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="validity_days" label="有效期" rules={[{ required: true }]}>
                <Select options={VALIDITY_OPTIONS} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="remark" label="备注">
                <Input placeholder="可选备注" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item>
            <Button
              type="primary"
              icon={<SafetyCertificateOutlined />}
              onClick={handleSubmit}
              loading={loading}
              disabled={!selectedCSR}
            >
              生成自签名证书
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default SelfSignPage;
