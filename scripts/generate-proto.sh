#!/usr/bin/env bash
# Generate all consumers from one schema; an optional output root enables preview.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_root="${1:-$repo_root}"
if (( $# > 1 )); then
  printf '%s\n' 'Usage: bash scripts/generate-proto.sh [output-root]' >&2
  exit 2
fi
command -v protoc >/dev/null || { printf '%s\n' 'Missing protoc (existing files use 3.21.12).' >&2; exit 1; }
go_plugin="${PROTOC_GEN_GO:-}"
if [[ -z "$go_plugin" ]]; then
  go_plugin="$(command -v protoc-gen-go || true)"
fi
if [[ -z "$go_plugin" ]] && command -v go >/dev/null; then
  go_plugin="$(go env GOPATH)/bin/protoc-gen-go"
fi
[[ -x "$go_plugin" ]] || { printf '%s\n' 'Install protoc-gen-go v1.36.11 or set PROTOC_GEN_GO to its executable path.' >&2; exit 1; }
ts_plugin="$repo_root/frontend/node_modules/.bin/protoc-gen-ts_proto"
[[ -x "$ts_plugin" ]] || { printf '%s\n' 'Missing ts-proto; run make install-frontend first.' >&2; exit 1; }

# Generate into a staging directory so generator failure leaves repository files intact.
staging_root="$(mktemp -d)"
trap 'rm -rf "$staging_root"' EXIT
cd "$repo_root"
for module in backend connection fanout; do
  mkdir -p "$staging_root/$module"
  protoc -I . --plugin="protoc-gen-go=$go_plugin" \
    --go_out="$staging_root/$module" --go_opt=paths=source_relative \
    --go_opt="Mproto/kafka/event.proto=$module/proto/kafka;kafkapb" \
    proto/kafka/event.proto
done
mkdir -p "$staging_root/frontend/src"
protoc -I . --plugin="protoc-gen-ts_proto=$ts_plugin" \
  --ts_proto_out="$staging_root/frontend/src" --ts_proto_opt=forceLong=string \
  proto/kafka/event.proto

for module in backend connection fanout; do
  mkdir -p "$output_root/$module/proto/kafka"
  cp "$staging_root/$module/proto/kafka/event.pb.go" "$output_root/$module/proto/kafka/event.pb.go"
done
mkdir -p "$output_root/frontend/src/proto/kafka"
cp "$staging_root/frontend/src/proto/kafka/event.ts" "$output_root/frontend/src/proto/kafka/event.ts"
printf 'Generated four KafkaEvent consumers in %s\n' "$output_root"
