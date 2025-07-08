<!-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk> -->
<!--  -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Instructions for Claude Code

- Ensure code is formatted by running `mise run format`
- Lint code with `mise run lint`
- Run unit tests with `mise run test:unit`
- Read [TESTING.md](TESTING.md) before any sort of modification or creation of tests
- Git commit messages must follow conventional commits
- When encountering the term "provider" in the codebase, know that there are multiple variations of
  the term in use:
  - API providers, responsible for managing public facing API's as part of our dataplane. This code
    lives in `internal/api/dataplane/providers`
  - Model providers, responsible for hosting and providing an API for Large Language Models. These
    are not implemented in our codebase, but our codebase does interact with them via the GAI
    GAI library. Furthermore, code that deals with "models" may refer to providers, in which
    case the term refers to a model provider

## Golang instructions

- Make sure the assert or require package from testify is used
- Use `any` instead of `interface{}`
