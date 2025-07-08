// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package providers

import (
	"context"
	"io"
	"time"

	"codeberg.org/gai-org/gai"
	"github.com/stretchr/testify/assert"
)

type mockModelRouter struct {
	models []gai.Model
	err    error
}

func (m *mockModelRouter) ListModels(ctx context.Context) ([]gai.Model, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.models, nil
}

type mockLLMClient struct {
	shouldStreamError bool
	streamChunks      []*gai.ResponseChunk
	toolCallResponse  *gai.Response
}

func (m *mockLLMClient) Generate(ctx context.Context, req gai.GenerateRequest) (*gai.Response, error) {
	if m.toolCallResponse != nil {
		return m.toolCallResponse, nil
	}
	return &gai.Response{
		ID:        "test-response-123",
		ModelID:   req.ModelID,
		Status:    "completed",
		Output:    []gai.OutputItem{gai.TextOutput{Text: "Hello! This is a test response."}},
		Usage:     &gai.TokenUsage{PromptTokens: 10, CompletionTokens: 15, TotalTokens: 25},
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockLLMClient) GenerateStream(ctx context.Context, req gai.GenerateRequest) (gai.ResponseStream, error) {
	if m.shouldStreamError {
		return nil, assert.AnError
	}

	chunks := m.streamChunks
	if chunks == nil {
		// Default test chunks
		chunks = []*gai.ResponseChunk{
			{
				ID:       "test-stream-123",
				Delta:    gai.OutputDelta{Text: "Hello"},
				Finished: false,
				Status:   "generating",
			},
			{
				ID:       "test-stream-123",
				Delta:    gai.OutputDelta{Text: " world"},
				Finished: false,
				Status:   "generating",
			},
			{
				ID:       "test-stream-123",
				Delta:    gai.OutputDelta{Text: "!"},
				Finished: true,
				Status:   "completed",
				Usage:    &gai.TokenUsage{PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15},
			},
		}
	}

	return &mockResponseStream{chunks: chunks}, nil
}

// mockResponseStream implements the ResponseStream interface
type mockResponseStream struct {
	chunks []*gai.ResponseChunk
	index  int
	closed bool
}

func (m *mockResponseStream) Next() (*gai.ResponseChunk, error) {
	if m.closed {
		return nil, io.EOF
	}
	if m.index >= len(m.chunks) {
		return nil, io.EOF
	}

	chunk := m.chunks[m.index]
	m.index++
	return chunk, nil
}

func (m *mockResponseStream) Close() error {
	m.closed = true
	return nil
}

func (m *mockResponseStream) Err() error {
	return nil
}
