<template>
  <div>
    <HeaderNav />
    <h2 class="text-xl font-medium mb-2">Peers</h2>
    <div v-if="app.peers.length === 0" class="text-gray-500">No peers</div>
    <ul class="list-disc ml-5">
      <li v-for="(p, i) in app.peers" :key="i">{{ p }}</li>
    </ul>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import { useAppStore } from '../stores/app'

export default {
  name: 'Peers',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    app.fetchPeers().catch(() => {})
    return { app }
  }
}
</script>
