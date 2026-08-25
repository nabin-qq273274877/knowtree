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
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { ElMessage, ElMessageBox } from 'element-plus'
import { MagicStick, FullScreen } from '@element-plus/icons-vue'

import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

import { useTreeStore, pushUndo } from '@/stores/tree'
import KnowledgeNode from '@/components/canvas/KnowledgeNode.vue'
import DetailDrawer from '@/components/canvas/DetailDrawer.vue'
import { LINE_STYLE, RELATION_LABEL, STATUS_META } from '@/utils/meta'
import { api } from '@/api/client'
import type { KEdge, KNode, EdgeRelation, NodeStatus } from '@/types'

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
// 连线模式开关：开启后点击节点用于连线（对齐 tree-link.html 点选交互），关闭时点击=打开详情
const connectMode = ref(false)

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

// 当前选中节点（操作函数共用）
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

async function setStatus(status: NodeStatus) {
  if (selectedNodeId.value) await store.setStatus(selectedNodeId.value, status)
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
      <el-button size="small" @click="openSubDialog">🤖 生成子树</el-button>
      <el-button :icon="FullScreen" size="small" @click="fitView({ padding: 0.12, duration: 250 })">
        适应屏幕
      </el-button>
      <el-divider direction="vertical" />
      <el-tooltip content="开启后：点击两个节点即建立所选类型的连线；关闭时点击节点打开详情" placement="bottom">
        <span class="mode-switch">
          连线模式
          <el-switch v-model="connectMode" size="small" />
        </span>
      </el-tooltip>
      <span class="hint">
        <template v-if="connectMode"><b>点选两节点</b>建立连线 · 或从锚点拉线</template>
        <template v-else><b>点击节点</b>查看详情 · 锚点拉线连线 · <b>Delete</b> 删除选中 · <b>Esc</b> 取消</template>
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
        @node-drag-start="onDragStart"
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
      <el-dialog v-model="subOpen" title="🤖 AI 生成知识点子树" width="560px">
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

.mode-switch {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #38445c;
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
