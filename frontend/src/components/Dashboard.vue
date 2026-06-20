<template>
  <div class="dashboard">
    <!-- Header -->
    <header class="dash-header">
      <div class="brand">
        <!-- Clean Envelope SVG -->
        <svg class="brand-svg-logo" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
          <polyline points="22,6 12,13 2,6"/>
        </svg>
        <div>
          <h1 class="brand-name">MailerCloud</h1>
          <p class="brand-sub">Real-Time Engagement Analytics</p>
        </div>
      </div>

      <!-- Action items -->
      <div class="header-actions">
        <!-- Live Speed (Lightning bolt) -->
        <div v-if="throughputSpeed > 0" class="speed-badge fade-in">
          <svg class="action-icon-svg text-amber" viewBox="0 0 24 24" fill="currentColor">
            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
          </svg>
          <span class="speed-value">{{ throughputSpeed.toLocaleString() }} ev/s</span>
        </div>

        <!-- Connection status -->
        <div class="status-indicator" :class="statusClass">
          <span class="status-dot"></span>
          <span class="status-text">{{ statusText }}</span>
        </div>

        <!-- Metric Guide toggle button -->
        <button class="action-icon-btn" @click="isGuideOpen = true" title="View Metric Guide">
          <svg class="action-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
            <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
          </svg>
          <span class="btn-lbl-text">Metric Guide</span>
        </button>

        <!-- Theme Toggle -->
        <button class="action-icon-btn theme-toggle" @click="toggleTheme" title="Toggle Theme">
          <svg v-if="theme === 'dark'" class="action-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="5"/>
            <line x1="12" y1="1" x2="12" y2="3"/>
            <line x1="12" y1="21" x2="12" y2="23"/>
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
            <line x1="1" y1="12" x2="3" y2="12"/>
            <line x1="21" y1="12" x2="23" y2="12"/>
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
          </svg>
          <svg v-else class="action-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
          </svg>
        </button>
      </div>
    </header>

    <!-- Campaign Selector Card -->
    <section class="campaign-selector glass fade-in-up">
      <div class="selector-row">
        <div class="field">
          <label for="campaign-input">Track Campaign ID</label>
          <div class="input-with-button">
            <input
              id="campaign-input"
              type="text"
              v-model.trim="campaignId"
              placeholder="e.g. camp-1"
              @keyup.enter="startPolling"
            />
            <button id="btn-track" class="btn-primary" @click="startPolling" :disabled="!campaignId">
              Track Campaign
            </button>
          </div>
        </div>
      </div>

      <!-- Pins & History -->
      <div class="quick-picks" v-if="pinnedCampaigns.length > 0 || quickPicks.length > 0">
        <span class="picks-label">Saved Campaigns:</span>
        <div class="chips-container">
          <!-- Pinned items -->
          <span
            v-for="id in pinnedCampaigns"
            :key="'pinned-' + id"
            class="pick-chip pinned-chip"
            :class="{ active: activeCampaign === id }"
            @click="campaignId = id; startPolling()"
          >
            <!-- Pin SVG -->
            <svg class="chip-pin-svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M16 12V4H17V2H7V4H8V12L6 14V16H11.2V22H12.8V16H18V14L16 12Z"/>
            </svg>
            {{ id }}
            <span class="unpin-icon" @click.stop="unpinCampaign(id)" title="Unpin">✕</span>
          </span>

          <!-- Separator if both exist -->
          <span v-if="pinnedCampaigns.length > 0 && quickPicks.length > 0" class="picks-divider">|</span>

          <!-- Quick pick defaults -->
          <span
            v-for="id in quickPicks"
            :key="'quick-' + id"
            class="pick-chip"
            :class="{ active: activeCampaign === id && !pinnedCampaigns.includes(id) }"
            @click="campaignId = id; startPolling()"
          >
            {{ id }}
          </span>
        </div>
      </div>
    </section>

    <!-- Main Analytics Section -->
    <section v-if="activeCampaign" class="stats-section">
      <!-- Metadata Bar -->
      <div class="meta-bar glass fade-in-up" style="animation-delay: 0.05s">
        <div class="meta-left">
          <span class="meta-campaign">
            Campaign: <strong>{{ activeCampaign }}</strong>
          </span>
          <button 
            class="pin-toggle-btn" 
            :class="{ 'is-pinned': isPinned }"
            @click="isPinned ? unpinCampaign(activeCampaign) : pinCampaign(activeCampaign)"
          >
            <span v-if="isPinned">Bookmarked</span>
            <span v-else>Bookmark</span>
          </button>
        </div>
        <div class="meta-right">
          <span class="meta-updated" :class="{ stale: isStale }">
            Last update: {{ lastUpdatedText }}
          </span>
          <span class="meta-poll">
            Connection: HTTP Polling (5s)
          </span>
        </div>
      </div>

      <!-- Loading skeleton -->
      <div v-if="state === 'loading'" class="stats-grid">
        <div v-for="n in 4" :key="n" class="skeleton-card">
          <div class="skeleton" style="width: 44px; height: 44px; border-radius: 6px;"></div>
          <div style="flex: 1; display: flex; flex-direction: column; gap: 8px;">
            <div class="skeleton" style="width: 60px; height: 12px;"></div>
            <div class="skeleton" style="width: 100px; height: 28px;"></div>
          </div>
        </div>
      </div>

      <!-- Error State -->
      <div v-else-if="state === 'error'" class="error-card glass fade-in-up">
        <svg class="error-card-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <div class="error-details">
          <p class="error-title">Database Connection Error</p>
          <p class="error-msg">{{ errorMsg }}</p>
        </div>
        <button class="btn-retry" @click="fetchStats">Retry Connection</button>
      </div>

      <!-- Empty State -->
      <div v-else-if="state === 'empty'" class="empty-card glass fade-in-up">
        <svg class="empty-card-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <path d="M8 12h8"/>
        </svg>
        <p class="empty-title">No events recorded for this campaign</p>
        <p class="empty-sub">Simulate event volume using the load generator control below.</p>
      </div>

      <!-- Active Dashboard Content (Ready) -->
      <div v-else class="dashboard-active-content fade-in">


        <!-- Stats Grid (Professional 4-Card Layout) -->
        <div class="stats-grid">
          <!-- Sent -->
          <StatsCard
            label="Sent"
            :value="stats.sent"
            icon="sent"
            color="blue"
            tooltip="Total email campaign dispatches processed."
            subtitle="Dispatched Events"
            delay="0.05s"
          />

          <!-- Opened -->
          <StatsCard
            label="Opened"
            :value="stats.opened"
            icon="opened"
            color="blue"
            :rate="openRate"
            rateLabel="Open Rate"
            tooltip="Sent emails opened by recipients. Monitors reader hook interest."
            subtitle="Unique Read Opens"
            delay="0.1s"
          />

          <!-- Clicked -->
          <StatsCard
            label="Clicked"
            :value="stats.clicked"
            icon="clicked"
            color="amber"
            :rate="clickRate"
            rateLabel="CTR"
            tooltip="Recipients clicking links. Measures campaign-wide click engagement."
            subtitle="Unique Link Clicks"
            delay="0.15s"
          />

          <!-- Bounced -->
          <StatsCard
            label="Bounced"
            :value="stats.bounced"
            icon="bounced"
            color="rose"
            :rate="bounceRate"
            rateLabel="Bounce"
            tooltip="Sent emails rejected by recipient email servers. Target is under 2.0."
            subtitle="Rejected Deliveries"
            delay="0.2s"
          />
        </div>
      </div>

      <!-- Load Generator control widget -->
      <LoadGenerator :campaign-id="activeCampaign" />
    </section>

    <!-- Welcome screen (No campaign active) -->
    <section v-else class="welcome glass fade-in-up" style="animation-delay: 0.1s">
      <svg class="welcome-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
        <line x1="8" y1="21" x2="16" y2="21"/>
        <line x1="12" y1="17" x2="12" y2="21"/>
      </svg>
      <h2>System Engagement Monitor</h2>
      <p>Enter a Campaign ID above to begin polling live statistics from MySQL & Redis streams.</p>
    </section>

    <!-- Terminology reference guide slideout -->
    <MetricGuide :isOpen="isGuideOpen" @close="isGuideOpen = false" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useCampaignStats } from '../composables/useCampaignStats.js'
import StatsCard from './StatsCard.vue'
import LoadGenerator from './LoadGenerator.vue'
import MetricGuide from './MetricGuide.vue'


const quickPicks = ['camp-1', 'camp-2', 'camp-3']
const isGuideOpen = ref(false)

const {
  campaignId,
  activeCampaign,
  state,
  stats,
  errorMsg,
  lastUpdated,
  throughputSpeed,
  pinnedCampaigns,
  theme,
  openRate,
  clickRate,
  bounceRate,
  isStale,
  lastUpdatedText,
  statusClass,
  statusText,
  isPinned,
  startPolling,
  fetchStats,
  pinCampaign,
  unpinCampaign,
  toggleTheme,
} = useCampaignStats()

// Restore tracking if there was a previously saved active campaign
onMounted(() => {
  if (campaignId.value) {
    startPolling()
  }
})
</script>

<style scoped>
.dashboard {
  max-width: 1000px;
  width: 100%;
  margin: 0 auto;
  padding: 32px 24px 64px;
}

/* ── Header ─────────────────────────────────────────────── */
.dash-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  flex-wrap: wrap;
  gap: 16px;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 20px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-svg-logo {
  width: 28px;
  height: 28px;
  color: var(--accent-blue);
}

.brand-name {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}

.brand-sub {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 500;
  letter-spacing: 0.02em;
  margin-top: 1px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.speed-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 600;
  font-family: var(--font-mono);
}

.action-icon-svg {
  width: 15px;
  height: 15px;
  stroke-width: 2;
}

.text-amber {
  color: var(--accent-amber);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 600;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.status-live .status-dot { background: var(--accent-emerald); }
.status-loading .status-dot { background: var(--accent-blue); animation: pulse-soft 1s infinite; }
.status-error .status-dot { background: var(--accent-rose); }
.status-idle .status-dot { background: var(--text-muted); }

.action-icon-btn {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all var(--transition-fast);
}

.action-icon-btn:hover {
  background: var(--bg-card-hover);
  border-color: var(--border-color-hover);
  color: var(--text-primary);
}

/* ── Campaign Selector ──────────────────────────────────── */
.campaign-selector {
  padding: 20px;
  border-radius: var(--radius-md);
  margin-bottom: 24px;
}

.selector-row {
  display: flex;
  width: 100%;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.field label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.input-with-button {
  display: flex;
  gap: 10px;
  width: 100%;
}

.input-with-button input {
  flex: 1;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-color);
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 14px;
  font-family: var(--font-mono);
  outline: none;
  transition: border-color var(--transition-fast);
}

.input-with-button input:focus {
  border-color: var(--accent-blue);
}

.btn-primary {
  padding: 8px 16px;
  border-radius: var(--radius-sm);
  border: none;
  background: var(--accent-blue);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background var(--transition-fast);
  white-space: nowrap;
}

.btn-primary:hover:not(:disabled) {
  background: #1d4ed8;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Bookmarks & picks */
.quick-picks {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 14px;
  flex-wrap: wrap;
}

.picks-label {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  white-space: nowrap;
}

.chips-container {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.pick-chip {
  padding: 3px 10px;
  border-radius: 12px;
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  font-family: var(--font-mono);
  cursor: pointer;
  transition: all var(--transition-fast);
  display: inline-flex;
  align-items: center;
  gap: 4px;
  user-select: none;
}

.pick-chip:hover {
  border-color: var(--border-color-hover);
  color: var(--text-primary);
}

.pick-chip.active {
  background: var(--bg-card-hover);
  border-color: var(--accent-blue);
  color: var(--text-primary);
  font-weight: 600;
}

.chip-pin-svg {
  width: 10px;
  height: 10px;
  color: var(--text-muted);
}

.pinned-chip {
  border-color: var(--border-color);
  background: rgba(0, 0, 0, 0.05);
}

.pinned-chip.active {
  border-color: var(--accent-violet);
}

.pinned-chip.active .chip-pin-svg {
  color: var(--accent-violet);
}

.unpin-icon {
  font-size: 9px;
  color: var(--text-muted);
  cursor: pointer;
  padding: 1px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 12px;
  height: 12px;
  transition: all var(--transition-fast);
}

.unpin-icon:hover {
  background: var(--border-color-hover);
  color: var(--accent-rose);
}

.picks-divider {
  color: var(--border-color);
  font-size: 12px;
  user-select: none;
}

/* ── Meta Bar ───────────────────────────────────────────── */
.meta-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 20px;
  font-size: 12px;
  color: var(--text-secondary);
  padding: 10px 16px;
  border-radius: var(--radius-md);
}

.meta-left, .meta-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.meta-campaign strong {
  color: var(--text-primary);
}

.pin-toggle-btn {
  background: transparent;
  border: 1px solid var(--border-color);
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--transition-fast);
}

.pin-toggle-btn:hover {
  border-color: var(--border-color-hover);
  color: var(--text-primary);
}

.pin-toggle-btn.is-pinned {
  border-color: var(--accent-violet);
  color: var(--accent-violet);
  background: var(--accent-violet-glow);
}

.meta-updated.stale {
  color: var(--accent-rose);
  font-weight: 600;
}

/* ── Stats Grid ─────────────────────────────────────────── */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
  margin-top: 16px;
}

@media (min-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

.skeleton-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 18px;
  border-radius: var(--radius-md);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
}

/* ── Error State ────────────────────────────────────────── */
.error-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  border-radius: var(--radius-md);
  border-color: var(--accent-rose) !important;
  background: rgba(239, 68, 68, 0.02);
  flex-wrap: wrap;
}

.error-card-svg {
  width: 32px;
  height: 32px;
  color: var(--accent-rose);
}

.error-details {
  flex: 1;
  min-width: 200px;
}

.error-title { 
  font-weight: 600; 
  color: var(--accent-rose); 
  margin-bottom: 2px; 
  font-size: 14px;
}

.error-msg { 
  font-size: 12px; 
  color: var(--text-secondary); 
}

.btn-retry {
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--accent-rose);
  background: transparent;
  color: var(--accent-rose);
  font-weight: 600;
  font-size: 12px;
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.btn-retry:hover {
  background: var(--accent-rose-glow);
}

/* ── Empty State ────────────────────────────────────────── */
.empty-card {
  text-align: center;
  padding: 40px 20px;
  border-radius: var(--radius-md);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
}

.empty-card-svg {
  width: 40px;
  height: 40px;
  color: var(--text-muted);
  margin-bottom: 8px;
  display: inline-block;
}

.empty-title { 
  font-size: 14px; 
  font-weight: 600; 
  margin-bottom: 4px; 
}

.empty-sub { 
  font-size: 12px; 
  color: var(--text-muted); 
  max-width: 400px; 
  margin: 0 auto; 
}

/* ── Welcome State ──────────────────────────────────────── */
.welcome {
  text-align: center;
  padding: 60px 24px;
  border-radius: var(--radius-md);
  color: var(--text-secondary);
}

.welcome-svg {
  width: 48px;
  height: 48px;
  color: var(--text-muted);
  margin-bottom: 12px;
  display: inline-block;
}

.welcome h2 { 
  font-size: 16px; 
  font-weight: 600; 
  color: var(--text-primary);
  margin-bottom: 6px; 
}

.welcome p { 
  font-size: 12px; 
  color: var(--text-muted); 
  max-width: 380px; 
  margin: 0 auto; 
}
</style>
