// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"codeberg.org/MadsRC/llmgw"
)

func (s *SsoCallback) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.options.Logger.Info("SSO handler received request", "method", r.Method, "path", r.URL.Path)

	pathSegments := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	s.options.Logger.Debug("Path segments", "segments", pathSegments)

	if len(pathSegments) < 1 {
		s.options.Logger.Warn("Invalid path - no segments")
		http.NotFound(w, r)
		return
	}

	// Handle OIDC organization-based routes
	var provider llmgw.SsoProvider
	var ctx = r.Context()

	if pathSegments[0] == "oidc" {
		if len(pathSegments) < 2 {
			s.options.Logger.Warn("OIDC path too short", "segments", pathSegments)
			http.NotFound(w, r)
			return
		}
		orgName := pathSegments[1]
		s.options.Logger.Debug("Processing OIDC request", "orgName", orgName)
		provider = s.options.Providers["oidc"]
		if provider == nil {
			s.options.Logger.Error("OIDC provider not found")
			http.NotFound(w, r)
			return
		}
		ctx = context.WithValue(ctx, llmgw.OrganizationContextKey, orgName)
		s.options.Logger.Debug("Set organization in context", "orgName", orgName, "contextKey", "organization")

		// Adjust path segments to handle the rest of the routing
		if len(pathSegments) > 2 {
			pathSegments = append([]string{"oidc"}, pathSegments[2:]...)
		} else {
			// For /oidc/orgname (no trailing path), treat as auth init
			pathSegments = []string{"oidc"}
		}
	} else {
		provider = s.options.Providers[pathSegments[0]]
		if provider == nil {
			http.NotFound(w, r)
			return
		}
	}

	// Handle device flow endpoints
	if len(pathSegments) >= 3 && pathSegments[1] == "device" {
		switch pathSegments[2] {
		case "start":
			if r.Method == http.MethodPost {
				s.handleDeviceStart(w, r.WithContext(ctx), provider)
				return
			}
		case "poll":
			if r.Method == http.MethodPost {
				s.handleDevicePoll(w, r.WithContext(ctx), provider)
				return
			}
		}
	}

	// Handle authorization code flow endpoints
	switch {
	case len(pathSegments) == 1 && r.Method == http.MethodGet:
		s.handleAuthInit(w, r.WithContext(ctx), provider)
	case len(pathSegments) > 1 && pathSegments[1] == "callback" && r.Method == http.MethodGet:
		s.handleCallback(w, r.WithContext(ctx), provider)
	default:
		http.NotFound(w, r)
	}
}

func (s *SsoCallback) handleAuthInit(w http.ResponseWriter, r *http.Request, provider llmgw.SsoProvider) {
	state := generateSecureState()

	// Set the state as a cookie with HttpOnly flag
	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil, // Set Secure flag if connection is HTTPS
		MaxAge:   int(15 * 60), // 15 minutes in seconds
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	redirectURL, err := provider.GetAuthURL(r.Context(), state)
	if err != nil {
		s.options.Logger.Error("Failed to get auth URL", "error", err)
		http.Error(w, "Failed to initiate authentication", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (s *SsoCallback) handleCallback(w http.ResponseWriter, r *http.Request, provider llmgw.SsoProvider) {
	// Verify the state parameter to prevent CSRF attacks
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		s.options.Logger.Error("Missing state cookie", "error", err)
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	stateParam := r.URL.Query().Get("state")
	if stateParam == "" || stateParam != stateCookie.Value {
		s.options.Logger.Error("State parameter mismatch", "expected", stateCookie.Value, "received", stateParam)
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	user, err := provider.HandleCallback(r.Context(), code)
	if err != nil {
		s.options.Logger.Error("SSO callback handling failed", "error", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	// Log the authenticated user details
	s.options.Logger.Info("SSO authentication successful",
		"userID", user.ID,
		"email", user.Email,
		"name", user.Name,
		"externalID", user.ExternalID,
		"provider", user.Provider,
		"organizationID", user.OrganizationID,
		"systemAdmin", user.SystemAdmin,
	)

	// Create a session for the authenticated user
	session, err := s.options.SessionStore.Create(user)
	if err != nil {
		s.options.Logger.Error("Failed to create session", "error", err)
		http.Error(w, "session creation failed", http.StatusInternalServerError)
		return
	}

	// Log the session creation
	s.options.Logger.Info("Session created successfully",
		"sessionID", session.ID,
		"userEmail", session.User.Email,
		"userName", session.User.Name,
		"userID", session.User.ID,
		"expiresAt", session.ExpiresAt,
	)

	// Set the session cookie
	sessionCookie := &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   int(24 * time.Hour.Seconds()), // 24 hours
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, sessionCookie)

	// Redirect to the frontend root - React router will handle navigation
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleDeviceStart initiates the device authorization flow
func (s *SsoCallback) handleDeviceStart(w http.ResponseWriter, r *http.Request, provider llmgw.SsoProvider) {
	response, err := provider.StartDeviceAuth(r.Context())
	if err != nil {
		s.options.Logger.Error("Failed to start device auth", "error", err)
		http.Error(w, "failed to initiate device flow", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleDevicePoll checks the status of a device authorization
func (s *SsoCallback) handleDevicePoll(w http.ResponseWriter, r *http.Request, provider llmgw.SsoProvider) {
	var request struct {
		DeviceCode string `json:"device_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := provider.CheckDeviceAuth(r.Context(), request.DeviceCode)
	if err != nil {
		if strings.Contains(err.Error(), "device auth pending") {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		s.options.Logger.Error("Device auth failed", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create session and set cookie (same as callback flow)
	session, err := s.options.SessionStore.Create(user)
	if err != nil {
		s.options.Logger.Error("Failed to create session", "error", err)
		http.Error(w, "session creation failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   int(24 * time.Hour.Seconds()),
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user)
}

func generateSecureState() string {
	// Generate a cryptographically secure random state
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// In case of error, return a fallback value
		return "secure-state-generation-failed"
	}
	return base64.URLEncoding.EncodeToString(b)
}
