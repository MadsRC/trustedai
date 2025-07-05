// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"codeberg.org/MadsRC/llmgw/internal/api/dataplane"
	"codeberg.org/gai-org/gai"
)

type AnthropicProvider struct {
	options   *dataplane.ProviderOptions
	llmClient dataplane.LLMClient
}

type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type AnthropicContentBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Source struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	} `json:"source,omitempty"`
}

type AnthropicRequest struct {
	Model         string             `json:"model"`
	MaxTokens     int                `json:"max_tokens"`
	Messages      []AnthropicMessage `json:"messages"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	TopK          *int               `json:"top_k,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	System        string             `json:"system,omitempty"`
	Tools         []AnthropicTool    `json:"tools,omitempty"`
	ToolChoice    interface{}        `json:"tool_choice,omitempty"`
}

type AnthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type AnthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []AnthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason"`
	StopSequence *string                 `json:"stop_sequence"`
	Usage        *AnthropicUsage         `json:"usage"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicStreamEvent struct {
	Type  string          `json:"type"`
	Index int             `json:"index,omitempty"`
	Delta interface{}     `json:"delta,omitempty"`
	Usage *AnthropicUsage `json:"usage,omitempty"`
}

type AnthropicStreamDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewAnthropicProvider(options ...dataplane.ProviderOption) *AnthropicProvider {
	opts := &dataplane.ProviderOptions{
		Logger: slog.Default(),
	}

	for _, option := range options {
		option.Apply(opts)
	}

	return &AnthropicProvider{
		options: opts,
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) SetLLMClient(client dataplane.LLMClient) {
	p.llmClient = client
}

func (p *AnthropicProvider) SetupRoutes(mux *http.ServeMux, baseAuth func(http.Handler) http.Handler) {
	if baseAuth != nil {
		mux.Handle("POST /anthropic/v1/messages", baseAuth(http.HandlerFunc(p.handleMessages)))
	} else {
		mux.HandleFunc("POST /anthropic/v1/messages", p.handleMessages)
	}
}

func (p *AnthropicProvider) handleMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req AnthropicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		p.options.Logger.Error("Failed to decode messages request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	p.options.Logger.Info("Received messages request",
		"model", req.Model,
		"messages_count", len(req.Messages),
		"stream", req.Stream)

	gaiReq := p.convertToGaiRequest(req)

	if req.Stream {
		p.handleStreamingResponse(w, r.Context(), gaiReq, req.Model)
		return
	}

	if p.llmClient == nil {
		p.options.Logger.Error("LLM client not set")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	gaiResp, err := p.llmClient.Generate(r.Context(), gaiReq)
	if err != nil {
		p.options.Logger.Error("Failed to generate response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	anthropicResp := p.convertFromGaiResponse(gaiResp, req.Model)

	if err := json.NewEncoder(w).Encode(anthropicResp); err != nil {
		p.options.Logger.Error("Failed to encode messages response", "error", err)
	}
}

func (p *AnthropicProvider) convertToGaiRequest(req AnthropicRequest) gai.GenerateRequest {
	var messages []gai.Message
	var instructions string

	if req.System != "" {
		instructions = req.System
	}

	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			userMessages := p.extractMessagesFromAnthropicMessage(msg)
			messages = append(messages, userMessages...)
		case "assistant":
			if textContent := p.extractTextFromAnthropicMessage(msg); textContent != "" {
				messages = append(messages, gai.Message{
					Role:    gai.RoleAssistant,
					Content: gai.TextInput{Text: textContent},
				})
			}
		}
	}

	var input gai.Input
	if len(messages) > 0 {
		input = gai.Conversation{Messages: messages}
	} else {
		input = gai.TextInput{Text: ""}
	}

	gaiReq := gai.GenerateRequest{
		ModelID:      req.Model,
		Instructions: instructions,
		Input:        input,
		Stream:       req.Stream,
	}

	if req.Temperature != nil {
		gaiReq.Temperature = float32(*req.Temperature)
	}
	if req.TopP != nil {
		gaiReq.TopP = float32(*req.TopP)
	}
	if req.MaxTokens > 0 {
		gaiReq.MaxOutputTokens = req.MaxTokens
	}

	if len(req.Tools) > 0 {
		gaiReq.Tools = p.convertToolsToGai(req.Tools)
	}

	if req.ToolChoice != nil {
		gaiReq.ToolChoice = p.convertToolChoiceToGai(req.ToolChoice)
	}

	return gaiReq
}

func (p *AnthropicProvider) extractTextFromAnthropicMessage(msg AnthropicMessage) string {
	switch content := msg.Content.(type) {
	case string:
		return content
	case []interface{}:
		var textParts []string
		for _, part := range content {
			if partMap, ok := part.(map[string]interface{}); ok {
				if partType, ok := partMap["type"].(string); ok && partType == "text" {
					if text, ok := partMap["text"].(string); ok {
						textParts = append(textParts, text)
					}
				}
			}
		}
		return strings.Join(textParts, " ")
	}
	return ""
}

func (p *AnthropicProvider) extractMessagesFromAnthropicMessage(msg AnthropicMessage) []gai.Message {
	switch content := msg.Content.(type) {
	case string:
		return []gai.Message{{
			Role:    gai.RoleUser,
			Content: gai.TextInput{Text: content},
		}}
	case []interface{}:
		var messages []gai.Message
		for _, part := range content {
			if partMap, ok := part.(map[string]interface{}); ok {
				if partType, ok := partMap["type"].(string); ok {
					switch partType {
					case "text":
						if text, ok := partMap["text"].(string); ok {
							messages = append(messages, gai.Message{
								Role:    gai.RoleUser,
								Content: gai.TextInput{Text: text},
							})
						}
					case "image":
						if source, ok := partMap["source"].(map[string]interface{}); ok {
							if mediaType, ok := source["media_type"].(string); ok {
								if data, ok := source["data"].(string); ok {
									imageURL := fmt.Sprintf("data:%s;base64,%s", mediaType, data)
									messages = append(messages, gai.Message{
										Role:    gai.RoleUser,
										Content: gai.ImageInput{URL: imageURL},
									})
								}
							}
						}
					}
				}
			}
		}
		return messages
	}
	return []gai.Message{}
}

func (p *AnthropicProvider) convertToolsToGai(anthropicTools []AnthropicTool) []gai.Tool {
	gaiTools := make([]gai.Tool, len(anthropicTools))
	for i, tool := range anthropicTools {
		gaiTools[i] = gai.Tool{
			Type: "function",
			Function: gai.Function{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema.(map[string]any),
			},
		}
	}
	return gaiTools
}

func (p *AnthropicProvider) convertToolChoiceToGai(toolChoice interface{}) *gai.ToolChoice {
	switch tc := toolChoice.(type) {
	case string:
		return &gai.ToolChoice{Type: tc}
	case map[string]interface{}:
		if tcType, ok := tc["type"].(string); ok {
			choice := &gai.ToolChoice{Type: tcType}
			if tcType == "tool" {
				if name, ok := tc["name"].(string); ok {
					choice.Function = &gai.ToolChoiceFunction{Name: name}
				}
			}
			return choice
		}
	}
	return nil
}

func (p *AnthropicProvider) convertFromGaiResponse(gaiResp *gai.Response, modelID string) *AnthropicResponse {
	var content []AnthropicContentBlock
	for _, output := range gaiResp.Output {
		if textOutput, ok := output.(gai.TextOutput); ok {
			content = append(content, AnthropicContentBlock{
				Type: "text",
				Text: textOutput.Text,
			})
		}
	}

	response := &AnthropicResponse{
		ID:         gaiResp.ID,
		Type:       "message",
		Role:       "assistant",
		Content:    content,
		Model:      modelID,
		StopReason: "end_turn",
	}

	if gaiResp.Usage != nil {
		response.Usage = &AnthropicUsage{
			InputTokens:  gaiResp.Usage.PromptTokens,
			OutputTokens: gaiResp.Usage.CompletionTokens,
		}
	}

	return response
}

func (p *AnthropicProvider) handleStreamingResponse(w http.ResponseWriter, ctx context.Context, gaiReq gai.GenerateRequest, modelID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		p.options.Logger.Error("Streaming not supported by response writer")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	stream, err := p.llmClient.GenerateStream(ctx, gaiReq)
	if err != nil {
		p.options.Logger.Error("Failed to start streaming", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := stream.Close(); err != nil {
			p.options.Logger.Error("Failed to close stream", "error", err)
		}
	}()

	for {
		chunk, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			p.options.Logger.Error("Error reading stream chunk", "error", err)
			break
		}

		event := p.convertChunkToAnthropicEvent(chunk, modelID)
		data, err := json.Marshal(event)
		if err != nil {
			p.options.Logger.Error("Failed to marshal chunk", "error", err)
			continue
		}

		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			p.options.Logger.Error("Failed to write SSE data", "error", err)
			break
		}
		flusher.Flush()

		if chunk.Finished {
			break
		}
	}
}

func (p *AnthropicProvider) convertChunkToAnthropicEvent(chunk *gai.ResponseChunk, modelID string) AnthropicStreamEvent {
	if chunk.Finished {
		event := AnthropicStreamEvent{
			Type: "message_stop",
		}
		if chunk.Usage != nil {
			event.Usage = &AnthropicUsage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
			}
		}
		return event
	}

	return AnthropicStreamEvent{
		Type:  "content_block_delta",
		Index: 0,
		Delta: AnthropicStreamDelta{
			Type: "text_delta",
			Text: chunk.Delta.Text,
		},
	}
}

func (p *AnthropicProvider) Shutdown(ctx context.Context) error {
	p.options.Logger.Info("Shutting down Anthropic provider")
	return nil
}
