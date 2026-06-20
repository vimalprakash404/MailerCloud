/**
 * Composable: useCampaignStats
 *
 * Extracts stats tracking, state management, and computed rates.
 * Now refactored to use WebSockets for real-time campaign stats streaming.
 */

import { ref, computed, onUnmounted } from 'vue'
import { fetchCampaignStats } from '../api/client.js'

export function useCampaignStats(pollInterval = 5000) {
  // ── Reactive state ──────────────────────────────────────────
  const campaignId = ref(localStorage.getItem('mailercloud_last_campaign') || '')
  const activeCampaign = ref('')
  const state = ref('idle')              // idle | loading | ready | empty | error
  const stats = ref({ sent: 0, opened: 0, clicked: 0, bounced: 0 })
  const errorMsg = ref('')
  const lastUpdated = ref(null)
  
  // Real-time speed & history
  const throughputSpeed = ref(0)

  
  // Bookmarked campaigns
  const pinnedCampaigns = ref([])
  try {
    pinnedCampaigns.value = JSON.parse(localStorage.getItem('mailercloud_pinned') || '[]')
  } catch (e) {
    pinnedCampaigns.value = []
  }

  // Active theme
  const theme = ref(localStorage.getItem('mailercloud_theme') || 'dark')
  
  let pollTimer = null
  let abortController = null
  const prevTotal = ref(0)
  const prevTime = ref(null)

  // ── Derived rates ───────────────────────────────────────────
  const openRate = computed(() =>
    stats.value.sent > 0 ? +(stats.value.opened / stats.value.sent * 100).toFixed(1) : null
  )
  const clickRate = computed(() =>
    stats.value.sent > 0 ? +(stats.value.clicked / stats.value.sent * 100).toFixed(1) : null
  )
  const bounceRate = computed(() =>
    stats.value.sent > 0 ? +(stats.value.bounced / stats.value.sent * 100).toFixed(1) : null
  )


  const isStale = computed(() => {
    if (!lastUpdated.value) return false
    return Date.now() - lastUpdated.value.getTime() > 15000
  })

  // Timer tick for active timing text
  const now = ref(Date.now())
  const lastUpdatedText = computed(() => {
    if (!lastUpdated.value) return 'Never'
    const secs = Math.max(0, Math.round((now.value - lastUpdated.value.getTime()) / 1000))
    if (secs < 5) return 'Just now'
    if (secs < 60) return `${secs}s ago`
    return `${Math.floor(secs / 60)}m ago`
  })

  const statusClass = computed(() => {
    if (state.value === 'error') return 'status-error'
    if (state.value === 'loading') return 'status-loading'
    if (activeCampaign.value) return 'status-live'
    return 'status-idle'
  })

  const statusText = computed(() => {
    if (state.value === 'error') return 'Disconnected'
    if (state.value === 'loading') return 'Fetching...'
    if (activeCampaign.value) return 'Active (HTTP)'
    return 'Idle'
  })

  // ── Campaign Pinning Actions ────────────────────────────────
  function pinCampaign(id) {
    if (!id || pinnedCampaigns.value.includes(id)) return
    pinnedCampaigns.value.push(id)
    localStorage.setItem('mailercloud_pinned', JSON.stringify(pinnedCampaigns.value))
  }

  // unpinCampaign
  function unpinCampaign(id) {
    pinnedCampaigns.value = pinnedCampaigns.value.filter(c => c !== id)
    localStorage.setItem('mailercloud_pinned', JSON.stringify(pinnedCampaigns.value))
  }

  const isPinned = computed(() => {
    return activeCampaign.value && pinnedCampaigns.value.includes(activeCampaign.value)
  })

  // ── Theme Toggle Actions ────────────────────────────────────
  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
    localStorage.setItem('mailercloud_theme', theme.value)
    applyTheme()
  }

  function applyTheme() {
    if (theme.value === 'light') {
      document.documentElement.classList.add('light')
    } else {
      document.documentElement.classList.remove('light')
    }
  }

  // Initial theme application
  applyTheme()

  // ── HTTP Polling Connection Lifecycle ──────────────────────
  async function fetchStatsHTTP() {
    if (!activeCampaign.value) return

    if (abortController) {
      abortController.abort()
    }
    abortController = new AbortController()

    try {
      const data = await fetchCampaignStats(activeCampaign.value, abortController.signal)
      stats.value = data
      const currentTime = Date.now()
      lastUpdated.value = new Date(currentTime)

      // Calculate event throughput speed
      const currentTotal = data.sent + data.opened + data.clicked + data.bounced
      if (prevTime.value && prevTotal.value > 0 && currentTime > prevTime.value) {
        const elapsedSecs = (currentTime - prevTime.value) / 1000
        const diff = Math.max(0, currentTotal - prevTotal.value)
        throughputSpeed.value = Math.round(diff / elapsedSecs)
      } else {
        throughputSpeed.value = 0
      }

      prevTotal.value = currentTotal
      prevTime.value = currentTime

      const total = data.sent + data.opened + data.clicked + data.bounced
      state.value = total === 0 ? 'empty' : 'ready'
      errorMsg.value = ''


    } catch (err) {
      if (err.name === 'AbortError') return
      state.value = 'error'
      errorMsg.value = err.message || 'Failed to fetch campaign stats'
      throughputSpeed.value = 0
    }
  }

  function startPolling() {
    if (!campaignId.value) return

    stopPolling()
    activeCampaign.value = campaignId.value
    localStorage.setItem('mailercloud_last_campaign', campaignId.value)
    
    // Clear metrics for new campaign
    throughputSpeed.value = 0
    prevTotal.value = 0
    prevTime.value = null
    
    state.value = 'loading'
    
    fetchStatsHTTP().then(() => {
      if (activeCampaign.value) {
        pollTimer = setInterval(fetchStatsHTTP, pollInterval)
      }
    })
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
    if (abortController) {
      abortController.abort()
      abortController = null
    }
    activeCampaign.value = ''
    state.value = 'idle'
    throughputSpeed.value = 0
  }

  // fetchStats for retry action
  function fetchStats() {
    state.value = 'loading'
    fetchStatsHTTP().then(() => {
      if (activeCampaign.value && !pollTimer) {
        pollTimer = setInterval(fetchStatsHTTP, pollInterval)
      }
    })
  }

  // ── Cleanup on unmount ──────────────────────────────────────
  onUnmounted(stopPolling)

  const tickTimer = setInterval(() => { now.value = Date.now() }, 1000)
  onUnmounted(() => clearInterval(tickTimer))

  // ── Public API ──────────────────────────────────────────────
  return {
    // State
    campaignId,
    activeCampaign,
    state,
    stats,
    errorMsg,
    lastUpdated,
    throughputSpeed,
    pinnedCampaigns,
    theme,

    // Computed
    openRate,
    clickRate,
    bounceRate,

    isStale,
    lastUpdatedText,
    statusClass,
    statusText,
    isPinned,

    // Actions
    startPolling,
    stopPolling,
    fetchStats,
    pinCampaign,
    unpinCampaign,
    toggleTheme,
  }
}
