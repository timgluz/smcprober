#!/usr/bin/env bash

set -e

VERSION=$(cat VERSION)
COMMIT_COUNT=$(git rev-list --count HEAD)
COMMIT_SHA=$(git rev-parse --short HEAD)
BRANCH=$(git rev-parse --abbrev-ref HEAD)


# Check for unstaged changes
if ! git diff --quiet || ! git diff --cached --quiet; then
  UNSTAGED="-unstaged"
else
  UNSTAGED=""
fi

ARTIFACT_VERSION=""
if [ "$BRANCH" = "main" ]; then
  ARTIFACT_VERSION="${VERSION}+${COMMIT_COUNT}${UNSTAGED}"
else
  ARTIFACT_VERSION="${VERSION}-dev.${COMMIT_SHA}${UNSTAGED}"
fi

if semver "$ARTIFACT_VERSION" > /dev/null 2>&1; then
  echo "$ARTIFACT_VERSION"
else
  echo "Invalid semver"
  exit 1
fi
