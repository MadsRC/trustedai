// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package llmgw

import "errors"

var (
	// ErrNotFound should be returned when a requested resource cannot be found
	ErrNotFound = errors.New("not found")

	// ErrDuplicateEntry should be returned when a resource would violate unique constraints
	ErrDuplicateEntry = errors.New("duplicate entry")

	// ErrUnauthorized should be returned when a user lacks permission for an operation
	ErrUnauthorized = errors.New("unauthorized")
)
