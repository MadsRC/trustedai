// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { type User } from "../gen/proto/madsrc/trustedai/v1/iam_pb";

export interface AuthContextType {
  isLoggedIn: boolean;
  authMethod?: "sso" | "apikey";
  token?: string;
  user?: User;
  loading: boolean;
  error?: string;
  login: (authMethod: "sso" | "apikey", token?: string) => Promise<void>;
  logout: () => void;
  refreshCurrentUser: () => Promise<void>;
}
