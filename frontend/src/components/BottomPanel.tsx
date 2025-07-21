// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

function BottomPanel() {
  return (
    <div className="bg-gray-100 border-t border-gray-200 px-6 py-3 flex items-center justify-between text-sm text-gray-600">
      <div className="flex items-center space-x-4">
        <span>System Status: Online</span>
        <span className="w-2 h-2 bg-green-500 rounded-full"></span>
      </div>

      <div className="flex items-center space-x-6">
        <span>Last updated: {new Date().toLocaleTimeString()}</span>
        <span>© 2025 Admin Dashboard</span>
      </div>
    </div>
  );
}

export default BottomPanel;
