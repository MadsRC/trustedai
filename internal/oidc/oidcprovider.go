// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"codeberg.org/MadsRC/llmgw"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// Ensure Provider implements the SsoProvider interface
var _ llmgw.SsoProvider = (*Provider)(nil)

// Provider implements the OIDC provider
type Provider struct {
	options *providerOptions
}

// NewProvider creates a new OIDC provider
func NewProvider(options ...ProviderOption) (*Provider, error) {
	opts := defaultProviderOptions
	for _, opt := range options {
		opt.apply(&opts)
	}

	// Validate required options
	if opts.OrgRepo == nil {
		return nil, fmt.Errorf("organization repository is required")
	}
	if opts.UserRepo == nil {
		return nil, fmt.Errorf("user repository is required")
	}
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	return &Provider{
		options: &opts,
	}, nil
}

// GetAuthURL returns the OIDC authorization URL
func (p *Provider) GetAuthURL(ctx context.Context, state string) (string, error) {
	orgName, err := getOrgNameFromContext(ctx)
	if err != nil {
		return "", err
	}

	config, err := p.getOrgOIDCConfig(ctx, orgName)
	if err != nil {
		return "", fmt.Errorf("failed to get OIDC config: %w", err)
	}

	provider, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return "", fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return oauth2Config.AuthCodeURL(state), nil
}

// HandleCallback processes the OIDC callback
func (p *Provider) HandleCallback(ctx context.Context, code string) (*llmgw.User, error) {
	orgName, err := getOrgNameFromContext(ctx)
	if err != nil {
		return nil, err
	}

	config, err := p.getOrgOIDCConfig(ctx, orgName)
	if err != nil {
		return nil, fmt.Errorf("failed to get OIDC config: %w", err)
	}

	provider, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	// Exchange the authorization code for a token
	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		p.options.Logger.Error("Failed to exchange OIDC code for token", "error", err)
		return nil, fmt.Errorf("failed to exchange OIDC code: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	// Verify the ID token
	verifier := provider.Verifier(&oidc.Config{ClientID: config.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		p.options.Logger.Error("Failed to verify ID token", "error", err)
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims from the ID token
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Subject       string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		p.options.Logger.Error("Failed to parse ID token claims", "error", err)
		return nil, fmt.Errorf("failed to parse ID token claims: %w", err)
	}

	// Ensure we have a verified email
	if !claims.EmailVerified {
		return nil, fmt.Errorf("email not verified")
	}

	// Create or retrieve user
	user := &llmgw.User{
		Email:      claims.Email,
		Name:       claims.Name,
		ExternalID: claims.Subject,
		Provider:   "oidc:" + orgName,
	}

	// Check if user already exists
	existingUser, err := p.options.UserRepo.GetByExternalID(ctx, user.Provider, user.ExternalID)
	if err == nil && existingUser != nil {
		return existingUser, nil
	}

	// Get organization
	org, err := p.options.OrgRepo.GetByName(ctx, orgName)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	// Set organization ID and generate user ID
	user.OrganizationID = org.ID
	user.ID = uuid.New().String()

	// Create new user
	err = p.options.UserRepo.Create(ctx, user)
	if err != nil {
		p.options.Logger.Error("Failed to create user", "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// ValidateToken validates an OIDC token
func (p *Provider) ValidateToken(ctx context.Context, token string) (bool, map[string]any, error) {
	orgName, err := getOrgNameFromContext(ctx)
	if err != nil {
		return false, nil, err
	}

	config, err := p.getOrgOIDCConfig(ctx, orgName)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get OIDC config: %w", err)
	}

	provider, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return false, nil, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: config.ClientID})
	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		return false, nil, nil
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return false, nil, err
	}

	return true, claims, nil
}

// StartDeviceAuth initiates the device authorization flow
func (p *Provider) StartDeviceAuth(ctx context.Context) (*llmgw.DeviceAuthResponse, error) {
	orgName, err := getOrgNameFromContext(ctx)
	if err != nil {
		return nil, err
	}

	config, err := p.getOrgOIDCConfig(ctx, orgName)
	if err != nil {
		return nil, fmt.Errorf("failed to get OIDC config: %w", err)
	}

	// Check if the provider supports device flow
	provider, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	// Get device authorization endpoint from provider metadata
	var metadata struct {
		DeviceAuthEndpoint string `json:"device_authorization_endpoint"`
	}
	if err := provider.Claims(&metadata); err != nil {
		return nil, fmt.Errorf("failed to get provider metadata: %w", err)
	}

	if metadata.DeviceAuthEndpoint == "" {
		return nil, fmt.Errorf("OIDC provider does not support device flow")
	}

	// Create request with proper headers
	form := url.Values{
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
		"scope":         {"openid profile email"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", metadata.DeviceAuthEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create device auth request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.options.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to start device auth: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.options.Logger.Error("OIDC device auth failed",
			"status", resp.Status,
			"response", string(body))
		return nil, fmt.Errorf("device auth failed with status %d", resp.StatusCode)
	}

	// Parse response
	var authResp struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, fmt.Errorf("failed to decode device auth response: %w", err)
	}

	// Use verification_uri_complete if available, otherwise use verification_uri
	verificationURI := authResp.VerificationURI
	if authResp.VerificationURIComplete != "" {
		verificationURI = authResp.VerificationURIComplete
	}

	return &llmgw.DeviceAuthResponse{
		DeviceCode:      authResp.DeviceCode,
		UserCode:        authResp.UserCode,
		VerificationURI: verificationURI,
		ExpiresIn:       time.Duration(authResp.ExpiresIn) * time.Second,
		Interval:        time.Duration(authResp.Interval) * time.Second,
	}, nil
}

// CheckDeviceAuth polls for the status of a device authorization
func (p *Provider) CheckDeviceAuth(ctx context.Context, deviceCode string) (*llmgw.User, error) {
	orgName, err := getOrgNameFromContext(ctx)
	if err != nil {
		return nil, err
	}

	config, err := p.getOrgOIDCConfig(ctx, orgName)
	if err != nil {
		return nil, fmt.Errorf("failed to get OIDC config: %w", err)
	}

	// Get token endpoint from provider
	provider, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	// Create request to token endpoint
	form := url.Values{
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
		"device_code":   {deviceCode},
		"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint().TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.options.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check device auth: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != "" {
			if errorResp.Error == "authorization_pending" {
				return nil, fmt.Errorf("device auth pending: %s", errorResp.ErrorDescription)
			}
			return nil, fmt.Errorf("device auth error: %s - %s", errorResp.Error, errorResp.ErrorDescription)
		}
		return nil, fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	// Parse token response
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.IDToken == "" {
		return nil, fmt.Errorf("no ID token in response")
	}

	// Verify the ID token
	verifier := provider.Verifier(&oidc.Config{ClientID: config.ClientID})
	idToken, err := verifier.Verify(ctx, tokenResp.IDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Subject       string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse ID token claims: %w", err)
	}

	// Ensure we have a verified email
	if !claims.EmailVerified {
		return nil, fmt.Errorf("email not verified")
	}

	// Create or retrieve user
	user := &llmgw.User{
		Email:      claims.Email,
		Name:       claims.Name,
		ExternalID: claims.Subject,
		Provider:   "oidc",
	}

	// Check if user already exists
	existingUser, err := p.options.UserRepo.GetByExternalID(ctx, user.Provider, user.ExternalID)
	if err == nil && existingUser != nil {
		return existingUser, nil
	}

	// Get organization
	org, err := p.options.OrgRepo.GetByName(ctx, orgName)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	// Set organization ID and generate user ID
	user.OrganizationID = org.ID
	user.ID = uuid.New().String()

	// Create new user
	err = p.options.UserRepo.Create(ctx, user)
	if err != nil {
		p.options.Logger.Error("Failed to create user", "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Helper function to get organization name from context
func getOrgNameFromContext(ctx context.Context) (string, error) {
	orgName, ok := ctx.Value("organization").(string)
	if !ok || orgName == "" {
		return "", fmt.Errorf("organization name missing in context")
	}
	return orgName, nil
}

// Helper function to get OIDC configuration for an organization
func (p *Provider) getOrgOIDCConfig(ctx context.Context, orgName string) (*oidcConfig, error) {
	org, err := p.options.OrgRepo.GetByName(ctx, orgName)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	// Extract OIDC configuration from organization's SSO config
	config := &oidcConfig{
		ClientID:     getStringConfig(org.SSOConfig, "oidc_client_id"),
		ClientSecret: getStringConfig(org.SSOConfig, "oidc_client_secret"),
		IssuerURL:    getStringConfig(org.SSOConfig, "oidc_issuer_url"),
		RedirectURL:  fmt.Sprintf("%s/sso/oidc/%s/callback", p.options.BaseURL, orgName),
	}

	// Validate required fields
	if config.ClientID == "" {
		return nil, fmt.Errorf("OIDC client ID not configured for organization %s", orgName)
	}
	if config.IssuerURL == "" {
		return nil, fmt.Errorf("OIDC issuer URL not configured for organization %s", orgName)
	}

	return config, nil
}

// Helper to safely get string values from SSOConfig
func getStringConfig(config map[string]any, key string) string {
	if val, ok := config[key].(string); ok {
		return val
	}
	return ""
}

// OIDC configuration
type oidcConfig struct {
	ClientID     string
	ClientSecret string
	IssuerURL    string
	RedirectURL  string
}

// Provider options
type providerOptions struct {
	Logger     *slog.Logger
	UserRepo   llmgw.UserRepository
	OrgRepo    llmgw.OrganizationRepository
	HTTPClient *http.Client
	BaseURL    string
}

// Default provider options
var defaultProviderOptions = providerOptions{
	Logger: slog.Default(),
	HTTPClient: &http.Client{
		Timeout: 10 * time.Second,
	},
}

// ProviderOption is an option for configuring a Provider
type ProviderOption interface {
	apply(*providerOptions)
}

// funcProviderOption is a function that implements ProviderOption
type funcProviderOption struct {
	f func(*providerOptions)
}

func (fpo *funcProviderOption) apply(opts *providerOptions) {
	fpo.f(opts)
}

func newFuncProviderOption(f func(*providerOptions)) *funcProviderOption {
	return &funcProviderOption{f: f}
}

// WithProviderLogger sets the logger for the provider
func WithProviderLogger(logger *slog.Logger) ProviderOption {
	return newFuncProviderOption(func(opts *providerOptions) {
		opts.Logger = logger
	})
}

// WithProviderUserRepo sets the user repository for the provider
func WithProviderUserRepo(userRepo llmgw.UserRepository) ProviderOption {
	return newFuncProviderOption(func(opts *providerOptions) {
		opts.UserRepo = userRepo
	})
}

// WithProviderOrgRepo sets the organization repository for the provider
func WithProviderOrgRepo(orgRepo llmgw.OrganizationRepository) ProviderOption {
	return newFuncProviderOption(func(opts *providerOptions) {
		opts.OrgRepo = orgRepo
	})
}

// WithProviderHTTPClient sets the HTTP client for the provider
func WithProviderHTTPClient(client *http.Client) ProviderOption {
	return newFuncProviderOption(func(opts *providerOptions) {
		opts.HTTPClient = client
	})
}

// WithProviderBaseURL sets the base URL for the provider
func WithProviderBaseURL(baseURL string) ProviderOption {
	return newFuncProviderOption(func(opts *providerOptions) {
		opts.BaseURL = baseURL
	})
}
