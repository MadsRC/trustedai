// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, { useState, useEffect, useCallback } from "react";
import { iamClient } from "../services/api";
import { getCurrentUserId, isCurrentUserAdmin } from "../utils/auth";
import { User } from "../types";

interface CreateTokenProps {
  isOpen: boolean;
  onClose: () => void;
  onTokenCreated: (token: any) => void;
}

const CreateToken: React.FC<CreateTokenProps> = ({
  isOpen,
  onClose,
  onTokenCreated,
}) => {
  const [description, setDescription] = useState("");
  const [selectedUserId, setSelectedUserId] = useState("");
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [createdToken, setCreatedToken] = useState<{
    token: any;
    rawToken: string;
  } | null>(null);

  const currentUserId = getCurrentUserId();
  const isAdmin = isCurrentUserAdmin();

  const fetchAllUsers = useCallback(async () => {
    try {
      setLoading(true);
      const orgsResponse = await iamClient.listOrganizations({});

      let allUsers: User[] = [];
      for (const org of orgsResponse.organizations) {
        try {
          const usersResponse = await iamClient.listUsersByOrganization({
            organizationId: org.id,
          });
          allUsers = [...allUsers, ...usersResponse.users];
        } catch (err) {
          console.warn(`Failed to fetch users for org ${org.id}:`, err);
        }
      }

      setUsers(allUsers);
      if (allUsers.length > 0 && !selectedUserId) {
        setSelectedUserId(currentUserId || allUsers[0].id);
      }
    } catch (err) {
      setError("Failed to load users");
      console.error("Fetch users error:", err);
    } finally {
      setLoading(false);
    }
  }, [currentUserId, selectedUserId]);

  useEffect(() => {
    if (isOpen && isAdmin) {
      fetchAllUsers();
    } else if (isOpen && currentUserId) {
      setSelectedUserId(currentUserId);
    }
  }, [isOpen, isAdmin, currentUserId, fetchAllUsers]);

  const handleCreateToken = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!description.trim()) {
      setError("Description is required");
      return;
    }

    const targetUserId = isAdmin ? selectedUserId : currentUserId;
    if (!targetUserId) {
      setError("No user selected");
      return;
    }

    setLoading(true);

    try {
      const response = await iamClient.createToken({
        userId: targetUserId,
        description: description.trim(),
        expiresAt: undefined, // Default expiration (1 year)
      });

      setCreatedToken({
        token: response.token,
        rawToken: response.rawToken || "Token not available",
      });

      onTokenCreated(response.token);
    } catch (err: any) {
      if (err?.message?.includes("403") || err?.code === "permission_denied") {
        setError(
          "Access denied. You do not have permission to create tokens for this user.",
        );
      } else {
        setError("Failed to create token. Please try again.");
      }
      console.error("Create token error:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setDescription("");
    setSelectedUserId("");
    setUsers([]);
    setError(null);
    setCreatedToken(null);
    onClose();
  };

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
    } catch (err) {
      console.error("Failed to copy token:", err);
    }
  };

  if (!isOpen) return null;

  if (createdToken) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-80 flex items-center justify-center z-50">
        <div className="bg-black border border-green-500 rounded-lg p-6 w-full max-w-md mx-4">
          <h2
            className="text-xl font-semibold text-green-400 mb-4"
            style={{ fontFamily: "monospace" }}
          >
            {">"} TOKEN GENERATION COMPLETE
          </h2>

          <div className="space-y-4">
            <div className="bg-green-900 border border-green-500 rounded p-3">
              <div
                className="text-green-300 text-sm"
                style={{ fontFamily: "monospace" }}
              >
                {">"} WARNING: This is the only time you'll see the full token.
                Copy and store it securely.
              </div>
            </div>

            <div>
              <label
                className="block text-sm font-medium text-green-400 mb-1"
                style={{ fontFamily: "monospace" }}
              >
                {">"} API TOKEN:
              </label>
              <div className="flex">
                <input
                  type="text"
                  value={createdToken.rawToken}
                  readOnly
                  className="flex-1 border border-green-500 bg-black text-green-300 px-3 py-2 rounded-l text-sm"
                  style={{ fontFamily: "monospace" }}
                />
                <button
                  onClick={() => copyToClipboard(createdToken.rawToken)}
                  className="px-3 py-2 bg-green-500 text-black rounded-r hover:bg-green-400 text-sm font-bold transition-all"
                  style={{
                    fontFamily: "monospace",
                  }}
                >
                  [COPY]
                </button>
              </div>
            </div>

            <div>
              <label
                className="block text-sm font-medium text-green-400 mb-1"
                style={{ fontFamily: "monospace" }}
              >
                {">"} DESCRIPTION:
              </label>
              <div
                className="text-sm text-yellow-300"
                style={{ fontFamily: "monospace" }}
              >
                {createdToken.token.description}
              </div>
            </div>
          </div>

          <div className="flex justify-end mt-6">
            <button
              onClick={handleClose}
              className="px-4 py-2 bg-green-500 text-black rounded hover:bg-green-400 font-bold transition-all"
              style={{ fontFamily: "monospace" }}
            >
              [DONE]
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-80 flex items-center justify-center z-50">
      <div className="bg-black border border-yellow-500 rounded-lg p-6 w-full max-w-md mx-4">
        <h2
          className="text-xl font-semibold text-yellow-400 mb-4"
          style={{ fontFamily: "monospace" }}
        >
          {">"} CREATE API TOKEN TERMINAL
        </h2>

        <form onSubmit={handleCreateToken} className="space-y-4">
          {error && (
            <div className="bg-red-900 border border-red-500 rounded p-3">
              <div
                className="text-red-300 text-sm"
                style={{ fontFamily: "monospace" }}
              >
                {">"} ERROR: {error}
              </div>
            </div>
          )}

          {isAdmin && (
            <div>
              <label
                className="block text-sm font-medium text-green-400 mb-1"
                style={{ fontFamily: "monospace" }}
              >
                {">"} TARGET USER:
              </label>
              <select
                value={selectedUserId}
                onChange={(e) => setSelectedUserId(e.target.value)}
                className="w-full border border-green-500 bg-black text-green-300 px-3 py-2 rounded focus:outline-none focus:border-green-300 focus:shadow-lg focus:shadow-green-500/30 transition-all"
                style={{ fontFamily: "monospace" }}
                required
              >
                <option
                  value=""
                  style={{ backgroundColor: "#000", color: "#86efac" }}
                >
                  Select target user...
                </option>
                {users.map((user) => (
                  <option
                    key={user.id}
                    value={user.id}
                    style={{ backgroundColor: "#000", color: "#86efac" }}
                  >
                    {user.name} ({user.email})
                  </option>
                ))}
              </select>
            </div>
          )}

          <div>
            <label
              className="block text-sm font-medium text-yellow-400 mb-1"
              style={{ fontFamily: "monospace" }}
            >
              {">"} TOKEN DESCRIPTION: *
            </label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full border border-yellow-500 bg-black text-yellow-300 px-3 py-2 rounded focus:outline-none focus:border-yellow-300 focus:shadow-lg focus:shadow-yellow-500/30 transition-all"
              style={{ fontFamily: "monospace" }}
              placeholder="Development API key..."
              required
            />
          </div>

          <div
            className="text-sm text-green-600"
            style={{ fontFamily: "monospace" }}
          >
            {">>"} Token will expire in 1 year from creation
          </div>

          <div className="flex space-x-3">
            <button
              type="button"
              onClick={handleClose}
              className="flex-1 px-4 py-2 text-red-400 border border-red-500 rounded hover:bg-red-900 hover:text-red-300 bg-black transition-all"
              style={{ fontFamily: "monospace" }}
            >
              [CANCEL]
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 px-4 py-2 bg-green-500 text-black border border-green-500 rounded hover:bg-green-400 disabled:opacity-50 disabled:cursor-not-allowed transition-all font-bold"
              style={{ fontFamily: "monospace" }}
            >
              {loading ? (
                <span className="flex items-center justify-center">
                  <span className="animate-spin mr-2">⟳</span>
                  GENERATING...
                </span>
              ) : (
                "[GENERATE TOKEN]"
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateToken;
