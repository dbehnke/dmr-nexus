import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import Peers from '../views/Peers.vue'
import Bridges from '../views/Bridges.vue'
import Activity from '../views/Activity.vue'
import Settings from '../views/Settings.vue'

const routes = [
  { path: '/', name: 'Dashboard', component: Dashboard },
  { path: '/peers', name: 'Peers', component: Peers },
  { path: '/bridges', name: 'Bridges', component: Bridges },
  { path: '/activity', name: 'Activity', component: Activity },
  { path: '/settings', name: 'Settings', component: Settings },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
