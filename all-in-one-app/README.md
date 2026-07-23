# All-in-One IBM MQ Demo

A single Quarkus application that demonstrates **putting** and **getting**
messages to/from IBM MQ, with a live Vue 3 UI served from the same process.
Messages you send are enqueued to a queue; a background consumer reads them back
and streams them to the browser over WebSocket in real time.

The "all-in-one" name means one deployable artifact contains everything: the REST
API, the JMS producer/consumer, the WebSocket endpoint, and the compiled
front-end (bundled into the JAR as static resources).

---

## What it does

```
                         ┌─────────────────────────────────────────────┐
                         │            Quarkus app (:8080)               │
  Browser (Vue SPA)      │                                              │
  ┌───────────────┐      │  REST /api/messages ──emitter──┐             │
  │  PUT panel    │─POST─┼─▶ (SmallRye Reactive Messaging) │            │
  │               │      │                                 ▼            │      ┌──────────┐
  │               │      │                          smallrye-jms ──PUT──┼─────▶│          │
  │               │      │                                              │      │  IBM MQ  │
  │  GET panel    │◀─WS──┼── MessageWebSocket ◀─ MQConsumer ◀───GET─────┼──────│  queue   │
  │  (live feed)  │      │   (broadcast)         (background thread)    │      │          │
  │               │      │                                              │      └──────────┘
  │  status chip  │◀poll─┼── REST /api/info (connectivity probe)        │
  └───────────────┘      └─────────────────────────────────────────────┘
```

- **Send (PUT):** the UI POSTs text to `/api/messages`. The resource hands the
  message to a SmallRye Reactive Messaging `Emitter`, and the `smallrye-jms`
  connector puts it on the configured queue.
- **Receive (GET):** a background consumer thread (`MQConsumer`) reads messages
  from the same queue and broadcasts each one to all connected browsers over the
  `/ws/messages` WebSocket.
- **Status:** the UI polls `/api/info` every 10s to show whether the app is
  connected to MQ and which queue manager/host it resolved to.

> In the default config, PUT and GET use the **same** queue
> (`DEV.DEMO.QL.IN`), so a message you send comes straight back into the live
> feed — an easy end-to-end round-trip demo.

---

## Tech stack

| Layer | Technology |
|-------|-----------|
| Runtime | Quarkus 3.37.x, Java 25 |
| Messaging | SmallRye Reactive Messaging + `smallrye-jms` connector, IBM MQ Jakarta JMS client 9.4.3.0 |
| REST / API docs | Quarkus REST (Jackson) + SmallRye OpenAPI / Swagger UI |
| Realtime | Quarkus WebSockets Next |
| Frontend | Vue 3 + Vite (built by `frontend-maven-plugin`, bundled into the JAR) |
| Observability | Micrometer + Prometheus, OpenTelemetry (OTLP) |
| Build / packaging | Maven, multi-arch container image |

---

## Project structure

```
all-in-one-app/
├── pom.xml                         # Maven build (backend + frontend + container)
├── build_jvm_container-multi-arch.sh
├── src/main/
│   ├── java/com/example/
│   │   ├── config/
│   │   │   ├── MQConfig.java                     # typed @ConfigMapping (ibm.mq.*)
│   │   │   ├── MQConnectionFactoryProducer.java  # produces JMS ConnectionFactory(ies)
│   │   │   └── ProbeConnection.java              # CDI qualifier for the /info probe factory
│   │   ├── messaging/
│   │   │   └── MQConsumer.java                   # background GET loop + reconnect
│   │   ├── resource/
│   │   │   ├── MessageResource.java              # POST /api/messages (PUT), counts
│   │   │   ├── ConsumerResource.java             # start/stop/status/count of consumer
│   │   │   └── InfoResource.java                 # GET /api/info (connectivity probe)
│   │   └── ws/
│   │       └── MessageWebSocket.java             # /ws/messages broadcast
│   ├── resources/
│   │   ├── application.properties               # all configuration
│   │   └── META-INF/resources/                  # built frontend lands here (served statically)
│   ├── frontend/                                # Vue 3 + Vite source
│   └── docker/Dockerfile.hummingbird            # runtime image
└── src/test/java/…                              # (currently empty)
```

---

## Configuration

All configuration lives in `src/main/resources/application.properties`. MQ
settings are bound to the typed `MQConfig` interface via `@ConfigMapping(prefix =
"ibm.mq")`.

### MQ connection

```properties
ibm.mq.connection-list=localhost(1414),localhost(1415),localhost(1416)
ibm.mq.application-name=all-in-one
ibm.mq.channel=DEV.APP.SVRCONN
ibm.mq.queue-manager=QM1
ibm.mq.username=app
ibm.mq.password=passw0rd
ibm.mq.queue=DEV.DEMO.QL.IN
```

Two mutually exclusive ways to locate the queue manager:

- **Connection list** (default): `ibm.mq.connection-list` + `ibm.mq.channel`.
- **CCDT**: uncomment `ibm.mq.ccdt-url` (e.g. `file:../ccdt/ccdt.cluster.json`).
  When a CCDT URL is set, the connection list and channel are ignored.

### Two connection factories

`MQConnectionFactoryProducer` produces **two** `ConnectionFactory` beans that
share the same target/credentials but differ in reconnect policy:

| Factory | Used by | Reconnect |
|---------|---------|-----------|
| default `connectionFactory()` | consumer + outbound producer | `WMQ_CLIENT_RECONNECT`, 30s timeout — long-lived connections survive QM outages |
| `@ProbeConnection` factory | `/api/info` only | `WMQ_CLIENT_RECONNECT_DISABLED` — a point-in-time probe that fails fast |

### Outbound messaging channel

```properties
mp.messaging.outgoing.mq-put.connector=smallrye-jms
mp.messaging.outgoing.mq-put.destination=${ibm.mq.queue}
mp.messaging.outgoing.mq-put.destination-type=queue
```

### HTTP / observability

```properties
quarkus.http.port=8080
quarkus.micrometer.export.prometheus.enabled=true
quarkus.otel.enabled=true
quarkus.otel.sdk.disabled=true               # flip to false to export traces/metrics
quarkus.otel.exporter.otlp.endpoint=http://localhost:4317
```

---

## API reference

Interactive docs (dev mode): **http://localhost:8080/q/swagger-ui**

### Messages — PUT

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/messages` | Send a text message to MQ. Body: `{ "text": "..." }`. Returns `202` with the enqueued body once the put is acknowledged, `400` if blank, or `500` if the enqueue fails. |
| `GET`  | `/api/messages/count` | Number of messages sent since startup. |

### Consumer — GET control

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/consumer/start` | Start the background consumer thread. |
| `POST` | `/api/consumer/stop` | Stop the consumer. |
| `GET`  | `/api/consumer/status` | `running` or `stopped`. |
| `GET`  | `/api/consumer/count` | Number of messages consumed since startup. |

### Info

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/info` | Connectivity probe → `{ queueManager, host, port, connected }`. Bounded to ~3s; reports `connected=false` on timeout. |

### WebSocket

| Path | Description |
|------|-------------|
| `/ws/messages` | Server pushes each consumed message as a text frame to all open connections. |

### Ops endpoints

- Prometheus metrics: `/q/metrics`
- Health / dev UI (dev mode): `/q/dev`

---

## How the message flow works (detail)

**PUT path** — `MessageResource.putMessage()`
1. Validates the request (`400` if `text` is blank).
2. Prefixes a running counter (`[#N] <text>`) and calls `emitter.send(body)`.
3. The `mq-put` outgoing channel (`smallrye-jms`) puts it on the queue.
4. Awaits the emitter acknowledgement (via `Uni<Response>`, off the I/O thread)
   and responds `202 Accepted` on success or `500` if the enqueue fails; on
   failure the sent-counter is rolled back.

**GET path** — `MQConsumer`
1. `start()` opens a JMS connection/session/consumer and launches a daemon
   thread (`mq-consumer`).
2. `readLoop()` calls `receive(500)`; each `TextMessage` is logged, broadcast to
   WebSocket clients, and counted.
3. On an unexpected `JMSException` it enters `reconnectLoop()` (retry every 1s)
   until reconnected or stopped. Unexpected runtime errors are logged and the
   loop continues, so one bad WebSocket send can't kill the consumer.
4. `stop()` / `@PreDestroy` interrupt the thread and close JMS resources.

**Broadcast** — `MessageWebSocket.broadcast()` sends to each open connection
fire-and-forget, isolating failures so a slow/dead client can't block or stop
delivery to the others.

**UI status** — `useMQStatus.js` polls `/api/info` every 10s (shared across
components; polling starts on first mount, stops on last unmount).

---

## Running locally

### Prerequisites
- JDK 25, Maven (or the bundled `./mvnw`)
- A running IBM MQ queue manager reachable per `application.properties`
  (default expects `QM1` on `localhost:1414-1416`, channel `DEV.APP.SVRCONN`,
  queue `DEV.DEMO.QL.IN`, user `app`).

A quick local MQ for testing:
```sh
docker run -d --name QM1 -e LICENSE=accept -e MQ_QMGR_NAME=QM1 \
  -e MQ_APP_PASSWORD=passw0rd -p 1414:1414 -p 9443:9443 \
  icr.io/ibm-messaging/mq:latest
```

### Dev mode (live reload)
```sh
./mvnw quarkus:dev
```
- App + bundled UI: http://localhost:8080
- Swagger UI: http://localhost:8080/q/swagger-ui
- Dev UI: http://localhost:8080/q/dev

Then open the UI, click **Start** on the GET panel, send a message from the PUT
panel, and watch it appear in the live feed.

### Frontend dev server (optional, hot-reload UI)
The frontend is normally built into the JAR, but for fast UI iteration:
```sh
cd src/main/frontend
npm install
npm run dev        # Vite dev server; proxies /api and /ws to :8080
```

---

## Building

```sh
./mvnw clean package
```
This also runs the `frontend-maven-plugin`, which installs Node/npm, runs
`npm install` and `npm run build`, and emits the compiled SPA into
`src/main/resources/META-INF/resources/` so it's served by Quarkus.

Run the packaged app:
```sh
java -jar target/quarkus-app/quarkus-run.jar
```

> **Packaging gotcha (why the UI 404'd in the container).** The `quarkus-maven-plugin`
> runs only the `build` goal — not `generate-code`/`generate-code-tests`. Those
> codegen goals run at `generate-sources` and snapshot the app's resources *before*
> Maven copies the built Vue UI into `target/classes` (at `process-resources`), which
> left `META-INF/resources` out of the fast-jar. The symptom: `/api/*` worked but the
> UI returned `Resource not found` in the container, while `quarkus:dev` was fine
> (dev mode serves resources live from disk). This project has no codegen extension
> (gRPC/Avro/proto), so dropping those goals is safe. If you add one later, re-add
> them and ensure the UI is built before `generate-sources`.

---

## Container image

Build a multi-arch (amd64 + arm64) image and push it:
```sh
./build_jvm_container-multi-arch.sh
```
The script runs `mvn clean package -DskipTests=true`, then builds and pushes a
manifest using `src/main/docker/Dockerfile.hummingbird` (OpenJDK 25 runtime,
layered Quarkus app, runs as non-root UID 65532, exposes 8080). Edit
`CONTAINER_NAME`/`TAG` at the top of the script for your registry.

---

## Notes & limitations

- **No tests** yet, though JUnit 5 + REST Assured are wired in `pom.xml`.
- **Plaintext password** in `application.properties` is fine for a local demo;
  externalize (env var / secret) before real use.
- PUT now waits for the emitter acknowledgement before responding (`202` on a
  confirmed enqueue, `500` on failure). Note this is delivery confirmation to the
  connector, not a full transactional/at-least-once guarantee across a crash.
- See `CODE_REVIEW_CHANGES.md` for the recent robustness fixes to the consumer,
  broadcast, and `/info` probe.
