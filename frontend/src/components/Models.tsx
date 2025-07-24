// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { useState, useEffect, useMemo, useCallback } from "react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  Cpu,
  Settings,
  Activity,
  DollarSign,
  Zap,
  Plus,
  Edit,
  Trash2,
} from "lucide-react";
import { ModelManagementService } from "../gen/proto/madsrc/llmgw/v1/model_management_pb";
import {
  type Model,
  type Provider,
  type SupportedCredentialType,
  type OpenRouterCredential,
  ModelManagementServiceListModelsRequestSchema,
  ModelManagementServiceListSupportedProvidersRequestSchema,
  ModelManagementServiceListSupportedCredentialTypesRequestSchema,
  ModelManagementServiceListOpenRouterCredentialsRequestSchema,
  ModelManagementServiceListSupportedModelsForProviderRequestSchema,
  ModelManagementServiceCreateModelRequestSchema,
  ModelManagementServiceUpdateModelRequestSchema,
  ModelManagementServiceDeleteModelRequestSchema,
  ModelSchema,
  ModelPricingSchema,
  ModelCapabilitiesSchema,
  CredentialType,
  ProviderId,
} from "../gen/proto/madsrc/llmgw/v1/model_management_pb";
import { create } from "@bufbuild/protobuf";
import { useAuth } from "../hooks/useAuth";

function Models() {
  const [models, setModels] = useState<Model[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [credentialTypes, setCredentialTypes] = useState<
    SupportedCredentialType[]
  >([]);
  const [credentials, setCredentials] = useState<OpenRouterCredential[]>([]);
  const [supportedModels, setSupportedModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingCreate, setLoadingCreate] = useState(false);
  const [loadingCredentials, setLoadingCredentials] = useState(false);
  const [loadingSupportedModels, setLoadingSupportedModels] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModel, setShowCreateModel] = useState(false);
  const [showEditModel, setShowEditModel] = useState(false);
  const [editingModel, setEditingModel] = useState<Model | null>(null);
  const [selectedSupportedModel, setSelectedSupportedModel] =
    useState<string>("");
  const [showSupportedModels, setShowSupportedModels] = useState(false);
  const [formData, setFormData] = useState({
    id: "",
    name: "",
    providerId: "",
    credentialId: "",
    credentialType: CredentialType.UNSPECIFIED,
    inputTokenPrice: 0,
    outputTokenPrice: 0,
    supportsStreaming: false,
    supportsJson: false,
    supportsTools: false,
    supportsVision: false,
    supportsReasoning: false,
    maxInputTokens: 0,
    maxOutputTokens: 0,
    enabled: true,
    metadata: {} as { [key: string]: string },
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

  const fetchModels = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const request = create(ModelManagementServiceListModelsRequestSchema, {
        includeDisabled: true,
        providerId: "",
        credentialType: CredentialType.UNSPECIFIED,
      });
      const response = await client.listModels(request);
      setModels(response.models);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch models");
    } finally {
      setLoading(false);
    }
  }, [client]);

  const fetchSupportedProviders = useCallback(async () => {
    try {
      const request = create(
        ModelManagementServiceListSupportedProvidersRequestSchema,
        {},
      );
      const response = await client.listSupportedProviders(request);
      setProviders(response.providers);
    } catch (err) {
      console.error("Failed to fetch supported providers:", err);
    }
  }, [client]);

  const fetchSupportedCredentialTypes = useCallback(async () => {
    try {
      const request = create(
        ModelManagementServiceListSupportedCredentialTypesRequestSchema,
        {},
      );
      const response = await client.listSupportedCredentialTypes(request);
      setCredentialTypes(response.credentialTypes);
    } catch (err) {
      console.error("Failed to fetch supported credential types:", err);
    }
  }, [client]);

  const fetchCredentials = useCallback(
    async (credentialType: CredentialType) => {
      if (credentialType === CredentialType.UNSPECIFIED) {
        setCredentials([]);
        return;
      }

      try {
        setLoadingCredentials(true);

        if (credentialType === CredentialType.OPENROUTER) {
          const request = create(
            ModelManagementServiceListOpenRouterCredentialsRequestSchema,
            {
              includeDisabled: false,
            },
          );
          const response = await client.listOpenRouterCredentials(request);
          setCredentials(response.credentials);
        }
        // Add more credential type handlers here as they become available
      } catch (err) {
        console.error("Failed to fetch credentials:", err);
        setCredentials([]);
      } finally {
        setLoadingCredentials(false);
      }
    },
    [client],
  );

  const fetchSupportedModels = useCallback(
    async (providerId: string) => {
      if (!providerId) {
        setSupportedModels([]);
        return;
      }

      try {
        setLoadingSupportedModels(true);

        // Map provider ID string to ProviderId enum
        let providerIdEnum: ProviderId = ProviderId.UNSPECIFIED;
        const provider = providers.find((p) => p.id === providerId);

        if (
          provider?.providerType === "openrouter" ||
          provider?.name?.toLowerCase().includes("openrouter")
        ) {
          providerIdEnum = ProviderId.OPENROUTER;
        }

        if (providerIdEnum !== ProviderId.UNSPECIFIED) {
          const request = create(
            ModelManagementServiceListSupportedModelsForProviderRequestSchema,
            {
              providerId: providerIdEnum,
            },
          );
          const response = await client.listSupportedModelsForProvider(request);
          setSupportedModels(response.models);
        } else {
          setSupportedModels([]);
        }
      } catch (err) {
        console.error("Failed to fetch supported models:", err);
        setSupportedModels([]);
      } finally {
        setLoadingSupportedModels(false);
      }
    },
    [client, providers],
  );

  const createModel = useCallback(async () => {
    try {
      setLoadingCreate(true);
      setError(null);

      const pricing = create(ModelPricingSchema, {
        inputTokenPrice: formData.inputTokenPrice,
        outputTokenPrice: formData.outputTokenPrice,
      });

      const capabilities = create(ModelCapabilitiesSchema, {
        supportsStreaming: formData.supportsStreaming,
        supportsJson: formData.supportsJson,
        supportsTools: formData.supportsTools,
        supportsVision: formData.supportsVision,
        supportsReasoning: formData.supportsReasoning,
        maxInputTokens: formData.maxInputTokens,
        maxOutputTokens: formData.maxOutputTokens,
      });

      const modelToCreate = create(ModelSchema, {
        id: formData.id,
        name: formData.name,
        providerId: formData.providerId,
        credentialId: formData.credentialId,
        credentialType: formData.credentialType,
        pricing,
        capabilities,
        enabled: formData.enabled,
        metadata: formData.metadata,
      });

      const request = create(ModelManagementServiceCreateModelRequestSchema, {
        model: modelToCreate,
      });

      await client.createModel(request);
      setShowCreateModel(false);
      setFormData({
        id: "",
        name: "",
        providerId: "",
        credentialId: "",
        credentialType: CredentialType.UNSPECIFIED,
        inputTokenPrice: 0,
        outputTokenPrice: 0,
        supportsStreaming: false,
        supportsJson: false,
        supportsTools: false,
        supportsVision: false,
        supportsReasoning: false,
        maxInputTokens: 0,
        maxOutputTokens: 0,
        enabled: true,
        metadata: {},
      });
      fetchModels();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create model");
    } finally {
      setLoadingCreate(false);
    }
  }, [client, formData, fetchModels]);

  const updateModel = useCallback(async () => {
    if (!editingModel) return;

    try {
      setLoadingCreate(true);
      setError(null);

      const pricing = create(ModelPricingSchema, {
        inputTokenPrice: formData.inputTokenPrice,
        outputTokenPrice: formData.outputTokenPrice,
      });

      const capabilities = create(ModelCapabilitiesSchema, {
        supportsStreaming: formData.supportsStreaming,
        supportsJson: formData.supportsJson,
        supportsTools: formData.supportsTools,
        supportsVision: formData.supportsVision,
        supportsReasoning: formData.supportsReasoning,
        maxInputTokens: formData.maxInputTokens,
        maxOutputTokens: formData.maxOutputTokens,
      });

      const modelToUpdate = create(ModelSchema, {
        id: formData.id,
        name: formData.name,
        providerId: formData.providerId,
        credentialId: formData.credentialId,
        credentialType: formData.credentialType,
        pricing,
        capabilities,
        enabled: formData.enabled,
        metadata: formData.metadata,
      });

      const request = create(ModelManagementServiceUpdateModelRequestSchema, {
        model: modelToUpdate,
      });

      await client.updateModel(request);
      setShowEditModel(false);
      setEditingModel(null);
      fetchModels();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update model");
    } finally {
      setLoadingCreate(false);
    }
  }, [client, formData, editingModel, fetchModels]);

  const deleteModel = useCallback(
    async (modelId: string, modelName: string) => {
      if (!confirm(`Are you sure you want to delete the model "${modelName}"?`))
        return;

      try {
        setError(null);

        const request = create(ModelManagementServiceDeleteModelRequestSchema, {
          id: modelId,
        });

        await client.deleteModel(request);
        fetchModels();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to delete model");
      }
    },
    [client, fetchModels],
  );

  const handleCreateModel = () => {
    setFormData({
      id: "",
      name: "",
      providerId: "",
      credentialId: "",
      credentialType: CredentialType.UNSPECIFIED,
      inputTokenPrice: 0,
      outputTokenPrice: 0,
      supportsStreaming: false,
      supportsJson: false,
      supportsTools: false,
      supportsVision: false,
      supportsReasoning: false,
      maxInputTokens: 0,
      maxOutputTokens: 0,
      enabled: true,
      metadata: {},
    });
    setSelectedSupportedModel("");
    setShowSupportedModels(false);
    setSupportedModels([]);
    setShowCreateModel(true);
  };

  useEffect(() => {
    fetchModels();
    fetchSupportedProviders();
    fetchSupportedCredentialTypes();
  }, [fetchModels, fetchSupportedProviders, fetchSupportedCredentialTypes]);

  // Fetch credentials when credential type changes
  useEffect(() => {
    if (formData.credentialType === CredentialType.OPENROUTER) {
      fetchCredentials(formData.credentialType);
    } else {
      setCredentials([]);
    }
  }, [formData.credentialType, credentialTypes, fetchCredentials]);

  // Fetch supported models when provider changes
  useEffect(() => {
    if (formData.providerId) {
      fetchSupportedModels(formData.providerId);
    } else {
      setSupportedModels([]);
      setSelectedSupportedModel("");
    }
  }, [formData.providerId, fetchSupportedModels]);

  // Apply model_reference when supported model is selected
  useEffect(() => {
    if (selectedSupportedModel) {
      const model = supportedModels.find(
        (m) => m.name === selectedSupportedModel,
      );
      if (model) {
        // Get the provider name to construct model_reference
        const provider = providers.find((p) => p.id === formData.providerId);
        const providerName = provider?.name?.toLowerCase() || "";

        // Set model_reference in metadata and auto-populate ID and name from hardcoded model
        setFormData((prev) => ({
          ...prev,
          id: model.id || model.name, // Use model ID or fall back to name
          name: model.name,
          inputTokenPrice: 0, // Clear to inherit from hardcoded model
          outputTokenPrice: 0, // Clear to inherit from hardcoded model
          supportsStreaming: false, // Clear to inherit from hardcoded model
          supportsJson: false, // Clear to inherit from hardcoded model
          supportsTools: false, // Clear to inherit from hardcoded model
          supportsVision: false, // Clear to inherit from hardcoded model
          supportsReasoning: false, // Clear to inherit from hardcoded model
          maxInputTokens: 0, // Clear to inherit from hardcoded model
          maxOutputTokens: 0, // Clear to inherit from hardcoded model
          metadata: {
            ...prev.metadata,
            model_reference: `${providerName}:${model.id}`,
          },
        }));
        setShowSupportedModels(false); // Hide dropdown after selection
      }
    }
  }, [selectedSupportedModel, supportedModels, providers, formData.providerId]);

  const getProviderName = (providerId: string) => {
    const provider = providers.find((p) => p.id === providerId);
    return provider?.name || providerId;
  };

  const getStatusColor = (enabled: boolean) => {
    return enabled
      ? "text-green-600 bg-green-100"
      : "text-gray-600 bg-gray-100";
  };

  const getStatusIcon = (enabled: boolean) => {
    return enabled ? (
      <Activity className="w-4 h-4" />
    ) : (
      <Settings className="w-4 h-4" />
    );
  };

  const formatPrice = (price: number) => {
    return `$${price.toFixed(6)}`;
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
            <Cpu className="w-8 h-8 mr-3 text-blue-600" />
            Models
          </h1>
          <p className="text-gray-600 mt-2">
            Manage and monitor inference gateway models
          </p>
        </div>

        <div className="bg-white rounded-lg shadow p-8">
          <div className="text-center">
            <div className="text-red-600 mb-2">Error loading models</div>
            <div className="text-gray-600">{error}</div>
            <button
              onClick={fetchModels}
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
          <Cpu className="w-8 h-8 mr-3 text-blue-600" />
          Models
        </h1>
        <p className="text-gray-600 mt-2">
          Manage and monitor inference gateway models
        </p>
      </div>

      <div className="bg-white rounded-lg shadow">
        <div className="p-6 border-b border-gray-200">
          <div className="flex justify-between items-center">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">
                Available Models
              </h2>
              <p className="text-sm text-gray-600 mt-1">
                {models.length} model{models.length !== 1 ? "s" : ""} configured
              </p>
            </div>
            <button
              onClick={handleCreateModel}
              className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center space-x-2"
            >
              <Plus size={16} />
              <span>Add Model</span>
            </button>
          </div>
        </div>

        <div className="divide-y divide-gray-200">
          {models.map((model) => (
            <div key={model.id} className="p-6 hover:bg-gray-50">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center">
                    <h3 className="text-lg font-medium text-gray-900">
                      {model.name}
                    </h3>
                    <span
                      className={`ml-3 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(
                        model.enabled,
                      )}`}
                    >
                      {getStatusIcon(model.enabled)}
                      <span className="ml-1 capitalize">
                        {model.enabled ? "Active" : "Disabled"}
                      </span>
                    </span>
                  </div>
                  <p className="text-sm text-gray-500 font-mono mt-1">
                    ID: {model.id}
                  </p>
                  <div className="mt-2 space-y-1">
                    <p className="text-sm text-gray-600">
                      Provider: {getProviderName(model.providerId)}
                    </p>
                    <p className="text-sm text-gray-600">
                      Credential Type: {model.credentialType}
                    </p>
                    {model.pricing && (
                      <div className="flex items-center space-x-4 text-sm text-gray-600">
                        <span className="flex items-center">
                          <DollarSign className="w-3 h-3 mr-1" />
                          In: {formatPrice(model.pricing.inputTokenPrice)}
                        </span>
                        <span className="flex items-center">
                          <DollarSign className="w-3 h-3 mr-1" />
                          Out: {formatPrice(model.pricing.outputTokenPrice)}
                        </span>
                      </div>
                    )}
                    {model.capabilities && (
                      <div className="flex flex-wrap gap-2 mt-2">
                        {model.capabilities.supportsStreaming && (
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs bg-blue-100 text-blue-800">
                            <Zap className="w-3 h-3 mr-1" />
                            Streaming
                          </span>
                        )}
                        {model.capabilities.supportsJson && (
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs bg-green-100 text-green-800">
                            JSON
                          </span>
                        )}
                        {model.capabilities.supportsTools && (
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs bg-purple-100 text-purple-800">
                            Tools
                          </span>
                        )}
                        {model.capabilities.supportsVision && (
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs bg-orange-100 text-orange-800">
                            Vision
                          </span>
                        )}
                        {model.capabilities.supportsReasoning && (
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs bg-indigo-100 text-indigo-800">
                            Reasoning
                          </span>
                        )}
                      </div>
                    )}
                    {model.capabilities && (
                      <div className="text-sm text-gray-500 mt-1">
                        Max tokens:{" "}
                        {model.capabilities.maxInputTokens.toLocaleString()} in,{" "}
                        {model.capabilities.maxOutputTokens.toLocaleString()}{" "}
                        out
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  <button
                    onClick={() => {
                      setEditingModel(model);
                      setFormData({
                        id: model.id,
                        name: model.name,
                        providerId: model.providerId,
                        credentialId: model.credentialId,
                        credentialType: model.credentialType,
                        inputTokenPrice: model.pricing?.inputTokenPrice || 0,
                        outputTokenPrice: model.pricing?.outputTokenPrice || 0,
                        supportsStreaming:
                          model.capabilities?.supportsStreaming || false,
                        supportsJson: model.capabilities?.supportsJson || false,
                        supportsTools:
                          model.capabilities?.supportsTools || false,
                        supportsVision:
                          model.capabilities?.supportsVision || false,
                        supportsReasoning:
                          model.capabilities?.supportsReasoning || false,
                        maxInputTokens: model.capabilities?.maxInputTokens || 0,
                        maxOutputTokens:
                          model.capabilities?.maxOutputTokens || 0,
                        enabled: model.enabled,
                        metadata: model.metadata || {},
                      });
                      setShowEditModel(true);
                    }}
                    className="text-blue-600 hover:text-blue-900 p-1 rounded hover:bg-blue-50"
                    title="Edit model"
                  >
                    <Edit className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => deleteModel(model.id, model.name)}
                    className="text-red-600 hover:text-red-900 p-1 rounded hover:bg-red-50"
                    title="Delete model"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>

        {models.length === 0 && (
          <div className="p-12 text-center">
            <Cpu className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              No models configured
            </h3>
            <p className="text-gray-600">
              Models will appear here once they are configured in the inference
              gateway.
            </p>
          </div>
        )}
      </div>

      {/* Create Model Modal */}
      {showCreateModel && (
        <div className="fixed inset-0 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl border p-6 max-w-2xl w-full mx-4 max-h-96 overflow-y-auto">
            <h2 className="text-lg font-medium text-gray-900 mb-4">
              Add Model
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Provider
                </label>
                <select
                  value={formData.providerId}
                  onChange={(e) =>
                    setFormData({ ...formData, providerId: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="">Select Provider</option>
                  {providers.map((provider) => (
                    <option key={provider.id} value={provider.id}>
                      {provider.name}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Credential Type
                </label>
                <select
                  value={formData.credentialType}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      credentialType: parseInt(
                        e.target.value,
                      ) as CredentialType,
                      credentialId: "",
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="">Select Credential Type</option>
                  {credentialTypes.map((credType) => (
                    <option
                      key={credType.type}
                      value={credType.type.toString()}
                    >
                      {credType.displayName}
                    </option>
                  ))}
                </select>
              </div>
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Model Name
                </label>
                <div className="relative">
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => {
                      setFormData({ ...formData, name: e.target.value });
                      // Clear model reference if user is typing custom name
                      if (e.target.value !== selectedSupportedModel) {
                        setFormData((prev) => {
                          // eslint-disable-next-line @typescript-eslint/no-unused-vars
                          const { model_reference, ...restMetadata } =
                            prev.metadata;
                          return {
                            ...prev,
                            metadata: restMetadata,
                          };
                        });
                        setSelectedSupportedModel("");
                      }
                    }}
                    onFocus={() =>
                      setShowSupportedModels(supportedModels.length > 0)
                    }
                    onBlur={() =>
                      setTimeout(() => setShowSupportedModels(false), 150)
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                    placeholder="Enter model name or select from supported models"
                  />
                  {showSupportedModels && supportedModels.length > 0 && (
                    <div className="absolute z-10 w-full mt-1 bg-white border border-gray-300 rounded-md shadow-lg max-h-60 overflow-y-auto">
                      <div className="px-3 py-2 text-xs text-gray-500 border-b border-gray-200">
                        Supported Models ({supportedModels.length} available)
                      </div>
                      {loadingSupportedModels ? (
                        <div className="px-3 py-2 text-sm text-gray-500">
                          Loading models...
                        </div>
                      ) : (
                        supportedModels.map((model) => (
                          <button
                            key={model.name}
                            type="button"
                            onClick={() => {
                              setSelectedSupportedModel(model.name);
                              setFormData((prev) => ({
                                ...prev,
                                name: model.name,
                              }));
                            }}
                            className="w-full text-left px-3 py-2 text-sm hover:bg-blue-50 focus:bg-blue-50 focus:outline-none"
                          >
                            <div className="font-medium">{model.name}</div>
                            {model.metadata?.model_reference && (
                              <div className="text-xs text-gray-500">
                                References: {model.metadata.model_reference}
                              </div>
                            )}
                          </button>
                        ))
                      )}
                    </div>
                  )}
                </div>
                <p className="text-xs text-gray-500 mt-1">
                  Click in the field to see available hardcoded models, or type
                  a custom name
                </p>
                {selectedSupportedModel &&
                  formData.metadata.model_reference && (
                    <div className="mt-2 p-3 bg-blue-50 border border-blue-200 rounded-md">
                      <p className="text-sm text-blue-800">
                        <strong>Model Reference Mode:</strong> This model will
                        inherit pricing, capabilities, and token limits from the
                        hardcoded model configuration. You can override specific
                        properties below if needed.
                      </p>
                    </div>
                  )}
              </div>
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Model ID *
                </label>
                <input
                  type="text"
                  value={formData.id}
                  onChange={(e) =>
                    setFormData({ ...formData, id: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="e.g., gpt-4o, claude-sonnet, my-custom-model"
                />
                <p className="text-xs text-gray-500 mt-1">
                  This is the identifier users will use to reference this model
                  in API requests
                </p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Status
                </label>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        enabled: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    Enabled
                  </label>
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Credential ID
                </label>
                <select
                  value={formData.credentialId}
                  onChange={(e) =>
                    setFormData({ ...formData, credentialId: e.target.value })
                  }
                  disabled={!formData.credentialType || loadingCredentials}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
                >
                  <option value="">Select Credential</option>
                  {loadingCredentials ? (
                    <option disabled>Loading credentials...</option>
                  ) : (
                    credentials.map((credential) => (
                      <option key={credential.id} value={credential.id}>
                        {credential.name} ({credential.id})
                      </option>
                    ))
                  )}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Input Token Price
                </label>
                <input
                  type="number"
                  step="0.000001"
                  value={formData.inputTokenPrice}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      inputTokenPrice: parseFloat(e.target.value) || 0,
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="0.000001"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Output Token Price
                </label>
                <input
                  type="number"
                  step="0.000001"
                  value={formData.outputTokenPrice}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      outputTokenPrice: parseFloat(e.target.value) || 0,
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="0.000001"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Max Input Tokens
                </label>
                <input
                  type="number"
                  value={formData.maxInputTokens}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      maxInputTokens: parseInt(e.target.value) || 0,
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="100000"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Max Output Tokens
                </label>
                <input
                  type="number"
                  value={formData.maxOutputTokens}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      maxOutputTokens: parseInt(e.target.value) || 0,
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="4096"
                />
              </div>
            </div>

            <div className="mt-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Metadata (Key-Value Pairs)
              </label>
              <div className="space-y-2 mb-4">
                {Object.entries(formData.metadata).map(
                  ([key, value], index) => (
                    <div key={index} className="flex gap-2">
                      <input
                        type="text"
                        placeholder="Key"
                        value={key}
                        onChange={(e) => {
                          const newMetadata = { ...formData.metadata };
                          delete newMetadata[key];
                          if (e.target.value) {
                            newMetadata[e.target.value] = value;
                          }
                          setFormData({ ...formData, metadata: newMetadata });
                        }}
                        className="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                      />
                      <input
                        type="text"
                        placeholder="Value"
                        value={value}
                        onChange={(e) => {
                          const newMetadata = { ...formData.metadata };
                          newMetadata[key] = e.target.value;
                          setFormData({ ...formData, metadata: newMetadata });
                        }}
                        className="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                      />
                      <button
                        type="button"
                        onClick={() => {
                          const newMetadata = { ...formData.metadata };
                          delete newMetadata[key];
                          setFormData({ ...formData, metadata: newMetadata });
                        }}
                        className="px-3 py-2 text-red-600 border border-red-300 rounded-md hover:bg-red-50"
                      >
                        ✕
                      </button>
                    </div>
                  ),
                )}
                <button
                  type="button"
                  onClick={() => {
                    const newKey = `key${Object.keys(formData.metadata).length + 1}`;
                    setFormData({
                      ...formData,
                      metadata: { ...formData.metadata, [newKey]: "" },
                    });
                  }}
                  className="px-3 py-2 text-blue-600 border border-blue-300 rounded-md hover:bg-blue-50"
                >
                  + Add Metadata
                </button>
              </div>
            </div>

            <div className="mt-4">
              <label className="block text-sm font-medium text-gray-700 mb-3">
                Capabilities
              </label>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.supportsStreaming}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        supportsStreaming: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    Streaming
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.supportsJson}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        supportsJson: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    JSON
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.supportsTools}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        supportsTools: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    Tools
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.supportsVision}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        supportsVision: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    Vision
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.supportsReasoning}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        supportsReasoning: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    Reasoning
                  </label>
                </div>
              </div>
            </div>

            <div className="flex justify-end space-x-3 pt-6">
              <button
                onClick={() => setShowCreateModel(false)}
                className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
              >
                Cancel
              </button>
              <button
                onClick={createModel}
                disabled={
                  !formData.id ||
                  !formData.name ||
                  !formData.providerId ||
                  !formData.credentialId ||
                  !formData.credentialType ||
                  loadingCreate
                }
                className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loadingCreate ? "Creating..." : "Add Model"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit Model Modal */}
      {showEditModel && editingModel && (
        <div className="fixed inset-0 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl border p-6 max-w-2xl w-full mx-4 max-h-96 overflow-y-auto">
            <h2 className="text-lg font-medium text-gray-900 mb-4">
              Edit Model: {editingModel.name}
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Provider
                </label>
                <select
                  value={formData.providerId}
                  onChange={(e) =>
                    setFormData({ ...formData, providerId: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="">Select Provider</option>
                  {providers.map((provider) => (
                    <option key={provider.id} value={provider.id}>
                      {provider.name}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Credential Type
                </label>
                <select
                  value={formData.credentialType}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      credentialType: parseInt(
                        e.target.value,
                      ) as CredentialType,
                      credentialId: "",
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="">Select Credential Type</option>
                  {credentialTypes.map((credType) => (
                    <option
                      key={credType.type}
                      value={credType.type.toString()}
                    >
                      {credType.displayName}
                    </option>
                  ))}
                </select>
              </div>
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Model Name
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Enter model name"
                />
              </div>
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Model ID
                </label>
                <input
                  type="text"
                  value={formData.id}
                  disabled
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm bg-gray-100 text-gray-500 cursor-not-allowed"
                  placeholder="Model ID cannot be changed"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Model ID cannot be changed after creation
                </p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Status
                </label>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        enabled: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    Enabled
                  </label>
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Credential ID
                </label>
                <select
                  value={formData.credentialId}
                  onChange={(e) =>
                    setFormData({ ...formData, credentialId: e.target.value })
                  }
                  disabled={!formData.credentialType || loadingCredentials}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
                >
                  <option value="">Select Credential</option>
                  {loadingCredentials ? (
                    <option disabled>Loading credentials...</option>
                  ) : (
                    credentials.map((credential) => (
                      <option key={credential.id} value={credential.id}>
                        {credential.name} ({credential.id})
                      </option>
                    ))
                  )}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Input Token Price
                </label>
                <input
                  type="number"
                  step="0.000001"
                  value={formData.inputTokenPrice}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      inputTokenPrice: parseFloat(e.target.value) || 0,
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="0.000001"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Output Token Price
                </label>
                <input
                  type="number"
                  step="0.000001"
                  value={formData.outputTokenPrice}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      outputTokenPrice: parseFloat(e.target.value) || 0,
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="0.000001"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Max Input Tokens
                </label>
                <input
                  type="number"
                  value={formData.maxInputTokens}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      maxInputTokens: parseInt(e.target.value) || 0,
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="100000"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Max Output Tokens
                </label>
                <input
                  type="number"
                  value={formData.maxOutputTokens}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      maxOutputTokens: parseInt(e.target.value) || 0,
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="4096"
                />
              </div>
            </div>

            <div className="mt-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Metadata (Key-Value Pairs)
              </label>
              <div className="space-y-2 mb-4">
                {Object.entries(formData.metadata).map(
                  ([key, value], index) => (
                    <div key={index} className="flex gap-2">
                      <input
                        type="text"
                        placeholder="Key"
                        value={key}
                        onChange={(e) => {
                          const newMetadata = { ...formData.metadata };
                          delete newMetadata[key];
                          if (e.target.value) {
                            newMetadata[e.target.value] = value;
                          }
                          setFormData({ ...formData, metadata: newMetadata });
                        }}
                        className="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                      />
                      <input
                        type="text"
                        placeholder="Value"
                        value={value}
                        onChange={(e) => {
                          const newMetadata = { ...formData.metadata };
                          newMetadata[key] = e.target.value;
                          setFormData({ ...formData, metadata: newMetadata });
                        }}
                        className="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                      />
                      <button
                        type="button"
                        onClick={() => {
                          const newMetadata = { ...formData.metadata };
                          delete newMetadata[key];
                          setFormData({ ...formData, metadata: newMetadata });
                        }}
                        className="px-3 py-2 text-red-600 border border-red-300 rounded-md hover:bg-red-50"
                      >
                        ✕
                      </button>
                    </div>
                  ),
                )}
                <button
                  type="button"
                  onClick={() => {
                    const newKey = `key${Object.keys(formData.metadata).length + 1}`;
                    setFormData({
                      ...formData,
                      metadata: { ...formData.metadata, [newKey]: "" },
                    });
                  }}
                  className="px-3 py-2 text-blue-600 border border-blue-300 rounded-md hover:bg-blue-50"
                >
                  + Add Metadata
                </button>
              </div>
            </div>

            <div className="mt-4">
              <label className="block text-sm font-medium text-gray-700 mb-3">
                Capabilities
              </label>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.supportsStreaming}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        supportsStreaming: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    Streaming
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.supportsJson}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        supportsJson: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    JSON
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.supportsTools}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        supportsTools: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    Tools
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.supportsVision}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        supportsVision: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    Vision
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={formData.supportsReasoning}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        supportsReasoning: e.target.checked,
                      })
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">
                    Reasoning
                  </label>
                </div>
              </div>
            </div>

            <div className="flex justify-end space-x-3 pt-6">
              <button
                onClick={() => {
                  setShowEditModel(false);
                  setEditingModel(null);
                }}
                className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
              >
                Cancel
              </button>
              <button
                onClick={updateModel}
                disabled={
                  !formData.id ||
                  !formData.name ||
                  !formData.providerId ||
                  !formData.credentialId ||
                  !formData.credentialType ||
                  loadingCreate
                }
                className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loadingCreate ? "Updating..." : "Update Model"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default Models;
