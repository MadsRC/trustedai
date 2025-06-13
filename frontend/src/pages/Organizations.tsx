// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, { useEffect, useState } from "react";
import { iamClient } from "../services/api";
import { Organization } from "../types";

const Organizations: React.FC = () => {
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingOrg, setEditingOrg] = useState<Organization | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    displayName: "",
    ssoType: "",
    ssoConfig: "",
  });

  useEffect(() => {
    const fetchOrganizations = async () => {
      try {
        setLoading(true);
        const response = await iamClient.listOrganizations({});
        setOrganizations(response.organizations);
      } catch (err) {
        setError("Failed to load organizations");
        console.error("Organizations error:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchOrganizations();
  }, []);

  const formatDate = (timestamp: any) => {
    if (!timestamp) return "N/A";
    const seconds =
      typeof timestamp.seconds === "bigint"
        ? Number(timestamp.seconds)
        : timestamp.seconds;
    const date = new Date(seconds * 1000);
    return date.toLocaleDateString();
  };

  const getSSOTypeDisplay = (ssoType: string) => {
    if (!ssoType) return "None";
    return ssoType.toUpperCase();
  };

  const handleEditClick = (org: Organization) => {
    setEditingOrg(org);
    setFormData({
      name: org.name,
      displayName: org.displayName || "",
      ssoType: org.ssoType || "",
      ssoConfig: org.ssoConfig || "",
    });
  };

  const handleCreateClick = () => {
    setIsCreating(true);
    setFormData({
      name: "",
      displayName: "",
      ssoType: "",
      ssoConfig: "",
    });
  };

  const handleCancelEdit = () => {
    setEditingOrg(null);
    setIsCreating(false);
    setFormData({
      name: "",
      displayName: "",
      ssoType: "",
      ssoConfig: "",
    });
  };

  const handleSaveEdit = async () => {
    if (!editingOrg) return;

    try {
      const response = await iamClient.updateOrganization({
        organization: {
          id: editingOrg.id,
          name: formData.name,
          displayName: formData.displayName,
          isSystem: editingOrg.isSystem,
          ssoType: formData.ssoType,
          ssoConfig: formData.ssoConfig,
          createdAt: editingOrg.createdAt,
        },
      });

      // Update the organization in the list
      setOrganizations((orgs) =>
        orgs.map((org) =>
          org.id === editingOrg.id ? response.organization! : org,
        ),
      );

      handleCancelEdit();
    } catch (err) {
      setError("Failed to update organization");
      console.error("Update organization error:", err);
    }
  };

  const handleCreateOrganization = async () => {
    try {
      const response = await iamClient.createOrganization({
        organization: {
          id: "", // Server will generate ID
          name: formData.name,
          displayName: formData.displayName,
          isSystem: false, // New organizations are not system by default
          ssoType: formData.ssoType,
          ssoConfig: formData.ssoConfig,
          createdAt: undefined, // Server will set creation time
        },
      });

      // Add the new organization to the list
      setOrganizations((orgs) => [...orgs, response.organization!]);

      handleCancelEdit();
    } catch (err) {
      setError("Failed to create organization");
      console.error("Create organization error:", err);
    }
  };

  const handleInputChange = (field: string, value: string) => {
    setFormData((prev) => ({
      ...prev,
      [field]: value,
    }));
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
            {">"} ORGANIZATION MANAGEMENT TERMINAL
          </h1>
          <p
            className="mt-2 text-green-300"
            style={{ fontFamily: "monospace" }}
          >
            {">"} Manage organizations and their SSO configuration
          </p>
        </div>
        <button
          onClick={handleCreateClick}
          className="bg-green-500 text-black px-4 py-2 border border-green-500 rounded hover:bg-green-600 font-bold transition-all"
          style={{ fontFamily: "monospace" }}
        >
          [CREATE ORGANIZATION]
        </button>
      </div>

      {loading ? (
        <div className="flex justify-center items-center h-64">
          <div
            className="text-lg text-green-400"
            style={{ fontFamily: "monospace" }}
          >
            {">"} LOADING ORGANIZATION DATABASE...
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
                    {">"} ORGANIZATION
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} TYPE
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-green-400 uppercase tracking-wider"
                    style={{ fontFamily: "monospace" }}
                  >
                    {">"} SSO TYPE
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
                    {">"} ACTIONS
                  </th>
                </tr>
              </thead>
              <tbody className="bg-black divide-y divide-green-500">
                {organizations.map((org) => (
                  <tr
                    key={org.id}
                    className="hover:bg-green-900 border-b border-green-500"
                  >
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div>
                        <div
                          className="text-sm font-medium text-green-300"
                          style={{ fontFamily: "monospace" }}
                        >
                          {org.displayName || org.name}
                        </div>
                        <div
                          className="text-sm text-green-400"
                          style={{ fontFamily: "monospace" }}
                        >
                          {org.name}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`inline-flex px-2 py-1 text-xs font-medium rounded border ${
                          org.isSystem
                            ? "bg-red-900 text-red-300 border-red-500"
                            : "bg-green-900 text-green-300 border-green-500"
                        }`}
                        style={{ fontFamily: "monospace" }}
                      >
                        {org.isSystem ? "[SYSTEM]" : "[REGULAR]"}
                      </span>
                    </td>
                    <td
                      className="px-6 py-4 whitespace-nowrap text-sm text-green-300"
                      style={{ fontFamily: "monospace" }}
                    >
                      {getSSOTypeDisplay(org.ssoType)}
                    </td>
                    <td
                      className="px-6 py-4 whitespace-nowrap text-sm text-green-300"
                      style={{ fontFamily: "monospace" }}
                    >
                      {formatDate(org.createdAt)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                      <button
                        onClick={() => handleEditClick(org)}
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
          {organizations.length === 0 && (
            <div
              className="text-center py-8 text-green-400"
              style={{ fontFamily: "monospace" }}
            >
              {">"} NO ORGANIZATIONS FOUND
            </div>
          )}
        </div>
      )}

      {/* Edit/Create Organization Modal */}
      {(editingOrg || isCreating) && (
        <div className="fixed inset-0 bg-black bg-opacity-80 flex items-center justify-center z-50">
          <div className="bg-black border border-green-500 rounded-lg p-6 w-full max-w-md mx-4">
            <h2
              className="text-xl font-semibold text-green-400 mb-4"
              style={{ fontFamily: "monospace" }}
            >
              {">"}{" "}
              {editingOrg
                ? "EDIT ORGANIZATION TERMINAL"
                : "CREATE ORGANIZATION TERMINAL"}
            </h2>

            <div className="space-y-4">
              <div>
                <label
                  className="block text-sm font-medium text-green-400 mb-1"
                  style={{ fontFamily: "monospace" }}
                >
                  {">"} NAME:
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => handleInputChange("name", e.target.value)}
                  className="w-full border border-green-500 bg-black text-green-300 px-3 py-2 rounded focus:outline-none focus:border-green-300 focus:shadow-lg focus:shadow-green-500/30 transition-all"
                  style={{ fontFamily: "monospace" }}
                />
              </div>

              <div>
                <label
                  className="block text-sm font-medium text-green-400 mb-1"
                  style={{ fontFamily: "monospace" }}
                >
                  {">"} DISPLAY NAME:
                </label>
                <input
                  type="text"
                  value={formData.displayName}
                  onChange={(e) =>
                    handleInputChange("displayName", e.target.value)
                  }
                  className="w-full border border-green-500 bg-black text-green-300 px-3 py-2 rounded focus:outline-none focus:border-green-300 focus:shadow-lg focus:shadow-green-500/30 transition-all"
                  style={{ fontFamily: "monospace" }}
                />
              </div>

              <div>
                <label
                  className="block text-sm font-medium text-green-400 mb-1"
                  style={{ fontFamily: "monospace" }}
                >
                  {">"} SSO TYPE:
                </label>
                <select
                  value={formData.ssoType}
                  onChange={(e) => handleInputChange("ssoType", e.target.value)}
                  className="w-full border border-green-500 bg-black text-green-300 px-3 py-2 rounded focus:outline-none focus:border-green-300 focus:shadow-lg focus:shadow-green-500/30 transition-all"
                  style={{ fontFamily: "monospace" }}
                >
                  <option
                    value=""
                    style={{ backgroundColor: "#000", color: "#86efac" }}
                  >
                    NONE
                  </option>
                  <option
                    value="oidc"
                    style={{ backgroundColor: "#000", color: "#86efac" }}
                  >
                    OIDC
                  </option>
                  <option
                    value="saml"
                    style={{ backgroundColor: "#000", color: "#86efac" }}
                  >
                    SAML
                  </option>
                </select>
              </div>

              <div>
                <label
                  className="block text-sm font-medium text-green-400 mb-1"
                  style={{ fontFamily: "monospace" }}
                >
                  {">"} SSO CONFIG (JSON):
                </label>
                <textarea
                  value={formData.ssoConfig}
                  onChange={(e) =>
                    handleInputChange("ssoConfig", e.target.value)
                  }
                  rows={4}
                  className="w-full border border-green-500 bg-black text-green-300 px-3 py-2 rounded focus:outline-none focus:border-green-300 focus:shadow-lg focus:shadow-green-500/30 transition-all"
                  style={{ fontFamily: "monospace" }}
                  placeholder='{"key": "value"}'
                />
              </div>
            </div>

            <div className="flex justify-end space-x-3 mt-8">
              <button
                onClick={handleCancelEdit}
                className="px-4 py-2 text-red-400 border border-red-500 rounded hover:bg-red-900 hover:text-red-300 bg-black transition-all"
                style={{ fontFamily: "monospace" }}
              >
                [CANCEL]
              </button>
              <button
                onClick={editingOrg ? handleSaveEdit : handleCreateOrganization}
                className="px-4 py-2 bg-green-500 text-black border border-green-500 rounded hover:bg-green-400 font-bold transition-all"
                style={{ fontFamily: "monospace" }}
              >
                {editingOrg ? "[SAVE CHANGES]" : "[CREATE ORG]"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Organizations;
