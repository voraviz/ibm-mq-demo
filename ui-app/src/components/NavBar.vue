<template>
  <nav class="navbar">
    <div class="navbar__inner">
      <span class="navbar__logo">
        <svg xmlns="http://www.w3.org/2000/svg" width="23" height="23" viewBox="0 0 23 23" fill="white" aria-label="IBM logo">
          <path d="M0 3h23v2H0zM0 6h23v2H0zM3 9h17v2H3zM3 12h17v2H3zM0 15h23v2H0zM0 18h23v2H0z"/>
        </svg>
      </span>
      <span class="navbar__title">IBM MQ Demo</span>
    </div>
  </nav>
</template>

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
</style>
