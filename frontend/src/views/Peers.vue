<template>
  <div>
    <HeaderNav />
    <h2 class="text-xl font-medium mb-4">Connected Peers</h2>
    <div v-if="app.peers.length === 0" class="text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-800 p-4 rounded-lg border border-gray-200 dark:border-gray-700">
      No peers connected
    </div>
    <div v-else class="space-y-3">
      <div 
        v-for="peer in app.peers" 
        :key="peer.id"
        class="bg-white dark:bg-gray-800 p-4 rounded-lg shadow border border-gray-200 dark:border-gray-700"
      >
        <div class="flex justify-between items-start">
          <div>
            <div class="font-semibold text-lg">{{ peer.callsign || `Peer ${peer.id}` }}</div>
            <div class="text-sm text-gray-500 dark:text-gray-400">ID: {{ peer.id }} • {{ peer.location || 'Unknown location' }}</div>
            <div class="text-sm text-gray-500 dark:text-gray-400 mt-1">{{ peer.address }}</div>
          </div>
          <div class="text-right">
            <span 
              class="inline-block px-2 py-1 text-xs rounded-full"
              :class="peer.state === 'connected' ? 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200' : 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200'"
            >
              {{ peer.state }}
            </span>
            <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              ↓ {{ formatBytes(peer.bytes_rx) }} ↑ {{ formatBytes(peer.bytes_tx) }}
            </div>
          </div>
        </div>
        <div v-if="peer.ts1 && peer.ts1.length > 0 || peer.ts2 && peer.ts2.length > 0" class="mt-2 text-sm">
          <div class="text-gray-500 dark:text-gray-400">Subscriptions:</div>
          <div class="flex gap-2 mt-1">
            <div v-if="peer.ts1 && peer.ts1.length > 0">
              <span class="text-xs text-gray-500 dark:text-gray-400">TS1:</span> 
              <span class="text-xs">{{ peer.ts1.join(', ') }}</span>
            </div>
            <div v-if="peer.ts2 && peer.ts2.length > 0">
              <span class="text-xs text-gray-500 dark:text-gray-400">TS2:</span> 
              <span class="text-xs">{{ peer.ts2.join(', ') }}</span>
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
