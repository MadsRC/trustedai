// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dataplane

import (
	"context"
	"time"

	"codeberg.org/MadsRC/llmgw/internal/api/dataplane/interfaces"
	"codeberg.org/gai-org/gai"
)

// TrackedLLMClient wraps an LLMClient to automatically handle usage tracking
type TrackedLLMClient struct {
	client          LLMClient
	usageMiddleware interfaces.UsageMiddleware
}

// NewTrackedLLMClient creates a new LLMClient wrapper that automatically tracks usage
func NewTrackedLLMClient(client LLMClient, usageMiddleware interfaces.UsageMiddleware) *TrackedLLMClient {
	return &TrackedLLMClient{
		client:          client,
		usageMiddleware: usageMiddleware,
	}
}

// Generate implements the LLMClient interface with automatic usage tracking
func (t *TrackedLLMClient) Generate(ctx context.Context, req gai.GenerateRequest) (*gai.Response, error) {
	startTime := time.Now()

	resp, err := t.client.Generate(ctx, req)
	duration := time.Since(startTime)

	// Track usage event if middleware is available
	if t.usageMiddleware != nil {
		var usage *interfaces.TokenUsage
		status := "success"

		if err != nil {
			status = "failed"
		} else if resp != nil && resp.Usage != nil {
			usage = &interfaces.TokenUsage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			}
		}

		t.usageMiddleware.UpdateEvent(ctx, req.ModelID, usage, status, duration)
	}

	return resp, err
}

// GenerateStream implements the LLMClient interface with automatic usage tracking
func (t *TrackedLLMClient) GenerateStream(ctx context.Context, req gai.GenerateRequest) (gai.ResponseStream, error) {
	startTime := time.Now()

	stream, err := t.client.GenerateStream(ctx, req)
	if err != nil {
		// Track failed streaming request
		if t.usageMiddleware != nil {
			duration := time.Since(startTime)
			t.usageMiddleware.UpdateEvent(ctx, req.ModelID, nil, "failed", duration)
		}
		return nil, err
	}

	// Wrap the stream to track usage when streaming is complete
	return &trackedResponseStream{
		stream:          stream,
		usageMiddleware: t.usageMiddleware,
		ctx:             ctx,
		modelID:         req.ModelID,
		startTime:       startTime,
	}, nil
}

// trackedResponseStream wraps a gai.ResponseStream to track usage when streaming completes
type trackedResponseStream struct {
	stream          gai.ResponseStream
	usageMiddleware interfaces.UsageMiddleware
	ctx             context.Context
	modelID         string
	startTime       time.Time
	totalUsage      *gai.TokenUsage
}

// Next forwards to the underlying stream and captures final usage data
func (t *trackedResponseStream) Next() (*gai.ResponseChunk, error) {
	chunk, err := t.stream.Next()

	// Capture usage from chunks (final chunk typically contains total usage)
	if chunk != nil && chunk.Usage != nil {
		t.totalUsage = chunk.Usage
	}

	// If this is the final chunk or there's an error, track the usage event
	if err != nil || (chunk != nil && chunk.Finished) {
		if t.usageMiddleware != nil {
			duration := time.Since(t.startTime)
			var usage *interfaces.TokenUsage
			status := "success"

			if err != nil {
				status = "failed"
			} else if t.totalUsage != nil {
				usage = &interfaces.TokenUsage{
					PromptTokens:     t.totalUsage.PromptTokens,
					CompletionTokens: t.totalUsage.CompletionTokens,
					TotalTokens:      t.totalUsage.TotalTokens,
				}
			}

			t.usageMiddleware.UpdateEvent(t.ctx, t.modelID, usage, status, duration)
		}
	}

	return chunk, err
}

// Close forwards to the underlying stream
func (t *trackedResponseStream) Close() error {
	return t.stream.Close()
}

// Err forwards to the underlying stream
func (t *trackedResponseStream) Err() error {
	return t.stream.Err()
}
