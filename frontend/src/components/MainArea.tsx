// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useState, useEffect, useCallback, useMemo } from "react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  Users,
  Activity,
  DollarSign,
  Database,
  TrendingUp,
  RefreshCw,
} from "lucide-react";
import {
  UsageAnalyticsService,
  type UsageSummary,
  type UsageEvent,
  UsageAnalyticsServiceGetUsageSummaryRequestSchema,
  UsageAnalyticsServiceGetUsageDetailsRequestSchema,
} from "../gen/proto/madsrc/trustedai/v1/usage_analytics_pb";
import {
  IAMService,
  type Organization,
  IAMServiceListOrganizationsRequestSchema,
} from "../gen/proto/madsrc/trustedai/v1/iam_pb";
import {
  ModelManagementService,
  ModelManagementServiceListModelsRequestSchema,
  type Model,
} from "../gen/proto/madsrc/trustedai/v1/model_management_pb";
import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { useAuth } from "../hooks/useAuth";
import { formatCost, formatNumber } from "../utils/formatters";

const formatTimeAgo = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return "N/A";

  let date: Date;
  if (typeof timestamp === "object" && "toDate" in timestamp) {
    date = (timestamp as { toDate(): Date }).toDate();
  } else if (typeof timestamp === "object" && "seconds" in timestamp) {
    const seconds =
      typeof timestamp.seconds === "bigint"
        ? Number(timestamp.seconds)
        : timestamp.seconds;
    date = new Date(seconds * 1000);
  } else {
    return "N/A";
  }

  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "Just now";
  if (diffMins < 60)
    return `${diffMins} minute${diffMins === 1 ? "" : "s"} ago`;
  if (diffHours < 24)
    return `${diffHours} hour${diffHours === 1 ? "" : "s"} ago`;
  return `${diffDays} day${diffDays === 1 ? "" : "s"} ago`;
};

function MainArea() {
  const [summary, setSummary] = useState<UsageSummary | null>(null);
  const [recentEvents, setRecentEvents] = useState<UsageEvent[]>([]);
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { token } = useAuth();

  const analyticsClient = useMemo(() => {
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
    return createClient(UsageAnalyticsService, transport);
  }, [token]);

  const iamClient = useMemo(() => {
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

  const modelClient = useMemo(() => {
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

  const fetchDashboardData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const now = new Date();
      const startOfWeek = new Date(now);
      startOfWeek.setDate(now.getDate() - 7);

      const timeRange = {
        start: {
          seconds: BigInt(Math.floor(startOfWeek.getTime() / 1000)),
          nanos: (startOfWeek.getTime() % 1000) * 1000000,
        },
        end: {
          seconds: BigInt(Math.floor(now.getTime() / 1000)),
          nanos: (now.getTime() % 1000) * 1000000,
        },
      };

      // Fetch data in parallel
      const [summaryResponse, eventsResponse, orgsResponse, modelsResponse] =
        await Promise.allSettled([
          analyticsClient.getUsageSummary(
            create(UsageAnalyticsServiceGetUsageSummaryRequestSchema, {
              period: "week",
              start: timeRange.start,
              end: timeRange.end,
            }),
          ),
          analyticsClient.getUsageDetails(
            create(UsageAnalyticsServiceGetUsageDetailsRequestSchema, {
              start: timeRange.start,
              end: timeRange.end,
              limit: 5,
              offset: 0,
            }),
          ),
          iamClient.listOrganizations(
            create(IAMServiceListOrganizationsRequestSchema, {}),
          ),
          modelClient.listModels(
            create(ModelManagementServiceListModelsRequestSchema, {
              includeDisabled: false,
              providerId: "",
              credentialType: 0,
            }),
          ),
        ]);

      if (summaryResponse.status === "fulfilled") {
        setSummary(summaryResponse.value.summary || null);
      }

      if (eventsResponse.status === "fulfilled") {
        setRecentEvents(eventsResponse.value.events);
      }

      if (orgsResponse.status === "fulfilled") {
        setOrganizations(orgsResponse.value.organizations);
      }

      if (modelsResponse.status === "fulfilled") {
        setModels(modelsResponse.value.models);
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch dashboard data",
      );
    } finally {
      setLoading(false);
    }
  }, [analyticsClient, iamClient, modelClient]);

  useEffect(() => {
    fetchDashboardData();
  }, [fetchDashboardData]);

  if (loading) {
    return (
      <div className="flex-1 bg-gray-50 p-8">
        <div className="max-w-7xl mx-auto">
          <div className="animate-pulse">
            <div className="h-8 bg-gray-200 rounded w-1/3 mb-8"></div>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
              {[...Array(4)].map((_, i) => (
                <div key={i} className="h-32 bg-gray-200 rounded-lg"></div>
              ))}
            </div>
            <div className="h-64 bg-gray-200 rounded-lg"></div>
          </div>
        </div>
      </div>
    );
  }

  const activeModels = models.filter((model) => model.enabled).length;

  return (
    <div className="flex-1 bg-gray-50 p-8">
      <div className="max-w-7xl mx-auto">
        <div className="flex justify-between items-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900">
            Dashboard Overview
          </h1>
          <button
            onClick={fetchDashboardData}
            className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </button>
        </div>

        {error && (
          <div className="mb-6 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md">
            <strong>Error:</strong> {error}
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <div className="bg-white p-6 rounded-lg shadow">
            <div className="flex items-center">
              <Users className="h-8 w-8 text-blue-600" />
              <div className="ml-4">
                <h3 className="text-sm font-medium text-gray-500">
                  Organizations
                </h3>
                <p className="text-2xl font-bold text-gray-900">
                  {formatNumber(organizations.length)}
                </p>
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow">
            <div className="flex items-center">
              <Activity className="h-8 w-8 text-green-600" />
              <div className="ml-4">
                <h3 className="text-sm font-medium text-gray-500">
                  API Requests (7d)
                </h3>
                <p className="text-2xl font-bold text-gray-900">
                  {summary ? formatNumber(summary.totalRequests) : "0"}
                </p>
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow">
            <div className="flex items-center">
              <Database className="h-8 w-8 text-purple-600" />
              <div className="ml-4">
                <h3 className="text-sm font-medium text-gray-500">
                  Active Models
                </h3>
                <p className="text-2xl font-bold text-gray-900">
                  {formatNumber(activeModels)}
                </p>
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow">
            <div className="flex items-center">
              <DollarSign className="h-8 w-8 text-yellow-600" />
              <div className="ml-4">
                <h3 className="text-sm font-medium text-gray-500">
                  Total Costs (7d)
                </h3>
                <p className="text-2xl font-bold text-gray-900">
                  {summary ? formatCost(summary.totalCostCents) : "$0.00"}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Usage Metrics */}
        {summary && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
            <div className="bg-white p-6 rounded-lg shadow">
              <div className="flex items-center mb-4">
                <TrendingUp className="h-6 w-6 text-blue-600 mr-2" />
                <h2 className="text-lg font-semibold text-gray-900">
                  Token Usage (7 days)
                </h2>
              </div>
              <div className="space-y-4">
                <div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600">Input Tokens</span>
                    <span className="text-sm font-medium text-gray-900">
                      {formatNumber(summary.totalInputTokens)}
                    </span>
                  </div>
                </div>
                <div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600">Output Tokens</span>
                    <span className="text-sm font-medium text-gray-900">
                      {formatNumber(summary.totalOutputTokens)}
                    </span>
                  </div>
                </div>
                <div className="pt-2 border-t border-gray-200">
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-600">
                      Total Tokens
                    </span>
                    <span className="text-sm font-bold text-gray-900">
                      {formatNumber(
                        summary.totalInputTokens + summary.totalOutputTokens,
                      )}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-white p-6 rounded-lg shadow">
              <div className="flex items-center mb-4">
                <Database className="h-6 w-6 text-green-600 mr-2" />
                <h2 className="text-lg font-semibold text-gray-900">
                  Top Models (7 days)
                </h2>
              </div>
              <div className="space-y-3">
                {summary.models.length > 0 ? (
                  summary.models.slice(0, 3).map((model) => (
                    <div
                      key={model.modelId}
                      className="flex justify-between items-center"
                    >
                      <span className="text-sm text-gray-600 truncate">
                        {model.modelId}
                      </span>
                      <div className="text-right">
                        <div className="text-sm font-medium text-gray-900">
                          {formatNumber(model.requests)} req
                        </div>
                        <div className="text-xs text-gray-500">
                          {formatCost(model.costCents)}
                        </div>
                      </div>
                    </div>
                  ))
                ) : (
                  <p className="text-sm text-gray-500">
                    No model usage data available
                  </p>
                )}
              </div>
            </div>
          </div>
        )}

        <div className="bg-white p-6 rounded-lg shadow">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            Recent Activity
          </h2>
          <div className="space-y-3">
            {recentEvents.length > 0 ? (
              recentEvents.map((event) => (
                <div
                  key={event.id}
                  className="flex items-center justify-between py-2 border-b border-gray-100 last:border-b-0"
                >
                  <div className="flex items-center space-x-3">
                    <div
                      className={`w-2 h-2 rounded-full ${
                        event.status === "success"
                          ? "bg-green-400"
                          : "bg-red-400"
                      }`}
                    />
                    <span className="text-sm text-gray-600">
                      API request to {event.modelId}
                    </span>
                    {event.totalCostCents && (
                      <span className="text-xs text-gray-400">
                        ({formatCost(Number(event.totalCostCents))})
                      </span>
                    )}
                  </div>
                  <span className="text-xs text-gray-400">
                    {formatTimeAgo(event.timestamp)}
                  </span>
                </div>
              ))
            ) : (
              <div className="text-center py-8">
                <Activity className="mx-auto h-12 w-12 text-gray-300 mb-4" />
                <p className="text-sm text-gray-500">
                  No recent activity found. Activity will appear here once API
                  requests are made.
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default MainArea;
