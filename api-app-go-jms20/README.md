# api-app-go-jms20

Go REST API for IBM MQ using the [mq-golang-jms20](https://github.com/ibm-messaging/mq-golang-jms20) JMS-style library.

Functionally identical to `api-app-go` — same endpoints, same message format, same WebSocket broadcast — but the IBM MQ layer uses the higher-level JMS20 API (`JMSContext`, `JMSProducer`, `JMSConsumer`) instead of the raw `ibmmq` C binding.

## IBM MQ Library

| | api-app-go | api-app-go-jms20 |
|--|--|--|
| Library | `mq-golang/v5/ibmmq` | `mq-golang-jms20` |
| Send | `qObj.Put(md, pmo, bytes)` | `ctx.CreateProducer().SendString(dest, body)` |
| Receive | `qObj.Get(md, gmo, buf)` | `consumer.Receive(500)` |
| No message | `MQRC_NO_MSG_AVAILABLE` (rc 2033) | `Receive()` returns `nil, nil` |

Both libraries use CGO and require the IBM MQ C client redistributable at build time.

## Configuration

| Environment Variable | Default | Description |
|---|---|---|
| `IBM_MQ_HOST` | `localhost` | Single MQ host (fallback when CONNECTION_LIST is empty) |
| `IBM_MQ_PORT` | `1414` | MQ port |
| `IBM_MQ_CONNECTION_LIST` | `localhost(1414),localhost(1415),localhost(1416)` | Multi-host HA failover list |
| `IBM_MQ_CHANNEL` | `DEV.APP.SVRCONN` | Server-connection channel |
| `IBM_MQ_QUEUE_MANAGER` | `QM1` | Queue manager name |
| `IBM_MQ_USERNAME` | `app` | Username |
| `IBM_MQ_PASSWORD` | `passw0rd` | Password |
| `IBM_MQ_QUEUE` | `DEV.DEMO.QL.IN` | Queue to read/write |
| `IBM_MQ_RECONNECT_TIMEOUT` | `30` | Heartbeat/reconnect timeout in seconds |
| `SERVER_PORT` | `8081` | HTTP listen port |

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

## Build & Run

### Local (requires IBM MQ C client headers)

```sh
# Set CGO flags pointing at your local MQ installation
export CGO_CFLAGS="-I/opt/mqm/inc -D_REENTRANT"
export CGO_LDFLAGS="-L/opt/mqm/lib64 -lmqm_r"
go build -o api-app-go-jms20 .
./api-app-go-jms20
```

### Container (multi-arch)

```sh
chmod +x build_container-multi-arch.sh
./build_container-multi-arch.sh
```

Pushes to `quay.io/voravitl/simple-mq-api-go-jms20:latest` for `linux/amd64` and `linux/arm64`.

### Docker (single arch)

```sh
docker build -t api-app-go-jms20 .
docker run -p 8081:8081 \
  -e IBM_MQ_HOST=<your-host> \
  -e IBM_MQ_PASSWORD=<your-password> \
  api-app-go-jms20
```

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

### Multi-host HA

When `IBM_MQ_CONNECTION_LIST` is set, an `MQOptions` callback overrides the JMS20
connection factory's single-host `ConnectionName` with the full multi-host list and
enables `MQCNO_RECONNECT_Q_MGR`. This gives automatic transparent failover to the
active Native HA node — identical behaviour to `api-app-go`.
