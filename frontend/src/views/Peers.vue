<template>
  <div style="padding:1rem;">
    <HeaderNav />
    <h2>Peers</h2>
    <div v-if="peers.length === 0">No peers</div>
    <ul>
      <li v-for="(p, i) in peers" :key="i">{{ p }}</li>
    </ul>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import axios from 'axios'

export default {
  name: 'Peers',
  components: { HeaderNav },
  data() {
    return { peers: [] }
  },
  async mounted() {
    try {
      const res = await axios.get('/api/peers')
      this.peers = res.data
    } catch (e) {
      this.peers = []
    }
  }
}
</script>
