# knowtree 需求文档

| 项 | 内容 |
| --- | --- |
| 文档版本 | v1.1 |
| 状态 | 关键决策已确认（剩余开放问题见 §8） |
| 产品名称 | knowtree（知识树） |
| 更新日期 | 2026-02-05 |

**变更记录**

| 版本 | 变更 |
| --- | --- |
| v1.0 | 初稿 |
| v1.1 | ① 确认连线区分「前置依赖/一般关联」两类；② 确认首次启动可一键生成学段-学科骨架；③ 新增 FR-9 练习题自动生成；④ 前端技术栈由 React 改为 **Vue 3**；⑤ 后端不再以打包二进制为前提，改为以 Docker 为基准的选型对比；⑥ 数据来源新增「公网公开资源调研」通道 |

---

## 1. 项目概述

### 1.1 背景

个人使用的知识点管理系统，覆盖从幼儿园到大学全部学段的知识点。核心诉求是：把庞杂的知识组织成**树形结构**（层级归属），同时支持在任意知识点之间建立**自由关联**（网状联系），并跟踪每个知识点的**学习状态**，接入大语言模型辅助讲解、内容生产与练习出题。

### 1.2 目标

- 单人使用的本地优先（local-first）应用，部署简单、数据自主可控
- 知识点的树形管理与自由关联图谱双视图
- 学习状态可视化追踪
- LLM 辅助：知识点讲解、按主题批量生成知识点子树、按知识点生成练习习题
- 一条 `docker compose up -d` 即可跑起来

### 1.3 非目标（本期不做）

- ❌ 多用户系统、注册登录、权限管理
- ❌ 云端多人协作 / 实时同步（未来可通过导出/网盘同步数据文件解决）
- ❌ 预置"全量知识点数据"——系统是**管理工具**而非数据库本身，数据来源见第 4 节
- ❌ 移动端原生 App（Web 页面保证基本可用即可）

### 1.4 使用者画像

| 角色 | 说明 |
| --- | --- |
| 唯一用户 | 即所有者本人。桌面浏览器为主要使用环境，无并发编辑场景 |

---

## 2. 功能需求

优先级定义：P0 = v1 必须；P1 = v1 尽量有；P2 = 后续版本。

### FR-1 知识树管理（P0）

树形视图展示知识点层级（如：学段 → 学科 → 章节 → 知识点 → 子知识点），深度不限。

每个节点支持：

| 操作 | 说明 | 优先级 |
| --- | --- | --- |
| 新增下级节点 | 在任意节点下添加子节点 | P0 |
| 新增同级节点 | 在当前节点后插入兄弟节点 | P0 |
| 编辑 | 修改标题（及备注） | P0 |
| 删除 | 删除节点；**含子树的级联删除必须二次确认并提示影响范围** | P0 |
| 拖拽移动 | 拖拽改变父节点（挂到其他节点下）与同层排序；拖拽过程要有明确的放置指示 | P0 |
| 展开/折叠 | 支持展开全部/折叠全部、记忆展开状态 | P1 |
| 搜索定位 | 按标题搜索并高亮定位到树中节点 | P0 |

约束：

- **首次启动可一键生成「幼儿园→大学 × 主要学科」顶层空骨架**（v1.1 已确认），之后完全自由组织
- 节点数量预期可达数千，树组件需虚拟滚动/懒加载

### FR-2 关联图谱（画布视图）（P0）

除树形视图外，提供自由布局的**图视图**：

| 操作 | 说明 | 优先级 |
| --- | --- | --- |
| 节点呈现 | 每个知识点是一个卡片节点，显示标题与状态色 | P0 |
| 节点拖拽 | 自由拖动节点位置，坐标持久化 | P0 |
| 建立连线 | 任意两个节点之间可拉线连接 | P0 |
| 连线类型 | 区分「前置依赖」（有方向，表示学习顺序）与「一般关联」（无方向）—— **v1.1 已确认要做** | P0 |
| 删除连线 | 点选连线后删除 | P0 |
| 连线拖拽 | 拖动连线端点，将其改接到其他节点（重连） | P1 |
| 画布导航 | 缩放、平移、小地图（minimap）、适应屏幕 | P1 |

约定：

- **父子关系不算连线**：树的层级由 `parent_id` 表达；连线专指跨层级的自由关联，二者分开存储
- 图视图可按"当前树中选中的分支"过滤显示范围，避免数千节点全量铺开卡死
- 从连线点击节点可打开详情（与树视图一致）

### FR-3 节点详情（P0）

点击任意节点（树视图或图视图）打开详情面板：

- 标题、所属路径（面包屑）
- **正文内容：Markdown 编辑与渲染，必须支持数学公式（LaTeX → KaTeX 渲染）**——从 K12 到大学数学公式是刚需
- 学习状态切换（见 FR-4）
- 关联教学资源列表（见 FR-5）
- 关联知识点列表（通过连线关联的节点，可点击跳转）
- 练习题区（见 FR-9）
- 上级/下级导航

### FR-4 学习状态体系（P0）

每个节点具有互斥的学习状态，共 5 种起步：

| 状态 | 含义 | 视觉 |
| --- | --- | --- |
| 未学 | 尚未开始 | 灰 |
| 学习中 | 正在学 | 蓝 |
| 部分学会 | 掌握了一部分 | 黄 |
| 已学会 | 已掌握 | 绿 |
| 已遗忘 | 曾经学会但已遗忘，需重学 | 红 |

- 树视图与图视图中均以颜色/图标直观区分
- 支持按状态筛选（如只看"未学"）、统计各状态占比
- 状态变更记录时间戳（为将来的遗忘曲线/复习提醒留数据基础）

### FR-5 教学资源关联（P0）

每个节点可挂载多条教学资源：

- 类型：**外部链接**（文档网页、视频页如 B 站/YouTube、在线课程等）为 P0；**本地上传文件**（PDF 等，存数据目录）为 P1
- 字段：标题、URL/文件、类型标签（视频/文档/习题/其他）、备注
- 视频类链接尽量支持站内嵌入式播放（如 B 站 iframe 嵌入），至少支持一键新开
- 列表可排序、可删除

### FR-6 LLM 集成（P0）

通过 **OpenAI 兼容 API** 接入大模型（天然兼容 DeepSeek、Kimi、OpenRouter、通义、本地 Ollama/vLLM 等）：

| 能力 | 说明 | 优先级 |
| --- | --- | --- |
| 讲解知识点 | 详情面板内发起，结合节点上下文（标题、正文、父级/前置节点）生成讲解；**流式输出**；支持追加提问（简易会话） | P0 |
| 生成知识点子树 | 输入主题（如"人教版八年级物理·浮力"），LLM 生成层级知识点 JSON，**预览确认后再入库**（不直接写库） | P1 |
| 生成练习习题 | 见 FR-9 | P0 |
| 用量记录 | 记录每次调用的 token 消耗，设置面板可见累计用量 | P2 |

安全与健壮性：API Key 仅存本地数据库，界面脱敏显示；请求失败有明确错误提示与重试。

### FR-7 设置面板（P0）

- **LLM 配置**：API Base URL、API Key、模型名、temperature、最大 tokens；提供「测试连接」按钮
- **数据管理**：数据目录位置展示、一键备份（下载/另存 SQLite 副本或 JSON）、从备份恢复
- **界面偏好**：暗色/亮色主题（P1）、树默认展开层数
- 所有配置即时生效，无需重启

### FR-8 数据来源方案（P0，回应"知识点数据从哪来"）

系统不预置数据，提供四条获取通道，可叠加使用：

1. **LLM 生成（主通道）**：按学段/学科/主题让 LLM 批量生成知识点树草稿 → 人工预览、删改 → 确认入库。适合快速铺开某个学科的骨架。
2. **手动编辑**：日常增删改，随学习进度逐步充实正文内容。
3. **导入自有格式**：
   - JSON（knowtree 导出格式）—— P0
   - Markdown 有序/无序列表大纲（缩进即层级，便于从笔记迁移）—— P1
4. **公网公开资源调研与导入（v1.1 新增）**：
   - 调研对象（初步清单，实施时再逐一验证可得性与许可证）：
     - [OpenKG 开放知识图谱](http://openkg.cn/) 的教育类数据集（各学科知识图谱）
     - GitHub 上开源的 K12/学科知识图谱项目（多含 CSV/JSON/Neo4j 导出）
     - 国家中小学智慧教育平台的课程/章节目录结构（作为骨架参考）
     - 教育部课程标准文本（非结构化，人工 + LLM 辅助整理）
     - Wikipedia/Wikidata 分类结构（大学阶段概念关联参考）
   - 做法：**不做通用适配器**，对选定来源写一次性转换脚本 → 转成 knowtree 标准 JSON → 走导入通道
   - 注意保留来源与许可标注字段；仅个人使用，版权风险低但仍需注明
   - 优先级：调研 P1，转换脚本按需 P2
5. **导出（数据自主权兜底）**：全量导出 JSON（P0）；按子树导出（P1）

> 结论：数据来源以「LLM 初稿 + 人工校对」为主线，公开数据集作为高质量补充，JSON 导入导出保证不被锁定。

### FR-9 练习题自动生成（P0，v1.1 新增）

针对**指定知识点**，由 LLM 自动生成练习习题：

**生成流程**

1. 详情面板点击「生成练习」→ 配置参数：题型（单选/多选/判断/填空/简答）、数量、难度
2. LLM 上下文 = 该节点标题 + 正文 + 其**前置依赖节点**摘要（保证题目考察的是学过这个点所需的前置之上）
3. 结果先进入**预览列表**，可逐题删改后再保存入库（不直接写库）
4. 同一节点可多次生成追加

**习题数据**：题干（Markdown + LaTeX）、选项、答案、解析、题型、难度、创建时间、最近作答结果

**使用方式**

- 节点详情面板内逐题作答：选择题点选、填空/简答文字输入后对照答案
- 自评对/错，记录最近结果（P1）；「显示解析」随时可看
- P2：作答结果联动学习状态建议（如连续答错 → 提示改为"学习中"）；错题汇总页

---

## 3. 技术栈决策（v1.1 修订）

### 3.1 前提变化

- 前端确定使用 **Vue**（不用 React）
- 后端**不以打包二进制为必要条件**，统一以 **Docker 部署**为基准形态来比较选型

### 3.2 前端选型（Vue 3 技术栈）

| 层 | 选型 | 理由 |
| --- | --- | --- |
| 框架 | **Vue 3（Composition API + `<script setup>`）+ Vite + TypeScript** | 用户指定；Vite 本就是 Vue 团队出品，配合最顺 |
| UI 组件库 | **Element Plus**（备选：Naive UI） | 组件最全最成熟、中文文档一流；其 `el-tree` **自带拖拽换父/排序**，直接覆盖 FR-1 大半交互 |
| 树组件 | Element Plus `el-tree`（大量子节点配懒加载） | 原生 draggable，无需自研 |
| 图画布 | **Vue Flow（`@vue-flow/core`）** | React Flow 的 Vue 官方同门实现：节点拖拽、拉线连接、**端点重连（edges-updatable）**、删除连线、minimap、缩放平移全部内置，与 FR-2 一一对应，是最关键的选型 |
| Markdown 编辑/渲染 | **md-editor-v3** | 内置 KaTeX 公式（S1 刚需）、图片粘贴上传（Q7）、暗色主题 |
| 状态管理 | Pinia | Vue 官方标准 |
| 服务端状态 | @tanstack/vue-query | 请求缓存/失效策略，顺带解决多标签一致性（S11） |
| 路由 | vue-router | 树/图/统计/设置等视图切换 |

### 3.3 后端选型对比（以 Docker 部署为基准）

既然不要求二进制分发，比较维度回到本质：**与前端的语言协同、开发效率、运行资源、SQLite/SSE 生态、维护成本**。

| 维度 | **Node.js 22 + Hono（TS）** | Bun + Hono（TS） | Go（Gin/Echo） | Python FastAPI | Java/Kotlin Spring |
| --- | --- | --- | --- | --- | --- |
| 与前端（Vue+TS）类型共享 | ✅ 同语言，DTO/校验 schema 前后端复用 | ✅ 同 | ❌ 双语言双份类型 | ❌ | ❌ |
| 开发效率（CRUD + LLM 代理） | 高 | 高 | 中（样板偏多） | 中高 | 中低（样板重） |
| 单人场景运行资源 | 够用（~50-80MB 内存） | 够用，启动最快 | **最优**（~10-20MB） | 一般 | 重（数百 MB 起） |
| Docker 镜像体积（多阶段构建） | ~150MB | ~100MB | **~20MB**（静态编译） | ~150MB | 300MB+ |
| SQLite 生态 | 成熟（better-sqlite3 + Drizzle） | 内置 bun:sqlite，很快 | 可用（modernc 纯 Go 驱动） | 一般（aiosqlite 性能平平） | 一般 |
| SSE 流式（LLM 必需） | 原生简单 | 原生简单 | 简单 | 简单 | 略繁琐 |
| 静态资源自托管（替代 nginx） | 内置（serveStatic） | 内置 | go:embed 极简 | StaticFiles | 内置 |
| 未来若想打单文件二进制 | 不成熟 | ✅ compile 一条命令 | ✅ 天然能力 | 差 | 臃肿 |
| 长期维护心智 | 大众、资料最多 | 较新但活跃 | 稳定、部署心智最低 | 稳定 | 稳定但笨重 |

**决策：采用 Node.js 22 (LTS) + Hono + TypeScript + Drizzle ORM + better-sqlite3。**

理由：

1. 本项目是**单人、低并发、IO 密集型（LLM 流式转发）**应用，任何一门语言的运行时性能都不构成瓶颈，瓶颈在开发迭代速度——TS 全栈与 Vue 前端同语言，类型定义、校验逻辑前后端复用，迭代最快
2. Docker 部署抹平了各语言分发差异后，Go 的传统优势（单二进制、超小内存）吸引力下降；而它的代价（双语言、类型重复、CRUD 样板）依然存在
3. Hono 轻、快、TS 原生，内置静态资源服务与 SSE，正好覆盖"后端自代理前端 + 流式转发 LLM"两个核心诉求；API 面小（约 20 个端点），不需要 NestJS 级别的工程框架
4. 逃生舱：接口层保持纯 REST + JSON，未来若迁移 Go，仅重写后端，前端零改动

> 备注：Bun 可作为 Node 的**直接替换运行时**（同一套代码，`bun` 启动即可，镜像更小、启动更快），待 v1 稳定后可顺手切换，不构成架构分叉。

### 3.4 部署形态

- **主形态：Docker**。多阶段构建（前端 vite build → 后端镜像内嵌产物），提供 `docker-compose.yml`，volume 挂载数据目录，一条命令起停升级
- 后端通过 `serveStatic` 托管前端构建产物，**单进程单端口，全程不依赖 nginx**
- （可选彩蛋）Bun `compile` 可额外产出单文件可执行程序，作为无 Docker 场景的分发补充，非承诺项

---

## 4. 系统架构草案

```
┌────────────────────────── 浏览器 ──────────────────────────┐
│ 树视图(el-tree)  图视图(Vue Flow)  详情面板(md-editor-v3)    │
└──────────────▲───────────────────────────▲────────────────┘
               │ 静态资源                   │ REST + SSE (/api/*)
┌──────────────┴───────────────────────────┴────────────────┐
│              单进程后端（Node 22 + Hono, TS）               │
│  serveStatic(前端产物) · REST API · LLM 代理(SSE 流式转发)  │
│                    Drizzle ORM                             │
└───────────────────────────┬───────────────────────────────┘
                            │
                    SQLite（单文件，数据目录内）
              nodes / edges / resources / exercises / settings
```

要点：

- **单进程、单端口**：后端既提供 API 也托管前端静态文件，无 nginx/无独立前端服务器
- LLM 调用由后端代理转发（API Key 不下发到浏览器），流式响应用 SSE
- 数据目录默认 `/app/data`（容器卷挂载；本地跑用环境变量 `KNOWTREE_DATA_DIR` 覆盖），内含 `knowtree.db`

### API 草案（REST，前缀 `/api`）

```
GET/POST        /nodes                    列表/新增
PATCH/DELETE    /nodes/:id                编辑/删除
POST            /nodes/:id/move           拖拽换父/排序
GET/POST        /edges                    连线列表/新建
PATCH/DELETE    /edges/:id                重连端点/改类型/删除
GET/POST        /nodes/:id/resources      资源列表/新增
DELETE          /resources/:id
GET/POST        /nodes/:id/exercises      习题列表/保存生成的习题
PATCH/DELETE    /exercises/:id            记录作答结果/删除
GET             /search?q=                标题搜索
GET/PUT         /settings                 配置读写
POST            /llm/explain              讲解（SSE 流式）
POST            /llm/generate-subtree     生成子树草稿（返回 JSON，不直接入库）
POST            /llm/generate-exercises   生成练习题草稿（返回 JSON，不直接入库）
POST            /import                   导入 JSON
GET             /export                   导出 JSON
GET             /stats                    状态统计
```

---

## 5. 数据模型草案

```sql
-- 知识点（树用 parent_id 表达，无限层级）
CREATE TABLE nodes (
  id          TEXT PRIMARY KEY,
  title       TEXT NOT NULL,
  content_md  TEXT NOT NULL DEFAULT '',       -- Markdown 正文（含 LaTeX）
  status      TEXT NOT NULL DEFAULT 'not_started'
              CHECK (status IN ('not_started','learning','partial','mastered','forgotten')),
  stage       TEXT,                           -- 学段标签（幼儿园/小学/…，冗余便于筛选）
  sort_order  REAL NOT NULL DEFAULT 0,
  parent_id   TEXT REFERENCES nodes(id) ON DELETE CASCADE,
  pos_x REAL, pos_y REAL,                     -- 图视图坐标
  status_changed_at INTEGER,                  -- 状态变更时间（为复习提醒留底）
  source_note TEXT,                           -- 来源标注（LLM/导入来源/手动），公开数据导入时注明出处与许可
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

-- 自由关联连线（不含父子关系）
CREATE TABLE edges (
  id         TEXT PRIMARY KEY,
  source_id  TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  target_id  TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  relation   TEXT NOT NULL DEFAULT 'related'
             CHECK (relation IN ('prerequisite','related')), -- 前置依赖/一般关联
  label      TEXT,
  created_at INTEGER NOT NULL,
  UNIQUE (source_id, target_id, relation)
);

-- 教学资源
CREATE TABLE resources (
  id      TEXT PRIMARY KEY,
  node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  kind    TEXT NOT NULL DEFAULT 'link',       -- link / file
  title   TEXT NOT NULL,
  url     TEXT,                               -- link 时必填
  path    TEXT,                               -- file 时相对数据目录
  note    TEXT,
  created_at INTEGER NOT NULL
);

-- 练习题（FR-9）
CREATE TABLE exercises (
  id           TEXT PRIMARY KEY,
  node_id      TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  type         TEXT NOT NULL CHECK (type IN ('single_choice','multiple_choice','true_false','fill_blank','short_answer')),
  question_md  TEXT NOT NULL,                 -- 题干（Markdown + LaTeX）
  options_json TEXT,                          -- 选择题选项
  answer_md    TEXT NOT NULL,                 -- 答案
  analysis_md  TEXT,                          -- 解析
  difficulty   INTEGER DEFAULT 3,             -- 1-5
  last_result  TEXT,                          -- right / wrong / null
  created_at   INTEGER NOT NULL
);

-- 设置（KV）
CREATE TABLE settings ( key TEXT PRIMARY KEY, value_json TEXT NOT NULL );

-- FTS5 虚表：标题+正文全文搜索（P1）
```

---

## 6. 非功能需求

| 类别 | 要求 |
| --- | --- |
| 性能 | ≥ 5000 节点时树/图交互流畅（树懒加载；图按分支过滤渲染）；常规 API < 50ms |
| 部署 | `docker compose up -d` 一条命令；仅暴露一个端口（默认 `3000`，可配）；数据落挂载卷 |
| 前端托管 | 后端 `serveStatic` 自托管，**不依赖 nginx 等任何第三方 Web 服务器** |
| 监听地址 | 默认 `127.0.0.1`（单人本机）；允许配置为 `0.0.0.0` 局域网访问，此时应启用可选的访问口令（见 §7-S9） |
| 数据安全 | SQLite 单文件；提供一键备份/恢复；升级程序不改坏旧库（Drizzle 迁移向前兼容） |
| 浏览器兼容 | 最新版 Chrome/Edge/Firefox；不考虑 IE |
| 平台 | Docker 镜像 linux/amd64 + arm64；本地直跑支持 Windows/Linux/macOS |

---

## 7. 补充：原需求未提及但必要的项

| # | 项 | 说明 | 优先级 |
| --- | --- | --- | --- |
| S1 | **数学公式渲染** | K12~大学内容离不开 LaTeX，Markdown 渲染链路必须带 KaTeX（md-editor-v3 内置） | P0 |
| S2 | **搜索** | 几千个节点没有搜索不可用；标题搜索 P0，全文搜索 P1 | P0/P1 |
| S3 | **撤销/重做** | 树/图的误删误拖很常见，编辑操作需支持 Ctrl+Z/Ctrl+Y（至少删除类操作） | P1 |
| S4 | **删除保护** | 级联删除二次确认 + 明示影响节点数；P1 可加回收站（软删除） | P0/P1 |
| S5 | **备份与恢复** | 单文件数据库也怕误删/磁盘故障，设置面板内置备份入口 | P0 |
| S6 | **导入导出** | 见 FR-8，数据自主权底线 | P0 |
| S7 | **空状态与引导** | 首次启动提供「一键生成学段骨架」「导入 JSON」「空白开始」三选项（骨架方案 v1.1 已确认） | P0 |
| S8 | **统计概览** | 各学段/学科的状态分布、总进度，一张简单的仪表页 | P1 |
| S9 | **最小访问防护** | 无用户系统 ≠ 完全裸奔：可选访问口令（局域网暴露场景） | P1 |
| S10 | **LLM Key 安全** | Key 只存本地、后端代理调用、界面脱敏 | P0 |
| S11 | **多标签页一致性** | 同一浏览器多标签同时打开时数据不互相踩（vue-query 失效策略即可，不必上 WebSocket） | P1 |
| S12 | **快捷键** | Tab/Enter 快速建兄弟/子节点、Delete 删除、Ctrl+F 搜索——高频操作的效率底线 | P1 |

---

## 8. 决策记录与开放问题

### 8.1 已拍板（v1.1）

| # | 决策 |
| --- | --- |
| D1 | 连线区分「前置依赖 / 一般关联」两类 ✅ |
| D2 | 首次启动可一键生成「学段 × 学科」顶层骨架 ✅ |
| D3 | 前端使用 **Vue 3**（Element Plus + Vue Flow + md-editor-v3 + Pinia）✅ |
| D4 | 后端以 Docker 部署为前提选型，采用 **Node.js + Hono + TypeScript**；Go 为备选逃生舱 ✅ |
| D5 | 新增练习题自动生成能力（FR-9，P0）✅ |
| D6 | 数据来源增加公网公开资源调研通道（FR-8 第 4 条）✅ |

### 8.2 仍开放（不阻塞开工）

| # | 问题 | 建议 |
| --- | --- | --- |
| Q1 | 「已遗忘」要不要触发复习提醒（间隔重复/遗忘曲线）？ | v1 先记录状态时间戳不做提醒，v2 加复习队列 |
| Q2 | 知识点粒度怎么定（多细算一个节点）？ | 不做硬性规定，靠 LLM 生成时的 prompt 约定粒度 + 使用中自然形成习惯 |
| Q3 | 视频资源站内嵌入播放还是仅跳转？ | B 站等支持 iframe 嵌入的做嵌入，其余跳转 |
| Q4 | 正文图片上传/粘贴是否纳入 v1？ | 建议 P1 支持（md-editor-v3 已具备，成本很低） |

---

## 9. 里程碑建议

| 里程碑 | 内容 | 验收标志 |
| --- | --- | --- |
| M1 骨架 | 工程搭建（Vue3 + Hono monorepo）、SQLite+迁移、节点树 CRUD API + 树视图（增删改/拖拽/搜索）、docker-compose 跑通 | 能手工录一棵树并持久化 |
| M2 图谱 | Vue Flow 图视图、连线增删/重连/分类、坐标持久化、状态着色 | 双视图联动同一份数据 |
| M3 详情 | 详情面板、Markdown+KaTeX、资源链接管理、状态切换与筛选 | 节点内容完整可用 |
| M4 LLM | 设置面板、讲解（流式）、生成子树预览入库、**练习题生成与作答** | 四大 LLM 能力跑通 |
| M5 收尾 | 导入导出、备份恢复、统计页、撤销重做、公开数据源调研报告 + 首个转换脚本、镜像发布 | compose 一条命令部署成功 |

---

*本文档为 v1.1。关键决策已锁定（§8.1），剩余开放问题（§8.2）不阻塞开发，可随实现推进逐步敲定。*
