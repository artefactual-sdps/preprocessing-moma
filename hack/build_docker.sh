#!/usr/bin/env sh

set -eu

eval $(./hack/build_dist.sh shellvars)

DEFAULT_IMAGE_NAME="preprocessing-moma-worker:${VERSION_SHORT}"
TILT_EXPECTED_REF=${EXPECTED_REF:-}
IMAGE_NAME="${TILT_EXPECTED_REF:-$DEFAULT_IMAGE_NAME}"
BUILD_OPTS="${BUILD_OPTS:-}"

GO_VERSION=$(cat .go-version)
test -n "$GO_VERSION" || {
	echo "Error: .go-version is empty."
	exit 1
}

env DOCKER_BUILDKIT=1 docker build \
	-t "$IMAGE_NAME" \
	--build-arg="GO_VERSION=$GO_VERSION" \
	--build-arg="VERSION_PATH=$VERSION_PATH" \
	--build-arg="VERSION_LONG=$VERSION_LONG" \
	--build-arg="VERSION_SHORT=$VERSION_SHORT" \
	--build-arg="VERSION_GIT_HASH=$VERSION_GIT_HASH" \
	$BUILD_OPTS \
	.
