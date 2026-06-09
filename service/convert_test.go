package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestClaudeToOpenAIRequestAppliesMinUpstreamMaxTokens(t *testing.T) {
	maxTokens := uint(80)
	request := dto.ClaudeRequest{
		Model:     "claude-opus-4-7",
		MaxTokens: &maxTokens,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}

	openAI, err := ClaudeToOpenAIRequest(request, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelSetting: dto.ChannelSettings{MinUpstreamMaxTokens: 300},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, openAI.MaxTokens)
	require.Equal(t, uint(300), *openAI.MaxTokens)
}

func TestClaudeToOpenAIRequestKeepsLargerMaxTokens(t *testing.T) {
	maxTokens := uint(1200)
	request := dto.ClaudeRequest{
		Model:     "claude-opus-4-7",
		MaxTokens: &maxTokens,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}

	openAI, err := ClaudeToOpenAIRequest(request, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelSetting: dto.ChannelSettings{MinUpstreamMaxTokens: 300},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, openAI.MaxTokens)
	require.Equal(t, uint(1200), *openAI.MaxTokens)
}

func TestResponseOpenAI2ClaudeReturnsOriginModelWhenMapped(t *testing.T) {
	message := dto.Message{Role: "assistant"}
	message.SetStringContent("visible answer")
	response := &dto.OpenAITextResponse{
		Id:    "chatcmpl-test",
		Model: "glm-5",
		Choices: []dto.OpenAITextResponseChoice{
			{Message: message, FinishReason: "stop"},
		},
	}

	claude := ResponseOpenAI2Claude(response, &relaycommon.RelayInfo{
		OriginModelName: "claude-opus-4-7",
	})

	require.Equal(t, "claude-opus-4-7", claude.Model)
}

func TestStreamResponseOpenAI2ClaudeKeepsFinalDeltaBeforeUsageChunk(t *testing.T) {
	text := "5。"
	finish := "stop"
	info := &relaycommon.RelayInfo{
		SendResponseCount: 2,
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			LastMessagesType: relaycommon.LastMessageTypeNone,
		},
	}

	finalDelta := &dto.ChatCompletionsStreamResponse{
		Id:    "chatcmpl-test",
		Model: "glm-5",
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{
				Index:        0,
				FinishReason: &finish,
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
					Content: &text,
				},
			},
		},
	}

	responses := StreamResponseOpenAI2Claude(finalDelta, info)

	require.Len(t, responses, 2)
	require.Equal(t, "content_block_start", responses[0].Type)
	require.Equal(t, "content_block_delta", responses[1].Type)
	require.Equal(t, "text_delta", responses[1].Delta.Type)
	require.Equal(t, "5。", *responses[1].Delta.Text)
	require.False(t, info.ClaudeConvertInfo.Done)

	usageOnly := &dto.ChatCompletionsStreamResponse{
		Id:      "chatcmpl-test",
		Model:   "glm-5",
		Choices: nil,
		Usage: &dto.Usage{
			PromptTokens:     1,
			CompletionTokens: 2,
			TotalTokens:      3,
		},
	}

	responses = StreamResponseOpenAI2Claude(usageOnly, info)

	require.Len(t, responses, 3)
	require.Equal(t, "content_block_stop", responses[0].Type)
	require.Equal(t, "message_delta", responses[1].Type)
	require.Equal(t, "end_turn", *responses[1].Delta.StopReason)
	require.Equal(t, "message_stop", responses[2].Type)
	require.True(t, info.ClaudeConvertInfo.Done)
}
