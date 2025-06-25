<!-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk> -->
<!--  -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Testing Strategy

This document outlines the testing strategy for the GAI package. Our approach divides tests into three distinct categories, each with specific purposes and characteristics.

## Test Categories

### Unit Tests
- **Purpose**: Test isolated units of behavior
- **Speed**: Fast (run often during development)
- **Mocking**: Heavy reliance on mocks
- **Command**: `mise run test:unit`

Unit tests focus on testing individual functions, methods, or small components in isolation. They should be fast to execute since they're run frequently during development.

**Mocking Guidelines for Unit Tests:**
- Always keep mocks as simple as possible
- Avoid testing the mock instead of the actual code
- Mock external dependencies and collaborators
- Use testify's assert or require packages for assertions

### Integration Tests
- **Purpose**: Test how different parts of the codebase integrate
- **Speed**: Slower than unit tests, more involved
- **Mocking**: Less reliance on mocks, more actual dependencies
- **Command**: `mise run test:integration`

Integration tests verify that different components work together correctly. They can be slower and more complex than unit tests but should still maintain reasonable execution times.

### Acceptance Tests
- **Purpose**: Test the library from a user's perspective
- **Speed**: Slowest, most dependencies
- **Mocking**: Only real external dependencies
- **Command**: `mise run test:acceptance`

Acceptance tests verify that the library works as expected from an end-user perspective. They are the slowest category and have the most external dependencies.

**Important Rule**: Tests that communicate with actual LLMs must always be acceptance tests.

## Running Tests

- **All tests**: `mise run test`
- **Unit tests only**: `mise run test:unit`
- **Integration tests only**: `mise run test:integration`
- **Acceptance tests only**: `mise run test:acceptance`

## Test Guidelines

1. **Keep unit tests fast** - They run frequently during development
2. **Use simple mocks** - Avoid complex mock behavior that becomes a test target itself
3. **Real dependencies in acceptance tests** - No mocking of external services
4. **LLM communication = acceptance test** - Any test that talks to an actual LLM service
5. **Use testify** - Leverage assert or require packages for all Go tests

## Test Structure

Tests are organized to match the three-tier strategy:
- Unit tests: Fast, isolated, heavily mocked
- Integration tests: Medium speed, some real dependencies
- Acceptance tests: Slow, real dependencies, user perspective

### File Organization

- **Unit tests**: `*_test.go` files with `//go:build !integration && !acceptance` constraint
- **Integration tests**: `*_integration_test.go` files with `//go:build integration` constraint
- **Acceptance tests**: `*_acceptance_test.go` files with `//go:build acceptance` constraint

This file naming and build constraint strategy ensures that tests can be run selectively based on their category and requirements.

## Model Aliasing Feature Testing

### Feature Overview

The model aliasing feature allows users to create custom model aliases that reference hardcoded provider models. When creating a model, any missing information (name, pricing, capabilities) is automatically inferred from the hardcoded models.

### How It Works

1. **Model Reference Format**: `provider:model_id` (e.g., `openrouter:deepseek/deepseek-r1-0528-qwen3-8b:free`)
2. **Automatic Inference with Partial Support**: 
   - **Name**: If empty → uses hardcoded model name
   - **Pricing**: If not provided → uses hardcoded model pricing
     - **Partial Support**: If only `input_token_price` is provided → inherits `output_token_price` from hardcoded model
     - **Partial Support**: If only `output_token_price` is provided → inherits `input_token_price` from hardcoded model
   - **Capabilities**: If not provided → uses hardcoded model capabilities  
     - **Partial Support**: If only some capabilities are provided → inherits missing capabilities from hardcoded model
     - **Note**: For boolean fields, explicit `false` values will override hardcoded `true` values

### Example Usage

#### Create Model with Full Inference
```json
{
  "model": {
    "id": "my-deepseek-alias",
    "model_reference": "openrouter:deepseek/deepseek-r1-0528-qwen3-8b:free",
    "provider_id": "openrouter", 
    "credential_id": "uuid-here",
    "credential_type": "openrouter"
  }
}
```
This will create a model with:
- Custom ID: `my-deepseek-alias`
- Inferred name: `DeepSeek-R1-0528-Qwen3-8B`
- Inferred pricing: `{ input: 0.0, output: 0.0 }`
- Inferred capabilities: Full capabilities from hardcoded model

#### Create Model with Partial Override
```json
{
  "model": {
    "id": "custom-deepseek",
    "name": "My Custom DeepSeek",
    "model_reference": "openrouter:deepseek/deepseek-r1-0528-qwen3-8b:free",
    "provider_id": "openrouter",
    "credential_id": "uuid-here", 
    "credential_type": "openrouter"
  }
}
```
This will create a model with:
- Custom ID: `custom-deepseek`
- Custom name: `My Custom DeepSeek` 
- Inferred pricing and capabilities from hardcoded model

#### Create Model with Partial Pricing Override
```json
{
  "model": {
    "id": "premium-deepseek",
    "model_reference": "openrouter:deepseek/deepseek-r1-0528-qwen3-8b:free",
    "provider_id": "openrouter",
    "credential_id": "uuid-here",
    "credential_type": "openrouter",
    "pricing": {
      "input_token_price": 0.001
    }
  }
}
```
This will create a model with:
- Custom ID: `premium-deepseek`
- Inferred name: `DeepSeek-R1-0528-Qwen3-8B`
- **Partial pricing**: `input_token_price: 0.001`, `output_token_price: 0.0` (inherited)
- Inferred capabilities from hardcoded model

#### Create Model with Partial Capabilities Override
```json
{
  "model": {
    "id": "limited-deepseek", 
    "model_reference": "openrouter:deepseek/deepseek-r1-0528-qwen3-8b:free",
    "provider_id": "openrouter",
    "credential_id": "uuid-here",
    "credential_type": "openrouter",
    "capabilities": {
      "max_input_tokens": 16384,
      "supports_vision": false
    }
  }
}
```
This will create a model with:
- Custom ID: `limited-deepseek`
- Inferred name and pricing from hardcoded model
- **Partial capabilities**: `max_input_tokens: 16384`, `supports_vision: false`, all other capabilities inherited from hardcoded model

### Available Hardcoded Models

Currently available in `internal/models/config.go`:

1. `openrouter:deepseek/deepseek-r1-0528-qwen3-8b:free`
   - Name: "DeepSeek-R1-0528-Qwen3-8B"
   - Pricing: Free (0.0/0.0)
   - Capabilities: Full set with reasoning support

2. `openrouter:meta-llama/llama-4-maverick-17b-128e-instruct:free`
   - Name: "Llama-4-Maverick-17b-128e-instruct"
   - Pricing: Free (0.0/0.0)  
   - Capabilities: Full set with reasoning support

### Testing Steps

1. **Setup**: Ensure database has credentials table and models table
2. **Create Credential**: First create an OpenRouter credential
3. **Create Model Alias**: Use CreateModel API with model_reference
4. **Verify**: Check that model was created with inferred values
5. **List Models**: Verify model appears in ListModels response **with model_reference field**
6. **Get Model**: Verify GetModel response **includes model_reference field**

### Benefits

- **User Friendly**: Users can create memorable aliases like "my-fast-model"
- **Automatic Updates**: If hardcoded model info changes, aliases inherit updates
- **Consistent**: All models reference the same underlying provider model definitions
- **Flexible**: Users can override specific fields while inheriting others
