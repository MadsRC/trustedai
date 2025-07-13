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
	"codeberg.org/MadsRC/llmgw/internal/api/dataplane/interfaces"
	"codeberg.org/gai-org/gai"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

type OpenAIProvider struct {
	options         *dataplane.ProviderOptions
	llmClient       dataplane.LLMClient
	usageMiddleware interfaces.UsageMiddleware
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

func (p *OpenAIProvider) SetUsageMiddleware(middleware interfaces.UsageMiddleware) {
	p.usageMiddleware = middleware
}

func (p *OpenAIProvider) SetupRoutes(mux *http.ServeMux, baseAuth func(http.Handler) http.Handler) {
	if baseAuth != nil {
		mux.Handle("POST /openai/v1/chat/completions", baseAuth(http.HandlerFunc(p.handleChatCompletions)))
		mux.Handle("GET /openai/v1/models", baseAuth(http.HandlerFunc(p.handleListModels)))
		mux.Handle("POST /openai/v1/responses", baseAuth(http.HandlerFunc(p.handleCreateResponse)))
	} else {
		mux.HandleFunc("POST /openai/v1/chat/completions", p.handleChatCompletions)
		mux.HandleFunc("GET /openai/v1/models", p.handleListModels)
		mux.HandleFunc("POST /openai/v1/responses", p.handleCreateResponse)
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

	// Track request timing
	startTime := time.Now()

	gaiResp, err := p.llmClient.Generate(r.Context(), gaiReq)
	duration := time.Since(startTime)

	if err != nil {
		// Track failed request
		if p.usageMiddleware != nil {
			p.usageMiddleware.CreateEventFromGAIResponse(
				r.Context(),
				string(req.Model),
				nil, // No usage data on error
				"failed",
				duration,
			)
		}

		p.options.Logger.Error("Failed to generate response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Track successful request with token usage
	if p.usageMiddleware != nil {
		var usage *interfaces.TokenUsage
		if gaiResp.Usage != nil {
			usage = &interfaces.TokenUsage{
				PromptTokens:     gaiResp.Usage.PromptTokens,
				CompletionTokens: gaiResp.Usage.CompletionTokens,
				TotalTokens:      gaiResp.Usage.TotalTokens,
			}
		}

		p.usageMiddleware.CreateEventFromGAIResponse(
			r.Context(),
			string(req.Model),
			usage,
			"success",
			duration,
		)
	}

	// Convert gai response back to OpenAI response format
	openaiResp := p.convertFromGaiResponse(gaiResp, string(req.Model))

	if err := json.NewEncoder(w).Encode(openaiResp); err != nil {
		p.options.Logger.Error("Failed to encode chat completions response", "error", err)
	}
}

func (p *OpenAIProvider) handleListModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get models from ModelRouter
	if p.options.ModelRouter == nil {
		p.options.Logger.Error("ModelRouter not configured")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	models, err := p.options.ModelRouter.ListModels(r.Context())
	if err != nil {
		p.options.Logger.Error("Failed to list models", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert GAI models to OpenAI API format
	var openaiModels []map[string]any
	for _, model := range models {
		openaiModel := map[string]any{
			"id":       model.ID,
			"object":   "model",
			"owned_by": model.Provider,
		}

		// Use created timestamp from metadata if available, otherwise use current time
		if createdAt, ok := model.Metadata["created_at"].(string); ok {
			// Try to parse the timestamp and convert to Unix timestamp
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				openaiModel["created"] = t.Unix()
			} else {
				openaiModel["created"] = time.Now().Unix()
			}
		} else {
			openaiModel["created"] = time.Now().Unix()
		}

		openaiModels = append(openaiModels, openaiModel)
	}

	response := map[string]any{
		"object": "list",
		"data":   openaiModels,
	}

	w.WriteHeader(http.StatusOK)
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

func (p *OpenAIProvider) handleCreateResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CreateResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		p.options.Logger.Error("Failed to decode create response request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	p.options.Logger.Info("Received create response request",
		"model", req.Model,
		"stream", req.Stream != nil && *req.Stream)

	gaiReq := p.convertCreateResponseToGaiRequest(req)

	isStream := req.Stream != nil && *req.Stream

	if isStream {
		p.handleResponseStreaming(w, r.Context(), gaiReq, req.Model)
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

	response := p.convertGaiResponseToResponse(gaiResp, req.Model)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		p.options.Logger.Error("Failed to encode response", "error", err)
	}
}

func (p *OpenAIProvider) convertCreateResponseToGaiRequest(req CreateResponseRequest) gai.GenerateRequest {
	var input gai.Input
	var instructions string

	if req.Instructions != "" {
		instructions = req.Instructions
	}

	if req.Input != "" {
		input = gai.TextInput{Text: req.Input}
	} else if len(req.InputItems) > 0 {
		var messages []gai.Message
		for _, item := range req.InputItems {
			if item.Type == InputItemTypeMessage {
				var msgContent map[string]any
				if err := json.Unmarshal(item.Content, &msgContent); err == nil {
					if role, ok := msgContent["role"].(string); ok {
						if content, ok := msgContent["content"].(string); ok {
							var gaiRole gai.Role
							switch role {
							case "user":
								gaiRole = gai.RoleUser
							case "assistant":
								gaiRole = gai.RoleAssistant
							case "system", "developer":
								if instructions != "" {
									instructions += "\n\n" + content
								} else {
									instructions = content
								}
								continue
							default:
								gaiRole = gai.RoleUser
							}
							messages = append(messages, gai.Message{
								Role:    gaiRole,
								Content: gai.TextInput{Text: content},
							})
						}
					}
				}
			}
		}
		if len(messages) > 0 {
			input = gai.Conversation{Messages: messages}
		} else {
			input = gai.TextInput{Text: ""}
		}
	} else {
		input = gai.TextInput{Text: ""}
	}

	gaiReq := gai.GenerateRequest{
		ModelID:      req.Model,
		Instructions: instructions,
		Input:        input,
		Stream:       req.Stream != nil && *req.Stream,
	}

	if req.Temperature != nil {
		gaiReq.Temperature = float32(*req.Temperature)
	}
	if req.TopP != nil {
		gaiReq.TopP = float32(*req.TopP)
	}
	if req.MaxOutputTokens != nil {
		gaiReq.MaxOutputTokens = *req.MaxOutputTokens
	}

	return gaiReq
}

func (p *OpenAIProvider) convertGaiResponseToResponse(gaiResp *gai.Response, modelID string) *Response {
	response := NewResponse(gaiResp.ID)
	response.Model = modelID
	response.SetCompleted()

	for _, output := range gaiResp.Output {
		if textOutput, ok := output.(gai.TextOutput); ok {
			outputItemContent := map[string]any{
				"role":    "assistant",
				"content": textOutput.Text,
			}
			contentBytes, _ := json.Marshal(outputItemContent)
			response.AddOutputItem(OutputItem{
				Type:    OutputItemTypeMessage,
				Content: contentBytes,
			})
			if response.OutputText == nil {
				outputText := textOutput.Text
				response.OutputText = &outputText
			}
		}
	}

	if gaiResp.Usage != nil {
		response.Usage = &ResponseUsage{
			InputTokens:  int(gaiResp.Usage.PromptTokens),
			OutputTokens: int(gaiResp.Usage.CompletionTokens),
			TotalTokens:  int(gaiResp.Usage.TotalTokens),
			InputTokensDetails: InputTokensDetails{
				CachedTokens: 0,
			},
			OutputTokensDetails: OutputTokensDetails{
				ReasoningTokens: 0,
			},
		}
	}

	return response
}

func (p *OpenAIProvider) handleResponseStreaming(w http.ResponseWriter, ctx context.Context, gaiReq gai.GenerateRequest, modelID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		p.options.Logger.Error("Streaming not supported by response writer")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	if p.llmClient == nil {
		p.options.Logger.Error("LLM client not set")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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

	responseID := gaiReq.ModelID + "_" + fmt.Sprintf("%d", time.Now().UnixNano())

	for {
		chunk, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			p.options.Logger.Error("Error reading stream chunk", "error", err)
			break
		}

		streamEvent := p.convertChunkToResponseStreamEvent(chunk, responseID)
		data, err := json.Marshal(streamEvent)
		if err != nil {
			p.options.Logger.Error("Failed to marshal stream event", "error", err)
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

	if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
		p.options.Logger.Error("Failed to write final SSE message", "error", err)
	}
	flusher.Flush()
}

func (p *OpenAIProvider) convertChunkToResponseStreamEvent(chunk *gai.ResponseChunk, responseID string) ResponseStreamEvent {
	eventData := map[string]any{
		"id":      responseID,
		"object":  "response.stream_event",
		"created": time.Now().Unix(),
	}

	if chunk.Delta.Text != "" {
		eventData["type"] = "response.text.delta"
		eventData["delta"] = map[string]any{
			"text": chunk.Delta.Text,
		}
	} else if chunk.Finished {
		eventData["type"] = "response.completed"
		if chunk.Usage != nil {
			eventData["usage"] = map[string]any{
				"input_tokens":  chunk.Usage.PromptTokens,
				"output_tokens": chunk.Usage.CompletionTokens,
				"total_tokens":  chunk.Usage.TotalTokens,
			}
		}
	} else {
		eventData["type"] = "response.created"
	}

	eventDataBytes, _ := json.Marshal(eventData)
	return ResponseStreamEvent{
		Type: eventData["type"].(string),
		Data: eventDataBytes,
	}
}

func (p *OpenAIProvider) Shutdown(ctx context.Context) error {
	p.options.Logger.Info("Shutting down OpenAI provider")
	return nil
}
