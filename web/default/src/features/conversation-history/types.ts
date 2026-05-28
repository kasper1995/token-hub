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

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data: T
}

export interface PageResponse<T> {
  items: T[]
  total: number
  page?: number
  page_size?: number
}

export interface ConversationSessionSummary {
  session_key: string
  session_id: string
  user_id: number
  username: string
  log_count: number
  first_log_id: number
  last_log_id: number
  first_created_at: number
  last_created_at: number
  models: string[]
  token_names: string[]
  groups: string[]
  latest_user_text: string
  latest_assistant_text: string
  prompt_tokens: number
  completion_tokens: number
  status: string
}

export interface ConversationLogRecord {
  id: number
  request_id: string
  user_id: number
  username: string
  token_id: number
  token_name: string
  channel_id: number
  channel_name: string
  model_name: string
  request_path: string
  session_id: string
  user_text: string
  assistant_text: string
  request_body: string
  response_body: string
  response_reasoning_body: string
  request_body_truncated: boolean
  response_body_truncated: boolean
  response_reasoning_truncated: boolean
  prompt_tokens: number
  completion_tokens: number
  is_stream: boolean
  status: string
  error_message: string
  group: string
  created_at: number
  exported_at: number
  final_request_relay_format: string
  original_request_relay_format: string
}

export interface ConversationSessionDetail {
  summary: ConversationSessionSummary
  logs: ConversationLogRecord[]
  truncated: boolean
}

export interface GetConversationSessionsParams {
  p?: number
  page_size?: number
  limit?: number
  start_timestamp?: number
  end_timestamp?: number
  username?: string
  token_name?: string
  model_name?: string
  group?: string
  request_id?: string
  session_id?: string
  content?: string
  exported?: boolean
}

export interface GetConversationSessionDetailParams extends Omit<
  GetConversationSessionsParams,
  'p' | 'page_size'
> {
  session_key: string
  limit?: number
}
