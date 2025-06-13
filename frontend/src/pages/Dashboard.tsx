// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React, { useEffect, useState } from "react";
import { iamClient } from "../services/api";
import CreateToken from "../components/CreateToken";
import CreateUser from "../components/CreateUser";

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState({
    users: 0,
    organizations: 0,
    tokens: 0,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isCreateTokenOpen, setIsCreateTokenOpen] = useState(false);
  const [isCreateUserOpen, setIsCreateUserOpen] = useState(false);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        setLoading(true);

        // Fetch organizations
        const orgsResponse = await iamClient.listOrganizations({});
        const organizations = orgsResponse.organizations;

        // Count total users across all organizations
        let totalUsers = 0;
        for (const org of organizations) {
          try {
            const usersResponse = await iamClient.listUsersByOrganization({
              organizationId: org.id,
            });
            totalUsers += usersResponse.users.length;
          } catch (err) {
            console.warn(`Failed to fetch users for org ${org.id}:`, err);
          }
        }

        // Count total tokens across all users (simplified approach)
        let totalTokens = 0;
        for (const org of organizations) {
          try {
            const usersResponse = await iamClient.listUsersByOrganization({
              organizationId: org.id,
            });
            for (const user of usersResponse.users) {
              try {
                const tokensResponse = await iamClient.listUserTokens({
                  userId: user.id,
                });
                totalTokens += tokensResponse.tokens.length;
              } catch (err) {
                console.warn(
                  `Failed to fetch tokens for user ${user.id}:`,
                  err,
                );
              }
            }
          } catch (err) {
            console.warn(`Failed to fetch users for org ${org.id}:`, err);
          }
        }

        setStats({
          organizations: organizations.length,
          users: totalUsers,
          tokens: totalTokens,
        });
      } catch (err) {
        setError("Failed to load dashboard data");
        console.error("Dashboard error:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchStats();
  }, []);

  const handleTokenCreated = (newToken: any) => {
    // Update token count in stats
    setStats((prev) => ({ ...prev, tokens: prev.tokens + 1 }));
    // Don't close the modal here - let the CreateToken component handle it
    // after the user sees the token and clicks [DONE]
  };

  const handleUserCreated = (newUser: any) => {
    // Update user count in stats
    setStats((prev) => ({ ...prev, users: prev.users + 1 }));
    setIsCreateUserOpen(false);
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div
          className="text-lg text-green-400"
          style={{ fontFamily: "monospace" }}
        >
          {">"} LOADING DASHBOARD DATA...
        </div>
      </div>
    );
  }

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
      <div className="mb-8">
        <h1
          className="text-3xl font-bold text-green-400"
          style={{ fontFamily: "monospace" }}
        >
          {">"} CONTROL PANEL DASHBOARD
        </h1>
        <p className="mt-2 text-green-300" style={{ fontFamily: "monospace" }}>
          {">"} System overview and operational status
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-black p-6 border-2 border-green-500">
          <div className="flex items-center">
            <div className="flex-1">
              <h3
                className="text-lg font-medium text-green-400"
                style={{ fontFamily: "monospace" }}
              >
                [USERS]
              </h3>
              <p
                className="text-3xl font-bold text-green-300"
                style={{ fontFamily: "monospace" }}
              >
                {stats.users}
              </p>
            </div>
          </div>
        </div>

        <div className="bg-black p-6 border-2 border-green-500">
          <div className="flex items-center">
            <div className="flex-1">
              <h3
                className="text-lg font-medium text-green-400"
                style={{ fontFamily: "monospace" }}
              >
                [ORGANIZATIONS]
              </h3>
              <p
                className="text-3xl font-bold text-green-300"
                style={{ fontFamily: "monospace" }}
              >
                {stats.organizations}
              </p>
            </div>
          </div>
        </div>

        <div className="bg-black p-6 border-2 border-green-500">
          <div className="flex items-center">
            <div className="flex-1">
              <h3
                className="text-lg font-medium text-green-400"
                style={{ fontFamily: "monospace" }}
              >
                [API TOKENS]
              </h3>
              <p
                className="text-3xl font-bold text-green-300"
                style={{ fontFamily: "monospace" }}
              >
                {stats.tokens}
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="bg-black border-2 border-green-500 p-6">
        <h2
          className="text-xl font-semibold text-green-400 mb-4"
          style={{ fontFamily: "monospace" }}
        >
          {">"} QUICK ACTIONS
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <button
            onClick={() => setIsCreateUserOpen(true)}
            className="p-4 border-2 border-green-500 bg-black hover:border-green-400 hover:bg-green-900 transition-all text-left"
            style={{ fontFamily: "monospace" }}
          >
            <h3 className="font-medium text-green-400">[CREATE USER]</h3>
            <p className="text-sm text-green-300 mt-1">
              {">"} Add new user to organization
            </p>
          </button>
          <button
            onClick={() => (window.location.href = "/organizations")}
            className="p-4 border-2 border-green-500 bg-black hover:border-green-400 hover:bg-green-900 transition-all text-left"
            style={{ fontFamily: "monospace" }}
          >
            <h3 className="font-medium text-green-400">
              [CREATE ORGANIZATION]
            </h3>
            <p className="text-sm text-green-300 mt-1">
              {">"} Initialize new organization
            </p>
          </button>
          <button
            onClick={() => setIsCreateTokenOpen(true)}
            className="p-4 border-2 border-green-500 bg-black hover:border-green-400 hover:bg-green-900 transition-all text-left"
            style={{ fontFamily: "monospace" }}
          >
            <h3 className="font-medium text-green-400">[GENERATE TOKEN]</h3>
            <p className="text-sm text-green-300 mt-1">
              {">"} Create new API access token
            </p>
          </button>
        </div>
      </div>

      <CreateToken
        isOpen={isCreateTokenOpen}
        onClose={() => setIsCreateTokenOpen(false)}
        onTokenCreated={handleTokenCreated}
      />

      <CreateUser
        isOpen={isCreateUserOpen}
        onClose={() => setIsCreateUserOpen(false)}
        onUserCreated={handleUserCreated}
      />
    </div>
  );
};

export default Dashboard;
