#!/usr/bin/env bash
# knowtree 桌面客户端构建脚本（macOS / Linux）
# macOS 产物可直接运行；如需 .app 包可参考 .github/workflows/release.yml 的打包步骤
# 用法：bash scripts/build-desktop.sh [--frontend-only]
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> [1/3] 构建前端 (vue-tsc + vite build)"
cd "$root/frontend"
pnpm install --frozen-lockfile || pnpm install
pnpm build

echo "==> [2/3] 拷贝 frontend/dist -> web/dist"
cd "$root"
rm -rf web/dist && mkdir -p web/dist
cp -r frontend/dist/* web/dist/

if [[ "${1:-}" == "--frontend-only" ]]; then
  echo "==> 完成（仅前端）"
  exit 0
fi

echo "==> [3/3] 编译桌面客户端"
version="v0.1.0"
[ -f "$root/VERSION" ] && version="$(tr -d '[:space:]' < "$root/VERSION")"
build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
mkdir -p "$root/bin"

os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Darwin) out="knowtree-desktop-macos-$arch" ;;
  Linux)  out="knowtree-desktop-linux-$arch" ;;
  *) echo "unsupported os: $os"; exit 1 ;;
esac

go build -tags "desktop,production" -trimpath \
  -ldflags "-s -w -X main.version=$version -X main.buildTime=$buildTime" \
  -o "$root/bin/$out" ./cmd/knowtree-desktop

size=$(du -h "$root/bin/$out" | cut -f1 | tr -d '\n')
echo "==> 完成: bin/$out ($size, v$version)"
