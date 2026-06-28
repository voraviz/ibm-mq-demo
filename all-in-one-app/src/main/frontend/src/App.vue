<template>
  <div class="app">
    <NavBar />
    <main class="main">
      <div class="panels">
        <div class="panel-wrap">
          <PutPanel />
        </div>
        <div class="divider" />
        <div class="panel-wrap">
          <GetPanel />
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import NavBar from './components/NavBar.vue'
import PutPanel from './components/PutPanel.vue'
import GetPanel from './components/GetPanel.vue'
</script>

<style>
/* ── Google Fonts ───────────────────────────────────────────────────────── */
@import url('https://fonts.googleapis.com/css2?family=IBM+Plex+Mono&family=IBM+Plex+Sans:wght@300;400;600&display=swap');

/* ── Carbon Design System tokens ────────────────────────────────────────── */
:root {
  /* Colors */
  --cds-background: #ffffff;
  --cds-background-inverse: #161616;
  --cds-layer-01: #f4f4f4;
  --cds-layer-02: #e0e0e0;

  --cds-text-primary: #161616;
  --cds-text-secondary: #525252;
  --cds-text-placeholder: #6f6f6f;
  --cds-text-disabled: #8d8d8d;
  --cds-text-inverse: #ffffff;

  --cds-border-subtle: #c6c6c6;

  --cds-button-primary: #0f62fe;
  --cds-button-primary-hover: #0353e9;
  --cds-button-primary-active: #002d9c;
  --cds-button-disabled: #c6c6c6;

  --cds-focus: #0f62fe;
  --cds-field: #f4f4f4;

  --cds-support-error: #da1e28;
  --cds-support-success: #24a148;
  --cds-support-warning: #f1c21b;
  --cds-support-info: #0f62fe;

  --cds-link-primary: #0f62fe;
  --cds-link-primary-hover: #0043ce;

  /* Typography */
  --cds-font-family: 'IBM Plex Sans', 'Helvetica Neue', Arial, sans-serif;
  --cds-font-family-mono: 'IBM Plex Mono', Menlo, Courier, monospace;
}

/* ── Reset ──────────────────────────────────────────────────────────────── */
*, *::before, *::after {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html, body {
  height: 100%;
  font-family: var(--cds-font-family);
  background: var(--cds-background);
  color: var(--cds-text-primary);
  -webkit-font-smoothing: antialiased;
}

#app {
  height: 100%;
  display: flex;
  flex-direction: column;
}
</style>

<style scoped>
.app {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.main {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.panels {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.panel-wrap {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.divider {
  width: 1px;
  background: var(--cds-border-subtle);
  flex-shrink: 0;
}

/* Responsive: stack below 672px */
@media (max-width: 672px) {
  .panels {
    flex-direction: column;
    overflow-y: auto;
  }
  .divider {
    width: 100%;
    height: 1px;
  }
  .panel-wrap {
    overflow: visible;
    min-height: 50vh;
  }
}
</style>
