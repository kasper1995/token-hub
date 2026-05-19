package model

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ConversationLogStatusOK = "ok"
const conversationLogSearchTextMaxBytes = 8 * 1024

type conversationLogSearchFields struct {
	SessionId     string
	UserText      string
	AssistantText string
}

type ConversationLog struct {
	Id                         int    `json:"id" gorm:"index:idx_conversation_created_id,priority:2"`
	RequestId                  string `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	UserId                     int    `json:"user_id" gorm:"index"`
	Username                   string `json:"username" gorm:"index;default:''"`
	TokenId                    int    `json:"token_id" gorm:"index;default:0"`
	TokenName                  string `json:"token_name" gorm:"index;default:''"`
	ChannelId                  int    `json:"channel_id" gorm:"index;default:0"`
	ChannelName                string `json:"channel_name" gorm:"default:''"`
	ModelName                  string `json:"model_name" gorm:"index;default:''"`
	RequestPath                string `json:"request_path" gorm:"type:varchar(255);index;default:''"`
	SessionId                  string `json:"session_id" gorm:"type:varchar(128);index;default:''"`
	UserText                   string `json:"user_text" gorm:"type:text"`
	AssistantText              string `json:"assistant_text" gorm:"type:text"`
	RequestBody                string `json:"request_body" gorm:"type:longtext"`
	ResponseBody               string `json:"response_body" gorm:"type:longtext"`
	ResponseReasoningBody      string `json:"response_reasoning_body" gorm:"type:longtext"`
	RequestBodyTruncated       bool   `json:"request_body_truncated" gorm:"default:false"`
	ResponseBodyTruncated      bool   `json:"response_body_truncated" gorm:"default:false"`
	ResponseReasoningTruncated bool   `json:"response_reasoning_truncated" gorm:"default:false"`
	PromptTokens               int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens           int    `json:"completion_tokens" gorm:"default:0"`
	IsStream                   bool   `json:"is_stream" gorm:"default:false"`
	Status                     string `json:"status" gorm:"type:varchar(32);index;default:'ok'"`
	ErrorMessage               string `json:"error_message" gorm:"type:text"`
	Group                      string `json:"group" gorm:"index;default:''"`
	CreatedAt                  int64  `json:"created_at" gorm:"bigint;index:idx_conversation_created_id,priority:1"`
	ExportedAt                 int64  `json:"exported_at" gorm:"bigint;index;default:0"`
	FinalRequestRelayFormat    string `json:"final_request_relay_format" gorm:"type:varchar(32);default:''"`
	OriginalRequestRelayFormat string `json:"original_request_relay_format" gorm:"type:varchar(32);default:''"`
}

type RecordConversationLogParams struct {
	Usage        *dto.Usage
	Status       string
	ErrorMessage string
}

type ConversationLogQuery struct {
	StartTimestamp int64
	EndTimestamp   int64
	AfterId        int
	Limit          int
	Exported       *bool
	RequestId      string
	Username       string
	TokenName      string
	ModelName      string
	Group          string
	SessionId      string
	Content        string
}

type ConversationSessionQuery = ConversationLogQuery

type ConversationSessionSummary struct {
	SessionKey          string   `json:"session_key"`
	SessionId           string   `json:"session_id"`
	UserId              int      `json:"user_id"`
	Username            string   `json:"username"`
	LogCount            int      `json:"log_count"`
	FirstLogId          int      `json:"first_log_id"`
	LastLogId           int      `json:"last_log_id"`
	FirstCreatedAt      int64    `json:"first_created_at"`
	LastCreatedAt       int64    `json:"last_created_at"`
	Models              []string `json:"models"`
	TokenNames          []string `json:"token_names"`
	Groups              []string `json:"groups"`
	LatestUserText      string   `json:"latest_user_text"`
	LatestAssistantText string   `json:"latest_assistant_text"`
	PromptTokens        int      `json:"prompt_tokens"`
	CompletionTokens    int      `json:"completion_tokens"`
	Status              string   `json:"status"`
}

type ConversationSessionDetail struct {
	Summary   ConversationSessionSummary `json:"summary"`
	Logs      []*ConversationLog         `json:"logs"`
	Truncated bool                       `json:"truncated"`
}

func captureTextWithLimit(text string) (string, bool) {
	maxBytes := common.ConversationLogMaxBodyBytes
	if maxBytes <= 0 || len(text) <= maxBytes {
		return text, false
	}
	return text[:maxBytes], true
}

func requestBodyFromContext(c *gin.Context) (string, bool, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return "", false, err
	}
	body, err := storage.Bytes()
	if err != nil {
		return "", false, err
	}
	bodyText, truncated := captureTextWithLimit(string(body))
	return bodyText, truncated, nil
}

func buildConversationLogSearchFields(requestBody string, responseBody string) conversationLogSearchFields {
	fields := conversationLogSearchFields{
		AssistantText: conversationAssistantTextFromResponseBody(responseBody),
	}
	request, err := parseConversationCleanRequest(requestBody)
	if err != nil {
		return fields
	}
	fields.SessionId = extractSessionID(request.Metadata)
	fields.UserText = truncateConversationLogSearchText(strings.TrimSpace(lastUserMessageTextWithPlaceholders(request.Messages)))
	if fields.UserText == "" {
		fields.UserText = truncateConversationLogSearchText(strings.TrimSpace(lastUserInputTextWithPlaceholders(request.Input)))
	}
	return fields
}

func hydrateConversationLogDisplayFields(log *ConversationLog) {
	if log == nil {
		return
	}
	if log.Username == "" && log.UserId > 0 {
		if username, err := GetUsernameById(log.UserId, false); err == nil {
			log.Username = username
		}
	}
	if log.TokenName == "" && log.TokenId > 0 {
		if token, err := GetTokenById(log.TokenId); err == nil && token != nil {
			log.TokenName = token.Name
		}
	}

	var request *conversationRequestForClean
	loadRequest := func() *conversationRequestForClean {
		if request != nil {
			return request
		}
		parsed, err := parseConversationCleanRequest(log.RequestBody)
		if err != nil {
			return nil
		}
		request = parsed
		return request
	}

	if log.SessionId == "" {
		if parsed := loadRequest(); parsed != nil {
			log.SessionId = extractSessionID(parsed.Metadata)
		}
	}
	if log.UserText == "" {
		if parsed := loadRequest(); parsed != nil {
			log.UserText = truncateConversationLogSearchText(strings.TrimSpace(lastUserMessageTextWithPlaceholders(parsed.Messages)))
			if log.UserText == "" {
				log.UserText = truncateConversationLogSearchText(strings.TrimSpace(lastUserInputTextWithPlaceholders(parsed.Input)))
			}
		}
	}
	if log.AssistantText == "" {
		log.AssistantText = conversationAssistantTextFromResponseBody(log.ResponseBody)
	}
	if log.ModelName == "" {
		if parsed := loadRequest(); parsed != nil {
			log.ModelName = parsed.Model
		}
	}
}

func hydrateConversationLogDisplayFieldsFromBodies(logs []*ConversationLog) error {
	missingIds := make([]int, 0)
	logById := make(map[int]*ConversationLog, len(logs))
	for _, log := range logs {
		if log == nil {
			continue
		}
		hydrateConversationLogDisplayFields(log)
		logById[log.Id] = log
		if log.Id != 0 && (log.SessionId == "" || log.UserText == "" || log.AssistantText == "" || log.ModelName == "") {
			missingIds = append(missingIds, log.Id)
		}
	}
	if len(missingIds) == 0 {
		return nil
	}

	var bodies []*ConversationLog
	if err := LOG_DB.
		Model(&ConversationLog{}).
		Select("id, request_body, response_body").
		Where("id IN ?", missingIds).
		Find(&bodies).Error; err != nil {
		return err
	}
	for _, body := range bodies {
		log := logById[body.Id]
		if log == nil {
			continue
		}
		log.RequestBody = body.RequestBody
		log.ResponseBody = body.ResponseBody
		hydrateConversationLogDisplayFields(log)
	}
	return nil
}

func lastUserInputTextWithPlaceholders(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return lastUserTextFromInputValueWithPlaceholders(value)
}

func lastUserTextFromInputValueWithPlaceholders(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		for i := len(typed) - 1; i >= 0; i-- {
			if text := strings.TrimSpace(lastUserTextFromInputValueWithPlaceholders(typed[i])); text != "" {
				return text
			}
		}
	case map[string]any:
		if role := strings.TrimSpace(textFromValue(typed["role"])); role != "" && !strings.EqualFold(role, "user") {
			return ""
		}
		for _, key := range []string{"content", "input", "text"} {
			if raw, ok := typed[key]; ok {
				if text := strings.TrimSpace(textFromValueWithPlaceholders(raw)); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func conversationAssistantTextFromResponseBody(responseBody string) string {
	trimmed := strings.TrimSpace(responseBody)
	if trimmed == "" {
		return ""
	}

	var value any
	if err := common.Unmarshal([]byte(trimmed), &value); err != nil {
		return truncateConversationLogSearchText(trimmed)
	}
	if text := strings.TrimSpace(assistantTextFromResponseValue(value)); text != "" {
		return truncateConversationLogSearchText(text)
	}
	return truncateConversationLogSearchText(trimmed)
}

func assistantTextFromResponseValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(assistantTextFromResponseValue(item)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		for _, key := range []string{"message", "delta"} {
			if raw, ok := typed[key]; ok {
				if text := strings.TrimSpace(assistantTextFromResponseValue(raw)); text != "" {
					return text
				}
			}
		}
		for _, key := range []string{"text", "output_text", "content", "parts", "output", "choices", "candidates", "data"} {
			if raw, ok := typed[key]; ok {
				if text := strings.TrimSpace(assistantTextFromResponseValue(raw)); text != "" {
					return text
				}
			}
		}
		return ""
	default:
		return ""
	}
}

func lastUserMessageTextWithPlaceholders(messages []conversationMessageClean) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "user") {
			return textFromRawJSONWithPlaceholders(messages[i].Content)
		}
	}
	return ""
}

func textFromRawJSONWithPlaceholders(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return textFromValueWithPlaceholders(value)
}

func textFromValueWithPlaceholders(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(textFromValueWithPlaceholders(item)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		contentType := strings.TrimSpace(strings.ToLower(textFromValue(typed["type"])))
		if placeholder := conversationContentPlaceholder(contentType); placeholder != "" {
			return placeholder
		}
		for _, key := range []string{"text", "content", "input_text", "output_text"} {
			if raw, ok := typed[key]; ok {
				if text := strings.TrimSpace(textFromValueWithPlaceholders(raw)); text != "" {
					return text
				}
			}
		}
		if _, ok := typed["image_url"]; ok {
			return "[image]"
		}
		if _, ok := typed["file"]; ok {
			return "[file]"
		}
		return ""
	default:
		return ""
	}
}

func conversationContentPlaceholder(contentType string) string {
	switch {
	case contentType == "text" || contentType == "input_text" || contentType == "output_text":
		return ""
	case strings.Contains(contentType, "image"):
		return "[image]"
	case strings.Contains(contentType, "file"):
		return "[file]"
	case strings.Contains(contentType, "tool"):
		return "[tool]"
	default:
		return ""
	}
}

func truncateConversationLogSearchText(text string) string {
	if len(text) <= conversationLogSearchTextMaxBytes {
		return text
	}
	lastBoundary := 0
	for idx := range text {
		if idx > conversationLogSearchTextMaxBytes {
			break
		}
		lastBoundary = idx
	}
	if lastBoundary == conversationLogSearchTextMaxBytes {
		return text[:conversationLogSearchTextMaxBytes]
	}
	return text[:lastBoundary]
}

func SetConversationResponseBody(c *gin.Context, responseBody string) {
	if !common.ConversationLogEnabled || c == nil {
		return
	}
	body, truncated := captureTextWithLimit(responseBody)
	common.SetContextKey(c, constant.ContextKeyConversationResponseBody, body)
	common.SetContextKey(c, constant.ContextKeyConversationResponseTruncated, truncated)
}

func SetConversationReasoningBody(c *gin.Context, reasoningBody string) {
	if !common.ConversationLogEnabled || c == nil {
		return
	}
	body, truncated := captureTextWithLimit(reasoningBody)
	common.SetContextKey(c, constant.ContextKeyConversationReasoningBody, body)
	common.SetContextKey(c, constant.ContextKeyConversationReasoningTruncated, truncated)
}

func SetConversationResponseParts(c *gin.Context, responseBody string, reasoningBody string) {
	SetConversationResponseBody(c, responseBody)
	SetConversationReasoningBody(c, reasoningBody)
}

func RecordConversationLog(c *gin.Context, relayInfo *relaycommon.RelayInfo, params RecordConversationLogParams) {
	if !common.ConversationLogEnabled || c == nil || relayInfo == nil {
		return
	}
	requestBody, requestTruncated, err := requestBodyFromContext(c)
	if err != nil {
		common.SysError("failed to read conversation request body: " + err.Error())
	}
	responseBody := common.GetContextKeyString(c, constant.ContextKeyConversationResponseBody)
	reasoningBody := common.GetContextKeyString(c, constant.ContextKeyConversationReasoningBody)
	searchFields := buildConversationLogSearchFields(requestBody, responseBody)
	responseTruncated := common.GetContextKeyBool(c, constant.ContextKeyConversationResponseTruncated)
	reasoningTruncated := common.GetContextKeyBool(c, constant.ContextKeyConversationReasoningTruncated)
	status := strings.TrimSpace(params.Status)
	if status == "" {
		status = ConversationLogStatusOK
	}
	errorMessage := params.ErrorMessage
	if relayInfo.StreamStatus != nil && !relayInfo.StreamStatus.IsNormalEnd() {
		if relayInfo.StreamStatus.EndReason != "" {
			status = string(relayInfo.StreamStatus.EndReason)
		} else {
			status = "stream_error"
		}
		if errorMessage == "" && relayInfo.StreamStatus.EndError != nil {
			errorMessage = relayInfo.StreamStatus.EndError.Error()
		}
	}

	promptTokens := 0
	completionTokens := 0
	if params.Usage != nil {
		promptTokens = params.Usage.PromptTokens
		completionTokens = params.Usage.CompletionTokens
	}

	requestPath := ""
	if c.Request != nil && c.Request.URL != nil {
		requestPath = c.Request.URL.Path
	}

	log := &ConversationLog{
		RequestId:                  c.GetString(common.RequestIdKey),
		UserId:                     relayInfo.UserId,
		Username:                   c.GetString("username"),
		TokenId:                    relayInfo.TokenId,
		TokenName:                  c.GetString("token_name"),
		ChannelId:                  relayInfo.ChannelId,
		ChannelName:                c.GetString("channel_name"),
		ModelName:                  relayInfo.OriginModelName,
		RequestPath:                requestPath,
		SessionId:                  searchFields.SessionId,
		UserText:                   searchFields.UserText,
		AssistantText:              searchFields.AssistantText,
		RequestBody:                requestBody,
		ResponseBody:               responseBody,
		ResponseReasoningBody:      reasoningBody,
		RequestBodyTruncated:       requestTruncated,
		ResponseBodyTruncated:      responseTruncated,
		ResponseReasoningTruncated: reasoningTruncated,
		PromptTokens:               promptTokens,
		CompletionTokens:           completionTokens,
		IsStream:                   relayInfo.IsStream,
		Status:                     status,
		ErrorMessage:               errorMessage,
		Group:                      relayInfo.UsingGroup,
		CreatedAt:                  common.GetTimestamp(),
		FinalRequestRelayFormat:    string(relayInfo.GetFinalRequestRelayFormat()),
		OriginalRequestRelayFormat: string(relayInfo.RelayFormat),
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysError("failed to record conversation log: " + err.Error())
	}
}

func conversationLogQuery(q ConversationLogQuery) *gorm.DB {
	tx := LOG_DB.Model(&ConversationLog{})
	if q.StartTimestamp != 0 {
		tx = tx.Where("created_at >= ?", q.StartTimestamp)
	}
	if q.EndTimestamp != 0 {
		tx = tx.Where("created_at <= ?", q.EndTimestamp)
	}
	if q.AfterId != 0 {
		tx = tx.Where("id > ?", q.AfterId)
	}
	if q.Exported != nil {
		if *q.Exported {
			tx = tx.Where("exported_at > 0")
		} else {
			tx = tx.Where("exported_at = 0")
		}
	}
	if q.RequestId != "" {
		tx = tx.Where("request_id = ?", q.RequestId)
	}
	if q.Username != "" {
		tx = tx.Where("username = ?", q.Username)
	}
	if q.TokenName != "" {
		tx = tx.Where("token_name = ?", q.TokenName)
	}
	if q.ModelName != "" {
		tx = tx.Where("model_name = ?", q.ModelName)
	}
	if q.Group != "" {
		tx = tx.Where(clause.Eq{Column: clause.Column{Name: "group"}, Value: q.Group})
	}
	if q.SessionId != "" {
		tx = tx.Where("session_id = ?", q.SessionId)
	}
	if q.Content != "" {
		like := "%" + q.Content + "%"
		tx = tx.Where("(user_text LIKE ? OR assistant_text LIKE ? OR request_body LIKE ? OR response_body LIKE ?)", like, like, like, like)
	}
	return tx
}

func GetConversationLogs(q ConversationLogQuery, startIdx int, num int) (logs []*ConversationLog, total int64, err error) {
	tx := conversationLogQuery(q)
	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if num <= 0 || num > 1000 {
		num = 100
	}
	err = tx.Order("id asc").Limit(num).Offset(startIdx).Find(&logs).Error
	return logs, total, err
}

func GetConversationSessions(q ConversationSessionQuery, startIdx int, num int) (sessions []ConversationSessionSummary, total int64, err error) {
	if num <= 0 || num > 1000 {
		num = 100
	}
	var logs []*ConversationLog
	err = conversationLogQuery(q).
		Select([]string{"id", "request_id", "user_id", "username", "token_id", "token_name", "channel_id", "channel_name", "model_name", "request_path", "session_id", "user_text", "assistant_text", "prompt_tokens", "completion_tokens", "is_stream", "status", "error_message", "group", "created_at", "exported_at", "final_request_relay_format", "original_request_relay_format"}).
		Order("created_at desc, id desc").
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	if err = hydrateConversationLogDisplayFieldsFromBodies(logs); err != nil {
		return nil, 0, err
	}

	summaryByKey := make(map[string]*ConversationSessionSummary)
	keys := make([]string, 0)
	for _, log := range logs {
		key := conversationSessionKey(log)
		summary, ok := summaryByKey[key]
		if !ok {
			summary = &ConversationSessionSummary{
				SessionKey:          key,
				SessionId:           log.SessionId,
				UserId:              log.UserId,
				Username:            log.Username,
				FirstLogId:          log.Id,
				LastLogId:           log.Id,
				FirstCreatedAt:      log.CreatedAt,
				LastCreatedAt:       log.CreatedAt,
				LatestUserText:      log.UserText,
				LatestAssistantText: log.AssistantText,
				Status:              log.Status,
			}
			summaryByKey[key] = summary
			keys = append(keys, key)
		}
		mergeConversationSessionSummary(summary, log, true)
	}

	total = int64(len(keys))
	if startIdx >= len(keys) {
		return []ConversationSessionSummary{}, total, nil
	}
	endIdx := startIdx + num
	if endIdx > len(keys) {
		endIdx = len(keys)
	}
	sessions = make([]ConversationSessionSummary, 0, endIdx-startIdx)
	for _, key := range keys[startIdx:endIdx] {
		sessions = append(sessions, *summaryByKey[key])
	}
	return sessions, total, nil
}

func GetConversationSessionDetail(q ConversationSessionQuery, sessionKey string, limit int) (*ConversationSessionDetail, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	tx, err := conversationSessionDetailQuery(q, sessionKey)
	if err != nil {
		return nil, err
	}
	logs := make([]*ConversationLog, 0, limit+1)
	if err := tx.Order("created_at desc, id desc").Limit(limit + 1).Find(&logs).Error; err != nil {
		return nil, err
	}
	truncated := len(logs) > limit
	if truncated {
		logs = logs[:limit]
	}
	for _, log := range logs {
		hydrateConversationLogDisplayFields(log)
	}
	sort.SliceStable(logs, func(i, j int) bool {
		if logs[i].CreatedAt == logs[j].CreatedAt {
			return logs[i].Id < logs[j].Id
		}
		return logs[i].CreatedAt < logs[j].CreatedAt
	})

	summary := ConversationSessionSummary{SessionKey: sessionKey}
	for _, log := range logs {
		if summary.SessionKey == "" {
			summary.SessionKey = conversationSessionKey(log)
		}
		mergeConversationSessionSummary(&summary, log, false)
	}
	return &ConversationSessionDetail{Summary: summary, Logs: logs, Truncated: truncated}, nil
}

func conversationSessionKey(log *ConversationLog) string {
	if log == nil {
		return ""
	}
	if log.SessionId != "" {
		return fmt.Sprintf("user:%d:session:%s", log.UserId, log.SessionId)
	}
	return fmt.Sprintf("log:%d", log.Id)
}

func conversationSessionDetailQuery(q ConversationSessionQuery, sessionKey string) (*gorm.DB, error) {
	tx := conversationLogQuery(q)
	if strings.HasPrefix(sessionKey, "log:") {
		id, err := strconv.Atoi(strings.TrimPrefix(sessionKey, "log:"))
		if err != nil {
			return nil, err
		}
		return tx.Where("id = ?", id), nil
	}
	if strings.HasPrefix(sessionKey, "user:") {
		parts := strings.SplitN(strings.TrimPrefix(sessionKey, "user:"), ":session:", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid conversation session key")
		}
		userId, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}
		return tx.Where("user_id = ? AND session_id = ?", userId, parts[1]), nil
	}
	return nil, fmt.Errorf("invalid conversation session key")
}

func mergeConversationSessionSummary(summary *ConversationSessionSummary, log *ConversationLog, descending bool) {
	if summary == nil || log == nil {
		return
	}
	if summary.SessionKey == "" {
		summary.SessionKey = conversationSessionKey(log)
	}
	if summary.SessionId == "" {
		summary.SessionId = log.SessionId
	}
	if summary.UserId == 0 {
		summary.UserId = log.UserId
	}
	if summary.Username == "" {
		summary.Username = log.Username
	}
	if summary.FirstLogId == 0 || log.CreatedAt < summary.FirstCreatedAt || (log.CreatedAt == summary.FirstCreatedAt && log.Id < summary.FirstLogId) {
		summary.FirstLogId = log.Id
		summary.FirstCreatedAt = log.CreatedAt
	}
	if summary.LastLogId == 0 || log.CreatedAt > summary.LastCreatedAt || (log.CreatedAt == summary.LastCreatedAt && log.Id > summary.LastLogId) {
		summary.LastLogId = log.Id
		summary.LastCreatedAt = log.CreatedAt
		summary.LatestUserText = log.UserText
		summary.LatestAssistantText = log.AssistantText
		summary.Status = log.Status
	}
	if descending {
		summary.Models = prependUniqueConversationModel(summary.Models, log.ModelName)
		summary.TokenNames = prependUniqueConversationValue(summary.TokenNames, log.TokenName)
		summary.Groups = prependUniqueConversationValue(summary.Groups, log.Group)
	} else {
		summary.Models = appendUniqueConversationModel(summary.Models, log.ModelName)
		summary.TokenNames = appendUniqueConversationValue(summary.TokenNames, log.TokenName)
		summary.Groups = appendUniqueConversationValue(summary.Groups, log.Group)
	}
	summary.LogCount++
	summary.PromptTokens += log.PromptTokens
	summary.CompletionTokens += log.CompletionTokens
}

func appendUniqueConversationModel(models []string, modelName string) []string {
	return appendUniqueConversationValue(models, modelName)
}

func prependUniqueConversationModel(models []string, modelName string) []string {
	return prependUniqueConversationValue(models, modelName)
}

func appendUniqueConversationValue(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func prependUniqueConversationValue(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append([]string{value}, values...)
}

func ExportConversationLogs(q ConversationLogQuery) (logs []*ConversationLog, err error) {
	limit := q.Limit
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	err = conversationLogQuery(q).Order("id asc").Limit(limit).Find(&logs).Error
	return logs, err
}

func MarkConversationLogsExported(ids []int, exportedAt int64) error {
	if len(ids) == 0 {
		return nil
	}
	return LOG_DB.Model(&ConversationLog{}).Where("id IN ?", ids).Update("exported_at", exportedAt).Error
}
