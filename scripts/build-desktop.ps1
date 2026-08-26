# knowtree 桌面客户端构建脚本（Windows）
# 产物：bin\knowtree-desktop.exe（Wails 原生窗口，双击即用）
# 用法：pwsh scripts/build-desktop.ps1 [-FrontendOnly]
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

Write-Host '==> [3/3] 编译桌面客户端（windowsgui，无控制台）' -ForegroundColor Cyan
$version = if (Test-Path "$root\VERSION") { (Get-Content "$root\VERSION" -Raw).Trim() } else { 'dev' }
$buildTime = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$ldflags = "-s -w -H windowsgui -X main.version=$version -X main.buildTime=$buildTime"
New-Item -ItemType Directory -Force -Path "$root\bin" | Out-Null

$env:CGO_ENABLED = '0'
go build -trimpath -ldflags $ldflags -o "$root\bin\knowtree-desktop.exe" ./cmd/knowtree-desktop
if ($LASTEXITCODE -ne 0) { exit 1 }

$size = [math]::Round((Get-Item "$root\bin\knowtree-desktop.exe").Length / 1MB, 1)
Write-Host "==> 完成: bin\knowtree-desktop.exe (${size} MB, v$version)" -ForegroundColor Green
Write-Host '    双击运行（需要系统自带的 WebView2 运行时，Win10/11 一般已内置）'
Write-Host '    数据目录：exe 同级 data\；日志：data\knowtree-desktop.log'
