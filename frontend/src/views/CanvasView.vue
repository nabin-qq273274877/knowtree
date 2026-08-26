<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  VueFlow,
  useVueFlow,
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
import FloatingEdge from '@/components/canvas/FloatingEdge.vue'
import SettingsPanel from '@/components/panels/SettingsPanel.vue'
import StatsPanel from '@/components/panels/StatsPanel.vue'
import {
  GRADES,
  UNSET_GRADE,
  LINE_STYLE,
  matchGrade,
  gradeColumnIndex,
  gradeColumnRange,
  type GradeDef,
} from '@/utils/meta'
import { api } from '@/api/client'
import type { KNode } from '@/types'

const store = useTreeStore()
const { fitView, addNodes, zoomIn, zoomOut, setViewport, viewport } = useVueFlow()

onMounted(async () => {
  await store.loadAll()
  // 兼容旧数据：存在无坐标、或落在自己学段分区之外的节点时，
  // 先按学段分区静默重排一次（否则拖拽约束会立刻把节点拽回去）
  const outOfZone = store.nodes.some((n) => {
    const col = gradeColumnRange(gradeColumnIndex(matchGrade(n.stage)?.key))
    const x = n.pos_x ?? 0
    return x < col.x0 - ZONE_TOLERANCE + COL_PAD || x + nodeWidth > col.x1 + ZONE_TOLERANCE - COL_PAD
  })
  if (store.nodes.some((n) => n.pos_x == null || n.pos_y == null) || outOfZone) {
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
  stageColor?: string
  [key: string]: unknown
}

const selectedNodeId = ref<string | null>(null)
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
          selected: n.id === selectedNodeId.value,
          stageColor: matchGrade(n.stage)?.color ?? '#cbd5e1',
        } satisfies KtNodeData,
        draggable: true,
      }) as unknown as GraphNode,
  ),
)

const flowEdges = computed(() => {
  // 统一使用浮动智能边：渲染时自动取两节点最近的边框锚点，贴边无空隙
  const hier = store.nodes
    .filter((n) => n.parent_id)
    .map((n) => ({
      id: `h-${n.id}`,
      source: n.parent_id!,
      target: n.id,
      type: 'floating',
      style: { ...LINE_STYLE.hierarchy },
      selectable: false,
    }))
  const rel = store.edges.map((e) => {
    const anch = edgeAnchors.value.get(e.id)
    const data = anch ? { sh: anch.s, th: anch.t } : undefined
    // 手工连线与层级线保持同一样式（连线只表达学习先后，一种样式即可）
    return {
      id: e.id,
      source: e.source_id,
      target: e.target_id,
      type: 'floating',
      style: { ...LINE_STYLE.hierarchy },
      data,
    }
  })
  return [...hier, ...rel]
})

// ---------- 连线规则 ----------
// 连线只表达「学习先后」：只能从上层知识点连向下层知识点，
// 同层（含同级）不允许连线；上层可以同时连多个下层（多上连多下）。
function depthOf(id: string): number {
  let d = 0
  let cur = store.byId.get(id)
  while (cur?.parent_id && d < 64) {
    d++
    cur = store.byId.get(cur.parent_id)
  }
  return d
}

// 点击节点 = 选中并打开详情（连线直接从节点锚点拖出，无需专门模式）
function onNodeClick({ node }: NodeMouseEvent) {
  selectedEdgeId.value = null
  // 始终显式打开详情：即使重复点击同一节点 / 抽屉曾被关闭也能再次打开
  selectedNodeId.value = node.id as string
  drawerOpen.value = true
}

function onPaneClick() {
  selectedNodeId.value = null
  selectedEdgeId.value = null
}

function onEdgeClick({ edge }: EdgeMouseEvent) {
  selectedNodeId.value = null
  // 层级线不可选中删除
  selectedEdgeId.value = edge.id.startsWith('h-') ? null : edge.id
}

// 双击连线直接删除（层级线除外）
function onEdgeDblClick({ edge }: EdgeMouseEvent) {
  selectedNodeId.value = null
  if (edge.id.startsWith('h-')) {
    ElMessage.info('层级线由父子关系生成，删除子节点即可移除')
    return
  }
  void removeEdge(edge.id)
}

// ---------- 拖拽落点持久化（含位置撤销 + 学段分区约束）----------
const dragStartPos = ref<Map<string, { x: number; y: number }>>(new Map())

/** 节点必须停留在自己学段分区内（允许少量溢出），y 只限制在画布顶部以下 */
function clampToGradeCol(node: KNode): { x: number; y: number } {
  const col = gradeColumnRange(gradeColumnIndex(matchGrade(node.stage)?.key))
  const x0 = col.x0 - ZONE_TOLERANCE + COL_PAD
  const x1 = col.x1 + ZONE_TOLERANCE - COL_PAD
  const x = Math.min(Math.max(node.pos_x ?? 0, x0), Math.max(x0, x1 - nodeWidth))
  const y = Math.max(node.pos_y ?? 0, 40)
  return { x: Math.round(x), y: Math.round(y) }
}

function onDragStart(e: { node: GraphNode }) {
  dragStartPos.value.set(e.node.id, { x: e.node.position.x, y: e.node.position.y })
}

let lastClampWarnAt = 0
function onDragStop(e: { node: GraphNode }) {
  const n = store.byId.get(e.node.id)
  if (!n) return
  const dropped = { x: Math.round(e.node.position.x), y: Math.round(e.node.position.y) }
  const limited = clampToGradeCol({ ...n, pos_x: dropped.x, pos_y: dropped.y })
  const x = limited.x
  const y = limited.y
  // 只有被明显拽回（>30px）才提示，轻微贴边校正不打扰
  if (Math.abs(x - dropped.x) > 30 || Math.abs(y - dropped.y) > 30) {
    const now = Date.now()
    if (now - lastClampWarnAt > 2500) {
      lastClampWarnAt = now
      const g = matchGrade(n.stage)
      ElMessage.info(`知识点不能拖出「${g?.label ?? UNSET_GRADE.label}」学段分区`)
    }
  }
  if (n.pos_x !== x || n.pos_y !== y) {
    const old = dragStartPos.value.get(e.node.id)
    const id = n.id
    void store.savePosition(id, x, y)
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

// ---------- 自动排布（学段分区内的左→右层级树，参考 mindmap）----------
// 每个学段占一条纵向分区（与顶部彩条一一对应）；分区内做横向整齐树：
// 根在左侧，子节点向右一层层展开；兄弟纵向排列，父节点垂直居中于子块。
const COL_PAD = 28
const SIB_GAP = 18 // 兄弟节点垂直间距
const LEVEL_GAP_X = 44 // 层与层的水平间距
const TOP_Y = 60
// 分区两侧允许的少量溢出（深层级允许越过分区中线一点）
const ZONE_TOLERANCE = 240

function computeLayout(): Map<string, { x: number; y: number }> {
  const pos = new Map<string, { x: number; y: number }>()

  // 按学段分区分组
  const groups = new Map<number, KNode[]>()
  for (const n of store.nodes) {
    const gi = gradeColumnIndex(matchGrade(n.stage)?.key)
    if (!groups.has(gi)) groups.set(gi, [])
    groups.get(gi)!.push(n)
  }

  for (const [gi, list] of [...groups.entries()].sort((a, b) => a[0] - b[0])) {
    const col = gradeColumnRange(gi)
    const colX0 = col.x0 + COL_PAD

    const idsInGroup = new Set(list.map((n) => n.id))
    const childrenOf = new Map<string, KNode[]>()
    const roots: KNode[] = []
    for (const n of list) {
      // 父节点在同一学段内才构成组内层级，否则视为该组的根
      if (n.parent_id && idsInGroup.has(n.parent_id)) {
        if (!childrenOf.has(n.parent_id)) childrenOf.set(n.parent_id, [])
        childrenOf.get(n.parent_id)!.push(n)
      } else {
        roots.push(n)
      }
    }

    let cursorY = TOP_Y
    const visited = new Set<string>()

    /** 返回该子树占用的总高度 */
    const place = (n: KNode, depth: number): number => {
      visited.add(n.id)
      const kids = childrenOf.get(n.id) ?? []
      const x = colX0 + depth * (nodeWidth + LEVEL_GAP_X)
      if (!kids.length) {
        pos.set(n.id, { x, y: cursorY })
        cursorY += nodeHeight + SIB_GAP
        return nodeHeight + SIB_GAP
      }
      const startY = cursorY
      let totalH = 0
      for (const k of kids) totalH += place(k, depth + 1)
      const endY = cursorY - SIB_GAP // 最后一个子节点的下边界
      // 父节点垂直居中于子块
      const y = startY + Math.max(0, (endY - startY - nodeHeight) / 2)
      pos.set(n.id, { x, y })
      return totalH
    }

    for (const r of roots) void place(r, 0)

    // 组内兜底：环等异常数据未放置的，排在已用区域下方
    if (visited.size < list.length) {
      const maxY = Math.max(TOP_Y, ...[...pos.values()].map((p) => p.y))
      let fy = maxY + nodeHeight + SIB_GAP
      for (const n of list) {
        if (visited.has(n.id)) continue
        pos.set(n.id, { x: colX0, y: fy })
        fy += nodeHeight + SIB_GAP
      }
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

// 提供给节点内 hover 操作条做反向缩放补偿（保持屏幕尺寸恒定）
const vpZoom = computed(() => {
  void vpTick.value
  return readViewport().zoom
})

// 连线创建时用户拖拽的把手方向（会话内记忆，edgeId → {s,t}，t/r/b/l）
const edgeAnchors = ref(new Map<string, { s?: string; t?: string }>())

/** 建立连线：只能上层 → 下层；返回创建结果（null=被拒绝） */
async function createEdgeBetween(
  sourceId: string,
  targetId: string,
  handles?: { s?: string | null; t?: string | null },
): Promise<boolean> {
  const ds = depthOf(sourceId)
  const dt = depthOf(targetId)
  if (ds >= dt) {
    ElMessage.warning('连线表示学习先后：只能从上层知识点连向下层（同级或向上不允许）')
    return false
  }
  try {
    const e = await store.createEdge(sourceId, targetId, 'prerequisite')
    if (handles?.s || handles?.t) {
      edgeAnchors.value.set(e.id, { s: handles.s ?? undefined, t: handles.t ?? undefined })
    }
    pushUndo({
      label: '连线',
      undo: async () => {
        await store.deleteEdgeRaw(e.id)
      },
      redo: async () => {
        await store.createEdgeRaw({ id: e.id, source_id: e.source_id, target_id: e.target_id, relation: e.relation })
      },
    })
    ElMessage.success('已连接（学习先后）')
    return true
  } catch (e) {
    ElMessage.warning(e instanceof Error ? e.message : String(e))
    return false
  }
}

function onConnect(conn: Connection) {
  if (conn.source && conn.target && conn.source !== conn.target) {
    void createEdgeBetween(conn.source, conn.target, {
      s: conn.sourceHandle ?? null,
      t: conn.targetHandle ?? null,
    })
  }
}

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
  count: number
  start: number
  end: number
}

// 学段显示边界（世界坐标）：相邻学段按内容外沿取中点，空学段给固定槽位；
// 最左/最右学段的边界无限延伸，拖到尽头也不会收起。
const gradeSegments = computed<GradeSegment[]>(() => {
  const r = visibleRect.value

  // 每个学段的内容范围；无节点的学段用分区中心当作零宽锚点
  const items: StageBucket[] = [...GRADES, UNSET_GRADE as unknown as GradeDef].map((g) => ({
    key: g.key,
    label: g.label,
    color: g.color,
    count: 0,
    start: 0,
    end: 0,
  }))
  const byKey = new Map(items.map((it) => [it.key, it]))
  for (const n of store.nodes) {
    const it = byKey.get(matchGrade(n.stage)?.key ?? UNSET_GRADE.key)
    if (!it) continue
    const x0 = n.pos_x ?? 0
    if (it.count === 0) {
      it.start = x0
      it.end = x0 + nodeWidth
    } else {
      it.start = Math.min(it.start, x0)
      it.end = Math.max(it.end, x0 + nodeWidth)
    }
    it.count++
  }
  // 学段锚点：有节点 → 内容范围中点（学段宽度随节点分布自动伸展）；
  // 空学段 → 固定分区中心，但只有视口真正越过分界才会显示
  for (const it of items) {
    if (it.count === 0) {
      const col = gradeColumnRange(gradeColumnIndex(it.key))
      it.start = it.end = (col.x0 + col.x1) / 2
    } else {
      const mid = (it.start + it.end) / 2
      it.start = it.end = mid
    }
  }

  // 相邻学段边界 = 两侧锚点的中点；
  // 最左/最右学段的边界向外无限延伸——拖到画布尽头时端点学段始终保持点亮；
  // 再做一次包含性校正：任何学段的分界都不允许切进自己的节点内容
  const bounds: number[] = [Number.NEGATIVE_INFINITY]
  for (let k = 1; k < items.length; k++) {
    bounds.push((items[k - 1].end + items[k].start) / 2)
  }
  bounds.push(Number.POSITIVE_INFINITY)
  const CONTAIN_PAD = 16
  for (let k = 0; k < items.length; k++) {
    if (items[k].count === 0) continue
    if (bounds[k] > items[k].start - CONTAIN_PAD) {
      const v = items[k].start - CONTAIN_PAD
      if (Number.isFinite(bounds[k])) bounds[k] = Math.min(bounds[k], v)
      else bounds[k] = v
    }
    if (bounds[k + 1] < items[k].end + CONTAIN_PAD) {
      const v = items[k].end + CONTAIN_PAD
      if (Number.isFinite(bounds[k + 1])) bounds[k + 1] = Math.max(bounds[k + 1], v)
      else bounds[k + 1] = v
    }
  }

  return items.map((it, k) => ({
    key: it.key,
    label: it.label,
    color: it.color,
    count: it.count,
    inView: bounds[k] <= r.x1 && bounds[k + 1] >= r.x0,
  }))
})

/** 视口中心当前所处的学段标签（未知领域返回 ''），用于新建节点时给默认学段 */
function currentViewportGrade(): string {
  const r = visibleRect.value
  const cx = (r.x0 + r.x1) / 2
  const raw = [...GRADES, UNSET_GRADE as unknown as GradeDef].map((g) => {
    const list = store.nodes.filter((n) => (matchGrade(n.stage)?.key ?? UNSET_GRADE.key) === g.key)
    let c: number
    if (!list.length) {
      const col = gradeColumnRange(gradeColumnIndex(g.key))
      c = (col.x0 + col.x1) / 2
    } else {
      const minX = Math.min(...list.map((n) => n.pos_x ?? 0))
      const maxX = Math.max(...list.map((n) => (n.pos_x ?? 0) + nodeWidth))
      c = (minX + maxX) / 2
    }
    return { key: g.key, label: g.label, start: c, end: c }
  })
  const bs: number[] = [Number.NEGATIVE_INFINITY]
  for (let k = 1; k < raw.length; k++) bs.push((raw[k - 1].end + raw[k].start) / 2)
  bs.push(Number.POSITIVE_INFINITY)
  const CONTAIN_PAD = 16
  for (let k = 0; k < raw.length; k++) {
    if (bs[k] > raw[k].start - CONTAIN_PAD) bs[k] = Number.isFinite(bs[k]) ? Math.min(bs[k], raw[k].start - CONTAIN_PAD) : bs[k]
    if (bs[k + 1] < raw[k].end + CONTAIN_PAD) bs[k + 1] = Math.max(bs[k + 1], raw[k].end + CONTAIN_PAD)
  }
  for (let k = 0; k < raw.length; k++) {
    if (bs[k] <= cx && cx <= bs[k + 1]) {
      return raw[k].key === UNSET_GRADE.key ? '' : raw[k].label
    }
  }
  return ''
}

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

/** 中心按钮：定位到「当前节点」——优先选中节点，其次离视口中心最近的节点，保证不落在空页面 */
function focusCurrent() {
  let target = selectedNode.value
  if (!target && store.nodes.length) {
    const v = readViewport()
    const cx = (wrapSize.value.w / 2 - v.x) / v.zoom
    const cy = (wrapSize.value.h / 2 - v.y) / v.zoom
    target = [...store.nodes].sort((a, b) => {
      const da = ((a.pos_x ?? 0) + nodeWidth / 2 - cx) ** 2 + ((a.pos_y ?? 0) + nodeHeight / 2 - cy) ** 2
      const db = ((b.pos_x ?? 0) + nodeWidth / 2 - cx) ** 2 + ((b.pos_y ?? 0) + nodeHeight / 2 - cy) ** 2
      return da - db
    })[0]
  }
  if (!target) {
    ElMessage.info('画布还没有知识点')
    return
  }
  selectedNodeId.value = target.id
  void fitView({ nodes: [target.id], duration: 300, padding: 0.4 })
}

// ---------- 键盘 ----------
function onKeydown(ev: KeyboardEvent) {
  const tag = (ev.target as HTMLElement)?.tagName?.toLowerCase()
  if (tag === 'input' || tag === 'textarea') return
  if (ev.key === 'Escape') {
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

// 关联知识点跳转：切换详情并居中该节点
function jumpTo(id: string) {
  selectedNodeId.value = id
  void nextTick(() => fitView({ nodes: [id], duration: 300, padding: 0.4 }))
}

// 抽屉关闭时冲刷未保存的正文
function onDrawerClosed() {
  drawerRef.value?.flushSave()
}

// ---------- 新建节点：功能坞「新建」= 顶层/选中节点的子级；节点 hover 操作条 = 下级/同级 ----------
const selectedNode = computed<KNode | undefined>(() =>
  selectedNodeId.value ? store.byId.get(selectedNodeId.value) : undefined,
)

// 当前选中知识点所属学段：对应彩条始终展开并高亮
const activeGradeKey = computed(() => {
  const n = selectedNode.value
  return n ? (matchGrade(n.stage)?.key ?? UNSET_GRADE.key) : null
})

/** 在学段分区内为节点挑一个落点 */
function placeInGradeCol(stage: string | null, prefer?: { x: number; y: number }): { x: number; y: number } {
  const fake = { stage, pos_x: prefer?.x ?? 0, pos_y: prefer?.y ?? 60 } as KNode
  return clampToGradeCol(fake)
}

const SLOT_GAP_X = 70
const SLOT_GAP_Y = 22

function overlapsAny(x: number, y: number, ignoreId?: string): boolean {
  for (const n of store.nodes) {
    if (n.id === ignoreId) continue
    const nx = n.pos_x ?? 0
    const ny = n.pos_y ?? 0
    if (x < nx + nodeWidth + 8 && x + nodeWidth + 8 > nx && y < ny + nodeHeight + 8 && y + nodeHeight + 8 > ny) {
      return true
    }
  }
  return false
}

/**
 * 找一个不与现有节点重叠的空位（脑图式排布）：
 * 先从期望位置向下逐行扫描，再向上扫描，都满则继续下探。
 */
function findFreeSpot(stage: string | null, wantX: number, wantY: number): { x: number; y: number } {
  const base = placeInGradeCol(stage, { x: wantX, y: wantY })
  const stepY = nodeHeight + SLOT_GAP_Y
  for (let k = 0; k <= 24; k++) {
    for (const dir of k === 0 ? [1] : [1, -1]) {
      const cand = placeInGradeCol(stage, { x: base.x, y: base.y + dir * k * stepY })
      if (!overlapsAny(cand.x, cand.y)) return cand
    }
  }
  return base
}

async function createWithPosition(
  title: string,
  parentId: string | null,
  stage: string | null,
  x: number,
  y: number,
): Promise<KNode> {
  const n = await store.createNode(title, parentId, stage)
  await store.savePosition(n.id, x, y)
  if (!parentId) {
    addNodes([
      {
        id: n.id,
        type: 'kt',
        position: { x, y },
        data: { node: { ...n, pos_x: x, pos_y: y }, selected: false, pending: false },
      },
    ])
  }
  return n
}

// ---------- 新建节点对话框：标题 + 学段 ----------
interface AddDialogState {
  open: boolean
  mode: 'root' | 'child' | 'sibling'
  parentId: string | null
  parentTitle: string
  /** 锚点节点：child=父节点；sibling=参照同级节点（位置排在其下方） */
  anchorId: string | null
  title: string
  stage: string
}
const addDialog = ref<AddDialogState>({
  open: false,
  mode: 'root',
  parentId: null,
  parentTitle: '',
  anchorId: null,
  title: '',
  stage: '',
})
const lastUsedStage = ref('')

function openAddDialog(mode: AddDialogState['mode'], base?: KNode | null) {
  const parent = mode === 'child' ? (base ?? null) : mode === 'sibling' ? (base?.parent_id ? store.byId.get(base.parent_id) ?? null : null) : null
  addDialog.value = {
    open: true,
    mode,
    parentId: parent?.id ?? null,
    parentTitle: parent?.title ?? '',
    // 记住锚点，确认时以它定位（而不是当时的选中节点，避免错位）
    anchorId: mode === 'root' ? null : base?.id ?? null,
    title: '',
    // 默认学段：子级/同级继承所属父节点的学段；顶层用「当前视口所在学段」，其次上次选择
    stage: parent ? parent.stage ?? '' : currentViewportGrade() || lastUsedStage.value,
  }
}

/** 功能坞「新建」：有选中节点时挂为其子级，否则建为顶层节点 */
function addNodeManual() {
  const sel = selectedNode.value
  if (sel) openAddDialog('child', sel)
  else openAddDialog('root')
}

async function confirmAddNode() {
  const d = addDialog.value
  const title = d.title.trim()
  if (!title) {
    ElMessage.warning('标题不能为空')
    return
  }
  const stage = d.stage || null
  lastUsedStage.value = d.stage
  try {
    let x: number
    let y: number
    if (d.mode === 'child' && d.parentId) {
      const parent = store.byId.get(d.parentId)
      if (!parent) throw new Error('父节点不存在')
      x = (parent.pos_x ?? 0) + nodeWidth + SLOT_GAP_X
      y = parent.pos_y ?? 60
    } else if (d.mode === 'sibling') {
      // 同级节点排在本节点正下方（锚点=点击 ＋同级 的那个节点，而非当前选中）
      const anchor = d.anchorId ? store.byId.get(d.anchorId) : undefined
      x = anchor?.pos_x ?? 0
      y = (anchor?.pos_y ?? 60) + nodeHeight + SLOT_GAP_Y
    } else {
      const v = readViewport()
      x = Math.round((wrapSize.value.w / 2 - v.x) / v.zoom - nodeWidth / 2)
      y = Math.round((wrapSize.value.h / 2 - v.y) / v.zoom - nodeHeight / 2)
    }
    const p = findFreeSpot(stage, x, y)
    const n = await createWithPosition(title, d.mode === 'root' ? null : d.parentId, stage, p.x, p.y)
    addDialog.value.open = false
    selectedNodeId.value = n.id
    ElMessage.success(`已创建${stage ? `（${stage}）` : ''}`)
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
const subStage = ref('')
const subCount = ref(8)
const subGenerating = ref(false)
const subTree = ref<DraftTreeNode[]>([])
const inserting = ref(false)

function openSubDialog() {
  subTopic.value = ''
  subTree.value = []
  subParent.value = selectedNodeId.value
  // 默认学段：挂载点的学段；挂在根时用当前视口所在学段
  const parent = selectedNodeId.value ? store.byId.get(selectedNodeId.value) : undefined
  subStage.value = parent?.stage ?? currentViewportGrade() ?? ''
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

async function insertDraftTree(list: DraftTreeNode[], parentId: string | null, stage?: string | null): Promise<number> {
  let count = 0
  for (const d of list) {
    const n = await store.createNode(d.title, parentId, stage ?? null)
    count++
    if (d.children?.length) count += await insertDraftTree(d.children, n.id, stage ?? n.stage)
  }
  return count
}

async function confirmInsertSubtree() {
  inserting.value = true
  try {
    // 学段优先级：对话框所选 > 挂载点学段；整棵子树统一使用，保证落进对应分区
    const parentStage = subParent.value ? store.byId.get(subParent.value)?.stage ?? null : null
    const stage = subStage.value || parentStage
    const total = await insertDraftTree(subTree.value, subParent.value, stage)
    subOpen.value = false
    await autoLayout(true)
    ElMessage.success(`已插入 ${total} 个知识点${stage ? `（${stage}）` : '（未知领域）'}`)
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
// 打开设置/统计前先收起详情抽屉，避免两层模态互相遮挡导致点击无效
const settingsOpen = ref(false)
const statsOpen = ref(false)

function openSettings() {
  drawerOpen.value = false
  settingsOpen.value = true
}

function openStats() {
  drawerOpen.value = false
  statsOpen.value = true
}
</script>

<template>
  <div class="canvas-page">
    <div ref="flowWrapRef" class="flow-wrap" :style="{ '--vf-zoom': vpZoom }">
      <VueFlow
        :nodes="flowNodes"
        :edges="flowEdges"
        :connection-mode="ConnectionMode.Loose"
        :default-edge-options="{ type: 'floating' }"
        :min-zoom="0.25"
        :max-zoom="2.5"
        fit-view-on-init
        @node-click="onNodeClick"
        @edge-click="onEdgeClick"
        @edge-double-click="onEdgeDblClick"
        @pane-click="onPaneClick"
        @connect="onConnect"
        @node-drag-start="onDragStart"
        @node-drag-stop="onDragStop"
        @move="onFlowMove"
      >
        <template #node-kt="ktProps">
          <KnowledgeNode
            :data="ktProps.data"
            @add-child="openAddDialog('child', ktProps.data.node)"
            @add-sibling="openAddDialog('sibling', ktProps.data.node)"
            @remove="removeNode(ktProps.data.node.id)"
          />
        </template>
        <template #edge-floating="floatingProps">
          <FloatingEdge v-bind="floatingProps" />
        </template>
        <Background :gap="22" pattern-color="#c9d3e3" />
        <MiniMap pannable zoomable />
      </VueFlow>

      <!-- 顶部分级彩条：视口进入的学段分区展开，其余收缩成色点；选中节点的学段常驻高亮 -->
      <GradeBar :segments="gradeSegments" :active-key="activeGradeKey" @select="focusGrade" />

      <!-- 右上角设置 -->
      <button class="gear-btn" title="设置" @click="openSettings">
        <el-icon><Setting /></el-icon>
      </button>

      <!-- 左下角地图式缩放 / 方位面板 -->
      <NavControls
        :zoom-percent="zoomPercent"
        @pan="panBy"
        @zoom-in="zoomInStep"
        @zoom-out="zoomOutStep"
        @fit="focusCurrent"
        @fit-all="fitAll"
      />

      <!-- 底部功能坞 -->
      <div class="dock">
        <button class="dock-btn" title="新建知识点（有选中节点时挂为其子级）" @click="addNodeManual">
          <span class="ico">➕</span><small>新建</small>
        </button>
        <button class="dock-btn" title="按学段分区整理成层级树布局" :disabled="layouting" @click="autoLayout()">
          <span class="ico">✨</span><small>{{ layouting ? '排布中…' : '排布' }}</small>
        </button>
        <button class="dock-btn" title="AI 批量生成知识点子树" @click="openSubDialog">
          <span class="ico">🤖</span><small>AI 子树</small>
        </button>

        <span class="dock-sep" />

        <button class="dock-btn" title="撤销（Ctrl+Z）" @click="doUndo">
          <span class="ico">↩️</span><small>撤销</small>
        </button>
        <button class="dock-btn" title="重做（Ctrl+Y）" @click="doRedo">
          <span class="ico">↪️</span><small>重做</small>
        </button>
        <button class="dock-btn" title="学习统计" @click="openStats">
          <span class="ico">📊</span><small>统计</small>
        </button>

        <span class="dock-sep" />

        <el-popover placement="top-end" :width="320" trigger="click">
          <div class="legend">
            <div class="legend__title">图例与帮助</div>
            <div class="legend__row"><i class="ln hierarchy" />层级/学习先后：上层学完再学下层（多上可连多下）</div>
            <div class="legend__tip">
              建连线：hover 节点出现锚点，按住从一个节点拖到另一个节点<br />
              同层/同级之间不允许连线；方向必须自上而下<br />
              删除连线：双击连线，或单击选中后按 Delete<br />
              顶部彩条 = 学段分区，宽度随该学段知识点数量变化
            </div>
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
        size="640px"
        append-to-body
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
          <el-select v-model="subStage" clearable placeholder="学段（默认跟随挂载点）" style="width: 100%">
            <el-option value="" label="未知领域（暂不归类）" />
            <el-option v-for="g in GRADES" :key="g.key" :value="g.label" :label="g.label" />
          </el-select>
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

      <!-- 新建知识点对话框：标题 + 学段 -->
      <el-dialog v-model="addDialog.open" title="➕ 新建知识点" width="420px" append-to-body>
        <el-form label-width="72px" @submit.prevent>
          <el-form-item label="挂载位置">
            <span class="dim">
              {{ addDialog.mode === 'root' ? '顶层节点' : addDialog.parentTitle ? `「${addDialog.parentTitle}」的下级` : '顶层节点' }}
            </span>
          </el-form-item>
          <el-form-item label="标题">
            <el-input
              v-model="addDialog.title"
              placeholder="知识点标题"
              maxlength="60"
              autofocus
              @keyup.enter="confirmAddNode"
            />
          </el-form-item>
          <el-form-item label="学段">
            <el-select v-model="addDialog.stage" clearable placeholder="选择所属学段" style="width: 100%">
              <el-option value="" label="未知领域（暂不归类）" />
              <el-option v-for="g in GRADES" :key="g.key" :value="g.label" :label="g.label" />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="addDialog.open = false">取消</el-button>
          <el-button type="primary" @click="confirmAddNode">创建</el-button>
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
