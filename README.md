# knowtree · 知识树

单人使用的知识点管理系统：覆盖幼儿园到大学的知识点树形管理、任意节点自由关联、学习状态追踪、批注心得，以及 LLM 辅助讲解/出题/批改。

- **后端** Go (Gin + GORM + modernc SQLite) · **前端** Vue 3 + Element Plus（知识画布用 Vue Flow 定制）
- **部署** 单文件二进制（go:embed 内嵌前端，免安装直跑），Docker 为可选备选
- **数据** 本地 SQLite 单文件（WAL），导入导出 JSON 兜底，数据完全自主

## 文档

- [需求文档 v1.6](docs/requirements-v1.md)

## 快速开始

### 方式一：单文件二进制（推荐）

```powershell
# Windows（PowerShell）
.\scripts\build.ps1          # 构建前端 + 编译 bin/knowtree.exe
.\bin\knowtree.exe           # 双击亦可；默认 http://127.0.0.1:3000
```

```bash
# Linux / macOS
./scripts/build.sh
./bin/knowtree-*             # 默认 http://127.0.0.1:3000
```

首次启动自动建库建表。数据目录默认 `./data`（可用 `-data` 或环境变量 `KNOWTREE_DATA_DIR` 指定）；监听地址可用 `-addr` / `KNOWTREE_ADDR` 修改。

升级 = 替换可执行文件，数据目录原样保留。

### 方式二：Docker（可选）

```bash
docker compose up -d         # 数据挂载在 ./data
```

### 开发模式（两个终端）

```bash
# 终端 1：Go 后端（仅 API）
go run ./cmd/knowtree

# 终端 2：Vite 前端（热更新，API 自动代理到 3000 端口）
cd frontend && pnpm install && pnpm dev   # http://localhost:5173
```

## 目录结构

```
├─ cmd/knowtree/        # 入口
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
- [x] M1 骨架：monorepo、SQLite 迁移、节点 CRUD API、临时管理页、compose 跑通 ✅
- [ ] M2 知识画布：卡片拖拽 / 层级+关联连线 / 锚点拉线 / 缩放平移 / 自动排布（对齐 tree-link.html）
- [ ] M3 详情面板：Markdown+KaTeX 正文、教学资源、学习状态、批注
- [ ] M4 LLM：设置面板、流式讲解、生成子树、出题与批改
- [ ] M5 收尾与发布：导入导出、备份恢复、统计、撤销重做、CI 发布流水线与应用内自更新

## Git 提交约定

本地逐功能提交；远端 `origin` 已配置为 `github.com/nabin-qq273274877/knowtree`，待仓库创建后统一推送。
