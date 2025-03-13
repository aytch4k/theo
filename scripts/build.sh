#!/bin/sh
set -e

# Default values
GOOS=${GOOS:-linux}
GOARCH=${GOARCH:-amd64}
OUTPUT_NAME=${OUTPUT_NAME:-theo}
OUTPUT_DIR=${OUTPUT_DIR:-/app/dist}
CGO_ENABLED=${CGO_ENABLED:-0}

echo "Building for ${GOOS}/${GOARCH}..."

# Ensure output directory exists
mkdir -p ${OUTPUT_DIR}

# Build the application
CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags="-s -w" -o ${OUTPUT_DIR}/${OUTPUT_NAME} ./cmd/theo

# Create checksum
sha256sum ${OUTPUT_DIR}/${OUTPUT_NAME} > ${OUTPUT_DIR}/${OUTPUT_NAME}.sha256

echo "Build complete: ${OUTPUT_DIR}/${OUTPUT_NAME}"
echo "Checksum: $(cat ${OUTPUT_DIR}/${OUTPUT_NAME}.sha256)"

# Create a tar.gz archive (except for Windows)
if [ "${GOOS}" != "windows" ]; then
    tar -czf ${OUTPUT_DIR}/${OUTPUT_NAME}.tar.gz -C ${OUTPUT_DIR} ${OUTPUT_NAME}
    echo "Archive created: ${OUTPUT_DIR}/${OUTPUT_NAME}.tar.gz"
fi

# Create a zip archive for Windows
if [ "${GOOS}" = "windows" ]; then
    zip -j ${OUTPUT_DIR}/${OUTPUT_NAME}.zip ${OUTPUT_DIR}/${OUTPUT_NAME}
    echo "Archive created: ${OUTPUT_DIR}/${OUTPUT_NAME}.zip"
fi