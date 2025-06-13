// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { setSSOAuthData } from "../utils/auth";
import { iamClient } from "../services/api";
import { useAuth } from "../contexts/AuthContext";

interface SSOCallbackProps {
  onLogin: (apiKey: string, username: string) => void;
}

const SSOCallback: React.FC<SSOCallbackProps> = ({ onLogin }) => {
  const [status, setStatus] = useState<"processing" | "success" | "error">(
    "processing",
  );
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { refreshAuth } = useAuth();

  useEffect(() => {
    const processCallback = async () => {
      try {
        // Check if there was an error from the SSO provider
        const error = searchParams.get("error");
        if (error) {
          setStatus("error");
          setError(`SSO authentication failed: ${error}`);
          return;
        }

        // The SSO callback handler on the server should have already processed the
        // authorization code and created a session with a session cookie.
        // Now we can get the current user directly from the session.

        try {
          console.log(
            "=== SSO CALLBACK: Getting current user from session ===",
          );

          // Get the authenticated user from the session
          const currentUserResponse = await iamClient.getCurrentUser({});
          const currentUser = currentUserResponse.user;

          if (!currentUser) {
            setStatus("error");
            setError("No user returned from getCurrentUser");
            return;
          }

          console.log("=== SSO CALLBACK: Current user retrieved ===");
          console.log("User ID:", currentUser.id);
          console.log("User email:", currentUser.email);
          console.log("User name:", currentUser.name);
          console.log("User external ID:", currentUser.externalId);
          console.log("User provider:", currentUser.provider);
          console.log("User organization ID:", currentUser.organizationId);
          console.log("User system admin:", currentUser.systemAdmin);

          // Create SSO auth data with complete user info from session
          setSSOAuthData(
            currentUser.name || currentUser.email,
            "session-id", // The session is managed by cookies
            currentUser.id,
            currentUser.systemAdmin,
          );

          // Refresh auth state to trigger route re-evaluation
          refreshAuth();

          console.log(
            "Auth data set. SSO authentication complete - not calling onLogin to avoid overwriting SSO auth data",
          );

          setStatus("success");
          setTimeout(() => navigate("/"), 1000);
        } catch (apiError: any) {
          setStatus("error");
          if (
            apiError?.message?.includes("401") ||
            apiError?.code === "unauthenticated"
          ) {
            setError(
              "SSO authentication failed. Session was not created properly.",
            );
          } else {
            setError("Failed to verify authentication. Please try again.");
          }
        }
      } catch (err: any) {
        setStatus("error");
        setError(
          err.message ||
            "An unexpected error occurred during SSO callback processing",
        );
      }
    };

    processCallback();
  }, [searchParams, navigate, onLogin, refreshAuth]);

  if (status === "processing") {
    return (
      <div className="crt-monitor">
        <div className="crt-content min-h-screen bg-black flex flex-col justify-center py-12 sm:px-6 lg:px-8">
          <div className="sm:mx-auto sm:w-full sm:max-w-md">
            <div className="bg-black border-2 border-blue-500 rounded-lg py-8 px-4 sm:px-10">
              <div className="text-center">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-400 mx-auto mb-4"></div>
                <h2
                  className="text-xl font-semibold text-blue-400 mb-2"
                  style={{ fontFamily: "monospace" }}
                >
                  {"> PROCESSING SSO AUTHENTICATION"}
                </h2>
                <p
                  className="text-blue-300 text-sm"
                  style={{ fontFamily: "monospace" }}
                >
                  {"> Please wait while we complete your sign-in..."}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (status === "success") {
    return (
      <div className="crt-monitor">
        <div className="crt-content min-h-screen bg-black flex items-center justify-center p-4">
          <div className="w-full max-w-md">
            <div className="bg-black border-2 border-green-500 rounded-lg p-6">
              <div className="text-center space-y-4">
                <div
                  style={{ width: "48px", height: "48px" }}
                  className="border-2 border-green-500 rounded flex items-center justify-center mx-auto bg-green-900"
                >
                  <svg
                    style={{ width: "24px", height: "24px" }}
                    className="text-green-400"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M5 13l4 4L19 7"
                    />
                  </svg>
                </div>
                <h2
                  className="text-xl font-semibold text-green-400"
                  style={{ fontFamily: "monospace" }}
                >
                  {"> AUTHENTICATION SUCCESSFUL"}
                </h2>
                <p
                  className="text-green-300 text-sm"
                  style={{ fontFamily: "monospace" }}
                >
                  {"> Redirecting to dashboard..."}
                </p>
                <div
                  className="text-green-600 text-xs"
                  style={{ fontFamily: "monospace" }}
                >
                  {"> SECURE CONNECTION ESTABLISHED"}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="crt-monitor">
      <div className="crt-content min-h-screen bg-black flex flex-col justify-center py-12 sm:px-6 lg:px-8">
        <div className="sm:mx-auto sm:w-full sm:max-w-md">
          <div className="bg-black border-2 border-red-500 rounded-lg p-6">
            <div className="text-center">
              <div className="w-12 h-12 border-2 border-red-500 rounded flex items-center justify-center mx-auto mb-4 bg-red-900 flex-shrink-0">
                <svg
                  className="h-6 w-6 text-red-400"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                >
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                    clipRule="evenodd"
                  />
                </svg>
              </div>
              <h3
                className="text-xl font-semibold text-red-400 mb-2"
                style={{ fontFamily: "monospace" }}
              >
                {"> AUTHENTICATION FAILED"}
              </h3>
              <p
                className="mt-2 text-sm text-red-300"
                style={{ fontFamily: "monospace" }}
              >
                {"> ERROR: " + error}
              </p>
              <div className="mt-6">
                <button
                  onClick={() => navigate("/login")}
                  className="w-full flex justify-center py-2 px-4 border border-red-500 rounded-md shadow-sm text-sm font-medium text-red-300 bg-red-900 hover:bg-red-800 hover:text-red-200 transition-all"
                  style={{ fontFamily: "monospace" }}
                >
                  [BACK TO LOGIN]
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SSOCallback;
