// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useState, useEffect, useCallback, type ReactNode } from "react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  IAMService,
  type User,
  IAMServiceGetCurrentUserRequestSchema,
} from "../gen/proto/madsrc/llmgw/v1/iam_pb";
import { create } from "@bufbuild/protobuf";
import { type AuthContextType } from "../types/auth";
import { AuthContext } from "./AuthContextDefinition";

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [authMethod, setAuthMethod] = useState<"sso" | "apikey">();
  const [token, setToken] = useState<string>();
  const [user, setUser] = useState<User>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();

  const fetchCurrentUser = useCallback(
    async (authToken?: string): Promise<User> => {
      const tokenToUse = authToken || token;
      const tempTransport = createConnectTransport({
        baseUrl: "",
        fetch: (input, init) =>
          fetch(input, { ...init, credentials: "include" }),
        interceptors: [
          (next) => async (req) => {
            if (tokenToUse) {
              req.header.set("Authorization", `Bearer ${tokenToUse}`);
            }
            return next(req);
          },
        ],
      });
      const tempClient = createClient(IAMService, tempTransport);

      const request = create(IAMServiceGetCurrentUserRequestSchema, {});
      const response = await tempClient.getCurrentUser(request);
      if (!response.user) {
        throw new Error("No user data received");
      }
      return response.user;
    },
    [token],
  );

  const refreshCurrentUser = async () => {
    if (!isLoggedIn) return;

    try {
      setLoading(true);
      setError(undefined);
      const currentUser = await fetchCurrentUser();
      setUser(currentUser);
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : "Failed to fetch user data";
      setError(errorMessage);
      console.error("Failed to fetch current user:", err);
    } finally {
      setLoading(false);
    }
  };

  const login = useCallback(
    async (authMethod: "sso" | "apikey", token?: string) => {
      try {
        setLoading(true);
        setError(undefined);

        setAuthMethod(authMethod);
        setToken(token);
        setIsLoggedIn(true);

        if (authMethod === "apikey" && token) {
          localStorage.setItem("apiKey", token);
        }

        const currentUser = await fetchCurrentUser(token);
        setUser(currentUser);
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : "Authentication failed";
        setError(errorMessage);

        setIsLoggedIn(false);
        setAuthMethod(undefined);
        setToken(undefined);
        setUser(undefined);
        localStorage.removeItem("apiKey");

        throw new Error(errorMessage);
      } finally {
        setLoading(false);
      }
    },
    [fetchCurrentUser],
  );

  const logout = () => {
    setIsLoggedIn(false);
    setAuthMethod(undefined);
    setToken(undefined);
    setUser(undefined);
    setError(undefined);
    localStorage.removeItem("apiKey");
  };

  useEffect(() => {
    const savedApiKey = localStorage.getItem("apiKey");
    if (savedApiKey) {
      login("apikey", savedApiKey).catch(() => {
        localStorage.removeItem("apiKey");
      });
    } else {
      // Check if we have a session cookie from SSO login
      const cookies = document.cookie.split(";").map((c) => c.trim());
      const sessionCookie = cookies.find((c) => c.startsWith("session_id="));
      console.log("Available cookies:", cookies);
      console.log("Session cookie found:", sessionCookie);

      if (sessionCookie) {
        console.log("Attempting SSO login with session cookie");
        login("sso").catch((err) => {
          console.error("SSO session validation failed:", err);
        });
      } else {
        console.log("No session cookie found, staying on login page");
      }
    }
  }, [login]);

  const value: AuthContextType = {
    isLoggedIn,
    authMethod,
    token,
    user,
    loading,
    error,
    login,
    logout,
    refreshCurrentUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
