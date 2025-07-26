<!-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk> -->
<!--  -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Frontend Testing Strategy

This document outlines the testing strategy for the frontend React application. Our approach follows the same three-tier strategy as our Go backend, adapted for frontend development.

## Test Categories

### Unit Tests

- **Purpose**: Test isolated units of behavior (functions, hooks, utils)
- **Speed**: Fast (run often during development)
- **Mocking**: Heavy reliance on mocks for external dependencies
- **Command**: `npm run test:unit`
- **Files**: `*.test.{ts,tsx}`

Unit tests focus on testing individual functions, utilities, hooks, or small components in isolation. They should be fast to execute since they're run frequently during development.

**Guidelines for Unit Tests:**

- Mock all external dependencies (API calls, context providers, etc.)
- Test pure functions and utility functions
- Test custom hooks in isolation
- Use simple, focused assertions

### Integration Tests

- **Purpose**: Test how different components work together
- **Speed**: Slower than unit tests, more involved
- **Mocking**: Less reliance on mocks, test component interactions
- **Command**: `npm run test:integration`
- **Files**: `*.integration.test.{ts,tsx}`

Integration tests verify that different components work together correctly. They can be slower and more complex than unit tests but should still maintain reasonable execution times.

**Guidelines for Integration Tests:**

- Test component interactions and data flow
- Mock only external services (API calls)
- Test routing and navigation
- Test form submissions and user interactions

### Acceptance Tests

- **Purpose**: Test complete user workflows from end-to-end
- **Speed**: Slowest, most dependencies
- **Mocking**: Only mock external services outside our control
- **Command**: `npm run test:acceptance`
- **Files**: `*.acceptance.test.{ts,tsx}`

Acceptance tests verify that the application works as expected from an end-user perspective. They are the slowest category and have the most dependencies.

**Important Rule**: Tests that communicate with actual backend APIs should be acceptance tests.

## Running Tests

- **All tests**: `npm run test`
- **Unit tests only**: `npm run test:unit`
- **Integration tests only**: `npm run test:integration`
- **Acceptance tests only**: `npm run test:acceptance`
- **Watch mode**: `npm run test:watch`
- **With coverage**: `npm run test:coverage`

## Technology Stack

- **Test Framework**: Vitest (fast, Vite-native)
- **Testing Library**: React Testing Library (component testing)
- **User Interaction**: @testing-library/user-event
- **Assertions**: Vitest's built-in assertions + @testing-library/jest-dom
- **Environment**: jsdom (browser-like environment)

## Test Guidelines

1. **Keep unit tests fast** - They run frequently during development
2. **Use simple mocks** - Avoid complex mock behavior that becomes a test target itself
3. **Real backend calls = acceptance test** - Any test that calls actual API endpoints
4. **Test user behavior, not implementation** - Focus on what the user sees and does
5. **Use semantic queries** - Prefer `getByRole`, `getByLabelText` over `getByTestId`

## File Organization

Tests are organized to match the three-tier strategy:

- **Unit tests**: `*.test.{ts,tsx}` - Fast, isolated, heavily mocked
- **Integration tests**: `*.integration.test.{ts,tsx}` - Medium speed, component interactions
- **Acceptance tests**: `*.acceptance.test.{ts,tsx}` - Slow, full user workflows

## Test Structure Example

```typescript
// Unit test example
describe("formatCost", () => {
  it("should format small costs with higher precision", () => {
    expect(formatCost(0.5)).toBe("$0.005");
  });
});

// Integration test example
describe("Analytics Component Integration", () => {
  it("should display cost data when API returns data", async () => {
    // Mock API responses, test component rendering
  });
});

// Acceptance test example
describe("User Analytics Workflow", () => {
  it("should allow user to view analytics and change time periods", async () => {
    // Test complete user workflow with real backend calls
  });
});
```

This file naming and configuration strategy ensures that tests can be run selectively based on their category and requirements, following the same principles as our Go testing strategy.
