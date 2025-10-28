<template>
  <nav class="border-b border-gray-200 dark:border-gray-700 mb-6">
    <div class="flex gap-1">
      <router-link 
        to="/" 
        class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
        :class="route.path === '/' ? 'bg-white dark:bg-gray-800 text-primary-600 dark:text-primary-400 border-b-2 border-primary-600' : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100'"
      >
        Dashboard
      </router-link>
      <router-link 
        to="/peers" 
        class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
        :class="route.path.startsWith('/peers') ? 'bg-white dark:bg-gray-800 text-primary-600 dark:text-primary-400 border-b-2 border-primary-600' : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100'"
      >
        Peers
      </router-link>
      <router-link 
        to="/bridges" 
        class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
        :class="route.path.startsWith('/bridges') ? 'bg-white dark:bg-gray-800 text-primary-600 dark:text-primary-400 border-b-2 border-primary-600' : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100'"
      >
        Bridges
      </router-link>
      <router-link 
        to="/activity" 
        class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
        :class="route.path.startsWith('/activity') ? 'bg-white dark:bg-gray-800 text-primary-600 dark:text-primary-400 border-b-2 border-primary-600' : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100'"
      >
        Activity
      </router-link>
      <router-link 
        to="/settings" 
        class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
        :class="route.path.startsWith('/settings') ? 'bg-white dark:bg-gray-800 text-primary-600 dark:text-primary-400 border-b-2 border-primary-600' : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100'"
      >
        Settings
      </router-link>
    </div>
  </nav>
</template>

<script>
import { useRoute } from 'vue-router'

export default {
  name: 'HeaderNav',
  setup() {
    const route = useRoute()
    return { route }
  }
}
</script>
