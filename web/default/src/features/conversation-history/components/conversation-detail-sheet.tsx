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
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { formatTimestampToDate, formatTokens } from '@/lib/format'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { getConversationSessionDetail } from '../api'
import type {
  ConversationLogRecord,
  ConversationSessionSummary,
  GetConversationSessionDetailParams,
} from '../types'

interface ConversationDetailSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  session: ConversationSessionSummary | null
  filters: Omit<GetConversationSessionDetailParams, 'session_key' | 'limit'>
}

function RawBlock({ value }: { value: string }) {
  return (
    <pre className='bg-muted/30 max-h-72 overflow-auto rounded-md border p-3 text-xs leading-relaxed break-words whitespace-pre-wrap'>
      {value || '-'}
    </pre>
  )
}

function MessageBlock({
  title,
  value,
  tone,
}: {
  title: string
  value: string
  tone: 'user' | 'assistant'
}) {
  return (
    <section className='space-y-1.5'>
      <div className='text-muted-foreground text-xs font-medium'>{title}</div>
      <div
        className={
          tone === 'user'
            ? 'rounded-md border border-blue-200 bg-blue-50/70 px-3 py-2 text-sm leading-relaxed break-words whitespace-pre-wrap text-blue-950 dark:border-blue-900/60 dark:bg-blue-950/25 dark:text-blue-100'
            : 'bg-background rounded-md border px-3 py-2 text-sm leading-relaxed break-words whitespace-pre-wrap'
        }
      >
        {value || '-'}
      </div>
    </section>
  )
}

function RawDetails({ title, value }: { title: string; value: string }) {
  return (
    <details className='group bg-muted/10 rounded-md border'>
      <summary className='text-muted-foreground flex cursor-pointer list-none items-center justify-between px-3 py-2 text-xs font-medium'>
        <span>{title}</span>
        <span className='text-[11px] group-open:hidden'>+</span>
        <span className='hidden text-[11px] group-open:inline'>-</span>
      </summary>
      <div className='border-t p-3'>
        <RawBlock value={value} />
      </div>
    </details>
  )
}

function ConversationTurn({ log }: { log: ConversationLogRecord }) {
  const { t } = useTranslation()
  return (
    <article className='bg-card rounded-lg border p-4 shadow-xs'>
      <div className='text-muted-foreground mb-3 flex flex-wrap items-center gap-2 text-xs'>
        <Badge variant='outline'>#{log.id}</Badge>
        <span>{formatTimestampToDate(log.created_at)}</span>
        <span>
          {t('User')}: {log.username || `#${log.user_id}`}
        </span>
        {log.token_name && <Badge variant='outline'>{log.token_name}</Badge>}
        {log.group && <Badge variant='secondary'>{log.group}</Badge>}
        {log.model_name && <Badge variant='secondary'>{log.model_name}</Badge>}
        {log.channel_name && <span>{log.channel_name}</span>}
        <span>
          {formatTokens(log.prompt_tokens)} /{' '}
          {formatTokens(log.completion_tokens)}
        </span>
      </div>
      <div className='space-y-3'>
        <MessageBlock
          title={t('User message')}
          value={log.user_text}
          tone='user'
        />
        <MessageBlock
          title={t('Assistant response')}
          value={log.assistant_text}
          tone='assistant'
        />
        <RawDetails title={t('Raw request')} value={log.request_body} />
        <RawDetails title={t('Raw response')} value={log.response_body} />
        {log.response_reasoning_body && (
          <RawDetails
            title={t('Reasoning content')}
            value={log.response_reasoning_body}
          />
        )}
      </div>
    </article>
  )
}

export function ConversationDetailSheet(props: ConversationDetailSheetProps) {
  const { t } = useTranslation()
  const { open, onOpenChange, session, filters } = props

  const { data, isLoading } = useQuery({
    queryKey: ['conversation-session-detail', session?.session_key, filters],
    queryFn: () =>
      getConversationSessionDetail({
        ...filters,
        session_key: session!.session_key,
        limit: 200,
      }),
    enabled: open && Boolean(session),
  })

  const detail = data?.data
  const summary = detail?.summary || session

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='w-full gap-0 sm:max-w-4xl'>
        <SheetHeader className='border-b'>
          <SheetTitle className='pr-8'>
            {summary?.latest_user_text || t('Conversation Details')}
          </SheetTitle>
          <SheetDescription className='mt-2 flex flex-wrap gap-x-4 gap-y-1'>
            <span>
              {t('User')}: {summary?.username || `#${summary?.user_id || '-'}`}
            </span>
            <span>
              {t('Token')}: {summary?.token_names?.join(', ') || '-'}
            </span>
            <span>
              {t('Group')}: {summary?.groups?.join(', ') || '-'}
            </span>
            <span>
              {t('Session ID')}:{' '}
              {summary?.session_id || summary?.session_key || '-'}
            </span>
            <span>
              {t('Messages')}: {summary?.log_count || '-'}
            </span>
            <span>
              {t('Tokens')}: {formatTokens(summary?.prompt_tokens || 0)} /{' '}
              {formatTokens(summary?.completion_tokens || 0)}
            </span>
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className='min-h-0 flex-1'>
          <div className='space-y-4 p-4'>
            {detail?.truncated && (
              <div className='rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-300'>
                {t('Only the latest 200 records are shown for this session.')}
              </div>
            )}
            {isLoading && (
              <div className='text-muted-foreground text-sm'>
                {t('Loading...')}
              </div>
            )}
            {!isLoading && detail?.logs.length === 0 && (
              <div className='text-muted-foreground text-sm'>
                {t('No conversation records found.')}
              </div>
            )}
            {detail?.logs.map((log) => (
              <ConversationTurn key={log.id} log={log} />
            ))}
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  )
}
