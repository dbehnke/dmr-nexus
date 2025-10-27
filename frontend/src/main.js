import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { createPinia } from 'pinia'
import { useAppStore } from './stores/app'

// Import Quasar
import { Quasar, Dark } from 'quasar'
import '@quasar/extras/material-icons/material-icons.css'
import 'quasar/dist/quasar.css'

const pinia = createPinia()
const app = createApp(App)

app.use(Quasar, {
  plugins: {
    Dark
  },
  config: {
    dark: 'auto'
  }
})

app.use(pinia)
app.use(router)
app.mount('#app')

// Initialize theme after app is mounted
const store = useAppStore()
store.initTheme()
