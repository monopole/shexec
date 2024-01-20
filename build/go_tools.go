//go:build tools

// This file exists only to declare dependencies on Go-based executables
// used in linting, generating code, etc.
// The versions are declared in the nearby go.mod file.

package hack

import (
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/monopole/shexec/conch"
	_ "golang.org/x/lint/golint"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/cmd/stringer"
)
