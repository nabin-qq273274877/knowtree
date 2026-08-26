# 知树 · KnowTree

单人使用的知识点管理**桌面应用**：覆盖学前到博士的知识点树形管理、学习先后连线、学习状态追踪、批注心得，以及 LLM 辅助讲解/出题/批改。

- **形态**：桌面客户端（Wails v2 原生窗口 + WebView2），单文件、免安装、双击即用
- **前端** Vue 3 + Element Plus（知识画布用 Vue Flow 定制）· **后端** Go (Gin + GORM + modernc SQLite)，全部内嵌于一个 exe
- **数据** 本地 SQLite 单文件（WAL），导入导出 JSON 兜底，数据完全自主

## 文档

- [需求文档 v1.6](docs/requirements-v1.md)

## 快速开始

### 构建

```powershell
# Windows
.\scripts\build-desktop.ps1     # 产物 bin/knowtree-desktop.exe（无控制台，带图标）
```

```bash
# macOS / Linux（在对应系统上执行）
bash scripts/build-desktop.sh   # 产物 bin/knowtree-desktop-{darwin,linux}-{arch}
```

双击运行即用。数据目录在 exe 同级 `data\`；日志 `data\knowtree-desktop.log`。

### 从源码直接跑（开发调试）

```bash
# 桌面客户端（必须带 desktop,production 标签）
go run -tags desktop,production ./cmd/knowtree-desktop -data .\data

# 前端热更新调试：固定客户端端口，另开终端起 vite dev
cd frontend && pnpm dev                                  # http://localhost:6006
go run -tags desktop,production ./cmd/knowtree-desktop -addr 127.0.0.1:6010
# 浏览器打开 http://localhost:6006（API 已代理到 6010）
```

## 目录结构

```
├─ cmd/knowtree-desktop/  # 桌面客户端入口（Wails 原生窗口）
├─ internal/
│  ├─ api/               # REST handlers + DTO（nodes/edges/settings/search/version）
│  ├─ config/            # 运行配置
│  ├─ db/                # SQLite 打开（WAL）+ goose 迁移
│  │  └─ migrations/     # SQL 迁移文件（embed）
│  ├─ llm/               # OpenAI 兼容 LLM 客户端
│  └─ models/            # GORM 模型
├─ web/                  # go:embed 前端产物（构建时由脚本填充）
├─ frontend/             # Vue 3 + Vite + Element Plus
├─ build/                # 应用图标（appicon.ico / png）
├─ scripts/              # build-desktop.ps1 / build-desktop.sh
└─ docs/                 # 需求文档等
```

## 发布

推送 `v*` 标签后 GitHub Actions 自动构建三平台客户端并发布 Release：

- Windows：`knowtree-desktop-windows-amd64.exe`
- macOS：`知树-KnowTree.app`（Intel 与 Apple Silicon 各一份）
- Linux：`knowtree-desktop-linux-amd64`

升级 = 替换可执行文件，`data\` 目录原样保留。

## Git 提交约定

本地逐功能提交；远端 `origin` 已配置为 `github.com/nabin-qq273274877/knowtree`，待仓库创建后统一推送。
