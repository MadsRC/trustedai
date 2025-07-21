// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useState, useEffect, useMemo } from "react";
import { X } from "lucide-react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  IAMService,
  type Organization,
  IAMServiceUpdateOrganizationRequestSchema,
  OrganizationSchema,
} from "../gen/proto/madsrc/llmgw/v1/iam_pb";
import { create } from "@bufbuild/protobuf";
import { useAuth } from "../hooks/useAuth";

interface SSOConfig {
  issuer: string;
  client_id: string;
  client_secret: string;
}

interface SSOConfigModalProps {
  organization: Organization;
  isOpen: boolean;
  onClose: () => void;
  onUpdate: (org: Organization) => void;
}

function SSOConfigModal({
  organization,
  isOpen,
  onClose,
  onUpdate,
}: SSOConfigModalProps) {
  const [ssoType, setSsoType] = useState<string>("");
  const [ssoConfig, setSsoConfig] = useState<SSOConfig>({
    issuer: "",
    client_id: "",
    client_secret: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>("");
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

  useEffect(() => {
    if (isOpen) {
      setSsoType(organization.ssoType || "");
      try {
        if (organization.ssoConfig) {
          const config = JSON.parse(organization.ssoConfig) as SSOConfig;
          setSsoConfig(config);
        } else {
          setSsoConfig({
            issuer: "",
            client_id: "",
            client_secret: "",
          });
        }
      } catch {
        setSsoConfig({
          issuer: "",
          client_id: "",
          client_secret: "",
        });
      }
      setError("");
    }
  }, [isOpen, organization]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const updatedOrg = create(OrganizationSchema, {
        ...organization,
        ssoType: ssoType || "",
        ssoConfig: ssoType ? JSON.stringify(ssoConfig) : "",
      });

      const request = create(IAMServiceUpdateOrganizationRequestSchema, {
        organization: updatedOrg,
      });

      const response = await client.updateOrganization(request);

      if (response.organization) {
        onUpdate(response.organization);
        onClose();
      }
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to update SSO configuration",
      );
    } finally {
      setLoading(false);
    }
  };

  const handleSsoTypeChange = (value: string) => {
    setSsoType(value);
    if (!value) {
      setSsoConfig({
        issuer: "",
        client_id: "",
        client_secret: "",
      });
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 max-h-screen overflow-y-auto">
        <div className="flex items-center justify-between p-6 border-b">
          <h3 className="text-lg font-medium text-gray-900">
            SSO Configuration for{" "}
            {organization.displayName || organization.name}
          </h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-500"
          >
            <X size={20} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-6">
          {error && (
            <div className="bg-red-50 border border-red-200 rounded-md p-4">
              <div className="text-sm text-red-700">{error}</div>
            </div>
          )}

          <div>
            <label
              htmlFor="ssoType"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              SSO Provider Type
            </label>
            <select
              id="ssoType"
              value={ssoType}
              onChange={(e) => handleSsoTypeChange(e.target.value)}
              className="block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">None</option>
              <option value="oidc">OIDC</option>
            </select>
          </div>

          {ssoType === "oidc" && (
            <>
              <div>
                <label
                  htmlFor="issuer"
                  className="block text-sm font-medium text-gray-700 mb-2"
                >
                  OIDC Issuer URL <span className="text-red-500">*</span>
                </label>
                <input
                  type="url"
                  id="issuer"
                  value={ssoConfig.issuer}
                  onChange={(e) =>
                    setSsoConfig({ ...ssoConfig, issuer: e.target.value })
                  }
                  placeholder="https://your-oidc-provider.com"
                  className="block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  required
                />
                <p className="mt-1 text-xs text-gray-500">
                  The OIDC discovery endpoint URL (e.g.,
                  https://accounts.google.com)
                </p>
              </div>

              <div>
                <label
                  htmlFor="clientId"
                  className="block text-sm font-medium text-gray-700 mb-2"
                >
                  Client ID <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  id="clientId"
                  value={ssoConfig.client_id}
                  onChange={(e) =>
                    setSsoConfig({ ...ssoConfig, client_id: e.target.value })
                  }
                  placeholder="your-client-id"
                  className="block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  required
                />
              </div>

              <div>
                <label
                  htmlFor="clientSecret"
                  className="block text-sm font-medium text-gray-700 mb-2"
                >
                  Client Secret <span className="text-red-500">*</span>
                </label>
                <input
                  type="password"
                  id="clientSecret"
                  value={ssoConfig.client_secret}
                  onChange={(e) =>
                    setSsoConfig({
                      ...ssoConfig,
                      client_secret: e.target.value,
                    })
                  }
                  placeholder="your-client-secret"
                  className="block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  required
                />
              </div>

              <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
                <h4 className="text-sm font-medium text-blue-800 mb-2">
                  Redirect URL Configuration
                </h4>
                <p className="text-xs text-blue-700 mb-1">
                  Configure this URL in your OIDC provider:
                </p>
                <code className="text-xs bg-blue-100 px-2 py-1 rounded text-blue-800 break-all">
                  {window.location.origin}/sso/oidc/{organization.name}/callback
                </code>
              </div>
            </>
          )}

          <div className="flex space-x-3 pt-4">
            <button
              type="submit"
              disabled={loading}
              className="flex-1 bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
            >
              {loading ? "Saving..." : "Save Configuration"}
            </button>
            <button
              type="button"
              onClick={onClose}
              className="flex-1 bg-gray-300 text-gray-700 px-4 py-2 rounded-md hover:bg-gray-400 focus:outline-none focus:ring-2 focus:ring-gray-500"
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default SSOConfigModal;
