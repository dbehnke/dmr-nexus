<template>
  <nav class="flex gap-4 border-b mb-4">
    <router-link class="py-2" to="/">Dashboard</router-link>
    <router-link class="py-2" to="/peers">Peers</router-link>
    <router-link class="py-2" to="/bridges">Bridges</router-link>
    <router-link class="py-2" to="/activity">Activity</router-link>
    <router-link class="py-2" to="/settings">Settings</router-link>
  </nav>
</template>

<script>
export default {
  name: 'HeaderNav'
}
</script>
