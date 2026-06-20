/**
 * Composable: useCampaignStats
 *
 * Extracts polling logic, state management, and computed rates from
 * Dashboard.vue into a reusable composable (Vue 3 Composition API pattern).
 *
 * This enables:
 * - Reuse across different views (e.g., a mini-widget or a detail page)
 * - Unit testing the polling logic without mounting a component
 * - Cleaner Dashboard.vue (~10 lines of script instead of ~100)
 */

import { ref, computed, onUnmounted } from 'vue'
import { fetchCampaignStats } from '../api/client.js'

export function useCampaignStats(pollInterval = 5000) {
  // ── Reactive state ──────────────────────────────────────────
  const campaignId = ref('')
  const activeCampaign = ref('')
  const state = ref('idle')              // idle | loading | ready | empty | error
  const stats = ref({ sent: 0, opened: 0, clicked: 0, bounced: 0 })
  const errorMsg = ref('')
  const lastUpdated = ref(null)

  let pollTimer = null
  let abortController = null

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

  const lastUpdatedText = computed(() => {
    if (!lastUpdated.value) return 'Never'
    const secs = Math.round((Date.now() - lastUpdated.value.getTime()) / 1000)
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
    if (state.value === 'error') return 'Error'
    if (state.value === 'loading') return 'Loading...'
    if (activeCampaign.value) return 'Live'
    return 'Idle'
  })

  // ── Polling controls ────────────────────────────────────────
  function startPolling() {
    if (!campaignId.value) return

    stopPolling()
    activeCampaign.value = campaignId.value
    state.value = 'loading'
    fetchStats()
    pollTimer = setInterval(fetchStats, pollInterval)
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
  }

  async function fetchStats() {
    // Cancel any in-flight request (see DESIGN.md §6)
    if (abortController) abortController.abort()
    abortController = new AbortController()

    try {
      const data = await fetchCampaignStats(activeCampaign.value, abortController.signal)
      stats.value = data
      lastUpdated.value = new Date()

      const total = data.sent + data.opened + data.clicked + data.bounced
      state.value = total === 0 ? 'empty' : 'ready'
      errorMsg.value = ''
    } catch (err) {
      if (err.name === 'AbortError') return // intentional cancel
      state.value = 'error'
      errorMsg.value = err.message || 'Network error'
    }
  }

  // ── Cleanup on unmount ──────────────────────────────────────
  onUnmounted(stopPolling)

  // Tick timer to keep "last updated" text fresh
  const tickTimer = setInterval(() => { lastUpdated.value = lastUpdated.value }, 1000)
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

    // Computed
    openRate,
    clickRate,
    bounceRate,
    isStale,
    lastUpdatedText,
    statusClass,
    statusText,

    // Actions
    startPolling,
    stopPolling,
    fetchStats,
  }
}
