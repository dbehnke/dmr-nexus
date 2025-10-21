<template>
  <div>
    <HeaderNav />
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-xl font-medium">Activity Log</h2>
      <div class="flex items-center gap-2">
        <div 
          class="w-2 h-2 rounded-full"
          :class="connected ? 'bg-green-500 dark:bg-green-400' : 'bg-red-500 dark:bg-red-400'"
        ></div>
        <span class="text-sm text-gray-500 dark:text-gray-400">
          {{ connected ? 'Live' : 'Disconnected' }}
        </span>
      </div>
    </div>
    <div v-if="app.activity.length === 0" class="text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-800 p-4 rounded-lg border border-gray-200 dark:border-gray-700">
      No activity yet
    </div>
    <div v-else class="space-y-2">
      <div 
        v-for="(event, i) in app.activity" 
        :key="i"
        class="bg-white dark:bg-gray-800 p-3 rounded-lg border border-gray-200 dark:border-gray-700 text-sm"
      >
        <div class="flex justify-between items-start">
          <div class="flex-1">
            <span 
              class="inline-block px-2 py-1 text-xs rounded mr-2"
              :class="getEventTypeClass(event.type)"
            >
              {{ event.type }}
            </span>
            <span class="text-gray-700 dark:text-gray-300">{{ formatEventData(event) }}</span>
          </div>
          <div class="text-xs text-gray-500 dark:text-gray-400">
            {{ formatTime(event.timestamp) }}
          </div>
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
  name: 'Activity',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    const { connected } = useWS()
    
    onMounted(() => {
      app.fetchActivity().catch(() => {})
    })
    
    const getEventTypeClass = (type) => {
      const classes = {
        'peer_connected': 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200',
        'peer_disconnected': 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200',
        'heartbeat': 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200',
      }
      return classes[type] || 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200'
    }
    
    const formatEventData = (event) => {
      if (event.type === 'peer_connected' && event.data) {
        return `Peer ${event.data.id} (${event.data.callsign || 'Unknown'}) connected from ${event.data.addr}`
      }
      if (event.type === 'peer_disconnected' && event.data) {
        return `Peer ${event.data.id} disconnected`
      }
      if (event.type === 'heartbeat' && event.data) {
        return `System heartbeat (${event.data.clients} WebSocket clients)`
      }
      return JSON.stringify(event.data || {})
    }
    
    const formatTime = (timestamp) => {
      if (!timestamp) return ''
      const date = new Date(timestamp)
      return date.toLocaleTimeString()
    }
    
    return { app, connected, getEventTypeClass, formatEventData, formatTime }
  }
}
</script>
