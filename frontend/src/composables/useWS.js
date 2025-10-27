import { onMounted, onUnmounted, ref } from 'vue'
import { useAppStore } from '../stores/app'

export function useWS() {
  const connected = ref(false)
  let socket
  const app = useAppStore()

  onMounted(() => {
    try {
      const url = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws'
      socket = new WebSocket(url)
      socket.onopen = () => { connected.value = true }
      socket.onclose = () => { connected.value = false }
      socket.onerror = () => { /* noop */ }
      socket.onmessage = (ev) => {
        try {
          const evt = JSON.parse(ev.data)
          if (evt?.type) {
            // Handle activity events
            if (evt.type === 'peer_connected' || evt.type === 'peer_disconnected' || evt.type === 'heartbeat') {
              app.pushActivity(evt)
            }
            
            // Handle data update events
            if (evt.type === 'status_update' && evt.data) {
              app.status = evt.data.status || 'unknown'
              app.version = evt.data.version || app.version
            }
            if (evt.type === 'peers_update' && evt.data?.peers) {
              app.peers = Array.isArray(evt.data.peers) ? evt.data.peers : []
            }
            if (evt.type === 'bridges_update' && evt.data?.bridges) {
              const bridges = evt.data.bridges
              app.bridges = Array.isArray(bridges.static) ? bridges.static : []
              app.dynamicBridges = Array.isArray(bridges.dynamic) ? bridges.dynamic : []
            }
            if (evt.type === 'transmissions_update' && evt.data?.transmissions) {
              const tx = evt.data.transmissions
              app.transmissions = Array.isArray(tx.transmissions) ? tx.transmissions : []
              app.transmissionsTotal = tx.total || 0
            }
          }
        } catch { /* ignore */}
      }
    } catch {
      connected.value = false
    }
  })

  onUnmounted(() => {
    try { socket && socket.close() } catch { /* ignore */ }
  })

  return { connected }
}
