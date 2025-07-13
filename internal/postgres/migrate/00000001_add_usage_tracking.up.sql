-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- High-throughput usage events with complete audit trail
CREATE TABLE usage_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_id VARCHAR(100) NOT NULL,
    
    -- Token counts (nullable when unknown/unavailable)
    input_tokens INTEGER,
    output_tokens INTEGER, 
    cached_tokens INTEGER,
    reasoning_tokens INTEGER,
    
    -- Request outcome tracking  
    status VARCHAR(50) NOT NULL, -- 'success', 'failed', 'timeout', 'cancelled'
    failure_stage VARCHAR(50), -- 'pre_generation', 'during_generation', 'post_generation'
    error_type VARCHAR(100), -- 'auth_error', 'provider_timeout', 'rate_limit', etc.
    error_message TEXT, -- sanitized error details for debugging
    
    -- Data quality indicators
    usage_data_source VARCHAR(50) NOT NULL, -- 'provider_response', 'unavailable', 'streaming_incomplete'
    data_complete BOOLEAN NOT NULL DEFAULT false, -- true only when we're confident in token counts
    
    -- Timing information
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    duration_ms INTEGER, -- provider interaction duration (request sent to response received)
    
    CONSTRAINT valid_status CHECK (status IN ('success', 'failed', 'timeout', 'cancelled')),
    CONSTRAINT valid_failure_stage CHECK (failure_stage IN ('pre_generation', 'during_generation', 'post_generation')),
    CONSTRAINT valid_data_source CHECK (usage_data_source IN ('provider_response', 'unavailable', 'streaming_incomplete'))
);

-- Pre-aggregated billing summaries (only from data_complete = true events)
CREATE TABLE billing_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    total_requests INTEGER NOT NULL,
    total_input_tokens BIGINT NOT NULL,
    total_output_tokens BIGINT NOT NULL,
    total_cost_cents BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

