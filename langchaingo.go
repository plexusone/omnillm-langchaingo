// Package langchaingo provides a LangChainGo adapter for omnillm.
//
// This package implements the github.com/tmc/langchaingo/llms.Model interface
// using omnillm as the LLM backend, allowing you to use omnillm's unified
// provider abstraction with LangChainGo's chains, agents, and RAG pipelines.
//
// Usage:
//
//	import (
//	    "github.com/plexusone/omnillm"
//	    "github.com/plexusone/omnillm-langchaingo"
//	)
//
//	client := omnillm.NewClient(omnillm.ClientConfig{
//	    Provider: omnillm.ProviderNameOpenAI,
//	    APIKey:   os.Getenv("OPENAI_API_KEY"),
//	})
//
//	llm := langchaingo.New(client, "gpt-4o")
//
//	// Use with LangChainGo chains, agents, etc.
//	chain := chains.NewLLMChain(llm, prompt)
package langchaingo

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/plexusone/omnillm-core/provider"
	"github.com/tmc/langchaingo/llms"
)

// Model implements the langchaingo llms.Model interface using omnillm.
type Model struct {
	prov  provider.Provider
	model string
}

// Ensure Model implements llms.Model at compile time.
var _ llms.Model = (*Model)(nil)

// New creates a new LangChainGo Model adapter for an omnillm Provider.
//
// The provider can be obtained from omnillm.NewClient() or any omnillm.Provider
// implementation. The model parameter specifies which model to use for requests.
func New(prov provider.Provider, model string) *Model {
	return &Model{
		prov:  prov,
		model: model,
	}
}

// GenerateContent implements llms.Model.
//
// It converts LangChainGo message types to omnillm format, calls the underlying
// provider, and converts the response back to LangChainGo format.
func (m *Model) GenerateContent(
	ctx context.Context,
	messages []llms.MessageContent,
	options ...llms.CallOption,
) (*llms.ContentResponse, error) {
	opts := &llms.CallOptions{}
	for _, opt := range options {
		opt(opts)
	}

	// Use model from options if specified, otherwise use default
	model := m.model
	if opts.Model != "" {
		model = opts.Model
	}

	// Convert LangChainGo messages to omnillm messages
	omniMessages, err := convertMessagesToOmnillm(messages)
	if err != nil {
		return nil, err
	}

	// Build omnillm request
	req := &provider.ChatCompletionRequest{
		Model:    model,
		Messages: omniMessages,
	}

	// Map call options to request fields
	applyCallOptions(req, opts)

	// Handle streaming if callback is provided
	if opts.StreamingFunc != nil {
		return m.generateStreaming(ctx, req, opts.StreamingFunc)
	}

	// Non-streaming call
	resp, err := m.prov.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	return convertResponseToLangChain(resp), nil
}

// Call implements llms.Model (simplified text-only interface).
//
// This is a convenience method that wraps GenerateContent for simple
// text-in/text-out use cases.
func (m *Model) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	resp, err := m.GenerateContent(ctx, []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextContent{Text: prompt}},
		},
	}, options...)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	return resp.Choices[0].Content, nil
}

// generateStreaming handles streaming responses.
func (m *Model) generateStreaming(
	ctx context.Context,
	req *provider.ChatCompletionRequest,
	streamFunc func(ctx context.Context, chunk []byte) error,
) (*llms.ContentResponse, error) {
	stream, err := m.prov.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.Close() }()

	var fullContent string
	var toolCalls []llms.ToolCall

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			delta := chunk.Choices[0].Delta

			// Handle text content
			if delta.Content != "" {
				fullContent += delta.Content
				if err := streamFunc(ctx, []byte(delta.Content)); err != nil {
					return nil, err
				}
			}

			// Accumulate tool calls from streaming chunks
			for _, tc := range delta.ToolCalls {
				toolCalls = appendOrMergeToolCall(toolCalls, tc)
			}
		}
	}

	choice := &llms.ContentChoice{
		Content:   fullContent,
		ToolCalls: toolCalls,
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{choice},
	}, nil
}

// applyCallOptions maps LangChainGo CallOptions to omnillm request fields.
func applyCallOptions(req *provider.ChatCompletionRequest, opts *llms.CallOptions) {
	if opts.MaxTokens > 0 {
		maxTokens := opts.MaxTokens
		req.MaxTokens = &maxTokens
	}

	if opts.Temperature > 0 {
		temp := opts.Temperature
		req.Temperature = &temp
	}

	if opts.TopP > 0 {
		topP := opts.TopP
		req.TopP = &topP
	}

	if len(opts.StopWords) > 0 {
		req.Stop = opts.StopWords
	}

	if opts.Seed > 0 {
		seed := opts.Seed
		req.Seed = &seed
	}

	if opts.FrequencyPenalty > 0 {
		fp := opts.FrequencyPenalty
		req.FrequencyPenalty = &fp
	}

	if opts.PresencePenalty > 0 {
		pp := opts.PresencePenalty
		req.PresencePenalty = &pp
	}

	// Convert tools
	if len(opts.Tools) > 0 {
		req.Tools = convertToolsToOmnillm(opts.Tools)
	}

	// Handle JSON mode
	if opts.JSONMode {
		req.ResponseFormat = &provider.ResponseFormat{
			Type: "json_object",
		}
	}
}

// convertMessagesToOmnillm converts LangChainGo messages to omnillm format.
func convertMessagesToOmnillm(msgs []llms.MessageContent) ([]provider.Message, error) {
	result := make([]provider.Message, 0, len(msgs))

	for _, msg := range msgs {
		omniMsg := provider.Message{
			Role: mapRoleToOmnillm(msg.Role),
		}

		// Extract content and tool calls from parts
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case llms.TextContent:
				omniMsg.Content = p.Text
			case llms.ToolCall:
				omniMsg.ToolCalls = append(omniMsg.ToolCalls, provider.ToolCall{
					ID:   p.ID,
					Type: p.Type,
					Function: provider.ToolFunction{
						Name:      p.FunctionCall.Name,
						Arguments: p.FunctionCall.Arguments,
					},
				})
			case llms.ToolCallResponse:
				omniMsg.Role = provider.RoleTool
				omniMsg.Content = p.Content
				toolCallID := p.ToolCallID
				omniMsg.ToolCallID = &toolCallID
			case llms.ImageURLContent:
				return nil, errors.New("omnillm-langchaingo: ImageURLContent not yet supported")
			case llms.BinaryContent:
				return nil, errors.New("omnillm-langchaingo: BinaryContent not yet supported")
			default:
				return nil, fmt.Errorf("omnillm-langchaingo: unsupported content type %T", p)
			}
		}

		result = append(result, omniMsg)
	}

	return result, nil
}

// mapRoleToOmnillm maps LangChainGo roles to omnillm roles.
func mapRoleToOmnillm(role llms.ChatMessageType) provider.Role {
	switch role {
	case llms.ChatMessageTypeSystem:
		return provider.RoleSystem
	case llms.ChatMessageTypeHuman:
		return provider.RoleUser
	case llms.ChatMessageTypeAI:
		return provider.RoleAssistant
	case llms.ChatMessageTypeTool:
		return provider.RoleTool
	case llms.ChatMessageTypeFunction:
		return provider.RoleTool
	default:
		return provider.RoleUser
	}
}

// convertToolsToOmnillm converts LangChainGo tools to omnillm format.
func convertToolsToOmnillm(tools []llms.Tool) []provider.Tool {
	result := make([]provider.Tool, 0, len(tools))

	for _, tool := range tools {
		if tool.Function != nil {
			result = append(result, provider.Tool{
				Type: "function",
				Function: provider.ToolSpec{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			})
		}
	}

	return result
}

// convertResponseToLangChain converts omnillm response to LangChainGo format.
func convertResponseToLangChain(resp *provider.ChatCompletionResponse) *llms.ContentResponse {
	choices := make([]*llms.ContentChoice, 0, len(resp.Choices))

	for _, choice := range resp.Choices {
		lc := &llms.ContentChoice{
			Content: choice.Message.Content,
		}

		// Handle finish reason (pointer to string)
		if choice.FinishReason != nil {
			lc.StopReason = *choice.FinishReason
		}

		// Convert tool calls if present
		for _, tc := range choice.Message.ToolCalls {
			lc.ToolCalls = append(lc.ToolCalls, llms.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				FunctionCall: &llms.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}

		// Add usage info to generation info
		if resp.Usage.TotalTokens > 0 {
			lc.GenerationInfo = map[string]any{
				"prompt_tokens":     resp.Usage.PromptTokens,
				"completion_tokens": resp.Usage.CompletionTokens,
				"total_tokens":      resp.Usage.TotalTokens,
			}
		}

		choices = append(choices, lc)
	}

	return &llms.ContentResponse{
		Choices: choices,
	}
}

// appendOrMergeToolCall handles streaming tool call chunks by merging them.
func appendOrMergeToolCall(calls []llms.ToolCall, tc provider.ToolCall) []llms.ToolCall {
	// Find existing tool call by ID to merge
	for i := range calls {
		if calls[i].ID == tc.ID {
			// Merge arguments (streaming sends in chunks)
			if tc.Function.Arguments != "" {
				calls[i].FunctionCall.Arguments += tc.Function.Arguments
			}
			return calls
		}
	}

	// New tool call
	return append(calls, llms.ToolCall{
		ID:   tc.ID,
		Type: tc.Type,
		FunctionCall: &llms.FunctionCall{
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		},
	})
}
