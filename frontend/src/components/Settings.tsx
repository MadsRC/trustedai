// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { Settings as SettingsIcon } from "lucide-react";

function Settings() {
  return (
    <div className="flex-1 p-8 bg-gray-50">
      <div className="max-w-7xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-4">Settings</h1>
        </div>

        <div className="bg-white rounded-lg shadow p-8">
          <div className="text-center">
            <SettingsIcon className="h-16 w-16 mx-auto mb-4 text-gray-300" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              Settings Not Yet Implemented
            </h3>
            <p className="text-sm text-gray-500 max-w-md mx-auto">
              The settings functionality is coming soon. Check back later for
              configuration options and preferences.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

export default Settings;
