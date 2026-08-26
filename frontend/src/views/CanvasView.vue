<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  VueFlow,
  useVueFlow,
  MarkerType,
  ConnectionMode,
  type Connection,
  type NodeMouseEvent,
  type EdgeMouseEvent,
  type GraphNode,
} from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { MiniMap } from '@vue-flow/minimap'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Setting } from '@element-plus/icons-vue'

import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/minimap/dist/style.css'

import { useTreeStore, pushUndo } from '@/stores/tree'
import KnowledgeNode from '@/components/canvas/KnowledgeNode.vue'
import DetailDrawer from '@/components/canvas/DetailDrawer.vue'
import GradeBar from '@/components/canvas/GradeBar.vue'
import type { GradeSegment } from '@/components/canvas/GradeBar.vue'
import NavControls from '@/components/canvas/NavControls.vue'
import SettingsPanel from '@/components/panels/SettingsPanel.vue'
import StatsPanel from '@/components/panels/StatsPanel.vue'
import { GRADES, UNSET_GRADE, LINE_STYLE, RELATION_LABEL, matchGrade } from '@/utils/meta'
import { api } from '@/api/client'
import type { KNode, EdgeRelation } from '@/types'

const store = useTreeStore()
const { fitView, addNodes, zoomIn, zoomOut, setViewport, viewport } = useVueFlow()

onMounted(async () => {
  await store.loadAll()
  // 首次出现无坐标的节点时，自动排布一次并持久化
  if (store.nodes.some((n) => n.pos_x == null || n.pos_y == null)) {
    await nextTick()
    void autoLayout(true)
  } else {
    void nextTick(() => fitView({ padding: 0.15 }))
  }
})

// ---------- 数据 → Vue Flow ----------
const nodeWidth = 190
const nodeHeight = 64

interface KtNodeData {
  node: KNode
  selected: boolean
  pending: boolean
  stageColor?: string
  [key: string]: unknown
}

const selectedNodeId = ref<string | null>(null)
const pendingSourceId = ref<string | null>(null)
const selectedEdgeId = ref<string | null>(null)

const flowNodes = computed(() =>
  store.nodes.map(
    (n): GraphNode =>
      ({
        id: n.id,
        type: 'kt',
        position: { x: n.pos_x ?? 0, y: n.pos_y ?? 0 },
        data: {
          node: n,
          selected: n.id === selectedNodeId.value || n.id === pendingSourceId.value,
          pending: n.id === pendingSourceId.value,
          stageColor: matchGrade(n.stage)?.color ?? '#cbd5e1',
        } satisfies KtNodeData,
        draggable: true,
      }) as unknown as GraphNode,
  ),
)

const flowEdges = computed(() => {
  const hier = store.nodes
    .filter((n) => n.parent_id)
    .map((n) => ({
      id: `h-${n.id}`,
      source: n.parent_id!,
      target: n.id,
      style: { ...LINE_STYLE.hierarchy },
      selectable: false,
    }))
  const rel = store.edges.map((e) => {
    if (e.relation === 'prerequisite') {
      return {
        id: e.id,
        source: e.source_id,
        target: e.target_id,
        label: RELATION_LABEL.prerequisite,
        labelStyle: { fill: '#e8590c', fontSize: 11 },
        labelBgStyle: { fill: '#fff' },
        markerEnd: MarkerType.ArrowClosed,
        style: { ...LINE_STYLE.prerequisite },
      }
    }
    return {
      id: e.id,
      source: e.source_id,
      target: e.target_id,
      style: { ...LINE_STYLE.related },
    }
  })
  return [...hier, ...rel]
})

// ---------- 连线类型 ----------
const relationType = ref<EdgeRelation>('related')
// 连线模式开关：开启后点击节点用于连线，关闭时点击=打开详情
const connectMode = ref(false)

watch(connectMode, (on) => {
  ElMessage.info(on ? '连线模式：依次点击两个节点建立连线（Esc 退出）' : '已退出连线模式')
})

async function createEdgeBetween(sourceId: string, targetId: string) {
  try {
    const e = await store.createEdge(sourceId, targetId, relationType.value)
    pushUndo({
      label: '连线',
      undo: async () => {
        await store.deleteEdgeRaw(e.id)
      },
      redo: async () => {
        await store.createEdgeRaw({ id: e.id, source_id: e.source_id, target_id: e.target_id, relation: e.relation })
      },
    })
    ElMessage.success(`已连接（${RELATION_LABEL[relationType.value]}）`)
  } catch (e) {
    ElMessage.warning(e instanceof Error ? e.message : String(e))
  }
}

function onConnect(conn: Connection) {
  if (conn.source && conn.target && conn.source !== conn.target) {
    void createEdgeBetween(conn.source, conn.target)
  }
}

// 点选连线：仅在「连线模式」开启时生效；普通点击 = 选中并打开详情
function onNodeClick({ node }: NodeMouseEvent) {
  const id = node.id as string
  if (connectMode.value) {
    if (pendingSourceId.value && pendingSourceId.value !== id) {
      const src = pendingSourceId.value
      clearPending()
      void createEdgeBetween(src, id)
      return
    }
    if (pendingSourceId.value === id) {
      clearPending()
      return
    }
    selectedEdgeId.value = null
    selectedNodeId.value = id
    return
  }
  selectedEdgeId.value = null
  selectedNodeId.value = id
}

function onPaneClick() {
  clearPending()
  selectedNodeId.value = null
  selectedEdgeId.value = null
}

function onEdgeClick({ edge }: EdgeMouseEvent) {
  clearPending()
  selectedNodeId.value = null
  // 层级线不可选中删除
  selectedEdgeId.value = edge.id.startsWith('h-') ? null : edge.id
}

function clearPending() {
  pendingSourceId.value = null
}

// ---------- 拖拽落点持久化（含位置撤销）----------
const dragStartPos = ref<Map<string, { x: number; y: number }>>(new Map())

function onDragStart(e: { node: GraphNode }) {
  dragStartPos.value.set(e.node.id, { x: e.node.position.x, y: e.node.position.y })
}

function onDragStop(e: { node: GraphNode }) {
  const n = store.byId.get(e.node.id)
  if (!n) return
  const x = Math.round(e.node.position.x)
  const y = Math.round(e.node.position.y)
  if (n.pos_x !== x || n.pos_y !== y) {
    const old = dragStartPos.value.get(e.node.id)
    const id = n.id
    void store.savePosition(n.id, x, y)
    if (old) {
      pushUndo({
        label: '移动节点',
        undo: async () => {
          await store.savePosition(id, Math.round(old.x), Math.round(old.y))
        },
        redo: async () => {
          await store.savePosition(id, x, y)
        },
      })
    }
  }
}

// ---------- 自动排布（紧凑分层树）----------
function computeLayout(): Map<string, { x: number; y: number }> {
  const childrenOf = new Map<string, string[]>()
  for (const n of store.nodes) {
    const key = n.parent_id ?? '__root__'
    if (!childrenOf.has(key)) childrenOf.set(key, [])
    childrenOf.get(key)!.push(n.id)
  }
  const pos = new Map<string, { x: number; y: number }>()
  const H_GAP = 46
  const V_GAP = 120
  const TOP = 60
  let cursorX = 60

  const place = (id: string, depth: number): number => {
    const kids = childrenOf.get(id) ?? []
    if (!kids.length) {
      pos.set(id, { x: cursorX, y: TOP + depth * V_GAP })
      cursorX += nodeWidth + H_GAP
      return nodeWidth
    }
    for (const k of kids) place(k, depth + 1)
    const first = pos.get(kids[0])!
    const last = pos.get(kids[kids.length - 1])!
    pos.set(id, { x: Math.round((first.x + last.x + nodeWidth - nodeWidth) / 2), y: TOP + depth * V_GAP })
    return last.x + nodeWidth - first.x
  }

  for (const r of childrenOf.get('__root__') ?? []) place(r, 0)

  // 无父链可达的孤立节点（异常数据兜底）：排在最右侧一行
  for (const n of store.nodes) {
    if (!pos.has(n.id)) {
      pos.set(n.id, { x: cursorX, y: TOP })
      cursorX += nodeWidth + H_GAP
    }
  }
  return pos
}

const layouting = ref(false)
async function autoLayout(silent = false) {
  layouting.value = true
  try {
    // 记录旧坐标用于撤销
    const oldPositions = new Map(store.nodes.map((n) => [n.id, { x: n.pos_x ?? 0, y: n.pos_y ?? 0 }]))
    const pos = computeLayout()
    const newItems = [...pos.entries()].map(([id, p]) => ({ id, pos_x: p.x, pos_y: p.y }))
    await store.setPositions(newItems)
    pushUndo({
      label: '自动排布',
      undo: async () => {
        await store.setPositions([...oldPositions.entries()].map(([id, p]) => ({ id, pos_x: p.x, pos_y: p.y })))
      },
      redo: async () => {
        await store.setPositions(newItems)
      },
    })
    if (!silent) ElMessage.success('已自动排布')
    await nextTick(() => fitView({ padding: 0.12, duration: 250 }))
  } catch (e) {
    if (!silent) ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    layouting.value = false
  }
}

// ---------- 视口跟踪 → 分级彩条 ----------
const flowWrapRef = ref<HTMLElement | null>(null)
const wrapSize = ref({ w: 1, h: 1 })
let resizeObs: ResizeObserver | undefined

onMounted(() => {
  const el = flowWrapRef.value
  if (!el) return
  const measure = () => {
    wrapSize.value = { w: el.clientWidth || 1, h: el.clientHeight || 1 }
  }
  measure()
  resizeObs = new ResizeObserver(measure)
  resizeObs.observe(el)
})

onBeforeUnmount(() => resizeObs?.disconnect())

// 平移/缩放事件兜底触发，保证彩条实时刷新
const vpTick = ref(0)
function onFlowMove() {
  vpTick.value++
}

function readViewport(): { x: number; y: number; zoom: number } {
  const raw: unknown = viewport
  const v = (
    raw && typeof raw === 'object' && 'value' in (raw as Record<string, unknown>)
      ? (raw as { value: unknown }).value
      : raw
  ) as { x?: number; y?: number; zoom?: number } | undefined
  const zoom = typeof v?.zoom === 'number' && v.zoom > 0 ? v.zoom : 1
  return { x: v?.x ?? 0, y: v?.y ?? 0, zoom }
}

const zoomPercent = computed(() => {
  void vpTick.value
  return Math.round(readViewport().zoom * 100)
})

// 当前视口覆盖的世界坐标范围（外扩 40px 让彩条提前一点亮起）
const visibleRect = computed(() => {
  void vpTick.value
  const { w, h } = wrapSize.value
  const v = readViewport()
  return {
    x0: -v.x / v.zoom - 40,
    y0: -v.y / v.zoom - 40,
    x1: (w - v.x) / v.zoom + 40,
    y1: (h - v.y) / v.zoom + 40,
  }
})

interface StageBucket {
  key: string
  label: string
  color: string
  total: number
  inView: number
}

const gradeSegments = computed<GradeSegment[]>(() => {
  const r = visibleRect.value
  const buckets = new Map<string, StageBucket>()
  for (const g of GRADES) buckets.set(g.key, { key: g.key, label: g.label, color: g.color, total: 0, inView: 0 })
  buckets.set(UNSET_GRADE.key, {
    key: UNSET_GRADE.key,
    label: UNSET_GRADE.label,
    color: UNSET_GRADE.color,
    total: 0,
    inView: 0,
  })
  for (const n of store.nodes) {
    const b = buckets.get(matchGrade(n.stage)?.key ?? UNSET_GRADE.key)
    if (!b) continue
    b.total++
    const cx = (n.pos_x ?? 0) + nodeWidth / 2
    const cy = (n.pos_y ?? 0) + nodeHeight / 2
    if (cx >= r.x0 && cx <= r.x1 && cy >= r.y0 && cy <= r.y1) b.inView++
  }
  const segs = [...buckets.values()].map(
    (b): GradeSegment => ({ key: b.key, label: b.label, color: b.color, count: b.total, inView: b.inView }),
  )
  // 「未设置」段只有在确实存在未匹配节点时才显示
  return segs.filter((s) => s.key !== UNSET_GRADE.key || s.count > 0)
})

/** 点击彩条：聚焦到该学段的全部节点 */
function focusGrade(key: string) {
  const ids = store.nodes
    .filter((n) => (matchGrade(n.stage)?.key ?? UNSET_GRADE.key) === key)
    .map((n) => n.id)
  if (!ids.length) {
    ElMessage.info('该学段还没有知识点')
    return
  }
  void fitView({ nodes: ids, duration: 350, padding: 0.25 })
}

// ---------- 左下角地图式导航 ----------
async function panBy(dx: number, dy: number) {
  const v = readViewport()
  await setViewport({ x: v.x + dx, y: v.y + dy, zoom: v.zoom }, { duration: 120 })
}

function zoomInStep() {
  void zoomIn({ duration: 200 })
}

function zoomOutStep() {
  void zoomOut({ duration: 200 })
}

function fitAll() {
  void fitView({ padding: 0.12, duration: 250 })
}

// ---------- 键盘 ----------
function onKeydown(ev: KeyboardEvent) {
  const tag = (ev.target as HTMLElement)?.tagName?.toLowerCase()
  if (tag === 'input' || tag === 'textarea') return
  if (ev.key === 'Escape') {
    clearPending()
    selectedNodeId.value = null
    selectedEdgeId.value = null
    drawerOpen.value = false
  } else if ((ev.ctrlKey || ev.metaKey) && ev.key.toLowerCase() === 'z' && !ev.shiftKey) {
    if (store.canUndo()) {
      void store.undo()
      ElMessage.info('已撤销')
    }
  } else if ((ev.ctrlKey || ev.metaKey) && (ev.key.toLowerCase() === 'y' || (ev.shiftKey && ev.key.toLowerCase() === 'z'))) {
    if (store.canRedo()) {
      void store.redo()
      ElMessage.info('已重做')
    }
  } else if (ev.key === 'Delete' || ev.key === 'Backspace') {
    if (selectedEdgeId.value) {
      void removeEdge(selectedEdgeId.value)
    } else if (selectedNodeId.value) {
      void removeNode(selectedNodeId.value)
    }
  }
}
window.addEventListener('keydown', onKeydown)
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

async function removeEdge(id: string) {
  const e = store.edges.find((x) => x.id === id)
  try {
    await store.deleteEdgeRaw(id)
    selectedEdgeId.value = null
    if (e) {
      pushUndo({
        label: '删除连线',
        undo: async () => {
          await store.createEdgeRaw({ id: e.id, source_id: e.source_id, target_id: e.target_id, relation: e.relation })
        },
        redo: async () => {
          await store.deleteEdgeRaw(e.id)
        },
      })
    }
    ElMessage.success('已删除连线（Ctrl+Z 可撤销）')
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : String(err))
  }
}

// ---------- 节点操作 ----------
const drawerOpen = ref(false)
const drawerRef = ref<InstanceType<typeof DetailDrawer> | null>(null)

watch(selectedNodeId, (id) => {
  if (id && !connectMode.value) drawerOpen.value = true
})

// 关联知识点跳转：切换详情并居中该节点
function jumpTo(id: string) {
  selectedNodeId.value = id
  void nextTick(() => fitView({ nodes: [id], duration: 300, padding: 0.4 }))
}

// 抽屉关闭时冲刷未保存的正文
function onDrawerClosed() {
  drawerRef.value?.flushSave()
}

// ---------- 新建节点（底部功能坞）：有选中节点时挂为其子节点，否则建为根节点并放到视口中央 ----------
const selectedNode = computed<KNode | undefined>(() =>
  selectedNodeId.value ? store.byId.get(selectedNodeId.value) : undefined,
)

async function addNodeManual() {
  let title = ''
  try {
    const r = await ElMessageBox.prompt(
      selectedNode.value ? `作为「${selectedNode.value.title}」的子节点创建` : '将创建为顶层节点',
      '新建知识点',
      {
        confirmButtonText: '创建',
        cancelButtonText: '取消',
        inputPattern: /\S/,
        inputErrorMessage: '标题不能为空',
      },
    )
    title = r.value.trim()
  } catch {
    return
  }
  if (!title) return
  try {
    const parent = selectedNode.value
    const n = await store.createNode(title, parent?.id ?? null)
    // 有父节点放其右下；否则放视口中心
    let x: number
    let y: number
    if (parent) {
      x = (parent.pos_x ?? 0) + nodeWidth + 40
      y = (parent.pos_y ?? 0) + nodeHeight + 50
    } else {
      const v = readViewport()
      x = Math.round((wrapSize.value.w / 2 - v.x) / v.zoom - nodeWidth / 2)
      y = Math.round((wrapSize.value.h / 2 - v.y) / v.zoom - nodeHeight / 2)
    }
    await store.savePosition(n.id, x, y)
    if (!parent) {
      addNodes([
        {
          id: n.id,
          type: 'kt',
          position: { x, y },
          data: { node: { ...n, pos_x: x, pos_y: y }, selected: false, pending: false },
        },
      ])
    }
    selectedNodeId.value = n.id
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  }
}

// ---------- 撤销 / 重做（功能坞按钮） ----------
async function doUndo() {
  if (!store.canUndo()) {
    ElMessage.info('没有可撤销的操作')
    return
  }
  await store.undo()
  ElMessage.info('已撤销')
}

async function doRedo() {
  if (!store.canRedo()) {
    ElMessage.info('没有可重做的操作')
    return
  }
  await store.redo()
  ElMessage.info('已重做')
}

async function removeNode(id: string) {
  const n = store.byId.get(id)
  if (!n) return
  let count = 1
  const collect = (pid: string): number => {
    let c = 0
    for (const x of store.nodes.filter((k) => k.parent_id === pid)) c += 1 + collect(x.id)
    return c
  }
  count += collect(id)
  try {
    await ElMessageBox.confirm(`将删除「${n.title}」及其子树，共 ${count} 个节点。确定？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }

  // 快照子树节点 + 相关联线，供撤销重建（批注/资源不随撤销恢复）
  const doomed = new Set<string>([id])
  const collectIds = (pid: string) => {
    for (const x of store.nodes.filter((k) => k.parent_id === pid)) {
      doomed.add(x.id)
      collectIds(x.id)
    }
  }
  collectIds(id)
  const snapNodes = store.nodes.filter((x) => doomed.has(x.id))
  const snapEdges = store.edges.filter((e) => doomed.has(e.source_id) || doomed.has(e.target_id))

  const deleted = await store.deleteNodeCascadeRaw(id)
  selectedNodeId.value = null
  pushUndo({
    label: '删除节点',
    undo: async () => {
      for (const sn of snapNodes) {
        await store.createNodeRaw({
          id: sn.id, title: sn.title, parent_id: sn.parent_id, stage: sn.stage,
          status: sn.status, content_md: sn.content_md,
        })
        if (sn.pos_x != null && sn.pos_y != null) {
          await store.savePosition(sn.id, Math.round(sn.pos_x), Math.round(sn.pos_y))
        }
        if (!store.nodes.some((k) => k.id === sn.id)) store.nodes.push(sn)
      }
      for (const se of snapEdges) {
        try {
          await store.createEdgeRaw(se)
        } catch {
          /* 已存在则跳过 */
        }
      }
    },
    redo: async () => {
      await store.deleteNodeCascadeRaw(id)
    },
  })
  ElMessage.success(`已删除 ${deleted} 个节点（Ctrl+Z 可撤销）`)
}

// ---------- AI 生成子树 ----------
interface DraftTreeNode {
  title: string
  summary?: string
  children?: DraftTreeNode[]
}
const subOpen = ref(false)
const subTopic = ref('')
const subParent = ref<string | null>(null)
const subCount = ref(8)
const subGenerating = ref(false)
const subTree = ref<DraftTreeNode[]>([])
const inserting = ref(false)

function openSubDialog() {
  subTopic.value = ''
  subTree.value = []
  subParent.value = selectedNodeId.value
  subOpen.value = true
}

async function generateSubtree() {
  if (!subTopic.value.trim()) return
  subGenerating.value = true
  subTree.value = []
  try {
    const r = await api.post<{ tree: DraftTreeNode[] }>('/api/llm/generate-subtree', {
      parent_id: subParent.value,
      topic: subTopic.value.trim(),
      count: subCount.value,
    })
    subTree.value = r.tree ?? []
    if (!subTree.value.length) ElMessage.warning('模型没有生成内容，请重试')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    subGenerating.value = false
  }
}

function countDraft(list: DraftTreeNode[]): number {
  return list.reduce((s, d) => s + 1 + countDraft(d.children ?? []), 0)
}

async function insertDraftTree(list: DraftTreeNode[], parentId: string | null): Promise<number> {
  let count = 0
  for (const d of list) {
    const n = await store.createNode(d.title, parentId)
    count++
    if (d.children?.length) count += await insertDraftTree(d.children, n.id)
  }
  return count
}

async function confirmInsertSubtree() {
  inserting.value = true
  try {
    const total = await insertDraftTree(subTree.value, subParent.value)
    subOpen.value = false
    await autoLayout(true)
    ElMessage.success(`已插入 ${total} 个知识点`)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    inserting.value = false
  }
}

const subTreeOptions = computed(() => toElTree(subTree.value))

// 挂载点选择树
interface TreeNode {
  node: KNode
  children: TreeNode[]
}
const parentOptions = computed(() => toOptions(store.tree))
function toOptions(list: TreeNode[]): { value: string; label: string; children: unknown[] }[] {
  return list.map((t) => ({
    value: t.node.id,
    label: t.node.title,
    children: toOptions(t.children),
  }))
}

function toElTree(list: DraftTreeNode[]): { label: string; children: unknown[] }[] {
  return list.map((d) => ({
    label: d.summary ? `${d.title}（${d.summary}）` : d.title,
    children: toElTree(d.children ?? []),
  }))
}

// ---------- 面板开关 ----------
const settingsOpen = ref(false)
const statsOpen = ref(false)
</script>

<template>
  <div class="canvas-page">
    <div ref="flowWrapRef" class="flow-wrap">
      <VueFlow
        :nodes="flowNodes"
        :edges="flowEdges"
        :connection-mode="ConnectionMode.Loose"
        :default-edge-options="{ type: 'default' }"
        :min-zoom="0.25"
        :max-zoom="2.5"
        fit-view-on-init
        @node-click="onNodeClick"
        @edge-click="onEdgeClick"
        @pane-click="onPaneClick"
        @connect="onConnect"
        @node-drag-start="onDragStart"
        @node-drag-stop="onDragStop"
        @move="onFlowMove"
      >
        <template #node-kt="ktProps">
          <KnowledgeNode :data="ktProps.data" />
        </template>
        <Background :gap="22" pattern-color="#c9d3e3" />
        <MiniMap pannable zoomable />
      </VueFlow>

      <!-- 顶部分级彩条：视口内的学段展开，其余收缩成色点 -->
      <GradeBar :segments="gradeSegments" @select="focusGrade" />

      <!-- 右上角设置 -->
      <button class="gear-btn" title="设置" @click="settingsOpen = true">
        <el-icon><Setting /></el-icon>
      </button>

      <!-- 左下角地图式缩放 / 方位面板 -->
      <NavControls
        :zoom-percent="zoomPercent"
        @pan="panBy"
        @zoom-in="zoomInStep"
        @zoom-out="zoomOutStep"
        @fit="fitAll"
      />

      <!-- 底部功能坞 -->
      <div class="dock">
        <button class="dock-btn" title="新建知识点（有选中节点时挂为其子级）" @click="addNodeManual">
          <span class="ico">➕</span><small>新建</small>
        </button>
        <button class="dock-btn" title="自动整理整棵知识树的布局" :disabled="layouting" @click="autoLayout()">
          <span class="ico">✨</span><small>{{ layouting ? '排布中…' : '排布' }}</small>
        </button>
        <button class="dock-btn" title="AI 批量生成知识点子树" @click="openSubDialog">
          <span class="ico">🤖</span><small>AI 子树</small>
        </button>

        <span class="dock-sep" />

        <div class="dock-seg" title="新连线的类型">
          <button :class="{ active: relationType === 'related' }" @click="relationType = 'related'">关联</button>
          <button :class="{ active: relationType === 'prerequisite' }" @click="relationType = 'prerequisite'">前置</button>
        </div>
        <label class="mode-switch" title="开启后点击两个节点即建立连线；关闭时点击节点打开详情">
          <span>连线模式</span>
          <el-switch v-model="connectMode" size="small" />
        </label>

        <span class="dock-sep" />

        <button class="dock-btn" title="撤销（Ctrl+Z）" @click="doUndo">
          <span class="ico">↩️</span><small>撤销</small>
        </button>
        <button class="dock-btn" title="重做（Ctrl+Y）" @click="doRedo">
          <span class="ico">↪️</span><small>重做</small>
        </button>
        <button class="dock-btn" title="学习统计" @click="statsOpen = true">
          <span class="ico">📊</span><small>统计</small>
        </button>

        <span class="dock-sep" />

        <el-popover placement="top-end" :width="220" trigger="hover">
          <div class="legend">
            <div class="legend__title">图例</div>
            <div class="legend__row"><i class="ln hierarchy" />层级关系</div>
            <div class="legend__row"><i class="ln prerequisite" />前置依赖</div>
            <div class="legend__row"><i class="ln related" />一般关联</div>
            <div class="legend__tip">顶部彩条按学段着色 · 点击可聚焦该学段</div>
          </div>
          <template #reference>
            <button class="dock-btn" title="图例与帮助">
              <span class="ico">❔</span><small>图例</small>
            </button>
          </template>
        </el-popover>
      </div>

      <!-- 详情抽屉（FR-3）：正文 Markdown+KaTeX / 资源 / 批注 / 关联 -->
      <el-drawer
        v-model="drawerOpen"
        :with-header="false"
        size="560px"
        :append-to-body="false"
        @closed="onDrawerClosed"
      >
        <DetailDrawer
          ref="drawerRef"
          :node-id="selectedNodeId"
          @close="drawerOpen = false"
          @jump="jumpTo"
        />
      </el-drawer>

      <!-- AI 生成子树对话框 -->
      <el-dialog v-model="subOpen" title="🤖 AI 生成知识点子树" width="560px" append-to-body>
        <div v-if="!subTree.length" style="display: flex; flex-direction: column; gap: 12px">
          <el-input
            v-model="subTopic"
            placeholder="主题，如：人教版八年级物理·浮力"
            @keyup.enter="generateSubtree"
          />
          <el-tree-select
            v-model="subParent"
            :data="parentOptions"
            :props="{ label: 'label', value: 'value' }"
            check-strictly
            clearable
            filterable
            placeholder="挂载到（留空=根）"
          />
          <div style="display: flex; align-items: center; gap: 10px">
            <span class="dim">节点数</span>
            <el-input-number v-model="subCount" :min="3" :max="30" />
            <el-button type="primary" :loading="subGenerating" @click="generateSubtree">生成预览</el-button>
          </div>
        </div>
        <div v-else>
          <el-alert type="info" :closable="false" style="margin-bottom: 12px"
            :title="`共 ${countDraft(subTree)} 个节点，确认后入库并自动排布`" />
          <el-tree :data="subTreeOptions" default-expand-all :props="{ label: 'label', children: 'children' }" style="max-height: 46vh; overflow: auto" />
        </div>
        <template #footer>
          <span style="display: flex; justify-content: space-between; width: 100%">
            <span>
              <el-button v-if="subTree.length" @click="subTree = []">重新生成</el-button>
            </span>
            <span>
              <el-button @click="subOpen = false">取消</el-button>
              <el-button
                v-if="subTree.length"
                type="primary"
                :loading="inserting"
                @click="confirmInsertSubtree"
              >入库</el-button>
            </span>
          </span>
        </template>
      </el-dialog>

      <!-- 设置 / 统计面板 -->
      <SettingsPanel v-model="settingsOpen" />
      <StatsPanel v-model="statsOpen" />
    </div>
  </div>
</template>

<style scoped>
.canvas-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.flow-wrap {
  flex: 1;
  min-height: 0;
  position: relative;
}

.flow-wrap :deep(.vue-flow__pane) {
  cursor: grab;
}

/* 小地图贴右下角 */
.flow-wrap :deep(.vue-flow__minimap) {
  bottom: 16px;
  right: 14px;
  margin: 0;
}

/* ---------- 右上角设置按钮 ---------- */
.gear-btn {
  position: absolute;
  top: 12px;
  right: 14px;
  z-index: 21;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  border: 1px solid #e0e6ef;
  background: rgba(255, 255, 255, 0.95);
  color: #3c4a63;
  font-size: 17px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 3px 10px rgba(30, 50, 90, 0.18);
  transition:
    transform 0.25s,
    color 0.15s,
    box-shadow 0.15s;
}

.gear-btn:hover {
  transform: rotate(40deg);
  color: #2563eb;
  box-shadow: 0 4px 14px rgba(30, 50, 90, 0.26);
}

/* ---------- 底部功能坞 ---------- */
.dock {
  position: absolute;
  left: 50%;
  bottom: 14px;
  transform: translateX(-50%);
  z-index: 20;
  display: flex;
  align-items: center;
  max-width: calc(100vw - 200px);
  overflow-x: auto;
  scrollbar-width: none;
  background: rgba(23, 30, 44, 0.9);
  backdrop-filter: blur(8px);
  border-radius: 16px;
  padding: 5px 8px;
  box-shadow: 0 6px 24px rgba(15, 25, 45, 0.35);
}

.dock::-webkit-scrollbar {
  display: none;
}

.dock-btn {
  appearance: none;
  border: none;
  background: transparent;
  color: #cdd7e8;
  border-radius: 10px;
  width: 56px;
  height: 50px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  cursor: pointer;
  transition:
    background 0.12s,
    color 0.12s,
    opacity 0.12s;
}

.dock-btn .ico {
  font-size: 16px;
  line-height: 1;
}

.dock-btn small {
  font-size: 10px;
  line-height: 1;
  color: #93a1b8;
}

.dock-btn:hover {
  background: rgba(255, 255, 255, 0.09);
  color: #fff;
}

.dock-btn:hover small {
  color: #cdd7e8;
}

.dock-btn:disabled {
  opacity: 0.55;
  cursor: default;
}

.dock-sep {
  width: 1px;
  height: 26px;
  background: rgba(255, 255, 255, 0.14);
  margin: 0 6px;
  flex-shrink: 0;
}

.dock-seg {
  display: flex;
  background: rgba(255, 255, 255, 0.07);
  border-radius: 9px;
  padding: 2px;
  margin-right: 4px;
}

.dock-seg button {
  appearance: none;
  border: none;
  background: transparent;
  color: #aab6cc;
  font-size: 11.5px;
  padding: 6px 10px;
  border-radius: 7px;
  cursor: pointer;
  transition:
    background 0.12s,
    color 0.12s;
}

.dock-seg button.active {
  background: #5b8def;
  color: #fff;
  font-weight: 600;
}

.mode-switch {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: 12px;
  color: #cdd7e8;
  padding: 0 8px;
  cursor: pointer;
  white-space: nowrap;
}

/* ---------- 图例弹层 ---------- */
.legend__title {
  font-weight: 700;
  font-size: 13px;
  color: #26334d;
  margin-bottom: 6px;
}

.legend__row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #51607a;
  padding: 3px 0;
}

.legend .ln {
  display: inline-block;
  width: 24px;
  height: 0;
  border-top: 2px solid;
}

.ln.hierarchy {
  border-color: #9aa7bf;
  border-top-width: 1.5px;
}

.ln.prerequisite {
  border-color: #e8590c;
  border-top-style: dashed;
}

.ln.related {
  border-color: #5b8def;
}

.legend__tip {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed #e4e9f2;
  font-size: 11px;
  color: #98a2b3;
}

.dim {
  color: #98a2b3;
  font-size: 12px;
}
</style>
