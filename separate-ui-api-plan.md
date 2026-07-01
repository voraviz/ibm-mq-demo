# Plan: Separate UI and API into Two Applications

## Top-Level Overview

The existing `all-in-one-app/` is a monolithic Quarkus project that bundles a Vue 3 frontend alongside the IBM MQ backend. The goal is to split this into two independent, separately-deployable applications:

1. **`api-app/`** — A new Quarkus REST microservice (port **8081**) containing all IBM MQ logic and REST/WebSocket endpoints. The all-in-one-app stays untouched as reference.
2. **`ui-app/`** — A new standalone Vite + Vue 3 project (port **8080**) replicating the existing Vue UI. It uses a `VITE_API_BASE_URL` environment variable (default `http://localhost:8081`) to reach the API. In development, the Vite dev server proxies `/api` and `/ws` to port 8081.

The `all-in-one-app/` directory is **not modified** — it remains for reference only.

---

## Sub-Tasks

---

### Sub-Task 1: Create the `api-app/` Quarkus REST microservice

**Status**: [ ] pending

**Intent**  
Create a brand-new Quarkus Maven project at `api-app/` that contains only the backend IBM MQ logic — no Vue, no frontend Maven plugin. This is derived from `all-in-one-app/` with the frontend pieces stripped out and the port changed to 8081.

**Expected Outcomes**
- `api-app/` exists as a valid Quarkus Maven project
- All four backend files are present: `MQConfig`, `MQConnectionFactoryProducer`, `MQConsumer`, `MessageResource`, `ConsumerResource`, `MessageWebSocket`
- Default HTTP port is 8081
- CORS is enabled for all origins (so the UI on port 8080 can call it)
- No `frontend-maven-plugin`, no Vite build step, no `META-INF/resources` Vue files
- `mvn package` produces a runnable JAR

**Todo List**
1. Create `api-app/pom.xml` — copy dependencies from `all-in-one-app/pom.xml`, remove the `frontend-maven-plugin` execution block entirely, keep all IBM MQ, Quarkus REST, SmallRye Messaging, WebSocket, and Micrometer dependencies.
2. Create the Java source tree under `api-app/src/main/java/com/example/` — copy these five files verbatim from `all-in-one-app/`:
   - `config/MQConfig.java`
   - `config/MQConnectionFactoryProducer.java`
   - `messaging/MQConsumer.java`
   - `resource/MessageResource.java`
   - `resource/ConsumerResource.java`
   - `ws/MessageWebSocket.java`
3. Create `api-app/src/main/resources/application.properties` — copy from `all-in-one-app/` and change `quarkus.http.port=8081`. Keep all `ibm.mq.*`, `mp.messaging.*`, `quarkus.http.cors.*`, and `quarkus.micrometer.*` settings. Remove any static file or Vue-related config.
4. Create `api-app/src/test/java/` placeholder (empty, or copy any existing tests).

**Relevant Context**
- Reference: [`all-in-one-app/pom.xml`](all-in-one-app/pom.xml)
- Reference: [`all-in-one-app/src/main/java/com/example/`](all-in-one-app/src/main/java/com/example/)
- Reference: [`all-in-one-app/src/main/resources/application.properties`](all-in-one-app/src/main/resources/application.properties)
- The `frontend-maven-plugin` block in the all-in-one pom.xml must be **omitted** entirely in `api-app/pom.xml`
- CORS must allow `http://localhost:8080` (the UI origin) — set `quarkus.http.cors.origins=*` or `http://localhost:8080`

---

### Sub-Task 2: Create the `ui-app/` standalone Vue 3 + Vite project

**Status**: [ ] pending

**Intent**  
Create a new standalone Vite + Vue 3 project at `ui-app/` that replicates the existing frontend from `all-in-one-app/src/main/frontend/`. It must be a pure Node/npm project (no Quarkus), run on port 8080, and communicate with the API on port 8081 via a configurable base URL.

**Expected Outcomes**
- `ui-app/` is a valid standalone Vite + Vue 3 npm project
- Runs on port 8080 by default (`vite.config.js` sets `server.port: 8080`)
- All three Vue components are present and functionally identical to the all-in-one-app versions: `NavBar.vue`, `PutPanel.vue`, `GetPanel.vue`
- `App.vue` and `main.js` are identical (or near-identical) to the all-in-one-app versions
- API calls use `VITE_API_BASE_URL` environment variable, defaulting to `http://localhost:8081`
- WebSocket URL is derived from `VITE_API_BASE_URL` (replace `http` → `ws`)
- Vite dev server proxies `/api` → `http://localhost:8081` and `/ws` → `ws://localhost:8081` so relative-path calls also work in dev
- `npm run dev` starts the dev server on port 8080
- `npm run build` produces a `dist/` folder of static files
- A `.env` file (or `.env.example`) documents the `VITE_API_BASE_URL` variable

**Todo List**
1. Create `ui-app/package.json` — replicate from `all-in-one-app/src/main/frontend/package.json`. Ensure `name` is `mq-ui`, version is `1.0.0`, dev/prod dependencies include `vue`, `@vitejs/plugin-vue`, `vite`.
2. Create `ui-app/index.html` — copy from `all-in-one-app/src/main/frontend/index.html`.
3. Create `ui-app/vite.config.js`:
   - Plugin: `@vitejs/plugin-vue`
   - Dev server port: **8080**
   - Dev server proxy: `/api` → `http://localhost:8081`, `/ws` → `ws://localhost:8081` (ws: true)
   - Build output: default `dist/` (no special outDir)
4. Create `ui-app/src/main.js` — copy from `all-in-one-app/src/main/frontend/src/main.js`.
5. Create `ui-app/src/App.vue` — copy from all-in-one-app. No changes needed (layout is the same).
6. Create `ui-app/src/components/NavBar.vue` — copy verbatim.
7. Create `ui-app/src/components/PutPanel.vue` — copy and update API calls:
   - Replace hardcoded `/api/messages` with `${import.meta.env.VITE_API_BASE_URL ?? ''}/api/messages`
   - (Using empty string fallback so Vite proxy works in dev if no env var is set)
8. Create `ui-app/src/components/GetPanel.vue` — copy and update API/WebSocket URLs:
   - REST calls: `${import.meta.env.VITE_API_BASE_URL ?? ''}/api/consumer/...`
   - WebSocket URL: derive from `VITE_API_BASE_URL` — replace `http://` with `ws://` and `https://` with `wss://`, or default to empty string for proxy
9. Create `ui-app/.env` with `VITE_API_BASE_URL=http://localhost:8081` as the default.
10. Create `ui-app/.env.example` documenting the variable.
11. Create `ui-app/public/` directory (empty or with a favicon if present in all-in-one-app).

**Relevant Context**
- Reference frontend: [`all-in-one-app/src/main/frontend/`](all-in-one-app/src/main/frontend/)
- Components to replicate: [`PutPanel.vue`](all-in-one-app/src/main/frontend/src/components/PutPanel.vue), [`GetPanel.vue`](all-in-one-app/src/main/frontend/src/components/GetPanel.vue), [`NavBar.vue`](all-in-one-app/src/main/frontend/src/components/NavBar.vue)
- Design spec: [`design.md`](design.md) — IBM Carbon Design System tokens, typography, colors
- The `VITE_API_BASE_URL` env var approach: in development without the var set, Vite proxy handles `/api` and `/ws` transparently; with it set (e.g. in production), the full URL is used directly

---

### Sub-Task 3: Add README and verify project structure

**Status**: [ ] pending

**Intent**  
Provide clear developer instructions for each new application and confirm the top-level project layout is clean and correct.

**Expected Outcomes**
- `api-app/README.md` explains how to build and run the API (prerequisites, `mvn quarkus:dev`, port, env vars)
- `ui-app/README.md` explains how to install dependencies and run the UI (`npm install`, `npm run dev`, port, env vars)
- Root-level `README.md` (or existing one) is updated to describe the three-app structure
- Repo structure is:
  ```
  ibm_mq/
  ├── all-in-one-app/   # Original reference app (unchanged)
  ├── api-app/          # New Quarkus API microservice (port 8081)
  ├── ui-app/           # New Vue 3 UI (port 8080)
  ├── design.md
  ├── REQUIREMENTS.md
  └── README.md
  ```

**Todo List**
1. Create `api-app/README.md` — document prerequisites (Java 17+, Maven, running IBM MQ container), how to run (`mvn quarkus:dev`), and the port (8081).
2. Create `ui-app/README.md` — document prerequisites (Node 20+, npm), how to install and run (`npm install && npm run dev`), the port (8080), and the `VITE_API_BASE_URL` env var.
3. Update root [`README.md`](README.md) to add a section describing the three apps and how to start them together.

**Relevant Context**
- Existing [`README.md`](README.md) documents MQ container startup — preserve that content, only add to it
