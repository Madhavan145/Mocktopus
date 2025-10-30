#!/usr/bin/env sh
set -eu

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PROTO_DIR="$ROOT_DIR/packages/proto"
SDK_GO_DIR="$ROOT_DIR/packages/sdk-go"
SDK_TS_DIR="$ROOT_DIR/packages/sdk-ts"

if ! command -v protoc >/dev/null 2>&1; then
  echo "protoc is required but not found on PATH"
  exit 1
fi

mkdir -p "$SDK_GO_DIR" "$SDK_TS_DIR"

# Go generation (requires protoc-gen-go and protoc-gen-go-grpc on PATH)
protoc \
  -I "$PROTO_DIR" \
  --go_out "$SDK_GO_DIR" \
  --go-grpc_out "$SDK_GO_DIR" \
  "$PROTO_DIR"/*.proto

# TS generation hook placeholder (buf/protoc plugins can be wired in Phase 3)
echo "TS SDK generation scaffolded. Configure protoc TS plugin or buf in next phase."

echo "Proto generation completed."
