# knowtree · 知识树

单人使用的知识点管理系统：覆盖幼儿园到大学的知识点树形管理、任意节点自由关联、学习状态追踪、批注心得，以及 LLM 辅助讲解/出题/批改。

- **后端** Go (Gin + GORM + modernc SQLite) · **前端** Vue 3 + Element Plus（知识画布用 Vue Flow 定制）
- **部署** 单文件二进制（go:embed 内嵌前端，免安装直跑），另有 **桌面客户端**（Wails 原生窗口），Docker 为可选备选
- **数据** 本地 SQLite 单文件（WAL），导入导出 JSON 兜底，数据完全自主

## 文档

- [需求文档 v1.6](docs/requirements-v1.md)

## 快速开始

### 方式一：桌面客户端（推荐，双击即用）

```powershell
.\scripts\build-desktop.ps1   # 构建前端 + 编译 bin/knowtree-desktop.exe
.\bin\knowtree-desktop.exe    # 双击打开原生窗口；关窗即退出
```

- 基于 Wails v2（系统自带 WebView2，Win10/11 一般免安装），单文件、无控制台窗口
- 数据目录：exe 同级 `data\`；日志：`data\knowtree-desktop.log`
- 内置单实例锁：重复双击会聚焦已开窗口

### 方式二：单文件二进制（浏览器访问）

```powershell
# Windows（PowerShell）
.\scripts\build.ps1          # 构建前端 + 编译 bin/knowtree.exe
.\bin\knowtree.exe           # 双击亦可；默认 http://127.0.0.1:6006
```

```bash
# Linux / macOS
./scripts/build.sh
./bin/knowtree-*             # 默认 http://127.0.0.1:6006
```

首次启动自动建库建表。数据目录默认 `./data`（可用 `-data` 或环境变量 `KNOWTREE_DATA_DIR` 指定）；监听地址可用 `-addr` / `KNOWTREE_ADDR` 修改。

升级 = 替换可执行文件，数据目录原样保留。

### 方式三：Docker（可选）

```bash
docker compose up -d         # 数据挂载在 ./data
```

### 开发模式（两个终端）

```bash
# 终端 1：Go 后端（仅 API）
go run ./cmd/knowtree

# 终端 2：Vite 前端（热更新，API 自动代理到 6006 端口）
cd frontend && pnpm install && pnpm dev   # http://localhost:6006

# 桌面客户端开发调试（带控制台日志；必须带 desktop,production 标签）
go run -tags desktop,production ./cmd/knowtree-desktop -data .\data
```

## 目录结构

```
├─ cmd/knowtree/         # 服务端入口（浏览器访问）
├─ cmd/knowtree-desktop/ # 桌面客户端入口（Wails 原生窗口）
├─ internal/
│  ├─ api/              # REST handlers + DTO（nodes/edges/settings/search/version）
│  ├─ config/           # flag/env 配置
│  ├─ db/               # SQLite 打开（WAL）+ goose 迁移
│  │  └─ migrations/    # SQL 迁移文件（embed）
│  └─ models/           # GORM 模型
├─ web/                 # go:embed 前端产物（构建时由脚本填充）
├─ frontend/            # Vue 3 + Vite + Element Plus
├─ scripts/             # build.ps1 / build.sh
├─ Dockerfile           # 三阶段构建（node → go embed → alpine）
└─ docs/                # 需求文档等
```

## 计划（里程碑）

- [x] 需求文档 v1.6
- [x] M1 骨架：monorepo、SQLite 迁移、节点 CRUD API、compose 跑通 ✅
- [x] M2 知识画布：卡片拖拽 / 层级+关联连线 / 锚点拉线 / 点选成线 / 缩放平移 / 自动排布 ✅
- [x] M3 详情面板：Markdown+KaTeX 正文、教学资源、学习状态、批注（画布角标）✅
- [x] M4 LLM：设置面板+测试连接、SSE 流式讲解、生成子树预览入库、出题与批改闭环 ✅
- [ ] M5 收尾与发布（进行中）：导入导出 ✅ 备份恢复 ✅ 统计页 ✅ 撤销重做 ✅ 自更新端点 ✅ → 前端联调 + CI 发布流水线

## 发布

```bash
git tag v0.1.0 && git push origin v0.1.0
# GitHub Actions 自动构建五平台二进制 + checksums.txt 并发布 Release
# 应用内「设置 → 版本与更新」即可检测并一键自更新
```

## Git 提交约定

本地逐功能提交；远端 `origin` 已配置为 `github.com/nabin-qq273274877/knowtree`，待仓库创建后统一推送。
