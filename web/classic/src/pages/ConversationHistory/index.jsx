/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Empty,
  Form,
  Modal,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import { Eye, MessagesSquare } from 'lucide-react';
import { API, showError, timestamp2string } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

const { Text, Title } = Typography;

function parseDateValue(value) {
  if (!value) return undefined;
  if (value instanceof Date) return Math.floor(value.getTime() / 1000);
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? undefined : Math.floor(timestamp / 1000);
}

function formatTokens(value) {
  return Number(value || 0).toLocaleString();
}

function getFilters(formApi) {
  const values = formApi?.getValues() || {};
  const dateRange = values.dateRange || [];

  return {
    content: values.content || undefined,
    username: values.username || undefined,
    token_name: values.token_name || undefined,
    model_name: values.model_name || undefined,
    group: values.group || undefined,
    request_id: values.request_id || undefined,
    session_id: values.session_id || undefined,
    exported:
      values.exported === '' || values.exported === undefined
        ? undefined
        : values.exported === 'true',
    start_timestamp: parseDateValue(dateRange[0]),
    end_timestamp: parseDateValue(dateRange[1]),
  };
}

function RawBlock({ value }) {
  return (
    <pre className='max-h-72 overflow-auto whitespace-pre-wrap break-words rounded-md border border-gray-200 bg-gray-50 p-3 text-xs leading-relaxed'>
      {value || '-'}
    </pre>
  );
}

function MessageBlock({ title, value, tone }) {
  const toneClass =
    tone === 'user'
      ? 'border-blue-200 bg-blue-50 text-blue-950'
      : 'border-gray-200 bg-white text-gray-900';

  return (
    <div className='space-y-1'>
      <Text type='secondary' size='small'>
        {title}
      </Text>
      <div
        className={`whitespace-pre-wrap break-words rounded-md border px-3 py-2 text-sm leading-relaxed ${toneClass}`}
      >
        {value || '-'}
      </div>
    </div>
  );
}

function RawDetails({ title, value }) {
  return (
    <details className='rounded-md border border-gray-200 bg-gray-50/40'>
      <summary className='cursor-pointer px-3 py-2 text-xs font-medium text-gray-500'>
        {title}
      </summary>
      <div className='border-t border-gray-200 p-3'>
        <RawBlock value={value} />
      </div>
    </details>
  );
}

function ConversationDetailModal({ visible, onClose, session, filters }) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState(null);

  const loadDetail = useCallback(async () => {
    if (!session?.session_key) return;

    setLoading(true);
    try {
      const res = await API.get('/api/conversation_log/sessions/detail', {
        params: {
          ...filters,
          session_key: session.session_key,
          limit: 200,
        },
      });
      const { success, message, data } = res.data;
      if (success) {
        setDetail(data);
      } else {
        showError(message);
      }
    } finally {
      setLoading(false);
    }
  }, [filters, session?.session_key]);

  useEffect(() => {
    if (visible) {
      loadDetail();
    } else {
      setDetail(null);
    }
  }, [visible, loadDetail]);

  return (
    <Modal
      title={session?.latest_user_text || t('对话详情')}
      visible={visible}
      onCancel={onClose}
      footer={<Button onClick={onClose}>{t('关闭')}</Button>}
      width={980}
    >
      <div className='max-h-[72vh] overflow-y-auto space-y-4 pr-2'>
        <div className='grid gap-2 rounded-md border border-gray-200 bg-gray-50 p-3 text-xs text-gray-600 sm:grid-cols-2 lg:grid-cols-4'>
          <span>
            {t('用户')}: {session?.username || `#${session?.user_id || '-'}`}
          </span>
          <span>
            {t('令牌')}: {session?.token_names?.join(', ') || '-'}
          </span>
          <span>
            {t('分组')}: {session?.groups?.join(', ') || '-'}
          </span>
          <span>
            {t('会话 ID')}: {session?.session_id || session?.session_key || '-'}
          </span>
          <span>
            {t('消息数')}: {session?.log_count || '-'}
          </span>
          <span>
            Tokens: {formatTokens(session?.prompt_tokens)} /{' '}
            {formatTokens(session?.completion_tokens)}
          </span>
        </div>
        {loading && (
          <div className='flex items-center justify-center py-10'>
            <Spin size='large' tip={t('加载中...')} />
          </div>
        )}
        {!loading && detail?.truncated && (
          <Tag color='amber'>{t('仅显示该会话最新 200 条记录')}</Tag>
        )}
        {!loading && detail?.logs?.length === 0 && (
          <Empty description={t('没有找到对话记录')} />
        )}
        {!loading &&
          detail?.logs?.map((log) => (
            <Card
              key={log.id}
              className='!rounded-lg'
              bodyStyle={{ padding: 16 }}
              title={
                <Space wrap>
                  <Tag color='blue'>#{log.id}</Tag>
                  <Text type='secondary'>
                    {timestamp2string(log.created_at)}
                  </Text>
                  <Text type='secondary'>
                    {log.username || `#${log.user_id}`}
                  </Text>
                  {log.token_name && <Tag color='cyan'>{log.token_name}</Tag>}
                  {log.group && <Tag color='green'>{log.group}</Tag>}
                  {log.model_name && <Tag color='grey'>{log.model_name}</Tag>}
                  {log.channel_name && <Text>{log.channel_name}</Text>}
                  <Text type='secondary'>
                    {formatTokens(log.prompt_tokens)} /{' '}
                    {formatTokens(log.completion_tokens)}
                  </Text>
                </Space>
              }
            >
              <div className='space-y-3'>
                <MessageBlock
                  title={t('用户消息')}
                  value={log.user_text}
                  tone='user'
                />
                <MessageBlock
                  title={t('助手回复')}
                  value={log.assistant_text}
                  tone='assistant'
                />
                <RawDetails title={t('原始请求')} value={log.request_body} />
                <RawDetails title={t('原始响应')} value={log.response_body} />
                {log.response_reasoning_body && (
                  <RawDetails
                    title={t('推理内容')}
                    value={log.response_reasoning_body}
                  />
                )}
              </div>
            </Card>
          ))}
      </div>
    </Modal>
  );
}

export default function ConversationHistory() {
  const { t } = useTranslation();
  const [formApi, setFormApi] = useState(null);
  const [sessions, setSessions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [total, setTotal] = useState(0);
  const [detailOpen, setDetailOpen] = useState(false);
  const [selectedSession, setSelectedSession] = useState(null);
  const [activeFilters, setActiveFilters] = useState({});

  const loadSessions = useCallback(
    async (page = activePage, size = pageSize) => {
      const filters = getFilters(formApi);
      setLoading(true);
      try {
        const res = await API.get('/api/conversation_log/sessions', {
          params: {
            ...filters,
            p: page,
            page_size: size,
          },
        });
        const { success, message, data } = res.data;
        if (success) {
          setSessions(data.items || []);
          setTotal(data.total || 0);
          setActivePage(data.page || page);
          setPageSize(data.page_size || size);
          setActiveFilters(filters);
        } else {
          showError(message);
        }
      } finally {
        setLoading(false);
      }
    },
    [activePage, formApi, pageSize],
  );

  useEffect(() => {
    loadSessions(1, pageSize);
  }, [formApi]);

  const columns = useMemo(
    () => [
      {
        title: t('对话'),
        dataIndex: 'latest_user_text',
        width: 360,
        render: (_, record) => (
          <div className='max-w-[520px]'>
            <div className='flex items-center gap-2 font-medium'>
              <MessagesSquare size={16} />
              <span className='truncate'>
                {record.latest_user_text || t('未命名对话')}
              </span>
            </div>
            <div className='mt-1 line-clamp-2 text-xs text-gray-500'>
              {record.latest_assistant_text || '-'}
            </div>
          </div>
        ),
      },
      {
        title: t('用户'),
        dataIndex: 'username',
        width: 120,
        render: (_, record) => record.username || `#${record.user_id}`,
      },
      {
        title: t('令牌/分组'),
        width: 220,
        render: (_, record) => (
          <Space wrap>
            {(record.token_names || []).slice(0, 2).map((token) => (
              <Tag key={token} color='cyan'>
                {token}
              </Tag>
            ))}
            {(record.groups || []).slice(0, 2).map((group) => (
              <Tag key={group} color='green'>
                {group}
              </Tag>
            ))}
            {(!record.token_names || record.token_names.length === 0) &&
              (!record.groups || record.groups.length === 0) &&
              '-'}
          </Space>
        ),
      },
      {
        title: t('模型'),
        dataIndex: 'models',
        width: 220,
        render: (models = []) => (
          <Space wrap>
            {models.slice(0, 3).map((model) => (
              <Tag key={model} color='grey'>
                {model}
              </Tag>
            ))}
            {models.length > 3 && <Tag>+{models.length - 3}</Tag>}
            {models.length === 0 && '-'}
          </Space>
        ),
      },
      {
        title: t('消息数'),
        dataIndex: 'log_count',
        width: 90,
      },
      {
        title: t('Tokens'),
        width: 150,
        render: (_, record) =>
          `${formatTokens(record.prompt_tokens)} / ${formatTokens(record.completion_tokens)}`,
      },
      {
        title: t('最后活跃'),
        dataIndex: 'last_created_at',
        width: 170,
        render: (value) => timestamp2string(value),
      },
      {
        title: t('操作'),
        width: 100,
        render: (_, record) => (
          <Button
            theme='borderless'
            icon={<Eye size={15} />}
            onClick={() => {
              setSelectedSession(record);
              setDetailOpen(true);
            }}
          >
            {t('详情')}
          </Button>
        ),
      },
    ],
    [t],
  );

  const resetFilters = () => {
    formApi?.reset();
    setTimeout(() => loadSessions(1, pageSize), 100);
  };

  return (
    <div className='mt-[60px] px-2'>
      <Card className='!rounded-2xl shadow-sm border-0'>
        <div className='mb-4 flex flex-col gap-2 md:flex-row md:items-center md:justify-between'>
          <div>
            <Title heading={4}>{t('对话历史')}</Title>
            <Text type='secondary'>
              {t('按会话查看已采集的用户消息、助手回复和原始请求响应')}
            </Text>
          </div>
          <Button onClick={() => loadSessions(1, pageSize)} loading={loading}>
            {t('刷新')}
          </Button>
        </div>

        <Form
          getFormApi={setFormApi}
          onSubmit={() => loadSessions(1, pageSize)}
          allowEmpty
          layout='vertical'
          trigger='change'
        >
          <div className='grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-6'>
            <Form.Input
              field='content'
              prefix={<IconSearch />}
              placeholder={t('搜索用户或助手内容')}
              showClear
              pure
              size='small'
            />
            <Form.Input
              field='username'
              placeholder={t('用户名')}
              showClear
              pure
              size='small'
            />
            <Form.Input
              field='token_name'
              placeholder={t('令牌')}
              showClear
              pure
              size='small'
            />
            <Form.Input
              field='model_name'
              placeholder={t('模型')}
              showClear
              pure
              size='small'
            />
            <Form.Input
              field='group'
              placeholder={t('分组')}
              showClear
              pure
              size='small'
            />
            <Form.Input
              field='request_id'
              placeholder={t('请求 ID')}
              showClear
              pure
              size='small'
            />
            <Form.Input
              field='session_id'
              placeholder={t('会话 ID')}
              showClear
              pure
              size='small'
            />
            <Form.Select
              field='exported'
              placeholder={t('导出状态')}
              showClear
              pure
              size='small'
              optionList={[
                { label: t('未导出'), value: 'false' },
                { label: t('已导出'), value: 'true' },
              ]}
            />
            <div className='md:col-span-2 lg:col-span-3'>
              <Form.DatePicker
                field='dateRange'
                className='w-full'
                type='dateTimeRange'
                placeholder={[t('开始时间'), t('结束时间')]}
                showClear
                pure
                size='small'
              />
            </div>
            <div className='flex gap-2 md:col-span-2 lg:col-span-3 lg:justify-end'>
              <Button htmlType='submit' loading={loading} size='small'>
                {t('查询')}
              </Button>
              <Button type='tertiary' onClick={resetFilters} size='small'>
                {t('重置')}
              </Button>
            </div>
          </div>
        </Form>

        <div className='mt-4'>
          <Table
            columns={columns}
            dataSource={sessions}
            rowKey='session_key'
            loading={loading}
            scroll={{ x: 'max-content' }}
            empty={<Empty description={t('没有找到对话历史')} />}
            pagination={{
              currentPage: activePage,
              pageSize,
              total,
              pageSizeOptions: [10, 20, 50, 100],
              showSizeChanger: true,
              onPageChange: (page) => loadSessions(page, pageSize),
              onPageSizeChange: (size) => loadSessions(1, size),
            }}
          />
        </div>
      </Card>
      <ConversationDetailModal
        visible={detailOpen}
        onClose={() => setDetailOpen(false)}
        session={selectedSession}
        filters={activeFilters}
      />
    </div>
  );
}
