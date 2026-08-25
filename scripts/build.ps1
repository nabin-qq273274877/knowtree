# knowtree 构建发布脚本（Windows）
# 用法：pwsh scripts/build.ps1 [-FrontendOnly]
param(
    [switch]$FrontendOnly
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot

Write-Host '==> [1/3] 构建前端 (vue-tsc + vite build)' -ForegroundColor Cyan
Push-Location "$root\frontend"
pnpm install --frozen-lockfile
if ($LASTEXITCODE -ne 0) { pnpm install }
pnpm build
if ($LASTEXITCODE -ne 0) { Pop-Location; exit 1 }
Pop-Location

Write-Host '==> [2/3] 拷贝 frontend/dist -> web/dist' -ForegroundColor Cyan
Get-ChildItem "$root\web\dist" -Force -ErrorAction SilentlyContinue |
    Where-Object { $_.Name -ne '.placeholder' } |
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path "$root\web\dist" | Out-Null
Copy-Item "$root\frontend\dist\*" "$root\web\dist\" -Recurse -Force

if ($FrontendOnly) {
    Write-Host '==> 完成（仅前端）' -ForegroundColor Green
    exit 0
}

Write-Host '==> [3/3] 编译 Go 单文件二进制' -ForegroundColor Cyan
$version = if (Test-Path "$root\VERSION") { (Get-Content "$root\VERSION" -Raw).Trim() } else { 'dev' }
$buildTime = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$ldflags = "-s -w -X main.version=$version -X main.buildTime=$buildTime"
New-Item -ItemType Directory -Force -Path "$root\bin" | Out-Null

$env:CGO_ENABLED = '0'
go build -trimpath -ldflags $ldflags -o "$root\bin\knowtree.exe" ./cmd/knowtree
if ($LASTEXITCODE -ne 0) { exit 1 }

$size = [math]::Round((Get-Item "$root\bin\knowtree.exe").Length / 1MB, 1)
Write-Host "==> 完成: bin\knowtree.exe (${size} MB, v$version)" -ForegroundColor Green
Write-Host '    运行: .\bin\knowtree.exe  (数据目录默认 .\data)'
