<template>
  <div class="min-h-screen bg-gray-50 text-gray-900">
    <header class="bg-white border-b">
      <div class="max-w-6xl mx-auto px-4 py-4">
        <h1 class="text-2xl font-semibold">DMR-Nexus Dashboard</h1>
      </div>
    </header>
    <main class="max-w-6xl mx-auto px-4 py-6">
      <router-view />
    </main>
  </div>
  
</template>

<script>
export default {
  name: 'App'
}
</script>
