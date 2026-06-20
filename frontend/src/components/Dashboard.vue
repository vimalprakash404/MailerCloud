<template>
  <div class="dashboard">
    <!-- Header -->
    <header class="dash-header">
      <div class="brand">
        <span class="logo">📬</span>
        <div>
          <h1 class="brand-name">MailerCloud</h1>
          <p class="brand-sub">Engagement Analytics</p>
        </div>
      </div>

      <div class="header-right">
        <div class="status-indicator" :class="statusClass">
          <span class="status-dot"></span>
          <span class="status-text">{{ statusText }}</span>
        </div>
      </div>
    </header>

    <!-- Campaign Selector -->
    <section class="campaign-selector glass fade-in-up">
      <div class="selector-row">
        <div class="field">
          <label for="campaign-input">Campaign ID</label>
          <input
            id="campaign-input"
            type="text"
            v-model.trim="campaignId"
            placeholder="e.g. camp-1"
            @keyup.enter="startPolling"
          />
        </div>
        <button id="btn-track" class="btn-primary" @click="startPolling" :disabled="!campaignId">
          📡 Track Campaign
        </button>
      </div>

      <!-- Quick picks -->
      <div class="quick-picks">
        <span class="picks-label">Quick:</span>
        <button
          v-for="id in quickPicks"
          :key="id"
          class="pick-chip"
          :class="{ active: campaignId === id }"
          @click="campaignId = id; startPolling()"
        >
          {{ id }}
        </button>
      </div>
    </section>

    <!-- States -->
    <section v-if="activeCampaign" class="stats-section">
      <!-- Last updated -->
      <div class="meta-bar fade-in-up" style="animation-delay: 0.1s">
        <span class="meta-campaign">
          Campaign: <strong>{{ activeCampaign }}</strong>
        </span>
        <span class="meta-updated" :class="{ stale: isStale }">
          🕐 {{ lastUpdatedText }}
        </span>
        <span class="meta-poll">
          Auto-refresh: <strong>5s</strong>
        </span>
      </div>

      <!-- Loading skeleton -->
      <div v-if="state === 'loading'" class="stats-grid">
        <div v-for="n in 4" :key="n" class="skeleton-card">
          <div class="skeleton" style="width: 48px; height: 48px; border-radius: 12px;"></div>
          <div style="flex: 1;">
            <div class="skeleton" style="width: 80px; height: 14px; margin-bottom: 8px;"></div>
            <div class="skeleton" style="width: 120px; height: 32px;"></div>
          </div>
        </div>
      </div>

      <!-- Error state -->
      <div v-else-if="state === 'error'" class="error-card glass fade-in-up">
        <span class="error-icon">⚠️</span>
        <div>
          <p class="error-title">Failed to fetch stats</p>
          <p class="error-msg">{{ errorMsg }}</p>
        </div>
        <button class="btn-retry" @click="fetchStats">Retry</button>
      </div>

      <!-- Empty state -->
      <div v-else-if="state === 'empty'" class="empty-card glass fade-in-up">
        <span class="empty-icon">📭</span>
        <p class="empty-title">No events recorded yet</p>
        <p class="empty-sub">Use the load generator below to fire some events, or wait for real data to arrive.</p>
      </div>

      <!-- Stats grid -->
      <div v-else class="stats-grid">
        <StatsCard
          label="Sent"
          :value="stats.sent"
          icon="📤"
          color="blue"
          delay="0.15s"
        />
        <StatsCard
          label="Opened"
          :value="stats.opened"
          icon="👁️"
          color="emerald"
          :rate="openRate"
          delay="0.2s"
        />
        <StatsCard
          label="Clicked"
          :value="stats.clicked"
          icon="🖱️"
          color="amber"
          :rate="clickRate"
          delay="0.25s"
        />
        <StatsCard
          label="Bounced"
          :value="stats.bounced"
          icon="🔴"
          color="rose"
          :rate="bounceRate"
          delay="0.3s"
        />
      </div>

      <!-- Load Generator -->
      <LoadGenerator :campaign-id="activeCampaign" />
    </section>

    <!-- No campaign selected -->
    <section v-else class="welcome glass fade-in-up" style="animation-delay: 0.15s">
      <span class="welcome-icon">🚀</span>
      <h2>Enter a Campaign ID to begin</h2>
      <p>The dashboard will auto-refresh every 5 seconds to show live engagement statistics.</p>
    </section>
  </div>
</template>

<script setup>
import { useCampaignStats } from '../composables/useCampaignStats.js'
import StatsCard from './StatsCard.vue'
import LoadGenerator from './LoadGenerator.vue'

const quickPicks = ['camp-1', 'camp-2', 'camp-3']

// All polling, state, and computed rates are provided by the composable.
const {
  campaignId,
  activeCampaign,
  state,
  stats,
  errorMsg,
  lastUpdated,
  openRate,
  clickRate,
  bounceRate,
  isStale,
  lastUpdatedText,
  statusClass,
  statusText,
  startPolling,
  fetchStats,
} = useCampaignStats()
</script>

<style scoped>
.dashboard {
  max-width: 900px;
  margin: 0 auto;
  padding: 32px 20px 64px;
}

/* ── Header ─────────────────────────────────────────────── */
.dash-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 14px;
}

.logo {
  font-size: 36px;
}

.brand-name {
  font-size: 24px;
  font-weight: 800;
  background: linear-gradient(135deg, var(--accent-blue), var(--accent-violet));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.brand-sub {
  font-size: 13px;
  color: var(--text-muted);
  font-weight: 500;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 600;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-live { background: rgba(16, 185, 129, 0.1); color: var(--accent-emerald); }
.status-live .status-dot { background: var(--accent-emerald); animation: pulse-soft 1.5s infinite; }

.status-loading { background: rgba(245, 158, 11, 0.1); color: var(--accent-amber); }
.status-loading .status-dot { background: var(--accent-amber); animation: pulse-soft 0.8s infinite; }

.status-error { background: rgba(244, 63, 94, 0.1); color: var(--accent-rose); }
.status-error .status-dot { background: var(--accent-rose); }

.status-idle { background: rgba(100, 116, 139, 0.1); color: var(--text-muted); }
.status-idle .status-dot { background: var(--text-muted); }

/* ── Campaign Selector ──────────────────────────────────── */
.campaign-selector {
  padding: 24px;
  border-radius: var(--radius-lg);
  margin-bottom: 28px;
}

.selector-row {
  display: flex;
  gap: 16px;
  align-items: flex-end;
  flex-wrap: wrap;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
  min-width: 200px;
}

.field label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.field input {
  padding: 10px 16px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-color);
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 15px;
  font-family: var(--font-mono);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.field input:focus {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 3px var(--accent-blue-glow);
}

.btn-primary {
  padding: 10px 24px;
  border-radius: var(--radius-sm);
  border: none;
  background: linear-gradient(135deg, var(--accent-blue), var(--accent-violet));
  color: #fff;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.3s;
  white-space: nowrap;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 20px var(--accent-blue-glow);
}

.btn-primary:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.quick-picks {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
}

.picks-label {
  font-size: 12px;
  color: var(--text-muted);
  font-weight: 600;
  text-transform: uppercase;
}

.pick-chip {
  padding: 4px 14px;
  border-radius: 14px;
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-secondary);
  font-size: 13px;
  font-family: var(--font-mono);
  cursor: pointer;
  transition: all 0.2s;
}

.pick-chip:hover {
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

.pick-chip.active {
  background: var(--accent-blue-glow);
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

/* ── Meta bar ───────────────────────────────────────────── */
.meta-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 20px;
  font-size: 13px;
  color: var(--text-muted);
}

.meta-campaign strong,
.meta-poll strong {
  color: var(--text-secondary);
}

.meta-updated.stale {
  color: var(--accent-rose);
  font-weight: 600;
}

/* ── Stats grid ─────────────────────────────────────────── */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.skeleton-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 24px;
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
}

/* ── Error state ────────────────────────────────────────── */
.error-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 24px;
  border-radius: var(--radius-lg);
  border-color: rgba(244, 63, 94, 0.2) !important;
}

.error-icon { font-size: 32px; }
.error-title { font-weight: 700; color: var(--accent-rose); margin-bottom: 4px; }
.error-msg { font-size: 13px; color: var(--text-muted); }

.btn-retry {
  margin-left: auto;
  padding: 8px 20px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--accent-rose);
  background: transparent;
  color: var(--accent-rose);
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-retry:hover {
  background: rgba(244, 63, 94, 0.1);
}

/* ── Empty state ────────────────────────────────────────── */
.empty-card {
  text-align: center;
  padding: 48px 24px;
  border-radius: var(--radius-lg);
}

.empty-icon { font-size: 48px; display: block; margin-bottom: 12px; }
.empty-title { font-size: 18px; font-weight: 700; margin-bottom: 8px; }
.empty-sub { font-size: 14px; color: var(--text-muted); max-width: 400px; margin: 0 auto; }

/* ── Welcome state ──────────────────────────────────────── */
.welcome {
  text-align: center;
  padding: 64px 24px;
  border-radius: var(--radius-lg);
}

.welcome-icon { font-size: 56px; display: block; margin-bottom: 16px; }
.welcome h2 { font-size: 20px; font-weight: 700; margin-bottom: 8px; }
.welcome p { font-size: 14px; color: var(--text-muted); max-width: 400px; margin: 0 auto; }
</style>
