# Conversation Log List/Detail Split Design

## Background

The conversation history list currently mixes two responsibilities:

- listing lightweight conversation sessions;
- hydrating missing display fields by reading raw `request_body` and `response_body`.

Production data shows why this is unsafe. The `conversation_logs` table has about 11.5k rows, but `request_body` is about 1.5GB in total. Rows missing display fields can trigger close to 1GB of raw body reads during a list request. On a small instance, this can exhaust memory even when the UI only asks for the first page.

The root rule is: list requests must not load raw bodies.

## Goals

- Make the conversation list memory-stable.
- Preserve the current list/detail user workflow.
- Keep raw request and response inspection available in the detail view.
- Keep dataset export behavior separate from UI list behavior.
- Support SQLite, MySQL, and PostgreSQL.

## Non-Goals

- Do not introduce a new `conversation_sessions` summary table in this change.
- Do not redesign the entire audit log system.
- Do not remove raw body storage yet.
- Do not make default list search scan raw bodies.

## Recommended Approach

Use a two-layer model:

1. The list endpoint reads only lightweight columns and truncated preview fields.
2. The detail endpoint loads raw bodies only after the user opens a session.

Historical rows that lack summary fields are repaired by a backfill command. After backfill, list queries should not need to parse raw bodies.

## Backend Design

### Write Path

When recording a conversation log, continue storing raw bodies according to the configured body limit. Also persist lightweight display fields:

- `session_id`
- `user_text`
- `assistant_text`
- `model_name`
- token/user/channel metadata
- token counts
- status and timestamps

These fields are already present on `conversation_logs`. The implementation should make them reliable enough that list queries do not need to parse raw JSON.

### List Endpoint

`GET /api/conversation_log/sessions` should:

- select only lightweight columns;
- group logs into sessions using persisted lightweight fields;
- return truncated previews from `user_text` and `assistant_text`;
- never select `request_body`, `response_body`, or `response_reasoning_body`;
- never call body hydration helpers on the list path.

If a historical row still lacks a preview, the list may return an empty preview or a generic preview marker. It must not load raw body content to fill it.

### Detail Endpoint

`GET /api/conversation_log/sessions/detail` should:

- load only the selected session;
- keep the existing detail limit, defaulting to 200 and capped at 200;
- return full raw fields for those detail rows;
- hydrate missing display fields from raw bodies only inside the selected detail set.

This keeps expensive raw-body parsing behind explicit user action and a bounded row limit.

### Export Endpoint

`GET /api/conversation_log/export` is a separate heavy operation. It may read raw bodies because export is explicitly about raw dataset extraction.

Export should keep its own `limit`, `after_id`, timestamp, and exported-state filters. It should not share the list endpoint's light-query restrictions.

### Backfill Command

Add or extend a local command to repair historical rows in batches:

- scan rows missing `session_id`, `user_text`, `assistant_text`, or `model_name`;
- read raw bodies in small batches;
- parse summary fields with existing parser logic;
- update only the missing lightweight columns;
- support `--limit`, `--after-id`, `--start-timestamp`, and `--end-timestamp`;
- avoid loading the full table into memory.

This command should be safe to run repeatedly. It should skip rows that cannot be parsed and report counts.

### Search Behavior

Default list filters should use lightweight indexed or bounded fields:

- username
- token name
- model name
- group
- request id
- session id
- exported state
- timestamp range
- `user_text` and `assistant_text`

Raw-body search should not be part of the default list query. If raw-body search remains necessary, expose it as an explicit heavy mode with a clear parameter and strict limits.

## Frontend Design

The current UI shape is mostly correct:

- list page shows sessions and short previews;
- detail sheet loads one session on demand;
- raw request and raw response stay inside detail.

Frontend changes should be minimal:

- treat missing preview fields as empty or `No preview`;
- avoid assuming list data contains raw bodies;
- keep the detail sheet responsible for raw fields.

## Data and Memory Rules

- List query memory should scale with selected lightweight rows, not raw body size.
- Detail query memory should scale with at most the detail limit.
- Backfill memory should scale with batch size.
- No request path should accidentally read all raw bodies for pagination.

## Testing

Add backend tests that prove:

- session list returns expected summaries without selecting raw body columns;
- missing previews on list do not trigger raw body hydration;
- detail still hydrates missing display fields for the selected session;
- export behavior remains unchanged;
- backfill fills missing lightweight columns in batches.

Use the existing model-level conversation session tests as the first seam.

## Rollout

1. Deploy the list/detail query change.
2. Run the backfill command on production in small batches.
3. Monitor memory and request time for `/api/conversation_log/sessions`.
4. Lower `CONVERSATION_LOG_MAX_BODY_BYTES` if production still writes MB-scale request bodies.
5. Keep export paths separate and documented as heavy operations.

## Decision

Proceed with backfill plus a strict lightweight list path. Do not create a new session summary table for this fix. A summary table can be revisited later if the dataset grows beyond what grouped lightweight log rows can handle.
