<template>
  <div>
    <HeaderNav />
    
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-2xl font-bold">Activity Log</h2>
      <div class="flex items-center gap-2">
        <div 
          class="w-2 h-2 rounded-full"
          :class="connected ? 'bg-green-500' : 'bg-red-500'"
        ></div>
        <span class="text-sm text-gray-600 dark:text-gray-400">
          {{ connected ? 'Live' : 'Disconnected' }}
        </span>
      </div>
    </div>
    
    <div v-if="app.activity.length === 0">
      <div class="card p-6 text-center text-gray-500 dark:text-gray-400">
        No activity yet
      </div>
    </div>
    
    <div v-else class="space-y-2">
      <div 
        v-for="(event, i) in app.activity" 
        :key="i"
        class="card p-3"
      >
        <div class="flex items-start justify-between">
          <div class="flex-1">
            <span 
              class="inline-block px-2 py-1 rounded text-xs font-medium mr-2"
              :class="getEventTypeClass(event.type)"
            >
              {{ event.type }}
            </span>
            <span class="text-sm">{{ formatEventData(event) }}</span>
          </div>
          <div>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ formatTime(event.timestamp) }}
            </span>
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
        'peer_connected': 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
        'peer_disconnected': 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
        'heartbeat': 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
      }
      return classes[type] || 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
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
