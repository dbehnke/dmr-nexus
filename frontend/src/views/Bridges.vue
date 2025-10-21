<template>
  <div>
    <HeaderNav />
    <h2 class="text-xl font-medium mb-4">Bridge Configuration</h2>
    
    <!-- Static Bridges Section -->
    <div class="mb-6">
      <h3 class="text-lg font-medium mb-3 text-gray-700 dark:text-gray-300">Static Bridges</h3>
      <div v-if="app.bridges.length === 0" class="text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-800 p-4 rounded-lg border border-gray-200 dark:border-gray-700">
        No static bridges configured
      </div>
      <div v-else class="space-y-4">
        <div 
          v-for="bridge in app.bridges" 
          :key="bridge.name"
          class="bg-white dark:bg-gray-800 p-4 rounded-lg shadow border border-gray-200 dark:border-gray-700"
        >
          <h3 class="font-semibold text-lg mb-3">{{ bridge.name }}</h3>
          <div class="overflow-x-auto">
            <table class="min-w-full text-sm">
              <thead>
                <tr class="border-b border-gray-200 dark:border-gray-700">
                  <th class="text-left py-2 px-3">System</th>
                  <th class="text-left py-2 px-3">TGID</th>
                  <th class="text-left py-2 px-3">Timeslot</th>
                  <th class="text-left py-2 px-3">Status</th>
                </tr>
              </thead>
              <tbody>
                <tr 
                  v-for="(rule, idx) in bridge.rules" 
                  :key="idx"
                  class="border-b border-gray-100 dark:border-gray-700 last:border-0"
                >
                  <td class="py-2 px-3">{{ rule.system }}</td>
                  <td class="py-2 px-3">{{ rule.tgid }}</td>
                  <td class="py-2 px-3">TS{{ rule.timeslot }}</td>
                  <td class="py-2 px-3">
                    <span 
                      class="inline-block px-2 py-1 text-xs rounded-full"
                      :class="rule.active ? 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200' : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'"
                    >
                      {{ rule.active ? 'Active' : 'Inactive' }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Dynamic Bridges Section -->
    <div>
      <h3 class="text-lg font-medium mb-3 text-gray-700 dark:text-gray-300">Dynamic Bridges</h3>
      <div v-if="app.dynamicBridges.length === 0" class="text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-800 p-4 rounded-lg border border-gray-200 dark:border-gray-700">
        No active dynamic bridges
      </div>
      <div v-else class="space-y-4">
        <div 
          v-for="bridge in app.dynamicBridges" 
          :key="`${bridge.tgid}-${bridge.timeslot}`"
          class="bg-white dark:bg-gray-800 p-4 rounded-lg shadow border border-blue-200 dark:border-blue-700"
        >
          <div class="flex items-center justify-between mb-3">
            <div>
              <h3 class="font-semibold text-lg">TG {{ bridge.tgid }} - TS{{ bridge.timeslot }}</h3>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Active {{ formatTimestamp(bridge.last_activity) }}
              </p>
            </div>
            <span class="inline-block px-3 py-1 text-xs font-medium rounded-full bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200">
              Dynamic
            </span>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
            <div>
              <span class="text-gray-500 dark:text-gray-400">Subscribers:</span>
              <span class="ml-2 font-semibold">{{ bridge.subscribers.length }}</span>
            </div>
            <div>
              <span class="text-gray-500 dark:text-gray-400">Created:</span>
              <span class="ml-2 font-semibold">{{ formatTimestamp(bridge.created_at) }}</span>
            </div>
            <div>
              <span class="text-gray-500 dark:text-gray-400">Last Activity:</span>
              <span class="ml-2 font-semibold">{{ formatTimestamp(bridge.last_activity) }}</span>
            </div>
          </div>
          <div v-if="bridge.subscribers.length > 0" class="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700">
            <p class="text-xs text-gray-500 dark:text-gray-400 mb-2">Subscriber Peer IDs:</p>
            <div class="flex flex-wrap gap-2">
              <span 
                v-for="peerId in bridge.subscribers" 
                :key="peerId"
                class="inline-block px-2 py-1 text-xs rounded bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300"
              >
                {{ peerId }}
              </span>
            </div>
          </div>
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
  name: 'Bridges',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    let intervalId = null
    
    const formatTimestamp = (timestamp) => {
      if (!timestamp) return 'Unknown'
      const now = Math.floor(Date.now() / 1000)
      const diff = now - timestamp
      
      if (diff < 60) return `${diff}s ago`
      if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
      if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
      return `${Math.floor(diff / 86400)}d ago`
    }
    
    onMounted(() => {
      app.fetchBridges().catch(() => {})
      // Refresh dynamic bridges every 5 seconds
      intervalId = setInterval(() => {
        app.fetchBridges().catch(() => {})
      }, 5000)
    })
    
    onUnmounted(() => {
      if (intervalId) {
        clearInterval(intervalId)
      }
    })
    
    return { app, formatTimestamp }
  }
}
</script>
