<template>
  <nav class="flex gap-4 border-b border-gray-200 dark:border-gray-700 mb-4">
    <router-link 
      class="py-2 px-3 rounded-t transition-colors hover:bg-gray-100 dark:hover:bg-gray-800" 
      active-class="border-b-2 border-blue-500 dark:border-blue-400 font-medium"
      to="/"
    >
      Dashboard
    </router-link>
    <router-link 
      class="py-2 px-3 rounded-t transition-colors hover:bg-gray-100 dark:hover:bg-gray-800" 
      active-class="border-b-2 border-blue-500 dark:border-blue-400 font-medium"
      to="/peers"
    >
      Peers
    </router-link>
    <router-link 
      class="py-2 px-3 rounded-t transition-colors hover:bg-gray-100 dark:hover:bg-gray-800" 
      active-class="border-b-2 border-blue-500 dark:border-blue-400 font-medium"
      to="/bridges"
    >
      Bridges
    </router-link>
    <router-link 
      class="py-2 px-3 rounded-t transition-colors hover:bg-gray-100 dark:hover:bg-gray-800" 
      active-class="border-b-2 border-blue-500 dark:border-blue-400 font-medium"
      to="/activity"
    >
      Activity
    </router-link>
    <router-link 
      class="py-2 px-3 rounded-t transition-colors hover:bg-gray-100 dark:hover:bg-gray-800" 
      active-class="border-b-2 border-blue-500 dark:border-blue-400 font-medium"
      to="/settings"
    >
      Settings
    </router-link>
  </nav>
</template>

<script>
export default {
  name: 'HeaderNav'
}
</script>
