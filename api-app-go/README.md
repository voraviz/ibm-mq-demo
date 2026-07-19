# api-app-go — IBM MQ API in Go

A small Go HTTP service that **puts** and **gets** messages to/from IBM MQ, with a
live WebSocket stream and Prometheus metrics. It is a functional port of the
Quarkus [`api-app/`](../api-app/) (same endpoints, same JSON contracts), built on
the official IBM MQ Go client
[ibm-messaging/mq-golang](https://github.com/ibm-messaging/mq-golang).

It is HA-aware: it uses `MQCNO_RECONNECT` so connections survive Native HA
failover and participate in Uniform Cluster balancing, and `/api/info` resolves
the **active** queue manager node.

---

## How it works

```
                       ┌────────────────────────────────────────────────┐
                       │                api-app-go (:8081)              │
  HTTP / WS client     │                                                │
  ┌───────────────┐    │  POST /api/messages ─▶ Producer.Put            │
  │  send message │────┼──────────────────────▶ (fresh MQCONN per call)─┼──PUT─┐
  └───────────────┘    │                                                │      │  ┌──────────┐
                       │                                                │      └─▶│          │
  ┌───────────────┐    │  ws.Hub.Broadcast ◀── Consumer.readLoop ◀──────┼──GET────│  IBM MQ  │
  │ live feed (WS)│◀───┼── /ws/messages         (background goroutine)  │         │  queue   │
  └───────────────┘    │                                                │         └──────────┘
  ┌───────────────┐    │ GET /api/info ─▶ mq.Probe (resolve active node)│
  │ status / info │────┼─▶ (2s timeout)                                 │
  └───────────────┘    │ GET /metrics ─▶ Prometheus registry            │
                       └────────────────────────────────────────────────┘
```

- **Send (PUT)** — `POST /api/messages` → `Producer.Put` opens a fresh MQ
  connection, puts `"[#N] <text>"` on the queue, and disconnects (stateless per
  call). Returns `202` with the formatted body.
- **Receive (GET)** — `Consumer` runs a background goroutine that polls the queue
  (`MQGET`, 500ms wait) and hands each message to the WebSocket hub. Controlled via
  `/api/consumer/{start,stop,status,count}`.
- **Broadcast** — `ws.Hub` fans each received message out to all connected
  WebSocket clients, with a per-write deadline so a slow client can't stall the
  consumer.
- **Info** — `GET /api/info` opens a transient probe connection to verify
  connectivity and resolve the active HA node's queue manager name / host / port,
  bounded by a 3s timeout.
- **Metrics** — Prometheus registry at `/metrics` (Go runtime + process collectors
  and a `mq_messages_put_total` counter).

> In the default config, PUT and GET use the **same** queue (`DEV.DEMO.QL.IN`), so
> a message you send comes straight back into the live feed — an end-to-end
> round-trip demo.

---

## Project structure

```
api-app-go/
├── main.go                    # wiring: config, producer, consumer, hub, routes
├── config/config.go           # env-var configuration (Config, Load)
├── mq/
│   ├── connect.go             # MQCNO/MQCD/MQCSP dial; CCDT URL resolution
│   ├── producer.go            # Producer.Put — one connection per message
│   ├── consumer.go            # background read loop + reconnect
│   └── probe.go               # /info connectivity + active-node resolution
├── handlers/
│   ├── messages.go            # POST /api/messages, GET /api/messages/count
│   ├── consumer.go            # /api/consumer/{start,stop,status,count}
│   ├── info.go                # GET /api/info (timeout-bounded probe)
│   └── util.go                # writeJSON helper
├── ws/hub.go                  # WebSocket hub + broadcast
├── middleware/cors.go         # permissive CORS wrapper
├── Dockerfile                 # multi-stage build w/ MQ redistributable client
└── build_container-multi-arch.sh
```

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/messages` | Send a message to IBM MQ. Body: `{"text":"..."}`. `202` on success, `400` if blank. |
| `GET` | `/api/messages/count` | Total messages sent this session |
| `POST` | `/api/consumer/start` | Start the queue consumer |
| `POST` | `/api/consumer/stop` | Stop the queue consumer |
| `GET` | `/api/consumer/status` | Consumer state (`running` / `stopped`) |
| `GET` | `/api/consumer/count` | Total messages received this session |
| `GET` | `/api/info` | IBM MQ connection info (resolved QM name, host, port, connected) |
| `GET` | `/metrics` | Prometheus metrics |
| `WS` | `/ws/messages` | Live message stream (WebSocket) |

---

## Configuration — environment variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8081` | HTTP listen port |
| `IBM_MQ_CCDT_URL` | *(unset)* | CCDT file URL — if set, overrides connection list and channel |
| `IBM_MQ_CONNECTION_LIST` | `localhost(1414),localhost(1415),localhost(1416)` | Multi-host failover connection list |
| `IBM_MQ_HOST` | `localhost` | MQ host (used when connection list is empty) |
| `IBM_MQ_PORT` | `1414` | MQ port (used when connection list is empty) |
| `IBM_MQ_CHANNEL` | `DEV.APP.SVRCONN` | Server-connection channel (ignored when CCDT is set) |
| `IBM_MQ_QUEUE_MANAGER` | `QM1` | Queue manager name — use `*UNIQA` for Uniform Cluster |
| `IBM_MQ_USERNAME` | `app` | MQ username |
| `IBM_MQ_PASSWORD` | `passw0rd` | MQ password |
| `IBM_MQ_QUEUE` | `DEV.DEMO.QL.IN` | Queue name |
| `IBM_MQ_HEARTBEAT_INTERVAL` | `5` | Client channel heartbeat interval, in seconds |
| `MQAPPLNAME` | `API-APP-GO` | Application name shown in `DISPLAY APSTATUS` |

> **Note:** `IBM_MQ_HEARTBEAT_INTERVAL` sets the client channel
> `HeartbeatInterval` (`mq/connect.go`). It controls how quickly a dead
> connection is detected, not a reconnect timeout. Reconnect itself is handled
> by `MQCNO_RECONNECT` (client-level HA/cluster failover) plus the consumer's
> own 1s retry loop. Unlike the Java app's `client-reconnect-timeout`
> (`WMQ_CLIENT_RECONNECT_TIMEOUT`), this is a heartbeat, not a reconnect cap.

### Connection modes

**Connection list (default — Native HA):**

```bash
export IBM_MQ_CONNECTION_LIST="localhost(1414),localhost(1415),localhost(1416)"
export IBM_MQ_QUEUE_MANAGER=QM1
./api-app-go
```

**CCDT (Uniform Cluster):**

```bash
export IBM_MQ_CCDT_URL=file:../ccdt/ccdt.cluster.json
export IBM_MQ_QUEUE_MANAGER='*UNIQA'
./api-app-go
```

> The Go MQI C library requires an **absolute** `file:///path` CCDT URL. A relative
> `file:../...` URL is automatically resolved to absolute at startup
> (`resolveCcdtUrl` in `mq/connect.go`) — no manual adjustment needed.

---

## Prerequisites — IBM MQ C client libraries

The `mq-golang` library uses CGO and requires the IBM MQ C client headers and
shared libraries to be present **at build time**.

### macOS

```bash
brew tap ibm-messaging/ibmmq
brew install ibm-messaging/ibmmq/mqdevtoolkit
# installs to /opt/mqm — matches mq-golang's default CGO search path
```

### Linux

Download the IBM MQ Redistributable Client (no root required):

```bash
MQ_URL=https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist/9.4.5.1-IBM-MQC-Redist-LinuxX64.tar.gz
mkdir -p /opt/mqm && curl -fsSL "$MQ_URL" | tar -xz -C /opt/mqm
```

If installed to a non-default path, point CGO at it:

```bash
export CGO_CFLAGS="-I/your/path/inc -D_REENTRANT"
export CGO_LDFLAGS="-L/your/path/lib64 -lmqm_r -Wl,-rpath,/your/path/lib64"
```

---

## Build and run

```bash
cd api-app-go
go build -o api-app-go .
./api-app-go
```

Requires Go 1.26+ (see `go.mod`) and the MQ C client (above). A running IBM MQ
reachable per the configuration is needed at runtime; start the consumer with a
`POST /api/consumer/start` before expecting messages on the live feed.

## Build container

```bash
cd api-app-go
bash build_container-multi-arch.sh
```

Produces `quay.io/voravitl/simple-mq-api-go:latest` for **`linux/amd64` only** —
IBM ships the MQ redistributable C client for LinuxX64 only (no ARM64), and
`mq-golang` requires that native client via CGO.

The `Dockerfile` is multi-stage:
1. **build** — `hi/go` installs the MQ redistributable client and compiles with
   `CGO_ENABLED=1`, producing a binary dynamically linked to `libmqm_r.so` + glibc.
2. **runtime** — `registry.access.redhat.com/ubi9/ubi-micro:latest` (ships glibc +
   NSS + `/etc`). It copies in `/opt/mqm/lib64` (the client dlopen's its transport
   plugins, so the whole dir is copied), `libstdc++`/`libgcc_s` (the C++ runtime
   ubi-micro lacks that some MQ libs link), and the binary. Runs as non-root UID
   65532, `LD_LIBRARY_PATH=/opt/mqm/lib64`.

> If the container fails to start with a `cannot open shared object` error, that
> names a missing MQ dependency to add to the runtime stage. Verify the built image
> actually connects to MQ (including hostname resolution in the connection list).

---

## Usage examples

### Send a message

```bash
curl -X POST http://localhost:8081/api/messages \
  -H 'Content-Type: application/json' \
  -d '{"text":"hello"}'
# {"status":"sent","message":"[#1] hello"}
```

### Connection info

```bash
curl http://localhost:8081/api/info
# {"queueManager":"QM1","host":"127.0.0.1","port":1414,"connected":true}
```

### WebSocket live stream

```js
const ws = new WebSocket('ws://localhost:8081/ws/messages')
ws.onmessage = e => console.log(e.data)
```

---

## Application logs

**Native HA (connection list):**
```log
MQ connect: qmgr=QM1 conn=localhost(1414),localhost(1415),localhost(1416) channel=DEV.APP.SVRCONN user=app
```

**Uniform Cluster (CCDT):**
```log
MQ connect: qmgr=*UNIQA ccdt=file:///Users/.../ccdt/ccdt.cluster.json user=app
```

## IBM MQ application status

```bash
podman exec mq-node-1 bash -c "echo 'DISPLAY APSTATUS(*)' | runmqsc QM1"
```

**Native HA output:**
```
AMQ8932I: Display application status details.
   APPLNAME(API-APP-GO)                    CLUSTER( )
   COUNT(1)                                MOVCOUNT(0)
   BALANCED(NOTAPPLIC)
```

**Uniform Cluster output:**
```
AMQ8932I: Display application status details.
   APPLNAME(API-APP-GO)                    CLUSTER(UNIQA)
   COUNT(1)                                MOVCOUNT(0)
   BALANCED(NOTAPPLIC)                     TYPE(APPL)
```

> `BALANCED(NOTAPPLIC)` is expected — the MQI C client does not participate in IBM
> MQ's application rebalancing protocol. `CLUSTER(UNIQA)` confirms the connection is
> cluster-aware. HA failover via `MQCNO_RECONNECT` works correctly across all nodes.

---

## Notes

- **CORS is fully open** (`middleware/cors.go`) — intentional for the demo UI, not
  for production.
- **PUT is fire-and-forget-ish**: the producer confirms the MQ put succeeded before
  returning `202`, but there is no transactional guarantee across a client crash.
- **Graceful shutdown**: on `SIGINT`/`SIGTERM` the HTTP server drains in-flight
  requests (`http.Server.Shutdown`) and the MQ consumer is stopped cleanly.
- **Robustness for HA failover**: the consumer's read goroutine solely owns its MQ
  handles (`Stop()` waits rather than closing them mid-`MQGET`), the WebSocket hub
  applies a per-write deadline so a slow client can't stall consumption, `/api/info`
  is bounded by a 3s probe timeout, and oversized messages grow the receive buffer
  instead of spinning the reconnect loop.
