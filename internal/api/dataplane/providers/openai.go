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
	"time"

	"codeberg.org/MadsRC/llmgw/internal/api/dataplane"
	"codeberg.org/gai-org/gai"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

type OpenAIProvider struct {
	options   *dataplane.ProviderOptions
	llmClient dataplane.LLMClient
}

func NewOpenAIProvider(options ...dataplane.ProviderOption) *OpenAIProvider {
	opts := &dataplane.ProviderOptions{
		Logger: slog.Default(),
	}

	for _, option := range options {
		option.Apply(opts)
	}

	return &OpenAIProvider{
		options: opts,
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) SetLLMClient(client dataplane.LLMClient) {
	p.llmClient = client
}

func (p *OpenAIProvider) SetupRoutes(mux *http.ServeMux, baseAuth func(http.Handler) http.Handler) {
	if baseAuth != nil {
		mux.Handle("POST /openai/chat/completions", baseAuth(http.HandlerFunc(p.handleChatCompletions)))
		mux.Handle("GET /openai/models", baseAuth(http.HandlerFunc(p.handleListModels)))
	} else {
		mux.HandleFunc("POST /openai/chat/completions", p.handleChatCompletions)
		mux.HandleFunc("GET /openai/models", p.handleListModels)
	}
}

func (p *OpenAIProvider) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse raw JSON to detect streaming
	var rawReq map[string]any
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		p.options.Logger.Error("Failed to decode chat completion request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check for streaming
	isStream := false
	if stream, ok := rawReq["stream"].(bool); ok && stream {
		isStream = true
	}

	// Convert raw JSON back to structured request
	reqBytes, _ := json.Marshal(rawReq)
	var req openai.ChatCompletionNewParams
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		p.options.Logger.Error("Failed to parse chat completion request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	p.options.Logger.Info("Received chat completion request",
		"model", req.Model,
		"messages_count", len(req.Messages),
		"stream", isStream)

	// Convert OpenAI request to gai GenerateRequest
	gaiReq := p.convertToGaiRequest(req, isStream)

	if isStream {
		p.handleStreamingResponse(w, r.Context(), gaiReq, string(req.Model))
		return
	}

	// Generate response using LLM client
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

	// Convert gai response back to OpenAI response format
	openaiResp := p.convertFromGaiResponse(gaiResp, string(req.Model))

	if err := json.NewEncoder(w).Encode(openaiResp); err != nil {
		p.options.Logger.Error("Failed to encode chat completions response", "error", err)
	}
}

func (p *OpenAIProvider) handleListModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]any{
		"object": "list",
		"data": []map[string]any{
			{
				"id":       "gpt-3.5-turbo",
				"object":   "model",
				"created":  time.Now().Unix(),
				"owned_by": "openai",
			},
			{
				"id":       "gpt-4",
				"object":   "model",
				"created":  time.Now().Unix(),
				"owned_by": "openai",
			},
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		p.options.Logger.Error("Failed to encode models response", "error", err)
	}
}

func (p *OpenAIProvider) convertToGaiRequest(req openai.ChatCompletionNewParams, isStream bool) gai.GenerateRequest {
	// Convert OpenAI messages to gai Messages for Conversation input
	var messages []gai.Message
	var instructions string

	for _, msg := range req.Messages {
		role := p.extractRoleFromMessage(msg)

		switch role {
		case "system", "developer":
			// System/developer messages become instructions
			if textContent := p.extractTextFromMessage(msg); textContent != "" {
				if instructions != "" {
					instructions += "\n\n" + textContent
				} else {
					instructions = textContent
				}
			}
		case "user":
			// Convert user message to gai Message(s)
			// For multimodal content, create separate messages for each part
			userMessages := p.extractMessagesFromUserMessage(msg)
			messages = append(messages, userMessages...)
		case "assistant":
			// Convert assistant message to gai Message
			if textContent := p.extractTextFromMessage(msg); textContent != "" {
				messages = append(messages, gai.Message{
					Role:    gai.RoleAssistant,
					Content: gai.TextInput{Text: textContent},
				})
			}
		case "tool":
			// Convert tool response to gai Message
			if textContent := p.extractTextFromMessage(msg); textContent != "" {
				messages = append(messages, gai.Message{
					Role:    gai.RoleTool,
					Content: gai.TextInput{Text: textContent},
				})
			}
		}
	}

	// Create conversation input or fallback to text input
	var input gai.Input
	if len(messages) > 0 {
		input = gai.Conversation{Messages: messages}
	} else {
		input = gai.TextInput{Text: ""}
	}

	gaiReq := gai.GenerateRequest{
		ModelID:      string(req.Model),
		Instructions: instructions,
		Input:        input,
		Stream:       isStream,
	}

	// Set optional parameters if present
	if !param.IsOmitted(req.Temperature) {
		gaiReq.Temperature = float32(req.Temperature.Value)
	}
	if !param.IsOmitted(req.TopP) {
		gaiReq.TopP = float32(req.TopP.Value)
	}
	if !param.IsOmitted(req.MaxCompletionTokens) {
		gaiReq.MaxOutputTokens = int(req.MaxCompletionTokens.Value)
	} else if !param.IsOmitted(req.MaxTokens) {
		gaiReq.MaxOutputTokens = int(req.MaxTokens.Value)
	}

	// Handle tools if present
	if len(req.Tools) > 0 {
		gaiReq.Tools = p.convertToolsToGai(req.Tools)
	}

	// Handle tool choice if present
	if !param.IsOmitted(req.ToolChoice) {
		gaiReq.ToolChoice = p.convertToolChoiceToGai(req.ToolChoice)
	}

	return gaiReq
}

func (p *OpenAIProvider) extractRoleFromMessage(msg openai.ChatCompletionMessageParamUnion) string {
	// Check which field is set to determine the role
	if msg.OfSystem != nil {
		return "system"
	}
	if msg.OfUser != nil {
		return "user"
	}
	if msg.OfAssistant != nil {
		return "assistant"
	}
	if msg.OfTool != nil {
		return "tool"
	}
	if msg.OfFunction != nil {
		return "function"
	}
	if msg.OfDeveloper != nil {
		return "developer"
	}
	return ""
}

func (p *OpenAIProvider) extractTextFromMessage(msg openai.ChatCompletionMessageParamUnion) string {
	// Handle different message types
	if msg.OfSystem != nil {
		return p.extractTextFromSystemMessage(msg.OfSystem)
	}
	if msg.OfUser != nil {
		return p.extractTextFromUserMessage(msg.OfUser)
	}
	if msg.OfAssistant != nil {
		return p.extractTextFromAssistantMessage(msg.OfAssistant)
	}
	if msg.OfTool != nil {
		return p.extractTextFromToolMessage(msg.OfTool)
	}
	if msg.OfFunction != nil {
		return p.extractTextFromFunctionMessage(msg.OfFunction)
	}
	if msg.OfDeveloper != nil {
		return p.extractTextFromDeveloperMessage(msg.OfDeveloper)
	}
	return ""
}

func (p *OpenAIProvider) extractTextFromSystemMessage(msg *openai.ChatCompletionSystemMessageParam) string {
	if msg == nil {
		return ""
	}
	// Handle string content
	if !param.IsOmitted(msg.Content.OfString) {
		return msg.Content.OfString.Value
	}
	// Handle array of content parts - for system messages these are ChatCompletionContentPartTextParam
	if msg.Content.OfArrayOfContentParts != nil {
		var textParts []string
		for _, part := range msg.Content.OfArrayOfContentParts {
			textParts = append(textParts, part.Text)
		}
		return strings.Join(textParts, " ")
	}
	return ""
}

func (p *OpenAIProvider) extractTextFromUserMessage(msg *openai.ChatCompletionUserMessageParam) string {
	if msg == nil {
		return ""
	}
	// Handle string content
	if !param.IsOmitted(msg.Content.OfString) {
		return msg.Content.OfString.Value
	}
	// Handle array of content parts - for user messages these are ChatCompletionContentPartUnionParam
	if msg.Content.OfArrayOfContentParts != nil {
		var textParts []string
		for _, part := range msg.Content.OfArrayOfContentParts {
			if !param.IsOmitted(part.OfText) {
				textParts = append(textParts, part.OfText.Text)
			}
		}
		return strings.Join(textParts, " ")
	}
	return ""
}

func (p *OpenAIProvider) extractMessagesFromUserMessage(msg openai.ChatCompletionMessageParamUnion) []gai.Message {
	if msg.OfUser == nil {
		// Fallback for non-user messages - treat as single text message
		if textContent := p.extractTextFromMessage(msg); textContent != "" {
			return []gai.Message{{
				Role:    gai.RoleUser,
				Content: gai.TextInput{Text: textContent},
			}}
		}
		return []gai.Message{}
	}

	userMsg := msg.OfUser

	// Handle string content
	if !param.IsOmitted(userMsg.Content.OfString) {
		return []gai.Message{{
			Role:    gai.RoleUser,
			Content: gai.TextInput{Text: userMsg.Content.OfString.Value},
		}}
	}

	// Handle array of content parts - create separate messages for each part
	if userMsg.Content.OfArrayOfContentParts != nil {
		var messages []gai.Message
		for _, part := range userMsg.Content.OfArrayOfContentParts {
			if !param.IsOmitted(part.OfText) {
				messages = append(messages, gai.Message{
					Role:    gai.RoleUser,
					Content: gai.TextInput{Text: part.OfText.Text},
				})
			} else if !param.IsOmitted(part.OfImageURL) {
				messages = append(messages, gai.Message{
					Role:    gai.RoleUser,
					Content: gai.ImageInput{URL: part.OfImageURL.ImageURL.URL},
				})
			}
		}
		return messages
	}

	return []gai.Message{}
}

func (p *OpenAIProvider) extractTextFromAssistantMessage(msg *openai.ChatCompletionAssistantMessageParam) string {
	if msg == nil {
		return ""
	}
	// Handle string content
	if !param.IsOmitted(msg.Content.OfString) {
		return msg.Content.OfString.Value
	}
	// Handle array of content parts
	if msg.Content.OfArrayOfContentParts != nil {
		var textParts []string
		for _, part := range msg.Content.OfArrayOfContentParts {
			if text := part.GetText(); text != nil {
				textParts = append(textParts, *text)
			}
		}
		return strings.Join(textParts, " ")
	}
	return ""
}

func (p *OpenAIProvider) extractTextFromToolMessage(msg *openai.ChatCompletionToolMessageParam) string {
	if msg == nil {
		return ""
	}
	// Handle string content
	if !param.IsOmitted(msg.Content.OfString) {
		return msg.Content.OfString.Value
	}
	// Handle array of content parts - for tool messages these are ChatCompletionContentPartTextParam
	if msg.Content.OfArrayOfContentParts != nil {
		var textParts []string
		for _, part := range msg.Content.OfArrayOfContentParts {
			textParts = append(textParts, part.Text)
		}
		return strings.Join(textParts, " ")
	}
	return ""
}

// Keep function message support for backwards compatibility
// Will be removed when no longer available in upstream library
//
//nolint:staticcheck // Suppress deprecation warning for backwards compatibility
func (p *OpenAIProvider) extractTextFromFunctionMessage(msg *openai.ChatCompletionFunctionMessageParam) string {
	if msg == nil {
		return ""
	}
	if !param.IsOmitted(msg.Content) {
		return msg.Content.Value
	}
	return ""
}

func (p *OpenAIProvider) extractTextFromDeveloperMessage(msg *openai.ChatCompletionDeveloperMessageParam) string {
	if msg == nil {
		return ""
	}
	// Handle string content
	if !param.IsOmitted(msg.Content.OfString) {
		return msg.Content.OfString.Value
	}
	// Handle array of content parts - for developer messages these are ChatCompletionContentPartTextParam
	if msg.Content.OfArrayOfContentParts != nil {
		var textParts []string
		for _, part := range msg.Content.OfArrayOfContentParts {
			textParts = append(textParts, part.Text)
		}
		return strings.Join(textParts, " ")
	}
	return ""
}

func (p *OpenAIProvider) convertToolsToGai(openaiTools []openai.ChatCompletionToolParam) []gai.Tool {
	gaiTools := make([]gai.Tool, len(openaiTools))
	for i, tool := range openaiTools {
		gaiTools[i] = gai.Tool{
			Type: string(tool.Type),
			Function: gai.Function{
				Name:        tool.Function.Name,
				Description: tool.Function.Description.Value,
				Parameters:  tool.Function.Parameters,
			},
		}
	}
	return gaiTools
}

func (p *OpenAIProvider) convertToolChoiceToGai(openaiToolChoice openai.ChatCompletionToolChoiceOptionUnionParam) *gai.ToolChoice {
	// Handle string tool choice options
	if !param.IsOmitted(openaiToolChoice.OfAuto) {
		return &gai.ToolChoice{
			Type: openaiToolChoice.OfAuto.Value,
		}
	}

	// Handle named function tool choice
	if !param.IsOmitted(openaiToolChoice.OfChatCompletionNamedToolChoice) {
		namedChoice := openaiToolChoice.OfChatCompletionNamedToolChoice
		return &gai.ToolChoice{
			Type: string(namedChoice.Type),
			Function: &gai.ToolChoiceFunction{
				Name: namedChoice.Function.Name,
			},
		}
	}

	return nil
}

func (p *OpenAIProvider) convertFromGaiResponse(gaiResp *gai.Response, modelID string) map[string]any {
	// Extract text content from gai response
	var content string
	for _, output := range gaiResp.Output {
		if textOutput, ok := output.(gai.TextOutput); ok {
			content = textOutput.Text
			break
		}
	}

	// Convert finish reason - default to "stop" if not available
	finishReason := "stop"
	// Note: Response struct doesn't have FinishReason field based on godoc
	// We'll use a default value for now

	// Convert to OpenAI response format
	response := map[string]any{
		"id":      "chatcmpl-" + gaiResp.ID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   modelID,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": finishReason,
			},
		},
	}

	// Add usage information if available
	if gaiResp.Usage != nil {
		response["usage"] = map[string]any{
			"prompt_tokens":     gaiResp.Usage.PromptTokens,
			"completion_tokens": gaiResp.Usage.CompletionTokens,
			"total_tokens":      gaiResp.Usage.TotalTokens,
		}
	}

	return response
}

func (p *OpenAIProvider) handleStreamingResponse(w http.ResponseWriter, ctx context.Context, gaiReq gai.GenerateRequest, modelID string) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		p.options.Logger.Error("Streaming not supported by response writer")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Start streaming from LLM client
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

	// Process streaming chunks
	for {
		chunk, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			p.options.Logger.Error("Error reading stream chunk", "error", err)
			break
		}

		// Convert chunk to OpenAI SSE format
		openaiChunk := p.convertChunkToOpenAI(chunk, modelID)
		data, err := json.Marshal(openaiChunk)
		if err != nil {
			p.options.Logger.Error("Failed to marshal chunk", "error", err)
			continue
		}

		// Send SSE data
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			p.options.Logger.Error("Failed to write SSE data", "error", err)
			break
		}
		flusher.Flush()

		if chunk.Finished {
			break
		}
	}

	// Send final [DONE] message
	if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
		p.options.Logger.Error("Failed to write final SSE message", "error", err)
	}
	flusher.Flush()
}

func (p *OpenAIProvider) convertChunkToOpenAI(chunk *gai.ResponseChunk, modelID string) map[string]any {
	response := map[string]any{
		"id":      "chatcmpl-" + chunk.ID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   modelID,
		"choices": []map[string]any{
			{
				"index": 0,
				"delta": map[string]any{
					"content": chunk.Delta.Text,
				},
				"finish_reason": nil,
			},
		},
	}

	// Set finish_reason for the final chunk
	if chunk.Finished {
		choices := response["choices"].([]map[string]any)
		choices[0]["finish_reason"] = "stop"
		choices[0]["delta"] = map[string]any{} // Empty delta for final chunk
	}

	// Add usage information if available
	if chunk.Usage != nil {
		response["usage"] = map[string]any{
			"prompt_tokens":     chunk.Usage.PromptTokens,
			"completion_tokens": chunk.Usage.CompletionTokens,
			"total_tokens":      chunk.Usage.TotalTokens,
		}
	}

	return response
}

func (p *OpenAIProvider) Shutdown(ctx context.Context) error {
	p.options.Logger.Info("Shutting down OpenAI provider")
	return nil
}
