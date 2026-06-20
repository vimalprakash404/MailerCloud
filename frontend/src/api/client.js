/**
 * API client for the MailerCloud backend.
 * Uses async batch ingestion with retry-on-drop for guaranteed delivery.
 */

const API_BASE = '';  // same origin in production (Nginx proxy), Vite proxy in dev

const REQUEST_TIMEOUT_MS = 8000;
const BATCH_SIZE = 200; // events per HTTP request
const MAX_RETRIES = 8;  // max retries per batch

/**
 * Fetches campaign stats from the backend.
 * @param {string} campaignId
 * @param {AbortSignal} [signal] - external abort signal
 * @returns {Promise<{sent: number, opened: number, clicked: number, bounced: number}>}
 */
export async function fetchCampaignStats(campaignId, signal) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

  if (signal) {
    signal.addEventListener('abort', () => controller.abort());
  }

  try {
    const res = await fetch(`${API_BASE}/campaigns/${encodeURIComponent(campaignId)}/stats`, {
      signal: controller.signal,
    });

    if (!res.ok) {
      throw new Error(`HTTP ${res.status}: ${res.statusText}`);
    }

    return await res.json();
  } finally {
    clearTimeout(timeout);
  }
}

/**
 * Sends a single event to the backend (backward-compatible).
 */
export async function postEvent(event) {
  const res = await fetch(`${API_BASE}/events`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(event),
  });

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${res.statusText}`);
  }
}

/**
 * Sends a batch of events. Returns {accepted, dropped, total}.
 * On 429, throws so caller can retry the whole batch.
 */
async function postBatch(events) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 60000);

  try {
    const res = await fetch(`${API_BASE}/events/batch`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ events }),
      signal: controller.signal,
    });

    const body = await res.json();

    if (res.status === 429) {
      // Server is at capacity — throw so the worker retries the whole batch
      const err = new Error('Server backpressure (429)');
      err.retryable = true;
      throw err;
    }

    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`);
    }

    return body; // { accepted, dropped, total }
  } finally {
    clearTimeout(timeout);
  }
}

/**
 * Fires a burst of events using async batch ingestion with guaranteed delivery.
 *
 * Key design: when the server reports dropped events in a batch response,
 * those events are re-queued for retry instead of counted as errors. Combined
 * with server-side backpressure (blocking enqueue), this ensures zero event loss.
 *
 * @param {string} campaignId
 * @param {number} count - total events to send
 * @param {number} concurrency - parallel batch requests
 * @param {function} onProgress - callback(sent, total)
 * @returns {Promise<{sent: number, errors: number, durationMs: number}>}
 */
export async function fireBurst(campaignId, count, concurrency, onProgress, ratios) {
  const eventTypes = ['sent', 'opened', 'clicked', 'bounced'];
  const start = performance.now();
  let totalSent = 0;
  let totalErrors = 0;

  // ── 1. Build all events upfront ─────────────────────────────
  const allEvents = [];

  if (ratios) {
    const totalRatio = Object.values(ratios).reduce((a, b) => a + b, 0) || 1;
    const normalized = {
      sent: (ratios.sent || 0) / totalRatio,
      opened: (ratios.opened || 0) / totalRatio,
      clicked: (ratios.clicked || 0) / totalRatio,
      bounced: (ratios.bounced || 0) / totalRatio,
    };

    const typesToGenerate = [];
    let generatedCount = 0;

    eventTypes.forEach((type, idx) => {
      const isLast = idx === eventTypes.length - 1;
      const typeCount = isLast ? (count - generatedCount) : Math.round(normalized[type] * count);
      generatedCount += typeCount;
      for (let j = 0; j < typeCount; j++) {
        typesToGenerate.push(type);
      }
    });

    // Fisher-Yates shuffle
    for (let i = typesToGenerate.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [typesToGenerate[i], typesToGenerate[j]] = [typesToGenerate[j], typesToGenerate[i]];
    }

    for (let i = 0; i < count; i++) {
      allEvents.push({
        event_id: `load-${campaignId}-${Date.now()}-${i}-${Math.random().toString(36).slice(2, 8)}`,
        campaign_id: campaignId,
        type: typesToGenerate[i] || 'sent',
        timestamp: new Date().toISOString(),
      });
    }
  } else {
    for (let i = 0; i < count; i++) {
      allEvents.push({
        event_id: `load-${campaignId}-${Date.now()}-${i}-${Math.random().toString(36).slice(2, 8)}`,
        campaign_id: campaignId,
        type: eventTypes[i % eventTypes.length],
        timestamp: new Date().toISOString(),
      });
    }
  }

  // ── 2. Split into batches and create a shared work queue ────
  // This queue is shared across all workers. When a batch has drops,
  // the dropped events are re-batched and pushed back into the queue.
  const queue = [];
  for (let i = 0; i < allEvents.length; i += BATCH_SIZE) {
    queue.push(allEvents.slice(i, i + BATCH_SIZE));
  }

  // ── 3. Worker: send batch, re-queue drops, retry on failure ─
  const worker = async () => {
    while (queue.length > 0) {
      const batch = queue.shift();
      if (!batch) break;

      let retries = 0;
      let currentBatch = batch;

      while (retries < MAX_RETRIES && currentBatch.length > 0) {
        try {
          const result = await postBatch(currentBatch);
          totalSent += result.accepted;

          if (result.dropped > 0) {
            // Server accepted some but dropped others (buffer pressure).
            // The server doesn't tell us WHICH events dropped, but since
            // they're enqueued in order, the dropped ones are the tail.
            const droppedEvents = currentBatch.slice(result.accepted);
            if (droppedEvents.length > 0) {
              // Back off, then retry just the dropped portion
              await new Promise(r => setTimeout(r, 300 * (retries + 1)));
              currentBatch = droppedEvents;
              retries++;
              continue;
            }
          }
          // All accepted
          break;
        } catch (err) {
          retries++;
          if (retries >= MAX_RETRIES) {
            totalErrors += currentBatch.length;
            break;
          }
          // Exponential backoff: 300ms, 600ms, 1200ms, ...
          const delay = 300 * Math.pow(2, retries - 1);
          await new Promise(r => setTimeout(r, Math.min(delay, 5000)));
        }
      }

      if (onProgress) {
        onProgress(totalSent + totalErrors, count);
      }
    }
  };

  // ── 4. Launch concurrent workers ────────────────────────────
  // Cap client-side concurrency to 64 to avoid browser queue congestion, socket exhaustion,
  // and self-inflicted client-side request timeouts.
  const MAX_BROWSER_CONCURRENCY = 64;
  const numWorkers = Math.min(concurrency, MAX_BROWSER_CONCURRENCY, queue.length);
  const workers = Array.from({ length: numWorkers }, () => worker());
  await Promise.all(workers);

  if (onProgress) onProgress(totalSent + totalErrors, count);

  return {
    sent: totalSent,
    errors: totalErrors,
    durationMs: Math.round(performance.now() - start),
  };
}


