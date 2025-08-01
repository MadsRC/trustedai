// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package providers

import (
	"encoding/json"
	"fmt"
	"time"
)

// InputUnion represents a union type that can be either a string or a slice of InputItem
type InputUnion struct {
	StringValue string      `json:"-"`
	ArrayValue  []InputItem `json:"-"`
	isString    bool
}

// NewInputUnionFromString creates an InputUnion from a string
func NewInputUnionFromString(s string) InputUnion {
	return InputUnion{
		StringValue: s,
		isString:    true,
	}
}

// NewInputUnionFromArray creates an InputUnion from a slice of InputItem
func NewInputUnionFromArray(items []InputItem) InputUnion {
	return InputUnion{
		ArrayValue: items,
		isString:   false,
	}
}

// IsString returns true if the union contains a string value
func (u InputUnion) IsString() bool {
	return u.isString
}

// String returns the string value if the union contains a string
func (u InputUnion) String() (string, error) {
	if !u.isString {
		return "", fmt.Errorf("union does not contain a string value")
	}
	return u.StringValue, nil
}

// Array returns the array value if the union contains an array
func (u InputUnion) Array() ([]InputItem, error) {
	if u.isString {
		return nil, fmt.Errorf("union does not contain an array value")
	}
	return u.ArrayValue, nil
}

// MarshalJSON implements json.Marshaler for InputUnion
func (u InputUnion) MarshalJSON() ([]byte, error) {
	if u.isString {
		return json.Marshal(u.StringValue)
	}
	return json.Marshal(u.ArrayValue)
}

// UnmarshalJSON implements json.Unmarshaler for InputUnion
func (u *InputUnion) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		u.StringValue = str
		u.isString = true
		return nil
	}

	// Try to unmarshal as array
	var items []InputItem
	if err := json.Unmarshal(data, &items); err == nil {
		u.ArrayValue = items
		u.isString = false
		return nil
	}

	return fmt.Errorf("input must be either a string or an array of input items")
}

type CreateResponseRequest struct {
	Input              InputUnion          `json:"input"`
	InputItems         []InputItem         `json:"input_items,omitempty"`
	Include            []Includable        `json:"include,omitempty"`
	ParallelToolCalls  *bool               `json:"parallel_tool_calls,omitempty"`
	Store              *bool               `json:"store,omitempty"`
	Instructions       string              `json:"instructions,omitempty"`
	Stream             *bool               `json:"stream,omitempty"`
	PreviousResponseID string              `json:"previous_response_id,omitempty"`
	Model              string              `json:"model"`
	Reasoning          *Reasoning          `json:"reasoning,omitempty"`
	Background         *bool               `json:"background,omitempty"`
	MaxOutputTokens    *int                `json:"max_output_tokens,omitempty"`
	MaxToolCalls       *int                `json:"max_tool_calls,omitempty"`
	Text               *TextResponseFormat `json:"text,omitempty"`
	Metadata           map[string]any      `json:"metadata,omitempty"`
	TopLogprobs        *int                `json:"top_logprobs,omitempty"`
	Temperature        *float64            `json:"temperature,omitempty"`
	TopP               *float64            `json:"top_p,omitempty"`
	User               string              `json:"user,omitempty"`
	ServiceTier        *ServiceTier        `json:"service_tier,omitempty"`
}

type Response struct {
	ID                string              `json:"id"`
	Object            string              `json:"object"`
	Status            ResponseStatus      `json:"status"`
	CreatedAt         int64               `json:"created_at"`
	Error             *ResponseError      `json:"error"`
	IncompleteDetails *IncompleteDetails  `json:"incomplete_details"`
	Output            []OutputItem        `json:"output"`
	Instructions      any                 `json:"instructions"`
	OutputText        *string             `json:"output_text"`
	Usage             *ResponseUsage      `json:"usage"`
	ParallelToolCalls bool                `json:"parallel_tool_calls"`
	Model             string              `json:"model"`
	Reasoning         *Reasoning          `json:"reasoning,omitempty"`
	Background        bool                `json:"background"`
	MaxOutputTokens   *int                `json:"max_output_tokens,omitempty"`
	MaxToolCalls      *int                `json:"max_tool_calls,omitempty"`
	Text              *TextResponseFormat `json:"text,omitempty"`
	Metadata          map[string]any      `json:"metadata,omitempty"`
	TopLogprobs       *int                `json:"top_logprobs,omitempty"`
	Temperature       *float64            `json:"temperature,omitempty"`
	TopP              *float64            `json:"top_p,omitempty"`
	User              string              `json:"user,omitempty"`
	ServiceTier       *ServiceTier        `json:"service_tier,omitempty"`
}

type ResponseStatus string

const (
	ResponseStatusCompleted  ResponseStatus = "completed"
	ResponseStatusFailed     ResponseStatus = "failed"
	ResponseStatusInProgress ResponseStatus = "in_progress"
	ResponseStatusCancelled  ResponseStatus = "cancelled"
	ResponseStatusQueued     ResponseStatus = "queued"
	ResponseStatusIncomplete ResponseStatus = "incomplete"
)

type ResponseError struct {
	Code    ResponseErrorCode `json:"code"`
	Message string            `json:"message"`
}

type ResponseErrorCode string

const (
	ResponseErrorServerError                 ResponseErrorCode = "server_error"
	ResponseErrorRateLimitExceeded           ResponseErrorCode = "rate_limit_exceeded"
	ResponseErrorInvalidPrompt               ResponseErrorCode = "invalid_prompt"
	ResponseErrorVectorStoreTimeout          ResponseErrorCode = "vector_store_timeout"
	ResponseErrorInvalidImage                ResponseErrorCode = "invalid_image"
	ResponseErrorInvalidImageFormat          ResponseErrorCode = "invalid_image_format"
	ResponseErrorInvalidBase64Image          ResponseErrorCode = "invalid_base64_image"
	ResponseErrorInvalidImageURL             ResponseErrorCode = "invalid_image_url"
	ResponseErrorImageTooLarge               ResponseErrorCode = "image_too_large"
	ResponseErrorImageTooSmall               ResponseErrorCode = "image_too_small"
	ResponseErrorImageParseError             ResponseErrorCode = "image_parse_error"
	ResponseErrorImageContentPolicyViolation ResponseErrorCode = "image_content_policy_violation"
	ResponseErrorInvalidImageMode            ResponseErrorCode = "invalid_image_mode"
	ResponseErrorImageFileTooLarge           ResponseErrorCode = "image_file_too_large"
	ResponseErrorUnsupportedImageMediaType   ResponseErrorCode = "unsupported_image_media_type"
	ResponseErrorEmptyImageFile              ResponseErrorCode = "empty_image_file"
	ResponseErrorFailedToDownloadImage       ResponseErrorCode = "failed_to_download_image"
	ResponseErrorImageFileNotFound           ResponseErrorCode = "image_file_not_found"
)

type IncompleteDetails struct {
	Reason IncompleteReason `json:"reason"`
}

type IncompleteReason string

const (
	IncompleteReasonMaxOutputTokens IncompleteReason = "max_output_tokens"
	IncompleteReasonContentFilter   IncompleteReason = "content_filter"
)

type ResponseUsage struct {
	InputTokens         int                 `json:"input_tokens"`
	InputTokensDetails  InputTokensDetails  `json:"input_tokens_details"`
	OutputTokens        int                 `json:"output_tokens"`
	OutputTokensDetails OutputTokensDetails `json:"output_tokens_details"`
	TotalTokens         int                 `json:"total_tokens"`
}

type InputTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type OutputTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

type InputItem struct {
	Type    InputItemType   `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
}

type InputItemType string

const (
	InputItemTypeMessage InputItemType = "message"
	InputItemTypeItem    InputItemType = "item"
)

type OutputItem struct {
	Type    OutputItemType  `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
}

type OutputItemType string

const (
	OutputItemTypeMessage                 OutputItemType = "message"
	OutputItemTypeFileSearchToolCall      OutputItemType = "file_search_tool_call"
	OutputItemTypeFunctionToolCall        OutputItemType = "function_tool_call"
	OutputItemTypeWebSearchToolCall       OutputItemType = "web_search_tool_call"
	OutputItemTypeComputerToolCall        OutputItemType = "computer_tool_call"
	OutputItemTypeReasoningItem           OutputItemType = "reasoning_item"
	OutputItemTypeImageGenToolCall        OutputItemType = "image_gen_tool_call"
	OutputItemTypeCodeInterpreterToolCall OutputItemType = "code_interpreter_tool_call"
	OutputItemTypeLocalShellToolCall      OutputItemType = "local_shell_tool_call"
	OutputItemTypeMCPToolCall             OutputItemType = "mcp_tool_call"
	OutputItemTypeMCPListTools            OutputItemType = "mcp_list_tools"
	OutputItemTypeMCPApprovalRequest      OutputItemType = "mcp_approval_request"
)

type Includable string

const (
	IncludableCodeInterpreterCallOutputs Includable = "code_interpreter_call.outputs"
	IncludableComputerCallOutputImageURL Includable = "computer_call_output.output.image_url"
	IncludableFileSearchCallResults      Includable = "file_search_call.results"
	IncludableMessageInputImageImageURL  Includable = "message.input_image.image_url"
	IncludableMessageOutputTextLogprobs  Includable = "message.output_text.logprobs"
	IncludableReasoningEncryptedContent  Includable = "reasoning.encrypted_content"
)

type Reasoning struct {
	Effort string `json:"effort,omitempty"`
}

type TextResponseFormat struct {
	Format TextResponseFormatType `json:"format,omitempty"`
}

type TextResponseFormatType string

const (
	TextResponseFormatText TextResponseFormatType = "text"
	TextResponseFormatJSON TextResponseFormatType = "json"
)

type ServiceTier string

const (
	ServiceTierAuto    ServiceTier = "auto"
	ServiceTierDefault ServiceTier = "default"
)

type ResponseItemList struct {
	Object  string         `json:"object"`
	Data    []ItemResource `json:"data"`
	HasMore bool           `json:"has_more"`
	FirstID string         `json:"first_id"`
	LastID  string         `json:"last_id"`
}

type ItemResource struct {
	ID        string         `json:"id"`
	Object    string         `json:"object"`
	Type      string         `json:"type"`
	CreatedAt int64          `json:"created_at"`
	Content   map[string]any `json:"content"`
}

type ResponseStreamEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type CreateResponseRequestParams struct {
	Include       []Includable `json:"include,omitempty"`
	Stream        *bool        `json:"stream,omitempty"`
	StartingAfter *int         `json:"starting_after,omitempty"`
}

type GetResponseParams struct {
	ResponseID    string       `json:"response_id"`
	Include       []Includable `json:"include,omitempty"`
	Stream        *bool        `json:"stream,omitempty"`
	StartingAfter *int         `json:"starting_after,omitempty"`
}

type ListInputItemsParams struct {
	ResponseID string       `json:"response_id"`
	Limit      *int         `json:"limit,omitempty"`
	Order      *string      `json:"order,omitempty"`
	After      *string      `json:"after,omitempty"`
	Before     *string      `json:"before,omitempty"`
	Include    []Includable `json:"include,omitempty"`
}

func NewResponse(id string) *Response {
	return &Response{
		ID:                id,
		Object:            "response",
		Status:            ResponseStatusInProgress,
		CreatedAt:         time.Now().Unix(),
		Error:             nil,
		IncompleteDetails: nil,
		Output:            []OutputItem{},
		ParallelToolCalls: true,
		Background:        false,
	}
}

func (r *Response) SetCompleted() {
	r.Status = ResponseStatusCompleted
}

func (r *Response) SetFailed(errorCode ResponseErrorCode, message string) {
	r.Status = ResponseStatusFailed
	r.Error = &ResponseError{
		Code:    errorCode,
		Message: message,
	}
}

func (r *Response) SetCancelled() {
	r.Status = ResponseStatusCancelled
}

func (r *Response) AddOutputItem(item OutputItem) {
	r.Output = append(r.Output, item)
}
