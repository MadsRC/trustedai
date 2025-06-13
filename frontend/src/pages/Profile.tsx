// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, { useEffect, useState } from "react";
import {
  getCurrentUsername,
  getCurrentUserId,
  isCurrentUserAdmin,
  getAuthMethod,
  getAuthData,
} from "../utils/auth";
import { iamClient } from "../services/api";

interface UserProfile {
  id: string;
  email: string;
  name: string;
  organizationId: string;
  organizationName?: string;
  externalId: string;
  provider: string;
  systemAdmin: boolean;
  createdAt: string;
  lastLogin: string;
}

const Profile: React.FC = () => {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const currentUsername = getCurrentUsername();
  const currentUserId = getCurrentUserId();
  const isAdmin = isCurrentUserAdmin();
  const authMethod = getAuthMethod();
  const authData = getAuthData();

  useEffect(() => {
    const fetchProfile = async () => {
      if (!currentUserId && !currentUsername) {
        setError("No user information available");
        setLoading(false);
        return;
      }

      try {
        setLoading(true);

        // Get the current user directly from the session/API key
        const currentUserResponse = await iamClient.getCurrentUser({});
        const user = currentUserResponse.user;

        if (!user) {
          setError("No user returned from getCurrentUser");
          return;
        }

        // Get organization name for display
        let orgName: string | undefined;
        try {
          const orgsResponse = await iamClient.listOrganizations({});
          const userOrg = orgsResponse.organizations.find(
            (org) => org.id === user.organizationId,
          );
          orgName = userOrg?.name;
        } catch (err) {
          console.warn("Failed to fetch organization name:", err);
          orgName = "Unknown";
        }

        // Create user profile from the current user
        const userProfile: UserProfile = {
          id: user.id,
          email: user.email,
          name: user.name,
          organizationId: user.organizationId,
          organizationName: orgName,
          externalId: user.externalId,
          provider: user.provider,
          systemAdmin: user.systemAdmin,
          createdAt: user.createdAt?.toDate().toISOString() || "",
          lastLogin: user.lastLogin?.toDate().toISOString() || "",
        };

        setProfile(userProfile);
      } catch (err: any) {
        setError(err.message || "Failed to fetch profile");
      } finally {
        setLoading(false);
      }
    };

    fetchProfile();
  }, [currentUserId, currentUsername]);

  if (loading) {
    return (
      <div className="max-w-3xl mx-auto">
        <div className="bg-black border border-green-500 rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <div className="animate-pulse">
              <div className="h-4 bg-green-900 rounded w-1/4 mb-4"></div>
              <div className="space-y-3">
                <div className="h-4 bg-green-900 rounded"></div>
                <div className="h-4 bg-green-900 rounded w-5/6"></div>
                <div className="h-4 bg-green-900 rounded w-4/6"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-3xl mx-auto">
        <div className="bg-red-900 border border-red-500 rounded-md p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg
                className="h-5 w-5 text-red-400"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                  clipRule="evenodd"
                />
              </svg>
            </div>
            <div className="ml-3">
              <h3
                className="text-sm font-medium text-red-300"
                style={{ fontFamily: "monospace" }}
              >
                {"> ERROR: Error loading profile"}
              </h3>
              <p
                className="mt-1 text-sm text-red-300"
                style={{ fontFamily: "monospace" }}
              >
                {error}
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto">
      <div className="bg-black border border-green-500 rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <h3
            className="text-lg leading-6 font-medium text-green-400 mb-6"
            style={{ fontFamily: "monospace" }}
          >
            {"> USER PROFILE TERMINAL"}
          </h3>

          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
            {/* Basic Information */}
            <div className="col-span-2">
              <h4
                className="text-md font-medium text-green-400 mb-3"
                style={{ fontFamily: "monospace" }}
              >
                [BASIC INFORMATION]
              </h4>
              <dl className="grid grid-cols-1 gap-x-4 gap-y-3 sm:grid-cols-2">
                <div>
                  <dt
                    className="text-sm font-medium text-green-600"
                    style={{ fontFamily: "monospace" }}
                  >
                    NAME:
                  </dt>
                  <dd
                    className="mt-1 text-sm text-green-300"
                    style={{ fontFamily: "monospace" }}
                  >
                    {profile?.name || currentUsername}
                  </dd>
                </div>
                <div>
                  <dt
                    className="text-sm font-medium text-green-600"
                    style={{ fontFamily: "monospace" }}
                  >
                    EMAIL:
                  </dt>
                  <dd
                    className="mt-1 text-sm text-green-300"
                    style={{ fontFamily: "monospace" }}
                  >
                    {profile?.email || "N/A"}
                  </dd>
                </div>
                <div>
                  <dt
                    className="text-sm font-medium text-green-600"
                    style={{ fontFamily: "monospace" }}
                  >
                    USER ID:
                  </dt>
                  <dd className="mt-1 text-sm text-green-300 font-mono">
                    {profile?.id || currentUserId}
                  </dd>
                </div>
                <div>
                  <dt
                    className="text-sm font-medium text-green-600"
                    style={{ fontFamily: "monospace" }}
                  >
                    ORGANIZATION:
                  </dt>
                  <dd
                    className="mt-1 text-sm text-green-300"
                    style={{ fontFamily: "monospace" }}
                  >
                    {profile?.organizationName || "Unknown"}
                  </dd>
                </div>
              </dl>
            </div>

            {/* Authentication Information */}
            <div className="col-span-2">
              <h4
                className="text-md font-medium text-green-400 mb-3"
                style={{ fontFamily: "monospace" }}
              >
                [AUTHENTICATION]
              </h4>
              <dl className="grid grid-cols-1 gap-x-4 gap-y-3 sm:grid-cols-2">
                <div>
                  <dt
                    className="text-sm font-medium text-green-600"
                    style={{ fontFamily: "monospace" }}
                  >
                    AUTH METHOD:
                  </dt>
                  <dd className="mt-1">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded border ${
                        authMethod === "sso"
                          ? "bg-blue-900 text-blue-300 border-blue-500"
                          : "bg-gray-900 text-gray-300 border-gray-500"
                      }`}
                      style={{ fontFamily: "monospace" }}
                    >
                      {authMethod === "sso" ? "SSO" : "API KEY"}
                    </span>
                  </dd>
                </div>
                <div>
                  <dt
                    className="text-sm font-medium text-green-600"
                    style={{ fontFamily: "monospace" }}
                  >
                    PROVIDER:
                  </dt>
                  <dd
                    className="mt-1 text-sm text-green-300"
                    style={{ fontFamily: "monospace" }}
                  >
                    {profile?.provider || "N/A"}
                  </dd>
                </div>
                <div>
                  <dt
                    className="text-sm font-medium text-green-600"
                    style={{ fontFamily: "monospace" }}
                  >
                    EXTERNAL ID:
                  </dt>
                  <dd className="mt-1 text-sm text-green-300 font-mono">
                    {profile?.externalId || "N/A"}
                  </dd>
                </div>
                <div>
                  <dt
                    className="text-sm font-medium text-green-600"
                    style={{ fontFamily: "monospace" }}
                  >
                    SYSTEM ADMIN:
                  </dt>
                  <dd className="mt-1">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded border ${
                        profile?.systemAdmin || isAdmin
                          ? "bg-red-900 text-red-300 border-red-500"
                          : "bg-green-900 text-green-300 border-green-500"
                      }`}
                      style={{ fontFamily: "monospace" }}
                    >
                      {profile?.systemAdmin || isAdmin ? "YES" : "NO"}
                    </span>
                  </dd>
                </div>
              </dl>
            </div>

            {/* Session Information */}
            {authMethod === "sso" && (
              <div className="col-span-2">
                <h4
                  className="text-md font-medium text-green-400 mb-3"
                  style={{ fontFamily: "monospace" }}
                >
                  [SESSION INFORMATION]
                </h4>
                <dl className="grid grid-cols-1 gap-x-4 gap-y-3 sm:grid-cols-2">
                  <div>
                    <dt
                      className="text-sm font-medium text-green-600"
                      style={{ fontFamily: "monospace" }}
                    >
                      SESSION ID:
                    </dt>
                    <dd className="mt-1 text-sm text-green-300 font-mono">
                      {authData?.sessionId || "N/A"}
                    </dd>
                  </div>
                  <div>
                    <dt
                      className="text-sm font-medium text-green-600"
                      style={{ fontFamily: "monospace" }}
                    >
                      LOGIN TIME:
                    </dt>
                    <dd
                      className="mt-1 text-sm text-green-300"
                      style={{ fontFamily: "monospace" }}
                    >
                      {authData?.timestamp
                        ? new Date(authData.timestamp).toLocaleString()
                        : "N/A"}
                    </dd>
                  </div>
                  {profile?.createdAt && profile.createdAt !== "" && (
                    <div>
                      <dt
                        className="text-sm font-medium text-green-600"
                        style={{ fontFamily: "monospace" }}
                      >
                        ACCOUNT CREATED:
                      </dt>
                      <dd
                        className="mt-1 text-sm text-green-300"
                        style={{ fontFamily: "monospace" }}
                      >
                        {new Date(profile.createdAt).toLocaleString()}
                      </dd>
                    </div>
                  )}
                  {profile?.lastLogin && profile.lastLogin !== "" && (
                    <div>
                      <dt
                        className="text-sm font-medium text-green-600"
                        style={{ fontFamily: "monospace" }}
                      >
                        LAST LOGIN:
                      </dt>
                      <dd
                        className="mt-1 text-sm text-green-300"
                        style={{ fontFamily: "monospace" }}
                      >
                        {new Date(profile.lastLogin).toLocaleString()}
                      </dd>
                    </div>
                  )}
                </dl>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Profile;
