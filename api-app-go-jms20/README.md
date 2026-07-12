# api-app-go-jms20

Go REST API for IBM MQ using the [mq-golang-jms20](https://github.com/ibm-messaging/mq-golang-jms20) JMS-style library.

Functionally identical to `api-app-go` — same endpoints, same message format, same WebSocket broadcast — but the IBM MQ layer uses the higher-level JMS20 API (`JMSContext`, `JMSProducer`, `JMSConsumer`) instead of the raw `ibmmq` C binding.

## IBM MQ Library comparison

| | `api-app-go` | `api-app-go-jms20` |
|--|--|--|
| Library | `mq-golang/v5/ibmmq` | `mq-golang-jms20` |
| Send | `qObj.Put(md, pmo, bytes)` | `ctx.CreateProducer().SendString(dest, body)` |
| Receive | `qObj.Get(md, gmo, buf)` | `consumer.Receive(500)` |
| No message | `MQRC_NO_MSG_AVAILABLE` (rc 2033) | `Receive()` returns `nil, nil` |
| Underlying layer | MQI C binding | MQI C binding (via `mq-golang/v5`) |
| BALANCED (Uniform Cluster) | `NOTAPPLIC` | `NOTAPPLIC` |

Both libraries use CGO and require the IBM MQ C client redistributable at build time.

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
| `IBM_MQ_USERNAME` | `app` | Username |
| `IBM_MQ_PASSWORD` | `passw0rd` | Password |
| `IBM_MQ_QUEUE` | `DEV.DEMO.QL.IN` | Queue to read/write |
| `IBM_MQ_RECONNECT_TIMEOUT` | `30` | Heartbeat/reconnect timeout in seconds |
| `MQAPPLNAME` | `API-APP-GO-JMS20` | Application name shown in `DISPLAY APSTATUS` |

### Connection modes

**Connection list (default — Native HA):**

```bash
export IBM_MQ_CONNECTION_LIST="localhost(1414),localhost(1415),localhost(1416)"
export IBM_MQ_QUEUE_MANAGER=QM1
./api-app-go-jms20
```

**CCDT (Uniform Cluster):**

```bash
export IBM_MQ_CCDT_URL=file:../ccdt/ccdt.cluster.json
export IBM_MQ_QUEUE_MANAGER='*UNIQA'
./api-app-go-jms20
```

> The Go MQI C library requires an **absolute** `file:///path` CCDT URL. A relative `file:../...` URL is automatically resolved to absolute at startup — no manual adjustment needed.

## API Endpoints

| Method | Path | Description | Response |
|---|---|---|---|
| `POST` | `/api/messages` | Send message to MQ | `202 {"status":"sent","message":"[#N] text"}` |
| `GET` | `/api/messages/count` | Total messages sent | `200 {"count":N}` |
| `POST` | `/api/consumer/start` | Start background consumer | `200 {"status":"started"}` |
| `POST` | `/api/consumer/stop` | Stop background consumer | `200 {"status":"stopped"}` |
| `GET` | `/api/consumer/status` | Consumer running state | `200 {"status":"running"\|"stopped"}` |
| `GET` | `/api/consumer/count` | Total messages received | `200 {"count":N}` |
| `GET` | `/api/info` | MQ connectivity probe | `200 {"queueManager":"QM1","host":"...","port":1414,"connected":true}` |
| `GET` | `/metrics` | Prometheus metrics | Text exposition format |
| `WS` | `/ws/messages` | Live message stream | WebSocket text frames |

## Prerequisites — IBM MQ C client libraries

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

## Build & Run

### Local

```bash
cd api-app-go-jms20
go build -o api-app-go-jms20 .
./api-app-go-jms20
```

### Container (multi-arch)

```bash
chmod +x build_container-multi-arch.sh
./build_container-multi-arch.sh
```

Pushes to `quay.io/voravitl/simple-mq-api-go-jms20:latest` for `linux/amd64` and `linux/arm64`.

### Docker (single arch)

```bash
docker build -t api-app-go-jms20 .
docker run -p 8081:8081 \
  -e IBM_MQ_HOST=<your-host> \
  -e IBM_MQ_PASSWORD=<your-password> \
  api-app-go-jms20
```

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
   APPLNAME(API-APP-GO-JMS20)              CLUSTER( )
   COUNT(1)                                MOVCOUNT(0)
   BALANCED(NOTAPPLIC)
```

**Uniform Cluster output:**
```
AMQ8932I: Display application status details.
   APPLNAME(API-APP-GO-JMS20)              CLUSTER(UNIQA)
   COUNT(1)                                MOVCOUNT(0)
   BALANCED(NOTAPPLIC)                     TYPE(APPL)
```

> `BALANCED(NOTAPPLIC)` is expected — `mq-golang-jms20` wraps MQI C (not the Java JMS provider), so the application rebalancing protocol is not available. `CLUSTER(UNIQA)` confirms the connection is cluster-aware. HA failover via `MQCNO_RECONNECT` works correctly across all nodes.

## Architecture

```
HTTP Client
    │
CORS Middleware
    │
HTTP Mux
    ├─ POST /api/messages ──────► Producer.Put()
    │                                 └─ JMSContext.CreateProducer().SendString()
    ├─ POST /api/consumer/start ──► Consumer.Start()
    │                                 └─ goroutine: JMSConsumer.Receive(500ms loop)
    │                                       └─ Hub.Broadcast() ──► WebSocket clients
    ├─ GET  /api/info ──────────► mq.Probe()
    │                                 └─ JMSContext (transient)
    └─ WS   /ws/messages ───────► Hub.ServeWS()
```

### Connection strategy

When `IBM_MQ_CCDT_URL` is set, an `MQOptions` callback sets `cno.CCDTUrl` (resolved to an absolute path) and enables `MQCNO_RECONNECT`. When only `IBM_MQ_CONNECTION_LIST` is set, the callback overrides the JMS20 connection factory's single-host `ConnectionName` with the full multi-host list. Both paths enable automatic transparent failover across Native HA nodes and Uniform Cluster queue managers.
