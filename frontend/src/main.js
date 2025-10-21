import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './index.css'
import { createPinia } from 'pinia'
import { useAppStore } from './stores/app'

const pinia = createPinia()
const app = createApp(App)
app.use(pinia)
app.use(router)
app.mount('#app')

// Initialize theme after app is mounted
const store = useAppStore()
store.initTheme()
