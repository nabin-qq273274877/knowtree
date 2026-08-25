<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { Promotion } from '@element-plus/icons-vue'
import { MdPreview } from 'md-editor-v3'
import { streamSSE } from '@/api/sse'

const props = defineProps<{ nodeId: string }>()

interface ChatMsg {
  role: 'user' | 'assistant'
  content: string
}

const msgs = ref<ChatMsg[]>([])
const busy = ref(false)
const input = ref('')
const listEl = ref<HTMLElement | null>(null)

watch(
  () => props.nodeId,
  () => {
    msgs.value = []
    input.value = ''
  },
)

function scrollBottom() {
  void nextTick(() => {
    listEl.value?.scrollTo({ top: listEl.value.scrollHeight })
  })
}

async function send(preset?: string) {
  const question = (preset ?? input.value).trim()
  if (!question || busy.value) return
  input.value = ''
  msgs.value.push({ role: 'user', content: question })
  const asst: ChatMsg = { role: 'assistant', content: '' }
  msgs.value.push(asst)
  busy.value = true
  scrollBottom()
  try {
    // 历史不含刚 push 的两条占位（后端会拼当前问题）
    const history = msgs.value.slice(0, -2).slice(-12)
    for await (const ev of streamSSE('/api/llm/explain', {
      node_id: props.nodeId,
      question,
      history,
    })) {
      if (ev.error) throw new Error(ev.error)
      if (ev.delta) {
        asst.content += ev.delta
        scrollBottom()
      }
    }
    if (!asst.content) asst.content = '（模型没有返回内容）'
  } catch (e) {
    asst.content += `\n\n> ⚠️ ${e instanceof Error ? e.message : String(e)}`
  } finally {
    busy.value = false
    scrollBottom()
  }
}
</script>

<template>
  <div class="explain">
    <div v-if="!msgs.length" class="explain__empty">
      <el-button type="primary" :disabled="busy" @click="send('请系统讲解这个知识点。')">
        🤖 开始讲解
      </el-button>
      <div class="explain__tips">讲解会结合正文、前置知识与你的批注</div>
    </div>

    <div v-show="msgs.length" ref="listEl" class="explain__list">
      <template v-for="(m, i) in msgs" :key="i">
        <div v-if="m.role === 'user'" class="bubble user">{{ m.content }}</div>
        <div v-else class="bubble ai">
          <MdPreview :model-value="m.content" theme="light" preview-theme="github" />
          <span v-if="busy && i === msgs.length - 1" class="cursor">▌</span>
        </div>
      </template>
    </div>

    <div class="explain__input">
      <el-input
        v-model="input"
        type="textarea"
        :rows="2"
        placeholder="追问一句，如：「给我出个例子」/「我还是不懂通分」（Ctrl+Enter 发送）"
        @keydown.ctrl.enter.prevent="send()"
      />
      <el-button
        type="primary"
        :icon="Promotion"
        :loading="busy"
        :disabled="!input.trim() && !msgs.length"
        @click="send()"
      >
        发送
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.explain {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 260px);
  min-height: 320px;
}

.explain__empty {
  text-align: center;
  padding: 40px 0;
}

.explain__tips {
  margin-top: 10px;
  color: #98a2b3;
  font-size: 12px;
}

.explain__list {
  flex: 1;
  overflow: auto;
  padding-right: 4px;
}

.bubble.user {
  background: #ecf3ff;
  border-radius: 10px;
  padding: 8px 12px;
  margin: 8px 0;
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-all;
}

.bubble.ai {
  margin: 8px 0;
  font-size: 13px;
}

.bubble.ai :deep(.md-editor-preview-wrapper) {
  padding: 0;
}

.cursor {
  color: #409eff;
  animation: blink 1s infinite;
}

@keyframes blink {
  50% {
    opacity: 0;
  }
}

.explain__input {
  margin-top: 10px;
  display: flex;
  gap: 8px;
  align-items: flex-end;
  flex-shrink: 0;
}
</style>
