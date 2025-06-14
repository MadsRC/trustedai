// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package providers

import (
	"testing"

	"codeberg.org/gai-org/gai"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProvider_convertToGaiRequest(t *testing.T) {
	provider := NewOpenAIProvider()

	tests := []struct {
		name     string
		req      openai.ChatCompletionNewParams
		isStream bool
		want     func(*testing.T, gai.ResponseRequest)
	}{
		{
			name: "simple user message",
			req: openai.ChatCompletionNewParams{
				Model: "gpt-3.5-turbo",
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage("Hello, world!"),
				},
			},
			isStream: false,
			want: func(t *testing.T, result gai.ResponseRequest) {
				assert.Equal(t, "gpt-3.5-turbo", result.ModelID)
				assert.Equal(t, "", result.Instructions)
				assert.False(t, result.Stream)

				textInput, ok := result.Input.(gai.TextInput)
				require.True(t, ok, "Expected TextInput, got %T", result.Input)
				assert.Equal(t, "Hello, world!", textInput.Text)
			},
		},
		{
			name: "system message with user message",
			req: openai.ChatCompletionNewParams{
				Model: "gpt-4",
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.SystemMessage("You are a helpful assistant."),
					openai.UserMessage("What's the weather like?"),
				},
			},
			isStream: true,
			want: func(t *testing.T, result gai.ResponseRequest) {
				assert.Equal(t, "gpt-4", result.ModelID)
				assert.Equal(t, "You are a helpful assistant.", result.Instructions)
				assert.True(t, result.Stream)

				textInput, ok := result.Input.(gai.TextInput)
				require.True(t, ok, "Expected TextInput, got %T", result.Input)
				assert.Equal(t, "What's the weather like?", textInput.Text)
			},
		},
		{
			name: "multiple system messages",
			req: openai.ChatCompletionNewParams{
				Model: "gpt-4",
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.SystemMessage("You are a helpful assistant."),
					openai.DeveloperMessage("Respond concisely."),
					openai.UserMessage("Tell me about cats"),
				},
			},
			isStream: false,
			want: func(t *testing.T, result gai.ResponseRequest) {
				assert.Equal(t, "gpt-4", result.ModelID)
				assert.Equal(t, "You are a helpful assistant.\n\nRespond concisely.", result.Instructions)

				textInput, ok := result.Input.(gai.TextInput)
				require.True(t, ok, "Expected TextInput, got %T", result.Input)
				assert.Equal(t, "Tell me about cats", textInput.Text)
			},
		},
		{
			name: "conversation with assistant message",
			req: openai.ChatCompletionNewParams{
				Model: "gpt-3.5-turbo",
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.SystemMessage("You are helpful."),
					openai.UserMessage("Hello"),
					openai.AssistantMessage("Hi there! How can I help?"),
					openai.UserMessage("Tell me a joke"),
				},
			},
			isStream: false,
			want: func(t *testing.T, result gai.ResponseRequest) {
				assert.Equal(t, "gpt-3.5-turbo", result.ModelID)
				assert.Equal(t, "You are helpful.", result.Instructions)

				multiInput, ok := result.Input.(gai.MultiInput)
				require.True(t, ok, "Expected MultiInput, got %T", result.Input)
				require.Len(t, multiInput.Items, 3)

				// First item should be first user message
				textItem1, ok := multiInput.Items[0].(gai.TextInputItem)
				require.True(t, ok)
				assert.Equal(t, "Hello", textItem1.Text)

				// Second item should be assistant message
				textItem2, ok := multiInput.Items[1].(gai.TextInputItem)
				require.True(t, ok)
				assert.Equal(t, "Assistant: Hi there! How can I help?", textItem2.Text)

				// Third item should be second user message
				textItem3, ok := multiInput.Items[2].(gai.TextInputItem)
				require.True(t, ok)
				assert.Equal(t, "Tell me a joke", textItem3.Text)
			},
		},
		{
			name: "multi-modal user message",
			req: openai.ChatCompletionNewParams{
				Model: "gpt-4-vision",
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
						openai.TextContentPart("What's in this image?"),
						openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
							URL: "https://example.com/image.jpg",
						}),
					}),
				},
			},
			isStream: false,
			want: func(t *testing.T, result gai.ResponseRequest) {
				assert.Equal(t, "gpt-4-vision", result.ModelID)

				multiInput, ok := result.Input.(gai.MultiInput)
				require.True(t, ok, "Expected MultiInput, got %T", result.Input)
				require.Len(t, multiInput.Items, 2)

				// First item should be text
				textItem, ok := multiInput.Items[0].(gai.TextInputItem)
				require.True(t, ok)
				assert.Equal(t, "What's in this image?", textItem.Text)

				// Second item should be image
				imageItem, ok := multiInput.Items[1].(gai.ImageInputItem)
				require.True(t, ok)
				assert.Equal(t, "https://example.com/image.jpg", imageItem.URL)
			},
		},
		{
			name: "with temperature and max tokens",
			req: openai.ChatCompletionNewParams{
				Model:               "gpt-3.5-turbo",
				Temperature:         param.NewOpt(0.7),
				TopP:                param.NewOpt(0.9),
				MaxCompletionTokens: param.NewOpt(int64(100)),
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage("Test"),
				},
			},
			isStream: false,
			want: func(t *testing.T, result gai.ResponseRequest) {
				assert.Equal(t, "gpt-3.5-turbo", result.ModelID)
				assert.Equal(t, float32(0.7), result.Temperature)
				assert.Equal(t, float32(0.9), result.TopP)
				assert.Equal(t, 100, result.MaxOutputTokens)
			},
		},
		{
			name: "with legacy max_tokens",
			req: openai.ChatCompletionNewParams{
				Model:     "gpt-3.5-turbo",
				MaxTokens: param.NewOpt(int64(150)),
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage("Test"),
				},
			},
			isStream: false,
			want: func(t *testing.T, result gai.ResponseRequest) {
				assert.Equal(t, 150, result.MaxOutputTokens)
			},
		},
		{
			name: "max_completion_tokens takes precedence over max_tokens",
			req: openai.ChatCompletionNewParams{
				Model:               "gpt-3.5-turbo",
				MaxTokens:           param.NewOpt(int64(150)),
				MaxCompletionTokens: param.NewOpt(int64(200)),
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage("Test"),
				},
			},
			isStream: false,
			want: func(t *testing.T, result gai.ResponseRequest) {
				assert.Equal(t, 200, result.MaxOutputTokens)
			},
		},
		{
			name: "empty messages",
			req: openai.ChatCompletionNewParams{
				Model:    "gpt-3.5-turbo",
				Messages: []openai.ChatCompletionMessageParamUnion{},
			},
			isStream: false,
			want: func(t *testing.T, result gai.ResponseRequest) {
				assert.Equal(t, "gpt-3.5-turbo", result.ModelID)
				assert.Equal(t, "", result.Instructions)

				textInput, ok := result.Input.(gai.TextInput)
				require.True(t, ok, "Expected TextInput, got %T", result.Input)
				assert.Equal(t, "", textInput.Text)
			},
		},
		{
			name: "tool message",
			req: openai.ChatCompletionNewParams{
				Model: "gpt-3.5-turbo",
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage("What's the weather?"),
					openai.ToolMessage("The weather is sunny", "tool-call-123"),
				},
			},
			isStream: false,
			want: func(t *testing.T, result gai.ResponseRequest) {
				multiInput, ok := result.Input.(gai.MultiInput)
				require.True(t, ok, "Expected MultiInput, got %T", result.Input)
				require.Len(t, multiInput.Items, 2)

				// First item should be user message
				textItem1, ok := multiInput.Items[0].(gai.TextInputItem)
				require.True(t, ok)
				assert.Equal(t, "What's the weather?", textItem1.Text)

				// Second item should be tool response
				textItem2, ok := multiInput.Items[1].(gai.TextInputItem)
				require.True(t, ok)
				assert.Equal(t, "Tool response: The weather is sunny", textItem2.Text)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.convertToGaiRequest(tt.req, tt.isStream)
			tt.want(t, result)
		})
	}
}

func TestOpenAIProvider_extractTextFromMessage(t *testing.T) {
	provider := NewOpenAIProvider()

	tests := []struct {
		name string
		msg  openai.ChatCompletionMessageParamUnion
		want string
	}{
		{
			name: "simple user message",
			msg:  openai.UserMessage("Hello world"),
			want: "Hello world",
		},
		{
			name: "simple system message",
			msg:  openai.SystemMessage("You are helpful"),
			want: "You are helpful",
		},
		{
			name: "simple assistant message",
			msg:  openai.AssistantMessage("Hi there!"),
			want: "Hi there!",
		},
		{
			name: "multi-part user message",
			msg: openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.TextContentPart("Hello"),
				openai.TextContentPart("world"),
			}),
			want: "Hello world",
		},
		{
			name: "tool message",
			msg:  openai.ToolMessage("Tool result", "tool-123"),
			want: "Tool result",
		},
		{
			name: "developer message",
			msg:  openai.DeveloperMessage("Debug info"),
			want: "Debug info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.extractTextFromMessage(tt.msg)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOpenAIProvider_extractInputItemsFromMessage(t *testing.T) {
	provider := NewOpenAIProvider()

	tests := []struct {
		name string
		msg  openai.ChatCompletionMessageParamUnion
		want []gai.InputItem
	}{
		{
			name: "simple user text message",
			msg:  openai.UserMessage("Hello world"),
			want: []gai.InputItem{
				gai.TextInputItem{Text: "Hello world"},
			},
		},
		{
			name: "multi-modal user message",
			msg: openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.TextContentPart("What's in this image?"),
				openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL: "https://example.com/image.jpg",
				}),
			}),
			want: []gai.InputItem{
				gai.TextInputItem{Text: "What's in this image?"},
				gai.ImageInputItem{URL: "https://example.com/image.jpg"},
			},
		},
		{
			name: "non-user message falls back to text extraction",
			msg:  openai.SystemMessage("You are helpful"),
			want: []gai.InputItem{
				gai.TextInputItem{Text: "You are helpful"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.extractInputItemsFromMessage(tt.msg)
			assert.Equal(t, tt.want, result)
		})
	}
}
