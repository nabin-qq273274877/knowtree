<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Plus } from '@element-plus/icons-vue'
import { useTreeStore, type TreeNode } from '@/stores/tree'
import type { NodeStatus } from '@/types'
import { dayjs } from '@/utils/day'

const store = useTreeStore()

onMounted(() => {
  void store.loadAll()
})

// ---- 状态展示配置 ----
const statusMeta: Record<NodeStatus, { label: string; type: 'info' | 'primary' | 'warning' | 'success' | 'danger' }> = {
  not_started: { label: '未学', type: 'info' },
  learning: { label: '学习中', type: 'primary' },
  partial: { label: '部分学会', type: 'warning' },
  mastered: { label: '已学会', type: 'success' },
  forgotten: { label: '已遗忘', type: 'danger' },
}

// ---- 新增节点表单 ----
const newTitle = ref('')
const newParent = ref<string | null>(null)
const creating = ref(false)

async function handleCreate() {
  const title = newTitle.value.trim()
  if (!title) return
  creating.value = true
  try {
    await store.createNode(title, newParent.value)
    newTitle.value = ''
    ElMessage.success('节点已创建')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    creating.value = false
  }
}

// ---- 父级选择树 ----
const parentOptions = computed(() => toOptions(store.tree))
function toOptions(list: TreeNode[]): { value: string; label: string; children: unknown[] }[] {
  return list.map((t) => ({
    value: t.node.id,
    label: t.node.title,
    children: toOptions(t.children),
  }))
}

function parentTitle(id: string | null): string {
  if (!id) return '—（根）—'
  return store.byId.get(id)?.title ?? '(未知)'
}

// ---- 删除 ----
function descendantCount(t: TreeNode): number {
  return t.children.reduce((sum, c) => sum + 1 + descendantCount(c), 0)
}
async function handleDelete(id: string) {
  const root = store.tree.find((t) => t.node.id === id)
  let count = 1
  if (root) {
    count = 1 + descendantCount(root)
  }
  try {
    await ElMessageBox.confirm(
      `将删除「${store.byId.get(id)?.title}」及其子树，共 ${count} 个节点。确定？`,
      '删除确认',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  try {
    const deleted = await store.deleteNode(id)
    ElMessage.success(`已删除 ${deleted} 个节点`)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  }
}

// ---- 状态切换 ----
async function handleStatus(id: string, status: NodeStatus) {
  try {
    await store.setStatus(id, status)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  }
}
</script>

<template>
  <div class="page">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      title="M1 临时管理页"
      description="画布视图将在 M2 提供（空间卡片+连线）。当前页面用于验证数据链路。"
      style="margin-bottom: 16px"
    />

    <el-card shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 12px">
          <span style="font-weight: 600">新增知识点</span>
          <el-input
            v-model="newTitle"
            placeholder="标题，如：初中数学"
            style="width: 280px"
            clearable
            @keyup.enter="handleCreate"
          />
          <el-tree-select
            v-model="newParent"
            :data="parentOptions"
            :props="{ label: 'label', value: 'value' }"
            value-key="value"
            check-strictly
            clearable
            filterable
            placeholder="父节点（留空=根）"
            style="width: 220px"
          />
          <el-button type="primary" :icon="Plus" :loading="creating" @click="handleCreate">
            创建
          </el-button>
          <span style="margin-left: auto; color: #909399; font-size: 13px">
            共 {{ store.nodes.length }} 个节点 · {{ store.edges.length }} 条连线
          </span>
        </div>
      </template>

      <el-table :data="store.nodes" v-loading="store.loading" size="default" row-key="id">
        <el-table-column label="标题" min-width="220">
          <template #default="{ row }">{{ row.title }}</template>
        </el-table-column>
        <el-table-column label="父节点" min-width="160">
          <template #default="{ row }">{{ parentTitle(row.parent_id) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="170">
          <template #default="{ row }">
            <el-select
              :model-value="row.status"
              size="small"
              @change="(v: NodeStatus) => handleStatus(row.id, v)"
            >
              <el-option
                v-for="(meta, key) in statusMeta"
                :key="key"
                :value="key"
                :label="meta.label"
              />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="状态标签" width="110">
          <template #default="{ row }">
            <el-tag :type="statusMeta[row.status as NodeStatus].type" size="small">
              {{ statusMeta[row.status as NodeStatus].label }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="120">
          <template #default="{ row }">{{ dayjs.format(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="" width="80" align="right">
          <template #default="{ row }">
            <el-button
              type="danger"
              :icon="Delete"
              size="small"
              circle
              plain
              @click="handleDelete(row.id)"
            />
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="还没有节点，先创建一个吧（M2 将支持导入与一键生成学段骨架）" />
        </template>
      </el-table>
    </el-card>
  </div>
</template>
