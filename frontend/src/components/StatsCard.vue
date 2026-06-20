<template>
  <div class="stats-card glass fade-in-up" :class="glowClass" :style="{ animationDelay: delay }">
    <!-- Top Row: Icon and Tooltip -->
    <div class="card-header">
      <div class="card-icon" :style="{ background: iconBg }">
        <!-- Render appropriate corporate SVG based on icon string prop -->
        <svg v-if="icon === 'sent'" class="card-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="22" y1="2" x2="11" y2="13" />
          <polygon points="22 2 15 22 11 13 2 9 22 2" />
        </svg>
        <svg v-else-if="icon === 'delivered'" class="card-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
          <polyline points="22 4 12 14.01 9 11.01" />
        </svg>
        <svg v-else-if="icon === 'bounced'" class="card-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        <svg v-else-if="icon === 'opened'" class="card-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
          <circle cx="12" cy="12" r="3" />
        </svg>
        <svg v-else-if="icon === 'clicked'" class="card-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 3l7.07 16.97 2.51-7.39 7.39-2.51L3 3z" />
          <path d="M13 13l6 6" />
        </svg>
        <svg v-else-if="icon === 'ctor'" class="card-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <circle cx="12" cy="12" r="6" />
          <circle cx="12" cy="12" r="2" />
        </svg>
        <!-- Fallback standard bar chart -->
        <svg v-else class="card-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="20" x2="18" y2="10" />
          <line x1="12" y1="20" x2="12" y2="4" />
          <line x1="6" y1="20" x2="6" y2="14" />
        </svg>
      </div>
      
      <div v-if="tooltip" class="tooltip-trigger">
        <!-- Clean small info icon -->
        <svg class="info-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="16" x2="12" y2="12" />
          <line x1="12" y1="8" x2="12.01" y2="8" />
        </svg>
        <span class="tooltip-text">{{ tooltip }}</span>
      </div>
    </div>

    <!-- Main Content -->
    <div class="card-content">
      <div class="card-main">
        <span class="card-label">{{ label }}</span>
        <h2 class="card-value" :class="{ counting: isAnimating }">
          {{ displayValue.toLocaleString() }}
        </h2>
        <span v-if="subtitle" class="card-subtitle">{{ subtitle }}</span>
      </div>

      <!-- Circular Progress Ring for Rate -->
      <div v-if="rate !== null" class="progress-ring-wrapper">
        <div class="progress-ring-container">
          <svg class="progress-ring" width="36" height="36">
            <circle 
              class="progress-ring-bg" 
              stroke="var(--border-color)" 
              stroke-width="3" 
              fill="transparent" 
              r="14" 
              cx="18" 
              cy="18" 
            />
            <circle 
              class="progress-ring-fill" 
              :style="{ 
                stroke: strokeColor,
                strokeDasharray: '88',
                strokeDashoffset: strokeDashoffset
              }" 
              stroke-width="3" 
              stroke-linecap="round" 
              fill="transparent" 
              r="14" 
              cx="18" 
              cy="18" 
            />
          </svg>
          <span class="progress-text" :style="{ color: strokeColor }">
            {{ Math.round(rate) }}%
          </span>
        </div>
        <span class="rate-label" :class="rateClass">{{ rateLabel || 'Rate' }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'

const props = defineProps({
  label:     { type: String, required: true },
  value:     { type: Number, default: 0 },
  icon:      { type: String, default: 'chart' },
  color:     { type: String, default: 'blue' },   // blue | emerald | amber | rose | violet
  rate:      { type: [Number, null], default: null },
  rateLabel: { type: String, default: '' },
  tooltip:   { type: String, default: '' },
  subtitle:  { type: String, default: '' },
  delay:     { type: String, default: '0s' },
})

const displayValue = ref(props.value)
const isAnimating = ref(false)

// Animate count changes
watch(() => props.value, (newVal, oldVal) => {
  if (newVal === oldVal) return
  isAnimating.value = true

  const diff = newVal - displayValue.value
  const steps = Math.min(Math.max(Math.abs(diff), 8), 24)
  const increment = diff / steps
  let step = 0

  const timer = setInterval(() => {
    step++
    if (step >= steps) {
      displayValue.value = newVal
      isAnimating.value = false
      clearInterval(timer)
    } else {
      displayValue.value = Math.round(displayValue.value + increment)
    }
  }, 20)
})

const glowClass = computed(() => `glow-${props.color}`)
const iconBg = computed(() => {
  const map = {
    blue:    'var(--accent-blue-glow)',
    emerald: 'var(--accent-emerald-glow)',
    amber:   'var(--accent-amber-glow)',
    rose:    'var(--accent-rose-glow)',
    violet:  'var(--accent-violet-glow)',
  }
  return map[props.color] || map.blue
})

const strokeColor = computed(() => {
  const map = {
    blue:    'var(--accent-blue)',
    emerald: 'var(--accent-emerald)',
    amber:   'var(--accent-amber)',
    rose:    'var(--accent-rose)',
    violet:  'var(--accent-violet)',
  }
  return map[props.color] || map.blue
})

// Dasharray perimeter for radius 14 is 2 * PI * 14 = 87.96 (~88)
const strokeDashoffset = computed(() => {
  if (props.rate === null) return 88
  const capped = Math.min(Math.max(props.rate, 0), 100)
  return 88 - (capped / 100) * 88
})

const rateClass = computed(() => {
  if (props.rate === null) return ''
  if (props.rate >= 50) return 'rate-good'
  if (props.rate >= 20) return 'rate-ok'
  return 'rate-low'
})
</script>

<style scoped>
.stats-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 18px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  transition: transform var(--transition-normal), background-color var(--transition-normal), box-shadow var(--transition-normal);
  position: relative;
}

.stats-card:hover {
  background: var(--bg-card-hover);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
  color: v-bind(strokeColor);
}

.card-icon-svg {
  width: 20px;
  height: 20px;
}

.info-icon-svg {
  width: 14px;
  height: 14px;
  color: var(--text-muted);
  cursor: help;
  transition: color var(--transition-fast);
}

.info-icon-svg:hover {
  color: var(--text-secondary);
}

.card-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 12px;
}

.card-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.card-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.card-value {
  font-size: 26px;
  font-weight: 700;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
  line-height: 1.1;
  letter-spacing: -0.01em;
}

.card-subtitle {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Progress Ring Layout */
.progress-ring-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.progress-ring-container {
  position: relative;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.progress-ring {
  transform: rotate(-90deg);
}

.progress-ring-fill {
  transition: stroke-dashoffset var(--transition-slow) ease;
}

.progress-text {
  position: absolute;
  font-size: 9px;
  font-weight: 700;
  font-family: var(--font-mono);
  letter-spacing: -0.02em;
}

.rate-label {
  font-size: 9px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.rate-good { color: var(--accent-emerald) !important; }
.rate-ok   { color: var(--accent-amber) !important; }
.rate-low  { color: var(--accent-rose) !important; }
</style>
