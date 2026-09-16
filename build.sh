#!/usr/bin/env bash
set -Eeuo pipefail

APP="mavlink-doctor"
DIST="dist"
MODULE="github.com/khairnar2960/mavlink-doctor"

mkdir -p "$DIST"

TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
BUILD=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
COMMITS=$(git rev-list "${TAG}..HEAD" --count 2>/dev/null || echo 0)

if [ "$COMMITS" -eq 0 ]; then
    VERSION="$TAG"
else
    VERSION="${TAG}-dev"
fi

COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION=$(go version | awk '{print $3}')

PLATFORMS=(
    "linux amd64"
    "linux arm64"
    "darwin amd64"
    "darwin arm64"
    "windows amd64"
    "windows arm64"
)

echo "Building ${APP} version ${VERSION} (commit: ${COMMIT}, date: ${DATE})"

for platform in "${PLATFORMS[@]}"; do
    read -r GOOS GOARCH <<< "$platform"

    EXT=""
    if [ "$GOOS" = "windows" ]; then
        EXT=".exe"
    fi

    OUTPUT="${DIST}/${APP}_${VERSION}_${GOOS}_${GOARCH}${EXT}"

    echo "==> Building ${OUTPUT}..."

    GOOS=$GOOS \
    GOARCH=$GOARCH \
    CGO_ENABLED=0 \
    go build \
        -trimpath \
        -ldflags="
            -s -w
            -X '${MODULE}/internal/version.Version=${VERSION}'
            -X '${MODULE}/internal/version.Build=${BUILD}'
            -X '${MODULE}/internal/version.Commit=${COMMIT}'
            -X '${MODULE}/internal/version.BuildDate=${DATE}'
            -X '${MODULE}/internal/version.GoVersion=${GO_VERSION}'
        " \
        -o "$OUTPUT" \
        ./cmd/mavlink-doctor
done

cd "$DIST"
sha256sum * > SHA256SUMS 2>/dev/null || shasum -a 256 * > SHA256SUMS
echo "==> Checksums generated in ${DIST}/SHA256SUMS"
echo "==> Build complete!"
