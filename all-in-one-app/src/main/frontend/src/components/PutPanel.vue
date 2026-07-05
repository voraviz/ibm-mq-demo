<template>
  <div class="panel">
    <div class="panel__header">
      <div class="panel__title-row">
        <h2 class="panel__title">Put Message</h2>
        <span class="count-badge">Session: {{ sessionCount }} | Total: {{ totalCount }}</span>
      </div>
      <p class="panel__desc">Send one or more messages to the IBM MQ queue.</p>
    </div>

    <div class="panel__controls">
      <!-- Message text -->
      <div class="field">
        <label class="field__label" for="msg-input">Message</label>
        <textarea
          id="msg-input"
          class="field__textarea"
          v-model="text"
          placeholder="Type your message here…"
          rows="4"
          :disabled="sending"
        />
      </div>

      <!-- Count + Delay row -->
      <div class="field-row">
        <div class="field field--sm">
          <label class="field__label" for="msg-count">Number of messages</label>
          <input
            id="msg-count"
            type="number"
            class="field__input"
            v-model.number="count"
            min="1"
            max="1000"
            :disabled="sending"
          />
        </div>
        <div class="field field--sm">
          <label class="field__label" for="msg-delay">Delay between messages (ms)</label>
          <input
            id="msg-delay"
            type="number"
            class="field__input"
            v-model.number="delayMs"
            min="0"
            max="60000"
            step="100"
            :disabled="sending"
          />
        </div>
      </div>

      <!-- Progress bar (visible while sending multiple) -->
      <div v-if="sending && count > 1" class="progress">
        <div class="progress__bar" :style="{ width: progressPct + '%' }" />
        <span class="progress__label">{{ sentCount }} / {{ count }} sent</span>
      </div>

      <!-- MQ disconnected warning -->
      <div v-if="resolved && !connected" class="notification notification--error">
        IBM MQ is unreachable — messages cannot be sent.
      </div>

      <!-- Action buttons -->
      <div class="btn-row">
        <button
          class="btn btn--primary"
          :disabled="sending || !text.trim() || !connected"
          @click="send"
        >
          {{ sending ? 'Sending…' : count > 1 ? `Send ${count} Messages` : 'Send Message' }}
        </button>
        <button
          v-if="sending"
          class="btn btn--danger"
          @click="cancel"
        >
          Cancel
        </button>
        <button class="btn btn--ghost" @click="clearMessages" title="Clear messages">
          Clear
        </button>
      </div>

      <!-- Notification -->
      <div v-if="notification" :class="['notification', `notification--${notification.type}`]">
        {{ notification.message }}
      </div>
    </div>

    <!-- Sent message list -->
    <div class="message-list" ref="listEl">
      <div v-if="messages.length === 0" class="message-list__empty">
        No messages sent yet.
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
import { ref, computed, nextTick, onMounted } from 'vue'
import { useMQStatus } from '../composables/useMQStatus.js'

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? ''

const { resolved, connected } = useMQStatus()

const text       = ref('')
const count      = ref(1)
const delayMs    = ref(0)
const sending    = ref(false)
const sentCount  = ref(0)
const messages   = ref([])
const listEl     = ref(null)
const notification = ref(null)
const totalCount   = ref(0)

const sessionCount = computed(() => messages.value.length)

let notifTimer   = null
let cancelSignal = false

const progressPct = computed(() =>
  count.value > 0 ? Math.round((sentCount.value / count.value) * 100) : 0
)

function showNotification(type, message) {
  notification.value = { type, message }
  clearTimeout(notifTimer)
  notifTimer = setTimeout(() => (notification.value = null), 5000)
}

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

async function fetchTotalCount() {
  try {
    const res = await fetch(`${API_BASE}/api/messages/count`)
    if (res.ok) {
      const data = await res.json()
      totalCount.value = data.count
    }
  } catch (err) {
    // non-critical — ignore
  }
}

onMounted(fetchTotalCount)

function clearMessages() {
  messages.value = []
}

async function sendOne() {
  const res = await fetch(`${API_BASE}/api/messages`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text: text.value }),
  })

  const data = await res.json().catch(() => ({}))

  if (!res.ok) {
    throw new Error(data.message || `HTTP ${res.status}`)
  }

  const now = new Date().toLocaleTimeString()
  messages.value.unshift({ text: data.message, time: now })
  nextTick(() => {
    if (listEl.value) listEl.value.scrollTop = 0
  })
  fetchTotalCount()
}

async function send() {
  if (!text.value.trim()) return

  sending.value      = true
  sentCount.value    = 0
  cancelSignal       = false
  notification.value = null

  const total = Math.max(1, Math.floor(count.value))

  try {
    for (let i = 0; i < total; i++) {
      if (cancelSignal) {
        showNotification('warning', `Cancelled after ${sentCount.value} of ${total} messages.`)
        return
      }

      await sendOne()
      sentCount.value = i + 1

      if (i < total - 1 && delayMs.value > 0) {
        await sleep(delayMs.value)
      }
    }

    if (total === 1) {
      showNotification('success', 'Message sent successfully.')
    } else {
      showNotification('success', `${total} messages sent successfully.`)
    }
  } catch (err) {
    showNotification('error', err.message || 'Network error — could not reach the server.')
  } finally {
    sending.value = false
  }
}

function cancel() {
  cancelSignal = true
}
</script>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* Header */
.panel__header {
  padding: 24px 24px 16px;
  border-bottom: 1px solid var(--cds-border-subtle);
}
.panel__title-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
}
.count-badge {
  font-family: var(--cds-font-family);
  font-size: 12px;
  font-weight: 400;
  letter-spacing: 0.32px;
  color: var(--cds-text-secondary);
  background: var(--cds-layer-01);
  padding: 2px 8px;
  border-radius: 12px;
  white-space: nowrap;
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
  font-weight: 400;
  color: var(--cds-text-secondary);
  letter-spacing: 0.16px;
  margin: 0;
}

/* Controls (form area) */
.panel__controls {
  padding: 16px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border-bottom: 1px solid var(--cds-border-subtle);
  flex-shrink: 0;
}

/* Field */
.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.field--sm {
  flex: 1;
}
.field__label {
  font-family: var(--cds-font-family);
  font-size: 12px;
  font-weight: 400;
  letter-spacing: 0.32px;
  color: var(--cds-text-secondary);
}
.field__textarea,
.field__input {
  background: var(--cds-field);
  color: var(--cds-text-primary);
  font-family: var(--cds-font-family-mono);
  font-size: 14px;
  letter-spacing: 0.16px;
  line-height: 1.43;
  border: none;
  border-bottom: 2px solid transparent;
  border-radius: 0;
  padding: 12px 16px;
  outline: none;
  transition: border-bottom-color 0.1s;
  width: 100%;
  box-sizing: border-box;
}
.field__textarea {
  resize: vertical;
}
.field__input {
  height: 40px;
  padding: 0 16px;
}
.field__textarea::placeholder,
.field__input::placeholder {
  color: var(--cds-text-placeholder);
}
.field__textarea:focus,
.field__input:focus {
  border-bottom-color: var(--cds-focus);
}
.field__textarea:disabled,
.field__input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Side-by-side field row */
.field-row {
  display: flex;
  gap: 16px;
}
@media (max-width: 480px) {
  .field-row { flex-direction: column; }
}

/* Progress bar */
.progress {
  position: relative;
  height: 8px;
  background: var(--cds-layer-02);
  border-radius: 0;
  overflow: hidden;
}
.progress__bar {
  height: 100%;
  background: var(--cds-button-primary);
  transition: width 0.2s ease;
}
.progress__label {
  position: absolute;
  top: 12px;
  left: 0;
  font-family: var(--cds-font-family);
  font-size: 12px;
  letter-spacing: 0.32px;
  color: var(--cds-text-secondary);
}

/* Button row */
.btn-row {
  display: flex;
  gap: 8px;
  align-items: center;
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
.btn--primary:disabled {
  background: var(--cds-button-disabled);
  color: var(--cds-text-disabled);
  cursor: not-allowed;
}
.btn--danger {
  background: var(--cds-support-error);
  color: #ffffff;
  padding: 14px 63px 14px 15px;
}
.btn--danger:hover { background: #b81921; }
.btn--danger:active { background: #750e13; }
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

/* Notification */
.notification {
  font-family: var(--cds-font-family);
  font-size: 14px;
  letter-spacing: 0.16px;
  padding: 12px 16px;
  border-left: 4px solid;
}
.notification--success {
  background: #defbe6;
  color: #044317;
  border-left-color: var(--cds-support-success);
}
.notification--error {
  background: #fff1f1;
  color: #750e13;
  border-left-color: var(--cds-support-error);
}
.notification--warning {
  background: #fdf6dd;
  color: #4d3600;
  border-left-color: var(--cds-support-warning);
}

/* Message list — identical to GetPanel */
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
</style>
