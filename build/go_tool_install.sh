#!/bin/bash

# Installs Go-based tools specified in the nearby "go_tools.go".
# The tool version is inferred from the nearby go.mod file.
# Run from top of repo, e.g.
#
#  ./build/go_tool_install.sh conch
#
go mod download
grep $1 build/go_tools.go | awk -F\" '{print $2}' | xargs -t go install
