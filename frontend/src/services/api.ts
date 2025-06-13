// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { createConnectTransport } from "@connectrpc/connect-web";
import { createPromiseClient } from "@connectrpc/connect";
import { IAMService } from "../gen/proto/madsrc/llmgw/v1/iam_connect";
import { getAuthHeaders } from "../utils/auth";

const transport = createConnectTransport({
  baseUrl: "http://localhost:9999",
  fetch: (input, init) => {
    const authHeaders = getAuthHeaders();

    return fetch(input, {
      ...init,
      mode: "cors",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...init?.headers,
        ...authHeaders,
      },
    });
  },
});

export const iamClient = createPromiseClient(IAMService, transport);
