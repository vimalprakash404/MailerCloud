<template>
  <div class="load-gen glass" :class="{ 'is-running': isRunning }">
    <h3 class="section-title">⚡ Load Generator</h3>

    <div class="controls">
      <div class="field">
        <label for="lg-count">Events</label>
        <input
          id="lg-count"
          type="number"
          v-model.number="count"
          min="1"
          max="100000"
          :disabled="isRunning"
        />
      </div>

      <div class="field">
        <label for="lg-concurrency">Concurrency</label>
        <input
          id="lg-concurrency"
          type="number"
          v-model.number="concurrency"
          min="1"
          max="200"
          :disabled="isRunning"
        />
      </div>

      <button
        id="lg-fire"
        class="btn-fire"
        @click="fire"
        :disabled="isRunning || !campaignId"
      >
        <span v-if="!isRunning">🚀 Fire Burst</span>
        <span v-else>⏳ Sending...</span>
      </button>
    </div>

    <!-- Progress bar -->
    <div v-if="isRunning" class="progress-container">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: progressPct + '%' }"></div>
      </div>
      <span class="progress-label">{{ progress }} / {{ count }}</span>
    </div>

    <!-- Results -->
    <div v-if="result" class="result-card fade-in-up">
      <span class="result-item">✅ Sent: <strong>{{ result.sent.toLocaleString() }}</strong></span>
      <span class="result-item">❌ Errors: <strong>{{ result.errors }}</strong></span>
      <span class="result-item">⏱️ Duration: <strong>{{ result.durationMs.toLocaleString() }}ms</strong></span>
      <span class="result-item">📈 Rate: <strong>{{ eventsPerSec }}/sec</strong></span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { fireBurst } from '../api/client.js'

const props = defineProps({
  campaignId: { type: String, default: '' },
})

const count = ref(1000)
const concurrency = ref(50)
const isRunning = ref(false)
const progress = ref(0)
const result = ref(null)

const progressPct = computed(() =>
  count.value > 0 ? Math.round((progress.value / count.value) * 100) : 0
)

const eventsPerSec = computed(() => {
  if (!result.value || result.value.durationMs === 0) return '—'
  return Math.round((result.value.sent / result.value.durationMs) * 1000).toLocaleString()
})

async function fire() {
  if (!props.campaignId || isRunning.value) return

  isRunning.value = true
  progress.value = 0
  result.value = null

  try {
    result.value = await fireBurst(
      props.campaignId,
      count.value,
      concurrency.value,
      (sent, total) => { progress.value = sent }
    )
  } catch (err) {
    result.value = { sent: 0, errors: count.value, durationMs: 0 }
  } finally {
    isRunning.value = false
  }
}
</script>

<style scoped>
.load-gen {
  padding: 24px;
  border-radius: var(--radius-lg);
  margin-top: 24px;
}

.load-gen.is-running {
  border-color: var(--accent-amber);
}

.section-title {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 16px;
  color: var(--text-primary);
}

.controls {
  display: flex;
  gap: 16px;
  align-items: flex-end;
  flex-wrap: wrap;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.field input {
  width: 140px;
  padding: 10px 14px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-color);
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 15px;
  font-family: var(--font-mono);
  outline: none;
  transition: border-color 0.2s;
}

.field input:focus {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 3px var(--accent-blue-glow);
}

.field input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-fire {
  padding: 10px 28px;
  border-radius: var(--radius-sm);
  border: none;
  background: linear-gradient(135deg, var(--accent-violet), var(--accent-blue));
  color: #fff;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.3s, opacity 0.2s;
  white-space: nowrap;
}

.btn-fire:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 20px var(--accent-violet-glow);
}

.btn-fire:active:not(:disabled) {
  transform: translateY(0);
}

.btn-fire:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Progress */
.progress-container {
  margin-top: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-bar {
  flex: 1;
  height: 8px;
  background: var(--bg-input);
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-blue), var(--accent-emerald));
  border-radius: 4px;
  transition: width 0.3s ease;
}

.progress-label {
  font-size: 13px;
  font-family: var(--font-mono);
  color: var(--text-secondary);
  min-width: 100px;
  text-align: right;
}

/* Results */
.result-card {
  margin-top: 16px;
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
  padding: 16px;
  border-radius: var(--radius-sm);
  background: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.result-item {
  font-size: 14px;
  color: var(--text-secondary);
}

.result-item strong {
  color: var(--text-primary);
}
</style>
