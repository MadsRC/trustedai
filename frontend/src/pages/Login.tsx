// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { iamClient } from "../services/api";
import { updateAuthWithUserInfo } from "../utils/auth";
import RainingLetters from "../components/RainingLetters";
import MatrixLoginForm from "../components/MatrixLoginForm";
import MatrixSSOForm from "../components/MatrixSSOForm";

interface LoginProps {
  onLogin: (apiKey: string, username: string) => void;
}

const Login: React.FC<LoginProps> = ({ onLogin }) => {
  const [authMethod, setAuthMethod] = useState<"sso" | "apikey" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLogging, setIsLogging] = useState(false);
  const [organizationName, setOrganizationName] = useState("");
  const [hoverText, setHoverText] = useState<string | undefined>(undefined);
  const navigate = useNavigate();

  const handleApiKeyLogin = async (username: string, apiKey: string) => {
    setError(null);

    if (!username.trim()) {
      setError("Username is required");
      return;
    }

    if (!apiKey.trim()) {
      setError("API key is required");
      return;
    }

    setIsLogging(true);

    try {
      // Temporarily store the credentials to test them
      onLogin(apiKey.trim(), username.trim());

      // Test the API key by making a simple API call
      await iamClient.listOrganizations({});

      // Try to get user info to determine admin status
      try {
        const orgs = await iamClient.listOrganizations({});
        console.log("Found organizations:", orgs.organizations); // Debug log

        // Try to find the user by username/email across all organizations
        let foundUser = null;
        for (const org of orgs.organizations) {
          try {
            const users = await iamClient.listUsersByOrganization({
              organizationId: org.id,
            });
            console.log(`Users in org ${org.name}:`, users.users); // Debug log

            // Try matching by name, email, or partial match
            foundUser = users.users.find(
              (u) =>
                u.name === username.trim() ||
                u.email === username.trim() ||
                u.name?.toLowerCase().includes(username.trim().toLowerCase()) ||
                u.email?.toLowerCase().includes(username.trim().toLowerCase()),
            );

            if (foundUser) {
              console.log("Found matching user:", foundUser); // Debug log
              break;
            }
          } catch (e) {
            console.warn(`Failed to fetch users for org ${org.id}:`, e);
          }
        }

        if (foundUser) {
          console.log(
            "Updating auth with user info:",
            foundUser.id,
            foundUser.systemAdmin,
          ); // Debug log
          updateAuthWithUserInfo(foundUser.id, foundUser.systemAdmin);
        } else {
          console.warn("No matching user found for username:", username.trim());
        }
      } catch (e) {
        // User info fetch failed, but login was successful
        console.warn("Could not fetch user info:", e);
      }

      // If we get here, the API key is valid
      navigate("/");
    } catch (err: any) {
      // Clear the stored credentials since they're invalid
      localStorage.removeItem("llmgw_auth");

      // Show appropriate error message
      if (err?.message?.includes("401") || err?.code === "unauthenticated") {
        setError("Invalid API key. Please check your credentials.");
      } else if (
        err?.message?.includes("403") ||
        err?.code === "permission_denied"
      ) {
        setError(
          "Access denied. Your API key does not have sufficient permissions.",
        );
      } else {
        setError("Login failed. Please check your credentials and try again.");
      }
      console.error("Login error:", err);
    } finally {
      setIsLogging(false);
    }
  };

  const handleSSOLogin = (orgName?: string) => {
    const orgNameToUse = orgName || organizationName;

    if (!orgNameToUse.trim()) {
      setError("Please enter an organization name");
      return;
    }

    setError(null);
    setIsLogging(true);

    try {
      // Use the organization name provided by the user
      const trimmedOrgName = orgNameToUse.trim();

      // Redirect to the SSO endpoint
      const ssoUrl = `http://localhost:9999/sso/oidc/${encodeURIComponent(trimmedOrgName)}`;
      window.location.href = ssoUrl;
    } catch (err: any) {
      setError("Failed to initiate SSO login");
      setIsLogging(false);
    }
  };

  if (authMethod === null) {
    return (
      <RainingLetters title="LLM Gateway" hoverText={hoverText}>
        <div className="mt-8 flex items-center justify-center space-x-4">
          <button
            onClick={() => setAuthMethod("sso")}
            onMouseEnter={() => setHoverText("SSO Authentication")}
            onMouseLeave={() => setHoverText(undefined)}
            className="px-8 py-3 bg-green-500 hover:bg-green-600 text-white font-bold rounded transition-all duration-300 hover:shadow-lg hover:shadow-green-500/50 border-2 border-green-500 hover:border-green-400"
            style={{ fontFamily: "monospace" }}
          >
            SIGN IN WITH SSO
          </button>

          <button
            onClick={() => setAuthMethod("apikey")}
            onMouseEnter={() => setHoverText("API Key Authentication")}
            onMouseLeave={() => setHoverText(undefined)}
            className="px-8 py-3 bg-red-500 hover:bg-red-600 text-white font-bold rounded transition-all duration-300 hover:shadow-lg hover:shadow-red-500/50 border-2 border-red-500 hover:border-red-400"
            style={{ fontFamily: "monospace" }}
          >
            SIGN IN WITH API KEY
          </button>
        </div>
      </RainingLetters>
    );
  }

  if (authMethod === "sso") {
    return (
      <RainingLetters title="SSO Authentication">
        <MatrixSSOForm
          onClose={() => setAuthMethod(null)}
          onSubmit={(orgName) => {
            setOrganizationName(orgName);
            handleSSOLogin(orgName);
          }}
          isLoading={isLogging}
          error={error}
          title="SSO ACCESS TERMINAL"
          subtitle="> ORGANIZATION AUTHENTICATION"
        />
      </RainingLetters>
    );
  }

  return (
    <RainingLetters title="API Key Authentication">
      <MatrixLoginForm
        onClose={() => setAuthMethod(null)}
        onSubmit={(username, password) => {
          handleApiKeyLogin(username, password);
        }}
        isLoading={isLogging}
        error={error}
        title="API ACCESS TERMINAL"
        subtitle="> API KEY AUTHENTICATION"
      />
    </RainingLetters>
  );
};

export default Login;
