// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import {
  isAuthenticatedAsync,
} from "../utils/auth";

interface AuthContextType {
  isAuthenticated: boolean;
  isLoading: boolean;
  refreshAuth: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const refreshAuth = async (showLoading: boolean = true) => {
    if (showLoading) {
      setIsLoading(true);
    }
    const authenticated = await isAuthenticatedAsync();
    setIsAuthenticated(authenticated);
    if (showLoading) {
      setIsLoading(false);
    }
  };

  // Check auth status on mount
  useEffect(() => {
    refreshAuth();
  }, []);

  // Set up periodic auth check only when authenticated
  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }

    // Set up periodic auth check (every 5 seconds) to handle token expiration
    const interval = setInterval(() => {
      refreshAuth(false); // Don't show loading for background checks
    }, 5000);

    return () => clearInterval(interval);
  }, [isAuthenticated]);

  // Listen for storage changes (for logout in other tabs)
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "llmgw_auth") {
        refreshAuth(false); // Don't show loading for storage changes
      }
    };

    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, []);

  return (
    <AuthContext.Provider value={{ isAuthenticated, isLoading, refreshAuth }}>
      {children}
    </AuthContext.Provider>
  );
};
