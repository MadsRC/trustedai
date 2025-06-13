// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, { useEffect, useState } from "react";
import { iamClient } from "../services/api";
import { User, Organization } from "../types";
import CreateUser from "../components/CreateUser";
import EditUser from "../components/EditUser";

const Users: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedOrgId, setSelectedOrgId] = useState<string>("");
  const [isCreateUserOpen, setIsCreateUserOpen] = useState(false);
  const [isEditUserOpen, setIsEditUserOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);

  useEffect(() => {
    const fetchOrganizations = async () => {
      try {
        const response = await iamClient.listOrganizations({});
        setOrganizations(response.organizations);
        if (response.organizations.length > 0) {
          setSelectedOrgId(response.organizations[0].id);
        }
      } catch (err) {
        setError("Failed to load organizations");
        console.error("Organizations error:", err);
      }
    };

    fetchOrganizations();
  }, []);

  useEffect(() => {
    if (!selectedOrgId) return;

    const fetchUsers = async () => {
      try {
        setLoading(true);
        const response = await iamClient.listUsersByOrganization({
          organizationId: selectedOrgId,
        });
        setUsers(response.users);
      } catch (err) {
        setError("Failed to load users");
        console.error("Users error:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchUsers();
  }, [selectedOrgId]);

  const formatDate = (timestamp: any) => {
    if (!timestamp) return "Never";
    const seconds =
      typeof timestamp.seconds === "bigint"
        ? Number(timestamp.seconds)
        : timestamp.seconds;
    const date = new Date(seconds * 1000);
    return date.toLocaleDateString();
  };

  const getOrgName = (orgId: string) => {
    const org = organizations.find((o) => o.id === orgId);
    return org?.displayName || org?.name || orgId;
  };

  const handleUserCreated = (newUser: User) => {
    // If the new user belongs to the currently selected organization, add them to the list
    if (newUser.organizationId === selectedOrgId) {
      setUsers((prev) => [...prev, newUser]);
    }
    setIsCreateUserOpen(false);
  };

  const handleEditUser = (user: User) => {
    setEditingUser(user);
    setIsEditUserOpen(true);
  };

  const handleUserUpdated = (updatedUser: User) => {
    setUsers((prev) =>
      prev.map((user) => (user.id === updatedUser.id ? updatedUser : user)),
    );
    setIsEditUserOpen(false);
    setEditingUser(null);
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
            {">"} USER MANAGEMENT TERMINAL
          </h1>
          <p
            className="mt-2 text-green-300"
            style={{ fontFamily: "monospace" }}
          >
            {">"} Manage users across organizations
          </p>
        </div>
        <button
          onClick={() => setIsCreateUserOpen(true)}
          className="bg-green-500 text-black px-4 py-2 border border-green-500 rounded hover:bg-green-600 font-bold transition-all"
          style={{ fontFamily: "monospace" }}
        >
          [CREATE USER]
        </button>
      </div>

      <div className="mb-6">
        <label
          htmlFor="organization"
          className="block text-sm font-medium text-green-400 mb-2"
          style={{ fontFamily: "monospace" }}
        >
          {">"} SELECT ORGANIZATION:
        </label>
        <select
          id="organization"
          value={selectedOrgId}
          onChange={(e) => setSelectedOrgId(e.target.value)}
          className="border border-green-500 bg-black text-green-300 rounded px-3 py-2 w-64 focus:outline-none focus:border-green-300 focus:shadow-lg focus:shadow-green-500/30 transition-all"
          style={{ fontFamily: "monospace" }}
        >
          {organizations.map((org) => (
            <option
              key={org.id}
              value={org.id}
              style={{ backgroundColor: "#000", color: "#4ade80" }}
            >
              {org.displayName || org.name}
            </option>
          ))}
        </select>
      </div>

      {loading ? (
        <div className="flex justify-center items-center h-64">
          <div
            className="text-lg text-green-400"
            style={{ fontFamily: "monospace" }}
          >
            {">"} LOADING USER DATABASE...
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
                    {">"} USER
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} ORGANIZATION
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} PROVIDER
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} SYSTEM ADMIN
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} LAST LOGIN
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
                {users.map((user) => (
                  <tr
                    key={user.id}
                    className="hover:bg-green-900 border-b border-green-500"
                  >
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div>
                        <div
                          className="text-sm font-medium text-green-300"
                          style={{ fontFamily: "monospace" }}
                        >
                          {user.name}
                        </div>
                        <div
                          className="text-sm text-green-400"
                          style={{ fontFamily: "monospace" }}
                        >
                          {user.email}
                        </div>
                      </div>
                    </td>
                    <td
                      className="px-6 py-4 whitespace-nowrap text-sm text-green-300"
                      style={{ fontFamily: "monospace" }}
                    >
                      {getOrgName(user.organizationId)}
                    </td>
                    <td
                      className="px-6 py-4 whitespace-nowrap text-sm text-green-300"
                      style={{ fontFamily: "monospace" }}
                    >
                      {user.provider.toUpperCase()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`inline-flex px-2 py-1 text-xs font-medium rounded border ${
                          user.systemAdmin
                            ? "bg-red-900 text-red-300 border-red-500"
                            : "bg-gray-900 text-gray-300 border-gray-500"
                        }`}
                        style={{ fontFamily: "monospace" }}
                      >
                        {user.systemAdmin ? "[ADMIN]" : "[USER]"}
                      </span>
                    </td>
                    <td
                      className="px-6 py-4 whitespace-nowrap text-sm text-green-300"
                      style={{ fontFamily: "monospace" }}
                    >
                      {formatDate(user.lastLogin)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                      <button
                        onClick={() => handleEditUser(user)}
                        className="text-green-400 hover:text-green-300 mr-3 border border-green-500 px-2 py-1 rounded hover:bg-green-900 bg-black transition-all"
                        style={{ fontFamily: "monospace" }}
                      >
                        [EDIT]
                      </button>
                      <button
                        className="text-red-400 hover:text-red-300 border border-red-500 px-2 py-1 rounded hover:bg-red-900 bg-black transition-all"
                        style={{ fontFamily: "monospace" }}
                      >
                        [DELETE]
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {users.length === 0 && (
            <div
              className="text-center py-8 text-green-400"
              style={{ fontFamily: "monospace" }}
            >
              {">"} NO USERS FOUND IN THIS ORGANIZATION
            </div>
          )}
        </div>
      )}

      <CreateUser
        isOpen={isCreateUserOpen}
        onClose={() => setIsCreateUserOpen(false)}
        onUserCreated={handleUserCreated}
      />

      <EditUser
        isOpen={isEditUserOpen}
        onClose={() => {
          setIsEditUserOpen(false);
          setEditingUser(null);
        }}
        onUserUpdated={handleUserUpdated}
        user={editingUser}
      />
    </div>
  );
};

export default Users;
