<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { VersionInfo } from '@/types'

const version = ref<VersionInfo | null>(null)
const checking = ref(false)

onMounted(async () => {
  try {
    version.value = await api.get<VersionInfo>('/api/version')
  } catch {
    /* 忽略，界面显示未知 */
  }
})

// M5 接入 /api/update/check
async function checkUpdate() {
  checking.value = true
  try {
    await new Promise((r) => setTimeout(r, 600))
    ElMessagePlaceholder('更新检查将在 M5 提供（FR-11）')
  } finally {
    checking.value = false
  }
}

function ElMessagePlaceholder(msg: string) {
  // 避免在 script 顶部引入 element-plus 的副作用顺序问题；此处简单实现
  window.alert(msg)
}
</script>

<template>
  <div class="page">
    <el-card shadow="never" style="margin-bottom: 16px">
      <template #header><span style="font-weight: 600">关于 knowtree</span></template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="当前版本">{{ version?.version ?? '…' }}</el-descriptions-item>
        <el-descriptions-item label="构建时间">{{ version?.build_time ?? '—' }}</el-descriptions-item>
        <el-descriptions-item label="Git Commit" :span="2">
          <code>{{ version?.commit ?? '—' }}</code>
        </el-descriptions-item>
      </el-descriptions>
      <div style="margin-top: 14px; display: flex; gap: 10px">
        <el-button type="primary" :loading="checking" @click="checkUpdate">检查更新</el-button>
        <el-button disabled>自动检查（每日一次）</el-button>
      </div>
    </el-card>

    <el-card shadow="never" style="margin-bottom: 16px">
      <template #header><span style="font-weight: 600">LLM 配置</span></template>
      <el-empty description="M4 提供：API Base URL / API Key / 模型名 / 测试连接（FR-6、FR-7）" />
    </el-card>

    <el-card shadow="never">
      <template #header><span style="font-weight: 600">数据管理</span></template>
      <el-empty description="M5 提供：备份恢复、导入导出 JSON（FR-7、FR-8）" />
    </el-card>
  </div>
</template>
