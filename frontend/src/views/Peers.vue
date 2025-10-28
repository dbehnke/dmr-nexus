<template>
  <div>
    <HeaderNav />
    
    <h2 class="text-2xl font-bold mb-6">Connected Peers</h2>
    
    <div v-if="app.peers.length === 0">
      <div class="card p-6 text-center text-gray-500 dark:text-gray-400">
        No peers connected
      </div>
    </div>
    
    <div v-else class="space-y-4">
      <div 
        v-for="peer in app.peers" 
        :key="peer.id"
        class="card p-4"
      >
        <div class="flex items-start justify-between">
          <div class="flex-1">
            <h3 class="text-lg font-bold">{{ peer.callsign || `Peer ${peer.id}` }}</h3>
            <div class="text-sm text-gray-600 dark:text-gray-400">ID: {{ peer.id }} • {{ peer.location || 'Unknown location' }}</div>
            <div class="text-sm text-gray-600 dark:text-gray-400 mt-1">{{ peer.address }}</div>
          </div>
          <div class="text-right">
            <span 
              class="inline-block px-3 py-1 rounded-full text-sm font-medium"
              :class="peer.state === 'connected' ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'"
            >
              {{ peer.state }}
            </span>
            <div class="text-xs text-gray-500 dark:text-gray-400 mt-2">
              ↓ {{ formatBytes(peer.bytes_rx) }} ↑ {{ formatBytes(peer.bytes_tx) }}
            </div>
          </div>
        </div>
        
        <div v-if="peer.ts1 && peer.ts1.length > 0 || peer.ts2 && peer.ts2.length > 0" class="mt-4">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">Subscriptions:</div>
          <div class="flex gap-4">
            <div v-if="peer.ts1 && peer.ts1.length > 0">
              <span class="text-xs text-gray-500 dark:text-gray-400">TS1:</span> 
              <span class="text-sm">{{ peer.ts1.join(', ') }}</span>
            </div>
            <div v-if="peer.ts2 && peer.ts2.length > 0">
              <span class="text-xs text-gray-500 dark:text-gray-400">TS2:</span> 
              <span class="text-sm">{{ peer.ts2.join(', ') }}</span>
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
import { onMounted } from 'vue'

export default {
  name: 'Peers',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    
    onMounted(() => {
      app.fetchPeers().catch(() => {})
    })
    
    const formatBytes = (bytes) => {
      if (!bytes) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return Math.round(bytes / Math.pow(k, i) * 10) / 10 + ' ' + sizes[i]
    }
    
    return { app, formatBytes }
  }
}
</script>
