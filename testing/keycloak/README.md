<!--
SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>

SPDX-License-Identifier: AGPL-3.0-only
-->

# Keycloak Testing Readme

These are files exported from Keycloak and can be reused for testing.

It includes a test realm called `testrealm01` containing the following
resources:

- Users:
  - `user1` with password `password`
  - `user2` with passwird `password
- Clients
  - OIDC Client with ID `client01` with Authorization Code and Device Code
    flows enabled. Client secret can be found in the keycloak interface

Additionally includes a test realm called `testrealm02` containing the following
resources:

- Users:
  - `admin@localhost` with password `password`
  - `user1` with password `password`
- Clients
  - OIDC Client with ID `client01` with Authorization Code and Device Code
    flows enabled. Client secret can be found in the keycloak interface
