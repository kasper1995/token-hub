/*
Copyright (C) 2023-2026 QuantumNous

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
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import {
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
} from '@tanstack/react-table'
import { Eye, MessagesSquare } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  formatTimestampForInput,
  formatTimestampToDate,
  formatTokens,
  parseTimestampFromInput,
} from '@/lib/format'
import { cn } from '@/lib/utils'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DataTableColumnHeader, DataTablePage } from '@/components/data-table'
import { getConversationSessions } from '../api'
import type {
  ConversationSessionSummary,
  GetConversationSessionDetailParams,
  GetConversationSessionsParams,
} from '../types'
import { ConversationDetailSheet } from './conversation-detail-sheet'

const route = getRouteApi('/_authenticated/conversation-history/')

function valuesPreview(values: string[], maxVisible = 3) {
  const visible = values.slice(0, maxVisible)
  const hidden = Math.max(0, values.length - visible.length)
  return { visible, hidden }
}

function normalizeExported(value?: string) {
  if (value === 'true') return true
  if (value === 'false') return false
  return undefined
}

export function ConversationHistoryTable() {
  const { t } = useTranslation()
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const [selectedSession, setSelectedSession] =
    useState<ConversationSessionSummary | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)

  const {
    globalFilter,
    onGlobalFilterChange,
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search,
    navigate,
    pagination: { defaultPage: 1, defaultPageSize: 20 },
    globalFilter: { enabled: true, key: 'content' },
  })

  const setFilter = useCallback(
    (patch: Record<string, unknown>) => {
      navigate({
        search: (prev) => ({
          ...prev,
          page: undefined,
          ...patch,
        }),
      })
    },
    [navigate]
  )

  const hasAdditionalFilters = Boolean(
    search.username ||
    search.tokenName ||
    search.modelName ||
    search.group ||
    search.requestId ||
    search.sessionId ||
    search.exported ||
    search.startTime ||
    search.endTime
  )

  const resetFilters = useCallback(() => {
    navigate({
      search: (prev) => ({
        ...prev,
        page: undefined,
        content: undefined,
        username: undefined,
        tokenName: undefined,
        modelName: undefined,
        group: undefined,
        requestId: undefined,
        sessionId: undefined,
        exported: undefined,
        startTime: undefined,
        endTime: undefined,
      }),
    })
  }, [navigate])

  const queryParams = useMemo<GetConversationSessionsParams>(
    () => ({
      p: pagination.pageIndex + 1,
      page_size: pagination.pageSize,
      content: globalFilter || undefined,
      username: search.username || undefined,
      token_name: search.tokenName || undefined,
      model_name: search.modelName || undefined,
      group: search.group || undefined,
      request_id: search.requestId || undefined,
      session_id: search.sessionId || undefined,
      exported: normalizeExported(search.exported),
      start_timestamp: search.startTime,
      end_timestamp: search.endTime,
    }),
    [
      globalFilter,
      pagination.pageIndex,
      pagination.pageSize,
      search.endTime,
      search.exported,
      search.group,
      search.modelName,
      search.requestId,
      search.sessionId,
      search.startTime,
      search.tokenName,
      search.username,
    ]
  )

  const detailFilters = useMemo<
    Omit<GetConversationSessionDetailParams, 'session_key' | 'limit'>
  >(
    () => ({
      content: queryParams.content,
      username: queryParams.username,
      token_name: queryParams.token_name,
      model_name: queryParams.model_name,
      group: queryParams.group,
      request_id: queryParams.request_id,
      session_id: queryParams.session_id,
      exported: queryParams.exported,
      start_timestamp: queryParams.start_timestamp,
      end_timestamp: queryParams.end_timestamp,
    }),
    [queryParams]
  )

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['conversation-sessions', queryParams],
    queryFn: () => getConversationSessions(queryParams),
    placeholderData: (previousData) => previousData,
  })

  const sessions = data?.data.items || []
  const totalCount = data?.data.total || 0

  const columns = useMemo<ColumnDef<ConversationSessionSummary>[]>(
    () => [
      {
        id: 'summary',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Conversation')} />
        ),
        cell: ({ row }) => {
          const session = row.original
          return (
            <div className='max-w-[560px] space-y-1.5'>
              <div className='flex items-center gap-2'>
                <MessagesSquare className='text-muted-foreground h-4 w-4 shrink-0' />
                <span className='truncate text-sm font-medium'>
                  {session.latest_user_text || t('Untitled conversation')}
                </span>
              </div>
              <div className='text-muted-foreground line-clamp-2 text-xs leading-relaxed'>
                {session.latest_assistant_text || '-'}
              </div>
            </div>
          )
        },
        meta: { mobileTitle: true },
      },
      {
        accessorKey: 'username',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('User')} />
        ),
        cell: ({ row }) => row.original.username || `#${row.original.user_id}`,
        meta: { label: t('User') },
      },
      {
        accessorKey: 'models',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Models')} />
        ),
        cell: ({ row }) => {
          const { visible, hidden } = valuesPreview(row.original.models || [])
          return (
            <div className='flex max-w-[260px] flex-wrap gap-1'>
              {visible.map((model) => (
                <Badge
                  key={model}
                  variant='secondary'
                  className='max-w-40 truncate'
                >
                  {model}
                </Badge>
              ))}
              {hidden > 0 && <Badge variant='outline'>+{hidden}</Badge>}
              {visible.length === 0 && (
                <span className='text-muted-foreground'>-</span>
              )}
            </div>
          )
        },
        meta: { label: t('Models') },
      },
      {
        id: 'token_group',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Token')} />
        ),
        cell: ({ row }) => {
          const tokenPreview = valuesPreview(row.original.token_names || [], 2)
          const groupPreview = valuesPreview(row.original.groups || [], 2)
          return (
            <div className='max-w-[260px] space-y-1'>
              <div className='flex flex-wrap gap-1'>
                {tokenPreview.visible.map((token) => (
                  <Badge
                    key={token}
                    variant='outline'
                    className='max-w-40 truncate'
                  >
                    {token}
                  </Badge>
                ))}
                {tokenPreview.hidden > 0 && (
                  <Badge variant='outline'>+{tokenPreview.hidden}</Badge>
                )}
                {tokenPreview.visible.length === 0 && (
                  <span className='text-muted-foreground'>-</span>
                )}
              </div>
              <div className='flex flex-wrap gap-1'>
                {groupPreview.visible.map((group) => (
                  <Badge
                    key={group}
                    variant='secondary'
                    className='max-w-32 truncate'
                  >
                    {group}
                  </Badge>
                ))}
                {groupPreview.hidden > 0 && (
                  <Badge variant='outline'>+{groupPreview.hidden}</Badge>
                )}
              </div>
            </div>
          )
        },
        meta: { label: t('Token') },
      },
      {
        accessorKey: 'log_count',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Messages')} />
        ),
        cell: ({ row }) => row.original.log_count,
        meta: { label: t('Messages'), mobileBadge: true },
      },
      {
        id: 'tokens',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Tokens')} />
        ),
        cell: ({ row }) =>
          `${formatTokens(row.original.prompt_tokens)} / ${formatTokens(row.original.completion_tokens)}`,
        meta: { label: t('Tokens') },
      },
      {
        accessorKey: 'last_created_at',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('Last Active')} />
        ),
        cell: ({ row }) => formatTimestampToDate(row.original.last_created_at),
        meta: { label: t('Last Active') },
      },
      {
        id: 'actions',
        cell: ({ row }) => (
          <Button
            variant='ghost'
            size='sm'
            onClick={() => {
              setSelectedSession(row.original)
              setDetailOpen(true)
            }}
          >
            <Eye className='mr-1 h-4 w-4' />
            {t('Details')}
          </Button>
        ),
        meta: { mobileHidden: true },
      },
    ],
    [t]
  )

  const table = useReactTable({
    data: sessions,
    columns,
    pageCount: Math.ceil(totalCount / pagination.pageSize),
    state: { pagination, globalFilter },
    onPaginationChange,
    onGlobalFilterChange,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    manualFiltering: true,
  })

  const pageCount = table.getPageCount()
  useEffect(() => {
    ensurePageInRange(pageCount)
  }, [pageCount, ensurePageInRange])

  return (
    <>
      <DataTablePage
        table={table}
        columns={columns}
        isLoading={isLoading}
        isFetching={isFetching}
        emptyTitle={t('No conversation history found')}
        emptyDescription={t(
          'No conversation records are available. Set CONVERSATION_LOG_ENABLED=true to start collecting successful conversations.'
        )}
        skeletonKeyPrefix='conversation-history-skeleton'
        toolbarProps={{
          searchPlaceholder: t('Search user or assistant text...'),
          hasAdditionalFilters,
          onReset: resetFilters,
          additionalSearch: (
            <div className='grid w-full gap-2 sm:grid-cols-2 lg:grid-cols-6'>
              <Input
                placeholder={t('Username')}
                value={search.username || ''}
                onChange={(event) =>
                  setFilter({ username: event.target.value || undefined })
                }
              />
              <Input
                placeholder={t('Token')}
                value={search.tokenName || ''}
                onChange={(event) =>
                  setFilter({ tokenName: event.target.value || undefined })
                }
              />
              <Input
                placeholder={t('Model')}
                value={search.modelName || ''}
                onChange={(event) =>
                  setFilter({ modelName: event.target.value || undefined })
                }
              />
              <Input
                placeholder={t('Group')}
                value={search.group || ''}
                onChange={(event) =>
                  setFilter({ group: event.target.value || undefined })
                }
              />
              <Input
                placeholder={t('Request ID')}
                value={search.requestId || ''}
                onChange={(event) =>
                  setFilter({ requestId: event.target.value || undefined })
                }
              />
              <Input
                placeholder={t('Session ID')}
                value={search.sessionId || ''}
                onChange={(event) =>
                  setFilter({ sessionId: event.target.value || undefined })
                }
              />
              <Input
                type='datetime-local'
                value={
                  search.startTime
                    ? formatTimestampForInput(search.startTime)
                    : ''
                }
                onChange={(event) =>
                  setFilter({
                    startTime: event.target.value
                      ? parseTimestampFromInput(event.target.value)
                      : undefined,
                  })
                }
              />
              <Input
                type='datetime-local'
                value={
                  search.endTime ? formatTimestampForInput(search.endTime) : ''
                }
                onChange={(event) =>
                  setFilter({
                    endTime: event.target.value
                      ? parseTimestampFromInput(event.target.value)
                      : undefined,
                  })
                }
              />
              <select
                className={cn(
                  'border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring h-9 rounded-md border px-3 py-1 text-sm shadow-xs transition-colors focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50',
                  'sm:col-span-2 lg:col-span-1'
                )}
                value={search.exported || ''}
                onChange={(event) =>
                  setFilter({ exported: event.target.value || undefined })
                }
              >
                <option value=''>{t('All export states')}</option>
                <option value='false'>{t('Unexported')}</option>
                <option value='true'>{t('Exported')}</option>
              </select>
            </div>
          ),
        }}
      />
      <ConversationDetailSheet
        open={detailOpen}
        onOpenChange={setDetailOpen}
        session={selectedSession}
        filters={detailFilters}
      />
    </>
  )
}
