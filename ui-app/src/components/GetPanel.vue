<template>
  <div class="panel">
    <div class="panel__header">
      <h2 class="panel__title">Get Messages</h2>
      <p class="panel__desc">Live stream from the IBM MQ queue via WebSocket.</p>
    </div>

    <div class="panel__controls">
      <button
        class="btn"
        :class="running ? 'btn--danger' : 'btn--primary'"
        @click="toggleConsumer"
        :disabled="toggling"
      >
        {{ running ? 'Stop Consumer' : 'Start Consumer' }}
      </button>
      <span class="status-badge" :class="running ? 'status-badge--running' : 'status-badge--stopped'">
        {{ running ? 'Running' : 'Stopped' }}
      </span>
      <button class="btn btn--ghost" @click="clearMessages" title="Clear messages">
        Clear
      </button>
    </div>

    <div class="message-list" ref="listEl">
      <div v-if="messages.length === 0" class="message-list__empty">
        No messages yet. Start the consumer and put a message.
      </div>
      <div
        v-for="(msg, idx) in messages"
        :key="idx"
        class="message-tile"
      >
        <span class="message-tile__index">#{{ messages.length - idx }}</span>
        <pre class="message-tile__text">{{ msg.text }}</pre>
        <span class="message-tile__time">{{ msg.time }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? ''

// Derive WebSocket base from API_BASE:
//   http://host:port  → ws://host:port
//   https://host:port → wss://host:port
//   '' (empty, dev proxy mode) → use current page host
function wsBase() {
  if (!API_BASE) return ''
  return API_BASE.replace(/^https:\/\//, 'wss://').replace(/^http:\/\//, 'ws://')
}

const running = ref(false)
const toggling = ref(false)
const messages = ref([])
const listEl = ref(null)

let ws = null

onMounted(async () => {
  try {
    const res = await fetch(`${API_BASE}/api/consumer/status`)
    if (res.ok) {
      const data = await res.json()
      if (data.status === 'running') {
        running.value = true
        await connectWebSocket()
      }
    }
  } catch (err) {
    console.error('Failed to fetch consumer status:', err)
  }
})

async function toggleConsumer() {
  toggling.value = true
  const action = running.value ? 'stop' : 'start'
  try {
    if (action === 'start') {
      // Connect WebSocket first and wait for it to be open,
      // so no messages are missed when the consumer begins draining the queue.
      await connectWebSocket()
      const res = await fetch(`${API_BASE}/api/consumer/start`, { method: 'POST' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      running.value = true
    } else {
      const res = await fetch(`${API_BASE}/api/consumer/stop`, { method: 'POST' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      running.value = false
      disconnectWebSocket()
    }
  } catch (err) {
    console.error('Consumer toggle failed:', err)
    disconnectWebSocket()
  } finally {
    toggling.value = false
  }
}

function connectWebSocket() {
  // If already open, resolve immediately.
  if (ws && ws.readyState === WebSocket.OPEN) return Promise.resolve()
  // If connecting, wait for it to open.
  if (ws && ws.readyState === WebSocket.CONNECTING) {
    return new Promise((resolve, reject) => {
      ws.addEventListener('open', () => resolve(), { once: true })
      ws.addEventListener('error', () => reject(new Error('WebSocket error')), { once: true })
    })
  }
  // Otherwise create a new connection and wait for open.
  return new Promise((resolve, reject) => {
    const base = wsBase()
    const wsUrl = base
      ? `${base}/ws/messages`
      : `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws/messages`
    ws = new WebSocket(wsUrl)
    ws.onmessage = (event) => {
      const now = new Date().toLocaleTimeString()
      messages.value.unshift({ text: event.data, time: now })
      nextTick(() => {
        if (listEl.value) listEl.value.scrollTop = 0
      })
    }
    ws.onerror = (e) => {
      console.error('WebSocket error', e)
      reject(new Error('WebSocket error'))
    }
    ws.onclose = () => {
      if (running.value) {
        // reconnect after brief delay if still supposed to be running
        setTimeout(connectWebSocket, 1000)
      }
    }
    ws.onopen = () => resolve()
  })
}

function disconnectWebSocket() {
  if (ws) {
    ws.close()
    ws = null
  }
}

function clearMessages() {
  messages.value = []
}

onUnmounted(() => {
  disconnectWebSocket()
})
</script>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.panel__header {
  padding: 24px 24px 0;
  border-bottom: 1px solid var(--cds-border-subtle);
  padding-bottom: 16px;
}
.panel__title {
  font-family: var(--cds-font-family);
  font-size: 20px;
  font-weight: 400;
  color: var(--cds-text-primary);
  margin: 0 0 4px;
}
.panel__desc {
  font-family: var(--cds-font-family);
  font-size: 14px;
  color: var(--cds-text-secondary);
  letter-spacing: 0.16px;
  margin: 0;
}
.panel__controls {
  padding: 16px 24px;
  display: flex;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid var(--cds-border-subtle);
  flex-shrink: 0;
}

/* Status badge */
.status-badge {
  font-family: var(--cds-font-family);
  font-size: 12px;
  font-weight: 400;
  padding: 4px 8px;
  border-radius: 24px;
}
.status-badge--running {
  background: #defbe6;
  color: #044317;
}
.status-badge--stopped {
  background: var(--cds-layer-01);
  color: var(--cds-text-secondary);
}

/* Message list */
.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.message-list__empty {
  font-family: var(--cds-font-family);
  font-size: 14px;
  color: var(--cds-text-placeholder);
  letter-spacing: 0.16px;
  text-align: center;
  padding: 48px 0;
}
.message-tile {
  background: var(--cds-layer-01);
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  border-radius: 0;
}
.message-tile:hover {
  background: #e8e8e8;
}
.message-tile__index {
  font-family: var(--cds-font-family);
  font-size: 12px;
  font-weight: 400;
  letter-spacing: 0.32px;
  color: var(--cds-text-secondary);
}
.message-tile__text {
  font-family: var(--cds-font-family-mono);
  font-size: 14px;
  letter-spacing: 0.16px;
  line-height: 1.43;
  color: var(--cds-text-primary);
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}
.message-tile__time {
  font-family: var(--cds-font-family);
  font-size: 12px;
  letter-spacing: 0.32px;
  color: var(--cds-text-placeholder);
}

/* Buttons */
.btn {
  font-family: var(--cds-font-family);
  font-size: 14px;
  font-weight: 400;
  letter-spacing: 0.16px;
  height: 48px;
  padding: 14px 15px;
  border: 1px solid transparent;
  border-radius: 0;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  transition: background 0.1s;
}
.btn--primary {
  background: var(--cds-button-primary);
  color: #ffffff;
  padding: 14px 63px 14px 15px;
}
.btn--primary:hover:not(:disabled) { background: var(--cds-button-primary-hover); }
.btn--primary:active:not(:disabled) { background: var(--cds-button-primary-active); }
.btn--danger {
  background: var(--cds-support-error);
  color: #ffffff;
  padding: 14px 63px 14px 15px;
}
.btn--danger:hover:not(:disabled) { background: #b81921; }
.btn--danger:active:not(:disabled) { background: #750e13; }
.btn--ghost {
  background: transparent;
  color: var(--cds-button-primary);
  border: 1px solid var(--cds-button-primary);
  padding: 14px 16px;
}
.btn--ghost:hover { background: #edf5ff; }
.btn:disabled {
  background: var(--cds-button-disabled);
  color: var(--cds-text-disabled);
  cursor: not-allowed;
}
</style>
