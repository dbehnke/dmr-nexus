<template>
  <div>
    <HeaderNav />
    
    <div class="text-h5 q-mb-md">Bridge Configuration</div>
    
    <!-- Static Bridges Section -->
    <div class="q-mb-lg">
      <div class="text-h6 q-mb-md text-grey-8">Static Bridges</div>
      
      <div v-if="app.bridges.length === 0">
        <q-card>
          <q-card-section class="text-grey-6">
            No static bridges configured
          </q-card-section>
        </q-card>
      </div>
      
      <div v-else class="q-gutter-md">
        <q-card 
          v-for="bridge in app.bridges" 
          :key="bridge.name"
        >
          <q-card-section>
            <div class="text-h6 q-mb-md">{{ bridge.name }}</div>
            
            <q-table
              :rows="bridge.rules"
              :columns="staticBridgeColumns"
              row-key="system"
              flat
              :rows-per-page-options="[0]"
              hide-pagination
              dense
            >
              <template v-slot:body-cell-status="props">
                <q-td :props="props">
                  <q-badge 
                    :color="props.row.active ? 'positive' : 'grey'"
                    :label="props.row.active ? 'Active' : 'Inactive'"
                  />
                </q-td>
              </template>
            </q-table>
          </q-card-section>
        </q-card>
      </div>
    </div>

    <!-- Dynamic Bridges Section -->
    <div>
      <div class="text-h6 q-mb-md text-grey-8">Dynamic Bridges</div>
      
      <div v-if="app.dynamicBridges.length === 0">
        <q-card>
          <q-card-section class="text-grey-6">
            No active dynamic bridges
          </q-card-section>
        </q-card>
      </div>
      
      <div v-else class="q-gutter-md">
        <q-card 
          v-for="bridge in app.dynamicBridges" 
          :key="`${bridge.tgid}-${bridge.timeslot}`"
          bordered
        >
          <q-card-section>
            <div class="row items-center justify-between q-mb-md">
              <div class="col">
                <div class="text-h6">TG {{ bridge.tgid }} - TS{{ bridge.timeslot }}</div>
                <div class="text-caption text-grey-6 q-mt-xs">
                  Active {{ formatTimestamp(bridge.last_activity) }}
                </div>
              </div>
              <div class="col-auto">
                <q-badge color="info" label="Dynamic" />
              </div>
            </div>
            
            <div class="row q-col-gutter-md">
              <div class="col-12 col-md-4">
                <div class="text-caption text-grey-6">Subscribers:</div>
                <div class="text-weight-bold">{{ bridge.subscribers.length }}</div>
              </div>
              <div class="col-12 col-md-4">
                <div class="text-caption text-grey-6">Created:</div>
                <div class="text-weight-bold">{{ formatTimestamp(bridge.created_at) }}</div>
              </div>
              <div class="col-12 col-md-4">
                <div class="text-caption text-grey-6">Last Activity:</div>
                <div class="text-weight-bold">{{ formatTimestamp(bridge.last_activity) }}</div>
              </div>
            </div>
            
            <div v-if="bridge.subscribers.length > 0" class="q-mt-md q-pt-md" style="border-top: 1px solid rgba(0,0,0,0.12)">
              <div class="text-caption text-grey-6 q-mb-sm">Subscriber Peer IDs:</div>
              <div class="row q-gutter-xs">
                <q-chip 
                  v-for="peerId in bridge.subscribers" 
                  :key="peerId"
                  dense
                  size="sm"
                >
                  {{ peerId }}
                </q-chip>
              </div>
            </div>
          </q-card-section>
        </q-card>
      </div>
    </div>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import { useAppStore } from '../stores/app'
import { onMounted, onUnmounted } from 'vue'

export default {
  name: 'Bridges',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    let intervalId = null
    
    const staticBridgeColumns = [
      {
        name: 'system',
        required: true,
        label: 'System',
        align: 'left',
        field: 'system',
        sortable: true
      },
      {
        name: 'tgid',
        label: 'TGID',
        align: 'left',
        field: 'tgid',
        sortable: true
      },
      {
        name: 'timeslot',
        label: 'Timeslot',
        align: 'left',
        field: row => `TS${row.timeslot}`,
        sortable: true
      },
      {
        name: 'status',
        label: 'Status',
        align: 'left',
        field: 'active'
      }
    ]
    
    const formatTimestamp = (timestamp) => {
      if (!timestamp) return 'Unknown'
      const now = Math.floor(Date.now() / 1000)
      const diff = now - timestamp
      
      if (diff < 60) return `${diff}s ago`
      if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
      if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
      return `${Math.floor(diff / 86400)}d ago`
    }
    
    onMounted(() => {
      app.fetchBridges().catch(() => {})
      // Refresh dynamic bridges every 5 seconds
      intervalId = setInterval(() => {
        app.fetchBridges().catch(() => {})
      }, 5000)
    })
    
    onUnmounted(() => {
      if (intervalId) {
        clearInterval(intervalId)
      }
    })
    
    return { app, staticBridgeColumns, formatTimestamp }
  }
}
</script>
