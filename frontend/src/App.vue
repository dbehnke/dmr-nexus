<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100 transition-colors">
    <header class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
      <div class="max-w-6xl mx-auto px-4 py-4 flex justify-between items-center">
        <h1 class="text-2xl font-semibold">DMR-Nexus Dashboard</h1>
        <button 
          @click="cycleTheme" 
          class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          :title="themeTitle"
        >
          <svg v-if="store.theme === 'light'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
          </svg>
          <svg v-else-if="store.theme === 'dark'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
          </svg>
          <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
        </button>
      </div>
    </header>
    <main class="max-w-6xl mx-auto px-4 py-6">
      <router-view />
    </main>
  
      <footer class="max-w-6xl mx-auto px-4 py-6 text-center text-sm text-gray-500 dark:text-gray-400">
        <div>© 2025 DMR Nexus. {{ store.version }}</div>
        <div class="mt-1">Made with <span aria-hidden="true">❤️</span> in Macomb, MI</div>
      </footer>
  </div>
  
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
    
    const themeTitle = computed(() => {
      return `Theme: ${store.theme} (click to cycle)`
    })
    
    return { store, cycleTheme, themeTitle }
  }
}
</script>
