// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import {
  BarChart3,
  Calendar,
  DollarSign,
  Activity,
  RefreshCw,
} from "lucide-react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  UsageAnalyticsService,
  type UsageSummary,
  type UsageEvent,
  type CostBreakdown,
  UsageAnalyticsServiceGetUsageSummaryRequestSchema,
  UsageAnalyticsServiceGetUsageDetailsRequestSchema,
  UsageAnalyticsServiceGetUsageCostsRequestSchema,
} from "../gen/proto/madsrc/llmgw/v1/usage_analytics_pb";
import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { useAuth } from "../hooks/useAuth";
import { formatCost, formatNumber } from "../utils/formatters";

type TimeRange = "day" | "week" | "month";

const formatTimestamp = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return "N/A";

  // Handle different types of protobuf timestamps
  if (typeof timestamp === "object" && "toDate" in timestamp) {
    return (timestamp as { toDate(): Date }).toDate().toLocaleString();
  }

  // Fallback for protobuf timestamps that have seconds/nanos
  if (typeof timestamp === "object" && "seconds" in timestamp) {
    const seconds =
      typeof timestamp.seconds === "bigint"
        ? Number(timestamp.seconds)
        : timestamp.seconds;
    return new Date(seconds * 1000).toLocaleString();
  }

  return "N/A";
};

function Analytics() {
  const [summary, setSummary] = useState<UsageSummary | null>(null);
  const [usageEvents, setUsageEvents] = useState<UsageEvent[]>([]);
  const [costBreakdown, setCostBreakdown] = useState<CostBreakdown[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>("month");
  const [selectedModel, setSelectedModel] = useState<string>("");
  const [isRefreshingEvents, setIsRefreshingEvents] = useState(false);
  const eventsIntervalRef = useRef<NodeJS.Timeout | null>(null);
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

    return createClient(UsageAnalyticsService, transport);
  }, [token]);

  const getTimeRangeDates = useCallback((range: TimeRange) => {
    const now = new Date();
    const end = new Date(now);
    const start = new Date(now);

    switch (range) {
      case "day":
        start.setDate(now.getDate() - 1);
        break;
      case "week":
        start.setDate(now.getDate() - 7);
        break;
      case "month":
        start.setMonth(now.getMonth() - 1);
        break;
    }

    return {
      start: {
        seconds: BigInt(Math.floor(start.getTime() / 1000)),
        nanos: (start.getTime() % 1000) * 1000000,
      },
      end: {
        seconds: BigInt(Math.floor(end.getTime() / 1000)),
        nanos: (end.getTime() % 1000) * 1000000,
      },
    };
  }, []);

  const fetchUsageSummary = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const { start, end } = getTimeRangeDates(timeRange);
      const request = create(
        UsageAnalyticsServiceGetUsageSummaryRequestSchema,
        {
          period: timeRange,
          start,
          end,
          modelId: selectedModel || undefined,
        },
      );

      const response = await client.getUsageSummary(request);
      setSummary(response.summary);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch usage summary",
      );
    } finally {
      setLoading(false);
    }
  }, [client, timeRange, selectedModel, getTimeRangeDates]);

  const fetchUsageDetails = useCallback(
    async (showLoading = false) => {
      try {
        if (showLoading) {
          setIsRefreshingEvents(true);
        }

        const { start, end } = getTimeRangeDates(timeRange);
        const request = create(
          UsageAnalyticsServiceGetUsageDetailsRequestSchema,
          {
            start,
            end,
            modelId: selectedModel || undefined,
            limit: 100,
            offset: 0,
          },
        );

        const response = await client.getUsageDetails(request);
        setUsageEvents(response.events);
      } catch {
        // Silently fail for usage details
      } finally {
        if (showLoading) {
          setIsRefreshingEvents(false);
        }
      }
    },
    [client, timeRange, selectedModel, getTimeRangeDates],
  );

  const fetchUsageCosts = useCallback(async () => {
    try {
      const { start, end } = getTimeRangeDates(timeRange);
      const request = create(UsageAnalyticsServiceGetUsageCostsRequestSchema, {
        period: timeRange,
        start,
        end,
        modelId: selectedModel || undefined,
      });

      const response = await client.getUsageCosts(request);
      setCostBreakdown(response.costBreakdown);
    } catch {
      // Silently fail for usage costs
    }
  }, [client, timeRange, selectedModel, getTimeRangeDates]);

  useEffect(() => {
    fetchUsageSummary();
    fetchUsageDetails();
    fetchUsageCosts();
  }, [fetchUsageSummary, fetchUsageDetails, fetchUsageCosts]);

  // Auto-refresh usage events every 10 seconds
  useEffect(() => {
    // Clear any existing interval
    if (eventsIntervalRef.current) {
      clearInterval(eventsIntervalRef.current);
    }

    // Set up new interval for auto-refresh
    eventsIntervalRef.current = setInterval(() => {
      fetchUsageDetails(false); // Don't show loading for auto-refresh
    }, 10000);

    // Cleanup interval on unmount or dependency change
    return () => {
      if (eventsIntervalRef.current) {
        clearInterval(eventsIntervalRef.current);
      }
    };
  }, [fetchUsageDetails]);

  if (loading && !summary) {
    return (
      <div className="flex-1 p-8">
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 p-8 bg-gray-50">
      <div className="max-w-7xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-4">Analytics</h1>

          <div className="flex flex-wrap gap-4 items-center">
            <div className="flex items-center space-x-2">
              <Calendar size={16} className="text-gray-500" />
              <select
                value={timeRange}
                onChange={(e) => setTimeRange(e.target.value as TimeRange)}
                className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="day">Last 24 Hours</option>
                <option value="week">Last 7 Days</option>
                <option value="month">Last 30 Days</option>
              </select>
            </div>

            <div className="flex items-center space-x-2">
              <BarChart3 size={16} className="text-gray-500" />
              <select
                value={selectedModel}
                onChange={(e) => setSelectedModel(e.target.value)}
                className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">All Models</option>
                {summary?.models.map((model) => (
                  <option key={model.modelId} value={model.modelId}>
                    {model.modelId}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>

        {error && (
          <div className="mb-6 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md">
            <strong>Error:</strong> {error}
          </div>
        )}

        {!loading && !error && !summary && (
          <div className="bg-gray-50 border border-gray-200 rounded-lg p-8 text-center">
            <div className="text-gray-500 mb-4">
              <BarChart3 className="h-16 w-16 mx-auto mb-4 text-gray-300" />
              <h3 className="text-lg font-medium text-gray-900 mb-2">
                No Analytics Data Available
              </h3>
              <p className="text-sm">
                No usage data found for the selected time period. This could
                mean:
              </p>
              <ul className="text-sm mt-2 space-y-1">
                <li>• No API requests have been made yet</li>
                <li>• The analytics service is not collecting data</li>
                <li>• The selected time range has no activity</li>
              </ul>
            </div>
          </div>
        )}

        {summary && (
          <>
            {/* Summary Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
              <div className="bg-white rounded-lg shadow p-6">
                <div className="flex items-center">
                  <Activity className="h-8 w-8 text-blue-600" />
                  <div className="ml-4">
                    <p className="text-sm font-medium text-gray-500">
                      Total Requests
                    </p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatNumber(summary.totalRequests)}
                    </p>
                  </div>
                </div>
              </div>

              <div className="bg-white rounded-lg shadow p-6">
                <div className="flex items-center">
                  <BarChart3 className="h-8 w-8 text-green-600" />
                  <div className="ml-4">
                    <p className="text-sm font-medium text-gray-500">
                      Input Tokens
                    </p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatNumber(summary.totalInputTokens)}
                    </p>
                  </div>
                </div>
              </div>

              <div className="bg-white rounded-lg shadow p-6">
                <div className="flex items-center">
                  <BarChart3 className="h-8 w-8 text-purple-600" />
                  <div className="ml-4">
                    <p className="text-sm font-medium text-gray-500">
                      Output Tokens
                    </p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatNumber(summary.totalOutputTokens)}
                    </p>
                  </div>
                </div>
              </div>

              <div className="bg-white rounded-lg shadow p-6">
                <div className="flex items-center">
                  <DollarSign className="h-8 w-8 text-yellow-600" />
                  <div className="ml-4">
                    <p className="text-sm font-medium text-gray-500">
                      Total Cost
                    </p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatCost(summary.totalCostCents)}
                    </p>
                  </div>
                </div>
              </div>
            </div>

            {/* Model Usage Table */}
            {summary.models.length > 0 && (
              <div className="bg-white rounded-lg shadow mb-8">
                <div className="px-6 py-4 border-b border-gray-200">
                  <h2 className="text-xl font-semibold text-gray-900">
                    Usage by Model
                  </h2>
                </div>
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Model
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Requests
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Input Tokens
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Output Tokens
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Cost
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {summary.models.map((model) => (
                        <tr key={model.modelId} className="hover:bg-gray-50">
                          <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                            {model.modelId}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatNumber(model.requests)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatNumber(model.inputTokens)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatNumber(model.outputTokens)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatCost(model.costCents)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* Cost Breakdown */}
            {costBreakdown.length > 0 && (
              <div className="bg-white rounded-lg shadow mb-8">
                <div className="px-6 py-4 border-b border-gray-200">
                  <h2 className="text-xl font-semibold text-gray-900">
                    Cost Breakdown
                  </h2>
                </div>
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Model
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Requests
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Input Cost
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Output Cost
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Total Cost
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {costBreakdown.map((cost) => (
                        <tr key={cost.modelId} className="hover:bg-gray-50">
                          <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                            {cost.modelId}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatNumber(cost.requests)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatCost(cost.inputCostCents)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatCost(cost.outputCostCents)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatCost(cost.totalCostCents)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* Recent Usage Events */}
            {usageEvents.length > 0 && (
              <div className="bg-white rounded-lg shadow">
                <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
                  <div>
                    <h2 className="text-xl font-semibold text-gray-900">
                      Recent Usage Events
                    </h2>
                    <p className="text-sm text-gray-500 mt-1">
                      Auto-refreshes every 10 seconds
                    </p>
                  </div>
                  <button
                    onClick={() => fetchUsageDetails(true)}
                    disabled={isRefreshingEvents}
                    className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <RefreshCw
                      className={`h-4 w-4 mr-2 ${isRefreshingEvents ? "animate-spin" : ""}`}
                    />
                    {isRefreshingEvents ? "Refreshing..." : "Refresh"}
                  </button>
                </div>
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Timestamp
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Model
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Status
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Tokens
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Cost
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {usageEvents.slice(0, 10).map((event) => (
                        <tr key={event.id} className="hover:bg-gray-50">
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatTimestamp(event.timestamp)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                            {event.modelId}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span
                              className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                                event.status === "success"
                                  ? "bg-green-100 text-green-800"
                                  : "bg-red-100 text-red-800"
                              }`}
                            >
                              {event.status}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {event.inputTokens
                              ? `${event.inputTokens}/${event.outputTokens}`
                              : "-"}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {event.totalCostCents
                              ? formatCost(Number(event.totalCostCents))
                              : "-"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

export default Analytics;
