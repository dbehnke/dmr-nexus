<template>
  <div>
    <HeaderNav />
    <h2 class="text-xl font-medium mb-2">Activity</h2>
    <div class="text-sm text-gray-500 mb-2">WS: {{ connected ? 'connected' : 'disconnected' }}</div>
    <div v-if="app.activity.length === 0" class="text-gray-500">No activity</div>
    <ul class="list-disc ml-5">
      <li v-for="(a, i) in app.activity" :key="i">{{ a }}</li>
    </ul>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import { useAppStore } from '../stores/app'
import { useWS } from '../composables/useWS'

export default {
  name: 'Activity',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    app.fetchActivity().catch(() => {})
    const { connected } = useWS()
    return { app, connected }
  }
}
</script>
