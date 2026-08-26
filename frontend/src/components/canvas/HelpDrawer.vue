<script setup lang="ts">
// 帮助抽屉：从左侧打开的使用文档（界面导览 / 节点 / 连线 / 状态 / AI / 数据）
import { computed, ref } from 'vue'
import { STATUS_META } from '@/utils/meta'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const bodyRef = ref<HTMLElement | null>(null)

interface Section {
  id: string
  title: string
}
const sections: Section[] = [
  { id: 'help-quick', title: '快速上手' },
  { id: 'help-ui', title: '界面导览' },
  { id: 'help-node', title: '节点操作' },
  { id: 'help-status', title: '学习状态' },
  { id: 'help-edge', title: '连线' },
  { id: 'help-detail', title: '详情抽屉' },
  { id: 'help-ai', title: 'AI 功能' },
  { id: 'help-data', title: '数据与更新' },
]

function jumpTo(id: string) {
  bodyRef.value?.querySelector(`#${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const statusList = Object.entries(STATUS_META).map(([key, m]) => ({
  key,
  label: m.label,
  color: m.color,
}))
</script>

<template>
  <el-drawer
    v-model="visible"
    direction="ltr"
    :with-header="false"
    size="70%"
    append-to-body
  >
    <div class="help">
      <header class="help__head">
        <h2>知树 · 使用帮助</h2>
        <button class="help__close" title="关闭" @click="visible = false">✕</button>
      </header>

      <nav class="help__nav">
        <button v-for="s in sections" :key="s.id" class="help__chip" @click="jumpTo(s.id)">
          {{ s.title }}
        </button>
      </nav>

      <div ref="bodyRef" class="help__body">
        <!-- 快速上手 -->
        <section :id="sections[0].id">
          <h3>快速上手</h3>
          <ol class="steps">
            <li>
              <b>新建知识点</b>：点底部功能坞「➕ 新建」，或把鼠标悬停在已有节点上，用「＋下级 / ＋同级」快速扩展。
            </li>
            <li>
              <b>组织层级</b>：按学习顺序把知识点组织成树，上层是先学的，下层是后学的。
            </li>
            <li>
              <b>自动排布</b>：点「✨ 排布」，所有节点按学段分区整理成从左到右的树状布局，不会互相遮挡。
            </li>
            <li>
              <b>开始学习</b>：点击任意节点打开右侧详情抽屉，记录正文、批注心得，配合 AI 讲解与习题巩固。
            </li>
          </ol>
        </section>

        <!-- 界面导览 -->
        <section :id="sections[1].id">
          <h3>界面导览</h3>
          <ul class="items">
            <li><b>顶部彩条</b>：学段分区指示。视口进入某个学段的范围时该彩条展开显示名称与节点数；拖动画布时边界随内容自然过渡；最左和最右的学段常驻点亮。</li>
            <li><b>画布中央</b>：知识树。根在左、子级向右一层层展开（脑图式）。滚轮缩放，按住空白处拖动平移。</li>
            <li><b>左下角罗盘</b>：方向键平移画布；中心按钮<b>单击定位当前节点</b>（没有选中时定位离屏幕中心最近的节点），<b>双击全览全部节点</b>；下方胶囊控制缩放。</li>
            <li><b>右下角小地图</b>：整体鸟瞰，可点击/拖拽快速导航。</li>
            <li><b>底部功能坞</b>：新建、排布、撤销重做、统计、帮助。</li>
            <li><b>右上角齿轮</b>：设置（LLM 配置、数据管理、版本更新）。</li>
          </ul>
        </section>

        <!-- 节点操作 -->
        <section :id="sections[2].id">
          <h3>节点操作</h3>
          <ul class="items">
            <li><b>新建</b>：功能坞「➕ 新建」建顶层节点；hover 节点上的「＋下级」「＋同级」分别在其下方、正下方追加。子级默认继承父节点的学段。</li>
            <li><b>移动</b>：按住节点拖拽。<b>左右方向不能拖出自己的学段分区</b>（允许少量越界）；上下方向自由。</li>
            <li><b>删除</b>：hover 节点上的「🗑 删除」。删除父节点会连带删除其全部子孙，可 Ctrl+Z 撤销。</li>
            <li><b>改学段</b>：在详情抽屉里修改学段字段，节点会换到对应分区（颜色随学段变化）。</li>
          </ul>
        </section>

        <!-- 学习状态 -->
        <section :id="sections[3].id">
          <h3>学习状态</h3>
          <p class="muted">每个知识点有六种状态，卡片左上角的色点与统计分布共用同一套配色：</p>
          <div class="status-grid">
            <span v-for="s in statusList" :key="s.key" class="status-item">
              <i class="dot" :style="{ background: s.color }" />{{ s.label }}
            </span>
          </div>
          <p class="muted">在详情抽屉顶部的状态下拉里切换；「📊 统计」可查看总体分布与掌握占比。</p>
        </section>

        <!-- 连线 -->
        <section :id="sections[4].id">
          <h3>连线</h3>
          <ul class="items">
            <li><b>含义</b>：连线表示「学习先后」——上层学完再学下层；一条线也可以只作层级归属。</li>
            <li><b>建立</b>：鼠标悬停到节点边缘出现锚点圆点，按住从一个节点拖到另一个节点。方向必须自上而下；同层之间不允许连线。</li>
            <li><b>删除</b>：双击连线，或单击选中后按 Delete 键。</li>
          </ul>
        </section>

        <!-- 详情抽屉 -->
        <section :id="sections[5].id">
          <h3>详情抽屉</h3>
          <ul class="items">
            <li><b>正文</b>：支持 Markdown 与数学公式（KaTeX），输入即自动保存，不弹提示；编辑按钮左侧显示保存状态。</li>
            <li><b>资源</b>：记录书籍页码、链接等参考资料。</li>
            <li><b>批注心得</b>：随时记下学习感悟。</li>
            <li><b>习题/试卷</b>：让 AI 出题、在线作答、自动批改（需先配置 LLM）。</li>
          </ul>
        </section>

        <!-- AI 功能 -->
        <section :id="sections[6].id">
          <h3>AI 功能</h3>
          <ul class="items">
            <li><b>讲解</b>：详情抽屉里对当前知识点生成通俗讲解，逐字流式输出。</li>
            <li><b>出题与批改</b>：习题/试卷标签页里按难度与数量生成练习题，提交答案后自动判分讲解。</li>
            <li><b>配置</b>：右上角 ⚙ 设置 → LLM 配置。内置 27 家服务商预设（DeepSeek / OpenAI / Kimi / 通义 / Ollama 本地模型等），选好服务商填入 API Key 即可；也支持任意 OpenAI 兼容接口。</li>
          </ul>
        </section>

        <!-- 数据与更新 -->
        <section :id="sections[7].id">
          <h3>数据与更新</h3>
          <ul class="items">
            <li><b>存储位置</b>：本地 SQLite 单文件，位于 exe 同级 <code>data\</code> 目录，数据完全留在本机。</li>
            <li><b>导入导出</b>：设置 → 数据管理，可导出 JSON 备份、跨设备迁移。</li>
            <li><b>备份恢复</b>：支持一键备份数据库文件并恢复。</li>
            <li><b>版本更新</b>：设置 → 关于。当前版本 v0.1.0；打开设置时会自动检查新版本，有新版会显示版本号。</li>
          </ul>
        </section>

        <footer class="help__foot">知树·KnowTree — 个人知识点管理 · 数据自主可控</footer>
      </div>
    </div>
  </el-drawer>
</template>

<style scoped>
.help {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.help__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 12px;
  border-bottom: 1px solid #e7ecf4;
}

.help__head h2 {
  margin: 0;
  font-size: 18px;
  color: #26334d;
}

.help__close {
  border: none;
  background: transparent;
  font-size: 16px;
  color: #8a94a6;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 6px;
}
.help__close:hover {
  background: #f0f3f9;
  color: #26334d;
}

.help__nav {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 0;
  border-bottom: 1px dashed #e7ecf4;
}

.help__chip {
  border: 1px solid #dbe3ef;
  background: #f7f9fd;
  color: #51607a;
  font-size: 12px;
  padding: 4px 12px;
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.15s;
}
.help__chip:hover {
  border-color: #9fc3ff;
  color: #2469f6;
  background: #eff5ff;
}

.help__body {
  flex: 1;
  overflow-y: auto;
  padding-top: 6px;
}

.help__body section {
  scroll-margin-top: 10px;
  padding: 14px 2px;
  border-bottom: 1px solid #f0f3f9;
}

.help__body h3 {
  margin: 0 0 10px;
  font-size: 15px;
  color: #26334d;
}

.steps {
  margin: 0;
  padding-left: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
  line-height: 1.7;
  color: #51607a;
}

.items {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
  line-height: 1.7;
  color: #51607a;
}
.items b {
  color: #33415c;
}

.muted {
  font-size: 12.5px;
  color: #98a2b3;
  margin: 4px 0 10px;
}

.status-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 22px;
  margin-bottom: 10px;
}

.status-item {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: 13px;
  color: #40506c;
}

.dot {
  width: 11px;
  height: 11px;
  border-radius: 50%;
  display: inline-block;
}

code {
  background: #f2f5fa;
  border: 1px solid #e4e9f2;
  border-radius: 4px;
  padding: 1px 6px;
  font-size: 12px;
  color: #47618e;
}

.help__foot {
  padding: 18px 2px 8px;
  text-align: center;
  font-size: 12px;
  color: #b6bfce;
}
</style>
