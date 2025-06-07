<!--
SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>

SPDX-License-Identifier: AGPL-3.0-only
-->

# Identity and Access Management (IAM) in llmgw

## IAM Model

The following primitives exists, or is planned, in the IAM model:

| Name | Status |
|------|--------|
| `user` | ✅ Implemented |
| `organization` | ✅ Implemented |
| `group` | ⏳To Implement |

llmgw is a Single-Sign-On (SSO) only service and does not support local
authentication. All users must be authenticated through an external identity
provider (IdP).

There are 1 type of Single-Sign-On (SSO) supported:

- Enterprise-grade Single-Sign-On (SSO) via providers such as Okta, Microsoft
  EntraID etc.

### Regarding the Client Credentials flow

The Client Credentials flow is under consideration for use in
Machine-to-Machine communication. This will necessitate the creation of a
subtype of user called a Service Account. The reason this is listed as "under
consideration" and not as "to implement", is that this will require the
application to support local authentication (if only for Service Accounts).

This increase in scope and complexity may be warranted, but it is not a
priority at this time.

## Enterprise-grade Single-Sign-On (SSO)

> Please note that Enterprise-grade SSO is not yet fully implemented

Users are able to configure Enterprise-grade SSO at the organizational level
in llmgw, by leveraging the OIDC (OpenID Connect) protocol.

Furthermore, with Enterprise-grade SSO, users are able to configure either
support provisioning of users into their organization via Just-in-Time (JIT)
or System for Cross-domain Identity Management (SCIM) provisioning.

With JIT provisioning, users will be provisioned as they are sign in using the
organizations SSO provider. This is the default behavior.

With SCIM provisioning, users will be provisioned in bulk using the SCIM
protocol. This is a more complex setup, but allows for more control over
which users are provisioned and how they are managed. This is useful for
large organizations with many users, or organizations that have
specific requirements for user management.

llmgw are planning on supporting the following Enterprise-grade SSO
providers:

- Keycloak
- Microsoft EntraID
- Okta

| SSO Kind | Status |
| -------- | ------ |
| OIDC | ✅ Implemented |
| SAML | ⏳Not Implemented |

## Session Management

During authentication, the user proves their identity to the Identity Provider (IdP)
and the IdP issues an access token, which is then used to mint a session-token.

The session-token is provided on each subsequent request to the llmgw
API, and serves as a proof of authentication.

Within llmgw, we've opted to not use JSON Web Tokens (JWT), but instead
rely on a local session store and the creation of randomly generated session
tokens. The main reason for this is to ensure sessions can be easily revoked.

## Authorization

Authorization within llmgw is planned to be a 2 stage system, with the
first stage being the distinction between system organizations and regular
organizations and the second stage being Role Based Access Control (RBAC)
within individual organizations.

### System vs Regular Organizations

An organization can be a System Organization, meaning its users are considered
to be operators of the llmgw platform. This is a special type of
organization that is allowed to do things like listing all organizations
or all users.

A regular organization, also sometimes referred to as a "customer
organization", is only allowed to operate on resources within its own
organization. A call to list all users would only return users within
the organization.

### Role Based Access Control (RBAC)

Role Based Access Control (RBAC) will allow the grouping of users into
one or more groups, with permissions being assigned to the groups. This will
allow for a more granular level of control over what users can do within
an organization.

We are looking at using the Cedar project management of policies and
enforcement of these.
