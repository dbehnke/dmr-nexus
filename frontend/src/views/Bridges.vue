<template>
  <div>
    <HeaderNav />
    
    <h2 class="text-2xl font-bold mb-6">Bridge Configuration</h2>
    
    <!-- Static Bridges Section -->
    <div class="mb-8">
      <h3 class="text-xl font-bold mb-4 text-gray-800 dark:text-gray-200">Static Bridges</h3>
      
      <div v-if="app.bridges.length === 0">
        <div class="card p-6 text-center text-gray-500 dark:text-gray-400">
          No static bridges configured
        </div>
      </div>
      
      <div v-else class="space-y-4">
        <div 
          v-for="bridge in app.bridges" 
          :key="bridge.name"
          class="card p-4"
        >
          <h4 class="text-lg font-bold mb-4">{{ bridge.name }}</h4>
          
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead class="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">System</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">TGID</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Timeslot</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
                </tr>
              </thead>
              <tbody class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                <tr v-for="rule in bridge.rules" :key="rule.system">
                  <td class="px-4 py-2 text-sm">{{ rule.system }}</td>
                  <td class="px-4 py-2 text-sm">{{ rule.tgid }}</td>
                  <td class="px-4 py-2 text-sm">TS{{ rule.timeslot }}</td>
                  <td class="px-4 py-2 text-sm">
                    <span 
                      class="inline-block px-2 py-1 rounded text-xs font-medium"
                      :class="rule.active ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'"
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
      <h3 class="text-xl font-bold mb-4 text-gray-800 dark:text-gray-200">Dynamic Bridges</h3>
      
      <div v-if="app.dynamicBridges.length === 0">
        <div class="card p-6 text-center text-gray-500 dark:text-gray-400">
          No active dynamic bridges
        </div>
      </div>
      
      <div v-else class="space-y-4">
        <div 
          v-for="bridge in app.dynamicBridges" 
          :key="`${bridge.tgid}-${bridge.timeslot}`"
          class="card p-4 border-2 border-gray-200 dark:border-gray-700"
        >
          <div class="flex items-center justify-between mb-4">
            <div>
              <h4 class="text-lg font-bold">TG {{ bridge.tgid }} - TS{{ bridge.timeslot }}</h4>
              <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Active {{ formatTimestamp(bridge.last_activity) }}
              </div>
            </div>
            <div>
              <span class="inline-block px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                Dynamic
              </span>
            </div>
          </div>
          
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">Subscribers:</div>
              <div class="font-bold">{{ bridge.subscribers.length }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">Created:</div>
              <div class="font-bold">{{ formatTimestamp(bridge.created_at) }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">Last Activity:</div>
              <div class="font-bold">{{ formatTimestamp(bridge.last_activity) }}</div>
            </div>
          </div>
          
          <div v-if="bridge.subscribers.length > 0" class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
            <div class="text-xs text-gray-500 dark:text-gray-400 mb-2">Subscriber Peer IDs:</div>
            <div class="flex flex-wrap gap-2">
              <span 
                v-for="peerId in bridge.subscribers" 
                :key="peerId"
                class="inline-block px-2 py-1 rounded bg-gray-100 dark:bg-gray-700 text-sm"
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
