<template>
  <div class="stats-card fade-in-up" :class="glowClass" :style="{ animationDelay: delay }">
    <div class="card-icon" :style="{ background: iconBg }">
      <span class="icon-text">{{ icon }}</span>
    </div>
    <div class="card-body">
      <span class="card-label">{{ label }}</span>
      <span class="card-value" :class="{ counting: isAnimating }">
        {{ displayValue.toLocaleString() }}
      </span>
      <span v-if="rate !== null" class="card-rate" :class="rateClass">
        {{ rate }}%
      </span>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'

const props = defineProps({
  label:    { type: String, required: true },
  value:    { type: Number, default: 0 },
  icon:     { type: String, default: '📊' },
  color:    { type: String, default: 'blue' },   // blue | emerald | amber | rose
  rate:     { type: [Number, null], default: null },
  delay:    { type: String, default: '0s' },
})

const displayValue = ref(props.value)
const isAnimating = ref(false)

// Animate count changes
watch(() => props.value, (newVal, oldVal) => {
  if (newVal === oldVal) return
  isAnimating.value = true

  const diff = newVal - displayValue.value
  const steps = Math.min(Math.abs(diff), 30)
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
  }
  return map[props.color] || map.blue
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
  align-items: center;
  gap: 16px;
  padding: 24px;
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  transition: transform 0.2s ease, box-shadow 0.3s ease;
}

.stats-card:hover {
  transform: translateY(-2px);
  background: var(--bg-card-hover);
}

.card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}

.icon-text {
  font-size: 24px;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.card-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.card-value {
  font-size: 32px;
  font-weight: 800;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
  transition: color 0.3s;
}

.card-value.counting {
  color: var(--accent-blue);
}

.card-rate {
  font-size: 14px;
  font-weight: 600;
  font-family: var(--font-mono);
}

.rate-good { color: var(--accent-emerald); }
.rate-ok   { color: var(--accent-amber); }
.rate-low  { color: var(--accent-rose); }
</style>
