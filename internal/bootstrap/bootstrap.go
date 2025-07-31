// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/MadsRC/trustedai"
	"github.com/google/uuid"
)

const (
	SystemOrgName        = "system"
	SystemOrgDisplayName = "System Administration"
	DefaultAdminEmail    = "admin@localhost"
	DefaultAdminName     = "System Administrator"
)

// CheckAndBootstrap checks if the system needs bootstrapping and performs it if necessary
func CheckAndBootstrap(
	ctx context.Context,
	logger *slog.Logger,
	orgRepo trustedai.OrganizationRepository,
	userRepo trustedai.UserRepository,
	tokenRepo trustedai.TokenRepository,
) error {
	logger.Info("Checking if system needs bootstrapping...")

	// Check if system organization exists
	systemOrg, err := findSystemOrganization(ctx, orgRepo)
	if err != nil && !errors.Is(err, trustedai.ErrNotFound) {
		return fmt.Errorf("failed to check for system organization: %w", err)
	}

	if errors.Is(err, trustedai.ErrNotFound) {
		logger.Info("No system organization found, bootstrapping required")
	}

	needsBootstrap := false

	// If no system organization exists, we need to bootstrap
	if systemOrg == nil {
		needsBootstrap = true
	} else {
		// Check if there's at least one system admin in the system org
		hasSystemAdmin, err := hasSystemAdministrator(ctx, userRepo, systemOrg.ID)
		if err != nil {
			return fmt.Errorf("failed to check for system administrators: %w", err)
		}

		if !hasSystemAdmin {
			needsBootstrap = true
			logger.Info("No system administrator found, bootstrapping required")
		}
	}

	if !needsBootstrap {
		logger.Info("System already bootstrapped, continuing startup...")
		return nil
	}

	// Perform bootstrap
	logger.Warn("BOOTSTRAPPING SYSTEM - This will create initial admin credentials")
	_, err = performBootstrap(ctx, logger, orgRepo, userRepo, tokenRepo, systemOrg)
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}
	return nil
}

// findSystemOrganization looks for an organization with IsSystem=true
func findSystemOrganization(ctx context.Context, orgRepo trustedai.OrganizationRepository) (*trustedai.Organization, error) {
	orgs, err := orgRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, org := range orgs {
		if org.IsSystem {
			return org, nil
		}
	}

	return nil, trustedai.ErrNotFound
}

// hasSystemAdministrator checks if there's at least one system admin user in the given organization
func hasSystemAdministrator(ctx context.Context, userRepo trustedai.UserRepository, orgID string) (bool, error) {
	users, err := userRepo.ListByOrganization(ctx, orgID)
	if err != nil {
		return false, err
	}

	for _, user := range users {
		if user.SystemAdmin {
			return true, nil
		}
	}

	return false, nil
}

// performBootstrap creates the system organization, admin user, and initial token
func performBootstrap(
	ctx context.Context,
	logger *slog.Logger,
	orgRepo trustedai.OrganizationRepository,
	userRepo trustedai.UserRepository,
	tokenRepo trustedai.TokenRepository,
	existingSystemOrg *trustedai.Organization,
) (string, error) {
	var systemOrg *trustedai.Organization
	var err error

	// Create system organization if it doesn't exist
	if existingSystemOrg == nil {
		logger.Info("Creating system organization...")
		systemOrg = &trustedai.Organization{
			ID:          func() string { id, _ := uuid.NewV7(); return id.String() }(),
			Name:        SystemOrgName,
			DisplayName: SystemOrgDisplayName,
			IsSystem:    true,
			CreatedAt:   time.Now(),
			SSOType:     "",
			SSOConfig:   make(trustedai.SSOConfig),
		}

		if err := orgRepo.Create(ctx, systemOrg); err != nil {
			return "", fmt.Errorf("failed to create system organization: %w", err)
		}
		logger.Info("System organization created", "id", systemOrg.ID, "name", systemOrg.Name)
	} else {
		systemOrg = existingSystemOrg
		logger.Info("Using existing system organization", "id", systemOrg.ID, "name", systemOrg.Name)
	}

	// Create system administrator user
	logger.Info("Creating system administrator...")
	adminUser := &trustedai.User{
		ID:             func() string { id, _ := uuid.NewV7(); return id.String() }(),
		Email:          DefaultAdminEmail,
		Name:           DefaultAdminName,
		OrganizationID: systemOrg.ID,
		ExternalID:     "",
		Provider:       "none",
		SystemAdmin:    true,
		CreatedAt:      time.Now(),
		LastLogin:      time.Time{}, // Never logged in
	}

	if err := userRepo.Create(ctx, adminUser); err != nil {
		return "", fmt.Errorf("failed to create system administrator: %w", err)
	}
	logger.Info("System administrator created", "id", adminUser.ID, "email", adminUser.Email)

	// Create initial API token
	logger.Info("Creating initial API token...")
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hour expiry
	_, token, err := tokenRepo.CreateToken(ctx, adminUser.ID, "Bootstrap Token - REPLACE IMMEDIATELY", expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to create initial token: %w", err)
	}

	logger.Warn("Bootstrap completed successfully", "token", token, "expires_at", expiresAt)
	return token, nil
}
