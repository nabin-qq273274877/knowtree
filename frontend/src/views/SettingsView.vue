<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '@/api/client'
import type { VersionInfo } from '@/types'

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
const updateSource = ref('')

onMounted(async () => {
  try {
    version.value = await api.get<VersionInfo>('/api/version')
  } catch {
    /* 显示未知 */
  }
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

async function loadUpdateSource() {
  const s = await api.get<Record<string, unknown>>('/api/settings')
  if (typeof s['update.base_url'] === 'string') updateSource.value = s['update.base_url'] as string
}
void loadUpdateSource()

async function saveUpdateSource() {
  await api.put('/api/settings', { 'update.base_url': updateSource.value.trim() })
  ElMessage.success('更新源已保存')
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
  <div class="page" style="max-width: 760px">
    <el-card shadow="never" style="margin-bottom: 16px">
      <template #header><span style="font-weight: 600">LLM 配置</span></template>
      <el-form label-width="110px" v-loading="llmLoading">
        <el-form-item label="API Base URL">
          <el-input v-model="llmForm.base_url" placeholder="https://api.deepseek.com/v1（OpenAI 兼容地址）" />
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
          <el-input v-model="llmForm.model" placeholder="deepseek-chat / gpt-4o-mini / qwen-plus …" />
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
      <template #header><span style="font-weight: 600">版本与更新（FR-11）</span></template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="当前版本">{{ version?.version ?? '…' }}</el-descriptions-item>
        <el-descriptions-item label="构建时间">{{ version?.build_time ?? '—' }}</el-descriptions-item>
        <el-descriptions-item label="Git Commit" :span="2"><code>{{ version?.commit ?? '—' }}</code></el-descriptions-item>
      </el-descriptions>

      <div v-if="updateInfo" style="margin-top: 12px">
        <el-alert :type="updateInfo.has_update ? 'success' : 'info'" :closable="false" show-icon>
          <template #title>
            <span v-if="updateInfo.has_update">发现新版本 <b>{{ updateInfo.latest }}</b>（当前 {{ updateInfo.current }}）</span>
            <span v-else>已是最新版本{{ updateInfo.latest ? `：${updateInfo.latest}` : '' }}</span>
          </template>
          <pre v-if="updateInfo.notes" class="notes">{{ updateInfo.notes }}</pre>
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
      <div class="src-row">
        <span class="dim">更新源：</span>
        <el-input v-model="updateSource" size="small" placeholder="默认 GitHub Releases，可填镜像 API 地址" style="width: 340px" />
        <el-button size="small" @click="saveUpdateSource">保存</el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
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

.update-info {
  margin-top: 12px;
}

.notes {
  margin: 6px 0 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  max-height: 200px;
  overflow: auto;
}

.src-row {
  margin-top: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.dim {
  color: #98a2b3;
  font-size: 12px;
}
</style>
