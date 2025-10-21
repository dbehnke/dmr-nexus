<template>
  <nav style="display:flex; gap:1rem; padding:0.5rem 1rem; border-bottom:1px solid #eee;">
    <router-link to="/">Dashboard</router-link>
    <router-link to="/peers">Peers</router-link>
    <router-link to="/bridges">Bridges</router-link>
    <router-link to="/activity">Activity</router-link>
    <router-link to="/settings">Settings</router-link>
  </nav>
</template>

<script>
export default {
  name: 'HeaderNav'
}
</script>
