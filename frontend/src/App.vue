<template>
  <q-layout view="hHh lpR fFf">
    <q-header elevated class="bg-primary text-white">
      <q-toolbar>
        <q-toolbar-title>
          DMR-Nexus Dashboard
        </q-toolbar-title>
        
        <q-btn 
          flat 
          round 
          dense 
          :icon="themeIcon"
          @click="cycleTheme"
        >
          <q-tooltip>{{ themeTitle }}</q-tooltip>
        </q-btn>
      </q-toolbar>
    </q-header>

    <q-page-container>
      <q-page padding>
        <router-view />
      </q-page>
    </q-page-container>
  
    <q-footer>
      <div class="q-pa-md text-center">
        <div>© 2025 DMR Nexus. {{ store.version }}</div>
        <div class="q-mt-xs">Made with ❤️ in Macomb, MI</div>
      </div>
    </q-footer>
  </q-layout>
</template>

<script>
import { computed } from 'vue'
import { useAppStore } from './stores/app'

export default {
  name: 'App',
  setup() {
    const store = useAppStore()
    
    const cycleTheme = () => {
      const themes = ['system', 'light', 'dark']
      const currentIndex = themes.indexOf(store.theme)
      const nextTheme = themes[(currentIndex + 1) % themes.length]
      store.setTheme(nextTheme)
    }
    
    const themeIcon = computed(() => {
      if (store.theme === 'light') return 'light_mode'
      if (store.theme === 'dark') return 'dark_mode'
      return 'brightness_auto'
    })
    
    const themeTitle = computed(() => {
      return `Theme: ${store.theme} (click to cycle)`
    })
    
    return { store, cycleTheme, themeIcon, themeTitle }
  }
}
</script>
