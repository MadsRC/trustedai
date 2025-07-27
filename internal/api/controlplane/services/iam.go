// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/MadsRC/trustedai"
	trustedaiv1 "github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1"
	"github.com/MadsRC/trustedai/gen/proto/madsrc/trustedai/v1/trustedaiv1connect"
	"github.com/MadsRC/trustedai/internal/api/controlplane/auth"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Ensure Iam implements the required interfaces
var _ trustedaiv1connect.IAMServiceHandler = (*Iam)(nil)

// User Service Methods

// CreateUser creates a new user
func (s *Iam) CreateUser(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceCreateUserRequest],
) (*connect.Response[trustedaiv1.IAMServiceCreateUserResponse], error) {
	s.options.Logger.Debug("[IAMService] CreateUser invoked", "email", req.Msg.GetUser().GetEmail())

	// Validate request
	if req.Msg.GetUser() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user is required"))
	}

	if req.Msg.GetUser().GetEmail() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: email is required"))
	}

	if req.Msg.GetUser().GetOrganizationId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization ID is required"))
	}

	// Check if organization exists
	_, err := s.options.OrganizationRepository.Get(ctx, req.Msg.GetUser().GetOrganizationId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
		}
		s.options.Logger.Error("Failed to get organization", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization: %w", err))
	}

	// Generate ID if not provided
	userID := req.Msg.GetUser().GetId()
	if userID == "" {
		userID = func() string { id, _ := uuid.NewV7(); return id.String() }()
	}

	// Set creation time if not provided
	createdAt := time.Now().UTC()
	if req.Msg.GetUser().GetCreatedAt() != nil {
		createdAt = req.Msg.GetUser().GetCreatedAt().AsTime()
	}

	// Set last login if not provided
	lastLogin := time.Now().UTC()
	if req.Msg.GetUser().GetLastLogin() != nil {
		lastLogin = req.Msg.GetUser().GetLastLogin().AsTime()
	}

	// Create user domain object
	user := &trustedai.User{
		ID:             userID,
		Email:          req.Msg.GetUser().GetEmail(),
		Name:           req.Msg.GetUser().GetName(),
		OrganizationID: req.Msg.GetUser().GetOrganizationId(),
		ExternalID:     req.Msg.GetUser().GetExternalId(),
		Provider:       req.Msg.GetUser().GetProvider(),
		SystemAdmin:    req.Msg.GetUser().GetSystemAdmin(),
		CreatedAt:      createdAt,
		LastLogin:      lastLogin,
	}

	// Create user in repository
	err = s.options.UserRepository.Create(ctx, user)
	if err != nil {
		if errors.Is(err, trustedai.ErrDuplicateEntry) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("iam service: user already exists: %w", err))
		}
		s.options.Logger.Error("Failed to create user", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to create user: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceCreateUserResponse{
		User: userToProto(user),
	}

	return connect.NewResponse(response), nil
}

// GetUser retrieves a user by ID
func (s *Iam) GetUser(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceGetUserRequest],
) (*connect.Response[trustedaiv1.IAMServiceGetUserResponse], error) {
	s.options.Logger.Debug("[IAMService] GetUser invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user ID is required"))
	}

	// Get user from repository
	user, err := s.options.UserRepository.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceGetUserResponse{
		User: userToProto(user),
	}

	return connect.NewResponse(response), nil
}

// GetUserByEmail retrieves a user by email
func (s *Iam) GetUserByEmail(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceGetUserByEmailRequest],
) (*connect.Response[trustedaiv1.IAMServiceGetUserByEmailResponse], error) {
	s.options.Logger.Debug("[IAMService] GetUserByEmail invoked", "email", req.Msg.GetEmail())

	// Validate request
	if req.Msg.GetEmail() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: email is required"))
	}

	// Get user from repository
	user, err := s.options.UserRepository.GetByEmail(ctx, req.Msg.GetEmail())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user by email", "error", err, "email", req.Msg.GetEmail())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user by email: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceGetUserByEmailResponse{
		User: userToProto(user),
	}

	return connect.NewResponse(response), nil
}

// GetUserByExternalID retrieves a user by external ID and provider
func (s *Iam) GetUserByExternalID(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceGetUserByExternalIDRequest],
) (*connect.Response[trustedaiv1.IAMServiceGetUserByExternalIDResponse], error) {
	s.options.Logger.Debug("[IAMService] GetUserByExternalID invoked",
		"provider", req.Msg.GetProvider(), "externalID", req.Msg.GetExternalId())

	// Validate request
	if req.Msg.GetProvider() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: provider is required"))
	}

	if req.Msg.GetExternalId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: external ID is required"))
	}

	// Get user from repository
	user, err := s.options.UserRepository.GetByExternalID(ctx, req.Msg.GetProvider(), req.Msg.GetExternalId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user by external ID",
			"error", err, "provider", req.Msg.GetProvider(), "externalID", req.Msg.GetExternalId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user by external ID: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceGetUserByExternalIDResponse{
		User: userToProto(user),
	}

	return connect.NewResponse(response), nil
}

// getUserFromConnection extracts the authenticated user from the connection context
// This function checks both session-based authentication (SSO) and API key authentication
func (s *Iam) getUserFromConnection(ctx context.Context) (*trustedai.User, error) {
	// Check for session-based authentication (SSO) first
	if session := auth.SessionFromContext(ctx); session != nil {
		s.options.Logger.Debug("[IAMService] Found session user",
			"userID", session.User.ID, "email", session.User.Email, "authMethod", "session")
		return session.User, nil
	}

	// Fall back to API key authentication
	if apiKeyUser := auth.UserFromContext(ctx); apiKeyUser != nil {
		s.options.Logger.Debug("[IAMService] Found API key user",
			"userID", apiKeyUser.ID, "email", apiKeyUser.Email, "authMethod", "apikey")
		return apiKeyUser, nil
	}

	// No authentication found
	return nil, connect.NewError(connect.CodeUnauthenticated,
		errors.New("iam service: no authenticated user found"))
}

// GetCurrentUser retrieves the current authenticated user (from session or API key)
func (s *Iam) GetCurrentUser(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceGetCurrentUserRequest],
) (*connect.Response[trustedaiv1.IAMServiceGetCurrentUserResponse], error) {
	s.options.Logger.Debug("[IAMService] GetCurrentUser invoked")

	user, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Return the user
	response := &trustedaiv1.IAMServiceGetCurrentUserResponse{
		User: userToProto(user),
	}

	return connect.NewResponse(response), nil
}

// ListUsersByOrganization retrieves users in an organization if the requester has permission
func (s *Iam) ListUsersByOrganization(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceListUsersByOrganizationRequest],
) (*connect.Response[trustedaiv1.IAMServiceListUsersByOrganizationResponse], error) {
	s.options.Logger.Debug("[IAMService] ListUsersByOrganization invoked", "organizationID", req.Msg.GetOrganizationId())

	// Validate request
	if req.Msg.GetOrganizationId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization ID is required"))
	}

	// Get authenticated user
	currentUser, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Check if organization exists
	_, err = s.options.OrganizationRepository.Get(ctx, req.Msg.GetOrganizationId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
		}
		s.options.Logger.Error("Failed to get organization", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization: %w", err))
	}

	// Get users from repository with authorization check
	users, err := s.options.UserRepository.ListByOrganizationForUser(ctx, currentUser, req.Msg.GetOrganizationId())
	if err != nil {
		if errors.Is(err, trustedai.ErrUnauthorized) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("iam service: insufficient permissions to list users in this organization"))
		}
		s.options.Logger.Error("Failed to list users by organization", "error", err, "organizationID", req.Msg.GetOrganizationId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to list users by organization: %w", err))
	}

	// Convert users to proto
	protoUsers := make([]*trustedaiv1.User, 0, len(users))
	for _, user := range users {
		protoUsers = append(protoUsers, userToProto(user))
	}

	// Return response
	response := &trustedaiv1.IAMServiceListUsersByOrganizationResponse{
		Users: protoUsers,
	}

	return connect.NewResponse(response), nil
}

// UpdateUser updates an existing user
func (s *Iam) UpdateUser(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceUpdateUserRequest],
) (*connect.Response[trustedaiv1.IAMServiceUpdateUserResponse], error) {
	s.options.Logger.Debug("[IAMService] UpdateUser invoked", "id", req.Msg.GetUser().GetId())

	// Validate request
	if req.Msg.GetUser() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user is required"))
	}

	if req.Msg.GetUser().GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user ID is required"))
	}

	// Get existing user
	existingUser, err := s.options.UserRepository.Get(ctx, req.Msg.GetUser().GetId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user for update", "error", err, "id", req.Msg.GetUser().GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user for update: %w", err))
	}

	// Update fields if provided
	if req.Msg.GetUser().GetEmail() != "" {
		existingUser.Email = req.Msg.GetUser().GetEmail()
	}

	if req.Msg.GetUser().GetName() != "" {
		existingUser.Name = req.Msg.GetUser().GetName()
	}

	if req.Msg.GetUser().GetOrganizationId() != "" && req.Msg.GetUser().GetOrganizationId() != existingUser.OrganizationID {
		// Check if new organization exists
		_, err := s.options.OrganizationRepository.Get(ctx, req.Msg.GetUser().GetOrganizationId())
		if err != nil {
			if errors.Is(err, trustedai.ErrNotFound) {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
			}
			s.options.Logger.Error("Failed to get organization", "error", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization: %w", err))
		}
		existingUser.OrganizationID = req.Msg.GetUser().GetOrganizationId()
	}

	if req.Msg.GetUser().GetExternalId() != "" {
		existingUser.ExternalID = req.Msg.GetUser().GetExternalId()
	}

	if req.Msg.GetUser().GetProvider() != "" {
		existingUser.Provider = req.Msg.GetUser().GetProvider()
	}

	// Update system admin status directly
	existingUser.SystemAdmin = req.Msg.GetUser().GetSystemAdmin()

	// Update last login if provided
	if req.Msg.GetUser().GetLastLogin() != nil {
		existingUser.LastLogin = req.Msg.GetUser().GetLastLogin().AsTime()
	}

	// Update user in repository
	err = s.options.UserRepository.Update(ctx, existingUser)
	if err != nil {
		s.options.Logger.Error("Failed to update user", "error", err, "id", req.Msg.GetUser().GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to update user: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceUpdateUserResponse{
		User: userToProto(existingUser),
	}

	return connect.NewResponse(response), nil
}

// DeleteUser removes a user
func (s *Iam) DeleteUser(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceDeleteUserRequest],
) (*connect.Response[trustedaiv1.IAMServiceDeleteUserResponse], error) {
	s.options.Logger.Debug("[IAMService] DeleteUser invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user ID is required"))
	}

	// Check if user exists
	_, err := s.options.UserRepository.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user for deletion", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user for deletion: %w", err))
	}

	// Delete user
	err = s.options.UserRepository.Delete(ctx, req.Msg.GetId())
	if err != nil {
		s.options.Logger.Error("Failed to delete user", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to delete user: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceDeleteUserResponse{
		Success: true,
	}

	return connect.NewResponse(response), nil
}

// Organization Service Methods

// CreateOrganization creates a new organization (system admins only)
func (s *Iam) CreateOrganization(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceCreateOrganizationRequest],
) (*connect.Response[trustedaiv1.IAMServiceCreateOrganizationResponse], error) {
	s.options.Logger.Debug("[IAMService] CreateOrganization invoked", "name", req.Msg.GetOrganization().GetName())

	// Get authenticated user
	currentUser, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Only system admins can create organizations
	if !currentUser.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("iam service: only system administrators can create organizations"))
	}

	// Validate request
	if req.Msg.GetOrganization() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization is required"))
	}

	if req.Msg.GetOrganization().GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization name is required"))
	}

	// Generate ID if not provided
	orgID := req.Msg.GetOrganization().GetId()
	if orgID == "" {
		orgID = func() string { id, _ := uuid.NewV7(); return id.String() }()
	}

	// Set creation time if not provided
	createdAt := time.Now().UTC()
	if req.Msg.GetOrganization().GetCreatedAt() != nil {
		createdAt = req.Msg.GetOrganization().GetCreatedAt().AsTime()
	}

	// Parse SSO config if provided
	var ssoConfig map[string]any
	if req.Msg.GetOrganization().GetSsoConfig() != "" {
		ssoConfig = make(map[string]any)
		err := json.Unmarshal([]byte(req.Msg.GetOrganization().GetSsoConfig()), &ssoConfig)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("iam service: invalid SSO config format: %w", err))
		}
	}

	// Create organization domain object
	org := &trustedai.Organization{
		ID:          orgID,
		Name:        req.Msg.GetOrganization().GetName(),
		DisplayName: req.Msg.GetOrganization().GetDisplayName(),
		IsSystem:    req.Msg.GetOrganization().GetIsSystem(),
		CreatedAt:   createdAt,
		SSOType:     req.Msg.GetOrganization().GetSsoType(),
		SSOConfig:   ssoConfig,
	}

	// Create organization in repository
	err = s.options.OrganizationRepository.Create(ctx, org)
	if err != nil {
		if errors.Is(err, trustedai.ErrDuplicateEntry) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("iam service: organization already exists: %w", err))
		}
		s.options.Logger.Error("Failed to create organization", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to create organization: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceCreateOrganizationResponse{
		Organization: organizationToProto(org),
	}

	return connect.NewResponse(response), nil
}

// GetOrganization retrieves an organization by ID
func (s *Iam) GetOrganization(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceGetOrganizationRequest],
) (*connect.Response[trustedaiv1.IAMServiceGetOrganizationResponse], error) {
	s.options.Logger.Debug("[IAMService] GetOrganization invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization ID is required"))
	}

	// Get organization from repository
	org, err := s.options.OrganizationRepository.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
		}
		s.options.Logger.Error("Failed to get organization", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceGetOrganizationResponse{
		Organization: organizationToProto(org),
	}

	return connect.NewResponse(response), nil
}

// GetOrganizationByName retrieves an organization by name
func (s *Iam) GetOrganizationByName(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceGetOrganizationByNameRequest],
) (*connect.Response[trustedaiv1.IAMServiceGetOrganizationByNameResponse], error) {
	s.options.Logger.Debug("[IAMService] GetOrganizationByName invoked", "name", req.Msg.GetName())

	// Validate request
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization name is required"))
	}

	// Get organization from repository
	org, err := s.options.OrganizationRepository.GetByName(ctx, req.Msg.GetName())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
		}
		s.options.Logger.Error("Failed to get organization by name", "error", err, "name", req.Msg.GetName())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization by name: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceGetOrganizationByNameResponse{
		Organization: organizationToProto(org),
	}

	return connect.NewResponse(response), nil
}

// ListOrganizations retrieves organizations visible to the authenticated user
func (s *Iam) ListOrganizations(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceListOrganizationsRequest],
) (*connect.Response[trustedaiv1.IAMServiceListOrganizationsResponse], error) {
	s.options.Logger.Debug("[IAMService] ListOrganizations invoked")

	// Get authenticated user
	currentUser, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Get organizations visible to this user
	orgs, err := s.options.OrganizationRepository.ListForUser(ctx, currentUser)
	if err != nil {
		s.options.Logger.Error("Failed to list organizations for user", "error", err, "userID", currentUser.ID)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to list organizations: %w", err))
	}

	// Convert organizations to proto
	protoOrgs := make([]*trustedaiv1.Organization, 0, len(orgs))
	for _, org := range orgs {
		protoOrgs = append(protoOrgs, organizationToProto(org))
	}

	// Return response
	response := &trustedaiv1.IAMServiceListOrganizationsResponse{
		Organizations: protoOrgs,
	}

	return connect.NewResponse(response), nil
}

// UpdateOrganization updates an existing organization (system admins only)
func (s *Iam) UpdateOrganization(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceUpdateOrganizationRequest],
) (*connect.Response[trustedaiv1.IAMServiceUpdateOrganizationResponse], error) {
	s.options.Logger.Debug("[IAMService] UpdateOrganization invoked", "id", req.Msg.GetOrganization().GetId())

	// Get authenticated user
	currentUser, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Only system admins can update organizations
	if !currentUser.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("iam service: only system administrators can update organizations"))
	}

	// Validate request
	if req.Msg.GetOrganization() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization is required"))
	}

	if req.Msg.GetOrganization().GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization ID is required"))
	}

	// Get existing organization
	existingOrg, err := s.options.OrganizationRepository.Get(ctx, req.Msg.GetOrganization().GetId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
		}
		s.options.Logger.Error("Failed to get organization for update", "error", err, "id", req.Msg.GetOrganization().GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization for update: %w", err))
	}

	// Update fields if provided
	if req.Msg.GetOrganization().GetName() != "" {
		existingOrg.Name = req.Msg.GetOrganization().GetName()
	}

	if req.Msg.GetOrganization().GetDisplayName() != "" {
		existingOrg.DisplayName = req.Msg.GetOrganization().GetDisplayName()
	}

	// Update system status directly
	existingOrg.IsSystem = req.Msg.GetOrganization().GetIsSystem()

	if req.Msg.GetOrganization().GetSsoType() != "" {
		existingOrg.SSOType = req.Msg.GetOrganization().GetSsoType()
	}

	// Parse and update SSO config if provided
	if req.Msg.GetOrganization().GetSsoConfig() != "" {
		ssoConfig := make(map[string]any)
		err := json.Unmarshal([]byte(req.Msg.GetOrganization().GetSsoConfig()), &ssoConfig)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("iam service: invalid SSO config format: %w", err))
		}
		existingOrg.SSOConfig = ssoConfig
	}

	// Update organization in repository
	err = s.options.OrganizationRepository.Update(ctx, existingOrg)
	if err != nil {
		if errors.Is(err, trustedai.ErrDuplicateEntry) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("iam service: organization name already exists: %w", err))
		}
		s.options.Logger.Error("Failed to update organization", "error", err, "id", req.Msg.GetOrganization().GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to update organization: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceUpdateOrganizationResponse{
		Organization: organizationToProto(existingOrg),
	}

	return connect.NewResponse(response), nil
}

// DeleteOrganization removes an organization (system admins only)
func (s *Iam) DeleteOrganization(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceDeleteOrganizationRequest],
) (*connect.Response[trustedaiv1.IAMServiceDeleteOrganizationResponse], error) {
	s.options.Logger.Debug("[IAMService] DeleteOrganization invoked", "id", req.Msg.GetId())

	// Get authenticated user
	currentUser, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Only system admins can delete organizations
	if !currentUser.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("iam service: only system administrators can delete organizations"))
	}

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization ID is required"))
	}

	// Check if organization exists
	_, err = s.options.OrganizationRepository.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
		}
		s.options.Logger.Error("Failed to get organization for deletion", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization for deletion: %w", err))
	}

	// Check if organization has users
	users, err := s.options.UserRepository.ListByOrganization(ctx, req.Msg.GetId())
	if err != nil {
		s.options.Logger.Error("Failed to list users for organization", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to check if organization has users: %w", err))
	}

	if len(users) > 0 && !req.Msg.GetForce() {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("iam service: organization has %d users; use force=true to delete anyway", len(users)))
	}

	// Delete organization
	err = s.options.OrganizationRepository.Delete(ctx, req.Msg.GetId())
	if err != nil {
		s.options.Logger.Error("Failed to delete organization", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to delete organization: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceDeleteOrganizationResponse{
		Success: true,
	}

	return connect.NewResponse(response), nil
}

// Token Service Methods

// CreateToken creates a new API token for a user
func (s *Iam) CreateToken(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceCreateTokenRequest],
) (*connect.Response[trustedaiv1.IAMServiceCreateTokenResponse], error) {
	s.options.Logger.Debug("[IAMService] CreateToken invoked", "userID", req.Msg.GetUserId())

	// Validate request
	if req.Msg.GetUserId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user ID is required"))
	}

	// Check if user exists
	_, err := s.options.UserRepository.Get(ctx, req.Msg.GetUserId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user for token creation", "error", err, "userID", req.Msg.GetUserId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user for token creation: %w", err))
	}

	// Set expiration time if not provided
	expiresAt := time.Now().UTC().AddDate(1, 0, 0) // Default: 1 year
	if req.Msg.GetExpiresAt() != nil {
		expiresAt = req.Msg.GetExpiresAt().AsTime()
	}

	// Create token
	token, rawToken, err := s.options.TokenRepository.CreateToken(
		ctx,
		req.Msg.GetUserId(),
		req.Msg.GetDescription(),
		expiresAt,
	)
	if err != nil {
		s.options.Logger.Error("Failed to create token", "error", err, "userID", req.Msg.GetUserId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to create token: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceCreateTokenResponse{
		Token: &trustedaiv1.APIToken{
			Id:          token.ID,
			UserId:      token.UserID,
			Description: token.Description,
			CreatedAt:   timestamppb.New(token.CreatedAt),
			ExpiresAt:   timestamppb.New(token.ExpiresAt),
		},
		RawToken: rawToken,
	}

	if token.LastUsedAt != nil && !token.LastUsedAt.IsZero() {
		response.Token.LastUsedAt = timestamppb.New(*token.LastUsedAt)
	}

	return connect.NewResponse(response), nil
}

// ListUserTokens retrieves tokens for a user if the requester has permission
func (s *Iam) ListUserTokens(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceListUserTokensRequest],
) (*connect.Response[trustedaiv1.IAMServiceListUserTokensResponse], error) {
	s.options.Logger.Debug("[IAMService] ListUserTokens invoked", "userID", req.Msg.GetUserId())

	// Validate request
	if req.Msg.GetUserId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user ID is required"))
	}

	// Get authenticated user
	currentUser, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Check if user exists
	_, err = s.options.UserRepository.Get(ctx, req.Msg.GetUserId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user for token listing", "error", err, "userID", req.Msg.GetUserId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user for token listing: %w", err))
	}

	// Get tokens from repository with authorization check
	tokens, err := s.options.TokenRepository.ListUserTokensForUser(ctx, currentUser, req.Msg.GetUserId())
	if err != nil {
		if errors.Is(err, trustedai.ErrUnauthorized) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("iam service: insufficient permissions to list tokens for this user"))
		}
		s.options.Logger.Error("Failed to list user tokens", "error", err, "userID", req.Msg.GetUserId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to list user tokens: %w", err))
	}

	// Convert tokens to proto
	protoTokens := make([]*trustedaiv1.APIToken, 0, len(tokens))
	for _, token := range tokens {
		protoToken := &trustedaiv1.APIToken{
			Id:          token.ID,
			UserId:      token.UserID,
			Description: token.Description,
			CreatedAt:   timestamppb.New(token.CreatedAt),
			ExpiresAt:   timestamppb.New(token.ExpiresAt),
		}

		if token.LastUsedAt != nil && !token.LastUsedAt.IsZero() {
			protoToken.LastUsedAt = timestamppb.New(*token.LastUsedAt)
		}

		protoTokens = append(protoTokens, protoToken)
	}

	// Return response
	response := &trustedaiv1.IAMServiceListUserTokensResponse{
		Tokens: protoTokens,
	}

	return connect.NewResponse(response), nil
}

// RevokeToken invalidates a token if the requester has permission
func (s *Iam) RevokeToken(
	ctx context.Context,
	req *connect.Request[trustedaiv1.IAMServiceRevokeTokenRequest],
) (*connect.Response[trustedaiv1.IAMServiceRevokeTokenResponse], error) {
	s.options.Logger.Debug("[IAMService] RevokeToken invoked", "tokenID", req.Msg.GetTokenId())

	// Validate request
	if req.Msg.GetTokenId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: token ID is required"))
	}

	// Get authenticated user
	currentUser, err := s.getUserFromConnection(ctx)
	if err != nil {
		return nil, err
	}

	// Revoke token with authorization check
	err = s.options.TokenRepository.RevokeTokenForUser(ctx, currentUser, req.Msg.GetTokenId())
	if err != nil {
		if errors.Is(err, trustedai.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: token not found: %w", err))
		}
		if errors.Is(err, trustedai.ErrUnauthorized) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("iam service: insufficient permissions to revoke this token"))
		}
		s.options.Logger.Error("Failed to revoke token", "error", err, "tokenID", req.Msg.GetTokenId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to revoke token: %w", err))
	}

	// Return response
	response := &trustedaiv1.IAMServiceRevokeTokenResponse{
		Success: true,
	}

	return connect.NewResponse(response), nil
}

// Helper functions

// userToProto converts a domain user to a protobuf user
func userToProto(user *trustedai.User) *trustedaiv1.User {
	protoUser := &trustedaiv1.User{
		Id:             user.ID,
		Email:          user.Email,
		Name:           user.Name,
		OrganizationId: user.OrganizationID,
		ExternalId:     user.ExternalID,
		Provider:       user.Provider,
		SystemAdmin:    user.SystemAdmin,
		CreatedAt:      timestamppb.New(user.CreatedAt),
		LastLogin:      timestamppb.New(user.LastLogin),
	}

	return protoUser
}

// organizationToProto converts a domain organization to a protobuf organization
func organizationToProto(org *trustedai.Organization) *trustedaiv1.Organization {
	// Convert SSOConfig map to JSON string
	ssoConfig := ""
	if org.SSOConfig != nil {
		configBytes, err := json.Marshal(org.SSOConfig)
		if err == nil {
			ssoConfig = string(configBytes)
		}
	}

	protoOrg := &trustedaiv1.Organization{
		Id:          org.ID,
		Name:        org.Name,
		DisplayName: org.DisplayName,
		IsSystem:    org.IsSystem,
		CreatedAt:   timestamppb.New(org.CreatedAt),
		SsoType:     org.SSOType,
		SsoConfig:   ssoConfig,
	}

	return protoOrg
}
