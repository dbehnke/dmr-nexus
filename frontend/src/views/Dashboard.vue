<template>
  <div style="padding:1rem;">
    <HeaderNav />
    <h2>Overview</h2>
    <p>Service status: {{ status }}</p>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import axios from 'axios'

export default {
  name: 'Dashboard',
  components: { HeaderNav },
  data() {
    return { status: 'unknown' }
  },
  async mounted() {
    try {
      const res = await axios.get('/api/status')
      this.status = res.data.status
    } catch (e) {
      this.status = 'error'
    }
  }
}
</script>
