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

	"codeberg.org/MadsRC/llmgw"
	llmgwv1 "codeberg.org/MadsRC/llmgw/gen/proto/madsrc/llmgw/v1"
	"codeberg.org/MadsRC/llmgw/gen/proto/madsrc/llmgw/v1/llmgwv1connect"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Ensure Iam implements the required interfaces
var _ llmgwv1connect.IAMServiceHandler = (*Iam)(nil)

// User Service Methods

// CreateUser creates a new user
func (s *Iam) CreateUser(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceUserCreateRequest],
) (*connect.Response[llmgwv1.IAMServiceUserCreateResponse], error) {
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
		if errors.Is(err, llmgw.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
		}
		s.options.Logger.Error("Failed to get organization", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization: %w", err))
	}

	// Generate ID if not provided
	userID := req.Msg.GetUser().GetId()
	if userID == "" {
		userID = uuid.New().String()
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
	user := &llmgw.User{
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
		if errors.Is(err, llmgw.ErrDuplicateEntry) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("iam service: user already exists: %w", err))
		}
		s.options.Logger.Error("Failed to create user", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to create user: %w", err))
	}

	// Return response
	response := &llmgwv1.IAMServiceUserCreateResponse{
		User: userToProto(user),
	}

	return connect.NewResponse(response), nil
}

// GetUser retrieves a user by ID
func (s *Iam) GetUser(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceUserGetRequest],
) (*connect.Response[llmgwv1.IAMServiceUserGetResponse], error) {
	s.options.Logger.Debug("[IAMService] GetUser invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user ID is required"))
	}

	// Get user from repository
	user, err := s.options.UserRepository.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user: %w", err))
	}

	// Return response
	response := &llmgwv1.IAMServiceUserGetResponse{
		User: userToProto(user),
	}

	return connect.NewResponse(response), nil
}

// GetUserByEmail retrieves a user by email
func (s *Iam) GetUserByEmail(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceUserGetByEmailRequest],
) (*connect.Response[llmgwv1.IAMServiceUserGetByEmailResponse], error) {
	s.options.Logger.Debug("[IAMService] GetUserByEmail invoked", "email", req.Msg.GetEmail())

	// Validate request
	if req.Msg.GetEmail() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: email is required"))
	}

	// Get user from repository
	user, err := s.options.UserRepository.GetByEmail(ctx, req.Msg.GetEmail())
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user by email", "error", err, "email", req.Msg.GetEmail())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user by email: %w", err))
	}

	// Return response
	response := &llmgwv1.IAMServiceUserGetByEmailResponse{
		User: userToProto(user),
	}

	return connect.NewResponse(response), nil
}

// GetUserByExternalID retrieves a user by external ID and provider
func (s *Iam) GetUserByExternalID(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceUserGetByExternalIDRequest],
) (*connect.Response[llmgwv1.IAMServiceUserGetByExternalIDResponse], error) {
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
		if errors.Is(err, llmgw.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user by external ID",
			"error", err, "provider", req.Msg.GetProvider(), "externalID", req.Msg.GetExternalId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user by external ID: %w", err))
	}

	// Return response
	response := &llmgwv1.IAMServiceUserGetByExternalIDResponse{
		User: userToProto(user),
	}

	return connect.NewResponse(response), nil
}

// ListUsersByOrganization retrieves all users in an organization
func (s *Iam) ListUsersByOrganization(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceUserListByOrganizationRequest],
) (*connect.Response[llmgwv1.IAMServiceUserListByOrganizationResponse], error) {
	s.options.Logger.Debug("[IAMService] ListUsersByOrganization invoked", "organizationID", req.Msg.GetOrganizationId())

	// Validate request
	if req.Msg.GetOrganizationId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization ID is required"))
	}

	// Check if organization exists
	_, err := s.options.OrganizationRepository.Get(ctx, req.Msg.GetOrganizationId())
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
		}
		s.options.Logger.Error("Failed to get organization", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization: %w", err))
	}

	// Get users from repository
	users, err := s.options.UserRepository.ListByOrganization(ctx, req.Msg.GetOrganizationId())
	if err != nil {
		s.options.Logger.Error("Failed to list users by organization", "error", err, "organizationID", req.Msg.GetOrganizationId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to list users by organization: %w", err))
	}

	// Convert users to proto
	protoUsers := make([]*llmgwv1.User, 0, len(users))
	for _, user := range users {
		protoUsers = append(protoUsers, userToProto(user))
	}

	// Return response
	response := &llmgwv1.IAMServiceUserListByOrganizationResponse{
		Users: protoUsers,
	}

	return connect.NewResponse(response), nil
}

// UpdateUser updates an existing user
func (s *Iam) UpdateUser(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceUserUpdateRequest],
) (*connect.Response[llmgwv1.IAMServiceUserUpdateResponse], error) {
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
		if errors.Is(err, llmgw.ErrNotFound) {
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
			if errors.Is(err, llmgw.ErrNotFound) {
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
	response := &llmgwv1.IAMServiceUserUpdateResponse{
		User: userToProto(existingUser),
	}

	return connect.NewResponse(response), nil
}

// DeleteUser removes a user
func (s *Iam) DeleteUser(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceUserDeleteRequest],
) (*connect.Response[llmgwv1.IAMServiceUserDeleteResponse], error) {
	s.options.Logger.Debug("[IAMService] DeleteUser invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user ID is required"))
	}

	// Check if user exists
	_, err := s.options.UserRepository.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
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
	response := &llmgwv1.IAMServiceUserDeleteResponse{
		Success: true,
	}

	return connect.NewResponse(response), nil
}

// Organization Service Methods

// CreateOrganization creates a new organization
func (s *Iam) CreateOrganization(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceOrganizationCreateRequest],
) (*connect.Response[llmgwv1.IAMServiceOrganizationCreateResponse], error) {
	s.options.Logger.Debug("[IAMService] CreateOrganization invoked", "name", req.Msg.GetOrganization().GetName())

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
		orgID = uuid.New().String()
	}

	// Set creation time if not provided
	createdAt := time.Now().UTC()
	if req.Msg.GetOrganization().GetCreatedAt() != nil {
		createdAt = req.Msg.GetOrganization().GetCreatedAt().AsTime()
	}

	// Parse SSO config if provided
	var ssoConfig map[string]interface{}
	if req.Msg.GetOrganization().GetSsoConfig() != "" {
		ssoConfig = make(map[string]interface{})
		err := json.Unmarshal([]byte(req.Msg.GetOrganization().GetSsoConfig()), &ssoConfig)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("iam service: invalid SSO config format: %w", err))
		}
	}

	// Create organization domain object
	org := &llmgw.Organization{
		ID:          orgID,
		Name:        req.Msg.GetOrganization().GetName(),
		DisplayName: req.Msg.GetOrganization().GetDisplayName(),
		IsSystem:    req.Msg.GetOrganization().GetIsSystem(),
		CreatedAt:   createdAt,
		SSOType:     req.Msg.GetOrganization().GetSsoType(),
		SSOConfig:   ssoConfig,
	}

	// Create organization in repository
	err := s.options.OrganizationRepository.Create(ctx, org)
	if err != nil {
		if errors.Is(err, llmgw.ErrDuplicateEntry) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("iam service: organization already exists: %w", err))
		}
		s.options.Logger.Error("Failed to create organization", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to create organization: %w", err))
	}

	// Return response
	response := &llmgwv1.IAMServiceOrganizationCreateResponse{
		Organization: organizationToProto(org),
	}

	return connect.NewResponse(response), nil
}

// GetOrganization retrieves an organization by ID
func (s *Iam) GetOrganization(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceOrganizationGetRequest],
) (*connect.Response[llmgwv1.IAMServiceOrganizationGetResponse], error) {
	s.options.Logger.Debug("[IAMService] GetOrganization invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization ID is required"))
	}

	// Get organization from repository
	org, err := s.options.OrganizationRepository.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
		}
		s.options.Logger.Error("Failed to get organization", "error", err, "id", req.Msg.GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization: %w", err))
	}

	// Return response
	response := &llmgwv1.IAMServiceOrganizationGetResponse{
		Organization: organizationToProto(org),
	}

	return connect.NewResponse(response), nil
}

// GetOrganizationByName retrieves an organization by name
func (s *Iam) GetOrganizationByName(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceOrganizationGetByNameRequest],
) (*connect.Response[llmgwv1.IAMServiceOrganizationGetByNameResponse], error) {
	s.options.Logger.Debug("[IAMService] GetOrganizationByName invoked", "name", req.Msg.GetName())

	// Validate request
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization name is required"))
	}

	// Get organization from repository
	org, err := s.options.OrganizationRepository.GetByName(ctx, req.Msg.GetName())
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: organization not found: %w", err))
		}
		s.options.Logger.Error("Failed to get organization by name", "error", err, "name", req.Msg.GetName())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get organization by name: %w", err))
	}

	// Return response
	response := &llmgwv1.IAMServiceOrganizationGetByNameResponse{
		Organization: organizationToProto(org),
	}

	return connect.NewResponse(response), nil
}

// ListOrganizations retrieves all organizations
func (s *Iam) ListOrganizations(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceOrganizationListRequest],
) (*connect.Response[llmgwv1.IAMServiceOrganizationListResponse], error) {
	s.options.Logger.Debug("[IAMService] ListOrganizations invoked")

	// Get organizations from repository
	orgs, err := s.options.OrganizationRepository.List(ctx)
	if err != nil {
		s.options.Logger.Error("Failed to list organizations", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to list organizations: %w", err))
	}

	// Convert organizations to proto
	protoOrgs := make([]*llmgwv1.Organization, 0, len(orgs))
	for _, org := range orgs {
		protoOrgs = append(protoOrgs, organizationToProto(org))
	}

	// Return response
	response := &llmgwv1.IAMServiceOrganizationListResponse{
		Organizations: protoOrgs,
	}

	return connect.NewResponse(response), nil
}

// UpdateOrganization updates an existing organization
func (s *Iam) UpdateOrganization(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceOrganizationUpdateRequest],
) (*connect.Response[llmgwv1.IAMServiceOrganizationUpdateResponse], error) {
	s.options.Logger.Debug("[IAMService] UpdateOrganization invoked", "id", req.Msg.GetOrganization().GetId())

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
		if errors.Is(err, llmgw.ErrNotFound) {
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
		ssoConfig := make(map[string]interface{})
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
		if errors.Is(err, llmgw.ErrDuplicateEntry) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("iam service: organization name already exists: %w", err))
		}
		s.options.Logger.Error("Failed to update organization", "error", err, "id", req.Msg.GetOrganization().GetId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to update organization: %w", err))
	}

	// Return response
	response := &llmgwv1.IAMServiceOrganizationUpdateResponse{
		Organization: organizationToProto(existingOrg),
	}

	return connect.NewResponse(response), nil
}

// DeleteOrganization removes an organization
func (s *Iam) DeleteOrganization(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceOrganizationDeleteRequest],
) (*connect.Response[llmgwv1.IAMServiceOrganizationDeleteResponse], error) {
	s.options.Logger.Debug("[IAMService] DeleteOrganization invoked", "id", req.Msg.GetId())

	// Validate request
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: organization ID is required"))
	}

	// Check if organization exists
	_, err := s.options.OrganizationRepository.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
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
	response := &llmgwv1.IAMServiceOrganizationDeleteResponse{
		Success: true,
	}

	return connect.NewResponse(response), nil
}

// Token Service Methods

// CreateToken creates a new API token for a user
func (s *Iam) CreateToken(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceTokenCreateRequest],
) (*connect.Response[llmgwv1.IAMServiceTokenCreateResponse], error) {
	s.options.Logger.Debug("[IAMService] CreateToken invoked", "userID", req.Msg.GetUserId())

	// Validate request
	if req.Msg.GetUserId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user ID is required"))
	}

	// Check if user exists
	_, err := s.options.UserRepository.Get(ctx, req.Msg.GetUserId())
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
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
	response := &llmgwv1.IAMServiceTokenCreateResponse{
		Token: &llmgwv1.APIToken{
			Id:          token.ID,
			UserId:      token.UserID,
			Description: token.Description,
			CreatedAt:   timestamppb.New(token.CreatedAt),
			ExpiresAt:   timestamppb.New(token.ExpiresAt),
		},
		RawToken: rawToken,
	}

	if !token.LastUsedAt.IsZero() {
		response.Token.LastUsedAt = timestamppb.New(token.LastUsedAt)
	}

	return connect.NewResponse(response), nil
}

// ListUserTokens retrieves all tokens for a user
func (s *Iam) ListUserTokens(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceTokenListUserTokensRequest],
) (*connect.Response[llmgwv1.IAMServiceTokenListUserTokensResponse], error) {
	s.options.Logger.Debug("[IAMService] ListUserTokens invoked", "userID", req.Msg.GetUserId())

	// Validate request
	if req.Msg.GetUserId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: user ID is required"))
	}

	// Check if user exists
	_, err := s.options.UserRepository.Get(ctx, req.Msg.GetUserId())
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: user not found: %w", err))
		}
		s.options.Logger.Error("Failed to get user for token listing", "error", err, "userID", req.Msg.GetUserId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to get user for token listing: %w", err))
	}

	// Get tokens from repository
	tokens, err := s.options.TokenRepository.ListUserTokens(ctx, req.Msg.GetUserId())
	if err != nil {
		s.options.Logger.Error("Failed to list user tokens", "error", err, "userID", req.Msg.GetUserId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to list user tokens: %w", err))
	}

	// Convert tokens to proto
	protoTokens := make([]*llmgwv1.APIToken, 0, len(tokens))
	for _, token := range tokens {
		protoToken := &llmgwv1.APIToken{
			Id:          token.ID,
			UserId:      token.UserID,
			Description: token.Description,
			CreatedAt:   timestamppb.New(token.CreatedAt),
			ExpiresAt:   timestamppb.New(token.ExpiresAt),
		}

		if !token.LastUsedAt.IsZero() {
			protoToken.LastUsedAt = timestamppb.New(token.LastUsedAt)
		}

		protoTokens = append(protoTokens, protoToken)
	}

	// Return response
	response := &llmgwv1.IAMServiceTokenListUserTokensResponse{
		Tokens: protoTokens,
	}

	return connect.NewResponse(response), nil
}

// RevokeToken invalidates a token
func (s *Iam) RevokeToken(
	ctx context.Context,
	req *connect.Request[llmgwv1.IAMServiceTokenRevokeRequest],
) (*connect.Response[llmgwv1.IAMServiceTokenRevokeResponse], error) {
	s.options.Logger.Debug("[IAMService] RevokeToken invoked", "tokenID", req.Msg.GetTokenId())

	// Validate request
	if req.Msg.GetTokenId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("iam service: token ID is required"))
	}

	// Revoke token
	err := s.options.TokenRepository.RevokeToken(ctx, req.Msg.GetTokenId())
	if err != nil {
		if errors.Is(err, llmgw.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("iam service: token not found: %w", err))
		}
		s.options.Logger.Error("Failed to revoke token", "error", err, "tokenID", req.Msg.GetTokenId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("iam service: failed to revoke token: %w", err))
	}

	// Return response
	response := &llmgwv1.IAMServiceTokenRevokeResponse{
		Success: true,
	}

	return connect.NewResponse(response), nil
}

// Helper functions

// userToProto converts a domain user to a protobuf user
func userToProto(user *llmgw.User) *llmgwv1.User {
	protoUser := &llmgwv1.User{
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
func organizationToProto(org *llmgw.Organization) *llmgwv1.Organization {
	// Convert SSOConfig map to JSON string
	ssoConfig := ""
	if org.SSOConfig != nil {
		configBytes, err := json.Marshal(org.SSOConfig)
		if err == nil {
			ssoConfig = string(configBytes)
		}
	}

	protoOrg := &llmgwv1.Organization{
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
