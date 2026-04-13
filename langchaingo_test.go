package langchaingo

import (
	"context"
	"testing"

	"github.com/plexusone/omnillm-core/provider"
	"github.com/tmc/langchaingo/llms"
)

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	response *provider.ChatCompletionResponse
	err      error
}

func (m *mockProvider) CreateChatCompletion(_ context.Context, _ *provider.ChatCompletionRequest) (*provider.ChatCompletionResponse, error) {
	return m.response, m.err
}

func (m *mockProvider) CreateChatCompletionStream(_ context.Context, _ *provider.ChatCompletionRequest) (provider.ChatCompletionStream, error) {
	return nil, nil
}

func (m *mockProvider) Close() error {
	return nil
}

func (m *mockProvider) Name() string {
	return "mock"
}

func TestNew(t *testing.T) {
	prov := &mockProvider{}
	model := New(prov, "test-model")

	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if model.model != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", model.model)
	}
}

func TestGenerateContent(t *testing.T) {
	finishReason := "stop"
	prov := &mockProvider{
		response: &provider.ChatCompletionResponse{
			Choices: []provider.ChatCompletionChoice{
				{
					Message: provider.Message{
						Role:    provider.RoleAssistant,
						Content: "Hello, world!",
					},
					FinishReason: &finishReason,
				},
			},
			Usage: provider.Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		},
	}

	model := New(prov, "test-model")

	resp, err := model.GenerateContent(context.Background(), []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextContent{Text: "Hello"}},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got '%s'", resp.Choices[0].Content)
	}

	if resp.Choices[0].StopReason != "stop" {
		t.Errorf("expected stop reason 'stop', got '%s'", resp.Choices[0].StopReason)
	}

	// Check usage info
	genInfo := resp.Choices[0].GenerationInfo
	if genInfo == nil {
		t.Fatal("expected generation info")
	}
	if genInfo["total_tokens"] != 15 {
		t.Errorf("expected total_tokens 15, got %v", genInfo["total_tokens"])
	}
}

func TestCall(t *testing.T) {
	prov := &mockProvider{
		response: &provider.ChatCompletionResponse{
			Choices: []provider.ChatCompletionChoice{
				{
					Message: provider.Message{
						Role:    provider.RoleAssistant,
						Content: "The answer is 42",
					},
				},
			},
		},
	}

	model := New(prov, "test-model")

	result, err := model.Call(context.Background(), "What is the meaning of life?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "The answer is 42" {
		t.Errorf("expected 'The answer is 42', got '%s'", result)
	}
}

func TestMapRoleToOmnillm(t *testing.T) {
	tests := []struct {
		input    llms.ChatMessageType
		expected provider.Role
	}{
		{llms.ChatMessageTypeSystem, provider.RoleSystem},
		{llms.ChatMessageTypeHuman, provider.RoleUser},
		{llms.ChatMessageTypeAI, provider.RoleAssistant},
		{llms.ChatMessageTypeTool, provider.RoleTool},
		{llms.ChatMessageTypeFunction, provider.RoleTool},
	}

	for _, tt := range tests {
		result := mapRoleToOmnillm(tt.input)
		if result != tt.expected {
			t.Errorf("mapRoleToOmnillm(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestConvertToolsToOmnillm(t *testing.T) {
	tools := []llms.Tool{
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "get_weather",
				Description: "Get the weather",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	result := convertToolsToOmnillm(tools)

	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	if result[0].Type != "function" {
		t.Errorf("expected type 'function', got '%s'", result[0].Type)
	}

	if result[0].Function.Name != "get_weather" {
		t.Errorf("expected name 'get_weather', got '%s'", result[0].Function.Name)
	}
}

func TestGenerateContentWithToolCalls(t *testing.T) {
	finishReason := "tool_calls"
	prov := &mockProvider{
		response: &provider.ChatCompletionResponse{
			Choices: []provider.ChatCompletionChoice{
				{
					Message: provider.Message{
						Role:    provider.RoleAssistant,
						Content: "",
						ToolCalls: []provider.ToolCall{
							{
								ID:   "call_123",
								Type: "function",
								Function: provider.ToolFunction{
									Name:      "get_weather",
									Arguments: `{"location":"Paris"}`,
								},
							},
						},
					},
					FinishReason: &finishReason,
				},
			},
		},
	}

	model := New(prov, "test-model")

	resp, err := model.GenerateContent(context.Background(), []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextContent{Text: "What's the weather in Paris?"}},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Choices[0].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Choices[0].ToolCalls))
	}

	tc := resp.Choices[0].ToolCalls[0]
	if tc.ID != "call_123" {
		t.Errorf("expected tool call ID 'call_123', got '%s'", tc.ID)
	}
	if tc.FunctionCall.Name != "get_weather" {
		t.Errorf("expected function name 'get_weather', got '%s'", tc.FunctionCall.Name)
	}
}
