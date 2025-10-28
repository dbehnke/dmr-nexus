<template>
  <div>
    <HeaderNav />
    
    <h2 class="text-2xl font-bold mb-6">Settings</h2>
    
    <div class="card p-6">
      <h3 class="text-lg font-bold mb-4">Appearance</h3>
      
      <div class="space-y-3">
        <label 
          v-for="option in themeOptions" 
          :key="option.value"
          class="flex items-start gap-3 p-3 rounded-lg border-2 cursor-pointer transition-colors hover:bg-gray-50 dark:hover:bg-gray-700"
          :class="store.theme === option.value ? 'border-primary-600 bg-primary-50 dark:bg-primary-900/20' : 'border-gray-200 dark:border-gray-700'"
        >
          <input 
            type="radio" 
            :value="option.value"
            :checked="store.theme === option.value"
            @change="store.setTheme(option.value)"
            class="mt-1"
          />
          <div class="flex-1">
            <div class="font-medium">{{ option.label }}</div>
            <div class="text-sm text-gray-600 dark:text-gray-400">{{ option.caption }}</div>
          </div>
        </label>
      </div>
    </div>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import { useAppStore } from '../stores/app'

export default {
  name: 'Settings',
  components: { HeaderNav },
  setup() {
    const store = useAppStore()
    
    const themeOptions = [
      { 
        label: 'System', 
        value: 'system',
        caption: 'Use system preference'
      },
      { 
        label: 'Light', 
        value: 'light',
        caption: 'Always use light mode'
      },
      { 
        label: 'Dark', 
        value: 'dark',
        caption: 'Always use dark mode'
      }
    ]
    
    return { store, themeOptions }
  }
}
</script>
