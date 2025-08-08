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

	"codeberg.org/gai-org/gai"
	"github.com/MadsRC/trustedai/internal/api/dataplane"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/responses"
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
		mux.Handle("POST /openai/v1/chat/completions", baseAuth(http.HandlerFunc(p.handleChatCompletions)))
		mux.Handle("POST /openai/v1/responses", baseAuth(http.HandlerFunc(p.handleResponses)))
		mux.Handle("GET /openai/v1/models", baseAuth(http.HandlerFunc(p.handleListModels)))
	} else {
		mux.HandleFunc("POST /openai/v1/chat/completions", p.handleChatCompletions)
		mux.HandleFunc("POST /openai/v1/responses", p.handleResponses)
		mux.HandleFunc("GET /openai/v1/models", p.handleListModels)
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

func (p *OpenAIProvider) handleResponses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse raw JSON to detect streaming
	var rawReq map[string]any
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		p.options.Logger.Error("Failed to decode responses request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check for streaming
	isStream := false
	if stream, ok := rawReq["stream"].(bool); ok && stream {
		isStream = true
	}

	// Preprocess input to handle shorthand message format (type field is optional per OpenAI API spec)
	p.preprocessResponseInput(rawReq)

	// Convert raw JSON back to structured request
	reqBytes, _ := json.Marshal(rawReq)
	var req responses.ResponseNewParams
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		p.options.Logger.Error("Failed to parse responses request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	p.options.Logger.Info("Received responses request",
		"model", req.Model,
		"stream", isStream)

	// Convert OpenAI request to gai GenerateRequest
	gaiReq := p.convertResponseToGaiRequest(req, isStream)

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

	// Convert gai response back to OpenAI responses format
	openaiResp := p.convertFromGaiResponseToResponsesFormat(gaiResp, string(req.Model))

	if err := json.NewEncoder(w).Encode(openaiResp); err != nil {
		p.options.Logger.Error("Failed to encode responses response", "error", err)
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

func (p *OpenAIProvider) convertResponseToGaiRequest(req responses.ResponseNewParams, isStream bool) gai.GenerateRequest {
	var input gai.Input
	var instructions string

	// Handle Instructions parameter
	if !param.IsOmitted(req.Instructions) {
		instructions = req.Instructions.Value
	}

	// Handle Input union type
	if !param.IsOmitted(req.Input.OfString) {
		// Simple string input
		input = gai.TextInput{Text: req.Input.OfString.Value}
	} else if req.Input.OfInputItemList != nil {
		// Complex input with array of items
		var messages []gai.Message

		for _, item := range req.Input.OfInputItemList {
			if item.OfMessage != nil {
				// Handle easy message items
				msg := item.OfMessage
				role := p.convertResponseMessageRoleToGai(string(msg.Role))

				// Convert message content - handle both string and array formats
				if !param.IsOmitted(msg.Content.OfString) {
					// Simple string content
					messages = append(messages, gai.Message{
						Role:    role,
						Content: gai.TextInput{Text: msg.Content.OfString.Value},
					})
				} else if msg.Content.OfInputItemContentList != nil {
					// Array of content items
					for _, content := range msg.Content.OfInputItemContentList {
						if content.OfInputText != nil {
							messages = append(messages, gai.Message{
								Role:    role,
								Content: gai.TextInput{Text: content.OfInputText.Text},
							})
						} else if content.OfInputImage != nil {
							var imageURL string
							if !param.IsOmitted(content.OfInputImage.ImageURL) {
								imageURL = content.OfInputImage.ImageURL.Value
							}
							detail := string(content.OfInputImage.Detail)
							messages = append(messages, gai.Message{
								Role:    role,
								Content: gai.ImageInput{URL: imageURL, Detail: detail},
							})
						}
					}
				}
			} else if item.OfInputMessage != nil {
				// Handle full message items (fallback for other message types)
				msg := item.OfInputMessage
				role := p.convertResponseMessageRoleToGai(msg.Role)

				// Convert message content
				for _, content := range msg.Content {
					if content.OfInputText != nil {
						messages = append(messages, gai.Message{
							Role:    role,
							Content: gai.TextInput{Text: content.OfInputText.Text},
						})
					} else if content.OfInputImage != nil {
						var imageURL string
						if !param.IsOmitted(content.OfInputImage.ImageURL) {
							imageURL = content.OfInputImage.ImageURL.Value
						}
						messages = append(messages, gai.Message{
							Role:    role,
							Content: gai.ImageInput{URL: imageURL},
						})
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
		// Fallback to empty text input
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
	if !param.IsOmitted(req.MaxOutputTokens) {
		gaiReq.MaxOutputTokens = int(req.MaxOutputTokens.Value)
	}

	// Handle tools if present
	if len(req.Tools) > 0 {
		gaiReq.Tools = p.convertResponseToolsToGai(req.Tools)
	}

	// Handle tool choice if present
	if !param.IsOmitted(req.ToolChoice) {
		gaiReq.ToolChoice = p.convertResponseToolChoiceToGai(req.ToolChoice)
	}

	return gaiReq
}

func (p *OpenAIProvider) convertResponseMessageRoleToGai(role string) gai.Role {
	switch role {
	case "user":
		return gai.RoleUser
	case "assistant":
		return gai.RoleAssistant
	case "system":
		return gai.RoleSystem
	case "tool":
		return gai.RoleTool
	default:
		return gai.RoleUser
	}
}

func (p *OpenAIProvider) convertResponseToolsToGai(responseTools []responses.ToolUnionParam) []gai.Tool {
	var gaiTools []gai.Tool

	for _, tool := range responseTools {
		if tool.OfFunction != nil {
			gaiTools = append(gaiTools, gai.Tool{
				Type: "function",
				Function: gai.Function{
					Name:        tool.OfFunction.Name,
					Description: tool.OfFunction.Description.Value,
					Parameters:  tool.OfFunction.Parameters,
				},
			})
		}
	}

	return gaiTools
}

func (p *OpenAIProvider) convertResponseToolChoiceToGai(toolChoice responses.ResponseNewParamsToolChoiceUnion) *gai.ToolChoice {
	// Handle different tool choice options
	if !param.IsOmitted(toolChoice.OfToolChoiceMode) {
		return &gai.ToolChoice{
			Type: string(toolChoice.OfToolChoiceMode.Value),
		}
	}

	if toolChoice.OfHostedTool != nil {
		return &gai.ToolChoice{
			Type: string(toolChoice.OfHostedTool.Type),
		}
	}

	if toolChoice.OfFunctionTool != nil {
		return &gai.ToolChoice{
			Type: "function",
			Function: &gai.ToolChoiceFunction{
				Name: toolChoice.OfFunctionTool.Name,
			},
		}
	}

	return nil
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

func (p *OpenAIProvider) convertFromGaiResponseToResponsesFormat(gaiResp *gai.Response, modelID string) map[string]any {
	// Extract text content from gai response
	var content string
	for _, output := range gaiResp.Output {
		if textOutput, ok := output.(gai.TextOutput); ok {
			content = textOutput.Text
			break
		}
	}

	// Create output items in responses format
	outputItems := []map[string]any{
		{
			"type": "message",
			"id":   "msg-" + gaiResp.ID,
			"role": "assistant",
			"content": []map[string]any{
				{
					"type": "output_text",
					"text": content,
				},
			},
			"status": "completed",
		},
	}

	// Convert to OpenAI responses format
	response := map[string]any{
		"id":         "resp-" + gaiResp.ID,
		"object":     "response",
		"created_at": float64(time.Now().Unix()),
		"model":      modelID,
		"output":     outputItems,
	}

	// Add usage information if available
	if gaiResp.Usage != nil {
		response["usage"] = map[string]any{
			"input_tokens":  gaiResp.Usage.PromptTokens,
			"output_tokens": gaiResp.Usage.CompletionTokens,
			"total_tokens":  gaiResp.Usage.TotalTokens,
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

func (p *OpenAIProvider) preprocessResponseInput(rawReq map[string]any) {
	// Check if there's an input field that's an array
	if input, ok := rawReq["input"].([]any); ok {
		for _, item := range input {
			if itemMap, ok := item.(map[string]any); ok {
				// Check if this looks like a message (has role and content, but no type)
				if _, hasRole := itemMap["role"]; hasRole {
					if _, hasContent := itemMap["content"]; hasContent {
						if _, hasType := itemMap["type"]; !hasType {
							// Add the missing type field
							itemMap["type"] = "message"
						}
					}
				}
			}
		}
	}
}

func (p *OpenAIProvider) Shutdown(ctx context.Context) error {
	p.options.Logger.Info("Shutting down OpenAI provider")
	return nil
}
