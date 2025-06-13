// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, { useState } from "react";

interface MatrixSSOFormProps {
  onClose: () => void;
  onSubmit: (organizationName: string) => void;
  isLoading?: boolean;
  error?: string | null;
  title?: string;
  subtitle?: string;
}

const MatrixSSOForm: React.FC<MatrixSSOFormProps> = ({
  onClose,
  onSubmit,
  isLoading = false,
  error = null,
  title = "SSO ACCESS TERMINAL",
  subtitle = "> ORGANIZATION AUTHENTICATION",
}) => {
  const [organizationName, setOrganizationName] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(organizationName);
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-80 flex items-center justify-center z-50">
      <div className="bg-black border border-green-500 rounded-lg p-8 w-96 max-w-md mx-4 shadow-2xl shadow-green-500/20">
        {/* Header */}
        <div className="text-center mb-6">
          <h2
            className="text-2xl font-bold text-green-400 mb-2"
            style={{ fontFamily: "monospace" }}
          >
            {title}
          </h2>
          <div className="text-green-300 text-sm">{subtitle}</div>
        </div>

        {/* Error Message */}
        {error && (
          <div className="bg-red-900 border border-red-500 rounded-md p-3 mb-4">
            <div
              className="text-red-300 text-sm"
              style={{ fontFamily: "monospace" }}
            >
              {"> ERROR: " + error}
            </div>
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              className="block text-green-400 text-sm mb-2"
              style={{ fontFamily: "monospace" }}
            >
              {"> ORGANIZATION NAME:"}
            </label>
            <input
              type="text"
              value={organizationName}
              onChange={(e) => setOrganizationName(e.target.value)}
              className="w-full bg-black border border-green-500 text-green-400 px-3 py-2 rounded focus:outline-none focus:border-green-300 focus:shadow-lg focus:shadow-green-500/30 transition-all"
              style={{ fontFamily: "monospace" }}
              placeholder="Enter organization name..."
              required
            />
            <p
              className="text-xs text-green-600 mt-1"
              style={{ fontFamily: "monospace" }}
            >
              {"> Enter the exact name of your organization"}
            </p>
          </div>

          {/* Buttons */}
          <div className="flex space-x-4 mt-6">
            <button
              type="submit"
              disabled={isLoading || !organizationName.trim()}
              className="flex-1 bg-green-500 hover:bg-green-600 text-white font-bold py-2 px-4 rounded transition-all duration-300 hover:shadow-lg hover:shadow-green-500/50 disabled:opacity-50 disabled:cursor-not-allowed"
              style={{ fontFamily: "monospace" }}
            >
              {isLoading ? (
                <span className="flex items-center justify-center">
                  <span className="animate-spin mr-2">⟳</span>
                  REDIRECTING...
                </span>
              ) : (
                "CONTINUE WITH SSO"
              )}
            </button>

            <button
              type="button"
              onClick={onClose}
              className="flex-1 bg-black border border-green-500 text-green-500 hover:bg-green-500 hover:text-white font-bold py-2 px-4 rounded transition-all duration-300"
              style={{ fontFamily: "monospace" }}
            >
              CANCEL
            </button>
          </div>
        </form>

        {/* Footer */}
        <div
          className="text-center mt-6 text-green-600 text-xs"
          style={{ fontFamily: "monospace" }}
        >
          {"> SECURE SSO CONNECTION ESTABLISHED"}
        </div>
      </div>
    </div>
  );
};

export default MatrixSSOForm;
