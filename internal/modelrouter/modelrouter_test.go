// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !integration && !acceptance

package modelrouter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractActualModelID(t *testing.T) {
	tests := []struct {
		name            string
		modelReference  string
		expectedModelID string
		expectError     bool
	}{
		{
			name:            "valid openrouter reference",
			modelReference:  "openrouter:deepseek/deepseek-r1-0528-qwen3-8b:free",
			expectedModelID: "deepseek/deepseek-r1-0528-qwen3-8b:free",
			expectError:     false,
		},
		{
			name:            "valid openrouter reference with simple model",
			modelReference:  "openrouter:gpt-4",
			expectedModelID: "gpt-4",
			expectError:     false,
		},
		{
			name:            "invalid reference without colon",
			modelReference:  "openrouter-invalid-model",
			expectedModelID: "",
			expectError:     true,
		},
		{
			name:            "invalid reference with only provider",
			modelReference:  "openrouter:",
			expectedModelID: "",
			expectError:     false, // This returns empty string but doesn't error
		},
		{
			name:            "empty reference",
			modelReference:  "",
			expectedModelID: "",
			expectError:     true,
		},
		{
			name:            "reference with multiple colons",
			modelReference:  "openrouter:meta-llama:llama-3.1-8b",
			expectedModelID: "meta-llama:llama-3.1-8b", // Should include everything after first colon
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualModelID, err := extractActualModelID(tt.modelReference)

			if tt.expectError {
				require.Error(t, err)
				assert.Empty(t, actualModelID)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedModelID, actualModelID)
			}
		})
	}
}
