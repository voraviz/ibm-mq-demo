# IBM MQ Demo Application — Implementation Plan

## Top-Level Overview

Build a full-stack IBM MQ demo application:

- **Backend**: Quarkus (JVM mode) with SmallRye Reactive Messaging + IBM MQ JMS connector, Micrometer metrics, and a WebSocket endpoint.
- **Frontend**: Vue 3 + Vite SPA bundled into `src/main/resources/META-INF/resources` and served by Quarkus as static assets.
- **UI Design**: IBM Carbon Design System aesthetic (as defined in `design.md`) — IBM Plex Sans, Gray 100 nav, Blue 60 accent, 0px border-radius, flat cards, bottom-border inputs, 8px spacing grid.
- **Layout**: Two-panel page — left panel "PUT" a message; right panel "GET" messages with a Start/Stop toggle that controls a backend reactive consumer connected to the browser via WebSocket.

Configuration (hostname, port, username, password, queue name) is externalized via `application.properties` / environment variables.

---

## Sub-Tasks

---

### Sub-Task 1 — Scaffold the Quarkus Project

**Intent**  
Create the Maven project skeleton with the correct Quarkus extensions so all subsequent sub-tasks have a valid compile target.

**Expected Outcomes**  
- `pom.xml` with Quarkus BOM, required extensions, and IBM MQ client dependency.
- `src/main/resources/application.properties` with placeholders for MQ connection, WebSocket, and Micrometer config.
- Project compiles with `./mvnw quarkus:dev` without errors.

**Todo List**  
1. Create project root with Maven wrapper (or use Quarkus CLI).
2. Add Quarkus extensions to `pom.xml`:
   - `quarkus-rest` (Jakarta REST)
   - `quarkus-messaging` (SmallRye Reactive Messaging)
   - `quarkus-messaging-jms` (JMS connector for SmallRye)
   - `quarkus-websockets-next` (WebSocket support)
   - `quarkus-micrometer-registry-prometheus` (metrics)
3. Add IBM MQ JMS client dependency: `com.ibm.mq:com.ibm.mq.allclient` (Maven Central).
4. Populate `application.properties` with the following keys and default values:

   **IBM MQ connection parameters** (read by `MQConfig` in Sub-Task 2):
   ```properties
   ibm.mq.host=localhost
   ibm.mq.port=1414
   ibm.mq.channel=DEV.APP.SVRCONN
   ibm.mq.queue-manager=QM1
   ibm.mq.username=app
   ibm.mq.password=passw0rd
   ibm.mq.queue=DEV.QUEUE.1
   ```

   **SmallRye Reactive Messaging — JMS channel bindings**:
   ```properties
   mp.messaging.outgoing.mq-put.connector=smallrye-jms
   mp.messaging.outgoing.mq-put.destination=DEV.QUEUE.1
   mp.messaging.outgoing.mq-put.destination-type=queue

   mp.messaging.incoming.mq-get.connector=smallrye-jms
   mp.messaging.incoming.mq-get.destination=DEV.QUEUE.1
   mp.messaging.incoming.mq-get.destination-type=queue
   ```

   **Micrometer / Prometheus**:
   ```properties
   quarkus.micrometer.export.prometheus.enabled=true
   ```

   **CORS (dev convenience)**:
   ```properties
   quarkus.http.cors=true
   ```
5. Verify `./mvnw compile` succeeds.

**Relevant Context**  
- [`REQUIREMENTS.md`](REQUIREMENTS.md) — IBM MQ container uses port 1414, queue manager `QM1`.
- IBM MQ AllClient artifact: `com.ibm.mq:com.ibm.mq.allclient:9.3.x.x` (use latest available on Maven Central).
- Quarkus SmallRye JMS connector requires a `javax.jms.ConnectionFactory` CDI bean — provide via `@Produces` in a config class.

**Status**: `[ ] pending`

---

### Sub-Task 2 — IBM MQ Connection Factory & JMS Configuration

**Intent**  
Create the CDI-managed `ConnectionFactory` bean that SmallRye Reactive Messaging will use for both producer and consumer channels.

**Expected Outcomes**  
- A `MQConnectionFactoryProducer` class (`@ApplicationScoped`) that reads MQ config from `application.properties` via `@ConfigProperty` and produces a `javax.jms.ConnectionFactory` bean.
- No hardcoded credentials — all values injected.
- Application connects to IBM MQ on startup (verified by checking dev logs against the running MQ container).

**Todo List**  
1. Create `src/main/java/com/example/config/MQConfig.java` — a `@ConfigProperties` (or `@ConfigProperty`-annotated `@ApplicationScoped`) bean holding host, port, channel, queue manager, username, password.
2. Create `src/main/java/com/example/config/MQConnectionFactoryProducer.java` — `@Produces @ApplicationScoped` method that builds a `MQConnectionFactory` (IBM class) and sets all properties from `MQConfig`.
3. Wire the factory to SmallRye channels in `application.properties`:
   - `mp.messaging.connector.smallrye-jms.connection-factory-jndi` → point to the CDI producer.
   - Configure outgoing channel `mq-put` and incoming channel `mq-get` with the target queue name.

**Relevant Context**  
- IBM MQ JMS class: `com.ibm.mq.jms.MQConnectionFactory`.
- SmallRye JMS connector docs: connection factory is resolved by CDI type `javax.jms.ConnectionFactory`.
- Sub-Task 1 must be complete before this sub-task.

**Status**: `[ ] pending`

---

### Sub-Task 3 — PUT REST Endpoint

**Intent**  
Expose a `POST /api/messages` endpoint that accepts a JSON body `{ "text": "..." }` and emits it onto the `mq-put` SmallRye outgoing channel, which forwards it to the IBM MQ queue.

**Expected Outcomes**  
- `POST /api/messages` with `{ "text": "hello" }` places a message on the IBM MQ queue (verifiable via MQ console at `https://localhost:9443`).
- Returns HTTP 202 Accepted with `{ "status": "sent" }`.

**Todo List**  
1. Create `src/main/java/com/example/resource/MessageResource.java` — `@Path("/api/messages")` JAX-RS resource.
2. Inject `@Channel("mq-put") Emitter<String> emitter` (SmallRye reactive emitter).
3. `POST` method accepts `MessageRequest` DTO (`{ "text": String }`), calls `emitter.send(request.text())`, returns 202.
4. Add `@Produces(MediaType.APPLICATION_JSON)` and CORS config in `application.properties` for dev mode (`quarkus.http.cors=true`).

**Relevant Context**  
- SmallRye `Emitter` API: `emitter.send(payload)` returns `CompletionStage` — await or ignore for fire-and-forget.
- Sub-Task 2 must be complete (channel must be wired to MQ).

**Status**: `[ ] pending`

---

### Sub-Task 4 — Consumer & WebSocket Endpoint

**Intent**  
Create a reactive consumer on the `mq-get` incoming channel and a WebSocket server endpoint that streams received messages to all connected browser clients. The consumer starts/stops based on a REST toggle command.

**Expected Outcomes**  
- `POST /api/consumer/start` activates the MQ incoming channel consumer.
- `POST /api/consumer/stop` pauses/stops the consumer.
- While running, each message received from the MQ queue is broadcast to all WebSocket clients at `ws://localhost:8080/ws/messages`.
- Browser clients connecting to the WebSocket receive live messages as plain-text frames.

**Todo List**  
1. Create `src/main/java/com/example/messaging/MQConsumer.java` — `@ApplicationScoped` bean with `@Incoming("mq-get")` method that processes `String` messages and notifies a WebSocket broadcaster.
2. Create `src/main/java/com/example/ws/MessageWebSocket.java` — `@ServerEndpoint("/ws/messages")` using `quarkus-websockets-next` API; maintains a set of open sessions; exposes `broadcast(String)`.
3. Create `src/main/java/com/example/resource/ConsumerResource.java` — `@Path("/api/consumer")` with `POST /start` and `POST /stop` that toggle channel consumption (use SmallRye `SubscriberBuilder` pause/resume or a simple `AtomicBoolean` gate inside the `@Incoming` handler to skip/pass messages).
4. The simplest correct implementation of stop: use an `AtomicBoolean running` flag; when false, the `@Incoming` handler receives but drops the message (does not call broadcast); when true, it broadcasts.

**Relevant Context**  
- `quarkus-websockets-next`: use `@WebSocket` annotation with injected `WebSocketConnection` broadcast API.
- True channel suspension requires SmallRye internal API which is complex — the `AtomicBoolean` gate is the minimal correct approach.
- Sub-Tasks 2 and 3 must be complete.

**Status**: `[ ] pending`

---

### Sub-Task 5 — Micrometer Metrics

**Intent**  
Enable and expose Prometheus-compatible metrics for the application (as required by the spec). No custom metrics are needed — the default Quarkus + SmallRye built-in metrics are sufficient.

**Expected Outcomes**  
- `GET /q/metrics` returns Prometheus text format with JVM, HTTP, and messaging metrics.
- No compile or runtime errors related to metrics.

**Todo List**  
1. Confirm `quarkus-micrometer-registry-prometheus` is in `pom.xml` (done in Sub-Task 1).
2. Ensure `application.properties` has `quarkus.micrometer.export.prometheus.enabled=true` (default is true when extension is present).
3. Optionally add a `@Counted` or `@Timed` annotation on `MessageResource.post()` for demo value.
4. Verify `curl http://localhost:8080/q/metrics` returns valid Prometheus output.

**Relevant Context**  
- Quarkus Micrometer auto-instruments JAX-RS endpoints and SmallRye messaging channels.
- Sub-Task 3 must be complete for HTTP metrics to appear.

**Status**: `[ ] pending`

---

### Sub-Task 6 — Vue 3 + Vite Frontend Scaffold

**Intent**  
Create the Vue 3 + Vite frontend project inside `src/main/frontend/`. Configure Vite to build output into `src/main/resources/META-INF/resources` so Quarkus serves the SPA as static files.

**Expected Outcomes**  
- `npm run build` inside `src/main/frontend/` produces `src/main/resources/META-INF/resources/index.html` + assets.
- Maven build triggers the frontend build automatically via `frontend-maven-plugin`.
- Accessing `http://localhost:8080` serves the Vue app (Quarkus dev mode).

**Todo List**  
1. Create `src/main/frontend/` with `npm create vue@latest` (Vue 3, Vite, no TypeScript required, no router needed for single-page).
2. Configure `vite.config.js` `build.outDir` to `../resources/META-INF/resources` and `build.emptyOutDir: true`.
3. Set Vite dev server `proxy` for `/api` and `/ws` → `http://localhost:8080` (for dev mode).
4. Add `frontend-maven-plugin` to `pom.xml` to run `npm install` + `npm run build` during Maven `generate-resources` phase.
5. Add `src/main/resources/META-INF/resources/` to `.gitignore` (build artifact).

**Relevant Context**  
- `frontend-maven-plugin` (eirslett): installs Node/npm locally and runs scripts — no system Node required.
- Sub-Task 1 must be complete (pom.xml must exist).

**Status**: `[ ] pending`

---

### Sub-Task 7 — UI Implementation (Carbon Design)

**Intent**  
Build the single-page Vue 3 UI with two panels (PUT left, GET right) following the IBM Carbon Design System aesthetic defined in [`design.md`](design.md).

**Expected Outcomes**  
- Left panel: a textarea input + "Send Message" primary button. POST to `/api/messages`. Shows success/error notification.
- Right panel: a scrollable message list with a "Start Consumer" / "Stop Consumer" toggle button. Connects to WebSocket at `/ws/messages`. New messages appear in real-time.
- Visual fidelity: IBM Plex Sans font (via Google Fonts CDN), Gray 100 `#161616` navbar, Blue 60 `#0f62fe` buttons, `#f4f4f4` card backgrounds, `0px` border-radius, bottom-border inputs.
- Responsive: stacks vertically below `672px`.

**Todo List**  
1. Install IBM Plex Sans via `@import` in `App.vue` global CSS (Google Fonts CDN).
2. Define CSS custom properties (`--cds-*`) in `App.vue` `<style>` matching `design.md` color tokens.
3. Create `NavBar.vue` — `#161616` background, 48px height, "IBM MQ Demo" title in IBM Plex Sans weight 300 white.
4. Create `PutPanel.vue` — textarea (bottom-border style, `#f4f4f4` bg), "Send Message" primary button, notification area for success/error.
5. Create `GetPanel.vue` — "Start Consumer" / "Stop Consumer" toggle button (primary/danger), scrollable message list (each message as a Gray 10 tile with `Code 01` monospace font), WebSocket connection logic.
6. Wire `PutPanel` → `POST /api/messages` via `fetch`.
7. Wire `GetPanel` → `POST /api/consumer/start|stop` + `new WebSocket('/ws/messages')` for live message stream.
8. Main layout: flex row with `NavBar` at top, two equal panels below; collapse to column at `672px`.

**Relevant Context**  
- [`design.md`](design.md) — full token reference and component spec.
- IBM Plex Sans Google Fonts URL: `https://fonts.googleapis.com/css2?family=IBM+Plex+Mono&family=IBM+Plex+Sans:wght@300;400;600&display=swap`
- WebSocket path must match Sub-Task 4 `@WebSocket` endpoint path `/ws/messages`.
- Sub-Tasks 4, 5, and 6 must be complete.

**Status**: `[ ] pending`

---

## Dependency Order

```
Sub-Task 1 (Scaffold)
  └── Sub-Task 2 (MQ Connection Factory)
        ├── Sub-Task 3 (PUT REST Endpoint)
        │     └── Sub-Task 5 (Metrics)
        └── Sub-Task 4 (Consumer + WebSocket)
Sub-Task 1 (Scaffold)
  └── Sub-Task 6 (Vue Frontend Scaffold)
        └── Sub-Task 7 (UI Implementation)
```

Sub-Tasks 6 and 7 are frontend work and can be developed in parallel with Sub-Tasks 3–5 once Sub-Task 1 is done.
