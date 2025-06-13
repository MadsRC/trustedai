// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, { useEffect, useState } from "react";
import { iamClient } from "../services/api";
import { APIToken, User } from "../types";
import CreateToken from "../components/CreateToken";
import { isCurrentUserAdmin, getCurrentUserId } from "../utils/auth";

const Tokens: React.FC = () => {
  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedUserId, setSelectedUserId] = useState<string>("");
  const [isCreateTokenOpen, setIsCreateTokenOpen] = useState(false);

  const isAdmin = isCurrentUserAdmin();
  const currentUserId = getCurrentUserId();

  useEffect(() => {
    // For demo purposes, we'll need to get users from an organization first
    // In a real implementation, you might have a different approach
    const fetchInitialData = async () => {
      try {
        const orgsResponse = await iamClient.listOrganizations({});
        if (orgsResponse.organizations.length > 0) {
          const usersResponse = await iamClient.listUsersByOrganization({
            organizationId: orgsResponse.organizations[0].id,
          });
          setUsers(usersResponse.users);
          if (usersResponse.users.length > 0) {
            // If user is admin, show first user by default
            // If not admin, try to select current user or first user
            if (isAdmin) {
              setSelectedUserId(usersResponse.users[0].id);
            } else {
              const currentUser = usersResponse.users.find(
                (u) => u.id === currentUserId,
              );
              setSelectedUserId(currentUser?.id || usersResponse.users[0].id);
            }
          }
        }
      } catch (err) {
        setError("Failed to load initial data");
        console.error("Initial data error:", err);
      }
    };

    fetchInitialData();
  }, [isAdmin, currentUserId]);

  useEffect(() => {
    if (!selectedUserId) return;

    const fetchTokens = async () => {
      try {
        setLoading(true);
        const response = await iamClient.listUserTokens({
          userId: selectedUserId,
        });
        setTokens(response.tokens);
      } catch (err) {
        setError("Failed to load tokens");
        console.error("Tokens error:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchTokens();
  }, [selectedUserId]);

  const formatDate = (timestamp: any) => {
    if (!timestamp) return "Never";
    const seconds =
      typeof timestamp.seconds === "bigint"
        ? Number(timestamp.seconds)
        : timestamp.seconds;
    const date = new Date(seconds * 1000);
    return date.toLocaleDateString();
  };

  const getUserName = (userId: string) => {
    const user = users.find((u) => u.id === userId);
    return user?.name || user?.email || userId;
  };

  const isTokenExpired = (expiresAt: any) => {
    if (!expiresAt) return false;
    const seconds =
      typeof expiresAt.seconds === "bigint"
        ? Number(expiresAt.seconds)
        : expiresAt.seconds;
    const expiry = new Date(seconds * 1000);
    return expiry < new Date();
  };

  const handleRevokeToken = async (tokenId: string) => {
    try {
      await iamClient.revokeToken({ tokenId });
      setTokens(tokens.filter((t) => t.id !== tokenId));
    } catch (err) {
      setError("Failed to revoke token");
      console.error("Revoke token error:", err);
    }
  };

  const handleTokenCreated = (newToken: any) => {
    // Add the new token to the list if it's for the currently selected user
    if (newToken.userId === selectedUserId) {
      setTokens((prev) => [...prev, newToken]);
    }
    // Don't close the modal here - let the CreateToken component handle it
    // after the user sees the token and clicks [DONE]
  };

  if (error) {
    return (
      <div className="bg-red-900 border-2 border-red-500 rounded p-4">
        <div className="text-red-300" style={{ fontFamily: "monospace" }}>
          {">"} ERROR: {error}
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-8 flex justify-between items-center">
        <div>
          <h1
            className="text-3xl font-bold text-green-400"
            style={{ fontFamily: "monospace" }}
          >
            {">"} API TOKEN MANAGEMENT TERMINAL
          </h1>
          <p
            className="mt-2 text-green-300"
            style={{ fontFamily: "monospace" }}
          >
            {">"} Manage API tokens for users
          </p>
        </div>
        <button
          onClick={() => setIsCreateTokenOpen(true)}
          className="bg-green-500 text-black px-4 py-2 border border-green-500 rounded hover:bg-green-600 font-bold transition-all"
          style={{ fontFamily: "monospace" }}
        >
          [CREATE TOKEN]
        </button>
      </div>

      {users.length > 0 && isAdmin && (
        <div className="mb-6">
          <label
            htmlFor="user"
            className="block text-sm font-medium text-green-400 mb-2"
            style={{ fontFamily: "monospace" }}
          >
            {">"} SELECT USER:
          </label>
          <select
            id="user"
            value={selectedUserId}
            onChange={(e) => setSelectedUserId(e.target.value)}
            className="border border-green-500 bg-black text-green-300 rounded px-3 py-2 w-64 focus:outline-none focus:border-green-300 focus:shadow-lg focus:shadow-green-500/30 transition-all"
            style={{ fontFamily: "monospace" }}
          >
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

      {!isAdmin && users.length > 0 && (
        <div className="mb-6">
          <div
            className="text-sm text-green-400"
            style={{ fontFamily: "monospace" }}
          >
            {">"} VIEWING TOKENS FOR:{" "}
            <span className="text-green-300 font-bold">
              {getUserName(selectedUserId).toUpperCase()}
            </span>
          </div>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center items-center h-64">
          <div
            className="text-lg text-green-400"
            style={{ fontFamily: "monospace" }}
          >
            {">"} LOADING TOKEN DATABASE...
          </div>
        </div>
      ) : (
        <div className="bg-black border-2 border-green-500 rounded">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-green-500">
              <thead className="bg-black border-b-2 border-green-500">
                <tr>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} DESCRIPTION
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} USER
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} CREATED
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} EXPIRES
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} LAST USED
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} STATUS
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} ACTIONS
                  </th>
                </tr>
              </thead>
              <tbody className="bg-black divide-y divide-green-500">
                {tokens.map((token) => (
                  <tr
                    key={token.id}
                    className="hover:bg-green-900 border-b border-green-500"
                  >
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div
                        className="text-sm font-medium text-green-300"
                        style={{ fontFamily: "monospace" }}
                      >
                        {token.description || "UNNAMED TOKEN"}
                      </div>
                      <div
                        className="text-sm text-green-400"
                        style={{ fontFamily: "monospace" }}
                      >
                        ID: {token.id}
                      </div>
                    </td>
                    <td
                      className="px-6 py-4 whitespace-nowrap text-sm text-green-300"
                      style={{ fontFamily: "monospace" }}
                    >
                      {getUserName(token.userId)}
                    </td>
                    <td
                      className="px-6 py-4 whitespace-nowrap text-sm text-green-300"
                      style={{ fontFamily: "monospace" }}
                    >
                      {formatDate(token.createdAt)}
                    </td>
                    <td
                      className="px-6 py-4 whitespace-nowrap text-sm text-green-300"
                      style={{ fontFamily: "monospace" }}
                    >
                      {formatDate(token.expiresAt)}
                    </td>
                    <td
                      className="px-6 py-4 whitespace-nowrap text-sm text-green-300"
                      style={{ fontFamily: "monospace" }}
                    >
                      {formatDate(token.lastUsedAt)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`inline-flex px-2 py-1 text-xs font-medium rounded border ${
                          isTokenExpired(token.expiresAt)
                            ? "bg-red-900 text-red-300 border-red-500"
                            : "bg-green-900 text-green-300 border-green-500"
                        }`}
                        style={{ fontFamily: "monospace" }}
                      >
                        {isTokenExpired(token.expiresAt)
                          ? "[EXPIRED]"
                          : "[ACTIVE]"}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                      <button
                        onClick={() => handleRevokeToken(token.id)}
                        className="text-red-400 hover:text-red-300 border border-red-500 px-2 py-1 rounded hover:bg-red-900 bg-black transition-all"
                        style={{ fontFamily: "monospace" }}
                      >
                        [REVOKE]
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {tokens.length === 0 && (
            <div
              className="text-center py-8 text-green-400"
              style={{ fontFamily: "monospace" }}
            >
              {">"} NO TOKENS FOUND FOR THIS USER
            </div>
          )}
        </div>
      )}

      <CreateToken
        isOpen={isCreateTokenOpen}
        onClose={() => setIsCreateTokenOpen(false)}
        onTokenCreated={handleTokenCreated}
      />
    </div>
  );
};

export default Tokens;
