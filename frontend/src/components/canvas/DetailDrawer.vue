<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { MdEditor, MdPreview } from 'md-editor-v3'
import type { ToolbarNames } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'

import { api } from '@/api/client'
import { useTreeStore } from '@/stores/tree'
import { GRADES, RELATION_LABEL, STATUS_META } from '@/utils/meta'
import { dayjs } from '@/utils/day'
import type { KAnnotation, KNode, KResource, NodeStatus } from '@/types'
import ExplainPane from './ExplainPane.vue'
import ExercisePanel from './ExercisePanel.vue'

const props = defineProps<{ nodeId: string | null }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'jump', id: string): void }>()

const store = useTreeStore()
const node = computed<KNode | undefined>(() => (props.nodeId ? store.byId.get(props.nodeId) : undefined))

// Tabs
const activeTab = ref('content')

// ---------- 面包屑 ----------
const breadcrumb = computed(() => {
  const path: KNode[] = []
  let cur = node.value
  let guard = 0
  while (cur && guard < 32) {
    path.unshift(cur)
    cur = cur.parent_id ? store.byId.get(cur.parent_id) : undefined
    guard++
  }
  return path
})

// ---------- 正文编辑（Markdown + KaTeX）----------
const toolbars: ToolbarNames[] = [
  'bold', 'italic', 'strikeThrough', '-',
  'title', 'quote', 'unorderedList', 'orderedList', '-',
  'code', 'table', 'katex', 'image', '-',
  'preview', 'catalog', '=',
]
const content = ref('')
const editing = ref(false)
const dirty = ref(false)
const savedAt = ref('')

watch(
  () => props.nodeId,
  () => {
    flushSave()
    content.value = node.value?.content_md ?? ''
    editing.value = false
    dirty.value = false
    loadResources()
    loadAnnotations()
  },
  { immediate: true },
)

let saveTimer: number | undefined
function onContentChange() {
  dirty.value = true
  window.clearTimeout(saveTimer)
  saveTimer = window.setTimeout(saveContent, 1200)
}

async function saveContent(silent = false) {
  window.clearTimeout(saveTimer)
  const n = node.value
  if (!n || !dirty.value) return
  try {
    await store.updateNode(n.id, { content_md: content.value })
    dirty.value = false
    savedAt.value = new Date().toLocaleTimeString()
    if (!silent) ElMessage.success('正文已保存')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  }
}

function flushSave() {
  if (dirty.value && node.value) void saveContent(true)
}

async function toggleEdit() {
  if (editing.value) {
    await saveContent()
  }
  editing.value = !editing.value
}

// ---------- 教学资源 ----------
const resources = ref<KResource[]>([])
const resLoading = ref(false)
const resForm = ref({ title: '', url: '', kind: 'link' as 'link' | 'file', note: '' })
const addingRes = ref(false)

async function loadResources() {
  if (!props.nodeId) return
  resLoading.value = true
  try {
    resources.value = await api.get<KResource[]>(`/api/nodes/${props.nodeId}/resources`)
  } finally {
    resLoading.value = false
  }
}

async function addResource() {
  if (!props.nodeId) return
  const f = resForm.value
  if (!f.title.trim()) {
    ElMessage.warning('请填写资源标题')
    return
  }
  if (f.kind === 'link' && !f.url.trim()) {
    ElMessage.warning('链接类型需要填写 URL')
    return
  }
  addingRes.value = true
  try {
    const r = await api.post<KResource>(`/api/nodes/${props.nodeId}/resources`, {
      title: f.title.trim(),
      kind: f.kind,
      url: f.url.trim() || null,
      note: f.note.trim() || null,
    })
    resources.value.push(r)
    resForm.value = { title: '', url: '', kind: 'link', note: '' }
    ElMessage.success('资源已添加')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    addingRes.value = false
  }
}

async function removeResource(id: string) {
  try {
    await api.delete(`/api/resources/${id}`)
    resources.value = resources.value.filter((r) => r.id !== id)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  }
}

const KIND_ICON: Record<string, string> = { link: '🔗', file: '📄' }

// ---------- 批注（FR-10）----------
const annotations = ref<KAnnotation[]>([])
const annLoading = ref(false)
const annDraft = ref('')
const postingAnn = ref(false)
const editingAnnId = ref<string | null>(null)
const editingAnnText = ref('')

async function loadAnnotations() {
  if (!props.nodeId) return
  annLoading.value = true
  try {
    annotations.value = await api.get<KAnnotation[]>(`/api/nodes/${props.nodeId}/annotations`)
  } finally {
    annLoading.value = false
  }
}

function adjustCount(delta: number) {
  const n = node.value
  if (n) n.annotation_count = Math.max(0, n.annotation_count + delta)
}

async function postAnnotation() {
  if (!props.nodeId || !annDraft.value.trim()) return
  postingAnn.value = true
  try {
    const a = await api.post<KAnnotation>(`/api/nodes/${props.nodeId}/annotations`, {
      content_md: annDraft.value,
    })
    annotations.value.unshift(a)
    annDraft.value = ''
    adjustCount(1)
    ElMessage.success('批注已保存')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    postingAnn.value = false
  }
}

function startEditAnn(a: KAnnotation) {
  editingAnnId.value = a.id
  editingAnnText.value = a.content_md
}

async function saveEditAnn(a: KAnnotation) {
  if (!editingAnnText.value.trim()) return
  try {
    const updated = await api.patch<KAnnotation>(`/api/annotations/${a.id}`, {
      content_md: editingAnnText.value,
    })
    const i = annotations.value.findIndex((x) => x.id === a.id)
    if (i >= 0) annotations.value[i] = updated
    editingAnnId.value = null
    ElMessage.success('批注已更新')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  }
}

async function removeAnn(a: KAnnotation) {
  try {
    await ElMessageBox.confirm('删除这条批注？', '确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
    await api.delete(`/api/annotations/${a.id}`)
    annotations.value = annotations.value.filter((x) => x.id !== a.id)
    adjustCount(-1)
  } catch {
    /* 取消 */
  }
}

// ---------- 关联知识点 ----------
interface RelatedItem {
  edgeId: string
  other: KNode | undefined
  relation: string
}
const relatedList = computed<RelatedItem[]>(() => {
  const n = node.value
  if (!n) return []
  return store.edges
    .filter((e) => e.source_id === n.id || e.target_id === n.id)
    .map((e) => ({
      edgeId: e.id,
      relation: e.relation,
      other: store.byId.get(e.source_id === n.id ? e.target_id : e.source_id),
    }))
})

async function setStatus(status: NodeStatus) {
  if (node.value) await store.setStatus(node.value.id, status)
}

// 设置学段：决定节点在画布的分区与彩条颜色
async function setStage(stage: string) {
  if (!node.value) return
  try {
    await store.updateNode(node.value.id, { stage: stage || null })
    ElMessage.success(stage ? `已归入「${stage}」学段` : '已归入「未知领域」')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  }
}

defineExpose({ flushSave })
</script>

<template>
  <div v-if="node" class="detail">
    <!-- 面包屑 -->
    <div class="crumbs">
      <template v-for="(p, i) in breadcrumb" :key="p.id">
        <span v-if="i > 0" class="sep">/</span>
        <a
          :class="{ self: p.id === node!.id }"
          @click.prevent="p.id !== node!.id && emit('jump', p.id)"
        >{{ p.title }}</a>
      </template>
    </div>

    <!-- 标题 + 状态 + 学段 -->
    <div class="head">
      <h2>{{ node.title }}</h2>
      <div class="head__selects">
        <el-select
          :model-value="node.stage ?? ''"
          size="small"
          style="width: 108px"
          title="所属学段：决定画布分区与顶部彩条颜色"
          @change="(v: string) => setStage(v)"
        >
          <el-option value="" label="未知领域" />
          <el-option v-for="g in GRADES" :key="g.key" :value="g.label" :label="g.label" />
        </el-select>
        <el-select :model-value="node.status" size="small" style="width: 130px" @change="(v: NodeStatus) => setStatus(v)">
          <el-option v-for="(m, k) in STATUS_META" :key="k" :value="k" :label="m.label" />
        </el-select>
      </div>
    </div>

    <!-- Tabs：正文 / AI讲解 / 资源 / 批注 / 练习 / 关联 -->
    <el-tabs v-model="activeTab">
      <!-- 正文 -->
      <el-tab-pane label="正文" name="content">
        <div class="sec__head" style="margin-bottom: 8px">
          <span class="save-state">{{ dirty ? '未保存…' : savedAt ? `已保存 ${savedAt}` : '' }}</span>
          <el-button size="small" @click="toggleEdit">{{ editing ? '完成' : '编辑' }}</el-button>
        </div>
        <MdEditor
          v-if="editing"
          v-model="content"
          :toolbars="toolbars"
          :style="{ height: '420px' }"
          language="zh-CN"
          placeholder="用 Markdown 记录知识点，支持 LaTeX 公式：$x^2$ 或 $$\\int_a^b f(x)dx$$"
          @on-change="onContentChange"
        />
        <div v-else class="preview-box">
          <MdPreview v-if="content" :model-value="content" theme="light" preview-theme="github" />
          <el-empty v-else description="暂无正文，点击「编辑」开始记录" :image-size="72" />
        </div>
      </el-tab-pane>

      <!-- AI 讲解 -->
      <el-tab-pane label="🤖 AI讲解" name="explain" lazy>
        <ExplainPane :node-id="node.id" />
      </el-tab-pane>

      <!-- 教学资源 -->
      <el-tab-pane :label="`资源 ${resources.length ? '(' + resources.length + ')' : ''}`" name="resources" lazy>
        <ul v-loading="resLoading" class="res-list">
          <li v-for="r in resources" :key="r.id" class="res-item">
            <span class="res-icon">{{ KIND_ICON[r.kind] ?? '🔗' }}</span>
            <a :href="r.url ?? '#'" target="_blank" rel="noopener" class="res-title">{{ r.title }}</a>
            <span v-if="r.note" class="res-note">{{ r.note }}</span>
            <el-button link type="danger" size="small" @click="removeResource(r.id)">删除</el-button>
          </li>
          <li v-if="!resources.length && !resLoading" class="empty">还没有资源</li>
        </ul>
        <div class="res-form">
          <el-input v-model="resForm.title" placeholder="标题，如：B站·分数入门课" size="small" style="width: 170px" />
          <el-select v-model="resForm.kind" size="small" style="width: 90px">
            <el-option value="link" label="链接" />
            <el-option value="file" label="文件(P1)" disabled />
          </el-select>
          <el-input v-model="resForm.url" placeholder="https://..." size="small" style="width: 190px" />
          <el-button type="primary" size="small" :loading="addingRes" @click="addResource">添加</el-button>
        </div>
      </el-tab-pane>

      <!-- 批注 -->
      <el-tab-pane :label="`批注 ${annotations.length ? '(' + annotations.length + ')' : ''}`" name="annotations" lazy>
        <div class="ann-input">
          <el-input
            v-model="annDraft"
            type="textarea"
            :rows="2"
            placeholder="随手记下你的理解、疑问或联想（支持 Markdown），Ctrl+Enter 发表"
            @keydown.ctrl.enter.prevent="postAnnotation"
          />
          <el-button type="primary" size="small" :loading="postingAnn" :disabled="!annDraft.trim()" @click="postAnnotation">
            发表
          </el-button>
        </div>
        <ul v-loading="annLoading" class="ann-list">
          <li v-for="a in annotations" :key="a.id" class="ann-item">
            <template v-if="editingAnnId === a.id">
              <el-input v-model="editingAnnText" type="textarea" :rows="3" />
              <div class="ann-actions">
                <el-button size="small" type="primary" @click="saveEditAnn(a)">保存</el-button>
                <el-button size="small" @click="editingAnnId = null">取消</el-button>
              </div>
            </template>
            <template v-else>
              <MdPreview :model-value="a.content_md" theme="light" preview-theme="github" />
              <div class="ann-foot">
                <span>{{ dayjs.format(a.updated_at) }}{{ a.updated_at !== a.created_at ? '（已编辑）' : '' }}</span>
                <span class="ann-actions-inline">
                  <el-button link size="small" @click="startEditAnn(a)">编辑</el-button>
                  <el-button link size="small" type="danger" @click="removeAnn(a)">删除</el-button>
                </span>
              </div>
            </template>
          </li>
          <li v-if="!annotations.length && !annLoading" class="empty">学过之后有心得？写在这里</li>
        </ul>
      </el-tab-pane>

      <!-- 练习题 -->
      <el-tab-pane label="📝 习题/试卷" name="exercises" lazy>
        <ExercisePanel :node-id="node.id" />
      </el-tab-pane>

      <!-- 关联知识点 -->
      <el-tab-pane :label="`关联 ${relatedList.length ? '(' + relatedList.length + ')' : ''}`" name="related" lazy>
        <ul class="rel-list">
          <li v-for="r in relatedList" :key="r.edgeId">
            <el-tag :type="r.relation === 'prerequisite' ? 'warning' : 'primary'" size="small">
              {{ RELATION_LABEL[r.relation as keyof typeof RELATION_LABEL] }}
            </el-tag>
            <a class="rel-link" @click.prevent="emit('jump', r.other!.id)">{{ r.other?.title ?? '(未知)' }}</a>
          </li>
          <li v-if="!relatedList.length" class="empty">在画布上连线即可建立关联</li>
        </ul>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.detail {
  padding: 4px 6px;
}

.crumbs {
  font-size: 12px;
  color: #8a94a6;
  margin-bottom: 8px;
  word-break: break-all;
}

.crumbs .sep {
  margin: 0 5px;
  color: #c4ccd9;
}

.crumbs a {
  color: #51607a;
  cursor: pointer;
}

.crumbs a:hover {
  color: #409eff;
}

.crumbs a.self {
  color: #26334d;
  font-weight: 600;
  cursor: default;
}

.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.head h2 {
  margin: 0;
  font-size: 18px;
  color: #1f2b45;
  word-break: break-all;
}

.head__selects {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

.sec {
  border-top: 1px solid #eef1f6;
  padding: 12px 0 4px;
  margin-bottom: 8px;
}

.sec__head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.sec__title {
  font-size: 13px;
  font-weight: 700;
  color: #38445c;
}

.save-state {
  font-size: 11px;
  color: #98a2b3;
  margin-left: auto;
}

.count {
  font-size: 11px;
  background: #f0f2f7;
  color: #68758c;
  padding: 1px 7px;
  border-radius: 8px;
}

.preview-box {
  border: 1px solid #eef1f6;
  border-radius: 8px;
  padding: 10px 14px;
  min-height: 60px;
  max-height: 420px;
  overflow: auto;
  background: #fafbfd;
}

.res-list,
.ann-list,
.rel-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.res-item {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 5px 0;
  font-size: 13px;
}

.res-icon {
  flex-shrink: 0;
}

.res-title {
  color: #2f6bdc;
  text-decoration: none;
  word-break: break-all;
}

.res-title:hover {
  text-decoration: underline;
}

.res-note {
  color: #98a2b3;
  font-size: 11.5px;
}

.res-form {
  display: flex;
  gap: 6px;
  margin-top: 8px;
  flex-wrap: wrap;
}

.ann-input {
  display: flex;
  gap: 6px;
  align-items: flex-end;
  margin-bottom: 10px;
}

.ann-item {
  border-bottom: 1px dashed #eef1f6;
  padding: 8px 0;
}

.ann-item :deep(.md-editor-preview-wrapper) {
  padding: 0;
}

.ann-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  color: #98a2b3;
  margin-top: 4px;
}

.ann-actions {
  margin-top: 6px;
  display: flex;
  gap: 6px;
}

.rel-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
  font-size: 13px;
}

.rel-link {
  color: #2f6bdc;
  cursor: pointer;
}

.rel-link:hover {
  text-decoration: underline;
}

.empty {
  color: #a5aec0;
  font-size: 12.5px;
  padding: 6px 0;
}
</style>
