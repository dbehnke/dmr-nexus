<template>
  <div>
    <HeaderNav />
    
    <div class="row items-center justify-between q-mb-md">
      <div class="col-auto">
        <div class="text-h5">Activity Log</div>
      </div>
      <div class="col-auto">
        <div class="row items-center q-gutter-xs">
          <q-icon 
            :name="connected ? 'circle' : 'circle'" 
            :color="connected ? 'positive' : 'negative'"
            size="xs"
          />
          <span class="text-caption text-grey-6">
            {{ connected ? 'Live' : 'Disconnected' }}
          </span>
        </div>
      </div>
    </div>
    
    <div v-if="app.activity.length === 0">
      <q-card>
        <q-card-section class="text-grey-6">
          No activity yet
        </q-card-section>
      </q-card>
    </div>
    
    <div v-else class="q-gutter-sm">
      <q-card 
        v-for="(event, i) in app.activity" 
        :key="i"
        flat
        bordered
      >
        <q-card-section class="q-pa-sm">
          <div class="row items-start justify-between">
            <div class="col">
              <q-badge 
                :color="getEventTypeColor(event.type)"
                :label="event.type"
                class="q-mr-sm"
              />
              <span class="text-body2">{{ formatEventData(event) }}</span>
            </div>
            <div class="col-auto">
              <span class="text-caption text-grey-6">
                {{ formatTime(event.timestamp) }}
              </span>
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
    
    const getEventTypeColor = (type) => {
      const colors = {
        'peer_connected': 'positive',
        'peer_disconnected': 'negative',
        'heartbeat': 'info',
      }
      return colors[type] || 'grey'
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
    
    return { app, connected, getEventTypeColor, formatEventData, formatTime }
  }
}
</script>
