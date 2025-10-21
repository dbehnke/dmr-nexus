<template>
  <div>
    <HeaderNav />
    <h2 class="text-xl font-medium mb-4">Overview</h2>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div class="bg-white dark:bg-gray-800 p-4 rounded-lg shadow border border-gray-200 dark:border-gray-700">
        <div class="text-sm text-gray-500 dark:text-gray-400">Status</div>
        <div class="text-2xl font-semibold mt-1" :class="app.status === 'running' ? 'text-green-600 dark:text-green-400' : 'text-gray-600 dark:text-gray-400'">
          {{ app.status }}
        </div>
      </div>
      <div class="bg-white dark:bg-gray-800 p-4 rounded-lg shadow border border-gray-200 dark:border-gray-700">
        <div class="text-sm text-gray-500 dark:text-gray-400">Connected Peers</div>
        <div class="text-2xl font-semibold mt-1">{{ app.peers.length }}</div>
      </div>
      <div class="bg-white dark:bg-gray-800 p-4 rounded-lg shadow border border-gray-200 dark:border-gray-700">
        <div class="text-sm text-gray-500 dark:text-gray-400">Active Bridges</div>
        <div class="text-2xl font-semibold mt-1">{{ app.bridges.length + app.dynamicBridges.length }}</div>
      </div>
    </div>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import { useAppStore } from '../stores/app'
import { onMounted } from 'vue'

export default {
  name: 'Dashboard',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    
    onMounted(() => {
      app.fetchStatus().catch(() => {})
      app.fetchPeers().catch(() => {})
      app.fetchBridges().catch(() => {})
    })
    
    return { app }
  }
}
</script>
