// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package llmgw

import (
	"embed"
	"io/fs"
)

//go:embed frontend/dist
var frontendFS embed.FS

// GetFrontendFS returns the embedded frontend filesystem
func GetFrontendFS() fs.FS {
	// Strip the "frontend/dist" prefix from the embedded filesystem
	subFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		panic("failed to create sub filesystem: " + err.Error())
	}
	return subFS
}
