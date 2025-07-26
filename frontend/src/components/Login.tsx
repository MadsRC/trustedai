// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useState } from "react";
import { useAuth } from "../hooks/useAuth";

function Login() {
  const { login, loading, error } = useAuth();
  const [useApiKey, setUseApiKey] = useState(false);
  const [apiKey, setApiKey] = useState("");
  const [orgName, setOrgName] = useState("");
  const [localError, setLocalError] = useState<string>("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLocalError("");

    try {
      if (useApiKey) {
        if (apiKey.trim()) {
          await login("apikey", apiKey.trim());
        } else {
          setLocalError("Please enter an API key");
        }
      } else {
        if (orgName.trim()) {
          window.location.href = `/sso/oidc/${encodeURIComponent(orgName.trim())}`;
        } else {
          setLocalError("Please enter your organization name");
        }
      }
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Login failed");
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
            TrustedAI
          </h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            Sign in to access the dashboard
          </p>
        </div>

        {(error || localError) && (
          <div className="bg-red-50 border border-red-200 rounded-md p-4">
            <div className="text-sm text-red-700">{error || localError}</div>
          </div>
        )}

        {!useApiKey ? (
          <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
            <div>
              <label
                htmlFor="orgName"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                Organization Name
              </label>
              <input
                id="orgName"
                type="text"
                value={orgName}
                onChange={(e) => setOrgName(e.target.value)}
                placeholder="Enter your organization name"
                className="block w-full px-3 py-2 border border-gray-300 rounded-md placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                required
              />
              <p className="mt-1 text-xs text-gray-500">
                This should match the organization name configured by your
                administrator
              </p>
            </div>
            <div>
              <button
                type="submit"
                disabled={loading}
                className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
              >
                {loading ? "Signing in..." : "Sign in with SSO"}
              </button>
            </div>
            <div className="text-center">
              <button
                type="button"
                onClick={() => setUseApiKey(true)}
                className="text-indigo-600 hover:text-indigo-500 text-sm"
              >
                Use an API key instead
              </button>
            </div>
          </form>
        ) : (
          <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
            <div>
              <label
                htmlFor="apikey"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                API Key
              </label>
              <input
                id="apikey"
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="Enter your API key"
                className="block w-full px-3 py-2 border border-gray-300 rounded-md placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                required
              />
            </div>
            <div>
              <button
                type="submit"
                disabled={loading}
                className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
              >
                {loading ? "Signing in..." : "Sign in with API Key"}
              </button>
            </div>
            <div className="text-center">
              <button
                type="button"
                onClick={() => setUseApiKey(false)}
                className="text-indigo-600 hover:text-indigo-500 text-sm"
              >
                Back to SSO login
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

export default Login;
