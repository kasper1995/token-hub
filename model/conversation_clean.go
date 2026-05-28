package model

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type CleanConversationOptions struct {
	IncludeNonOK      bool
	IncludeReasoning  bool
	SkipInternalCalls bool
}

type CleanConversationRecord struct {
	SourceLogId      int                        `json:"source_log_id"`
	RequestId        string                     `json:"request_id,omitempty"`
	UserId           int                        `json:"user_id,omitempty"`
	Username         string                     `json:"username,omitempty"`
	TokenId          int                        `json:"token_id,omitempty"`
	TokenName        string                     `json:"token_name,omitempty"`
	ChannelId        int                        `json:"channel_id,omitempty"`
	ChannelName      string                     `json:"channel_name,omitempty"`
	ModelName        string                     `json:"model_name,omitempty"`
	SessionId        string                     `json:"session_id,omitempty"`
	User             string                     `json:"user"`
	Assistant        string                     `json:"assistant"`
	Reasoning        string                     `json:"reasoning,omitempty"`
	Messages         []CleanConversationMessage `json:"messages,omitempty"`
	ContextMessages  []CleanConversationMessage `json:"context_messages,omitempty"`
	PromptTokens     int                        `json:"prompt_tokens,omitempty"`
	CompletionTokens int                        `json:"completion_tokens,omitempty"`
	Status           string                     `json:"status,omitempty"`
	RequestPath      string                     `json:"request_path,omitempty"`
	CreatedAt        int64                      `json:"created_at"`
}

type QAEImportRecord struct {
	Id         string         `json:"id"`
	Question   string         `json:"question"`
	Answer     string         `json:"answer"`
	Context    string         `json:"context,omitempty"`
	Source     string         `json:"source,omitempty"`
	Model      string         `json:"model,omitempty"`
	Difficulty string         `json:"difficulty,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type CleanConversationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CleanConversationSkip struct {
	LogId  int
	Reason string
}

type CleanConversationResult struct {
	Records []*CleanConversationRecord
	Skips   []CleanConversationSkip
	SeenIds []int
}

type conversationRequestForClean struct {
	Model    string                     `json:"model"`
	Messages []conversationMessageClean `json:"messages"`
	Input    json.RawMessage            `json:"input"`
	System   json.RawMessage            `json:"system"`
	Metadata map[string]any             `json:"metadata"`
}

type conversationMessageClean struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func CleanConversationLogs(logs []*ConversationLog, options CleanConversationOptions) CleanConversationResult {
	result := CleanConversationResult{
		Records: make([]*CleanConversationRecord, 0, len(logs)),
		Skips:   make([]CleanConversationSkip, 0),
		SeenIds: make([]int, 0, len(logs)),
	}
	for _, log := range logs {
		if log == nil {
			continue
		}
		result.SeenIds = append(result.SeenIds, log.Id)
		record, reason := BuildCleanConversationRecord(log, options)
		if reason != "" {
			result.Skips = append(result.Skips, CleanConversationSkip{LogId: log.Id, Reason: reason})
			continue
		}
		result.Records = append(result.Records, record)
	}
	return result
}

func BuildCleanConversationRecord(log *ConversationLog, options CleanConversationOptions) (*CleanConversationRecord, string) {
	if log == nil {
		return nil, "empty_log"
	}
	if !options.IncludeNonOK && log.Status != "" && log.Status != ConversationLogStatusOK {
		return nil, "non_ok_status"
	}

	assistant := strings.TrimSpace(log.ResponseBody)
	if assistant == "" {
		return nil, "empty_response"
	}
	if options.SkipInternalCalls && looksLikeInternalAssistantResponse(assistant) {
		return nil, "internal_response"
	}

	request, err := parseConversationCleanRequest(log.RequestBody)
	if err != nil {
		return nil, "invalid_request_json"
	}
	messages := normalizedConversationMessages(request.Messages)
	userText, userMessageIndex := lastUserCleanMessageTextWithIndex(messages)
	userText = strings.TrimSpace(userText)
	if userText == "" {
		userText = strings.TrimSpace(lastUserInputText(request.Input))
		if len(messages) == 0 {
			messages = normalizedInputMessages(request.Input)
			_, userMessageIndex = lastUserCleanMessageTextWithIndex(messages)
		}
	}
	if userText == "" {
		return nil, "empty_user_message"
	}
	if options.SkipInternalCalls && looksLikeInternalUserMessage(userText) {
		return nil, "internal_user_message"
	}
	if options.SkipInternalCalls && looksLikeInternalSystemPrompt(textFromRawJSON(request.System)) {
		return nil, "internal_system_prompt"
	}

	modelName := log.ModelName
	if modelName == "" {
		modelName = request.Model
	}

	record := &CleanConversationRecord{
		SourceLogId:      log.Id,
		RequestId:        log.RequestId,
		UserId:           log.UserId,
		Username:         log.Username,
		TokenId:          log.TokenId,
		TokenName:        log.TokenName,
		ChannelId:        log.ChannelId,
		ChannelName:      log.ChannelName,
		ModelName:        modelName,
		SessionId:        extractSessionID(request.Metadata),
		User:             userText,
		Assistant:        assistant,
		Messages:         messages,
		ContextMessages:  contextMessagesBeforeIndex(messages, userMessageIndex),
		PromptTokens:     log.PromptTokens,
		CompletionTokens: log.CompletionTokens,
		Status:           log.Status,
		RequestPath:      log.RequestPath,
		CreatedAt:        log.CreatedAt,
	}
	if options.IncludeReasoning {
		record.Reasoning = strings.TrimSpace(log.ResponseReasoningBody)
	}
	return record, ""
}

func (record *CleanConversationRecord) ToQAEImportRecord(sourcePrefix string) QAEImportRecord {
	if sourcePrefix == "" {
		sourcePrefix = "tokenhub"
	}
	source := sourcePrefix
	if record.ModelName != "" {
		source += "/" + record.ModelName
	}

	contextParts := []string{
		"来源日志ID：" + intToString(record.SourceLogId),
	}
	if record.RequestId != "" {
		contextParts = append(contextParts, "请求ID："+record.RequestId)
	}
	if record.Username != "" {
		contextParts = append(contextParts, "用户："+record.Username)
	}
	if record.SessionId != "" {
		contextParts = append(contextParts, "会话ID："+record.SessionId)
	}
	if record.TokenName != "" {
		contextParts = append(contextParts, "令牌："+record.TokenName)
	}
	if record.ChannelName != "" {
		contextParts = append(contextParts, "渠道："+record.ChannelName)
	}
	if record.RequestPath != "" {
		contextParts = append(contextParts, "路径："+record.RequestPath)
	}
	if record.CreatedAt != 0 {
		contextParts = append(contextParts, "时间戳："+int64ToString(record.CreatedAt))
	}
	context := strings.Join(contextParts, "；")
	if transcript := conversationTranscript(record.ContextMessages); transcript != "" {
		context += "\n\n对话上下文：\n" + transcript
	}

	return QAEImportRecord{
		Id:       sourcePrefix + ":" + intToString(record.SourceLogId),
		Question: record.User,
		Answer:   record.Assistant,
		Context:  context,
		Source:   source,
		Model:    record.ModelName,
		Metadata: record.qaeMetadata(),
	}
}

func (record *CleanConversationRecord) qaeMetadata() map[string]any {
	if record == nil {
		return nil
	}
	metadata := map[string]any{
		"source_log_id":      record.SourceLogId,
		"created_at":         record.CreatedAt,
		"prompt_tokens":      record.PromptTokens,
		"completion_tokens":  record.CompletionTokens,
		"context_turn_count": len(record.ContextMessages),
	}
	addStringMetadata(metadata, "request_id", record.RequestId)
	addStringMetadata(metadata, "username", record.Username)
	addStringMetadata(metadata, "token_name", record.TokenName)
	addStringMetadata(metadata, "channel_name", record.ChannelName)
	addStringMetadata(metadata, "model_name", record.ModelName)
	addStringMetadata(metadata, "session_id", record.SessionId)
	addStringMetadata(metadata, "status", record.Status)
	addStringMetadata(metadata, "request_path", record.RequestPath)
	if record.UserId != 0 {
		metadata["user_id"] = record.UserId
	}
	if record.TokenId != 0 {
		metadata["token_id"] = record.TokenId
	}
	if record.ChannelId != 0 {
		metadata["channel_id"] = record.ChannelId
	}
	if record.Reasoning != "" {
		metadata["reasoning"] = record.Reasoning
	}
	if len(record.Messages) > 0 {
		messages := make([]CleanConversationMessage, 0, len(record.Messages)+1)
		messages = append(messages, record.Messages...)
		if record.Assistant != "" {
			messages = append(messages, CleanConversationMessage{Role: "assistant", Content: record.Assistant})
		}
		metadata["messages"] = messages
	}
	if len(record.ContextMessages) > 0 {
		metadata["context_messages"] = record.ContextMessages
	}
	return metadata
}

func addStringMetadata(metadata map[string]any, key string, value string) {
	if value != "" {
		metadata[key] = value
	}
}

func CleanConversationRecordsToQAE(records []*CleanConversationRecord, sourcePrefix string) []QAEImportRecord {
	qaeRecords := make([]QAEImportRecord, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		qaeRecords = append(qaeRecords, record.ToQAEImportRecord(sourcePrefix))
	}
	return qaeRecords
}

func parseConversationCleanRequest(body string) (*conversationRequestForClean, error) {
	var request conversationRequestForClean
	if err := common.Unmarshal([]byte(body), &request); err != nil {
		return nil, err
	}
	return &request, nil
}

func lastUserMessageText(messages []conversationMessageClean) string {
	text, _ := lastUserMessageTextWithIndex(messages)
	return text
}

func lastUserMessageTextWithIndex(messages []conversationMessageClean) (string, int) {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "user") {
			return textFromRawJSON(messages[i].Content), i
		}
	}
	return "", -1
}

func lastUserCleanMessageTextWithIndex(messages []CleanConversationMessage) (string, int) {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "user") {
			return messages[i].Content, i
		}
	}
	return "", -1
}

func normalizedConversationMessages(messages []conversationMessageClean) []CleanConversationMessage {
	cleanMessages := make([]CleanConversationMessage, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(message.Role)
		content := strings.TrimSpace(textFromRawJSON(message.Content))
		if role == "" || content == "" {
			continue
		}
		cleanMessages = append(cleanMessages, CleanConversationMessage{
			Role:    strings.ToLower(role),
			Content: content,
		})
	}
	return cleanMessages
}

func normalizedInputMessages(raw json.RawMessage) []CleanConversationMessage {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return normalizedInputMessagesFromValue(value)
}

func normalizedInputMessagesFromValue(value any) []CleanConversationMessage {
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		return []CleanConversationMessage{{Role: "user", Content: text}}
	case []any:
		messages := make([]CleanConversationMessage, 0, len(typed))
		for _, item := range typed {
			messages = append(messages, normalizedInputMessagesFromValue(item)...)
		}
		return messages
	case map[string]any:
		role := strings.TrimSpace(textFromValue(typed["role"]))
		if role == "" {
			role = "user"
		}
		for _, key := range []string{"content", "input", "text"} {
			if raw, ok := typed[key]; ok {
				if text := strings.TrimSpace(textFromValue(raw)); text != "" {
					return []CleanConversationMessage{{
						Role:    strings.ToLower(role),
						Content: text,
					}}
				}
			}
		}
	}
	return nil
}

func contextMessagesBeforeIndex(messages []CleanConversationMessage, index int) []CleanConversationMessage {
	if index <= 0 || len(messages) == 0 {
		return nil
	}
	context := make([]CleanConversationMessage, 0, index)
	for _, message := range messages[:index] {
		if strings.EqualFold(message.Role, "system") ||
			strings.EqualFold(message.Role, "user") ||
			strings.EqualFold(message.Role, "assistant") {
			context = append(context, message)
		}
	}
	return context
}

func conversationTranscript(messages []CleanConversationMessage) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(message.Role)
		content := strings.TrimSpace(message.Content)
		if role == "" || content == "" {
			continue
		}
		parts = append(parts, role+": "+content)
	}
	return strings.Join(parts, "\n")
}

func lastUserInputText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return lastUserTextFromInputValue(value)
}

func lastUserTextFromInputValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		for i := len(typed) - 1; i >= 0; i-- {
			if text := strings.TrimSpace(lastUserTextFromInputValue(typed[i])); text != "" {
				return text
			}
		}
	case map[string]any:
		if role := strings.TrimSpace(textFromValue(typed["role"])); role != "" && !strings.EqualFold(role, "user") {
			return ""
		}
		for _, key := range []string{"content", "input", "text"} {
			if raw, ok := typed[key]; ok {
				if text := strings.TrimSpace(textFromValue(raw)); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func textFromRawJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(textFromValue(value))
}

func textFromValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(textFromValue(item)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		for _, key := range []string{"text", "content", "input_text", "output_text"} {
			if raw, ok := typed[key]; ok {
				if text := strings.TrimSpace(textFromValue(raw)); text != "" {
					return text
				}
			}
		}
		return ""
	default:
		return ""
	}
}

func looksLikeInternalUserMessage(text string) bool {
	lowered := strings.ToLower(strings.TrimSpace(text))
	if lowered == "" {
		return true
	}
	patterns := []string{
		"hello memory agent",
		"memory processing continued",
		"you are a claude-mem",
		"<observed_from_primary_session>",
		"<system-reminder>",
		"sessionstart hook additional context",
		"the following deferred tools are now available",
	}
	return containsAny(lowered, patterns)
}

func looksLikeInternalSystemPrompt(text string) bool {
	lowered := strings.ToLower(strings.TrimSpace(text))
	patterns := []string{
		"generate a concise, sentence-case title",
		"return json with a single \"title\" field",
		"you are a claude-mem",
		"specialized observer tool",
	}
	return containsAny(lowered, patterns)
}

func looksLikeInternalAssistantResponse(text string) bool {
	lowered := strings.ToLower(strings.TrimSpace(text))
	patterns := []string{
		"<observation>",
		"</observation>",
		"\"title\"",
	}
	if strings.HasPrefix(lowered, "{") && strings.Contains(lowered, "\"title\"") && !strings.Contains(lowered, "\"assistant\"") {
		return true
	}
	return containsAny(lowered, patterns[:2])
}

func containsAny(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func intToString(value int) string {
	return strconv.Itoa(value)
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

func extractSessionID(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	for _, key := range []string{"session_id", "sessionId", "conversation_id", "conversationId"} {
		if value, ok := metadata[key]; ok {
			if text := strings.TrimSpace(textFromValue(value)); text != "" {
				return text
			}
		}
	}
	if rawUserID, ok := metadata["user_id"]; ok {
		rawText := strings.TrimSpace(textFromValue(rawUserID))
		if rawText == "" {
			return ""
		}
		var nested map[string]any
		if err := common.Unmarshal([]byte(rawText), &nested); err == nil {
			return extractSessionID(nested)
		}
	}
	return ""
}
