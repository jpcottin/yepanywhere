#!/usr/bin/env bash
set -e

VERSION="$1"

if [ -z "$VERSION" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

if [[ ! "$VERSION" =~ ^[0-9] ]]; then
  echo "Error: Version must start with a number (e.g., 0.1.0, not v0.1.0)"
  exit 1
fi

# Fail if working tree is dirty
if ! git diff-index --quiet HEAD --; then
  echo "Error: Working tree has uncommitted changes. Please commit or stash first."
  git diff --stat
  exit 1
fi

TAG="bridge-v${VERSION}"

git tag "$TAG"
git push origin "$TAG"

echo "Created and pushed tag $TAG"
