<script setup lang="ts">
import { useRoute } from 'vue-router'
import { computed } from 'vue'

const route = useRoute()
const active = computed(() => route.name as string)
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        🌳 knowtree
        <small>知识点管理 · 幼儿园到大学</small>
      </div>
      <el-menu
        :default-active="active"
        router
        background-color="#1f2430"
        text-color="#aab6cc"
        active-text-color="#ffffff"
        style="border-right: none"
      >
        <el-menu-item index="canvas">
          <span>🗺️ 知识画布</span>
        </el-menu-item>
        <el-menu-item index="settings">
          <span>⚙️ 设置</span>
        </el-menu-item>
      </el-menu>
    </aside>
    <main class="main">
      <router-view />
    </main>
  </div>
</template>
