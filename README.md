<!--
SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>

SPDX-License-Identifier: AGPL-3.0-only
-->

# TrustedAI

<div align="center">

![TrustedAI Logo](assets/llmgw_logo_256.png)

**Enterprise-Grade LLM Gateway for Secure AI Adoption**

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go](https://img.shields.io/badge/Go-1.24.4-blue.svg)](https://golang.org)
[![React](https://img.shields.io/badge/React-18-blue.svg)](https://reactjs.org)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)](https://github.com/MadsRC/trustedai)

*Bringing enterprise security and governance to Large Language Model deployments*

</div>

## 🚀 Why TrustedAI?

Organizations adopting AI face a critical challenge: **How do you harness the power of LLMs while maintaining enterprise security, compliance, and cost control?**

TrustedAI solves this by providing a secure, observable, and governable gateway between your organization and LLM providers. No more shadow IT, ungoverned API keys, or compliance nightmares.

### The Problem We Solve

- 🔒 **Security Gaps**: Direct API access bypasses enterprise security controls
- 💸 **Cost Overruns**: No visibility into usage patterns or cost attribution  
- 📊 **Compliance Blind Spots**: Lack of audit trails and usage monitoring
- 🏢 **Governance Challenges**: Scattered API keys and ungoverned access
- 🔧 **Developer Friction**: Complex integration patterns for enterprise features

## ✨ Key Features

### 🛡️ Enterprise Security First
- **SSO-Only Authentication**: OIDC/SAML integration with Okta, EntraID, Keycloak
- **Zero Local Passwords**: Eliminate credential vulnerabilities
- **Session Management**: Secure token-based access with easy revocation
- **Multi-Tenant Architecture**: Organization-level isolation and controls

### 📊 Complete Observability
- **Real-Time Analytics**: Usage patterns, cost tracking, and performance metrics
- **OpenTelemetry Integration**: Seamless monitoring stack integration
- **Detailed Audit Trails**: Full request/response logging for compliance
- **Cost Attribution**: Per-user, per-model billing and quota management

### 🏗️ Production-Ready Architecture
- **Dual-Plane Design**: 
  - **Control Plane**: Management APIs, user interface, and administration
  - **Data Plane**: High-performance LLM request routing and processing
- **Multi-Provider Support**:
  - **Frontend APIs**: OpenAI, Anthropic (Gemini planned)
  - **Backend Routing**: OpenRouter with planned support for Bedrock, Vertex AI
- **Database-Driven**: PostgreSQL-backed configuration and state management

### ⚡ Developer Experience
- **Modern Web UI**: React/TypeScript dashboard with real-time updates
- **ConnectRPC APIs**: Type-safe, high-performance API layer with HTTP/2
- **Comprehensive Testing**: Unit, integration, and acceptance test suites
- **Docker Compose**: Simple local development setup

## 🔮 Roadmap

We're actively developing these enterprise-critical features:

- 🛡️ **AI Guardrails**: Content filtering, safety controls, and policy enforcement
- 📝 **Prompt Management**: Centralized prompt templates and version control
- 🎯 **Advanced Routing**: Load balancing, failover, and A/B testing
- 📊 **Enhanced Analytics**: Custom dashboards and reporting
- 🔗 **More Integrations**: Bedrock, Vertex AI, Azure OpenAI Service

## 🚀 Quick Start

### Prerequisites

- [Mise](https://mise.jdx.dev/) for tool management
- [Docker & Docker Compose](https://docs.docker.com/get-docker/) for local services

### 1. Clone and Setup

```bash
git clone https://github.com/MadsRC/trustedai.git
cd trustedai

# Install tools and dependencies
mise install
mise run dev/bootstrap.sh
```

### 2. Start Infrastructure

```bash
# Start PostgreSQL, Keycloak, and OTEL Collector
docker compose up -d

# Wait for services to be ready (especially database)
docker compose ps
```

### 3. Run TrustedAI

```bash
# Start the backend (control plane + data plane)
DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres" \
LLMGW_BASE_URL="http://localhost:5173" \
go run cmd/trustedai/main.go

# In another terminal, start the frontend
cd frontend
npm run dev
```

### 4. Access the Dashboard

Open [http://localhost:5173](http://localhost:3000) and sign in using:
- **SSO Provider**: `http://localhost:8080/realms/testrealm01` 
- **Admin Console**: [http://localhost:8080/admin](http://localhost:8080/admin) (admin/admin)

## 📚 Documentation

- **[IAM Overview](docs/iam.md)** - Identity and Access Management
- **[Model Aliasing](docs/model_aliasing.md)** - Model routing and aliases
- **[Testing Guide](TESTING.md)** - Running tests and contributing
- **[Development Setup](CLAUDE.md)** - Developer instructions

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐
│   React App     │    │  Control Plane  │
│  (Port 5173)    │◄──►│   (Port 9999)   │
└─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Data Plane    │
                       │   (Port 8081)   │
                       └─────────────────┘
                                │
                                ▼
                    ┌─────────────────────────┐
                    │    LLM Providers        │
                    │ OpenAI │ Anthropic │... │
                    └─────────────────────────┘
```

**Control Plane**: User management, configuration, analytics, and web UI
**Data Plane**: High-performance LLM request routing and response handling

## 🤝 Contributing

We welcome contributions! TrustedAI is in active development and we're looking for:

- 🐛 **Bug Reports**: Help us identify and fix issues
- 💡 **Feature Requests**: Share your enterprise AI governance needs  
- 🔧 **Code Contributions**: Check our issues for good first contributions
- 📖 **Documentation**: Help improve our guides and examples

### Development Workflow

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Follow our conventions**: Read [CLAUDE.md](CLAUDE.md) for coding standards
4. **Test your changes**: `mise run test:unit`
5. **Format code**: `mise run format`
6. **Lint code**: `mise run lint`
7. **Commit with conventional commits**: `feat: add amazing feature`
8. **Open a Pull Request**

## 📋 Requirements

- **Go**: 1.24.4+
- **Node.js**: 24.2.0+
- **PostgreSQL**: 17+
- **Docker**: For local development

## 📄 License

TrustedAI is licensed under the [GNU Affero General Public License v3.0](LICENSE).

We chose AGPL-3.0 because we believe enterprise AI infrastructure should be transparent, auditable, and community-driven. This ensures that improvements to TrustedAI benefit everyone in the ecosystem.

## 🚧 Project Status

**TrustedAI is currently in Alpha**. We're actively developing core features and welcome feedback from enterprise teams tackling AI governance challenges.

- ✅ **Core Architecture**: Control/Data plane separation
- ✅ **Authentication**: SSO integration with OIDC
- ✅ **Multi-Provider**: OpenAI, Anthropic, OpenRouter support
- ✅ **Observability**: Usage tracking and metrics
- 🚧 **Guardrails**: In development
- 🚧 **Prompt Management**: Planned
- 🚧 **Advanced Routing**: Planned

## 💬 Community & Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/MadsRC/trustedai/issues)
- **Discussions**: [Share ideas and ask questions](https://github.com/MadsRC/trustedai/discussions)

---

<div align="center">

**Ready to bring enterprise security to your AI deployment?**

⭐ **Star this repo** if TrustedAI solves a problem you're facing!

[Get Started](#-quick-start) • [View Issues](https://github.com/MadsRC/trustedai/issues) • [Join Discussions](https://github.com/MadsRC/trustedai/discussions)

</div>
