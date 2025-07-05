<!-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk> -->
<!--  -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Instructions for Claude Code

- Ensure code is formatted by running `mise run format`
- Lint code with `mise run lint`
- Run unit tests with `mise run test:unit`
- Read [TESTING.md](TESTING.md) before any sort of modification or creation of tests
- git commit messages must follow conventional commits

## Golang instructions

- Make sure the assert or require package from testify is used
- Use `any` instead of `interface{}`
