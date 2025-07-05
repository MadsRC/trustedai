<!-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk> -->
<!--  -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Model Aliasing Feature Testing

## Feature Overview

The model aliasing feature allows users to create custom model aliases that reference hardcoded provider models. When creating a model, any missing information (name, pricing, capabilities) is automatically inferred from the hardcoded models.

## How It Works

1. **Model Reference Format**: `provider:model_id` (e.g., `openrouter:deepseek/deepseek-r1-0528-qwen3-8b:free`)
2. **Automatic Inference with Partial Support**:
    - **Name**: If empty → uses hardcoded model name
    - **Pricing**: If not provided → uses hardcoded model pricing
        - **Partial Support**: If only `input_token_price` is provided → inherits `output_token_price` from hardcoded model
        - **Partial Support**: If only `output_token_price` is provided → inherits `input_token_price` from hardcoded model
    - **Capabilities**: If not provided → uses hardcoded model capabilities
        - **Partial Support**: If only some capabilities are provided → inherits missing capabilities from hardcoded model
        - **Note**: For boolean fields, explicit `false` values will override hardcoded `true` values

## Example Usage

### Create Model with Full Inference
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

### Create Model with Partial Override
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

### Create Model with Partial Pricing Override
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

### Create Model with Partial Capabilities Override
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

## Available Hardcoded Models

Currently available in `internal/models/config.go`:

1. `openrouter:deepseek/deepseek-r1-0528-qwen3-8b:free`
    - Name: "DeepSeek-R1-0528-Qwen3-8B"
    - Pricing: Free (0.0/0.0)
    - Capabilities: Full set with reasoning support

2. `openrouter:meta-llama/llama-4-maverick-17b-128e-instruct:free`
    - Name: "Llama-4-Maverick-17b-128e-instruct"
    - Pricing: Free (0.0/0.0)
    - Capabilities: Full set with reasoning support
