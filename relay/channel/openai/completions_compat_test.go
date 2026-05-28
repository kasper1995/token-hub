package openai

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func TestOpenAICompletionsCompatRequestURL(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeOpenAICompletions,
			ChannelBaseUrl: "http://example.test/base/",
		},
		RelayMode: relayconstant.RelayModeChatCompletions,
	}

	got, err := adaptor.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL returned error: %v", err)
	}
	want := "http://example.test/base/v1/completions"
	if got != want {
		t.Fatalf("GetRequestURL() = %q, want %q", got, want)
	}
}

func TestNormalizeOpenAICompletionsRequest(t *testing.T) {
	maxCompletionTokens := uint(200)
	request := &dto.GeneralOpenAIRequest{
		Model:               "glm-5",
		MaxCompletionTokens: &maxCompletionTokens,
		Messages: []dto.Message{
			{Role: "system", Content: "be concise"},
			{Role: "user", Content: "hi"},
		},
		Tools: []dto.ToolCallRequest{{Type: "function"}},
	}

	normalizeOpenAICompletionsRequest(request)

	if request.MaxTokens == nil || *request.MaxTokens != 200 {
		t.Fatalf("MaxTokens = %v, want 200", request.MaxTokens)
	}
	if request.MaxCompletionTokens != nil {
		t.Fatalf("MaxCompletionTokens was not removed")
	}
	if len(request.Messages) != 0 {
		t.Fatalf("Messages length = %d, want 0", len(request.Messages))
	}
	prompt, ok := request.Prompt.(string)
	if !ok {
		t.Fatalf("Prompt type = %T, want string", request.Prompt)
	}
	for _, part := range []string{"System: be concise", "User: hi", "Assistant:"} {
		if !strings.Contains(prompt, part) {
			t.Fatalf("Prompt %q does not contain %q", prompt, part)
		}
	}
	if request.Tools != nil {
		t.Fatalf("Tools was not removed")
	}
}

func TestNormalizeOpenAICompletionsRequestPrefersMaxCompletionTokens(t *testing.T) {
	maxTokens := uint(64)
	maxCompletionTokens := uint(200)
	request := &dto.GeneralOpenAIRequest{
		Model:               "glm-5",
		MaxTokens:           &maxTokens,
		MaxCompletionTokens: &maxCompletionTokens,
	}

	normalizeOpenAICompletionsRequest(request)

	if request.MaxTokens == nil || *request.MaxTokens != 200 {
		t.Fatalf("MaxTokens = %v, want 200", request.MaxTokens)
	}
	if request.MaxCompletionTokens != nil {
		t.Fatalf("MaxCompletionTokens was not removed")
	}
}

func TestNormalizeCompletionsStreamDataForChat(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenAICompletions,
		},
		RelayMode: relayconstant.RelayModeChatCompletions,
	}
	data := `{"id":"cmpl-1","object":"text_completion","created":1,"model":"glm-5","choices":[{"text":"hello","finish_reason":null,"index":0}]}`

	normalized, ok := normalizeCompletionsStreamDataForChat(info, data)
	if !ok {
		t.Fatalf("normalizeCompletionsStreamDataForChat did not normalize")
	}
	var response dto.ChatCompletionsStreamResponse
	if err := common.UnmarshalJsonStr(normalized, &response); err != nil {
		t.Fatalf("normalized response is invalid: %v", err)
	}
	if got := response.Choices[0].Delta.GetContentString(); got != "hello" {
		t.Fatalf("content = %q, want hello", got)
	}
}
