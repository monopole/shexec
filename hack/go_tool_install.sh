#!/bin/bash
# Installs a Go binary whose full location is specified in "go_tools.go".
# The tool version is inferred from the nearby go.mod file.
# Run from top of repo, e.g.
#
#  ./hack/go_tool_install.sh conch
#
go mod download
grep $1 hack/go_tools.go | awk -F\" '{print $2}' | xargs -t go install
