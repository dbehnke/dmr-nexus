<template>
  <div>
    <HeaderNav />
    <h2 class="text-xl font-medium mb-2">Bridges</h2>
    <div v-if="app.bridges.length === 0" class="text-gray-500">No bridges configured</div>
    <ul class="list-disc ml-5">
      <li v-for="(b, i) in app.bridges" :key="i">{{ b }}</li>
    </ul>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import { useAppStore } from '../stores/app'

export default {
  name: 'Bridges',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    app.fetchBridges().catch(() => {})
    return { app }
  }
}
</script>
