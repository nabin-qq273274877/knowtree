#!/usr/bin/env bash
# knowtree 构建脚本（Linux / macOS）
# 用法: ./scripts/build.sh            # 本机平台
#       GOOS=linux GOARCH=arm64 ./scripts/build.sh   # 交叉编译
set -euo pipefail
cd "$(dirname "$0")/.."

echo '==> [1/3] 构建前端'
cd frontend
pnpm install --frozen-lockfile || pnpm install
pnpm build
cd ..

echo '==> [2/3] 拷贝 frontend/dist -> web/dist'
rm -rf web/dist
mkdir -p web/dist
cp -r frontend/dist/* web/dist/

echo '==> [3/3] 编译 Go 二进制'
VERSION="$(cat VERSION 2>/dev/null || echo dev)"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
EXT=""
[ "${GOOS:-$(go env GOOS)}" = "windows" ] && EXT=".exe"
mkdir -p bin
CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
  -o "bin/knowtree-${GOOS:-$(go env GOOS)}-${GOARCH:-$(go env GOARCH)}${EXT}" \
  ./cmd/knowtree

echo "==> 完成: bin/knowtree-${GOOS:-$(go env GOOS)}-${GOARCH:-$(go env GOARCH)}${EXT}"
