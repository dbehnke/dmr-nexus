<template>
  <div>
    <HeaderNav />
    
    <h2 class="text-2xl font-bold mb-6">Overview</h2>
    
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
      <div class="card p-4">
        <div class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</div>
        <div class="text-2xl font-bold mt-2" :class="app.status === 'running' ? 'text-green-600 dark:text-green-400' : 'text-gray-500'">
          {{ app.status }}
        </div>
      </div>
      
      <div class="card p-4">
        <div class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Connected Peers</div>
        <div class="text-2xl font-bold mt-2">{{ app.peers.length }}</div>
      </div>
      
      <div class="card p-4">
        <div class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Active Bridges</div>
        <div class="text-2xl font-bold mt-2">{{ app.bridges.length + app.dynamicBridges.length }}</div>
      </div>
    </div>

    <!-- Active Bridges Grid -->
    <div class="mb-6">
      <h3 class="text-xl font-bold mb-4">Active Bridges</h3>
      <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2">
        <div
          v-for="bridge in app.dynamicBridges"
          :key="`dynamic-${bridge.tgid}`"
          class="card p-3 border-2"
          :class="bridge.active ? 'border-red-400 bg-red-50 dark:bg-red-900/20' : 'border-green-400 bg-green-50 dark:bg-green-900/20'"
        >
          <div class="flex items-start justify-between gap-2">
            <div class="flex-1 min-w-0">
              <div class="text-xl font-bold">
                {{ bridge.tgid }}
              </div>
              <!-- Always reserve space for transmission info to keep uniform card size -->
              <div class="mt-2 min-h-[60px] text-xs">
                <div v-show="bridge.active && bridge.active_radio_id">
                  <div class="font-bold" :class="bridge.active ? 'text-red-700 dark:text-red-400' : ''">
                    <a :href="`https://radioid.net/database/view?id=${bridge.active_radio_id}`" target="_blank" rel="noopener noreferrer" class="text-red-700 dark:text-red-400 hover:underline">
                      {{ bridge.active_radio_id }}
                    </a>
                  </div>
                  <div v-if="bridge.active_callsign" class="font-medium">
                    <a :href="`https://www.qrz.com/db/${bridge.active_callsign}`" target="_blank" rel="noopener noreferrer" class="text-primary-600 dark:text-primary-400 hover:underline">
                      {{ bridge.active_callsign }}
                    </a>
                  </div>
                  <div v-if="bridge.active_first_name || bridge.active_last_name" class="text-gray-600 dark:text-gray-400">
                    {{ bridge.active_first_name }} {{ bridge.active_last_name }}
                  </div>
                  <div v-if="bridge.active_location" class="text-gray-500 dark:text-gray-500">
                    {{ bridge.active_location }}
                  </div>
                </div>
              </div>
              <div class="text-xs text-gray-600 dark:text-gray-400 mt-2">
                {{ formatSubscribers(bridge.subscribers) }}
              </div>
            </div>
            <div v-if="bridge.active" class="flex-shrink-0">
              <svg class="w-5 h-5 text-red-600 dark:text-red-400" fill="currentColor" viewBox="0 0 20 20" title="Active Transmission">
                <path d="M10 12a2 2 0 100-4 2 2 0 000 4z" />
                <path fill-rule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clip-rule="evenodd" />
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
      <h3 class="text-xl font-bold mb-4">Recent Transmissions</h3>
      <div class="card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead class="bg-gray-50 dark:bg-gray-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Radio ID</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Callsign</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Talkgroup ID</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Timeslot</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Duration</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Time</th>
              </tr>
            </thead>
            <tbody class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              <tr v-for="row in app.transmissions" :key="row.id" class="hover:bg-gray-50 dark:hover:bg-gray-700">
                <td class="px-4 py-3 text-sm">
                  <a :href="`https://radioid.net/database/view?id=${row.radio_id}`" target="_blank" rel="noopener noreferrer" class="text-primary-600 dark:text-primary-400 hover:underline">
                    {{ row.radio_id }}
                  </a>
                </td>
                <td class="px-4 py-3 text-sm">
                  <a v-if="row.callsign" :href="`https://www.qrz.com/db/${row.callsign}`" target="_blank" rel="noopener noreferrer" class="text-primary-600 dark:text-primary-400 hover:underline">
                    {{ row.callsign }}
                  </a>
                  <span v-else class="text-gray-400">-</span>
                </td>
                <td class="px-4 py-3 text-sm">{{ row.talkgroup_id }}</td>
                <td class="px-4 py-3 text-sm">TS{{ row.timeslot }}</td>
                <td class="px-4 py-3 text-sm">{{ formatDuration(row.duration) }}</td>
                <td class="px-4 py-3 text-sm">{{ formatTime(row.start_time) }}</td>
              </tr>
              <tr v-if="app.transmissions.length === 0">
                <td colspan="6" class="px-4 py-8 text-center text-gray-500 dark:text-gray-400">
                  No recent transmissions
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import { useAppStore } from '../stores/app'
import { useWS } from '../composables/useWS'
import { onMounted } from 'vue'

export default {
  name: 'Dashboard',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    const { connected } = useWS() // Use WebSocket for real-time updates
    
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

    const formatSubscribers = (subscribers) => {
      if (!subscribers || subscribers.length === 0) {
        return 'No subscribers'
      }

      // Count subscribers by timeslot
      const ts1Only = subscribers.filter(s => s.timeslot === 1).length
      const ts2Only = subscribers.filter(s => s.timeslot === 2).length
      const both = subscribers.filter(s => s.timeslot === 3).length

      const parts = []
      if (ts1Only > 0) parts.push(`${ts1Only} TS1`)
      if (ts2Only > 0) parts.push(`${ts2Only} TS2`)
      if (both > 0) parts.push(`${both} both`)

      return parts.join(', ')
    }
    
    const fetchData = () => {
      app.fetchStatus().catch(() => {})
      app.fetchPeers().catch(() => {})
      app.fetchBridges().catch(() => {})
      app.fetchTransmissions().catch(() => {})
    }
    
    onMounted(() => {
      // Fetch initial data once - WebSocket will handle updates
      fetchData()
    })
    
    return { app, connected, formatDuration, formatTime, formatSubscribers }
  }
}
</script>
