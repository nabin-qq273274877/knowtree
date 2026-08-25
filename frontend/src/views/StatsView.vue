<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import { STATUS_META } from '@/utils/meta'
import type { NodeStatus } from '@/types'

interface StatsData {
  total_nodes: number
  mastered_pct: number
  by_status: Record<string, number>
  by_stage: { stage: string; total: number; mastered: number; learning: number }[]
  edge_count: number
  annotation_count: number
  exercise_count: number
}

const stats = ref<StatsData | null>(null)
const loading = ref(true)

onMounted(async () => {
  try {
    stats.value = await api.get<StatsData>('/api/stats')
  } finally {
    loading.value = false
  }
})

const statusOrder: NodeStatus[] = ['mastered', 'partial', 'learning', 'forgotten', 'not_started']
</script>

<template>
  <div class="page" style="max-width: 860px">
    <h2 style="margin-top: 0; color: #1f2b45">📊 学习统计</h2>

    <el-empty v-if="!loading && !stats" description="加载失败" />

    <template v-if="stats">
      <!-- 概览卡片 -->
      <div class="cards">
        <div class="card">
          <div class="card__num">{{ stats.total_nodes }}</div>
          <div class="card__label">知识点总数</div>
        </div>
        <div class="card accent">
          <div class="card__num">{{ stats.mastered_pct }}%</div>
          <div class="card__label">已学会占比</div>
        </div>
        <div class="card">
          <div class="card__num">{{ stats.edge_count }}</div>
          <div class="card__label">关联连线</div>
        </div>
        <div class="card">
          <div class="card__num">{{ stats.annotation_count }}</div>
          <div class="card__label">批注心得</div>
        </div>
        <div class="card">
          <div class="card__num">{{ stats.exercise_count }}</div>
          <div class="card__label">练习题</div>
        </div>
      </div>

      <!-- 状态分布 -->
      <el-card shadow="never" style="margin-bottom: 16px">
        <template #header><span style="font-weight: 600">学习状态分布</span></template>
        <div v-for="s in statusOrder" :key="s" class="bar-row">
          <span class="bar-label" :style="{ color: STATUS_META[s].color }">{{ STATUS_META[s].label }}</span>
          <div class="bar-track">
            <div
              class="bar-fill"
              :style="{
                width: (stats.by_status[s] ?? 0) * 100 / Math.max(stats.total_nodes, 1) + '%',
                background: STATUS_META[s].color,
              }"
            />
          </div>
          <span class="bar-num">{{ stats.by_status[s] ?? 0 }}</span>
        </div>
      </el-card>

      <!-- 分学段 -->
      <el-card shadow="never">
        <template #header><span style="font-weight: 600">分学段进度</span></template>
        <el-table :data="stats.by_stage" size="small">
          <el-table-column prop="stage" label="学段" width="120" />
          <el-table-column prop="total" label="节点数" width="90" />
          <el-table-column prop="mastered" label="已学会" width="90" />
          <el-table-column prop="learning" label="学习中/部分" width="120" />
          <el-table-column label="掌握进度">
            <template #default="{ row }">
              <el-progress
                :percentage="row.total ? Math.round(row.mastered * 100 / row.total) : 0"
                :stroke-width="10"
              />
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!stats.by_stage.length" description="还没有数据，去画布创建知识点吧" :image-size="72" />
      </el-card>
    </template>
  </div>
</template>

<style scoped>
.cards {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.card {
  flex: 1;
  min-width: 130px;
  background: #fff;
  border: 1px solid #e4e9f2;
  border-radius: 12px;
  padding: 16px;
  text-align: center;
}

.card.accent {
  background: linear-gradient(135deg, #e8f5e9, #f1f8e9);
  border-color: #b7dfc0;
}

.card__num {
  font-size: 26px;
  font-weight: 800;
  color: #1f2b45;
}

.card__label {
  margin-top: 4px;
  font-size: 12px;
  color: #8a94a6;
}

.bar-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 0;
}

.bar-label {
  width: 64px;
  text-align: right;
  font-size: 13px;
}

.bar-track {
  flex: 1;
  height: 14px;
  background: #eef1f6;
  border-radius: 7px;
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  border-radius: 7px;
  transition: width 0.3s;
}

.bar-num {
  width: 40px;
  font-size: 13px;
  color: #51607a;
}
</style>
