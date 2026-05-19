package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func buildConversationLogQuery(c *gin.Context) model.ConversationLogQuery {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	afterId, _ := strconv.Atoi(c.Query("after_id"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	var exported *bool
	if raw := c.Query("exported"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err == nil {
			exported = &value
		}
	}

	return model.ConversationLogQuery{
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		AfterId:        afterId,
		Limit:          limit,
		Exported:       exported,
		RequestId:      c.Query("request_id"),
		Username:       c.Query("username"),
		TokenName:      c.Query("token_name"),
		ModelName:      c.Query("model_name"),
		Group:          c.Query("group"),
		SessionId:      c.Query("session_id"),
		Content:        c.Query("content"),
	}
}

func GetConversationLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	logs, total, err := model.GetConversationLogs(buildConversationLogQuery(c), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}

func GetConversationSessions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	sessions, total, err := model.GetConversationSessions(buildConversationLogQuery(c), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(sessions)
	common.ApiSuccess(c, pageInfo)
}

func GetConversationSessionDetail(c *gin.Context) {
	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		common.ApiErrorMsg(c, "session_key is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	detail, err := model.GetConversationSessionDetail(buildConversationLogQuery(c), sessionKey, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}

func ExportConversationLogs(c *gin.Context) {
	query := buildConversationLogQuery(c)
	logs, err := model.ExportConversationLogs(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=conversation_logs.jsonl")
	c.Status(http.StatusOK)

	ids := make([]int, 0, len(logs))
	for _, log := range logs {
		line, err := common.Marshal(log)
		if err != nil {
			common.SysError("failed to encode conversation log: " + err.Error())
			return
		}
		line = append(line, '\n')
		if _, err := c.Writer.Write(line); err != nil {
			return
		}
		ids = append(ids, log.Id)
	}

	markExported, _ := strconv.ParseBool(c.DefaultQuery("mark_exported", "false"))
	if markExported {
		if err := model.MarkConversationLogsExported(ids, common.GetTimestamp()); err != nil {
			common.SysError("failed to mark conversation logs exported: " + err.Error())
		}
	}
}
