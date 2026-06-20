<template>
  <div class="load-gen glass" :class="{ 'is-running': isRunning }">
    <div class="lg-header">
      <h3 class="section-title">High-Throughput Load Generator</h3>
      <span class="info-badge">Async Batch Ingest</span>
    </div>

    <!-- Controls grid -->
    <div class="controls-grid">
      <div class="field">
        <label for="lg-count">Total Events</label>
        <div class="input-wrapper">
          <input
            id="lg-count"
            type="number"
            v-model.number="count"
            min="1"
            max="100000"
            :disabled="isRunning"
          />
          <span class="input-unit">evt</span>
        </div>
      </div>

      <div class="field">
        <label for="lg-concurrency">Concurrency</label>
        <div class="input-wrapper">
          <input
            id="lg-concurrency"
            type="number"
            v-model.number="concurrency"
            min="1"
            max="200"
            :disabled="isRunning"
          />
          <span class="input-unit">th</span>
        </div>
      </div>

      <div class="field lg-preset-field">
        <label for="lg-preset">Simulation Preset</label>
        <select id="lg-preset" v-model="selectedPreset" :disabled="isRunning" class="preset-select">
          <option value="standard">Standard Ingestion (25% Split)</option>
          <option value="engagement">High Engagement Campaign</option>
          <option value="spam">Spam Trap Simulation</option>
          <option value="viral">Viral Campaign Simulation</option>
          <option value="custom">Custom Ingestion Ratios...</option>
        </select>
      </div>

      <button
        id="lg-fire"
        class="btn-fire"
        @click="fire"
        :disabled="isRunning || !campaignId"
      >
        <span v-if="!isRunning">Start Load Generation</span>
        <span v-else class="loading-spin">Sending...</span>
      </button>
    </div>

    <!-- Custom Ratios Sliders -->
    <div v-if="selectedPreset === 'custom'" class="custom-sliders fade-in-up">
      <h5 class="sliders-title">Adjust Event Ratios</h5>
      <div class="sliders-grid">
        <div class="slider-item">
          <div class="slider-label">
            <span>Sent:</span>
            <strong>{{ customRatios.sent }}%</strong>
          </div>
          <input 
            type="range" 
            v-model.number="customRatios.sent" 
            min="0" 
            max="100" 
            :disabled="isRunning" 
            class="range-slider blue-slider"
          />
        </div>
        <div class="slider-item">
          <div class="slider-label">
            <span>Opened:</span>
            <strong>{{ customRatios.opened }}%</strong>
          </div>
          <input 
            type="range" 
            v-model.number="customRatios.opened" 
            min="0" 
            max="100" 
            :disabled="isRunning" 
            class="range-slider emerald-slider"
          />
        </div>
        <div class="slider-item">
          <div class="slider-label">
            <span>Clicked:</span>
            <strong>{{ customRatios.clicked }}%</strong>
          </div>
          <input 
            type="range" 
            v-model.number="customRatios.clicked" 
            min="0" 
            max="100" 
            :disabled="isRunning" 
            class="range-slider amber-slider"
          />
        </div>
        <div class="slider-item">
          <div class="slider-label">
            <span>Bounced:</span>
            <strong>{{ customRatios.bounced }}%</strong>
          </div>
          <input 
            type="range" 
            v-model.number="customRatios.bounced" 
            min="0" 
            max="100" 
            :disabled="isRunning" 
            class="range-slider rose-slider"
          />
        </div>
      </div>
      <p class="slider-note">Ratios will be normalized automatically on generation.</p>
    </div>

    <!-- Progress bar -->
    <div v-if="isRunning" class="progress-section fade-in">
      <div class="progress-bar-row">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: progressPct + '%' }"></div>
        </div>
        <span class="progress-label">{{ progress.toLocaleString() }} / {{ count.toLocaleString() }}</span>
      </div>
      <span class="progress-percentage">{{ progressPct }}% complete</span>
    </div>

    <!-- Results Card -->
    <div v-if="result" class="result-card fade-in-up">
      <div class="result-grid">
        <div class="result-stat-item">
          <span class="res-dot dot-blue"></span>
          <span class="res-lbl">Sent:</span>
          <strong>{{ result.sent.toLocaleString() }}</strong>
        </div>
        <div class="result-stat-item">
          <span class="res-dot dot-rose"></span>
          <span class="res-lbl">Errors:</span>
          <strong :class="{ 'has-errors': result.errors > 0 }">{{ result.errors }}</strong>
        </div>
        <div class="result-stat-item">
          <span class="res-dot dot-violet"></span>
          <span class="res-lbl">Duration:</span>
          <strong>{{ result.durationMs.toLocaleString() }}ms</strong>
        </div>
        <div class="result-stat-item">
          <span class="res-dot dot-emerald"></span>
          <span class="res-lbl">Throughput:</span>
          <strong class="speed-txt">{{ eventsPerSec }} ev/s</strong>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, reactive } from 'vue'
import { fireBurst } from '../api/client.js'

const props = defineProps({
  campaignId: { type: String, default: '' },
})

const count = ref(20000)
const concurrency = ref(50)
const isRunning = ref(false)
const progress = ref(0)
const result = ref(null)

const selectedPreset = ref('standard')
const customRatios = reactive({
  sent: 40,
  opened: 30,
  clicked: 20,
  bounced: 10
})

const presets = {
  standard: { sent: 25, opened: 25, clicked: 25, bounced: 25 },
  engagement: { sent: 40, opened: 35, clicked: 22, bounced: 3 },
  spam: { sent: 50, opened: 5, clicked: 1, bounced: 44 },
  viral: { sent: 30, opened: 28, clicked: 25, bounced: 17 }
}

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

  let ratios = null
  if (selectedPreset.value === 'custom') {
    ratios = { ...customRatios }
  } else {
    ratios = presets[selectedPreset.value] || null
  }

  try {
    result.value = await fireBurst(
      props.campaignId,
      count.value,
      concurrency.value,
      (sent, total) => { progress.value = sent },
      ratios
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
  padding: 20px;
  border-radius: var(--radius-md);
  margin-top: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  transition: border-color var(--transition-normal);
}

.load-gen.is-running {
  border-color: var(--accent-amber);
}

.lg-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.info-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  background: var(--accent-blue-glow);
  color: var(--accent-blue);
  text-transform: uppercase;
}

.controls-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 12px;
  align-items: flex-end;
}

@media (min-width: 600px) {
  .controls-grid {
    grid-template-columns: 1fr 1fr 2fr auto;
  }
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.input-wrapper {
  display: flex;
  align-items: center;
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  overflow: hidden;
  transition: border-color var(--transition-fast);
}

.input-wrapper:focus-within {
  border-color: var(--accent-blue);
}

.input-wrapper input {
  width: 100%;
  padding: 8px 12px;
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 13px;
  font-family: var(--font-mono);
  outline: none;
}

.input-unit {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  padding-right: 12px;
  user-select: none;
}

.preset-select {
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-color);
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  cursor: pointer;
  width: 100%;
  transition: border-color var(--transition-fast);
}

.preset-select:focus {
  border-color: var(--accent-blue);
}

.btn-fire {
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
  height: 33px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-fire:hover:not(:disabled) {
  background: #1d4ed8;
}

.btn-fire:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Custom ratio sliders */
.custom-sliders {
  margin-top: 16px;
  padding: 14px;
  border-radius: var(--radius-sm);
  background: rgba(0, 0, 0, 0.05);
  border: 1px solid var(--border-color);
}

.sliders-title {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--text-secondary);
  margin-bottom: 10px;
  letter-spacing: 0.05em;
}

.sliders-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.slider-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.slider-label {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: var(--text-secondary);
}

.slider-label strong {
  font-family: var(--font-mono);
}

.range-slider {
  -webkit-appearance: none;
  appearance: none;
  width: 100%;
  height: 4px;
  border-radius: 2px;
  background: var(--bg-input);
  outline: none;
}

.range-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  cursor: pointer;
}

.blue-slider::-webkit-slider-thumb { background: var(--accent-blue); }
.emerald-slider::-webkit-slider-thumb { background: var(--accent-emerald); }
.amber-slider::-webkit-slider-thumb { background: var(--accent-amber); }
.rose-slider::-webkit-slider-thumb { background: var(--accent-rose); }

.slider-note {
  font-size: 10px;
  color: var(--text-muted);
  margin-top: 8px;
}

/* Progress bar section */
.progress-section {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.progress-bar-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.progress-bar {
  flex: 1;
  height: 6px;
  background: var(--bg-input);
  border-radius: 3px;
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.progress-fill {
  height: 100%;
  background: var(--accent-blue);
  border-radius: 3px;
  transition: width 0.1s ease-out;
}

.progress-label {
  font-size: 11px;
  font-family: var(--font-mono);
  color: var(--text-secondary);
  min-width: 90px;
  text-align: right;
}

.progress-percentage {
  font-size: 10px;
  color: var(--text-muted);
  font-weight: 500;
}

/* Results section */
.result-card {
  margin-top: 16px;
  padding: 12px;
  border-radius: var(--radius-sm);
  background: var(--bg-card-hover);
  border: 1px solid var(--border-color);
}

.result-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(110px, 1fr));
  gap: 10px;
}

.result-stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
}

.res-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.dot-blue { background: var(--accent-blue); }
.dot-rose { background: var(--accent-rose); }
.dot-violet { background: var(--accent-violet); }
.dot-emerald { background: var(--accent-emerald); }

.res-lbl {
  color: var(--text-muted);
}

.result-stat-item strong {
  color: var(--text-primary);
  font-family: var(--font-mono);
}

.has-errors {
  color: var(--accent-rose) !important;
}

.speed-txt {
  color: var(--accent-emerald) !important;
}
</style>
