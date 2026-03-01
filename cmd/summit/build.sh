#!/bin/bash
set -euo pipefail

# Compute version including commits ahead of tag
VERSION=$(git describe --tags --always --dirty)
echo "Building Summit development version $VERSION"

go build -ldflags "-s -w -X github.com/RhykerWells/Summit/common.VERSION=$VERSION"