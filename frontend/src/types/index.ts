// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {
  User,
  Organization,
  APIToken,
} from "../gen/proto/madsrc/llmgw/v1/iam_pb";

export type { User, Organization, APIToken };

export interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  sessionToken: string | null;
}

export interface AppState {
  auth: AuthState;
}
