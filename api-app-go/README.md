# api-app-go — IBM MQ API in Go

Go implementation of [`api-app/`](../api-app/) with identical functionality.
Uses the official IBM MQ Go client: [ibm-messaging/mq-golang](https://github.com/ibm-messaging/mq-golang).

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/messages` | Send a message to IBM MQ |
| `GET` | `/api/messages/count` | Total messages sent this session |
| `POST` | `/api/consumer/start` | Start the queue consumer |
| `POST` | `/api/consumer/stop` | Stop the queue consumer |
| `GET` | `/api/consumer/status` | Consumer state (`running` / `stopped`) |
| `GET` | `/api/consumer/count` | Total messages received this session |
| `GET` | `/api/info` | IBM MQ connection info (resolved QM name, host, port) |
| `GET` | `/metrics` | Prometheus metrics |
| `WS` | `/ws/messages` | Live message stream (WebSocket) |

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
| `IBM_MQ_RECONNECT_TIMEOUT` | `30` | Seconds before a stalled reconnect attempt is abandoned |
| `MQAPPLNAME` | `API-APP-GO` | Application name shown in `DISPLAY APSTATUS` |

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

> The Go MQI C library requires an **absolute** `file:///path` CCDT URL. A relative `file:../...` URL is automatically resolved to absolute at startup — no manual adjustment needed.

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

## Build and run

```bash
cd api-app-go
go build -o api-app-go .
./api-app-go
```

## Build container (multi-arch)

```bash
cd api-app-go
bash build_container-multi-arch.sh
```

Produces `quay.io/voravitl/simple-mq-api-go:latest` for `linux/amd64` and `linux/arm64`.

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

> `BALANCED(NOTAPPLIC)` is expected — the MQI C client does not participate in IBM MQ's application rebalancing protocol. `CLUSTER(UNIQA)` confirms the connection is cluster-aware. HA failover via `MQCNO_RECONNECT` works correctly across all nodes.
