<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'
import type { VersionInfo } from '@/types'

// ---------- 版本 ----------
const version = ref<VersionInfo | null>(null)
const checking = ref(false)

onMounted(async () => {
  try {
    version.value = await api.get<VersionInfo>('/api/version')
  } catch {
    /* 显示未知 */
  }
})

async function checkUpdate() {
  checking.value = true
  try {
    await new Promise((r) => setTimeout(r, 400))
    ElMessage.info('更新检查将在 M5 提供（FR-11）')
  } finally {
    checking.value = false
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
      <template #header><span style="font-weight: 600">关于 knowtree</span></template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="当前版本">{{ version?.version ?? '…' }}</el-descriptions-item>
        <el-descriptions-item label="构建时间">{{ version?.build_time ?? '—' }}</el-descriptions-item>
        <el-descriptions-item label="Git Commit" :span="2"><code>{{ version?.commit ?? '—' }}</code></el-descriptions-item>
      </el-descriptions>
      <div style="margin-top: 14px; display: flex; gap: 10px">
        <el-button type="primary" :loading="checking" @click="checkUpdate">检查更新</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header><span style="font-weight: 600">数据管理</span></template>
      <el-empty description="M5 提供：备份恢复、导入导出 JSON（FR-7、FR-8）" />
    </el-card>
  </div>
</template>
