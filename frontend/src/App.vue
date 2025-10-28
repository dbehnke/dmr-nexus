<template>
  <div class="min-h-screen flex flex-col">
    <!-- Header -->
    <header class="bg-primary-600 dark:bg-primary-700 text-white shadow-lg">
      <div class="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
        <h1 class="text-xl font-bold">DMR-Nexus Dashboard</h1>
        
        <button 
          @click="cycleTheme"
          class="p-2 rounded-lg hover:bg-primary-700 dark:hover:bg-primary-600 transition-colors"
          :title="themeTitle"
        >
          <svg v-if="store.theme === 'light'" class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
          </svg>
          <svg v-else-if="store.theme === 'dark'" class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
          </svg>
          <svg v-else class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
          </svg>
        </button>
      </div>
    </header>

    <!-- Main Content -->
    <main class="flex-1 max-w-7xl mx-auto w-full px-4 py-6">
      <router-view />
    </main>

    <!-- Footer -->
    <footer class="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 py-4 mt-8">
      <div class="max-w-7xl mx-auto px-4 text-center text-sm text-gray-600 dark:text-gray-400">
        <div>© 2025 DMR Nexus. {{ store.version }}</div>
        <div class="mt-1">Made with ❤️ in Macomb, MI</div>
      </div>
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
