<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
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
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { ElMessage, ElMessageBox } from 'element-plus'
import { MagicStick, FullScreen } from '@element-plus/icons-vue'

import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

import { useTreeStore } from '@/stores/tree'
import KnowledgeNode from '@/components/canvas/KnowledgeNode.vue'
import { LINE_STYLE, RELATION_LABEL, STATUS_META } from '@/utils/meta'
import type { EdgeRelation, KNode, NodeStatus } from '@/types'

const store = useTreeStore()
const { fitView, addNodes } = useVueFlow()

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

async function createEdgeBetween(sourceId: string, targetId: string) {
  try {
    await store.createEdge(sourceId, targetId, relationType.value)
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

// 点选连线：选中 A 后再点 B 即连 A-B
function onNodeClick({ node }: NodeMouseEvent) {
  const id = node.id as string
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

// ---------- 拖拽落点持久化 ----------
function onDragStop(e: { node: GraphNode }) {
  const n = store.byId.get(e.node.id)
  if (!n) return
  const x = Math.round(e.node.position.x)
  const y = Math.round(e.node.position.y)
  if (n.pos_x !== x || n.pos_y !== y) {
    void store.savePosition(n.id, x, y)
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
    const pos = computeLayout()
    await store.setPositions([...pos.entries()].map(([id, p]) => ({ id, pos_x: p.x, pos_y: p.y })))
    if (!silent) ElMessage.success('已自动排布')
    await nextTick(() => fitView({ padding: 0.12, duration: 250 }))
  } catch (e) {
    if (!silent) ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    layouting.value = false
  }
}

// ---------- 键盘 ----------
function onKeydown(ev: KeyboardEvent) {
  const tag = (ev.target as HTMLElement)?.tagName?.toLowerCase()
  if (tag === 'input' || tag === 'textarea') return
  if (ev.key === 'Escape') {
    clearPending()
    selectedNodeId.value = null
    selectedEdgeId.value = null
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
  try {
    await store.deleteEdge(id)
    selectedEdgeId.value = null
    ElMessage.success('已删除连线')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  }
}

// ---------- 节点操作 ----------
const showInspector = computed(() => !!selectedNode.value)
const selectedNode = computed<KNode | undefined>(() =>
  selectedNodeId.value ? store.byId.get(selectedNodeId.value) : undefined,
)

async function renameSelected() {
  const n = selectedNode.value
  if (!n) return
  const { value } = await ElMessageBox.prompt('新标题', '重命名节点', {
    inputValue: n.title,
    confirmButtonText: '保存',
    cancelButtonText: '取消',
  })
  if (value?.trim()) await store.updateNode(n.id, { title: value.trim() })
}

async function addChild() {
  const n = selectedNode.value
  if (!n) return
  const { value } = await ElMessageBox.prompt('子节点标题', `在「${n.title}」下新增`, {
    confirmButtonText: '创建',
    cancelButtonText: '取消',
  })
  if (!value?.trim()) return
  const child = await store.createNode(value.trim(), n.id)
  // 放在父节点右下并持久化
  const x = (n.pos_x ?? 0) + nodeWidth + 40
  const y = (n.pos_y ?? 0) + nodeHeight + 50
  await store.savePosition(child.id, x, y)
  addNodes([
    {
      id: child.id,
      type: 'kt',
      position: { x, y },
      data: { node: { ...child, pos_x: x, pos_y: y }, selected: false, pending: false },
    },
  ])
  selectedNodeId.value = child.id
}

async function addSibling() {
  const n = selectedNode.value
  if (!n) return
  const { value } = await ElMessageBox.prompt('同级节点标题', `在「${n.title}」旁新增`, {
    confirmButtonText: '创建',
    cancelButtonText: '取消',
  })
  if (!value?.trim()) return
  const sib = await store.createNode(value.trim(), n.parent_id)
  const x = (n.pos_x ?? 0) + nodeWidth + 60
  const y = n.pos_y ?? 0
  await store.savePosition(sib.id, x, y)
  selectedNodeId.value = sib.id
  ElMessage.success('已创建')
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
  const deleted = await store.deleteNode(id)
  selectedNodeId.value = null
  ElMessage.success(`已删除 ${deleted} 个节点`)
}

async function setStatus(status: NodeStatus) {
  const n = selectedNode.value
  if (!n) return
  await store.setStatus(n.id, status)
}
</script>

<template>
  <div class="canvas-page">
    <!-- 工具栏 -->
    <div class="toolbar">
      <span class="brand">🗺️ 知识画布</span>
      <el-divider direction="vertical" />
      <el-radio-group v-model="relationType" size="small">
        <el-radio-button value="related">一般关联</el-radio-button>
        <el-radio-button value="prerequisite">前置依赖</el-radio-button>
      </el-radio-group>
      <el-button :icon="MagicStick" size="small" :loading="layouting" @click="autoLayout()">
        自动排布
      </el-button>
      <el-button :icon="FullScreen" size="small" @click="fitView({ padding: 0.12, duration: 250 })">
        适应屏幕
      </el-button>
      <span class="hint">
        <b>锚点拉线</b> 或 <b>点选两节点</b> 建立连线 · <b>Delete</b> 删除选中 · <b>Esc</b> 取消 · 双击空白处拖动平移
      </span>
    </div>

    <!-- 画布 -->
    <div class="flow-wrap">
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
        @node-drag-stop="onDragStop"
      >
        <template #node-kt="ktProps">
          <KnowledgeNode :data="ktProps.data" />
        </template>
        <Background :gap="22" pattern-color="#c9d3e3" />
        <Controls />
        <MiniMap pannable zoomable />
      </VueFlow>

      <!-- 图例 -->
      <div class="legend">
        <span><i class="line hierarchy"></i>层级</span>
        <span><i class="line prerequisite"></i>前置依赖</span>
        <span><i class="line related"></i>一般关联</span>
      </div>

      <!-- 右侧迷你操作面板（M3 将升级为完整详情面板） -->
      <transition name="slide">
        <div v-if="showInspector && selectedNode" class="inspector">
          <div class="inspector__title">{{ selectedNode.title }}</div>
          <div class="inspector__row">
            <span class="dot" :style="{ background: STATUS_META[selectedNode.status].color }" />
            {{ STATUS_META[selectedNode.status].label }}
            <span class="stage-tag" v-if="selectedNode.stage">{{ selectedNode.stage }}</span>
          </div>

          <div class="inspector__section">学习状态</div>
          <el-select :model-value="selectedNode.status" size="small" @change="(v: NodeStatus) => setStatus(v)">
            <el-option v-for="(m, k) in STATUS_META" :key="k" :value="k" :label="m.label" />
          </el-select>

          <div class="inspector__section">操作</div>
          <div class="inspector__btns">
            <el-button size="small" @click="addChild">＋ 下级</el-button>
            <el-button size="small" @click="addSibling">＋ 同级</el-button>
            <el-button size="small" @click="renameSelected">重命名</el-button>
            <el-button size="small" type="danger" plain @click="removeNode(selectedNode!.id)">删除</el-button>
          </div>

          <div class="inspector__section">关联</div>
          <div class="inspector__rel">
            {{
              store.edges.filter((e) => e.source_id === selectedNode!.id || e.target_id === selectedNode!.id).length
            }}
            条连线 · 详情面板 M3 提供
          </div>
        </div>
      </transition>
    </div>
  </div>
</template>

<style scoped>
.canvas-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.toolbar {
  height: 48px;
  background: #fff;
  border-bottom: 1px solid #e4e9f2;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 14px;
  flex-shrink: 0;
}

.brand {
  font-weight: 700;
  font-size: 14px;
  color: #26334d;
}

.hint {
  margin-left: auto;
  font-size: 12px;
  color: #8a94a6;
}

.hint b {
  color: #51607a;
}

.flow-wrap {
  flex: 1;
  min-height: 0;
  position: relative;
}

.flow-wrap :deep(.vue-flow__pane) {
  cursor: grab;
}

.legend {
  position: absolute;
  left: 14px;
  bottom: 12px;
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid #e4e9f2;
  border-radius: 8px;
  padding: 6px 10px;
  display: flex;
  gap: 14px;
  font-size: 12px;
  color: #51607a;
  z-index: 5;
}

.legend .line {
  display: inline-block;
  width: 22px;
  height: 0;
  vertical-align: middle;
  margin-right: 5px;
  border-top: 2px solid;
}

.line.hierarchy {
  border-color: #9aa7bf;
  border-top-width: 1.5px;
}

.line.prerequisite {
  border-color: #e8590c;
  border-top-style: dashed;
}

.line.related {
  border-color: #5b8def;
}

.inspector {
  position: absolute;
  top: 14px;
  right: 14px;
  width: 240px;
  background: #fff;
  border: 1px solid #e4e9f2;
  border-radius: 10px;
  box-shadow: 0 6px 20px rgba(30, 50, 90, 0.14);
  padding: 14px;
  z-index: 6;
}

.inspector__title {
  font-weight: 700;
  font-size: 14px;
  color: #26334d;
  margin-bottom: 8px;
  word-break: break-all;
}

.inspector__row {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 12.5px;
  color: #51607a;
}

.dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
}

.stage-tag {
  border: 1px solid #d5dbe5;
  border-radius: 8px;
  padding: 1px 7px;
  font-size: 11px;
  color: #8a94a6;
}

.inspector__section {
  margin: 12px 0 6px;
  font-size: 11px;
  color: #98a2b3;
  letter-spacing: 1px;
}

.inspector__btns {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.inspector__btns .el-button {
  margin-left: 0;
}

.inspector__rel {
  font-size: 12px;
  color: #8a94a6;
}

.slide-enter-active,
.slide-leave-active {
  transition:
    transform 0.18s ease,
    opacity 0.18s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(16px);
  opacity: 0;
}
</style>
