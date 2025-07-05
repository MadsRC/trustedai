<!-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk> -->
<!--  -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Testing Strategy

This document outlines the testing strategy for the project. Our approach divides tests into three distinct categories, each with specific purposes and characteristics.

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
