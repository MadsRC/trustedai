// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useState, useEffect, useCallback, useMemo } from "react";
import { Key, Plus, Copy, Trash2 } from "lucide-react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  IAMService,
  type Organization,
  type User,
  type APIToken,
  IAMServiceListOrganizationsRequestSchema,
  IAMServiceListUsersByOrganizationRequestSchema,
  IAMServiceListUserTokensRequestSchema,
  IAMServiceCreateTokenRequestSchema,
  IAMServiceRevokeTokenRequestSchema,
} from "../gen/proto/madsrc/llmgw/v1/iam_pb";
import { create } from "@bufbuild/protobuf";
import { useAuth } from "../hooks/useAuth";

function APIKeys() {
  const [showCreateToken, setShowCreateToken] = useState(false);
  const [selectedOrganization, setSelectedOrganization] =
    useState<string>("all");
  const [selectedUser, setSelectedUser] = useState<string>("all");
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [loadingTokens, setLoadingTokens] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newTokenDescription, setNewTokenDescription] = useState("");
  const [newTokenRaw, setNewTokenRaw] = useState<string | null>(null);
  const { token, user } = useAuth();

  const client = useMemo(() => {
    const transport = createConnectTransport({
      baseUrl: "",
      fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
      interceptors: [
        (next) => async (req) => {
          if (token) {
            req.header.set("Authorization", `Bearer ${token}`);
          }
          return next(req);
        },
      ],
    });

    return createClient(IAMService, transport);
  }, [token]);

  const fetchOrganizations = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const request = create(IAMServiceListOrganizationsRequestSchema, {});
      const response = await client.listOrganizations(request);
      setOrganizations(response.organizations);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch organizations",
      );
    } finally {
      setLoading(false);
    }
  }, [client]);

  const fetchUsersByOrganization = useCallback(
    async (organizationId: string) => {
      if (organizationId === "all") {
        setUsers([]);
        return;
      }

      try {
        setLoadingUsers(true);
        setError(null);

        const request = create(IAMServiceListUsersByOrganizationRequestSchema, {
          organizationId,
        });
        const response = await client.listUsersByOrganization(request);
        setUsers(response.users);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to fetch users");
      } finally {
        setLoadingUsers(false);
      }
    },
    [client],
  );

  useEffect(() => {
    fetchOrganizations();
  }, [fetchOrganizations]);

  useEffect(() => {
    fetchUsersByOrganization(selectedOrganization);
    setSelectedUser("all");
  }, [selectedOrganization, fetchUsersByOrganization]);

  const fetchTokens = useCallback(async () => {
    if (!user?.id) return;

    try {
      setLoadingTokens(true);
      setError(null);

      const request = create(IAMServiceListUserTokensRequestSchema, {
        userId: user.id,
      });
      const response = await client.listUserTokens(request);
      setTokens(response.tokens);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch tokens");
    } finally {
      setLoadingTokens(false);
    }
  }, [client, user?.id]);

  const createToken = useCallback(async () => {
    if (!user?.id || !newTokenDescription.trim()) return;

    try {
      setLoadingTokens(true);
      setError(null);

      const request = create(IAMServiceCreateTokenRequestSchema, {
        userId: user.id,
        description: newTokenDescription.trim(),
      });
      const response = await client.createToken(request);

      setNewTokenRaw(response.rawToken);
      setNewTokenDescription("");
      setShowCreateToken(false);
      await fetchTokens();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create token");
    } finally {
      setLoadingTokens(false);
    }
  }, [client, user?.id, newTokenDescription, fetchTokens]);

  const revokeToken = useCallback(
    async (tokenId: string) => {
      try {
        setLoadingTokens(true);
        setError(null);

        const request = create(IAMServiceRevokeTokenRequestSchema, {
          tokenId,
        });
        await client.revokeToken(request);
        await fetchTokens();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to revoke token");
      } finally {
        setLoadingTokens(false);
      }
    },
    [client, fetchTokens],
  );

  useEffect(() => {
    fetchTokens();
  }, [fetchTokens]);

  const filteredTokens = tokens.filter((token) => {
    if (selectedUser !== "all" && token.userId !== selectedUser) {
      return false;
    }
    return true;
  });

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatDate = (timestamp: unknown) => {
    if (!timestamp) return "N/A";
    try {
      return new Date(
        (timestamp as { toDate(): Date }).toDate(),
      ).toLocaleDateString();
    } catch {
      try {
        const ts = timestamp as {
          seconds: string | number;
          nanos: string | number;
        };
        return new Date(
          Number(ts.seconds) * 1000 + Number(ts.nanos) / 1000000,
        ).toLocaleDateString();
      } catch {
        return "N/A";
      }
    }
  };

  const getUserNameById = (userId: string) => {
    const foundUser = users.find((u) => u.id === userId);
    return foundUser ? foundUser.name : userId;
  };

  return (
    <div className="flex-1 p-6 bg-gray-50">
      <div className="max-w-7xl mx-auto">
        <div className="mb-6 flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 mb-2">API Keys</h1>
            <p className="text-gray-600">
              Manage API keys for accessing platform services
            </p>
          </div>
          <button
            onClick={() => setShowCreateToken(true)}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center space-x-2"
          >
            <Plus size={16} />
            <span>Create API Key</span>
          </button>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-6">
            <div className="flex">
              <div className="ml-3">
                <h3 className="text-sm font-medium text-red-800">Error</h3>
                <p className="text-sm text-red-700 mt-1">{error}</p>
              </div>
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div>
            <label
              htmlFor="organization-filter"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Filter by Organization
            </label>
            <select
              id="organization-filter"
              value={selectedOrganization}
              onChange={(e) => setSelectedOrganization(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              disabled={loading}
            >
              <option value="all">All Organizations</option>
              {organizations.map((org) => (
                <option key={org.id} value={org.id}>
                  {org.displayName || org.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label
              htmlFor="user-filter"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Filter by User
            </label>
            <select
              id="user-filter"
              value={selectedUser}
              onChange={(e) => setSelectedUser(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              disabled={loadingUsers || selectedOrganization === "all"}
            >
              <option value="all">
                {selectedOrganization === "all"
                  ? "Select an organization first"
                  : loadingUsers
                    ? "Loading users..."
                    : "All Users"}
              </option>
              {users.map((user) => (
                <option key={user.id} value={user.id}>
                  {user.name} ({user.email})
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Create Token Modal */}
        {showCreateToken && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
              <h2 className="text-lg font-medium text-gray-900 mb-4">
                Create API Token
              </h2>
              <div className="space-y-4">
                <div>
                  <label
                    htmlFor="description"
                    className="block text-sm font-medium text-gray-700 mb-2"
                  >
                    Description
                  </label>
                  <input
                    type="text"
                    id="description"
                    value={newTokenDescription}
                    onChange={(e) => setNewTokenDescription(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                    placeholder="Enter a description for this token"
                  />
                </div>
                <div className="flex justify-end space-x-3">
                  <button
                    onClick={() => {
                      setShowCreateToken(false);
                      setNewTokenDescription("");
                    }}
                    className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={createToken}
                    disabled={!newTokenDescription.trim() || loadingTokens}
                    className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {loadingTokens ? "Creating..." : "Create Token"}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* New Token Display Modal */}
        {newTokenRaw && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
              <h2 className="text-lg font-medium text-gray-900 mb-4">
                Token Created Successfully
              </h2>
              <div className="space-y-4">
                <div>
                  <p className="text-sm text-gray-600 mb-2">
                    Please copy this token now. You won't be able to see it
                    again.
                  </p>
                  <div className="bg-gray-50 rounded-md p-3">
                    <code className="text-sm font-mono text-gray-800 break-all">
                      {newTokenRaw}
                    </code>
                  </div>
                </div>
                <div className="flex justify-end space-x-3">
                  <button
                    onClick={() => copyToClipboard(newTokenRaw)}
                    className="px-4 py-2 text-sm bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200"
                  >
                    Copy Token
                  </button>
                  <button
                    onClick={() => setNewTokenRaw(null)}
                    className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700"
                  >
                    Done
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {loadingTokens ? (
          <div className="flex justify-center items-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        ) : (
          <div className="space-y-4">
            {filteredTokens.map((apiToken) => (
              <div key={apiToken.id} className="bg-white rounded-lg shadow p-6">
                <div className="flex items-start justify-between">
                  <div className="flex items-start space-x-4">
                    <div className="flex-shrink-0">
                      <div className="h-10 w-10 rounded-full bg-blue-100 flex items-center justify-center">
                        <Key className="h-5 w-5 text-blue-600" />
                      </div>
                    </div>
                    <div className="flex-1">
                      <div className="flex items-center space-x-2 mb-1">
                        <h3 className="text-lg font-medium text-gray-900">
                          {apiToken.description || "Unnamed Token"}
                        </h3>
                        <span className="inline-flex px-2 py-1 text-xs font-semibold rounded-full bg-green-100 text-green-800">
                          Active
                        </span>
                      </div>
                      <p className="text-sm text-gray-500 mb-2">
                        User: {getUserNameById(apiToken.userId)}
                      </p>

                      <div className="bg-gray-50 rounded-md p-3 mb-3">
                        <div className="flex items-center justify-between">
                          <code className="text-sm font-mono text-gray-800">
                            Token ID: {apiToken.id}
                          </code>
                          <div className="flex space-x-2 ml-4">
                            <button
                              onClick={() => copyToClipboard(apiToken.id)}
                              className="text-gray-500 hover:text-gray-700 p-1 rounded hover:bg-gray-200"
                              title="Copy token ID"
                            >
                              <Copy size={16} />
                            </button>
                          </div>
                        </div>
                      </div>

                      <div className="flex text-sm text-gray-500 space-x-4">
                        <span>Created: {formatDate(apiToken.createdAt)}</span>
                        {apiToken.lastUsedAt && (
                          <span>
                            Last used: {formatDate(apiToken.lastUsedAt)}
                          </span>
                        )}
                        {apiToken.expiresAt && (
                          <span>Expires: {formatDate(apiToken.expiresAt)}</span>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="flex space-x-2">
                    <button
                      onClick={() => revokeToken(apiToken.id)}
                      className="text-red-600 hover:text-red-900 p-2 rounded hover:bg-red-50"
                      title="Revoke token"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              </div>
            ))}

            {filteredTokens.length === 0 && (
              <div className="text-center py-12">
                <Key className="mx-auto h-12 w-12 text-gray-400 mb-4" />
                <h3 className="text-lg font-medium text-gray-900 mb-2">
                  No API Tokens Found
                </h3>
                <p className="text-gray-500 mb-4">
                  {selectedUser === "all"
                    ? "No API tokens have been created yet."
                    : "No API tokens found for the selected user."}
                </p>
                <button
                  onClick={() => setShowCreateToken(true)}
                  className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center space-x-2 mx-auto"
                >
                  <Plus size={16} />
                  <span>Create Your First API Token</span>
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export default APIKeys;
