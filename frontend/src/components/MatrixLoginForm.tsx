// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, { useState } from "react";

interface MatrixLoginFormProps {
  onClose: () => void;
  onSubmit: (username: string, password: string) => void;
  isLoading?: boolean;
  error?: string | null;
  title?: string;
  subtitle?: string;
}

const MatrixLoginForm: React.FC<MatrixLoginFormProps> = ({
  onClose,
  onSubmit,
  isLoading = false,
  error = null,
  title = "ACCESS TERMINAL",
  subtitle = "> AUTHENTICATION REQUIRED",
}) => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(username, password);
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-80 flex items-center justify-center z-50">
      <div className="bg-black border border-red-500 rounded-lg p-8 w-96 max-w-md mx-4 shadow-2xl shadow-red-500/20">
        {/* Header */}
        <div className="text-center mb-6">
          <h2
            className="text-2xl font-bold text-red-500 mb-2"
            style={{ fontFamily: "monospace" }}
          >
            {title}
          </h2>
          <div className="text-red-400 text-sm">{subtitle}</div>
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
              className="block text-red-400 text-sm mb-2"
              style={{ fontFamily: "monospace" }}
            >
              {"> USERNAME:"}
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full bg-black border border-red-500 text-red-400 px-3 py-2 rounded focus:outline-none focus:border-red-300 focus:shadow-lg focus:shadow-red-500/30 transition-all"
              style={{ fontFamily: "monospace" }}
              placeholder="Enter username..."
              required
            />
          </div>

          <div>
            <label
              className="block text-red-400 text-sm mb-2"
              style={{ fontFamily: "monospace" }}
            >
              {"> API KEY:"}
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-black border border-red-500 text-red-400 px-3 py-2 rounded focus:outline-none focus:border-red-300 focus:shadow-lg focus:shadow-red-500/30 transition-all"
              style={{ fontFamily: "monospace" }}
              placeholder="Enter API key..."
              required
            />
          </div>

          {/* Buttons */}
          <div className="flex space-x-4 mt-6">
            <button
              type="submit"
              disabled={isLoading}
              className="flex-1 bg-red-500 hover:bg-red-600 text-black font-bold py-2 px-4 rounded transition-all duration-300 hover:shadow-lg hover:shadow-red-500/50 disabled:opacity-50 disabled:cursor-not-allowed"
              style={{ fontFamily: "monospace" }}
            >
              {isLoading ? (
                <span className="flex items-center justify-center">
                  <span className="animate-spin mr-2">⟳</span>
                  ACCESSING...
                </span>
              ) : (
                "LOGIN"
              )}
            </button>

            <button
              type="button"
              onClick={onClose}
              className="flex-1 bg-black border border-red-500 text-red-500 hover:bg-red-500 hover:text-black font-bold py-2 px-4 rounded transition-all duration-300"
              style={{ fontFamily: "monospace" }}
            >
              CANCEL
            </button>
          </div>
        </form>

        {/* Footer */}
        <div
          className="text-center mt-6 text-red-600 text-xs"
          style={{ fontFamily: "monospace" }}
        >
          {"> SECURE CONNECTION ESTABLISHED"}
        </div>
      </div>
    </div>
  );
};

export default MatrixLoginForm;
