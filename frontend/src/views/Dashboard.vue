<template>
  <div>
    <HeaderNav />
    
    <div class="text-h5 q-mb-md">Overview</div>
    
    <div class="row q-col-gutter-md q-mb-md">
      <div class="col-12 col-md-4">
        <q-card>
          <q-card-section>
            <div class="text-caption text-grey-7">Status</div>
            <div class="text-h6 q-mt-xs" :class="app.status === 'running' ? 'text-positive' : 'text-grey'">
              {{ app.status }}
            </div>
          </q-card-section>
        </q-card>
      </div>
      
      <div class="col-12 col-md-4">
        <q-card>
          <q-card-section>
            <div class="text-caption text-grey-7">Connected Peers</div>
            <div class="text-h6 q-mt-xs">{{ app.peers.length }}</div>
          </q-card-section>
        </q-card>
      </div>
      
      <div class="col-12 col-md-4">
        <q-card>
          <q-card-section>
            <div class="text-caption text-grey-7">Active Bridges</div>
            <div class="text-h6 q-mt-xs">{{ app.bridges.length + app.dynamicBridges.length }}</div>
          </q-card-section>
        </q-card>
      </div>
    </div>

    <!-- Active Bridges Grid -->
    <div class="q-mb-md">
      <div class="text-h6 q-mb-md">Active Bridges</div>
      <div class="row q-col-gutter-sm">
        <div
          v-for="bridge in app.dynamicBridges"
          :key="`dynamic-${bridge.tgid}`"
          class="col-6 col-sm-4 col-md-3 col-lg-2"
        >
          <q-card 
            :class="bridge.active ? 'bg-red-1' : 'bg-green-1'"
            bordered
          >
            <q-card-section class="q-pa-sm">
              <div class="row items-start justify-between no-wrap">
                <div class="col">
                  <div class="text-h6 text-weight-bold">
                    {{ bridge.tgid }}
                  </div>
                  <div v-if="bridge.active && bridge.active_radio_id" class="q-mt-xs">
                    <div class="text-caption text-negative text-weight-bold">
                      <a :href="`https://radioid.net/database/view?id=${bridge.active_radio_id}`" target="_blank" rel="noopener noreferrer" class="text-negative">
                        {{ bridge.active_radio_id }}
                      </a>
                    </div>
                    <div v-if="bridge.active_callsign" class="text-caption text-weight-medium">
                      <a :href="`https://www.qrz.com/db/${bridge.active_callsign}`" target="_blank" rel="noopener noreferrer" class="text-dark">
                        {{ bridge.active_callsign }}
                      </a>
                    </div>
                    <div v-if="bridge.active_first_name || bridge.active_last_name" class="text-caption text-grey-7">
                      {{ bridge.active_first_name }} {{ bridge.active_last_name }}
                    </div>
                    <div v-if="bridge.active_location" class="text-caption text-grey-6">
                      {{ bridge.active_location }}
                    </div>
                  </div>
                  <div class="text-caption q-mt-sm">
                    {{ formatSubscribers(bridge.subscribers) }}
                  </div>
                </div>
                <div v-if="bridge.active" class="col-auto q-ml-xs">
                  <q-icon name="visibility" color="negative" size="20px">
                    <q-tooltip>Active Transmission</q-tooltip>
                  </q-icon>
                </div>
              </div>
            </q-card-section>
          </q-card>
        </div>
      </div>
      <div v-if="app.dynamicBridges.length === 0" class="text-center q-pa-lg text-grey-6">
        No active bridges
      </div>
    </div>

    <!-- Talk Log -->
    <div>
      <div class="text-h6 q-mb-md">Recent Transmissions</div>
      <q-card>
        <q-table
          :rows="app.transmissions"
          :columns="columns"
          row-key="id"
          flat
          :rows-per-page-options="[0]"
          hide-pagination
        >
          <template v-slot:body-cell-radio_id="props">
            <q-td :props="props">
              <a :href="`https://radioid.net/database/view?id=${props.row.radio_id}`" target="_blank" rel="noopener noreferrer" class="text-primary">
                {{ props.row.radio_id }}
              </a>
            </q-td>
          </template>
          
          <template v-slot:body-cell-callsign="props">
            <q-td :props="props">
              <a v-if="props.row.callsign" :href="`https://www.qrz.com/db/${props.row.callsign}`" target="_blank" rel="noopener noreferrer" class="text-primary">
                {{ props.row.callsign }}
              </a>
              <span v-else class="text-grey-5">-</span>
            </q-td>
          </template>
          
          <template v-slot:body-cell-timeslot="props">
            <q-td :props="props">
              TS{{ props.row.timeslot }}
            </q-td>
          </template>
          
          <template v-slot:body-cell-duration="props">
            <q-td :props="props">
              {{ formatDuration(props.row.duration) }}
            </q-td>
          </template>
          
          <template v-slot:body-cell-start_time="props">
            <q-td :props="props">
              {{ formatTime(props.row.start_time) }}
            </q-td>
          </template>
          
          <template v-slot:no-data>
            <div class="full-width row flex-center q-gutter-sm text-grey-6 q-pa-lg">
              <span>No recent transmissions</span>
            </div>
          </template>
        </q-table>
      </q-card>
    </div>
  </div>
</template>

<script>
import HeaderNav from '../components/HeaderNav.vue'
import { useAppStore } from '../stores/app'
import { onMounted, onUnmounted } from 'vue'

export default {
  name: 'Dashboard',
  components: { HeaderNav },
  setup() {
    const app = useAppStore()
    let refreshInterval = null
    
    const columns = [
      {
        name: 'radio_id',
        required: true,
        label: 'Radio ID',
        align: 'left',
        field: 'radio_id',
        sortable: true
      },
      {
        name: 'callsign',
        label: 'Callsign',
        align: 'left',
        field: 'callsign',
        sortable: true
      },
      {
        name: 'talkgroup_id',
        label: 'Talkgroup ID',
        align: 'left',
        field: 'talkgroup_id',
        sortable: true
      },
      {
        name: 'timeslot',
        label: 'Timeslot',
        align: 'left',
        field: 'timeslot',
        sortable: true
      },
      {
        name: 'duration',
        label: 'Duration',
        align: 'left',
        field: 'duration',
        sortable: true
      },
      {
        name: 'start_time',
        label: 'Time',
        align: 'left',
        field: 'start_time',
        sortable: true
      }
    ]
    
    const formatDuration = (seconds) => {
      if (seconds < 1) return '<1s'
      if (seconds < 60) return `${seconds.toFixed(1)}s`
      const mins = Math.floor(seconds / 60)
      const secs = Math.floor(seconds % 60)
      return `${mins}m ${secs}s`
    }

    const formatTime = (unixTimestamp) => {
      const date = new Date(unixTimestamp * 1000)
      const now = new Date()
      const diff = now - date

      // Less than 1 minute
      if (diff < 60000) {
        return 'just now'
      }
      // Less than 1 hour
      if (diff < 3600000) {
        const mins = Math.floor(diff / 60000)
        return `${mins}m ago`
      }
      // Less than 24 hours
      if (diff < 86400000) {
        const hours = Math.floor(diff / 3600000)
        return `${hours}h ago`
      }
      // Use locale time
      return date.toLocaleTimeString()
    }

    const formatSubscribers = (subscribers) => {
      if (!subscribers || subscribers.length === 0) {
        return 'No subscribers'
      }

      // Count subscribers by timeslot
      const ts1Only = subscribers.filter(s => s.timeslot === 1).length
      const ts2Only = subscribers.filter(s => s.timeslot === 2).length
      const both = subscribers.filter(s => s.timeslot === 3).length

      const parts = []
      if (ts1Only > 0) parts.push(`${ts1Only} TS1`)
      if (ts2Only > 0) parts.push(`${ts2Only} TS2`)
      if (both > 0) parts.push(`${both} both`)

      return parts.join(', ')
    }
    
    const fetchData = () => {
      app.fetchStatus().catch(() => {})
      app.fetchPeers().catch(() => {})
      app.fetchBridges().catch(() => {})
      app.fetchTransmissions().catch(() => {})
    }
    
    onMounted(() => {
      fetchData()
      // Refresh every 5 seconds
      refreshInterval = setInterval(fetchData, 5000)
    })
    
    onUnmounted(() => {
      if (refreshInterval) {
        clearInterval(refreshInterval)
      }
    })
    
    return { app, columns, formatDuration, formatTime, formatSubscribers }
  }
}
</script>
