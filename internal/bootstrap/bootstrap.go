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

	"codeberg.org/MadsRC/llmgw"
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
	orgRepo llmgw.OrganizationRepository,
	userRepo llmgw.UserRepository,
	tokenRepo llmgw.TokenRepository,
) error {
	logger.Info("Checking if system needs bootstrapping...")

	// Check if system organization exists
	systemOrg, err := findSystemOrganization(ctx, orgRepo)
	if err != nil && !errors.Is(err, llmgw.ErrNotFound) {
		return fmt.Errorf("failed to check for system organization: %w", err)
	}

	needsBootstrap := false

	// If no system org exists, we need to bootstrap
	if systemOrg == nil {
		needsBootstrap = true
		logger.Info("No system organization found, bootstrapping required")
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
	logger.Warn("🚀 BOOTSTRAPPING SYSTEM - This will create initial admin credentials")
	token, err := performBootstrap(ctx, logger, orgRepo, userRepo, tokenRepo, systemOrg)
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	// Print credentials to console
	printBootstrapCredentials(logger, token)
	return nil
}

// findSystemOrganization looks for an organization with IsSystem=true
func findSystemOrganization(ctx context.Context, orgRepo llmgw.OrganizationRepository) (*llmgw.Organization, error) {
	orgs, err := orgRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, org := range orgs {
		if org.IsSystem {
			return org, nil
		}
	}

	return nil, llmgw.ErrNotFound
}

// hasSystemAdministrator checks if there's at least one system admin user in the given organization
func hasSystemAdministrator(ctx context.Context, userRepo llmgw.UserRepository, orgID string) (bool, error) {
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
	orgRepo llmgw.OrganizationRepository,
	userRepo llmgw.UserRepository,
	tokenRepo llmgw.TokenRepository,
	existingSystemOrg *llmgw.Organization,
) (string, error) {
	var systemOrg *llmgw.Organization
	var err error

	// Create system organization if it doesn't exist
	if existingSystemOrg == nil {
		logger.Info("Creating system organization...")
		systemOrg = &llmgw.Organization{
			ID:          uuid.New().String(),
			Name:        SystemOrgName,
			DisplayName: SystemOrgDisplayName,
			IsSystem:    true,
			CreatedAt:   time.Now(),
			SSOType:     "",
			SSOConfig:   make(llmgw.SSOConfig),
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
	adminUser := &llmgw.User{
		ID:             uuid.New().String(),
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

	logger.Info("Bootstrap completed successfully")
	return token, nil
}

// printBootstrapCredentials outputs the initial credentials to the console
func printBootstrapCredentials(logger *slog.Logger, token string) {
	logger.Warn("╔══════════════════════════════════════════════════════════════════════════════╗")
	logger.Warn("║                          🔐 BOOTSTRAP CREDENTIALS 🔐                        ║")
	logger.Warn("║                                                                              ║")
	logger.Warn("║  IMPORTANT: Your system has been bootstrapped with initial credentials.     ║")
	logger.Warn("║  Please save these credentials and REPLACE THEM IMMEDIATELY for security.   ║")
	logger.Warn("║                                                                              ║")
	logger.Warn("╠══════════════════════════════════════════════════════════════════════════════╣")
	logger.Warn("║                                                                              ║")
	logger.Warn(fmt.Sprintf("║  Admin Email: %-58s ║", DefaultAdminEmail))
	logger.Warn("║  Organization: system                                                        ║")
	logger.Warn("║                                                                              ║")
	logger.Warn("║  API Token (expires in 24 hours):                                           ║")
	logger.Warn(fmt.Sprintf("║  %s ║", token))
	logger.Warn("║                                                                              ║")
	logger.Warn("║  Use this token in the Authorization header:                                ║")
	logger.Warn(fmt.Sprintf("║  Authorization: Bearer %s ║", token[:20]+"..."))
	logger.Warn("║                                                                              ║")
	logger.Warn("╠══════════════════════════════════════════════════════════════════════════════╣")
	logger.Warn("║                                                                              ║")
	logger.Warn("║  🚨 SECURITY NOTICE:                                                        ║")
	logger.Warn("║  1. This token expires in 24 hours                                          ║")
	logger.Warn("║  2. Create a new long-term token immediately                                ║")
	logger.Warn("║  3. Update the admin email from the default                                 ║")
	logger.Warn("║  4. Set up proper SSO authentication for your organization                  ║")
	logger.Warn("║                                                                              ║")
	logger.Warn("╚══════════════════════════════════════════════════════════════════════════════╝")
}
