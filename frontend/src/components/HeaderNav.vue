<template>
  <q-tabs
    v-model="tab"
    dense
    class="text-grey-7 q-mb-md"
    active-color="primary"
    indicator-color="primary"
    align="left"
  >
    <q-route-tab name="dashboard" to="/" label="Dashboard" />
    <q-route-tab name="peers" to="/peers" label="Peers" />
    <q-route-tab name="bridges" to="/bridges" label="Bridges" />
    <q-route-tab name="activity" to="/activity" label="Activity" />
    <q-route-tab name="settings" to="/settings" label="Settings" />
  </q-tabs>
</template>

<script>
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'

export default {
  name: 'HeaderNav',
  setup() {
    const route = useRoute()
    const tab = ref('dashboard')
    
    // Update tab based on current route
    watch(() => route.path, (newPath) => {
      if (newPath === '/') tab.value = 'dashboard'
      else if (newPath.startsWith('/peers')) tab.value = 'peers'
      else if (newPath.startsWith('/bridges')) tab.value = 'bridges'
      else if (newPath.startsWith('/activity')) tab.value = 'activity'
      else if (newPath.startsWith('/settings')) tab.value = 'settings'
    }, { immediate: true })
    
    return { tab }
  }
}
</script>
