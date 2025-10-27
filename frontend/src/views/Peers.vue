<template>
  <div>
    <HeaderNav />
    
    <div class="text-h5 q-mb-md">Connected Peers</div>
    
    <div v-if="app.peers.length === 0">
      <q-card>
        <q-card-section class="text-grey-6">
          No peers connected
        </q-card-section>
      </q-card>
    </div>
    
    <div v-else class="q-gutter-md">
      <q-card 
        v-for="peer in app.peers" 
        :key="peer.id"
      >
        <q-card-section>
          <div class="row items-start justify-between">
            <div class="col">
              <div class="text-h6">{{ peer.callsign || `Peer ${peer.id}` }}</div>
              <div class="text-caption text-grey-7">ID: {{ peer.id }} • {{ peer.location || 'Unknown location' }}</div>
              <div class="text-caption text-grey-7 q-mt-xs">{{ peer.address }}</div>
            </div>
            <div class="col-auto text-right">
              <q-badge 
                :color="peer.state === 'connected' ? 'positive' : 'grey'"
                :label="peer.state"
              />
              <div class="text-caption text-grey-6 q-mt-xs">
                ↓ {{ formatBytes(peer.bytes_rx) }} ↑ {{ formatBytes(peer.bytes_tx) }}
              </div>
            </div>
          </div>
          
          <div v-if="peer.ts1 && peer.ts1.length > 0 || peer.ts2 && peer.ts2.length > 0" class="q-mt-sm">
            <div class="text-caption text-grey-7">Subscriptions:</div>
            <div class="row q-gutter-sm q-mt-xs">
              <div v-if="peer.ts1 && peer.ts1.length > 0">
                <span class="text-caption text-grey-6">TS1:</span> 
                <span class="text-caption">{{ peer.ts1.join(', ') }}</span>
              </div>
              <div v-if="peer.ts2 && peer.ts2.length > 0">
                <span class="text-caption text-grey-6">TS2:</span> 
                <span class="text-caption">{{ peer.ts2.join(', ') }}</span>
              </div>
            </div>
          </div>
        </q-card-section>
      </q-card>
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
