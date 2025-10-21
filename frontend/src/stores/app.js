import { defineStore } from 'pinia'
import axios from 'axios'

export const useAppStore = defineStore('app', {
  state: () => ({
    status: 'unknown',
    peers: [],
    bridges: [],
    activity: [],
  }),
  actions: {
    async fetchStatus() {
      const res = await axios.get('/api/status')
      this.status = res.data?.status || 'unknown'
    },
    async fetchPeers() {
      const res = await axios.get('/api/peers')
      this.peers = Array.isArray(res.data) ? res.data : []
    },
    async fetchBridges() {
      const res = await axios.get('/api/bridges')
      this.bridges = Array.isArray(res.data) ? res.data : []
    },
    async fetchActivity() {
      const res = await axios.get('/api/activity')
      this.activity = Array.isArray(res.data) ? res.data : []
    },
    pushActivity(event) {
      this.activity.unshift(event)
      this.activity = this.activity.slice(0, 200)
    }
  }
})
