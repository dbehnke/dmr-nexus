<template>
  <div>
    <HeaderNav />
    <h2 class="text-xl font-medium mb-2">Overview</h2>
    <p>Service status: <span class="font-semibold">{{ app.status }}</span></p>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import { useAppStore } from '../stores/app'

export default {
  name: 'Dashboard',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    app.fetchStatus().catch(() => {})
    return { app }
  }
}
</script>
