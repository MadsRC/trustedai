// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React from "react";
import { Link, Outlet, useNavigate } from "react-router-dom";
import {
  getCurrentApiKey,
  getCurrentUsername,
  clearAuthData,
} from "../utils/auth";
import { useAuth } from "../contexts/AuthContext";

const Layout: React.FC = () => {
  const navigate = useNavigate();
  const { isAuthenticated: isUserAuthenticated, refreshAuth } = useAuth();
  const username = getCurrentUsername();
  const isLegacyApiKey =
    getCurrentApiKey() !== null && getCurrentUsername() === null;

  const handleLogout = () => {
    clearAuthData();
    refreshAuth(); // Trigger auth state refresh after logout
    navigate("/login");
  };

  return (
    <div className="crt-monitor">
      <div className="crt-content">
        <nav className="bg-black border-b-2 border-green-500">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between h-16">
              <div className="flex items-center">
                <Link
                  to="/"
                  className="text-xl font-semibold text-green-400"
                  style={{ fontFamily: "monospace" }}
                >
                  {">"} LLM GATEWAY CONTROL TERMINAL
                </Link>
                {isUserAuthenticated && (
                  <div className="ml-4 flex items-center space-x-2">
                    {username ? (
                      <Link
                        to="/profile"
                        className="px-2 py-1 bg-green-900 text-green-300 text-xs border border-green-500 hover:bg-green-800 hover:text-green-200 transition-colors"
                        style={{ fontFamily: "monospace" }}
                      >
                        USER: {username.toUpperCase()}
                      </Link>
                    ) : (
                      <span
                        className="px-2 py-1 bg-yellow-900 text-yellow-300 text-xs border border-yellow-500"
                        style={{ fontFamily: "monospace" }}
                      >
                        API KEY ACTIVE
                      </span>
                    )}
                  </div>
                )}
              </div>
              <div className="flex items-center space-x-4">
                <Link
                  to="/users"
                  className="text-green-400 hover:text-green-300 px-3 py-2 text-sm font-medium border border-transparent hover:border-green-500 transition-all"
                  style={{ fontFamily: "monospace" }}
                >
                  [USERS]
                </Link>
                <Link
                  to="/organizations"
                  className="text-green-400 hover:text-green-300 px-3 py-2 text-sm font-medium border border-transparent hover:border-green-500 transition-all"
                  style={{ fontFamily: "monospace" }}
                >
                  [ORGANIZATIONS]
                </Link>
                <Link
                  to="/tokens"
                  className="text-green-400 hover:text-green-300 px-3 py-2 text-sm font-medium border border-transparent hover:border-green-500 transition-all"
                  style={{ fontFamily: "monospace" }}
                >
                  [API TOKENS]
                </Link>
                {isUserAuthenticated && (
                  <button
                    onClick={handleLogout}
                    className="text-red-400 hover:text-red-300 px-3 py-2 text-sm font-medium border border-red-500 hover:border-red-400 hover:bg-red-900 transition-all bg-black"
                    style={{ fontFamily: "monospace" }}
                  >
                    [LOGOUT]
                  </button>
                )}
              </div>
            </div>
          </div>
        </nav>

        {isLegacyApiKey && (
          <div className="bg-yellow-900 border-b-2 border-yellow-500 px-4 py-2">
            <div className="max-w-7xl mx-auto">
              <p
                className="text-sm text-yellow-300"
                style={{ fontFamily: "monospace" }}
              >
                <strong>BOOTSTRAP MODE:</strong> Using API key from URL
                parameter for authentication. Remove the{" "}
                <code className="bg-yellow-800 px-1">?apikey=...</code>{" "}
                parameter for normal operation.
              </p>
            </div>
          </div>
        )}

        <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8 bg-black min-h-screen">
          <Outlet />
        </main>
      </div>
    </div>
  );
};

export default Layout;
