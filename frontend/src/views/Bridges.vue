<template>
  <div style="padding:1rem;">
    <HeaderNav />
    <h2>Bridges</h2>
    <div v-if="bridges.length === 0">No bridges configured</div>
    <ul>
      <li v-for="(b, i) in bridges" :key="i">{{ b }}</li>
    </ul>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import axios from 'axios'

export default {
  name: 'Bridges',
  components: { HeaderNav },
  data() {
    return { bridges: [] }
  },
  async mounted() {
    try {
      const res = await axios.get('/api/bridges')
      this.bridges = res.data
    } catch (e) {
      this.bridges = []
    }
  }
}
</script>
