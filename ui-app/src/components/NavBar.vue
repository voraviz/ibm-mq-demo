<template>
  <nav class="navbar">
    <div class="navbar__inner">
      <span class="navbar__logo">
        <svg xmlns="http://www.w3.org/2000/svg" width="23" height="23" viewBox="0 0 23 23" fill="white" aria-label="IBM logo">
          <path d="M0 3h23v2H0zM0 6h23v2H0zM3 9h17v2H3zM3 12h17v2H3zM0 15h23v2H0zM0 18h23v2H0z"/>
        </svg>
      </span>
      <span class="navbar__title">IBM MQ Demo</span>
      <template v-if="resolved">
        <span class="navbar__divider"></span>
        <span class="navbar__conn">
          <span class="navbar__dot" :class="connected ? 'navbar__dot--ok' : 'navbar__dot--err'"></span>
          <span v-if="info" class="navbar__conn-text">{{ info.queueManager }} @ {{ info.host }}:{{ info.port }}</span>
        </span>
      </template>
    </div>
  </nav>
</template>

<script setup>
import { useMQStatus } from '../composables/useMQStatus.js'

const { resolved, connected, info } = useMQStatus()
</script>

<style scoped>
.navbar {
  background: var(--cds-background-inverse);
  height: 48px;
  display: flex;
  align-items: center;
  width: 100%;
  flex-shrink: 0;
}
.navbar__inner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 16px;
}
.navbar__logo {
  display: flex;
  align-items: center;
}
.navbar__title {
  font-family: var(--cds-font-family);
  font-size: 14px;
  font-weight: 400;
  color: var(--cds-text-inverse);
  letter-spacing: 0.16px;
}
.navbar__divider {
  width: 1px;
  height: 16px;
  background: rgba(255, 255, 255, 0.25);
  flex-shrink: 0;
}
.navbar__conn {
  display: flex;
  align-items: center;
  gap: 6px;
}
.navbar__conn-text {
  font-family: var(--cds-font-family);
  font-size: 12px;
  color: var(--cds-text-inverse);
  opacity: 0.7;
  letter-spacing: 0.16px;
}
.navbar__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.navbar__dot--ok {
  background: #42be65;
}
.navbar__dot--err {
  background: #fa4d56;
}
</style>
