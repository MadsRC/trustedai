#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
# SPDX-License-Identifier: AGPL-3.0-only
#MISE description="Bootstrap the LLMGW application for development. Run this after a first boot of the app - requires admin API token as first param"

# Bootstrap script for llmgw development environment
# Creates test organization with OIDC SSO and updates system organization

set -e

if [ $# -ne 1 ]; then
    echo "Usage: $0 <api_key>"
    echo "Example: $0 your-api-key-here"
    exit 1
fi

API_KEY="$1"
SERVER="localhost:9999"
PROTO_PATH="proto/madsrc/llmgw/v1/iam.proto"

echo "Bootstrapping llmgw development environment..."

# Create testorg organization with OIDC SSO
echo "Creating testorg organization..."
grpcurl -plaintext -proto "$PROTO_PATH" \
    -H "Authorization: Bearer $API_KEY" \
    -d '{
        "organization": {
            "name": "testorg",
            "display_name": "Test Organization",
            "is_system": false,
            "sso_type": "oidc",
            "sso_config": "{\"client_id\":\"client01\",\"client_secret\":\"tJJtLCazP5JoR5LHWhii9Am8QXdLUVGI\",\"issuer\":\"http://localhost:8080/realms/testrealm01\",\"redirect_uri\":\"http://localhost:9999/sso/oidc/testorg/callback\",\"scopes\":[\"openid\",\"profile\",\"email\"]}"
        }
    }' \
    "$SERVER" llmgw.v1.IAMService/CreateOrganization

# Get system organization ID
echo "Getting system organization..."
SYSTEM_ORG=$(grpcurl -plaintext -proto "$PROTO_PATH" \
    -H "Authorization: Bearer $API_KEY" \
    -d '{"name": "system"}' \
    "$SERVER" llmgw.v1.IAMService/GetOrganizationByName)

# Extract system organization ID from response using jq
SYSTEM_ORG_ID=$(echo "$SYSTEM_ORG" | jq -r '.organization.id')

if [ -z "$SYSTEM_ORG_ID" ] || [ "$SYSTEM_ORG_ID" = "null" ]; then
    echo "Error: Could not find system organization"
    exit 1
fi

echo "Updating system organization SSO configuration..."
grpcurl -plaintext -proto "$PROTO_PATH" \
    -H "Authorization: Bearer $API_KEY" \
    -d '{
        "organization": {
            "id": "'"$SYSTEM_ORG_ID"'",
            "name": "system",
            "is_system": true,
            "sso_type": "oidc",
            "sso_config": "{\"client_id\":\"client01\",\"client_secret\":\"F9RVr1gvjNfi5mct6hij4rrwbrFH4jPI\",\"issuer\":\"http://localhost:8080/realms/testrealm02\",\"redirect_uri\":\"http://localhost:9999/sso/oidc/system/callback\",\"scopes\":[\"openid\",\"profile\",\"email\"]}"
        }
    }' \
    "$SERVER" llmgw.v1.IAMService/UpdateOrganization

# Get admin@localhost user
echo "Getting admin@localhost user..."
ADMIN_USER=$(grpcurl -plaintext -proto "$PROTO_PATH" \
    -H "Authorization: Bearer $API_KEY" \
    -d '{"email": "admin@localhost"}' \
    "$SERVER" llmgw.v1.IAMService/GetUserByEmail)

# Extract user information from response using jq
ADMIN_USER_ID=$(echo "$ADMIN_USER" | jq -r '.user.id')
ADMIN_USER_NAME=$(echo "$ADMIN_USER" | jq -r '.user.name')
ADMIN_USER_ORG_ID=$(echo "$ADMIN_USER" | jq -r '.user.organizationId')
ADMIN_USER_PROVIDER=$(echo "$ADMIN_USER" | jq -r '.user.provider')
ADMIN_USER_SYSTEM_ADMIN=$(echo "$ADMIN_USER" | jq -r '.user.systemAdmin')

if [ -z "$ADMIN_USER_ID" ] || [ "$ADMIN_USER_ID" = "null" ]; then
    echo "Error: Could not find admin@localhost user"
    exit 1
fi

echo "Updating admin@localhost user external ID..."
grpcurl -plaintext -proto "$PROTO_PATH" \
    -H "Authorization: Bearer $API_KEY" \
    -d '{
        "user": {
            "id": "'"$ADMIN_USER_ID"'",
            "email": "admin@localhost",
            "name": "'"$ADMIN_USER_NAME"'",
            "organization_id": "'"$ADMIN_USER_ORG_ID"'",
            "external_id": "07c7c64b-9a3e-451e-833d-2b86d684a4ab",
            "provider": "'"$ADMIN_USER_PROVIDER"'",
            "system_admin": '"$ADMIN_USER_SYSTEM_ADMIN"'
        }
    }' \
    "$SERVER" llmgw.v1.IAMService/UpdateUser

echo "Bootstrap completed successfully!"
