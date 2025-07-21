// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useAuth } from "../hooks/useAuth";

function TopPanel() {
  const { user, logout } = useAuth();

  return (
    <div className="bg-gray-900 text-white px-6 py-4 flex items-center justify-between">
      <div className="flex items-center space-x-4">
        <h1 className="text-xl font-bold">Admin Dashboard</h1>
        <span className="text-sm text-gray-300">v1.0.0</span>
      </div>

      <div className="flex items-center space-x-4">
        <span className="text-sm text-gray-300">
          Welcome, {user?.name || user?.email || "User"}
        </span>
        {user?.systemAdmin && (
          <span className="bg-red-600 px-2 py-0.5 rounded text-xs font-medium">
            Admin
          </span>
        )}
        <button
          onClick={logout}
          className="bg-red-600 hover:bg-red-700 px-3 py-1 rounded text-sm transition-colors"
        >
          Logout
        </button>
      </div>
    </div>
  );
}

export default TopPanel;
