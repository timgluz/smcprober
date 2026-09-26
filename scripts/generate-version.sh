#!/usr/bin/env bash
#
# Prints the version to use for a local build artifact.
#
# The VERSION file is the source of truth for releases, and the Tekton pipeline
# reads the same file. Builds on the default branch use that version as-is; other
# branches get a "-dev.<sha>" pre-release suffix so they can never collide with a
# released version.
#
# VERSION and git tags carry a leading "v" (v0.0.2); published artifact versions
# are bare SemVer (0.0.2) because Helm rejects a "v" prefix in a chart version.
#
# Note: Docker image tags only allow [A-Za-z0-9_.-], so semver build metadata
# ("+...") must not be used in a tag.

set -euo pipefail

VERSION="$(tr -d '[:space:]' < "$(dirname "$0")/../VERSION")"

if [ -z "${VERSION}" ]; then
  echo "VERSION file is empty" >&2
  exit 1
fi

if ! printf '%s' "${VERSION}" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "VERSION (${VERSION}) is not a vMAJOR.MINOR.PATCH version" >&2
  exit 1
fi

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
MAIN_BRANCH="${MAIN_BRANCH:-main}"

# Drop the leading "v" for the published artifact version.
ARTIFACT_VERSION="${VERSION#v}"

if [ "${BRANCH}" = "${MAIN_BRANCH}" ]; then
  echo "${ARTIFACT_VERSION}"
else
  echo "${ARTIFACT_VERSION}-dev.$(git rev-parse --short HEAD)"
fi
