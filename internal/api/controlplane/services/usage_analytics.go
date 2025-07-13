// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"context"
	"errors"

	"codeberg.org/MadsRC/llmgw"
	llmgwv1 "codeberg.org/MadsRC/llmgw/gen/proto/madsrc/llmgw/v1"
	"codeberg.org/MadsRC/llmgw/gen/proto/madsrc/llmgw/v1/llmgwv1connect"
	cauth "codeberg.org/MadsRC/llmgw/internal/api/controlplane/auth"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UsageAnalytics holds the dependencies for the usage analytics service
type UsageAnalytics struct {
	options UsageAnalyticsOptions
}

// Ensure UsageAnalytics implements the required interfaces
var _ llmgwv1connect.UsageAnalyticsServiceHandler = (*UsageAnalytics)(nil)

// GetUsageSummary retrieves usage summary for the authenticated user
func (s *UsageAnalytics) GetUsageSummary(
	ctx context.Context,
	req *connect.Request[llmgwv1.UsageAnalyticsServiceGetUsageSummaryRequest],
) (*connect.Response[llmgwv1.UsageAnalyticsServiceGetUsageSummaryResponse], error) {
	s.options.Logger.Debug("[UsageAnalyticsService] GetUsageSummary invoked")

	// Extract authenticated user from context
	user, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request
	if req.Msg.GetStart() == nil || req.Msg.GetEnd() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("usage analytics service: start and end times are required"))
	}

	start := req.Msg.GetStart().AsTime()
	end := req.Msg.GetEnd().AsTime()

	if start.After(end) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("usage analytics service: start time must be before end time"))
	}

	// Try to use billing summaries for efficient queries first
	billingSummaries, err := s.options.BillingRepository.ListBillingSummariesByUser(ctx, user.ID, 1000, 0)
	if err != nil {
		s.options.Logger.Error("Failed to query billing summaries", "error", err, "user_id", user.ID)
		return nil, connect.NewError(connect.CodeInternal, errors.New("usage analytics service: failed to query billing summaries"))
	}

	// Filter summaries by time range and model
	var filteredSummaries []*llmgw.BillingSummary
	for _, summary := range billingSummaries {
		if summary.PeriodStart.Before(end) && summary.PeriodEnd.After(start) {
			filteredSummaries = append(filteredSummaries, summary)
		}
	}

	// Aggregate the data
	totalRequests := int32(0)
	totalInputTokens := int64(0)
	totalOutputTokens := int64(0)
	totalCostCents := int64(0)
	modelUsageMap := make(map[string]*llmgwv1.ModelUsage)

	// If we have billing summaries, use them for aggregation
	if len(filteredSummaries) > 0 {
		for _, summary := range filteredSummaries {
			totalRequests += int32(summary.TotalRequests)
			totalInputTokens += summary.TotalInputTokens
			totalOutputTokens += summary.TotalOutputTokens
			totalCostCents += summary.TotalCostCents
		}
	} else {
		// Fall back to querying usage events directly
		events, err := s.options.UsageRepository.ListUsageEventsByPeriod(ctx, user.ID, start, end)
		if err != nil {
			s.options.Logger.Error("Failed to query usage events", "error", err, "user_id", user.ID)
			return nil, connect.NewError(connect.CodeInternal, errors.New("usage analytics service: failed to query usage events"))
		}

		// Aggregate from events
		for _, event := range events {
			// Filter by model if specified
			if req.Msg.ModelId != nil && *req.Msg.ModelId != event.ModelID {
				continue
			}

			totalRequests++
			if event.InputTokens != nil {
				totalInputTokens += int64(*event.InputTokens)
			}
			if event.OutputTokens != nil {
				totalOutputTokens += int64(*event.OutputTokens)
			}
			if event.TotalCostCents != nil {
				totalCostCents += *event.TotalCostCents
			}

			// Track model usage
			if modelUsage, exists := modelUsageMap[event.ModelID]; exists {
				modelUsage.Requests++
				if event.InputTokens != nil {
					modelUsage.InputTokens += int64(*event.InputTokens)
				}
				if event.OutputTokens != nil {
					modelUsage.OutputTokens += int64(*event.OutputTokens)
				}
				if event.TotalCostCents != nil {
					modelUsage.CostCents += *event.TotalCostCents
				}
			} else {
				inputTokens := int64(0)
				outputTokens := int64(0)
				costCents := int64(0)
				if event.InputTokens != nil {
					inputTokens = int64(*event.InputTokens)
				}
				if event.OutputTokens != nil {
					outputTokens = int64(*event.OutputTokens)
				}
				if event.TotalCostCents != nil {
					costCents = *event.TotalCostCents
				}

				modelUsageMap[event.ModelID] = &llmgwv1.ModelUsage{
					ModelId:      event.ModelID,
					Requests:     1,
					InputTokens:  inputTokens,
					OutputTokens: outputTokens,
					CostCents:    costCents,
				}
			}
		}
	}

	// Convert map to slice
	var models []*llmgwv1.ModelUsage
	for _, modelUsage := range modelUsageMap {
		models = append(models, modelUsage)
	}

	response := &llmgwv1.UsageAnalyticsServiceGetUsageSummaryResponse{
		Period: &llmgwv1.UsagePeriod{
			Start: req.Msg.GetStart(),
			End:   req.Msg.GetEnd(),
		},
		Summary: &llmgwv1.UsageSummary{
			TotalRequests:     totalRequests,
			TotalInputTokens:  totalInputTokens,
			TotalOutputTokens: totalOutputTokens,
			TotalCostCents:    totalCostCents,
			Models:            models,
		},
	}

	return connect.NewResponse(response), nil
}

// GetUsageDetails retrieves detailed usage events for the authenticated user
func (s *UsageAnalytics) GetUsageDetails(
	ctx context.Context,
	req *connect.Request[llmgwv1.UsageAnalyticsServiceGetUsageDetailsRequest],
) (*connect.Response[llmgwv1.UsageAnalyticsServiceGetUsageDetailsResponse], error) {
	s.options.Logger.Debug("[UsageAnalyticsService] GetUsageDetails invoked")

	// Extract authenticated user from context
	user, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request
	if req.Msg.GetStart() == nil || req.Msg.GetEnd() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("usage analytics service: start and end times are required"))
	}

	start := req.Msg.GetStart().AsTime()
	end := req.Msg.GetEnd().AsTime()

	if start.After(end) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("usage analytics service: start time must be before end time"))
	}

	// Validate pagination parameters
	if req.Msg.GetLimit() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("usage analytics service: limit must be positive"))
	}

	if req.Msg.GetOffset() < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("usage analytics service: offset must be non-negative"))
	}

	// Query usage events
	events, err := s.options.UsageRepository.ListUsageEventsByPeriod(ctx, user.ID, start, end)
	if err != nil {
		s.options.Logger.Error("Failed to query usage events", "error", err, "user_id", user.ID)
		return nil, connect.NewError(connect.CodeInternal, errors.New("usage analytics service: failed to query usage events"))
	}

	// Filter by model if specified
	var filteredEvents []*llmgw.UsageEvent
	for _, event := range events {
		if req.Msg.ModelId != nil && *req.Msg.ModelId != event.ModelID {
			continue
		}
		filteredEvents = append(filteredEvents, event)
	}

	totalCount := int32(len(filteredEvents))

	// Apply pagination
	offset := int(req.Msg.GetOffset())
	limit := int(req.Msg.GetLimit())

	start_idx := offset
	end_idx := offset + limit
	if start_idx > len(filteredEvents) {
		start_idx = len(filteredEvents)
	}
	if end_idx > len(filteredEvents) {
		end_idx = len(filteredEvents)
	}

	paginatedEvents := filteredEvents[start_idx:end_idx]

	// Convert to protobuf format
	var protoEvents []*llmgwv1.UsageEvent
	for _, event := range paginatedEvents {
		protoEvent := &llmgwv1.UsageEvent{
			Id:              event.ID,
			RequestId:       event.RequestID,
			UserId:          event.UserID,
			ModelId:         event.ModelID,
			Status:          event.Status,
			UsageDataSource: event.UsageDataSource,
			DataComplete:    event.DataComplete,
			Timestamp:       timestamppb.New(event.Timestamp),
		}

		// Handle optional fields
		if event.InputTokens != nil {
			val := int32(*event.InputTokens)
			protoEvent.InputTokens = &val
		}
		if event.OutputTokens != nil {
			val := int32(*event.OutputTokens)
			protoEvent.OutputTokens = &val
		}
		if event.CachedTokens != nil {
			val := int32(*event.CachedTokens)
			protoEvent.CachedTokens = &val
		}
		if event.ReasoningTokens != nil {
			val := int32(*event.ReasoningTokens)
			protoEvent.ReasoningTokens = &val
		}
		if event.FailureStage != nil {
			protoEvent.FailureStage = event.FailureStage
		}
		if event.ErrorType != nil {
			protoEvent.ErrorType = event.ErrorType
		}
		if event.ErrorMessage != nil {
			protoEvent.ErrorMessage = event.ErrorMessage
		}
		if event.DurationMs != nil {
			val := int32(*event.DurationMs)
			protoEvent.DurationMs = &val
		}
		if event.InputCostCents != nil {
			protoEvent.InputCostCents = event.InputCostCents
		}
		if event.OutputCostCents != nil {
			protoEvent.OutputCostCents = event.OutputCostCents
		}
		if event.TotalCostCents != nil {
			protoEvent.TotalCostCents = event.TotalCostCents
		}

		protoEvents = append(protoEvents, protoEvent)
	}

	response := &llmgwv1.UsageAnalyticsServiceGetUsageDetailsResponse{
		Events:     protoEvents,
		TotalCount: totalCount,
	}

	return connect.NewResponse(response), nil
}

// GetUsageCosts retrieves cost breakdown for the authenticated user
func (s *UsageAnalytics) GetUsageCosts(
	ctx context.Context,
	req *connect.Request[llmgwv1.UsageAnalyticsServiceGetUsageCostsRequest],
) (*connect.Response[llmgwv1.UsageAnalyticsServiceGetUsageCostsResponse], error) {
	s.options.Logger.Debug("[UsageAnalyticsService] GetUsageCosts invoked")

	// Extract authenticated user from context
	user, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request
	if req.Msg.GetStart() == nil || req.Msg.GetEnd() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("usage analytics service: start and end times are required"))
	}

	start := req.Msg.GetStart().AsTime()
	end := req.Msg.GetEnd().AsTime()

	if start.After(end) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("usage analytics service: start time must be before end time"))
	}

	// Query usage events to calculate cost breakdown by model
	events, err := s.options.UsageRepository.ListUsageEventsByPeriod(ctx, user.ID, start, end)
	if err != nil {
		s.options.Logger.Error("Failed to query usage events", "error", err, "user_id", user.ID)
		return nil, connect.NewError(connect.CodeInternal, errors.New("usage analytics service: failed to query usage events"))
	}

	// Calculate cost breakdown by model
	costBreakdownMap := make(map[string]*llmgwv1.CostBreakdown)
	totalCostCents := int64(0)

	for _, event := range events {
		// Filter by model if specified
		if req.Msg.ModelId != nil && *req.Msg.ModelId != event.ModelID {
			continue
		}

		// Only include events with cost data
		if event.InputCostCents == nil && event.OutputCostCents == nil {
			continue
		}

		inputCost := int64(0)
		outputCost := int64(0)
		eventTotalCost := int64(0)

		if event.InputCostCents != nil {
			inputCost = *event.InputCostCents
		}
		if event.OutputCostCents != nil {
			outputCost = *event.OutputCostCents
		}
		if event.TotalCostCents != nil {
			eventTotalCost = *event.TotalCostCents
		} else {
			eventTotalCost = inputCost + outputCost
		}

		totalCostCents += eventTotalCost

		if breakdown, exists := costBreakdownMap[event.ModelID]; exists {
			breakdown.InputCostCents += inputCost
			breakdown.OutputCostCents += outputCost
			breakdown.TotalCostCents += eventTotalCost
			breakdown.Requests++
		} else {
			costBreakdownMap[event.ModelID] = &llmgwv1.CostBreakdown{
				ModelId:         event.ModelID,
				InputCostCents:  inputCost,
				OutputCostCents: outputCost,
				TotalCostCents:  eventTotalCost,
				Requests:        1,
			}
		}
	}

	// Convert map to slice
	var costBreakdown []*llmgwv1.CostBreakdown
	for _, breakdown := range costBreakdownMap {
		costBreakdown = append(costBreakdown, breakdown)
	}

	response := &llmgwv1.UsageAnalyticsServiceGetUsageCostsResponse{
		Period: &llmgwv1.UsagePeriod{
			Start: req.Msg.GetStart(),
			End:   req.Msg.GetEnd(),
		},
		CostBreakdown:  costBreakdown,
		TotalCostCents: totalCostCents,
	}

	return connect.NewResponse(response), nil
}

// Organization methods (admin only)

// GetOrganizationUsageSummary retrieves usage summary for an organization (admin only)
func (s *UsageAnalytics) GetOrganizationUsageSummary(
	ctx context.Context,
	req *connect.Request[llmgwv1.UsageAnalyticsServiceGetOrganizationUsageSummaryRequest],
) (*connect.Response[llmgwv1.UsageAnalyticsServiceGetOrganizationUsageSummaryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("organization usage summary not yet implemented"))
}

// GetOrganizationUsageByUser retrieves organization usage broken down by user (admin only)
func (s *UsageAnalytics) GetOrganizationUsageByUser(
	ctx context.Context,
	req *connect.Request[llmgwv1.UsageAnalyticsServiceGetOrganizationUsageByUserRequest],
) (*connect.Response[llmgwv1.UsageAnalyticsServiceGetOrganizationUsageByUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("organization usage by user not yet implemented"))
}

// GetOrganizationUsageByModel retrieves organization usage broken down by model (admin only)
func (s *UsageAnalytics) GetOrganizationUsageByModel(
	ctx context.Context,
	req *connect.Request[llmgwv1.UsageAnalyticsServiceGetOrganizationUsageByModelRequest],
) (*connect.Response[llmgwv1.UsageAnalyticsServiceGetOrganizationUsageByModelResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("organization usage by model not yet implemented"))
}

// getUserFromConnection extracts the authenticated user from the connection context
// This function checks both session-based authentication (SSO) and API key authentication
func (s *UsageAnalytics) getUserFromConnection(ctx context.Context) (*llmgw.User, error) {
	// Check for session-based authentication (SSO) first
	if session := cauth.SessionFromContext(ctx); session != nil {
		s.options.Logger.Debug("[UsageAnalyticsService] Found session user",
			"userID", session.User.ID, "email", session.User.Email, "authMethod", "session")
		return session.User, nil
	}

	// Fall back to API key authentication
	if apiKeyUser := cauth.UserFromContext(ctx); apiKeyUser != nil {
		s.options.Logger.Debug("[UsageAnalyticsService] Found API key user",
			"userID", apiKeyUser.ID, "email", apiKeyUser.Email, "authMethod", "apikey")
		return apiKeyUser, nil
	}

	// No authentication found
	return nil, connect.NewError(connect.CodeUnauthenticated,
		errors.New("usage analytics service: no authenticated user found"))
}
