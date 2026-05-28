package model

import (
	"strings"
	"testing"
)

func TestBuildCleanConversationRecordUsesLastUserMessage(t *testing.T) {
	log := &ConversationLog{
		Id:           11,
		ModelName:    "deepseek-v4-pro",
		RequestBody:  `{"messages":[{"role":"user","content":[{"type":"text","text":"<system-reminder>internal</system-reminder>"}]},{"role":"assistant","content":"hi"},{"role":"user","content":[{"type":"text","text":"嗨嗨嗨"}]}],"metadata":{"user_id":"{\"session_id\":\"session-1\"}"}}`,
		ResponseBody: "嗨。有什么要做的？",
		Status:       ConversationLogStatusOK,
		CreatedAt:    1778637993,
	}

	record, reason := BuildCleanConversationRecord(log, CleanConversationOptions{SkipInternalCalls: true})
	if reason != "" {
		t.Fatalf("expected clean record, got skip reason %q", reason)
	}
	if record.User != "嗨嗨嗨" {
		t.Fatalf("expected last user message, got %q", record.User)
	}
	if record.Assistant != "嗨。有什么要做的？" {
		t.Fatalf("unexpected assistant text %q", record.Assistant)
	}
	if record.SessionId != "session-1" {
		t.Fatalf("expected session id from nested metadata, got %q", record.SessionId)
	}
}

func TestBuildCleanConversationRecordSkipsInternalCalls(t *testing.T) {
	cases := []struct {
		name string
		log  *ConversationLog
		want string
	}{
		{
			name: "memory agent user message",
			log: &ConversationLog{
				Id:           13,
				RequestBody:  `{"messages":[{"role":"user","content":[{"type":"text","text":"Hello memory agent, you are continuing to observe the primary Claude session."}]}]}`,
				ResponseBody: "some response",
				Status:       ConversationLogStatusOK,
			},
			want: "internal_user_message",
		},
		{
			name: "observation response",
			log: &ConversationLog{
				Id:           14,
				RequestBody:  `{"messages":[{"role":"user","content":"嗨嗨嗨"}]}`,
				ResponseBody: "<observation><type>discovery</type></observation>",
				Status:       ConversationLogStatusOK,
			},
			want: "internal_response",
		},
		{
			name: "empty response",
			log: &ConversationLog{
				Id:           15,
				RequestBody:  `{"messages":[{"role":"user","content":"嗨嗨嗨"}]}`,
				ResponseBody: "",
				Status:       ConversationLogStatusOK,
			},
			want: "empty_response",
		},
		{
			name: "client gone by default",
			log: &ConversationLog{
				Id:           16,
				RequestBody:  `{"messages":[{"role":"user","content":"嗨嗨嗨"}]}`,
				ResponseBody: "嗨。",
				Status:       "client_gone",
			},
			want: "non_ok_status",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, reason := BuildCleanConversationRecord(tc.log, CleanConversationOptions{SkipInternalCalls: true})
			if reason != tc.want {
				t.Fatalf("expected skip reason %q, got %q", tc.want, reason)
			}
		})
	}
}

func TestCleanConversationLogsTracksSeenIds(t *testing.T) {
	result := CleanConversationLogs([]*ConversationLog{
		{
			Id:           1,
			RequestBody:  `{"messages":[{"role":"user","content":"hello"}]}`,
			ResponseBody: "hi",
			Status:       ConversationLogStatusOK,
		},
		{
			Id:           2,
			RequestBody:  `{"messages":[{"role":"user","content":"hello"}]}`,
			ResponseBody: "",
			Status:       ConversationLogStatusOK,
		},
	}, CleanConversationOptions{SkipInternalCalls: true})

	if len(result.Records) != 1 {
		t.Fatalf("expected 1 clean record, got %d", len(result.Records))
	}
	if len(result.Skips) != 1 {
		t.Fatalf("expected 1 skipped record, got %d", len(result.Skips))
	}
	if len(result.SeenIds) != 2 || result.SeenIds[0] != 1 || result.SeenIds[1] != 2 {
		t.Fatalf("unexpected seen ids: %#v", result.SeenIds)
	}
}

func TestCleanConversationRecordToQAEImportRecord(t *testing.T) {
	record := &CleanConversationRecord{
		SourceLogId:  11,
		RequestId:    "req-1",
		Username:     "alice",
		ModelName:    "deepseek-v4-pro",
		SessionId:    "session-1",
		User:         "怎么部署 newapi？",
		Assistant:    "建议 systemd + mysql。",
		RequestPath:  "/v1/chat/completions",
		CreatedAt:    1778637993,
		ChannelName:  "main",
		TokenName:    "team-token",
		PromptTokens: 10,
	}

	qae := record.ToQAEImportRecord("tokenhub")
	if qae.Id != "tokenhub:11" {
		t.Fatalf("unexpected qae id %q", qae.Id)
	}
	if qae.Question != record.User {
		t.Fatalf("unexpected question %q", qae.Question)
	}
	if qae.Answer != record.Assistant {
		t.Fatalf("unexpected answer %q", qae.Answer)
	}
	if qae.Source != "tokenhub/deepseek-v4-pro" {
		t.Fatalf("unexpected source %q", qae.Source)
	}
	if qae.Model != "deepseek-v4-pro" {
		t.Fatalf("unexpected model %q", qae.Model)
	}
	for _, want := range []string{"来源日志ID：11", "请求ID：req-1", "用户：alice", "会话ID：session-1"} {
		if !strings.Contains(qae.Context, want) {
			t.Fatalf("expected context to contain %q, got %q", want, qae.Context)
		}
	}
}

func TestCleanConversationRecordToQAEImportRecordIncludesMultiturnContext(t *testing.T) {
	log := &ConversationLog{
		Id:           21,
		RequestId:    "req-multi",
		ModelName:    "gpt-4o",
		RequestBody:  `{"messages":[{"role":"system","content":"回答要简洁"},{"role":"user","content":"第一轮问题"},{"role":"assistant","content":"第一轮回答"},{"role":"user","content":"基于上面继续分析"}],"metadata":{"conversation_id":"conv-qa"}}`,
		ResponseBody: "第二轮回答",
		Status:       ConversationLogStatusOK,
		CreatedAt:    1778638000,
	}

	record, reason := BuildCleanConversationRecord(log, CleanConversationOptions{SkipInternalCalls: true})
	if reason != "" {
		t.Fatalf("expected clean record, got skip reason %q", reason)
	}
	qae := record.ToQAEImportRecord("tokenhub")

	if qae.Question != "基于上面继续分析" {
		t.Fatalf("unexpected question %q", qae.Question)
	}
	if qae.Answer != "第二轮回答" {
		t.Fatalf("unexpected answer %q", qae.Answer)
	}
	for _, want := range []string{
		"会话ID：conv-qa",
		"对话上下文：",
		"system: 回答要简洁",
		"user: 第一轮问题",
		"assistant: 第一轮回答",
	} {
		if !strings.Contains(qae.Context, want) {
			t.Fatalf("expected qae context to contain %q, got %q", want, qae.Context)
		}
	}
	if strings.Contains(qae.Context, "基于上面继续分析") {
		t.Fatalf("current question should not be duplicated in context: %q", qae.Context)
	}

	messages, ok := qae.Metadata["messages"].([]CleanConversationMessage)
	if !ok {
		t.Fatalf("expected metadata messages, got %#v", qae.Metadata["messages"])
	}
	if len(messages) != 5 {
		t.Fatalf("expected request messages plus current assistant, got %#v", messages)
	}
	if messages[4].Role != "assistant" || messages[4].Content != "第二轮回答" {
		t.Fatalf("expected current assistant appended to metadata messages, got %#v", messages[4])
	}
}

func TestBuildConversationLogSearchFieldsExtractsSessionAndText(t *testing.T) {
	fields := buildConversationLogSearchFields(
		`{"messages":[{"role":"user","content":"old question"},{"role":"assistant","content":"old answer"},{"role":"user","content":[{"type":"text","text":"分析这张图"},{"type":"image_url","image_url":{"url":"data:image/png;base64,abc"}},{"type":"file","file":{"filename":"report.pdf"}},{"type":"tool_result","content":"tool output"}]}],"metadata":{"conversationId":"conv-1"}}`,
		"图里是一张架构图。",
	)

	if fields.SessionId != "conv-1" {
		t.Fatalf("expected session id conv-1, got %q", fields.SessionId)
	}
	if fields.UserText != "分析这张图\n[image]\n[file]\n[tool]" {
		t.Fatalf("unexpected user text %q", fields.UserText)
	}
	if fields.AssistantText != "图里是一张架构图。" {
		t.Fatalf("unexpected assistant text %q", fields.AssistantText)
	}
}

func TestBuildConversationLogSearchFieldsExtractsResponsesInput(t *testing.T) {
	fields := buildConversationLogSearchFields(
		`{"model":"gpt-4.1","input":[{"role":"system","content":"be concise"},{"role":"user","content":[{"type":"input_text","text":"解释这段代码"},{"type":"input_image","image_url":"data:image/png;base64,abc"}]}],"metadata":{"session_id":"responses-session"}}`,
		"这段代码会解析输入。",
	)

	if fields.SessionId != "responses-session" {
		t.Fatalf("expected responses session id, got %q", fields.SessionId)
	}
	if fields.UserText != "解释这段代码\n[image]" {
		t.Fatalf("unexpected responses user text %q", fields.UserText)
	}
	if fields.AssistantText != "这段代码会解析输入。" {
		t.Fatalf("unexpected responses assistant text %q", fields.AssistantText)
	}
}

func TestBuildConversationLogSearchFieldsExtractsAssistantTextFromJSONResponse(t *testing.T) {
	fields := buildConversationLogSearchFields(
		`{"messages":[{"role":"user","content":"你好"}]}`,
		`{"choices":[{"message":{"role":"assistant","content":"你好，有什么可以帮你？"}}]}`,
	)

	if fields.UserText != "你好" {
		t.Fatalf("unexpected user text %q", fields.UserText)
	}
	if fields.AssistantText != "你好，有什么可以帮你？" {
		t.Fatalf("unexpected assistant text %q", fields.AssistantText)
	}
}

func TestBuildConversationLogSearchFieldsTruncatesSearchText(t *testing.T) {
	longText := strings.Repeat("a", conversationLogSearchTextMaxBytes+100)
	fields := buildConversationLogSearchFields(
		`{"messages":[{"role":"user","content":"`+longText+`"}]}`,
		longText,
	)

	if len(fields.UserText) != conversationLogSearchTextMaxBytes {
		t.Fatalf("expected user text length %d, got %d", conversationLogSearchTextMaxBytes, len(fields.UserText))
	}
	if len(fields.AssistantText) != conversationLogSearchTextMaxBytes {
		t.Fatalf("expected assistant text length %d, got %d", conversationLogSearchTextMaxBytes, len(fields.AssistantText))
	}
}
