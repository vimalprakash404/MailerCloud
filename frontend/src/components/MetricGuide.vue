<template>
  <Teleport to="body">
    <div v-if="isOpen" class="guide-overlay" @click="$emit('close')">
      <div class="guide-drawer glass" @click.stop>
        <!-- Header -->
        <div class="drawer-header">
          <h3>Metric Reference Guide</h3>
          <button class="btn-close" @click="$emit('close')">✕</button>
        </div>

        <!-- Content -->
        <div class="drawer-body">
          <section class="guide-section">
            <h4>Industry Standard Metrics</h4>
            <p class="section-intro">Understand what these engagement rates mean and how they measure campaign health.</p>



            <!-- Open Rate -->
            <div class="metric-definition">
              <div class="def-title">
                <svg class="def-icon-svg text-blue" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                  <circle cx="12" cy="12" r="3" />
                </svg>
                <h5>Open Rate</h5>
              </div>
              <p class="def-desc">The percentage of sent emails opened. Measures initial subscriber interest.</p>
              <div class="formula">Formula: <code>Opened / Sent * 100</code></div>
              <div class="benchmark">Benchmark: <strong class="blue">15.0% – 25.0%</strong></div>
            </div>

            <!-- Click Rate (CTR) -->
            <div class="metric-definition">
              <div class="def-title">
                <svg class="def-icon-svg text-amber" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M3 3l7.07 16.97 2.51-7.39 7.39-2.51L3 3z" />
                  <path d="M13 13l6 6" />
                </svg>
                <h5>Click Rate (CTR)</h5>
              </div>
              <p class="def-desc">Click-Through Rate represents the proportion of recipients who clicked a link in the email.</p>
              <div class="formula">Formula: <code>Clicked / Sent * 100</code></div>
              <div class="benchmark">Benchmark: <strong class="amber">1.5% – 3.0%</strong></div>
            </div>



            <!-- Bounce Rate -->
            <div class="metric-definition">
              <div class="def-title">
                <svg class="def-icon-svg text-rose" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="12" y1="8" x2="12" y2="12" />
                  <line x1="12" y1="16" x2="12.01" y2="16" />
                </svg>
                <h5>Bounce Rate</h5>
              </div>
              <p class="def-desc">The rate of emails rejected by destination mail servers. High bounce rates affect sender IP reputation.</p>
              <div class="formula">Formula: <code>Bounced / Sent * 100</code></div>
              <div class="benchmark">Benchmark: <strong class="rose">Less than 2.0%</strong></div>
            </div>
          </section>

          <hr class="divider" />

          <section class="guide-section">
            <h4>Ingestion Pipeline Architecture</h4>
            <p class="section-intro">MailerCloud processes up to <strong>20,000 events/second</strong> through a decoupled queue system:</p>

            <div class="pipeline-flow">
              <div class="flow-step">
                <div class="step-num">1</div>
                <div>
                  <h6>Fast Ingest (Redis Queue)</h6>
                  <p>HTTP POST requests validate JSON payloads and append instantly to an in-memory Redis List queue (LPUSH) in &lt;1ms. MySQL is bypassed in the request lifecycle.</p>
                </div>
              </div>

              <div class="flow-step">
                <div class="step-num">2</div>
                <div>
                  <h6>Multi-Worker Batcher</h6>
                  <p>8 parallel Go goroutines consume from the Redis list. To reduce TCP overhead, events are drained in bulk batches of up to 2,000 entries.</p>
                </div>
              </div>

              <div class="flow-step">
                <div class="step-num">3</div>
                <div>
                  <h6>Idempotent Database Flush</h6>
                  <p>Each batch is flushed in a MySQL InnoDB transaction using <code>INSERT IGNORE</code> (deduplication) and <code>ON DUPLICATE KEY UPDATE</code> (atomic counter increments).</p>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
defineProps({
  isOpen: { type: Boolean, default: false }
})
defineEmits(['close'])
</script>

<style scoped>
.guide-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(2px);
  z-index: 1000;
  display: flex;
  justify-content: flex-end;
  animation: fadeIn 0.15s ease-out;
}

.guide-drawer {
  width: 100%;
  max-width: 400px;
  height: 100%;
  border-left: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  box-shadow: -4px 0 15px rgba(0, 0, 0, 0.1);
  animation: slideIn 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
  background: var(--bg-card);
}

.drawer-header {
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.drawer-header h3 {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.btn-close {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 16px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.btn-close:hover {
  background: var(--bg-card-hover);
  color: var(--text-primary);
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.guide-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.guide-section h4 {
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 6px;
}

.section-intro {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.4;
}

.metric-definition {
  padding: 12px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-color);
  background: var(--bg-input);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.bg-highlight {
  border-left: 3px solid var(--accent-violet);
}

.def-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.def-icon-svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.def-title h5 {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-primary);
}

.def-desc {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.4;
}

.formula {
  font-size: 11px;
  font-family: var(--font-mono);
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  padding: 3px 6px;
  border-radius: var(--radius-sm);
  width: fit-content;
}

.benchmark {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.02em;
  color: var(--text-muted);
}

.benchmark strong {
  font-weight: 700;
}

.divider {
  border: none;
  border-top: 1px solid var(--border-color);
}

/* Pipeline Flow Design */
.pipeline-flow {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.flow-step {
  display: flex;
  gap: 12px;
}

.step-num {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--bg-card-hover);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  flex-shrink: 0;
}

.flow-step h6 {
  font-size: 12px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 2px;
}

.flow-step p {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.4;
}

/* Accent Colors */
.text-emerald { color: var(--accent-emerald); }
.text-blue { color: var(--accent-blue); }
.text-amber { color: var(--accent-amber); }
.text-violet { color: var(--accent-violet); }
.text-rose { color: var(--accent-rose); }

.emerald { color: var(--accent-emerald); }
.blue { color: var(--accent-blue); }
.amber { color: var(--accent-amber); }
.violet { color: var(--accent-violet); }
.rose { color: var(--accent-rose); }

@keyframes slideIn {
  from { transform: translateX(100%); }
  to { transform: translateX(0); }
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
</style>
