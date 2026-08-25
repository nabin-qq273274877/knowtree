# knowtree 需求文档

| 项 | 内容 |
| --- | --- |
| 文档版本 | v1.6 |
| 状态 | 关键决策已确认（剩余开放问题见 §8） |
| 产品名称 | knowtree（知识树） |
| 更新日期 | 2026-02-05 |

**变更记录**

| 版本 | 变更 |
| --- | --- |
| v1.0 | 初稿 |
| v1.1 | ① 确认连线区分「前置依赖/一般关联」；② 确认一键生成学段-学科骨架；③ 新增练习题生成；④ 前端定为 Vue 3；⑤ 以 Docker 为基准做后端选型对比；⑥ 数据来源新增公网公开资源通道 |
| v1.2 | ① 树展示废弃 el-tree 缩进列表方案，改为**统一知识画布**（空间卡片 + 连线，交互对齐 `D:\Project\workspace\tree-link.html` 参考实现）；② 后端**确定选用 Go**（接受前后端双语言，否决 Node.js 后端）；③ FR-9 增加 **LLM 批改**能力；④ 新增 FR-10 知识点**批注（学习心得）**功能 |
| v1.3 | 新增 §3.6 **数据库专项**：回应「SQLite 卡表」疑虑，明确 WAL + 写串行化 + 快照备份等保障方案，数据库定稿 SQLite |
| v1.4 | 部署形态反转：**单文件二进制升为主形态**（go:embed 内嵌前端，双击即用免安装），Docker 降为可选备选；补充数据目录策略与升级方式 |
| v1.5 | 新增 FR-11 **版本检测与升级**：版本信息展示、检查更新、应用内自更新（SHA256 校验 + 备份 + 原子替换 + 回滚）、更新源可配置；M5 纳入 GitHub Actions 发布流水线 |
| v1.6 | 明确**知识点内容统一采用 Markdown 格式**：正文/批注/习题/LLM 讲解输出全走 Markdown（含 LaTeX 扩展），数据库存原始文本、渲染在前端完成 |

---

## 1. 项目概述

### 1.1 背景

个人使用的知识点管理系统，覆盖从幼儿园到大学全部学段的知识点。核心诉求是：把庞杂的知识组织成**树形结构**（层级归属），同时支持在任意知识点之间建立**自由关联**（网状联系），跟踪每个知识点的**学习状态**，随时写下**学习批注**，并接入大语言模型辅助讲解、出题、批改。

### 1.2 目标

- 单人使用的本地优先（local-first）应用，部署简单、数据自主可控
- 一张**空间画布**上同时呈现知识树的层级连线与知识点间的自由关联
- 学习状态可视化追踪，学习心得随手批注
- LLM 辅助：讲解、生成子树、生成练习题、批改作业
- **单个可执行文件即可运行**（前端经 go:embed 内嵌，无需安装任何依赖）；Docker 为可选备选

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

### FR-1 知识画布（核心视图，P0）

> 参考实现：`D:\Project\workspace\tree-link.html`。**树不是缩进列表，而是空间画布上的连线树**：知识点是可自由摆放的卡片节点，层级归属与自由关联都以贝塞尔曲线呈现。

**呈现**

| 项 | 要求 | 优先级 |
| --- | --- | --- |
| 节点卡片 | 圆角卡片，显示标题与学习状态色边框/角标 | P0 |
| 层级连线 | 父子关系渲染为贝塞尔曲线（样式区别于关联线）；数据来源是 `parent_id` 派生，不在 edges 表存储 | P0 |
| 节点拖拽 | 自由拖动摆放，坐标持久化；连线实时跟随 | P0 |
| 画布缩放/平移 | 滚轮缩放（有上下限）、拖空白处平移、适应屏幕、minimap 小地图 | P0/P1* |
| 自动排布 | 一键按层级整理成规整的树形布局（逐层排列起底，可用 dagre/elkjs 增强） | P0 |
| 空状态引导 | 首次进入给出创建/导入/生成骨架入口 | P0 |

*缩放平移 P0，minimap P1。

**节点操作**

| 操作 | 说明 | 优先级 |
| --- | --- | --- |
| 新增下级 | 在任意节点下添加子节点，**画布上自动出现父→子连线** | P0 |
| 新增同级 | 在当前节点后插入兄弟节点（同父） | P0 |
| 编辑 | 双击节点或经详情面板修改标题 | P0 |
| 删除 | 删除节点；含子树的级联删除必须二次确认并提示影响范围；相关连线一并清理 | P0 |
| 点击选中 | 单击选中（高亮样式），打开详情面板 | P0 |
| 搜索定位 | Ctrl+F 搜索标题，定位并居中高亮该节点 | P0 |

补充：

- **首次启动可一键生成「幼儿园→大学 × 主要学科」顶层空骨架**（已确认），之后完全自由组织
- 节点数量预期可达数千：画布默认只展开**当前聚焦分支 + 邻近层级**，其余折叠为聚合占位节点（点击展开）；配合搜索跳转使用（P1 细化）

### FR-2 关联连线（P0）

在层级之外，任意两个节点之间可建立自由关联，交互对齐参考实现：

| 操作 | 说明 | 优先级 |
| --- | --- | --- |
| 锚点拉线 | hover 节点出现四向锚点，从锚点拖出虚线到目标节点松手即连 | P0 |
| 点选连线 | 先点选源节点再点目标节点即建立连线（连续多点互联） | P0 |
| 连线类型 | 「前置依赖」（带方向箭头，表示学习顺序）/「一般关联」（无向）；新建时可指定，默认一般关联 | P0 |
| 删除连线 | 点击连线选中（可见细线 + 底层透明宽命中线），Delete 或右键删除 | P0 |
| 重连端点 | 拖动连线端点改接到其他节点 | P1 |
| 防呆 | 禁止自环；同一对节点同类型连线去重（重复时 toast 提示） | P0 |
| 视觉规范 | 层级线 / 前置依赖 / 一般关联三种线型颜色与虚实明确区分，选中态统一高亮色 | P0 |

### FR-3 节点详情面板（P0）

> **知识点内容统一采用 Markdown 格式**（v1.6 明确）：正文、批注、习题题干/答案/解析、LLM 讲解输出的存储与展示均为 Markdown（含 LaTeX 数学公式扩展）。数据库只存**原始 Markdown 文本**，不做 HTML 落库，渲染统一在前端完成——保证导入导出无损、数据可被任何编辑器直接使用。

点击节点打开详情面板：

- 标题、所属路径（面包屑）
- **正文：Markdown 编辑与渲染，必须支持数学公式（LaTeX → KaTeX）**——从 K12 到大学数学公式是刚需
- 学习状态切换（见 FR-4）
- 关联教学资源列表（见 FR-5）
- 关联知识点列表（连线关联的节点，可点击跳转居中）
- 练习题区（见 FR-9）
- **批注区（见 FR-10）**
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

- 画布节点卡片与详情面板均直观区分状态
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
| 讲解知识点 | 详情面板内发起，结合节点上下文（标题、正文、父级/前置节点、批注）生成讲解；**流式输出**；支持追加提问（简易会话） | P0 |
| 生成知识点子树 | 输入主题（如"人教版八年级物理·浮力"），LLM 生成层级知识点 JSON，**预览确认后再入库**（不直接写库） | P1 |
| 出题 | 见 FR-9 | P0 |
| 批改 | 见 FR-9 | P0 |
| 用量记录 | 记录每次调用的 token 消耗，设置面板可见累计用量 | P2 |

安全与健壮性：API Key 仅存本地数据库，界面脱敏显示；请求失败有明确错误提示与重试。

### FR-7 设置面板（P0）

- **LLM 配置**：API Base URL、API Key、模型名、temperature、最大 tokens；提供「测试连接」按钮
- **数据管理**：数据目录位置展示、一键备份（下载/另存 SQLite 副本或 JSON）、从备份恢复
- **界面偏好**：暗色/亮色主题（P1）、画布默认缩放、聚焦分支展开深度
- **关于与更新**：当前版本信息、「检查更新」入口、更新源配置（见 FR-11）
- 所有配置即时生效，无需重启

### FR-8 数据来源方案（P0）

系统不预置数据，提供四条获取通道，可叠加使用：

1. **LLM 生成（主通道）**：按学段/学科/主题让 LLM 批量生成知识点树草稿 → 人工预览、删改 → 确认入库。适合快速铺开某个学科的骨架。
2. **手动编辑**：日常增删改，随学习进度逐步充实正文内容。
3. **导入自有格式**：
   - JSON（knowtree 导出格式）—— P0
   - Markdown 有序/无序列表大纲（缩进即层级，便于从笔记迁移）—— P1
4. **公网公开资源调研与导入**：
   - 调研对象（初步清单，实施时逐一验证可得性与许可证）：
     - [OpenKG 开放知识图谱](http://openkg.cn/) 的教育类数据集（各学科知识图谱）
     - GitHub 上开源的 K12/学科知识图谱项目（多含 CSV/JSON/Neo4j 导出）
     - 国家中小学智慧教育平台的课程/章节目录结构（作为骨架参考）
     - 教育部课程标准文本（非结构化，人工 + LLM 辅助整理）
     - Wikipedia/Wikidata 分类结构（大学阶段概念关联参考）
   - 做法：**不做通用适配器**，对选定来源写一次性转换脚本 → 转成 knowtree 标准 JSON → 走导入通道
   - 注意保留来源与许可标注字段（`source_note`）；仅个人使用，版权风险低但仍需注明
   - 优先级：调研 P1，转换脚本按需 P2
5. **导出（数据自主权兜底）**：全量导出 JSON（P0）；按子树导出（P1）

> 结论：数据来源以「LLM 初稿 + 人工校对」为主线，公开数据集作为高质量补充，JSON 导入导出保证不被锁定。

### FR-9 练习题：生成、作答、批改（P0）

针对**指定知识点**的完整练习闭环：出题 → 作答 → 批改 → 记录。

**① 生成**

1. 详情面板点击「生成练习」→ 配置参数：题型（单选/多选/判断/填空/简答）、数量、难度
2. LLM 上下文 = 该节点标题 + 正文 + 其**前置依赖节点**摘要（保证题目站在已学前置之上）
3. 结果先进入**预览列表**，可逐题删改后再保存入库（不直接写库）
4. 同一节点可多次生成追加

**② 作答**

- 选择/判断题点选选项；填空/简答题文字输入
- 作答即保存草稿，可中途退出继续

**③ 批改（v1.2 新增）**

| 题型 | 批改方式 |
| --- | --- |
| 单选/多选/判断 | **规则判分**（比对答案即可，不走 LLM），即时反馈对错 |
| 填空/简答 | 提交后由 **LLM 批改**：判定 对/部分对/错，给出得分（0-100 或对错）与**点评反馈**（指出答对点、遗漏点、错误原因），必要时附标准答案对照 |

- 批改结果持久化：用户作答内容、判定结果、得分、LLM 点评、批改时间
- 「显示解析」随时可看（题目自带解析）
- P2：作答结果联动学习状态建议（如简答连续判错 → 提示将节点改为"学习中"）；错题汇总页

### FR-10 知识点批注（P0，v1.2 新增）

让学习者**随时随地记下心得体会**，与教材性正文分开管理：

- 每个节点可添加**多条批注**（一学一遍有一遍的感悟，按时间累积）
- 批注为短篇 Markdown（支持 LaTeX），字段：内容、创建时间、更新时间
- 详情面板内：顶部常驻**快速输入框**（回车即存），下方时间倒序批注列表，可编辑/删除
- 画布节点卡片上有批注数量角标，一眼看出哪些节点有心得
- LLM 讲解时会参考批注内容（理解学习者的困惑点）
- P2：划词批注（针对正文中某段落的锚定批注）

### FR-11 版本检测与升级（P0，v1.5 新增）

单文件二进制形态的配套能力：应用自己知道有没有新版本，并能安全地完成升级。

| 能力 | 说明 | 优先级 |
| --- | --- | --- |
| 版本信息展示 | 当前版本号、构建时间、Git commit（编译期 `-ldflags` 注入），设置面板「关于」区可见 | P0 |
| 检查更新 | 「检查更新」按钮：查询更新源最新 Release，按语义化版本比对；有新版时展示版本号 + 更新说明（Release Notes） | P0 |
| 应用内自更新 | 一键完成：按当前平台与架构下载新二进制 → **SHA256 校验**（对照 Release 的 checksums）→ 旧程序备份为 `.bak` → **原子替换**（Windows 用"重命名运行中 exe 再落新文件"策略绕过文件锁）→ 提示重启生效；任一步失败自动回滚，**旧版任何时刻都可用** | P0 |
| 升级前保护 | 应用更新动作执行前自动做一次 `VACUUM INTO` 数据库快照备份；重启后 goose 自动迁移到新表结构 | P0 |
| 更新源可配置 | 默认 GitHub Releases（API + 附件下载）；更新源 Base URL 可改为镜像加速地址，适配网络环境 | P1 |
| 启动自动检查 | 可选项，默认关闭；开启后每 24h 至多查询一次，发现新版仅界面轻提示，不打扰使用 | P2 |

约定：

- 分发基于 **GitHub Releases**：推送 `v*` tag 触发 GitHub Actions 流水线，自动构建 Windows/Linux/macOS 二进制 + `checksums.txt` 并发布 Release（流水线随 M5 建立）
- 检查更新只向更新源发起匿名 GET 请求，不上报任何本机数据
- 兜底路径：始终提供「前往 Release 页面手动下载」链接；自更新不可用时手动替换 exe 同样完成升级，数据目录不受影响
- 实现选型：优先采用成熟的 [go-selfupdate](https://github.com/creativeprojects/go-selfupdate)（跨平台探测、下载校验、Windows 文件锁等已处理好）；不满足需求再手写精简版

---

## 3. 技术栈决策

### 3.1 决策摘要

- 前端：**Vue 3** 全家桶；树展示采用**统一知识画布**（不用任何现成树列表组件）
- 后端：**Go**（已确认接受前后端双语言；否决 Node.js 后端方案）
- 部署：**单文件二进制为主**——`go:embed` 把前端产物打进 Go 二进制，一个文件即完整应用；Docker 为可选备选

### 3.2 前端选型（Vue 3 技术栈）

| 层 | 选型 | 理由 |
| --- | --- | --- |
| 框架 | **Vue 3（Composition API + `<script setup>`）+ Vite + TypeScript** | 用户指定 |
| UI 组件库 | **Element Plus**（备选 Naive UI） | 仅用于常规部件：表单、弹窗、下拉、上传、设置面板等；**树/图画布不用 el-tree**（做不到空间连线效果，已否决） |
| 知识画布 | **首选 Vue Flow（`@vue-flow/core`）定制**；备选按参考实现自研轻量画布 | 见 §3.3 专项说明 |
| 自动布局 | dagre / elkjs（层级树形布局） | 「自动排布」按钮的实现基础 |
| Markdown 编辑/渲染 | **md-editor-v3** | 内置 KaTeX 公式（刚需）、图片粘贴、暗色主题 |
| 状态管理 | Pinia | Vue 官方标准 |
| 服务端状态 | @tanstack/vue-query | 请求缓存/失效策略，顺带解决多标签一致性 |
| 路由 | vue-router | 画布/统计/设置等视图切换 |

### 3.3 知识画布实现专项（关键决策）

参考 `tree-link.html` 的交互全集，作为画布验收基准：

1. 节点 = 绝对定位卡片，自由拖拽，坐标入库
2. 连线 = SVG 贝塞尔曲线，节点移动实时跟随
3. 四向锚点 hover 浮现，锚点拉线（虚线临时线跟随鼠标，落到目标节点成线）
4. 点选模式：选中 A 再点 B 即连 A-B
5. 连线命中：可见细线下垫一条透明宽描边路径，细线也易点中
6. 选中态：节点与连线共用统一高亮色
7. 滚轮缩放（0.3~2.5）、空白拖拽平移、Delete 删除选中、Esc 取消选中
8. 自动排布：按层级逐行整理

**实现路线：**

- **首选：Vue Flow 定制**。它本质上是这套交互的工程化封装（视口变换/缩放/命中检测/Handle 锚点/自定义贝塞尔边都是内置能力），我们只需：① 用**自定义边**实现三种线型（层级线/前置箭头/一般关联）；② 用 Handle 实现四向锚点拉线；③ 写一个「点选成线」的组合式函数维护 pending-source 状态；④ 层级边由 `parent_id` 动态合成进 edges 数据。风险最低、细节打磨最省力。
- **备选：自研轻量画布**（Vue 重写参考实现，约数百行）。优点是零依赖、行为与参考 100% 一致；缺点是缩放数学、命中区域、minimap、框选等都要自己补。若 Vue Flow 在定制中出现不可绕开的限制，切换此路线，不影响数据层与 API。
- 无论哪条路线，**交互验收以参考实现为准**，且三种线型视觉规范必须在 v1 达标。

### 3.4 后端选型对比（终版）

前提：Docker 部署为基准；接受双语言；Node.js 不作考虑（用户明确否决）。

| 维度 | **Go（Gin）** ✅ | Python FastAPI | Java/Kotlin Spring | Node/Bun（TS） |
| --- | --- | --- | --- | --- |
| 运行资源（单人场景） | **最优**（内存 ~10-20MB） | 一般 | 重（数百 MB 起） | 中 |
| Docker 镜像（多阶段） | **~15-20MB**（静态编译，可 scratch/alpine） | ~150MB | 300MB+ | ~100-150MB |
| SQLite | modernc.org/sqlite 纯 Go 无 CGO，交叉编译零负担 | aiosqlite 一般 | 一般 | 成熟 |
| SSE 流式转发（LLM 必需） | ✅ 原生 goroutine + channel 很顺 | ✅ | 略繁琐 | ✅ |
| 并发稳健性（备份、导入导出、批量请求） | **goroutine 模型最稳，无回调地狱** | 一般 | 好 | 回风格风格受限（单线程事件循环） |
| 静态托管（替代 nginx） | `go:embed` 编译期内嵌，最优雅 | StaticFiles | 内置 | serveStatic |
| 未来分发弹性 | ✅ 天然单二进制（Windows/Linux/macOS 交叉编译一条命令） | 差 | 臃肿 | Bun compile 可但非必要 |
| 开发效率（本项目约 20 个端点的 CRUD+SSE） | 中（样板可控，见下方缓解） | 高 | 低 | 高 |
| 类型契约 | 与前端 TS 不共享，靠 API 契约约束（规模小，手写 TS 类型可控） | 同左 | 同左 | （唯一优势项，已被否决） |

**决策：后端采用 Go。**

- Web 框架：**Gin**（生态最大、中文资料最多、中间件齐全）；更贴标准库的 chi 作为备选，不做过度设计
- 数据访问：**GORM + modernc.org/sqlite**（纯 Go 驱动免 CGO，Docker 构建与交叉编译都省心）；追求极致可换 sqlc，v1 不必
- 迁移：**goose**（SQL 文件式迁移，向前兼容，升级不改坏旧库）
- 配置：环境变量 + settings 表双层（部署配置 vs 用户配置分离）
- 开发效率缓解措施：API 面小而稳定（§4 草案一次性定型）、DTO 结构集中一个包内维护、用 oapi-codegen 从 OpenAPI 生成服务端桩与 TS 客户端类型（P1，先手写跑通）

### 3.5 部署形态（v1.4 定稿：单文件二进制为主）

Go 的 `go:embed` 让前后端真正合体：Vue 构建产物在**编译期**打入二进制，最终 **一个文件 = 完整应用**（API 服务 + 全部前端页面 + SQLite 引擎，纯 Go 编译无任何外部依赖）。

**构建流水线**

```
pnpm build（Vue → dist/）
        ↓
//go:embed all:dist  →  go build（CGO_ENABLED=0）
        ↓
knowtree.exe / knowtree-linux / knowtree-macos   （约 20-30MB）
```

- modernc.org/sqlite 纯 Go 无 CGO → `CGO_ENABLED=0` 一条命令交叉编译 Windows/Linux/macOS，无任何 C 依赖拖累
- 版本号经 `-ldflags "-X main.version=vx.y.z"` 注入，设置面板可见

**运行形态**

```
knowtree（双击或命令行）
    ├─ 默认监听 127.0.0.1:3000（-addr 参数或环境变量可改）
    ├─ 数据目录默认 ./data（exe 同级；-data 或 KNOWTREE_DATA_DIR 可指定）
    │     └─ knowtree.db（SQLite, WAL）+ 上传文件
    └─ 升级 = 应用内自更新（FR-11）或手动替换 exe；数据目录原样保留，goose 自动迁移
```

- 双击即用：**不需要 Docker、不需要 nginx、不需要装任何运行时**
- 首次启动自动建库建表并引导初始化，浏览器打开 `http://127.0.0.1:3000` 即进入

**可选备选：Docker**

保留一份多阶段构建的 `Dockerfile` + `docker-compose.yml`（数据目录挂载为 `/app/data` 卷），供 NAS / 常开服务器等偏好容器化管理的场景。与二进制是同一套代码的两条产物路径，功能完全一致。

| 形态 | 使用方式 | 适用场景 |
| --- | --- | --- |
| **单二进制（主形态）** | 双击 exe / 运行一条命令 | 本机日常使用 |
| Docker（可选） | `docker compose up -d` | NAS、挂机的服务器 |

### 3.6 数据库专项：SQLite 撑得住吗（v1.3 定稿）

**结论：撑得住，且是本场景的最优解。** 「SQLite 经常卡表」的说法需要拆开看——问题真实存在，但成因是**用法**而非能力上限，且本项目恰好全部规避。

**① 卡表问题的真实成因 vs 本项目情况**

| 常见成因 | 本项目是否命中 | 说明 |
| --- | --- | --- |
| 默认 rollback journal 模式：写时独占整个库文件，读写互斥 | ❌ 规避 | 强制开启 **WAL 模式**后，读不阻塞写、写不阻塞读（一写多读并行），这是 SQLite 现代部署的标配 |
| 多进程同时打开同一库文件争写（如程序运行中还用数据库工具改库） | ❌ 天然规避 | **单用户 + 单进程**架构：只有 Go 后端一个进程持有 `knowtree.db` |
| 未设置 `busy_timeout`，撞锁立即报 `database is locked` | ❌ 规避 | DSN 设置 `busy_timeout=5000`，偶发写冲突自动等待重试而非报错 |
| 长事务持锁过久 | ❌ 规避 | 本项目全是单行/小批量 CRUD，事务毫秒级；LLM 流式讲解全程**不写库** |
| 库文件放网络盘（NFS/SMB）导致锁失效甚至损坏 | ❌ 不涉及 | 数据在本地 volume |

**② 本项目的负载画像**

- 写入频率极低：保存节点/连线/批注/习题结果才写库，人肉操作频率下每秒写入次数趋近于零
- 数据量小：5 万节点 + 连线 + 批注 + 习题的库文件预计 < 100MB，SQLite 的舒适区上限是 GB 级、百万级并发读
- 读多写少 + 全部走索引，单次查询微秒级

**③ 保障方案（实施基线）**

```go
// DSN 统一配置（modernc.org/sqlite）
"file:/app/data/knowtree.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)"
```

1. **WAL 模式**：读写并行的基础
2. **写串行化**：所有写操作经后端单个串行队列（channel + 单 goroutine 或连接池 `SetMaxOpenConns(1)` 写通道）进库，从根上消除写写冲突——单人应用毫无吞吐损失
3. **快照备份**：备份用 `VACUUM INTO` / online backup API 出一致性快照，不裸拷运行中的库文件
4. **synchronous=NORMAL**：WAL 下兼顾安全与性能；断电最多丢最后一笔未 checkpoint 的事务（可接受，重要操作前有手动备份）

**④ 为什么不用别的**

| 备选 | 否决理由 |
| --- | --- |
| PostgreSQL / MySQL | 多一个常驻容器（内存 + 数百 MB）、多一套账号密码备份策略；换来的并发写入能力本项目根本用不上。杀鸡用牛刀 |
| bbolt 等 KV 库 | 无 SQL，树查询/统计/搜索全要手写，得不偿失 |
| DuckDB | 面向分析型（OLAP），高频小事务写入不擅长，方向就不对 |

**⑤ 逃生舱**：数据访问层经 GORM 收敛，SQL 方言差异集中可控；若未来真出现多用户/高并发需求（当前无任何迹象），迁移 PostgreSQL 主要是驱动与部署改动，前端零影响。

---

## 4. 系统架构草案

```
┌────────────────────────── 浏览器 ──────────────────────────┐
│      知识画布(Vue Flow 定制)   详情面板(md-editor-v3)        │
│   层级连线 · 关联连线 · 锚点拉线 · 拖拽 · 缩放 · 自动排布     │
└──────────────▲───────────────────────────▲────────────────┘
               │ 静态资源(go:embed)         │ REST + SSE (/api/*)
┌──────────────┴───────────────────────────┴────────────────┐
│                 单进程后端（Go + Gin）                      │
│  静态托管 · REST API · LLM 代理(SSE 流式转发·出题·批改)      │
│                   GORM + goose 迁移                         │
└───────────────────────────┬───────────────────────────────┘
                            │
              SQLite（modernc.org/sqlite 纯 Go 驱动）
       nodes / edges / resources / exercises / annotations / settings
```

要点：

- **单进程、单端口**：后端既提供 API 也托管前端静态文件
- LLM 调用由后端代理（API Key 不下发浏览器），讲解走 SSE 流式；出题/批改为同步 JSON 接口
- 数据目录默认 `./data`（exe 同级；`-data` 参数或 `KNOWTREE_DATA_DIR` 可指定；Docker 形态挂载为 `/app/data`），内含 `knowtree.db`

### API 草案（REST，前缀 `/api`）

```
GET/POST        /nodes                    列表/新增
PATCH/DELETE    /nodes/:id                编辑/删除
POST            /nodes/:id/move           拖拽换父/排序
GET/POST        /edges                    连线列表/新建（含类型）
PATCH/DELETE    /edges/:id                重连端点/改类型/删除
GET/POST        /nodes/:id/resources      资源列表/新增
DELETE          /resources/:id
GET/POST        /nodes/:id/exercises      习题列表/保存生成的习题
PATCH/DELETE    /exercises/:id            存作答草稿/删除
POST            /exercises/:id/submit     提交作答并批改（客观题规则判分；
                                          填空/简答内部调 LLM，返回判定+得分+点评）
GET/POST        /nodes/:id/annotations    批注列表/新增批注
PATCH/DELETE    /annotations/:id          编辑/删除批注
GET             /search?q=                标题搜索
GET/PUT         /settings                 配置读写
GET             /version                  当前版本与构建信息（版本/时间/commit）
POST            /update/check             检查新版本（查询更新源，返回版本与说明）
POST            /update/apply             下载校验并自更新（备份+原子替换，重启生效）
POST            /llm/test                 测试 LLM 连接
POST            /llm/explain              讲解（SSE 流式）
POST            /llm/generate-subtree     生成子树草稿（返回 JSON，不直接入库）
POST            /llm/generate-exercises   生成习题草稿（返回 JSON，不直接入库）
POST            /import                   导入 JSON
GET             /export                   导出 JSON
GET             /stats                    状态统计
```

---

## 5. 数据模型草案

```sql
-- 知识点（树用 parent_id 表达，无限层级；画布坐标持久化）
CREATE TABLE nodes (
  id          TEXT PRIMARY KEY,
  title       TEXT NOT NULL,
  content_md  TEXT NOT NULL DEFAULT '',       -- Markdown 正文（含 LaTeX）
  status      TEXT NOT NULL DEFAULT 'not_started'
              CHECK (status IN ('not_started','learning','partial','mastered','forgotten')),
  stage       TEXT,                           -- 学段标签（冗余便于筛选）
  sort_order  REAL NOT NULL DEFAULT 0,
  parent_id   TEXT REFERENCES nodes(id) ON DELETE CASCADE,
  pos_x REAL, pos_y REAL,
  status_changed_at INTEGER,
  source_note TEXT,                           -- 来源标注（LLM/导入出处/手动）
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

-- 自由关联连线（不含父子关系；层级连线由 parent_id 渲染派生）
CREATE TABLE edges (
  id         TEXT PRIMARY KEY,
  source_id  TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  target_id  TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  relation   TEXT NOT NULL DEFAULT 'related'
             CHECK (relation IN ('prerequisite','related')),
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
  url     TEXT,
  path    TEXT,
  note    TEXT,
  created_at INTEGER NOT NULL
);

-- 练习题（生成 → 作答 → 批改闭环）
CREATE TABLE exercises (
  id            TEXT PRIMARY KEY,
  node_id       TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  type          TEXT NOT NULL CHECK (type IN ('single_choice','multiple_choice','true_false','fill_blank','short_answer')),
  question_md   TEXT NOT NULL,                -- 题干（Markdown + LaTeX）
  options_json  TEXT,                         -- 选择题选项
  answer_md     TEXT NOT NULL,                -- 标准答案
  analysis_md   TEXT,                         -- 解析
  difficulty    INTEGER DEFAULT 3,            -- 1-5
  answer_draft  TEXT,                         -- 用户作答（草稿，未提交）
  result        TEXT CHECK (result IN ('right','partial','wrong')),  -- 批改判定
  score         INTEGER,                      -- 得分 0-100（主观题）
  feedback_md   TEXT,                         -- LLM 批改点评
  answered_at   INTEGER,                      -- 最近提交批改时间
  created_at    INTEGER NOT NULL
);

-- 知识点批注/学习心得（FR-10）
CREATE TABLE annotations (
  id          TEXT PRIMARY KEY,
  node_id     TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  content_md  TEXT NOT NULL,                  -- 短篇 Markdown（支持 LaTeX）
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL
);

-- 设置（KV）
CREATE TABLE settings ( key TEXT PRIMARY KEY, value_json TEXT NOT NULL );

-- FTS5 虚表：标题+正文全文搜索（P1）
```

---

## 6. 非功能需求

| 类别 | 要求 |
| --- | --- |
| 性能 | ≥ 5000 节点时画布交互流畅（聚焦分支渲染 + 折叠聚合；搜索跳转定位）；常规 API < 50ms |
| 部署 | 主形态：单个可执行文件直接运行（≤ 30MB）；可选 Docker（镜像 ≤ 30MB）；单端口（默认 `3000`，可配）；数据落本地 data 目录或挂载卷 |
| 前端托管 | 后端 `go:embed` 自托管，**不依赖 nginx 等任何第三方 Web 服务器** |
| 监听地址 | 默认 `127.0.0.1`；允许配置为 `0.0.0.0` 局域网访问，此时应启用可选访问口令（§7-S9） |
| 数据安全 | SQLite（WAL 模式）；一键备份用 `VACUUM INTO` 一致性快照；goose 迁移向前兼容，升级不改坏旧库 |
| 浏览器兼容 | 最新版 Chrome/Edge/Firefox；不考虑 IE |
| 平台 | Windows / Linux / macOS 三平台单文件程序（CGO_ENABLED=0 交叉编译）；Docker 镜像 linux/amd64 + arm64 |

---

## 7. 补充：原需求未提及但必要的项

| # | 项 | 说明 | 优先级 |
| --- | --- | --- | --- |
| S1 | **数学公式渲染** | K12~大学内容离不开 LaTeX（md-editor-v3 内置 KaTeX） | P0 |
| S2 | **搜索** | 几千节点没有搜索不可用；标题搜索+定位 P0，全文搜索 P1 | P0/P1 |
| S3 | **撤销/重做** | 画布误删误拖常见，至少删除类操作支持 Ctrl+Z/Ctrl+Y | P1 |
| S4 | **删除保护** | 级联删除二次确认 + 明示影响节点数；P1 回收站（软删除） | P0/P1 |
| S5 | **备份与恢复** | 设置面板内置备份入口 | P0 |
| S6 | **导入导出** | 见 FR-8，数据自主权底线 | P0 |
| S7 | **空状态与引导** | 首次启动提供「一键生成学段骨架」「导入 JSON」「空白开始」三选项 | P0 |
| S8 | **统计概览** | 各学段/学科的状态分布、总进度仪表页 | P1 |
| S9 | **最小访问防护** | 可选访问口令（局域网暴露场景） | P1 |
| S10 | **LLM Key 安全** | Key 只存本地、后端代理调用、界面脱敏 | P0 |
| S11 | **多标签页一致性** | vue-query 失效策略解决，不上 WebSocket | P1 |
| S12 | **快捷键** | Tab/Enter 建兄弟/子节点、Delete 删除、Ctrl+F 搜索、Esc 取消 | P1 |

---

## 8. 决策记录与开放问题

### 8.1 已拍板

| # | 决策 |
| --- | --- |
| D1 | 连线区分「前置依赖 / 一般关联」两类 ✅ |
| D2 | 首次启动可一键生成「学段 × 学科」顶层骨架 ✅ |
| D3 | 前端使用 **Vue 3** ✅ |
| D4 | 树展示为**统一知识画布**（空间卡片 + 贝塞尔连线），交互对齐 `tree-link.html` 参考，**不用 el-tree**（v1.2）✅ |
| D5 | 后端确定 **Go**（Gin + GORM + modernc.org/sqlite + goose + go:embed），接受双语言（v1.2 取代 v1.1 的 Node.js 结论）✅ |
| D6 | 练习题闭环：**生成 → 作答 → 批改**；客观题规则判分，填空/简答 LLM 批改（v1.2）✅ |
| D7 | 新增**知识点批注**功能，多条/时间序/画布角标/LLM 讲解引用（v1.2）✅ |
| D8 | 数据来源增加公网公开资源调研通道 ✅ |
| D9 | 数据库定稿 **SQLite**：WAL 模式 + 写串行化 + `VACUUM INTO` 快照备份，专项分析见 §3.6（v1.3）✅ |
| D10 | 部署定稿：**单文件二进制为主形态**（go:embed 内嵌前端，三平台免安装直跑），Docker 为可选备选（v1.4）✅ |
| D11 | 新增 **FR-11 版本检测与升级**：默认 GitHub Releases 为更新源，自更新走「SHA256 校验 → 备份 → 原子替换 → 可回滚」；发布流水线随 M5 建立（v1.5）✅ |
| D12 | 知识点内容统一 **Markdown 格式**（含 LaTeX 扩展）：存原始文本不落 HTML，渲染在前端；适用正文/批注/习题/讲解输出（v1.6）✅ |

### 8.2 仍开放（不阻塞开工）

| # | 问题 | 建议 |
| --- | --- | --- |
| Q1 | 「已遗忘」要不要触发复习提醒（间隔重复/遗忘曲线）？ | v1 只记录状态时间戳，v2 加复习队列 |
| Q2 | 知识点粒度怎么定？ | 不硬性规定，靠 LLM prompt 约定 + 使用中形成习惯 |
| Q3 | 视频资源嵌入播放还是仅跳转？ | 支持 iframe 的嵌入（B 站等），其余跳转 |
| Q4 | 正文图片上传/粘贴是否纳入 v1？ | 建议 P1（md-editor-v3 已具备，成本低） |

---

## 9. 里程碑建议

| 里程碑 | 内容 | 验收标志 |
| --- | --- | --- |
| M1 骨架 | monorepo 工程（Vue3 前端 + Go 后端）、SQLite + goose 迁移、节点 CRUD API、docker-compose 跑通 | curl 建一棵树并持久化 |
| M2 知识画布 | 画布核心（参考实现全部交互）：卡片拖拽、三种线型连线、锚点拉线、点选成线、缩放平移、自动排布、坐标持久化 | 对照 tree-link.html 逐项验收 |
| M3 详情与批注 | 详情面板、Markdown+KaTeX、资源链接、状态切换筛选、**批注 CRUD + 角标** | 节点内容与心得完整可用 |
| M4 LLM | 设置面板、流式讲解、生成子树预览入库、**出题 + 批改闭环** | 讲解/子树/出题/批改四大能力跑通 |
| M5 收尾与发布 | 导入导出、备份恢复、统计页、撤销重做、公开数据源调研报告 + 首个转换脚本、GitHub Actions 发布流水线（三平台二进制 + checksums）、**版本检测与应用内自更新联调** + 可选 Docker 镜像 | 双击 exe 即用；应用内能发现新版并完成自更新 |

---

*本文档为 v1.2。关键决策已锁定（§8.1），剩余开放问题（§8.2）不阻塞开发。*
