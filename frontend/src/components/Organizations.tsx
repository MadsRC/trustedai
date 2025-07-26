// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useState, useEffect, useCallback, useMemo } from "react";
import { Building2, Plus, Settings, Trash2 } from "lucide-react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  IAMService,
  type Organization,
  IAMServiceListOrganizationsRequestSchema,
  IAMServiceCreateOrganizationRequestSchema,
  OrganizationSchema,
} from "../gen/proto/madsrc/llmgw/v1/iam_pb";
import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { useAuth } from "../hooks/useAuth";
import SSOConfigModal from "./SSOConfigModal";

function Organizations() {
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loadingCreate, setLoadingCreate] = useState(false);
  const [selectedOrganization, setSelectedOrganization] =
    useState<Organization | null>(null);
  const [isSSOModalOpen, setIsSSOModalOpen] = useState(false);
  const [showCreateOrganization, setShowCreateOrganization] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    displayName: "",
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

  useEffect(() => {
    fetchOrganizations();
  }, [fetchOrganizations]);

  const formatDate = (timestamp: Timestamp | undefined) => {
    if (!timestamp) return "N/A";
    try {
      return new Date(
        (timestamp as unknown as { toDate(): Date }).toDate(),
      ).toLocaleDateString();
    } catch {
      try {
        const ts = timestamp as unknown as {
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

  const handleConfigureSSO = (organization: Organization) => {
    setSelectedOrganization(organization);
    setIsSSOModalOpen(true);
  };

  const handleSSOConfigUpdate = (updatedOrg: Organization) => {
    setOrganizations((orgs) =>
      orgs.map((org) => (org.id === updatedOrg.id ? updatedOrg : org)),
    );
  };

  const handleCloseSSOModal = () => {
    setIsSSOModalOpen(false);
    setSelectedOrganization(null);
  };

  const handleCreateOrganization = () => {
    setFormData({ name: "", displayName: "" });
    setShowCreateOrganization(true);
  };

  const createOrganization = async () => {
    try {
      setLoadingCreate(true);
      setError(null);

      const organizationToCreate = create(OrganizationSchema, {
        name: formData.name,
        displayName: formData.displayName,
      });

      const request = create(IAMServiceCreateOrganizationRequestSchema, {
        organization: organizationToCreate,
      });

      const response = await client.createOrganization(request);

      if (response.organization) {
        setOrganizations((prev) => [...prev, response.organization!]);
        setShowCreateOrganization(false);
        setFormData({ name: "", displayName: "" });
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create organization",
      );
    } finally {
      setLoadingCreate(false);
    }
  };

  return (
    <div className="flex-1 p-6 bg-gray-50">
      <div className="max-w-7xl mx-auto">
        <div className="mb-6 flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 mb-2">
              Organizations
            </h1>
            <p className="text-gray-600">
              Manage organizations and their access to your platform
            </p>
          </div>
          <button
            onClick={handleCreateOrganization}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center space-x-2"
          >
            <Plus size={16} />
            <span>Create Organization</span>
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

        {loading ? (
          <div className="flex justify-center items-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        ) : organizations.length > 0 ? (
          <div className="bg-white rounded-lg shadow">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Organization
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Type
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      SSO
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Created
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {organizations.map((org) => (
                    <tr key={org.id} className="hover:bg-gray-50">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <div className="flex-shrink-0 h-10 w-10">
                            <div className="h-10 w-10 rounded-full bg-blue-100 flex items-center justify-center">
                              <Building2 className="h-5 w-5 text-blue-600" />
                            </div>
                          </div>
                          <div className="ml-4">
                            <div className="text-sm font-medium text-gray-900">
                              {org.displayName || org.name}
                            </div>
                            <div className="text-sm text-gray-500">
                              ID: {org.id}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span
                          className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                            org.isSystem
                              ? "bg-purple-100 text-purple-800"
                              : "bg-green-100 text-green-800"
                          }`}
                        >
                          {org.isSystem ? "System" : "Regular"}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm text-gray-900">
                          {org.ssoType || "None"}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatDate(org.createdAt)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        <div className="flex space-x-2">
                          <button
                            onClick={() => handleConfigureSSO(org)}
                            className="text-blue-600 hover:text-blue-900 p-1 rounded hover:bg-blue-50"
                            title="Configure SSO"
                          >
                            <Settings size={16} />
                          </button>
                          <button className="text-red-600 hover:text-red-900 p-1 rounded hover:bg-red-50">
                            <Trash2 size={16} />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        ) : (
          <div className="text-center py-12">
            <Building2 className="mx-auto h-12 w-12 text-gray-400 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              No Organizations Found
            </h3>
            <p className="text-gray-500 mb-4">
              No organizations have been created yet.
            </p>
            <button
              onClick={handleCreateOrganization}
              className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center space-x-2 mx-auto"
            >
              <Plus size={16} />
              <span>Create Your First Organization</span>
            </button>
          </div>
        )}
      </div>

      {/* Create Organization Modal */}
      {showCreateOrganization && (
        <div className="fixed inset-0 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl border p-6 max-w-md w-full mx-4">
            <h2 className="text-lg font-medium text-gray-900 mb-4">
              Create Organization
            </h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Organization Name
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Enter organization name"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Display Name
                </label>
                <input
                  type="text"
                  value={formData.displayName}
                  onChange={(e) =>
                    setFormData({ ...formData, displayName: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Enter display name"
                />
              </div>
              <div className="flex justify-end space-x-3 pt-4">
                <button
                  onClick={() => setShowCreateOrganization(false)}
                  className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                >
                  Cancel
                </button>
                <button
                  onClick={createOrganization}
                  disabled={!formData.name || loadingCreate}
                  className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {loadingCreate ? "Creating..." : "Create Organization"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {selectedOrganization && (
        <SSOConfigModal
          organization={selectedOrganization}
          isOpen={isSSOModalOpen}
          onClose={handleCloseSSOModal}
          onUpdate={handleSSOConfigUpdate}
        />
      )}
    </div>
  );
}

export default Organizations;
