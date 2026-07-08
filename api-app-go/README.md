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
| `GET` | `/api/info` | IBM MQ connection info |
| `GET` | `/metrics` | Prometheus metrics |
| `WS` | `/ws/messages` | Live message stream (WebSocket) |

## Configuration — environment variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8081` | HTTP listen port |
| `IBM_MQ_HOST` | `localhost` | MQ host |
| `IBM_MQ_PORT` | `1414` | MQ port |
| `IBM_MQ_CHANNEL` | `DEV.APP.SVRCONN` | Server-connection channel |
| `IBM_MQ_QUEUE_MANAGER` | `QM1` | Queue manager name |
| `IBM_MQ_USERNAME` | `app` | MQ username |
| `IBM_MQ_PASSWORD` | `passw0rd` | MQ password |
| `IBM_MQ_QUEUE` | `DEV.DEMO.QL.IN` | Queue name |
| `IBM_MQ_CONNECTION_LIST` | `localhost(1414),...` | Failover connection list |

## Run locally

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

### Build and run

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

## POST /api/messages

```bash
curl -X POST http://localhost:8081/api/messages \
  -H 'Content-Type: application/json' \
  -d '{"text":"hello"}'
# {"status":"sent","message":"[#1] hello"}
```

## WebSocket

```js
const ws = new WebSocket('ws://localhost:8081/ws/messages')
ws.onmessage = e => console.log(e.data)
```
