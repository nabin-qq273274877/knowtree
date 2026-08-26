<script setup lang="ts">
// 设置面板（Dialog 版）：LLM 配置 / 数据管理 / 版本与更新
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '@/api/client'
import type { VersionInfo } from '@/types'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

// ---------- 版本 ----------
const version = ref<VersionInfo | null>(null)
const checking = ref(false)
const applying = ref(false)
const appliedOk = ref(false)

interface UpdateCheckResult {
  current: string
  latest: string
  has_update: boolean
  name: string
  notes: string
  asset_name: string
  asset_exists: boolean
}
const updateInfo = ref<UpdateCheckResult | null>(null)
const applyMsg = ref<{ ok: boolean; text: string } | null>(null)

onMounted(async () => {
  try {
    version.value = await api.get<VersionInfo>('/api/version')
  } catch {
    /* 显示未知 */
  }
})

// 每次打开面板都刷新版本与配置
watch(visible, (v) => {
  if (!v) return
  void (async () => {
    try {
      version.value = await api.get<VersionInfo>('/api/version')
    } catch {
      /* ignore */
    }
    await loadLLM()
    detectProvider(llmForm.value.base_url)
  })()
})

async function checkUpdate() {
  checking.value = true
  updateInfo.value = null
  applyMsg.value = null
  try {
    updateInfo.value = await api.post<UpdateCheckResult>('/api/update/check', {})
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    checking.value = false
  }
}

async function applyUpdate() {
  applying.value = true
  applyMsg.value = null
  try {
    const r = await api.post<{ ok: boolean; message?: string }>('/api/update/apply', {})
    if (r.ok) {
      appliedOk.value = true
      applyMsg.value = { ok: true, text: r.message ?? '更新完成，请重启应用' }
    } else {
      applyMsg.value = { ok: false, text: r.message ?? '更新失败' }
    }
  } catch (e) {
    applyMsg.value = { ok: false, text: e instanceof Error ? e.message : String(e) }
  } finally {
    applying.value = false
  }
}

async function restartApp() {
  try {
    await api.post('/api/update/restart', {})
    applyMsg.value = { ok: true, text: '正在重启，页面将在几秒后自动刷新…' }
    setTimeout(() => window.location.reload(), 2500)
  } catch (e) {
    // 服务已退出也视为正常，稍后刷新
    setTimeout(() => window.location.reload(), 1500)
    ElMessage.warning(e instanceof Error ? e.message : String(e))
  }
}

// ---------- 数据管理 ----------
interface UploadFile {
  raw: File
  name: string
}

function downloadUrl(path: string) {
  const a = document.createElement('a')
  a.href = path
  a.download = ''
  document.body.appendChild(a)
  a.click()
  a.remove()
}

function doExport() {
  downloadUrl('/api/export')
}

function doBackup() {
  downloadUrl('/api/backup')
}

async function onImportPick(f: UploadFile) {
  if (!f.raw) return
  try {
    await ElMessageBox.confirm(
      '导入将覆盖现有全部知识点、连线、资源与批注（LLM 配置保留）。确定继续？',
      '导入确认',
      { type: 'warning', confirmButtonText: '覆盖导入', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  const fd = new FormData()
  fd.append('file', f.raw)
  try {
    const res = await fetch('/api/import', { method: 'POST', body: fd })
    if (!res.ok) throw new Error((await res.json().catch(() => ({})))?.error ?? `HTTP ${res.status}`)
    const r = await res.json()
    ElMessage.success(`导入成功：${r.counts?.nodes ?? 0} 节点 / ${r.counts?.edges ?? 0} 连线`)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  }
}

async function onRestorePick(f: UploadFile) {
  if (!f.raw) return
  try {
    await ElMessageBox.confirm(
      '恢复将用备份完全替换当前数据。确定继续？',
      '恢复确认',
      { type: 'warning', confirmButtonText: '覆盖恢复', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  const fd = new FormData()
  fd.append('file', f.raw)
  try {
    const res = await fetch('/api/restore', { method: 'POST', body: fd })
    if (!res.ok) throw new Error((await res.json().catch(() => ({})))?.error ?? `HTTP ${res.status}`)
    ElMessage.success('恢复成功，页面即将刷新')
    setTimeout(() => window.location.reload(), 1200)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  }
}

// ---------- LLM 配置 ----------
// 常用服务商预设（统一 OpenAI 兼容协议），云端 / 本地分组
interface ProviderPreset {
  key: string
  label: string
  base: string
  models: string
  local?: boolean
}

const PROVIDER_GROUPS: { label: string; items: ProviderPreset[] }[] = [
  {
    label: '云端服务',
    items: [
      { key: 'deepseek', label: 'DeepSeek 深度求索', base: 'https://api.deepseek.com/v1', models: 'deepseek-chat / deepseek-reasoner' },
      { key: 'openai', label: 'OpenAI', base: 'https://api.openai.com/v1', models: 'gpt-4o / gpt-4o-mini / o3-mini' },
      { key: 'moonshot', label: 'Kimi 月之暗面', base: 'https://api.moonshot.cn/v1', models: 'moonshot-v1-8k / kimi-k2-0711-preview' },
      { key: 'qwen', label: '通义千问 DashScope', base: 'https://dashscope.aliyuncs.com/compatible-mode/v1', models: 'qwen-plus / qwen-turbo / qwen-max' },
      { key: 'zhipu', label: '智谱 GLM', base: 'https://open.bigmodel.cn/api/paas/v4', models: 'glm-4-plus / glm-4-air / glm-4-flash' },
      { key: 'doubao', label: '豆包 火山方舟', base: 'https://ark.cn-beijing.volces.com/api/v3', models: 'doubao-pro-32k（或接入点 ep-xxx）' },
      { key: 'hunyuan', label: '腾讯混元', base: 'https://api.hunyuan.cloud.tencent.com/v1', models: 'hunyuan-turbo / hunyuan-standard' },
      { key: 'qianfan', label: '百度千帆 V2', base: 'https://qianfan.baidubce.com/v2', models: 'ernie-4.0-8k / ernie-3.5-8k' },
      { key: 'minimax', label: 'MiniMax', base: 'https://api.minimax.chat/v1', models: 'abab6.5s-chat / abab6.5g-chat' },
      { key: 'lingyi', label: '零一万物', base: 'https://api.lingyiwanwu.com/v1', models: 'yi-large / yi-medium' },
      { key: 'stepfun', label: '阶跃星辰', base: 'https://api.stepfun.com/v1', models: 'step-1-8k / step-2-16k' },
      { key: 'baichuan', label: '百川智能', base: 'https://api.baichuan-ai.com/v1', models: 'Baichuan4 / Baichuan3-Turbo' },
      { key: 'sensecore', label: '商汤日日新', base: 'https://api.sensenova.cn/compatible-mode/v1', models: 'SenseChat-5 / SenseChat-Turbo' },
      { key: 'siliconflow', label: '硅基流动 SiliconFlow', base: 'https://api.siliconflow.cn/v1', models: 'deepseek-ai/DeepSeek-V3 / Qwen/Qwen2.5-72B-Instruct' },
      { key: 'openrouter', label: 'OpenRouter 聚合', base: 'https://openrouter.ai/api/v1', models: 'openai/gpt-4o-mini / deepseek/deepseek-chat …' },
      { key: 'groq', label: 'Groq', base: 'https://api.groq.com/openai/v1', models: 'llama-3.3-70b-versatile / mixtral-8x7b' },
      { key: 'mistral', label: 'Mistral', base: 'https://api.mistral.ai/v1', models: 'mistral-large-latest / mistral-small-latest' },
      { key: 'xai', label: 'xAI Grok', base: 'https://api.x.ai/v1', models: 'grok-2 / grok-beta' },
      { key: 'together', label: 'Together AI', base: 'https://api.together.xyz/v1', models: 'meta-llama/Llama-3.3-70B …' },
      { key: 'fireworks', label: 'Fireworks AI', base: 'https://api.fireworks.ai/inference/v1', models: 'accounts/fireworks/models/…' },
      { key: 'deepinfra', label: 'DeepInfra', base: 'https://api.deepinfra.com/v1/openai', models: 'meta-llama/… / Qwen/…' },
      { key: 'perplexity', label: 'Perplexity', base: 'https://api.perplexity.ai', models: 'llama-3.1-sonar-small-online …' },
    ],
  },
  {
    label: '本地部署',
    items: [
      { key: 'ollama', label: 'Ollama', base: 'http://localhost:11434/v1', models: 'llama3.1 / qwen2.5（Key 可留空）', local: true },
      { key: 'lmstudio', label: 'LM Studio', base: 'http://localhost:1234/v1', models: '已加载的模型名（Key 可留空）', local: true },
      { key: 'vllm', label: 'vLLM', base: 'http://localhost:8000/v1', models: '启动时 --model 指定的模型名', local: true },
      { key: 'localai', label: 'LocalAI', base: 'http://localhost:8080/v1', models: '本地配置的模型名（Key 可留空）', local: true },
      { key: 'jan', label: 'Jan', base: 'http://127.0.0.1:1337/v1', models: '已下载的模型名（Key 可留空）', local: true },
    ],
  },
]

const ALL_PROVIDERS = PROVIDER_GROUPS.flatMap((g) => g.items)

const providerKey = ref('')
const modelHint = ref('')

function applyProvider(key: string) {
  const p = ALL_PROVIDERS.find((x) => x.key === key)
  if (!p) return
  llmForm.value.base_url = p.base
  modelHint.value = p.models
}

function detectProvider(baseUrl: string) {
  const norm = (u: string) => u.trim().replace(/^https?:\/\//, '').replace(/\/$/, '')
  const hit = ALL_PROVIDERS.find((p) => norm(p.base) === norm(baseUrl))
  providerKey.value = hit?.key ?? ''
}

const llmForm = ref({
  base_url: 'https://api.deepseek.com/v1',
  api_key: '',
  model: 'deepseek-chat',
  temperature: 0.7,
  max_tokens: 2048,
})
const llmLoading = ref(false)
const llmSaving = ref(false)
const testing = ref(false)
const testResult = ref<{ ok: boolean; text: string } | null>(null)

async function loadLLM() {
  llmLoading.value = true
  try {
    const s = await api.get<Record<string, unknown>>('/api/settings')
    const g = (k: string, d: string) => (typeof s[k] === 'string' ? (s[k] as string) : d)
    const gn = (k: string, d: number) => (typeof s[k] === 'number' ? (s[k] as number) : d)
    llmForm.value = {
      base_url: g('llm.base_url', 'https://api.deepseek.com/v1'),
      api_key: g('llm.api_key', ''),
      model: g('llm.model', 'deepseek-chat'),
      temperature: gn('llm.temperature', 0.7),
      max_tokens: gn('llm.max_tokens', 2048),
    }
    detectProvider(llmForm.value.base_url)
  } finally {
    llmLoading.value = false
  }
}
void loadLLM()

function llmSettingsPayload() {
  return {
    'llm.base_url': llmForm.value.base_url.trim(),
    'llm.api_key': llmForm.value.api_key.trim(),
    'llm.model': llmForm.value.model.trim(),
    'llm.temperature': Number(llmForm.value.temperature),
    'llm.max_tokens': Math.round(Number(llmForm.value.max_tokens)),
  }
}

async function saveLLM() {
  if (!llmForm.value.base_url.trim() || !llmForm.value.model.trim()) {
    ElMessage.warning('Base URL 与模型名不能为空')
    return
  }
  llmSaving.value = true
  try {
    await api.put('/api/settings', llmSettingsPayload())
    ElMessage.success('LLM 配置已保存')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : String(e))
  } finally {
    llmSaving.value = false
  }
}

async function testLLM() {
  testResult.value = null
  await saveLLM()
  if (!llmForm.value.base_url.trim()) return
  testing.value = true
  try {
    const r = await api.post<{ ok: boolean; model: string; reply: string }>('/api/llm/test', {})
    testResult.value = { ok: true, text: `连接成功 · ${r.model} · 回复「${r.reply.slice(0, 40)}」` }
  } catch (e) {
    testResult.value = { ok: false, text: e instanceof Error ? e.message : String(e) }
  } finally {
    testing.value = false
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="⚙️ 设置"
    width="780px"
    top="4vh"
    append-to-body
    class="settings-dialog"
  >
    <div class="settings-body">
      <el-card shadow="never" style="margin-bottom: 16px">
        <template #header><span style="font-weight: 600">LLM 配置</span></template>
        <el-form label-width="110px" v-loading="llmLoading">
          <el-form-item label="常用服务商">
            <div style="width: 100%">
              <el-select
                v-model="providerKey"
                clearable
                filterable
                placeholder="选择后自动填入地址（也可自定义）"
                style="width: 100%"
                @change="applyProvider"
              >
                <el-option-group v-for="g in PROVIDER_GROUPS" :key="g.label" :label="g.label">
                  <el-option v-for="p in g.items" :key="p.key" :value="p.key" :label="p.label" />
                </el-option-group>
              </el-select>
              <div v-if="modelHint" class="provider-hint">参考模型：{{ modelHint }}</div>
            </div>
          </el-form-item>
          <el-form-item label="API Base URL">
            <el-input v-model="llmForm.base_url" placeholder="https://api.deepseek.com/v1（OpenAI 兼容地址）" @blur="detectProvider(llmForm.base_url)" />
          </el-form-item>
          <el-form-item label="API Key">
            <el-input
              v-model="llmForm.api_key"
              type="password"
              show-password
              placeholder="sk-...（仅存本地数据库）"
            />
          </el-form-item>
          <el-form-item label="模型名">
            <el-input v-model="llmForm.model" :placeholder="modelHint || 'deepseek-chat / gpt-4o-mini / qwen-plus …'" />
          </el-form-item>
          <el-form-item label="Temperature">
            <el-slider v-model="llmForm.temperature" :min="0" :max="2" :step="0.1" style="width: 260px" show-input />
          </el-form-item>
          <el-form-item label="Max Tokens">
            <el-input-number v-model="llmForm.max_tokens" :min="128" :max="32768" :step="128" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="llmSaving" @click="saveLLM">保存设置</el-button>
            <el-button :loading="testing" @click="testLLM">测试连接</el-button>
          </el-form-item>
          <el-form-item v-if="testResult">
            <el-alert
              :type="testResult.ok ? 'success' : 'error'"
              :title="testResult.text"
              :closable="false"
              show-icon
              style="width: 100%"
            />
          </el-form-item>
        </el-form>
      </el-card>

      <el-card shadow="never" style="margin-bottom: 16px">
        <template #header><span style="font-weight: 600">数据管理</span></template>
        <div class="data-grid">
          <div class="data-item">
            <div class="data-title">导出数据</div>
            <div class="data-desc">全量知识点/连线/资源/练习/批注导出为 JSON</div>
            <el-button size="small" @click="doExport">导出 JSON</el-button>
          </div>
          <div class="data-item">
            <div class="data-title">导入数据</div>
            <div class="data-desc">从 knowtree 导出的 JSON 恢复（<b>覆盖现有全部数据</b>）</div>
            <el-upload :show-file-list="false" accept=".json" :auto-upload="false" :on-change="onImportPick">
              <el-button size="small" type="warning" plain>选择 JSON 导入</el-button>
            </el-upload>
          </div>
          <div class="data-item">
            <div class="data-title">备份数据库</div>
            <div class="data-desc">SQLite 一致性快照（VACUUM INTO），可随时恢复</div>
            <el-button size="small" @click="doBackup">下载备份 (.db)</el-button>
          </div>
          <div class="data-item">
            <div class="data-title">从备份恢复</div>
            <div class="data-desc">上传此前下载的 .db 备份（<b>覆盖现有全部数据</b>）</div>
            <el-upload :show-file-list="false" accept=".db" :auto-upload="false" :on-change="onRestorePick">
              <el-button size="small" type="danger" plain>选择 .db 恢复</el-button>
            </el-upload>
          </div>
        </div>
      </el-card>

      <el-card shadow="never">
        <template #header><span style="font-weight: 600">版本与更新</span></template>
        <div class="ver-rows" v-loading="checking">
          <div class="ver-row">
            <span class="ver-label">当前版本</span>
            <span class="ver-val">{{ version?.version ?? '…' }}</span>
          </div>
          <div class="ver-row">
            <span class="ver-label">最新版本</span>
            <span class="ver-val">{{ updateInfo?.latest ?? '点击「检查更新」获取' }}</span>
          </div>
        </div>

        <div v-if="updateInfo && !updateInfo.has_update" style="margin-top: 12px">
          <el-alert type="success" :closable="false" show-icon title="已是最新版本" />
        </div>
        <div v-if="updateInfo?.notes" style="margin-top: 12px">
          <el-alert type="info" :closable="false" show-icon>
            <template #title>新版本说明</template>
            <pre class="notes">{{ updateInfo.notes }}</pre>
          </el-alert>
        </div>
        <div v-if="applyMsg" style="margin-top: 12px">
          <el-alert :type="applyMsg.ok ? 'success' : 'error'" :title="applyMsg.text" :closable="false" show-icon />
        </div>

        <div style="margin-top: 14px; display: flex; gap: 10px; flex-wrap: wrap">
          <el-button type="primary" :loading="checking" @click="checkUpdate">检查更新</el-button>
          <el-button
            v-if="updateInfo?.has_update && updateInfo.asset_exists"
            type="success"
            :loading="applying"
            @click="applyUpdate"
          >一键更新到 {{ updateInfo.latest }}</el-button>
          <el-button v-if="appliedOk" type="warning" @click="restartApp">重启应用</el-button>
        </div>
      </el-card>
    </div>
  </el-dialog>
</template>

<style scoped>
.settings-body {
  max-height: 78vh;
  overflow-y: auto;
  padding-right: 4px;
}

.data-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.data-item {
  border: 1px solid #eef1f6;
  border-radius: 10px;
  padding: 12px 14px;
}

.data-title {
  font-weight: 700;
  font-size: 13.5px;
  color: #26334d;
  margin-bottom: 4px;
}

.data-desc {
  font-size: 12px;
  color: #8a94a6;
  margin-bottom: 10px;
  min-height: 32px;
}

.notes {
  margin: 6px 0 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  max-height: 200px;
  overflow: auto;
}

.provider-hint {
  margin-top: 4px;
  font-size: 12px;
  color: #98a2b3;
  line-height: 1.5;
}

.ver-rows {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ver-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.ver-label {
  width: 72px;
  font-size: 13px;
  color: #8a94a6;
  text-align: right;
}

.ver-val {
  font-size: 15px;
  font-weight: 700;
  color: #1f2b45;
}

.dim {
  color: #98a2b3;
  font-size: 12px;
}
</style>
