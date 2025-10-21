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
            app.pushActivity(evt)
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
