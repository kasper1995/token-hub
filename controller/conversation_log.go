package controller

import (
	"encoding/json"
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
		ModelName:      c.Query("model_name"),
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

	encoder := json.NewEncoder(c.Writer)
	ids := make([]int, 0, len(logs))
	for _, log := range logs {
		if err := encoder.Encode(log); err != nil {
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
