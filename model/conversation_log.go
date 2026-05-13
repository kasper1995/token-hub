package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const ConversationLogStatusOK = "ok"

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
	ModelName      string
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
	if q.ModelName != "" {
		tx = tx.Where("model_name = ?", q.ModelName)
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
