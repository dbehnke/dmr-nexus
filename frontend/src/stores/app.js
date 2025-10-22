import { defineStore } from 'pinia'
import axios from 'axios'

export const useAppStore = defineStore('app', {
  state: () => ({
    status: 'unknown',
      version: 'dev',
    peers: [],
    bridges: [],
    dynamicBridges: [],
    activity: [],
    transmissions: [],
    transmissionsTotal: 0,
    // Dark mode: 'light', 'dark', or 'system'
    theme: localStorage.getItem('theme') || 'system',
  }),
  getters: {
    isDark(state) {
      if (state.theme === 'system') {
        return window.matchMedia('(prefers-color-scheme: dark)').matches
      }
      return state.theme === 'dark'
    }
  },
  actions: {
    async fetchStatus() {
      const res = await axios.get('/api/status')
      this.status = res.data?.status || 'unknown'
      this.version = res.data?.version || this.version
    },
    async fetchPeers() {
      const res = await axios.get('/api/peers')
      this.peers = Array.isArray(res.data) ? res.data : []
    },
    async fetchBridges() {
      const res = await axios.get('/api/bridges')
      if (res.data) {
        this.bridges = Array.isArray(res.data.static) ? res.data.static : []
        this.dynamicBridges = Array.isArray(res.data.dynamic) ? res.data.dynamic : []
      } else {
        this.bridges = []
        this.dynamicBridges = []
      }
    },
    async fetchActivity() {
      const res = await axios.get('/api/activity')
      this.activity = Array.isArray(res.data) ? res.data : []
    },
    async fetchTransmissions(page = 1, perPage = 50) {
      const res = await axios.get(`/api/transmissions?page=${page}&per_page=${perPage}`)
      if (res.data) {
        this.transmissions = Array.isArray(res.data.transmissions) ? res.data.transmissions : []
        this.transmissionsTotal = res.data.total || 0
      } else {
        this.transmissions = []
        this.transmissionsTotal = 0
      }
    },
    pushActivity(event) {
      this.activity.unshift(event)
      this.activity = this.activity.slice(0, 200)
    },
    setTheme(theme) {
      this.theme = theme
      localStorage.setItem('theme', theme)
      this.applyTheme()
    },
    applyTheme() {
      if (this.isDark) {
        document.documentElement.classList.add('dark')
      } else {
        document.documentElement.classList.remove('dark')
      }
    },
    initTheme() {
      this.applyTheme()
      // Listen for system theme changes
      window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (this.theme === 'system') {
          this.applyTheme()
        }
      })
    }
  }
})
