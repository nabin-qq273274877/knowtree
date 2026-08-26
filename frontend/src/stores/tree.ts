import { defineStore } from 'pinia'
import { api } from '@/api/client'
import type { KNode, NodeStatus, EdgeRelation, KEdge } from '@/types'

// ---------- 撤销/重做（S3，仅画布结构操作）----------

export interface UndoCommand {
  label: string
  undo: () => Promise<void>
  redo: () => Promise<void>
}

const undoStack: UndoCommand[] = []
const redoStack: UndoCommand[] = []
const MAX_HISTORY = 50

export function pushUndo(cmd: UndoCommand) {
  undoStack.push(cmd)
  if (undoStack.length > MAX_HISTORY) undoStack.shift()
  redoStack.length = 0
}

interface NodesState {
  nodes: KNode[]
  edges: KEdge[]
  loading: boolean
}

export const useTreeStore = defineStore('tree', {
  state: (): NodesState => ({
    nodes: [],
    edges: [],
    loading: false,
  }),

  getters: {
    byId(state): Map<string, KNode> {
      return new Map(state.nodes.map((n) => [n.id, n]))
    },
    // 树形结构：根 + children（M2 画布与后续大纲导航共用）
    tree(state): TreeNode[] {
      const childrenOf = new Map<string | null, KNode[]>()
      for (const n of state.nodes) {
        const key = n.parent_id ?? null
        if (!childrenOf.has(key)) childrenOf.set(key, [])
        childrenOf.get(key)!.push(n)
      }
      const build = (parent: string | null): TreeNode[] =>
        (childrenOf.get(parent) ?? []).map((n) => ({
          node: n,
          children: build(n.id),
        }))
      return build(null)
    },
  },

  actions: {
    async loadAll() {
      this.loading = true
      try {
        const [nodes, edges] = await Promise.all([
          api.get<KNode[]>('/api/nodes'),
          api.get<KEdge[]>('/api/edges'),
        ])
        // 空库时后端可能返回 null，统一兜底为数组
        this.nodes = Array.isArray(nodes) ? nodes : []
        this.edges = Array.isArray(edges) ? edges : []
      } finally {
        this.loading = false
      }
    },

    async createNode(title: string, parentId: string | null, stage?: string | null) {
      const n = await api.post<KNode>('/api/nodes', { title, parent_id: parentId, stage })
      this.nodes.push(n)
      return n
    },

    async setPositions(
      items: { id: string; pos_x: number; pos_y: number }[],
    ) {
      if (!items.length) return
      await api.post('/api/nodes/positions', { nodes: items })
      const map = new Map(items.map((i) => [i.id, i]))
      for (const n of this.nodes) {
        const p = map.get(n.id)
        if (p) {
          n.pos_x = p.pos_x
          n.pos_y = p.pos_y
        }
      }
    },

    async savePosition(id: string, pos_x: number, pos_y: number) {
      await this.setPositions([{ id, pos_x, pos_y }])
    },

    async updateNode(id: string, patch: Partial<Pick<KNode, 'title' | 'content_md' | 'status' | 'stage' | 'pos_x' | 'pos_y'>>) {
      const updated = await api.patch<KNode>(`/api/nodes/${id}`, patch)
      const i = this.nodes.findIndex((x) => x.id === id)
      if (i >= 0) this.nodes[i] = updated
      return updated
    },

    async moveNode(id: string, parentId: string | null) {
      const updated = await api.post<KNode>(`/api/nodes/${id}/move`, { parent_id: parentId })
      const i = this.nodes.findIndex((x) => x.id === id)
      if (i >= 0) this.nodes[i] = updated
      return updated
    },

    async deleteNode(id: string): Promise<number> {
      const res = await api.delete<{ deleted: number }>(`/api/nodes/${id}`)
      this.nodes = this.nodes.filter(
        (n) => n.id !== id && !this.isDescendantCached(n.id, id),
      )
      await this.loadAll() // 级联删除范围以服务端为准，简单起见全量刷新
      return res.deleted
    },

    isDescendantCached(nodeId: string, ancestorId: string): boolean {
      const map = this.byId
      let cur = map.get(nodeId)
      let guard = 0
      while (cur?.parent_id && guard < 64) {
        if (cur.parent_id === ancestorId) return true
        cur = map.get(cur.parent_id)
        guard++
      }
      return false
    },

    async createEdge(sourceId: string, targetId: string, relation: EdgeRelation) {
      const e = await api.post<KEdge>('/api/edges', {
        source_id: sourceId,
        target_id: targetId,
        relation,
      })
      this.edges.push(e)
      return e
    },

    async deleteEdge(id: string) {
      await api.delete(`/api/edges/${id}`)
      this.edges = this.edges.filter((e) => e.id !== id)
    },

    setStatus(id: string, status: NodeStatus) {
      return this.updateNode(id, { status })
    },

    // ---------- 撤销/重做底层原语（不记录命令）----------

    async createNodeRaw(payload: {
      id?: string
      title: string
      parent_id?: string | null
      stage?: string | null
      status?: NodeStatus
      content_md?: string
    }) {
      return api.post<KNode>('/api/nodes', payload)
    },

    async deleteNodeCascadeRaw(id: string): Promise<number> {
      const r = await api.delete<{ deleted: number }>(`/api/nodes/${id}`)
      this.nodes = this.nodes.filter((n) => n.id !== id)
      await this.loadAll()
      return r.deleted
    },

    async createEdgeRaw(e: { id: string; source_id: string; target_id: string; relation: EdgeRelation; label?: string | null }) {
      const created = await api.post<KEdge>('/api/edges', e)
      this.edges.push(created)
      return created
    },

    async deleteEdgeRaw(id: string) {
      await api.delete(`/api/edges/${id}`)
      this.edges = this.edges.filter((e) => e.id !== id)
    },

    // ---------- 命令栈操作 ----------

    canUndo(): boolean {
      return undoStack.length > 0
    },
    canRedo(): boolean {
      return redoStack.length > 0
    },
    async undo() {
      const cmd = undoStack.pop()
      if (!cmd) return
      await cmd.undo()
      redoStack.push(cmd)
    },
    async redo() {
      const cmd = redoStack.pop()
      if (!cmd) return
      await cmd.redo()
      undoStack.push(cmd)
    },
  },
})

export interface TreeNode {
  node: KNode
  children: TreeNode[]
}
