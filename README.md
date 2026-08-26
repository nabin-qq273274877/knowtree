# 知树 · KnowTree

单人使用的知识点管理**桌面应用**：覆盖学前到博士的知识点树形管理、学习先后连线、学习状态追踪、批注心得，以及 LLM 辅助讲解/出题/批改。

- **形态**：桌面客户端（Wails v2 原生窗口 + WebView2），单文件、免安装、双击即用
- **前端** Vue 3 + Element Plus（知识画布用 Vue Flow 定制）· **后端** Go (Gin + GORM + modernc SQLite)，全部内嵌于一个 exe
- **数据** 本地 SQLite 单文件（WAL），导入导出 JSON 兜底，数据完全自主

## 功能特性

- **知识树画布**：从左到右脑图式层级布局，一键自动排布（防重叠）；支持多学段分区、按内容动态显示的顶部彩条
- **节点操作**：新建 / ＋下级 / ＋同级 / 编辑（改名、改学段、换上一级）/ 删除，位置可自由拖拽并支持撤销重做
- **学习先后连线**：上层→下层的贝塞尔曲线，建线/删线简便，连线条样式统一
- **学习状态**：未学 / 学习中 / 部分学会 / 已学会 / 部分遗忘 / 已遗忘，卡片角标与统计分布同步
- **详情抽屉**：Markdown + 公式（KaTeX）正文自动保存、资源、批注心得、AI 习题/试卷出题作答批改
- **AI 辅助**：OpenAI 兼容接口，内置 27 家服务商预设，流式讲解 + 出题批改（输出长度不限制）
- **帮助抽屉**：左下角「帮助」打开左侧 70% 宽的文档抽屉
- **统计**：知识点总数 / 掌握占比 / 关联连线 / 批注 / 练习题分布

## 快速开始

### 构建

```powershell
# Windows
.\scripts\build-desktop.ps1     # 产物 bin/knowtree-desktop-v<版本>.exe（无控制台，带图标，文件名带版本号）
```

```bash
# macOS / Linux（在对应系统上执行）
bash scripts/build-desktop.sh   # 产物 bin/knowtree-desktop-v<版本>-{macos,linux}-{arch}
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
├─ build/                # 应用图标（appicon.png / appicon.ico / appicon.icns）
├─ scripts/              # build-desktop.ps1 / build-desktop.sh
└─ docs/                 # 需求文档等
```

## 发布

在 **`main` 分支**上推送 `v*` 标签后，GitHub Actions 自动构建三平台客户端并发布 Release（分支守卫：仅 main 上的标签才触发，feature 分支误打会被静默跳过）：

```bash
git push origin main
git tag v0.1.0
git push origin v0.1.0
```

产物统一命名为 `knowtree-desktop-<版本>-<平台>`：

- Windows：`knowtree-desktop-v0.1.0-windows-amd64.exe`
- macOS：`knowtree-desktop-v0.1.0-macos-amd64.app`（Intel）/ `knowtree-desktop-v0.1.0-macos-arm64.app`（Apple Silicon）
- Linux：`knowtree-desktop-v0.1.0-linux-amd64`

打包时自动附带 `checksums.txt`（SHA256 校验和）。升级 = 替换可执行文件，`data\` 目录原样保留。

