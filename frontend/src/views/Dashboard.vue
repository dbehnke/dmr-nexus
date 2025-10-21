<template>
  <div>
    <HeaderNav />
    <h2 class="text-xl font-medium mb-4">Overview</h2>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
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

    <!-- Active Bridges Grid -->
    <div class="mb-6">
      <h3 class="text-lg font-medium mb-3">Active Bridges</h3>
      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
        <div
          v-for="bridge in app.dynamicBridges"
          :key="`dynamic-${bridge.tgid}-${bridge.timeslot}`"
          :class="[
            'p-4 rounded-lg shadow border transition-colors duration-200',
            bridge.active
              ? 'bg-red-50 dark:bg-red-900/20 border-red-300 dark:border-red-700'
              : 'bg-green-50 dark:bg-green-900/20 border-green-300 dark:border-green-700'
          ]"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1">
              <div class="text-xl font-bold text-gray-900 dark:text-gray-100">
                {{ bridge.tgid }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                TS{{ bridge.timeslot }}
              </div>
              <div class="text-sm text-gray-600 dark:text-gray-300 mt-2">
                {{ bridge.subscribers.length }} subscriber{{ bridge.subscribers.length !== 1 ? 's' : '' }}
              </div>
            </div>
            <div v-if="bridge.active" class="flex-shrink-0 ml-2">
              <svg class="w-5 h-5 text-red-600 dark:text-red-400 animate-pulse" fill="currentColor" viewBox="0 0 20 20">
                <path d="M10 12a2 2 0 100-4 2 2 0 000 4z"/>
                <path fill-rule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clip-rule="evenodd"/>
              </svg>
            </div>
          </div>
        </div>
      </div>
      <div v-if="app.dynamicBridges.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
        No active bridges
      </div>
    </div>

    <!-- Talk Log -->
    <div>
      <h3 class="text-lg font-medium mb-3">Recent Transmissions</h3>
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead class="bg-gray-50 dark:bg-gray-900">
              <tr>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Radio ID
                </th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Talkgroup ID
                </th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Timeslot
                </th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Duration
                </th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Time
                </th>
              </tr>
            </thead>
            <tbody class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              <tr v-for="tx in app.transmissions" :key="tx.id" class="hover:bg-gray-50 dark:hover:bg-gray-700">
                <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-gray-100">
                  {{ tx.radio_id }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                  {{ tx.talkgroup_id }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-300">
                  TS{{ tx.timeslot }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-300">
                  {{ formatDuration(tx.duration) }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-300">
                  {{ formatTime(tx.start_time) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="app.transmissions.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
          No recent transmissions
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import { useAppStore } from '../stores/app'
import { onMounted, onUnmounted } from 'vue'

export default {
  name: 'Dashboard',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    let refreshInterval = null
    
    const formatDuration = (seconds) => {
      if (seconds < 1) return '<1s'
      if (seconds < 60) return `${seconds.toFixed(1)}s`
      const mins = Math.floor(seconds / 60)
      const secs = Math.floor(seconds % 60)
      return `${mins}m ${secs}s`
    }
    
    const formatTime = (unixTimestamp) => {
      const date = new Date(unixTimestamp * 1000)
      const now = new Date()
      const diff = now - date
      
      // Less than 1 minute
      if (diff < 60000) {
        return 'just now'
      }
      // Less than 1 hour
      if (diff < 3600000) {
        const mins = Math.floor(diff / 60000)
        return `${mins}m ago`
      }
      // Less than 24 hours
      if (diff < 86400000) {
        const hours = Math.floor(diff / 3600000)
        return `${hours}h ago`
      }
      // Use locale time
      return date.toLocaleTimeString()
    }
    
    const fetchData = () => {
      app.fetchStatus().catch(() => {})
      app.fetchPeers().catch(() => {})
      app.fetchBridges().catch(() => {})
      app.fetchTransmissions().catch(() => {})
    }
    
    onMounted(() => {
      fetchData()
      // Refresh every 5 seconds
      refreshInterval = setInterval(fetchData, 5000)
    })
    
    onUnmounted(() => {
      if (refreshInterval) {
        clearInterval(refreshInterval)
      }
    })
    
    return { app, formatDuration, formatTime }
  }
}
</script>
