<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { MdPreview } from 'md-editor-v3'
import { api } from '@/api/client'

const props = defineProps<{ nodeId: string }>()

// ---------- 类型 ----------
type ExType = 'single_choice' | 'multiple_choice' | 'true_false' | 'fill_blank' | 'short_answer'

interface ExOption {
  key: string
  text: string
}

interface KExercise {
  id: string
  node_id: string
  type: ExType
  question_md: string
  options_json: string | null
  answer_md?: string
  analysis_md?: string | null
  difficulty?: number | null
  answer_draft?: string | null
  result?: 'right' | 'partial' | 'wrong' | null
  score?: number | null
  feedback_md?: string | null
  answered_at?: number | null
  created_at?: number
}

interface Draft extends Omit<KExercise, 'options_json'> {
  options?: ExOption[]
}

const TYPE_LABEL: Record<ExType, string> = {
  single_choice: '单选',
  multiple_choice: '多选',
  true_false: '判断',
  fill_blank: '填空',
  short_answer: '简答',
}
const TYPE_TAG: Record<ExType, 'primary' | 'warning' | 'success' | 'info' | 'danger'> = {
  single_choice: 'primary',
  multiple_choice: 'warning',
  true_false: 'success',
  fill_blank: 'info',
  short_answer: 'danger',
}
const RESULT_LABEL = { right: '答对', partial: '部分对', wrong: '答错' } as const

function parseOptions(e: KExercise): ExOption[] {
  if (!e.options_json) return []
  try {
    return JSON.parse(e.options_json) as ExOption[]
  } catch {
    return []
  }
}

// ---------- 已保存习题 ----------
const exercises = ref<KExercise[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    exercises.value = await api.get<KExercise[]>(`/api/nodes/${props.nodeId}/exercises`)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.nodeId,
  () => {
    exercises.value = []
    drafts.value = []
    selectedDrafts.value.clear()
    void load()
  },
  { immediate: true },
)

// ---------- 作答状态（本地交互）----------
const selections = ref<Record<string, string[]>>({}) // choice 题已选项
const textAnswers = ref<Record<string, string>>({}) // 填空/简答输入
const showAnalysis = ref<Record<string, boolean>>({})

function toggleOption(exId: string, key: string, multi: boolean) {
  const cur = selections.value[exId] ?? []
  if (multi) {
    selections.value[exId] = cur.includes(key) ? cur.filter((k) => k !== key) : [...cur, key]
  } else {
    selections.value[exId] = [key]
  }
  // 选择即保存草稿
  void api.patch(`/api/exercises/${exId}`, {
    answer_draft: selections.value[exId].join(''),
  })
}

async function submitAnswer(e: KExercise) {
  let answer = ''
  if (e.type === 'single_choice' || e.type === 'multiple_choice') {
    const sel = [...(selections.value[e.id] ?? [])].sort().join('')
    if (!sel) return ElMessage.warning('请先选择答案')
    answer = sel
  } else if (e.type === 'true_false') {
    answer = selections.value[e.id]?.[0] ?? ''
    if (!answer) return ElMessage.warning('请先判断')
  } else {
    answer = (textAnswers.value[e.id] ?? '').trim()
    if (!answer) return ElMessage.warning('请先作答')
  }
  try {
    const updated = await api.post<KExercise>(`/api/exercises/${e.id}/submit`, { answer })
    const i = exercises.value.findIndex((x) => x.id === e.id)
    if (i >= 0) exercises.value[i] = updated
    ElMessage.success(updated.result === 'right' ? '回答正确 🎉' : `批改完成：${RESULT_LABEL[updated.result ?? 'wrong']}`)
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : String(err))
  }
}

async function removeExercise(id: string) {
  try {
    await api.delete(`/api/exercises/${id}`)
    exercises.value = exercises.value.filter((e) => e.id !== id)
  } catch (e2) {
    ElMessage.error(e2 instanceof Error ? e2.message : String(e2))
  }
}

// ---------- 生成 ----------
const genOpen = ref(false)
const genCount = ref(5)
const genDifficulty = ref(3)
const genTypes = ref<ExType[]>(['single_choice', 'true_false', 'short_answer'])
const generating = ref(false)
const drafts = ref<Draft[]>([])
const selectedDrafts = ref(new Set<number>())

async function generate() {
  generating.value = true
  drafts.value = []
  selectedDrafts.value.clear()
  try {
    const r = await api.post<{ items: Draft[] }>('/api/llm/generate-exercises', {
      node_id: props.nodeId,
      count: genCount.value,
      types: genTypes.value,
      difficulty: genDifficulty.value,
    })
    drafts.value = r.items
    r.items.forEach((_, i) => selectedDrafts.value.add(i))
    if (!r.items.length) ElMessage.warning('模型没有生成题目，请重试')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    generating.value = false
  }
}

const savingDrafts = ref(false)
async function saveSelected() {
  const items = drafts.value.filter((_, i) => selectedDrafts.value.has(i))
  if (!items.length) return
  savingDrafts.value = true
  try {
    const created = await api.post<KExercise[]>(`/api/nodes/${props.nodeId}/exercises`, {
      items: items.map((d) => ({
        type: d.type,
        question_md: d.question_md,
        options: d.options,
        answer_md: d.answer_md,
        analysis_md: d.analysis_md ?? null,
        difficulty: d.difficulty ?? null,
      })),
    })
    exercises.value.push(...created)
    drafts.value = []
    ElMessage.success(`已保存 ${created.length} 题`)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    savingDrafts.value = false
  }
}
</script>

<template>
  <div class="ex-panel" v-loading="loading">
    <!-- 生成入口 -->
    <div v-if="!genOpen && !drafts.length" class="gen-entry">
      <el-button type="primary" plain @click="genOpen = !genOpen">🤖 AI 生成练习</el-button>
    </div>

    <!-- 生成参数 -->
    <div v-if="genOpen" class="gen-form">
      <el-select v-model="genTypes" multiple collapse-tags size="small" style="width: 240px" placeholder="题型">
        <el-option v-for="(l, k) in TYPE_LABEL" :key="k" :value="k" :label="l" />
      </el-select>
      <el-input-number v-model="genCount" :min="1" :max="20" size="small" style="width: 100px" />
      <span class="dim">题</span>
      <el-rate v-model="genDifficulty" :max="5" size="small" />
      <el-button type="primary" size="small" :loading="generating" @click="generate">生成</el-button>
      <el-button size="small" @click="genOpen = false">收起</el-button>
    </div>

    <!-- 生成预览 -->
    <div v-if="drafts.length" class="draft-zone">
      <div class="draft-head">
        <el-checkbox
          :model-value="selectedDrafts.size === drafts.length"
          @change="(v: boolean) => { selectedDrafts = new Set(v ? drafts.map((_, i) => i) : []) }"
        >
          全选（{{ selectedDrafts.size }}/{{ drafts.length }}）
        </el-checkbox>
        <el-button type="primary" size="small" :loading="savingDrafts" :disabled="!selectedDrafts.size" @click="saveSelected">
          保存所选
        </el-button>
        <el-button size="small" @click="drafts = []">放弃</el-button>
      </div>
      <div v-for="(d, i) in drafts" :key="i" class="ex-card draft">
        <div class="ex-head">
          <el-checkbox
            :model-value="selectedDrafts.has(i)"
            @change="(v: boolean) => { v ? selectedDrafts.add(i) : selectedDrafts.delete(i); selectedDrafts = new Set(selectedDrafts) }"
          />
          <el-tag :type="TYPE_TAG[d.type as ExType]" size="small">{{ TYPE_LABEL[d.type as ExType] }}</el-tag>
          <span class="diff">{{ '★'.repeat(d.difficulty ?? 3) }}</span>
        </div>
        <MdPreview :model-value="d.question_md" theme="light" preview-theme="github" />
        <ul v-if="d.options?.length" class="opt-preview">
          <li v-for="o in d.options" :key="o.key">{{ o.key }}. {{ o.text }}</li>
        </ul>
        <div class="answer-line"><b>答案：</b>{{ d.answer_md }}</div>
        <div v-if="d.analysis_md" class="analysis"><b>解析：</b>{{ d.analysis_md }}</div>
      </div>
    </div>

    <!-- 已保存列表 -->
    <div v-for="(e, idx) in exercises" :key="e.id" class="ex-card">
      <div class="ex-head">
        <span class="ex-no">#{{ idx + 1 }}</span>
        <el-tag :type="TYPE_TAG[e.type]" size="small">{{ TYPE_LABEL[e.type] }}</el-tag>
        <span class="diff">{{ '★'.repeat(e.difficulty ?? 3) }}</span>
        <el-tag v-if="e.result" :type="e.result === 'right' ? 'success' : e.result === 'partial' ? 'warning' : 'danger'" size="small" effect="dark">
          {{ RESULT_LABEL[e.result] }}{{ e.score != null ? ` ${e.score}` : '' }}
        </el-tag>
        <el-button link type="danger" size="small" style="margin-left: auto" @click="removeExercise(e.id)">删除</el-button>
      </div>

      <MdPreview :model-value="e.question_md" theme="light" preview-theme="github" />

      <!-- 选择题选项 -->
      <div v-if="['single_choice', 'multiple_choice'].includes(e.type)" class="opts">
        <div
          v-for="o in parseOptions(e)"
          :key="o.key"
          class="opt"
          :class="{ picked: (selections[e.id] ?? []).includes(o.key), correct: e.result && o.key === e.answer_md }"
          @click="toggleOption(e.id, o.key, e.type === 'multiple_choice')"
        >
          <b>{{ o.key }}</b>. {{ o.text }}
        </div>
      </div>

      <!-- 判断 -->
      <div v-else-if="e.type === 'true_false'" class="opts">
        <div
          v-for="t in ['正确', '错误']"
          :key="t"
          class="opt"
          :class="{ picked: (selections[e.id] ?? [])[0] === t, correct: e.result && t === e.answer_md }"
          @click="toggleOption(e.id, t, false)"
        >
          {{ t }}
        </div>
      </div>

      <!-- 填空/简答 -->
      <el-input
        v-else
        v-model="textAnswers[e.id]"
        type="textarea"
        :rows="e.type === 'short_answer' ? 3 : 1"
        placeholder="在此作答"
        @blur="void api.patch(`/api/exercises/${e.id}`, { answer_draft: textAnswers[e.id] })"
      />

      <!-- 提交与反馈 -->
      <div class="ex-actions">
        <el-button type="primary" size="small" @click="submitAnswer(e)">
          {{ ['fill_blank', 'short_answer'].includes(e.type) ? '提交批改（AI）' : '提交' }}
        </el-button>
        <el-button size="small" @click="showAnalysis[e.id] = !showAnalysis[e.id]">
          {{ showAnalysis[e.id] ? '隐藏解析' : '显示解析' }}
        </el-button>
        <span v-if="e.answered_at" class="answered-hint">最近作答：{{ new Date((e.answered_at ?? 0) * 1000).toLocaleString() }}</span>
      </div>

      <div v-if="e.feedback_md && e.result !== 'right'" class="feedback" :class="e.result">
        <b>AI 点评：</b><span>{{ e.feedback_md }}</span>
      </div>
      <div v-if="showAnalysis[e.id]" class="analysis-box">
        <MdPreview :model-value="`**参考答案：** ${e.answer_md}\n\n${e.analysis_md ? '**解析：** ' + e.analysis_md : ''}`" theme="light" preview-theme="github" />
      </div>
    </div>

    <el-empty v-if="!loading && !exercises.length && !drafts.length" description="还没有练习题，用 AI 生成一组吧" :image-size="72" />
  </div>
</template>

<style scoped>
.gen-entry {
  margin-bottom: 12px;
}

.gen-form {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #f7f9fc;
  border: 1px dashed #c9d4e6;
  border-radius: 8px;
  padding: 10px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.dim {
  color: #98a2b3;
  font-size: 12px;
}

.draft-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.ex-card {
  border: 1px solid #e4e9f2;
  border-radius: 10px;
  padding: 10px 14px;
  margin-bottom: 12px;
  background: #fff;
}

.ex-card.draft {
  border-color: #bcd4ff;
  background: #fbfdff;
}

.ex-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.ex-no {
  font-weight: 700;
  color: #38445c;
  font-size: 13px;
}

.diff {
  color: #e6a23c;
  font-size: 11px;
  letter-spacing: -1px;
}

.ex-card :deep(.md-editor-preview-wrapper) {
  padding: 4px 0;
}

.opt-preview {
  list-style: none;
  margin: 4px 0;
  padding-left: 6px;
  color: #51607a;
  font-size: 12.5px;
}

.answer-line,
.analysis {
  font-size: 12.5px;
  color: #51607a;
  margin-top: 4px;
}

.opts {
  margin-top: 6px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.opt {
  border: 1px solid #e4e9f2;
  border-radius: 8px;
  padding: 7px 10px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.1s;
}

.opt:hover {
  border-color: #9dc3ff;
}

.opt.picked {
  border-color: #409eff;
  background: #ecf3ff;
}

.opt.correct {
  border-color: #67c23a;
  background: #f0f9eb;
}

.ex-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}

.answered-hint {
  font-size: 11px;
  color: #98a2b3;
  margin-left: auto;
}

.feedback {
  margin-top: 8px;
  border-radius: 8px;
  padding: 8px 12px;
  font-size: 12.5px;
}

.feedback.right {
  background: #f0f9eb;
}

.feedback.partial {
  background: #fdf6ec;
}

.feedback.wrong {
  background: #fef0f0;
}

.analysis-box {
  margin-top: 8px;
  background: #fafbfd;
  border-radius: 8px;
  padding: 4px 12px;
}

.analysis-box :deep(.md-editor-preview-wrapper) {
  padding: 4px 0;
}
</style>
