<!-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk> -->
<!--  -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# TrustedAI Acceptance Tests

This directory contains acceptance tests for the TrustedAI service. These tests are designed to run against a live TrustedAI API instance and test both the ControlPlane (ConnectRPC) and DataPlane (HTTP) APIs from an end-user perspective.

## Prerequisites

1. **Running TrustedAI Instance**: The tests expect a TrustedAI service to be already running and accessible.

2. **Fresh Environment**: These tests assume a fresh environment (as created by the bootstrap process) with only the system administrator user.

3. **Docker Compose**: Required for the automated test runner to extract the bootstrap token.

4. **jq**: Required for parsing JSON logs to extract the bootstrap token.

## Environment Setup

### Using Docker Compose (Recommended)

1. Start the test environment:
   ```bash
   cd testing/acceptance
   docker-compose up -d
   ```

2. Wait for the service to bootstrap (watch logs until you see "Bootstrap completed successfully"):
   ```bash
   docker-compose logs -f trustedai
   ```

3. Run tests using mise (automatically extracts token):
   ```bash
   mise run test:acceptance
   ```

### Manual Configuration

If you prefer to run tests manually, set the following environment variables:

- `TRUSTEDAI_BOOTSTRAP_TOKEN` (required): The API token from the bootstrap process
- `TRUSTEDAI_CONTROLPLANE_URL` (optional): ControlPlane API URL (default: http://localhost:9999)
- `TRUSTEDAI_DATAPLANE_URL` (optional): DataPlane API URL (default: http://localhost:8081)

Note: The URLs match the docker-compose port mappings where port 9999 is ControlPlane and port 8081 is DataPlane.

Extract the bootstrap token manually:
```bash
cd testing/acceptance
docker-compose logs --no-color --no-log-prefix trustedai | grep "Bootstrap completed successfully" | jq -r .token
```

## Running the Tests

The recommended approach is to use the `mise` command which automatically:
- Extracts the bootstrap token from docker-compose logs
- Sets all required environment variables
- Runs the specified tests

### Run All Acceptance Tests
```bash
mise run test:acceptance
```

### Run Specific Test Groups
```bash
# Run only ControlPlane tests
mise run test:acceptance "TestTrustedAIAcceptance/ControlPlane"

# Run only DataPlane tests  
mise run test:acceptance "TestTrustedAIAcceptance/DataPlane"

# Run only API key creation test
mise run test:acceptance "TestTrustedAIAcceptance/ControlPlane/CreateAPIKey"

# Run with verbose output
mise run test:acceptance "TestTrustedAIAcceptance/ControlPlane/CreateAPIKey" -v
```

### Available Test Paths
```bash
# Main test groups
"TestTrustedAIAcceptance/ControlPlane"
"TestTrustedAIAcceptance/DataPlane"

# Specific ControlPlane tests  
"TestTrustedAIAcceptance/ControlPlane/Authentication"
"TestTrustedAIAcceptance/ControlPlane/CreateAPIKey" 
"TestTrustedAIAcceptance/ControlPlane/UserManagement"
"TestTrustedAIAcceptance/ControlPlane/OrganizationManagement"
"TestTrustedAIAcceptance/ControlPlane/TokenManagement"
"TestTrustedAIAcceptance/ControlPlane/OIDCConfiguration"

# Specific DataPlane tests
"TestTrustedAIAcceptance/DataPlane/Authentication"
"TestTrustedAIAcceptance/DataPlane/OpenAICompatibility"
"TestTrustedAIAcceptance/DataPlane/AnthropicCompatibility"
```

### Manual Execution
If you need to run tests manually (requires setting environment variables):
```bash
# Set environment variables first
export TRUSTEDAI_BOOTSTRAP_TOKEN="your_token_here"
export TRUSTEDAI_CONTROLPLANE_URL="http://localhost:9999"
export TRUSTEDAI_DATAPLANE_URL="http://localhost:8081"

# Run tests
go test -v -count=1 --tags=acceptance ./testing/acceptance/... -run "TestTrustedAIAcceptance/ControlPlane"
```

## Test Structure

The acceptance tests are organized in a nested structure using `t.Run()`:

```
TestTrustedAIAcceptance/
├── ControlPlane/
│   ├── Authentication
│   ├── CreateAPIKey
│   ├── UserManagement/
│   │   ├── GetCurrentUser
│   │   └── ListUsersByOrganization
│   ├── OrganizationManagement/
│   │   ├── ListOrganizations
│   │   └── GetOrganizationByName
│   ├── TokenManagement/
│   │   └── ListUserTokens
│   └── OIDCConfiguration
└── DataPlane/
    ├── Authentication
    ├── OpenAICompatibility/
    │   └── ModelsEndpoint
    └── AnthropicCompatibility/
        └── MessagesEndpoint
```

## Key Test Features

### Token Management
- **Bootstrap Token Usage**: Tests start by authenticating with the provided bootstrap token
- **New Token Creation**: Creates a new API token during testing for use in subsequent tests
- **Token Validation**: Verifies that newly created tokens work for authentication

### Fresh Environment Assumptions
- Expects exactly one organization (the system organization)
- Expects exactly one user (the bootstrap admin user)
- Tests are designed for clean, newly bootstrapped environments

## Build Tags

The acceptance tests use the `//go:build acceptance` build tag to ensure they:
- Are only run when explicitly requested
- Don't interfere with unit or integration tests
- Can be excluded from regular development test runs

## Important Notes

1. **Environment Isolation**: These tests expect a fresh environment and may fail on systems with existing data.

2. **External Dependencies**: Tests communicate with real API endpoints and databases, making them slower than unit tests.

3. **Token Security**: Be careful with bootstrap tokens in CI/CD environments. Consider using temporary environments for testing.

4. **Test Order**: Tests are designed to be independent, but the `CreateAPIKey` test must run before other tests that depend on `config.CreatedToken`.

## Troubleshooting

### Common Issues

1. **Could not extract bootstrap token from logs**:
   ```
   Error: Could not extract bootstrap token from logs
   ```
   Solution: Ensure TrustedAI service has started and completed bootstrapping. Check logs with:
   ```bash
   docker-compose logs trustedai | grep "Bootstrap completed successfully"
   ```

2. **Connection Refused**:
   ```
   Error: dial tcp [::1]:9999: connect: connection refused
   ```
   Solution: Ensure TrustedAI service is running and accessible. Check service status:
   ```bash
   docker-compose ps
   ```

3. **Authentication Failed**:
   ```
   Error: Failed to authenticate with bootstrap token
   ```
   Solution: Verify the bootstrap token was extracted correctly and hasn't expired (24-hour default).

4. **No tests to run**:
   ```
   testing: warning: no tests to run
   ```
   Solution: Use the correct test path. For example, use `"TestTrustedAIAcceptance/ControlPlane/CreateAPIKey"` instead of just `"CreateAPIKey"`.

### Debug Mode

For additional debugging:
```bash
# Check docker-compose status
docker-compose ps

# View service logs
docker-compose logs trustedai

# Run tests with verbose output
mise run test:acceptance "TestTrustedAIAcceptance/ControlPlane" -v

# Manual token extraction for debugging
cd testing/acceptance
docker-compose logs --no-color --no-log-prefix trustedai | grep "Bootstrap completed successfully" | jq -r .token
```