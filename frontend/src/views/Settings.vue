<template>
  <div>
    <HeaderNav />
    
    <div class="text-h5 q-mb-md">Settings</div>
    
    <q-card>
      <q-card-section>
        <div class="text-h6 q-mb-md">Appearance</div>
        
        <q-option-group
          v-model="store.theme"
          :options="themeOptions"
          color="primary"
          @update:model-value="store.setTheme"
        >
          <template v-slot:label="opt">
            <div>
              <div class="text-weight-medium">{{ opt.label }}</div>
              <div class="text-caption text-grey-6">{{ opt.caption }}</div>
            </div>
          </template>
        </q-option-group>
      </q-card-section>
    </q-card>
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
