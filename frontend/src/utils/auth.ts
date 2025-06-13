// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Auth state management
const AUTH_STORAGE_KEY = "llmgw_auth";

interface AuthData {
  apiKey?: string;
  username: string;
  timestamp: number;
  userId?: string;
  isAdmin?: boolean;
  authMethod: "apikey" | "sso";
  sessionId?: string;
}

// Extract API key from URL query parameters (legacy support)
export function getApiKeyFromUrl(): string | null {
  const urlParams = new URLSearchParams(window.location.search);
  return urlParams.get("apikey");
}

// Store authentication data for API key auth
export function setAuthData(apiKey: string, username: string): void {
  const authData: AuthData = {
    apiKey,
    username,
    timestamp: Date.now(),
    authMethod: "apikey",
  };
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(authData));
}

// Store authentication data for SSO auth
export function setSSOAuthData(
  username: string,
  sessionId: string,
  userId?: string,
  isAdmin?: boolean,
): void {
  const authData: AuthData = {
    username,
    timestamp: Date.now(),
    authMethod: "sso",
    sessionId,
    userId,
    isAdmin,
  };
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(authData));
}

// Get stored authentication data
export function getAuthData(): AuthData | null {
  try {
    const stored = localStorage.getItem(AUTH_STORAGE_KEY);
    if (!stored) return null;

    const authData: AuthData = JSON.parse(stored);

    // Check if auth is older than 24 hours
    const twentyFourHours = 24 * 60 * 60 * 1000;
    if (Date.now() - authData.timestamp > twentyFourHours) {
      clearAuthData();
      return null;
    }

    return authData;
  } catch {
    return null;
  }
}

// Clear authentication data
export function clearAuthData(): void {
  localStorage.removeItem(AUTH_STORAGE_KEY);
}

// Check if user is authenticated (sync check for stored auth)
export function isAuthenticated(): boolean {
  return getAuthData() !== null || getApiKeyFromUrl() !== null;
}

// Check if user is authenticated via session cookies (async)
export async function checkSessionAuth(): Promise<boolean> {
  try {
    // Try to make an authenticated API call to check if session is valid
    const { iamClient } = await import("../services/api");
    const response = await iamClient.getCurrentUser({});

    // If we got a user but don't have stored auth data, populate it
    if (response.user && !getAuthData()) {
      setSSOAuthData(
        response.user.name || response.user.email,
        "session-cookie", // Placeholder since actual session is in cookies
        response.user.id,
        response.user.systemAdmin,
      );
    }

    return true;
  } catch {
    return false;
  }
}

// Combined auth check that includes both stored auth and session cookies
export async function isAuthenticatedAsync(): Promise<boolean> {
  // First check stored auth data (fast)
  if (isAuthenticated()) {
    return true;
  }

  // Then check session cookies (slower, requires API call)
  return await checkSessionAuth();
}

// Get authentication method
export function getAuthMethod(): "apikey" | "sso" | null {
  const authData = getAuthData();
  return authData?.authMethod || null;
}

// Get current API key (from storage or URL)
export function getCurrentApiKey(): string | null {
  const authData = getAuthData();
  if (authData) {
    return authData.apiKey || null;
  }

  // Fallback to URL parameter for backward compatibility
  return getApiKeyFromUrl();
}

// Get current username
export function getCurrentUsername(): string | null {
  const authData = getAuthData();
  return authData?.username || null;
}

// Get current user ID
export function getCurrentUserId(): string | null {
  const authData = getAuthData();
  return authData?.userId || null;
}

// Check if current user is admin
export function isCurrentUserAdmin(): boolean {
  const authData = getAuthData();
  return authData?.isAdmin || false;
}

// Update auth data with user info
export function updateAuthWithUserInfo(userId: string, isAdmin: boolean): void {
  const authData = getAuthData();
  if (authData) {
    authData.userId = userId;
    authData.isAdmin = isAdmin;
    localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(authData));
  }
}

// Get authorization headers for API requests
export function getAuthHeaders(): Record<string, string> {
  const authData = getAuthData();

  if (authData?.authMethod === "apikey" && authData.apiKey) {
    return {
      Authorization: `Bearer ${authData.apiKey}`,
    };
  }

  // For SSO, sessions are handled via cookies
  return {};
}
