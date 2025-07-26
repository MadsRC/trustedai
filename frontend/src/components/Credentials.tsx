// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useState, useEffect, useMemo, useCallback } from "react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { Key, Plus, Edit, Trash2, Eye, EyeOff } from "lucide-react";
import { ModelManagementService } from "../gen/proto/madsrc/llmgw/v1/model_management_pb";
import {
  type OpenRouterCredential,
  ModelManagementServiceListOpenRouterCredentialsRequestSchema,
  ModelManagementServiceCreateOpenRouterCredentialRequestSchema,
  ModelManagementServiceUpdateOpenRouterCredentialRequestSchema,
  ModelManagementServiceDeleteOpenRouterCredentialRequestSchema,
  OpenRouterCredentialSchema,
} from "../gen/proto/madsrc/llmgw/v1/model_management_pb";
import { create } from "@bufbuild/protobuf";
import { useAuth } from "../hooks/useAuth";

function Credentials() {
  const [credentials, setCredentials] = useState<OpenRouterCredential[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingCreate, setLoadingCreate] = useState(false);
  const [loadingUpdate, setLoadingUpdate] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingCredential, setEditingCredential] =
    useState<OpenRouterCredential | null>(null);
  const [showApiKeys, setShowApiKeys] = useState<{ [key: string]: boolean }>(
    {},
  );
  const [formData, setFormData] = useState({
    name: "",
    description: "",
    apiKey: "",
    siteName: "",
    httpReferer: "",
    enabled: true,
  });
  const { token } = useAuth();

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

    return createClient(ModelManagementService, transport);
  }, [token]);

  const fetchCredentials = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const request = create(
        ModelManagementServiceListOpenRouterCredentialsRequestSchema,
        {
          includeDisabled: true,
        },
      );
      const response = await client.listOpenRouterCredentials(request);
      setCredentials(response.credentials);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch credentials",
      );
    } finally {
      setLoading(false);
    }
  }, [client]);

  const createCredential = useCallback(async () => {
    try {
      setLoadingCreate(true);
      setError(null);

      const credentialToCreate = create(OpenRouterCredentialSchema, {
        name: formData.name,
        description: formData.description,
        apiKey: formData.apiKey,
        siteName: formData.siteName,
        httpReferer: formData.httpReferer,
        enabled: formData.enabled,
      });

      const request = create(
        ModelManagementServiceCreateOpenRouterCredentialRequestSchema,
        {
          credential: credentialToCreate,
        },
      );

      await client.createOpenRouterCredential(request);
      setShowCreateModal(false);
      resetFormData();
      fetchCredentials();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create credential",
      );
    } finally {
      setLoadingCreate(false);
    }
  }, [client, formData, fetchCredentials]);

  const updateCredential = useCallback(async () => {
    if (!editingCredential) return;

    try {
      setLoadingUpdate(true);
      setError(null);

      const credentialToUpdate = create(OpenRouterCredentialSchema, {
        id: editingCredential.id,
        name: formData.name,
        description: formData.description,
        apiKey: formData.apiKey,
        siteName: formData.siteName,
        httpReferer: formData.httpReferer,
        enabled: formData.enabled,
      });

      const request = create(
        ModelManagementServiceUpdateOpenRouterCredentialRequestSchema,
        {
          credential: credentialToUpdate,
          hasEnabled: true,
        },
      );

      await client.updateOpenRouterCredential(request);
      setShowEditModal(false);
      setEditingCredential(null);
      resetFormData();
      fetchCredentials();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to update credential",
      );
    } finally {
      setLoadingUpdate(false);
    }
  }, [client, formData, editingCredential, fetchCredentials]);

  const deleteCredential = useCallback(
    async (id: string) => {
      if (!confirm("Are you sure you want to delete this credential?")) return;

      try {
        setError(null);

        const request = create(
          ModelManagementServiceDeleteOpenRouterCredentialRequestSchema,
          {
            id,
          },
        );

        await client.deleteOpenRouterCredential(request);
        fetchCredentials();
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to delete credential",
        );
      }
    },
    [client, fetchCredentials],
  );

  const resetFormData = () => {
    setFormData({
      name: "",
      description: "",
      apiKey: "",
      siteName: "",
      httpReferer: "",
      enabled: true,
    });
  };

  const handleCreateCredential = () => {
    resetFormData();
    setShowCreateModal(true);
  };

  const handleEditCredential = (credential: OpenRouterCredential) => {
    setFormData({
      name: credential.name,
      description: credential.description,
      apiKey: credential.apiKey,
      siteName: credential.siteName,
      httpReferer: credential.httpReferer,
      enabled: credential.enabled,
    });
    setEditingCredential(credential);
    setShowEditModal(true);
  };

  const toggleApiKeyVisibility = (id: string) => {
    setShowApiKeys((prev) => ({
      ...prev,
      [id]: !prev[id],
    }));
  };

  const maskApiKey = (apiKey: string) => {
    if (apiKey.length <= 8) return "••••••••";
    return apiKey.slice(0, 4) + "••••••••" + apiKey.slice(-4);
  };

  useEffect(() => {
    fetchCredentials();
  }, [fetchCredentials]);

  const getStatusColor = (enabled: boolean) => {
    return enabled
      ? "text-green-600 bg-green-100"
      : "text-gray-600 bg-gray-100";
  };

  if (loading) {
    return (
      <div className="flex-1 p-8 bg-gray-50">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-300 rounded w-1/4 mb-6"></div>
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-20 bg-gray-300 rounded"></div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 p-8 bg-gray-50">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 flex items-center">
            <Key className="w-8 h-8 mr-3 text-blue-600" />
            Credentials
          </h1>
          <p className="text-gray-600 mt-2">
            Manage inference gateway credentials
          </p>
        </div>

        <div className="bg-white rounded-lg shadow p-8">
          <div className="text-center">
            <div className="text-red-600 mb-2">Error loading credentials</div>
            <div className="text-gray-600">{error}</div>
            <button
              onClick={fetchCredentials}
              className="mt-4 bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
            >
              Retry
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 p-8 bg-gray-50">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 flex items-center">
          <Key className="w-8 h-8 mr-3 text-blue-600" />
          Credentials
        </h1>
        <p className="text-gray-600 mt-2">
          Manage inference gateway credentials
        </p>
      </div>

      <div className="bg-white rounded-lg shadow">
        <div className="p-6 border-b border-gray-200">
          <div className="flex justify-between items-center">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">
                OpenRouter Credentials
              </h2>
              <p className="text-sm text-gray-600 mt-1">
                {credentials.length} credential
                {credentials.length !== 1 ? "s" : ""} configured
              </p>
            </div>
            <button
              onClick={handleCreateCredential}
              className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center space-x-2"
            >
              <Plus size={16} />
              <span>Add Credential</span>
            </button>
          </div>
        </div>

        <div className="divide-y divide-gray-200">
          {credentials.map((credential) => (
            <div key={credential.id} className="p-6 hover:bg-gray-50">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center">
                    <h3 className="text-lg font-medium text-gray-900">
                      {credential.name}
                    </h3>
                    <span
                      className={`ml-3 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(
                        credential.enabled,
                      )}`}
                    >
                      <span className="capitalize">
                        {credential.enabled ? "Active" : "Disabled"}
                      </span>
                    </span>
                  </div>
                  <div className="mt-2 space-y-1">
                    {credential.description && (
                      <p className="text-sm text-gray-600">
                        {credential.description}
                      </p>
                    )}
                    <div className="flex items-center space-x-2 text-sm text-gray-600">
                      <span>API Key:</span>
                      <span className="font-mono">
                        {showApiKeys[credential.id]
                          ? credential.apiKey
                          : maskApiKey(credential.apiKey)}
                      </span>
                      <button
                        onClick={() => toggleApiKeyVisibility(credential.id)}
                        className="text-gray-400 hover:text-gray-600"
                      >
                        {showApiKeys[credential.id] ? (
                          <EyeOff className="w-4 h-4" />
                        ) : (
                          <Eye className="w-4 h-4" />
                        )}
                      </button>
                    </div>
                    {credential.siteName && (
                      <p className="text-sm text-gray-600">
                        Site: {credential.siteName}
                      </p>
                    )}
                    {credential.httpReferer && (
                      <p className="text-sm text-gray-600">
                        Referer: {credential.httpReferer}
                      </p>
                    )}
                  </div>
                </div>
                <div className="flex items-center space-x-2 ml-4">
                  <button
                    onClick={() => handleEditCredential(credential)}
                    className="text-blue-600 hover:text-blue-800"
                    title="Edit credential"
                  >
                    <Edit className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => deleteCredential(credential.id)}
                    className="text-red-600 hover:text-red-800"
                    title="Delete credential"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>

        {credentials.length === 0 && (
          <div className="p-12 text-center">
            <Key className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              No credentials configured
            </h3>
            <p className="text-gray-600 mb-4">
              Add your first OpenRouter credential to get started.
            </p>
            <button
              onClick={handleCreateCredential}
              className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center space-x-2 mx-auto"
            >
              <Plus size={16} />
              <span>Add First Credential</span>
            </button>
          </div>
        )}
      </div>

      {/* Create Credential Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl border p-6 max-w-lg w-full mx-4">
            <h2 className="text-lg font-medium text-gray-900 mb-4">
              Add OpenRouter Credential
            </h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Name
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Enter credential name"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Description
                </label>
                <textarea
                  value={formData.description}
                  onChange={(e) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Optional description"
                  rows={3}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  API Key
                </label>
                <input
                  type="password"
                  value={formData.apiKey}
                  onChange={(e) =>
                    setFormData({ ...formData, apiKey: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Enter OpenRouter API key"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Site Name
                </label>
                <input
                  type="text"
                  value={formData.siteName}
                  onChange={(e) =>
                    setFormData({ ...formData, siteName: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Your site name"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  HTTP Referer
                </label>
                <input
                  type="text"
                  value={formData.httpReferer}
                  onChange={(e) =>
                    setFormData({ ...formData, httpReferer: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="https://yourdomain.com"
                />
              </div>
              <div className="flex items-center">
                <input
                  type="checkbox"
                  checked={formData.enabled}
                  onChange={(e) =>
                    setFormData({ ...formData, enabled: e.target.checked })
                  }
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label className="ml-2 block text-sm text-gray-900">
                  Enabled
                </label>
              </div>
              <div className="flex justify-end space-x-3 pt-4">
                <button
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                >
                  Cancel
                </button>
                <button
                  onClick={createCredential}
                  disabled={!formData.name || !formData.apiKey || loadingCreate}
                  className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {loadingCreate ? "Creating..." : "Add Credential"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Edit Credential Modal */}
      {showEditModal && (
        <div className="fixed inset-0 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl border p-6 max-w-lg w-full mx-4">
            <h2 className="text-lg font-medium text-gray-900 mb-4">
              Edit OpenRouter Credential
            </h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Name
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Enter credential name"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Description
                </label>
                <textarea
                  value={formData.description}
                  onChange={(e) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Optional description"
                  rows={3}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  API Key
                </label>
                <input
                  type="password"
                  value={formData.apiKey}
                  onChange={(e) =>
                    setFormData({ ...formData, apiKey: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Enter OpenRouter API key"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Site Name
                </label>
                <input
                  type="text"
                  value={formData.siteName}
                  onChange={(e) =>
                    setFormData({ ...formData, siteName: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Your site name"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  HTTP Referer
                </label>
                <input
                  type="text"
                  value={formData.httpReferer}
                  onChange={(e) =>
                    setFormData({ ...formData, httpReferer: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="https://yourdomain.com"
                />
              </div>
              <div className="flex items-center">
                <input
                  type="checkbox"
                  checked={formData.enabled}
                  onChange={(e) =>
                    setFormData({ ...formData, enabled: e.target.checked })
                  }
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label className="ml-2 block text-sm text-gray-900">
                  Enabled
                </label>
              </div>
              <div className="flex justify-end space-x-3 pt-4">
                <button
                  onClick={() => {
                    setShowEditModal(false);
                    setEditingCredential(null);
                  }}
                  className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                >
                  Cancel
                </button>
                <button
                  onClick={updateCredential}
                  disabled={!formData.name || !formData.apiKey || loadingUpdate}
                  className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {loadingUpdate ? "Updating..." : "Update Credential"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default Credentials;
