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

import { api } from '@/lib/api'
import type {
  ApiResponse,
  ConversationSessionDetail,
  ConversationSessionSummary,
  GetConversationSessionDetailParams,
  GetConversationSessionsParams,
  PageResponse,
} from './types'

export async function getConversationSessions(
  params: GetConversationSessionsParams
): Promise<ApiResponse<PageResponse<ConversationSessionSummary>>> {
  const res = await api.get('/api/conversation_log/sessions', { params })
  return res.data
}

export async function getConversationSessionDetail(
  params: GetConversationSessionDetailParams
): Promise<ApiResponse<ConversationSessionDetail>> {
  const res = await api.get('/api/conversation_log/sessions/detail', { params })
  return res.data
}

export async function exportConversationQAEJson(
  params: Omit<GetConversationSessionsParams, 'p' | 'page_size'>
): Promise<Blob> {
  const res = await api.get('/api/conversation_log/export', {
    params: {
      ...params,
      format: 'qae-json',
      source_prefix: 'tokenhub',
      limit: params.limit ?? 5000,
    },
    responseType: 'blob',
  })
  return res.data
}
