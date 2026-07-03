import { ref, onMounted, onUnmounted } from 'vue'

const POLL_INTERVAL_MS = 10000

const resolved = ref(false)
const connected = ref(false)
const info = ref(null)

let subscribers = 0
let pollTimer = null

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? ''

async function fetchInfo() {
  try {
    const res = await fetch(`${API_BASE}/api/info`)
    if (!res.ok) throw new Error('non-ok')
    const data = await res.json()
    info.value = data
    connected.value = data.connected
  } catch {
    connected.value = false
  } finally {
    resolved.value = true
  }
}

function startPolling() {
  fetchInfo()
  pollTimer = setInterval(fetchInfo, POLL_INTERVAL_MS)
}

function stopPolling() {
  clearInterval(pollTimer)
  pollTimer = null
}

export function useMQStatus() {
  onMounted(() => {
    subscribers++
    if (subscribers === 1) startPolling()
  })

  onUnmounted(() => {
    subscribers--
    if (subscribers === 0) stopPolling()
  })

  return { resolved, connected, info }
}
