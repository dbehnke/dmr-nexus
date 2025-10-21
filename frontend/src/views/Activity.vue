<template>
  <div style="padding:1rem;">
    <HeaderNav />
    <h2>Activity</h2>
    <div v-if="activity.length === 0">No activity</div>
    <ul>
      <li v-for="(a, i) in activity" :key="i">{{ a }}</li>
    </ul>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import axios from 'axios'

export default {
  name: 'Activity',
  components: { HeaderNav },
  data() {
    return { activity: [] }
  },
  async mounted() {
    try {
      const res = await axios.get('/api/activity')
      this.activity = res.data
    } catch (e) {
      this.activity = []
    }
  }
}
</script>
