// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useState, useCallback, useEffect, useMemo } from "react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { User2, Plus, Edit, Trash2, Shield } from "lucide-react";
import { IAMService } from "../gen/proto/madsrc/llmgw/v1/iam_pb";
import {
  type User,
  type Organization,
  IAMServiceListUsersByOrganizationRequestSchema,
  IAMServiceListOrganizationsRequestSchema,
  IAMServiceCreateUserRequestSchema,
  IAMServiceUpdateUserRequestSchema,
  IAMServiceDeleteUserRequestSchema,
  UserSchema,
} from "../gen/proto/madsrc/llmgw/v1/iam_pb";
import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { useAuth } from "../hooks/useAuth";

function Users() {
  const [users, setUsers] = useState<User[]>([]);
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedOrganization, setSelectedOrganization] =
    useState<string>("all");
  const [showCreateUser, setShowCreateUser] = useState(false);
  const [showEditUser, setShowEditUser] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [formData, setFormData] = useState({
    name: "",
    email: "",
    organizationId: "",
    systemAdmin: false,
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

  const fetchUsers = useCallback(
    async (orgId: string) => {
      if (orgId === "all") {
        setUsers([]);
        return;
      }

      try {
        setLoadingUsers(true);
        setError(null);

        const request = create(IAMServiceListUsersByOrganizationRequestSchema, {
          organizationId: orgId,
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

  const createUser = useCallback(async () => {
    try {
      setLoadingUsers(true);
      setError(null);

      const userToCreate = create(UserSchema, {
        name: formData.name,
        email: formData.email,
        organizationId: formData.organizationId,
        systemAdmin: formData.systemAdmin,
      });

      const request = create(IAMServiceCreateUserRequestSchema, {
        user: userToCreate,
      });

      await client.createUser(request);
      setShowCreateUser(false);
      setFormData({
        name: "",
        email: "",
        organizationId: "",
        systemAdmin: false,
      });

      if (selectedOrganization !== "all") {
        await fetchUsers(selectedOrganization);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create user");
    } finally {
      setLoadingUsers(false);
    }
  }, [client, formData, selectedOrganization, fetchUsers]);

  const updateUser = useCallback(async () => {
    if (!editingUser) return;

    try {
      setLoadingUsers(true);
      setError(null);

      const userToUpdate = create(UserSchema, {
        ...editingUser,
        name: formData.name,
        email: formData.email,
        organizationId: formData.organizationId,
        systemAdmin: formData.systemAdmin,
      });

      const request = create(IAMServiceUpdateUserRequestSchema, {
        user: userToUpdate,
        hasSystemAdmin: true,
      });

      await client.updateUser(request);
      setShowEditUser(false);
      setEditingUser(null);
      setFormData({
        name: "",
        email: "",
        organizationId: "",
        systemAdmin: false,
      });

      if (selectedOrganization !== "all") {
        await fetchUsers(selectedOrganization);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update user");
    } finally {
      setLoadingUsers(false);
    }
  }, [client, formData, editingUser, selectedOrganization, fetchUsers]);

  const deleteUser = useCallback(
    async (userId: string) => {
      if (!confirm("Are you sure you want to delete this user?")) return;

      try {
        setLoadingUsers(true);
        setError(null);

        const request = create(IAMServiceDeleteUserRequestSchema, {
          id: userId,
        });

        await client.deleteUser(request);

        if (selectedOrganization !== "all") {
          await fetchUsers(selectedOrganization);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to delete user");
      } finally {
        setLoadingUsers(false);
      }
    },
    [client, selectedOrganization, fetchUsers],
  );

  useEffect(() => {
    fetchOrganizations();
  }, [fetchOrganizations]);

  useEffect(() => {
    fetchUsers(selectedOrganization);
  }, [selectedOrganization, fetchUsers]);

  const handleEditUser = (user: User) => {
    setEditingUser(user);
    setFormData({
      name: user.name,
      email: user.email,
      organizationId: user.organizationId,
      systemAdmin: user.systemAdmin,
    });
    setShowEditUser(true);
  };

  const handleCreateUser = () => {
    setFormData({
      name: "",
      email: "",
      organizationId:
        selectedOrganization !== "all" ? selectedOrganization : "",
      systemAdmin: false,
    });
    setShowCreateUser(true);
  };

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

  return (
    <div className="flex-1 p-6 bg-gray-50">
      <div className="max-w-7xl mx-auto">
        <div className="mb-6 flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Users</h1>
            <p className="text-gray-600">
              Manage users and their access to your platform
            </p>
          </div>
          <button
            onClick={handleCreateUser}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center space-x-2"
            disabled={selectedOrganization === "all"}
          >
            <Plus size={16} />
            <span>Create User</span>
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

        <div className="mb-4">
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
            className="w-full max-w-sm px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
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

        {/* Create User Modal */}
        {showCreateUser && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
              <h2 className="text-lg font-medium text-gray-900 mb-4">
                Create User
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
                    placeholder="Enter user name"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Email
                  </label>
                  <input
                    type="email"
                    value={formData.email}
                    onChange={(e) =>
                      setFormData({ ...formData, email: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                    placeholder="Enter user email"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Organization
                  </label>
                  <select
                    value={formData.organizationId}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        organizationId: e.target.value,
                      })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  >
                    <option value="">Select Organization</option>
                    {organizations.map((org) => (
                      <option key={org.id} value={org.id}>
                        {org.displayName || org.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.systemAdmin}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        systemAdmin: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    System Administrator
                  </label>
                </div>
                <div className="flex justify-end space-x-3 pt-4">
                  <button
                    onClick={() => setShowCreateUser(false)}
                    className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={createUser}
                    disabled={
                      !formData.name ||
                      !formData.email ||
                      !formData.organizationId ||
                      loadingUsers
                    }
                    className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {loadingUsers ? "Creating..." : "Create User"}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Edit User Modal */}
        {showEditUser && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
              <h2 className="text-lg font-medium text-gray-900 mb-4">
                Edit User
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
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Email
                  </label>
                  <input
                    type="email"
                    value={formData.email}
                    onChange={(e) =>
                      setFormData({ ...formData, email: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Organization
                  </label>
                  <select
                    value={formData.organizationId}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        organizationId: e.target.value,
                      })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  >
                    {organizations.map((org) => (
                      <option key={org.id} value={org.id}>
                        {org.displayName || org.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.systemAdmin}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        systemAdmin: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    System Administrator
                  </label>
                </div>
                <div className="flex justify-end space-x-3 pt-4">
                  <button
                    onClick={() => {
                      setShowEditUser(false);
                      setEditingUser(null);
                    }}
                    className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={updateUser}
                    disabled={
                      !formData.name ||
                      !formData.email ||
                      !formData.organizationId ||
                      loadingUsers
                    }
                    className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {loadingUsers ? "Updating..." : "Update User"}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {loadingUsers ? (
          <div className="flex justify-center items-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        ) : users.length > 0 ? (
          <div className="bg-white rounded-lg shadow">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      User
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Role
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Provider
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Created
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Last Login
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {users.map((user) => (
                    <tr key={user.id} className="hover:bg-gray-50">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <div className="flex-shrink-0 h-10 w-10">
                            <div className="h-10 w-10 rounded-full bg-blue-100 flex items-center justify-center">
                              <User2 className="h-5 w-5 text-blue-600" />
                            </div>
                          </div>
                          <div className="ml-4">
                            <div className="text-sm font-medium text-gray-900">
                              {user.name || "No name"}
                            </div>
                            <div className="text-sm text-gray-500">
                              {user.email}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span
                          className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                            user.systemAdmin
                              ? "bg-red-100 text-red-800"
                              : "bg-green-100 text-green-800"
                          }`}
                        >
                          {user.systemAdmin ? (
                            <div className="flex items-center space-x-1">
                              <Shield className="h-3 w-3" />
                              <span>System Admin</span>
                            </div>
                          ) : (
                            "User"
                          )}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm text-gray-900">
                          {user.provider || "Internal"}
                        </div>
                        {user.externalId && (
                          <div className="text-sm text-gray-500">
                            ID: {user.externalId}
                          </div>
                        )}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatDate(user.createdAt)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatDate(user.lastLogin)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        <div className="flex space-x-2">
                          <button
                            onClick={() => handleEditUser(user)}
                            className="text-blue-600 hover:text-blue-900 p-1 rounded hover:bg-blue-50"
                            title="Edit user"
                          >
                            <Edit size={16} />
                          </button>
                          <button
                            onClick={() => deleteUser(user.id)}
                            className="text-red-600 hover:text-red-900 p-1 rounded hover:bg-red-50"
                            title="Delete user"
                          >
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
        ) : selectedOrganization === "all" ? (
          <div className="text-center py-12">
            <User2 className="mx-auto h-12 w-12 text-gray-400 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              Select an Organization
            </h3>
            <p className="text-gray-500">
              Choose an organization from the dropdown above to view its users.
            </p>
          </div>
        ) : (
          <div className="text-center py-12">
            <User2 className="mx-auto h-12 w-12 text-gray-400 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              No Users Found
            </h3>
            <p className="text-gray-500 mb-4">
              No users have been created for this organization yet.
            </p>
            <button
              onClick={handleCreateUser}
              className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center space-x-2 mx-auto"
            >
              <Plus size={16} />
              <span>Create First User</span>
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

export default Users;
