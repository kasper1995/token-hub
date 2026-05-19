package model

import (
	"fmt"
	"testing"
)

func resetConversationLogsForSessionTest(t *testing.T) {
	t.Helper()
	if err := LOG_DB.AutoMigrate(&ConversationLog{}); err != nil {
		t.Fatalf("failed to migrate conversation logs: %v", err)
	}
	if err := LOG_DB.Exec("DELETE FROM conversation_logs").Error; err != nil {
		t.Fatalf("failed to reset conversation logs: %v", err)
	}
	t.Cleanup(func() {
		LOG_DB.Exec("DELETE FROM conversation_logs")
	})
}

func insertConversationLogForSessionTest(t *testing.T, log *ConversationLog) {
	t.Helper()
	if err := LOG_DB.Create(log).Error; err != nil {
		t.Fatalf("failed to insert conversation log: %v", err)
	}
}

func TestGetConversationSessionsGroupsByUserAndSession(t *testing.T) {
	resetConversationLogsForSessionTest(t)
	insertConversationLogForSessionTest(t, &ConversationLog{UserId: 1, Username: "alice", TokenName: "team-token", Group: "default", SessionId: "same", ModelName: "gpt-4o", UserText: "first", AssistantText: "first answer", CreatedAt: 100, PromptTokens: 1, CompletionTokens: 2, Status: ConversationLogStatusOK})
	insertConversationLogForSessionTest(t, &ConversationLog{UserId: 1, Username: "alice", TokenName: "batch-token", Group: "vip", SessionId: "same", ModelName: "claude-sonnet", UserText: "latest", AssistantText: "latest answer", CreatedAt: 200, PromptTokens: 3, CompletionTokens: 4, Status: ConversationLogStatusOK})
	insertConversationLogForSessionTest(t, &ConversationLog{UserId: 2, Username: "bob", TokenName: "bob-token", Group: "default", SessionId: "same", ModelName: "deepseek", UserText: "bob question", AssistantText: "bob answer", CreatedAt: 250, PromptTokens: 5, CompletionTokens: 6, Status: ConversationLogStatusOK})
	insertConversationLogForSessionTest(t, &ConversationLog{UserId: 1, Username: "alice", TokenName: "single-token", Group: "default", ModelName: "gpt-4o-mini", UserText: "single", AssistantText: "single answer", CreatedAt: 300, PromptTokens: 7, CompletionTokens: 8, Status: ConversationLogStatusOK})

	sessions, total, err := GetConversationSessions(ConversationSessionQuery{}, 0, 10)
	if err != nil {
		t.Fatalf("GetConversationSessions returned error: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 sessions, got %d", total)
	}
	if len(sessions) != 3 {
		t.Fatalf("expected 3 returned sessions, got %d", len(sessions))
	}
	if sessions[0].SessionId != "" || sessions[0].LogCount != 1 || sessions[0].LatestUserText != "single" {
		t.Fatalf("unexpected first session: %#v", sessions[0])
	}
	if sessions[1].UserId != 2 || sessions[1].SessionId != "same" || sessions[1].LogCount != 1 {
		t.Fatalf("unexpected second session: %#v", sessions[1])
	}
	if sessions[2].UserId != 1 || sessions[2].SessionId != "same" || sessions[2].LogCount != 2 {
		t.Fatalf("unexpected third session: %#v", sessions[2])
	}
	if len(sessions[2].Models) != 2 || sessions[2].Models[0] != "gpt-4o" || sessions[2].Models[1] != "claude-sonnet" {
		t.Fatalf("unexpected session models: %#v", sessions[2].Models)
	}
	if len(sessions[2].TokenNames) != 2 || sessions[2].TokenNames[0] != "team-token" || sessions[2].TokenNames[1] != "batch-token" {
		t.Fatalf("unexpected session token names: %#v", sessions[2].TokenNames)
	}
	if len(sessions[2].Groups) != 2 || sessions[2].Groups[0] != "default" || sessions[2].Groups[1] != "vip" {
		t.Fatalf("unexpected session groups: %#v", sessions[2].Groups)
	}
	if sessions[2].LatestUserText != "latest" || sessions[2].LatestAssistantText != "latest answer" {
		t.Fatalf("unexpected latest summary: %#v", sessions[2])
	}
}

func TestGetConversationSessionsFiltersRecordsBeforeGrouping(t *testing.T) {
	resetConversationLogsForSessionTest(t)
	insertConversationLogForSessionTest(t, &ConversationLog{UserId: 1, Username: "alice", TokenName: "token-a", Group: "default", SessionId: "s1", ModelName: "gpt-4o", UserText: "ordinary", AssistantText: "ordinary answer", CreatedAt: 100, Status: ConversationLogStatusOK})
	insertConversationLogForSessionTest(t, &ConversationLog{UserId: 1, Username: "alice", TokenName: "token-b", Group: "vip", SessionId: "s1", ModelName: "gpt-4o", UserText: "needle question", AssistantText: "needle answer", CreatedAt: 200, Status: ConversationLogStatusOK})
	insertConversationLogForSessionTest(t, &ConversationLog{UserId: 1, Username: "alice", TokenName: "token-a", Group: "default", SessionId: "s2", ModelName: "gpt-4o", UserText: "ordinary", AssistantText: "ordinary answer", CreatedAt: 300, Status: ConversationLogStatusOK})

	sessions, total, err := GetConversationSessions(ConversationSessionQuery{Content: "needle"}, 0, 10)
	if err != nil {
		t.Fatalf("GetConversationSessions returned error: %v", err)
	}
	if total != 1 || len(sessions) != 1 {
		t.Fatalf("expected one matching session, got total=%d len=%d", total, len(sessions))
	}
	if sessions[0].SessionId != "s1" || sessions[0].LogCount != 1 || sessions[0].LatestUserText != "needle question" {
		t.Fatalf("unexpected filtered session: %#v", sessions[0])
	}

	sessions, total, err = GetConversationSessions(ConversationSessionQuery{TokenName: "token-b", Group: "vip"}, 0, 10)
	if err != nil {
		t.Fatalf("GetConversationSessions returned error: %v", err)
	}
	if total != 1 || len(sessions) != 1 {
		t.Fatalf("expected one token/group matching session, got total=%d len=%d", total, len(sessions))
	}
	if sessions[0].SessionId != "s1" || sessions[0].LogCount != 1 || sessions[0].TokenNames[0] != "token-b" || sessions[0].Groups[0] != "vip" {
		t.Fatalf("unexpected token/group filtered session: %#v", sessions[0])
	}
}

func TestGetConversationSessionsHydratesMissingDisplayFieldsFromBodies(t *testing.T) {
	resetConversationLogsForSessionTest(t)
	insertConversationLogForSessionTest(t, &ConversationLog{
		UserId:       8,
		Username:     "dora",
		RequestBody:  `{"model":"gpt-4o","messages":[{"role":"user","content":"旧数据能显示吗？"}],"metadata":{"conversationId":"legacy-session"}}`,
		ResponseBody: `{"choices":[{"message":{"role":"assistant","content":"可以，读取时会补齐摘要。"}}]}`,
		CreatedAt:    400,
		Status:       ConversationLogStatusOK,
	})

	sessions, total, err := GetConversationSessions(ConversationSessionQuery{}, 0, 10)
	if err != nil {
		t.Fatalf("GetConversationSessions returned error: %v", err)
	}
	if total != 1 || len(sessions) != 1 {
		t.Fatalf("expected one session, got total=%d len=%d", total, len(sessions))
	}
	if sessions[0].SessionId != "legacy-session" {
		t.Fatalf("expected hydrated session id, got %#v", sessions[0])
	}
	if sessions[0].LatestUserText != "旧数据能显示吗？" {
		t.Fatalf("expected hydrated user text, got %#v", sessions[0])
	}
	if sessions[0].LatestAssistantText != "可以，读取时会补齐摘要。" {
		t.Fatalf("expected hydrated assistant text, got %#v", sessions[0])
	}
	if len(sessions[0].Models) != 1 || sessions[0].Models[0] != "gpt-4o" {
		t.Fatalf("expected hydrated model, got %#v", sessions[0].Models)
	}
}

func TestGetConversationSessionDetailReturnsRecentLogsWithLimit(t *testing.T) {
	resetConversationLogsForSessionTest(t)
	for i := 1; i <= 205; i++ {
		insertConversationLogForSessionTest(t, &ConversationLog{UserId: 7, Username: "carol", SessionId: "long", ModelName: "gpt-4o", UserText: fmt.Sprintf("question %03d", i), AssistantText: fmt.Sprintf("answer %03d", i), CreatedAt: int64(i), Status: ConversationLogStatusOK})
	}

	detail, err := GetConversationSessionDetail(ConversationSessionQuery{}, "user:7:session:long", 200)
	if err != nil {
		t.Fatalf("GetConversationSessionDetail returned error: %v", err)
	}
	if !detail.Truncated {
		t.Fatalf("expected truncated detail")
	}
	if len(detail.Logs) != 200 {
		t.Fatalf("expected 200 logs, got %d", len(detail.Logs))
	}
	if detail.Logs[0].UserText != "question 006" || detail.Logs[199].UserText != "question 205" {
		t.Fatalf("unexpected recent log range: first=%q last=%q", detail.Logs[0].UserText, detail.Logs[199].UserText)
	}
}

func TestGetConversationSessionDetailHydratesMissingDisplayFieldsFromBodies(t *testing.T) {
	resetConversationLogsForSessionTest(t)
	log := &ConversationLog{
		UserId:       9,
		Username:     "erin",
		RequestBody:  `{"messages":[{"role":"user","content":"详情里也要显示"}]}`,
		ResponseBody: "详情回复",
		CreatedAt:    500,
		Status:       ConversationLogStatusOK,
	}
	insertConversationLogForSessionTest(t, log)

	detail, err := GetConversationSessionDetail(ConversationSessionQuery{}, fmt.Sprintf("log:%d", log.Id), 200)
	if err != nil {
		t.Fatalf("GetConversationSessionDetail returned error: %v", err)
	}
	if len(detail.Logs) != 1 {
		t.Fatalf("expected one log, got %d", len(detail.Logs))
	}
	if detail.Logs[0].UserText != "详情里也要显示" {
		t.Fatalf("expected hydrated user text, got %#v", detail.Logs[0])
	}
	if detail.Logs[0].AssistantText != "详情回复" {
		t.Fatalf("expected hydrated assistant text, got %#v", detail.Logs[0])
	}
}
