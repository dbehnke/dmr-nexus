<template>
  <div>
    <HeaderNav />
    <h2 class="text-xl font-medium mb-4">Settings</h2>
    <div class="bg-white dark:bg-gray-800 p-6 rounded-lg shadow border border-gray-200 dark:border-gray-700">
      <div class="mb-6">
        <h3 class="text-lg font-medium mb-4">Appearance</h3>
        <div class="space-y-3">
          <label class="flex items-center gap-3 cursor-pointer">
            <input 
              type="radio" 
              name="theme" 
              value="system" 
              :checked="store.theme === 'system'"
              @change="store.setTheme('system')"
              class="w-4 h-4"
            />
            <div>
              <div class="font-medium">System</div>
              <div class="text-sm text-gray-500 dark:text-gray-400">Use system preference</div>
            </div>
          </label>
          <label class="flex items-center gap-3 cursor-pointer">
            <input 
              type="radio" 
              name="theme" 
              value="light" 
              :checked="store.theme === 'light'"
              @change="store.setTheme('light')"
              class="w-4 h-4"
            />
            <div>
              <div class="font-medium">Light</div>
              <div class="text-sm text-gray-500 dark:text-gray-400">Always use light mode</div>
            </div>
          </label>
          <label class="flex items-center gap-3 cursor-pointer">
            <input 
              type="radio" 
              name="theme" 
              value="dark" 
              :checked="store.theme === 'dark'"
              @change="store.setTheme('dark')"
              class="w-4 h-4"
            />
            <div>
              <div class="font-medium">Dark</div>
              <div class="text-sm text-gray-500 dark:text-gray-400">Always use dark mode</div>
            </div>
          </label>
        </div>
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
    return { store }
  }
}
</script>
