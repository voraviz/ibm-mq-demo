# IBM MQ Native HA

IBM MQ Native HA provides automatic failover across a three-node group using the Raft consensus protocol. One node is elected **Active** and serves all client connections; the other two are **Replicas** that receive continuous log replication. When the active node fails, one replica is promoted to active within seconds and clients reconnect automatically through the connection name list.

## Table of Contents

- [IBM MQ Native HA](#ibm-mq-native-ha)
  - [Table of Contents](#table-of-contents)
  - [Architecture](#architecture)
  - [MQ Configuration](#mq-configuration)
    - [Prerequisites](#prerequisites)
    - [Create Network and Volumes](#create-network-and-volumes)
    - [Create Secrets](#create-secrets)
    - [Native HA Node Configuration](#native-ha-node-configuration)
    - [Start Containers](#start-containers)
    - [Verify the Cluster](#verify-the-cluster)
  - [Test Applications](#test-applications)
    - [CCDT (Client Channel Definition Table)](#ccdt-client-channel-definition-table)
    - [All-in-one Java Application](#all-in-one-java-application)
      - [Option A — Connection list (default)](#option-a--connection-list-default)
      - [Option B — CCDT](#option-b--ccdt)
      - [Build and run](#build-and-run)
      - [OpenAPI and Swagger UI](#openapi-and-swagger-ui)
      - [Implementation Notes](#implementation-notes)
    - [Golang REST API](#golang-rest-api)
    - [Golang JMS REST API](#golang-jms-rest-api)
    - [UI app (`ui-app`)](#ui-app-ui-app)
    - [Run apps as containers (Podman)](#run-apps-as-containers-podman)
      - [All-in-one app](#all-in-one-app)
      - [Go API](#go-api)
      - [UI app](#ui-app)
    - [Batch test with helper scripts](#batch-test-with-helper-scripts)
      - [Container batch test (all-in-one)](#container-batch-test-all-in-one)
      - [Native process batch test (Go)](#native-process-batch-test-go)
  - [Reconnect behaviour: producer vs consumer](#reconnect-behaviour-producer-vs-consumer)
    - [Consumer](#consumer)
    - [Producer](#producer)
  - [Failover Test](#failover-test)

---

## Architecture

```
                    +------------------------------------------+
                    |          Client Applications             |
                    |   Connects via Connection Name List      |
                    |  localhost(1414),localhost(1415),        |
                    |             localhost(1416)              |
                    +------------------------------------------+
                                        │
            ┌───────────────────────────┼───────────────────────────┐
            │ (Active Traffic)          │                           │
            ▼                           ▼                           ▼
+────────────────────────+  +────────────────────────+  +────────────────────────+
│ Container: mq-node-1   │  │ Container: mq-node-2   │  │ Container: mq-node-3   │
│ MQ port:   1414        │  │ MQ port:   1415        │  │ MQ port:   1416        │
│ Web:       9443        │  │ Web:       9444        │  │ Web:       9445        │
│                        │  │                        │  │                        │
│  ┌──────────────────┐  │  │  ┌──────────────────┐  │  │  ┌──────────────────┐  │
│  │  Queue Manager   │  │  │  │  Queue Manager   │  │  │  │  Queue Manager   │  │
│  │    (ACTIVE)      │  │  │  │   (REPLICA)      │  │  │  │   (REPLICA)      │  │
│  └─────────┬────────┘  │  │  └─────────┬────────┘  │  │  └─────────┬────────┘  │
+────────────┼───────────+  +────────────┼───────────+  +────────────┼───────────+
             │                           ▲                           ▲
             │      Log Replication      │                           │
             └───────────────────────────┴───────────────────────────┘
                           (Raft Consensus Protocol, port 4444)
```

---

## MQ Configuration

### Prerequisites

- [Podman](https://podman.io/) (or Docker) installed.
- On Apple Silicon (M-series), all `podman run` commands include `--platform linux/amd64` because IBM does not publish an arm64 MQ server container image — it is available only for `linux/amd64`, `linux/s390x`, and `linux/ppc64le`.

### Create Network and Volumes

Create a dedicated container network so nodes can resolve each other by hostname:

```bash
podman network create mq-ha-net
```

Create one persistent volume per node. The script removes existing volumes before re-creating them so you can re-run it on a clean slate:

```bash
for i in mq-node-1-data mq-node-2-data mq-node-3-data; do
  podman volume exists $i && podman volume rm $i
  podman volume create $i
done
```

### Create Secrets

Store the admin and application passwords as Podman secrets so they are never passed as plain-text environment variables:

```bash
printf "passw0rd" | podman secret create mqAdminPassword -
printf "passw0rd" | podman secret create mqAppPassword -
```

### Native HA Node Configuration

Each node requires an `native-ha.ini` file that declares the local instance and the full membership list. All three files share the same `NativeHAInstance` list — only `NativeHALocalInstance.Name` differs.

**Node 1** — [`mq-native-ha/config/qm-node1.ini`](mq-native-ha/config/qm-node1.ini)

```ini
NativeHALocalInstance:
  Name=node-1
  GroupName=HA-GROUP-1

NativeHAInstance:
  Name=node-1
  ReplicationAddress=mq-node-1(4444)

NativeHAInstance:
  Name=node-2
  ReplicationAddress=mq-node-2(4444)

NativeHAInstance:
  Name=node-3
  ReplicationAddress=mq-node-3(4444)
```

The MQSC configuration ([`mq-native-ha/config/config.auth.mqsc`](mq-native-ha/config/config.auth.mqsc)) defines the channel, queues, channel authentication rules, and object authority records used by the demo applications.

| Component        | File                                                                           |
|------------------|--------------------------------------------------------------------------------|
| MQSC             | [`mq-native-ha/config/config.auth.mqsc`](mq-native-ha/config/config.auth.mqsc)|
| Native HA Node 1 | [`mq-native-ha/config/qm-node1.ini`](mq-native-ha/config/qm-node1.ini)        |
| Native HA Node 2 | [`mq-native-ha/config/qm-node2.ini`](mq-native-ha/config/qm-node2.ini)        |
| Native HA Node 3 | [`mq-native-ha/config/qm-node3.ini`](mq-native-ha/config/qm-node3.ini)        |

### Start Containers

Use [`create-nativeha.sh`](create-nativeha.sh) to start (or recreate) the three-node cluster in one step:

```bash
chmod +x create-nativeha.sh
./create-nativeha.sh
```

The script:
1. Stops and force-removes **all** running containers (`podman stop -a; podman rm -a -f`).
2. Removes and recreates the three named volumes (`mq-node-{1,2,3}-data`).
3. Starts all three MQ nodes in a loop, incrementing ports for each node:

| Node | MQ port | Web Console | Prometheus |
|------|---------|-------------|------------|
| mq-node-1 | 1414 | 9443 | 9517 |
| mq-node-2 | 1415 | 9444 | 9518 |
| mq-node-3 | 1416 | 9445 | 9519 |

4. Waits 60 seconds for the initial leader election, then prints the active node:

```
QM1 Active on: mq-node-2
```

> **Warning:** `podman stop -a` stops **every** container on the host, not only the MQ nodes. Use it only on a dedicated dev machine or Podman VM.

**Manual commands (reference):**

```bash
TAG=10.0.0.0-r1-amd64
CONFIG=config.auth.mqsc

# ── Node 1 ──────────────────────────────────────────────────────────────────
podman run -d \
  --secret mqAdminPassword --secret mqAppPassword \
  --name mq-node-1 \
  --platform linux/amd64 \
  --network mq-ha-net \
  --hostname mq-node-1 \
  -p 1414:1414 \
  -p 9443:9443 \
  -p 9517:9517 \
  -v mq-node-1-data:/var/mqm \
  -v ./mq-native-ha/config/qm-node1.ini:/etc/mqm/native-ha.ini:ro \
  -v ./mq-native-ha/config/$CONFIG:/etc/mqm/config.mqsc:ro \
  -e LICENSE=accept \
  -e MQ_QMGR_NAME=QM1 \
  -e MQ_NATIVE_HA=true \
  -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
  -e MQ_ENABLE_METRICS=true \
  icr.io/ibm-messaging/mq:$TAG

# ── Node 2 ──────────────────────────────────────────────────────────────────
podman run -d \
  --secret mqAdminPassword --secret mqAppPassword \
  --name mq-node-2 \
  --platform linux/amd64 \
  --network mq-ha-net \
  --hostname mq-node-2 \
  -p 1415:1414 \
  -p 9444:9443 \
  -p 9518:9517 \
  -v mq-node-2-data:/var/mqm \
  -v ./mq-native-ha/config/qm-node2.ini:/etc/mqm/native-ha.ini:ro \
  -v ./mq-native-ha/config/$CONFIG:/etc/mqm/config.mqsc:ro \
  -e LICENSE=accept \
  -e MQ_QMGR_NAME=QM1 \
  -e MQ_NATIVE_HA=true \
  -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
  -e MQ_ENABLE_METRICS=true \
  icr.io/ibm-messaging/mq:$TAG

# ── Node 3 ──────────────────────────────────────────────────────────────────
podman run -d \
  --secret mqAdminPassword --secret mqAppPassword \
  --name mq-node-3 \
  --platform linux/amd64 \
  --network mq-ha-net \
  --hostname mq-node-3 \
  -p 1416:1414 \
  -p 9445:9443 \
  -p 9519:9517 \
  -v mq-node-3-data:/var/mqm \
  -v ./mq-native-ha/config/qm-node3.ini:/etc/mqm/native-ha.ini:ro \
  -v ./mq-native-ha/config/$CONFIG:/etc/mqm/config.mqsc:ro \
  -e LICENSE=accept \
  -e MQ_QMGR_NAME=QM1 \
  -e MQ_NATIVE_HA=true \
  -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
  -e MQ_ENABLE_METRICS=true \
  icr.io/ibm-messaging/mq:$TAG
```

### Verify the Cluster

**Check all containers are running:**

```bash
podman ps --format "table {{.Names}}\t{{.Ports}}\t{{.Status}}"
```

Expected output:

```
NAMES       PORTS                                                                               STATUS
mq-node-1   0.0.0.0:1414->1414/tcp, 0.0.0.0:9443->9443/tcp, 0.0.0.0:9517->9517/tcp            Up 14 seconds
mq-node-2   0.0.0.0:1415->1414/tcp, 0.0.0.0:9444->9443/tcp, 0.0.0.0:9518->9517/tcp            Up 12 seconds
mq-node-3   0.0.0.0:1416->1414/tcp, 0.0.0.0:9445->9443/tcp, 0.0.0.0:9519->9517/tcp            Up 10 seconds
```

**Check Native HA status** — wait ~30 seconds for the initial leader election to complete:

```bash
podman exec mq-node-1 dspmq -o nativeha -x
```

Expected output (active node may vary):

```
QMNAME(QM1)       ROLE(Replica) INSTANCE(node-1) INSYNC(yes) QUORUM(3/3) GRPLSN(<0:0:36:38367>) GRPNAME(HA-GROUP-1) GRPROLE(Live)
 INSTANCE(node-1) ROLE(Replica) REPLADDR(mq-node-1) CONNACTV(yes) INSYNC(yes) BACKLOG(0) CONNINST(yes) ACKLSN(<0:0:36:38367>) HASTATUS(Normal) SYNCTIME(2026-07-07T10:37:36.077777Z) ALTDATE(2026-07-07) ALTTIME(10.37.37)
 INSTANCE(node-2) ROLE(Active)  REPLADDR(mq-node-2) CONNACTV(yes) INSYNC(yes) BACKLOG(0) CONNINST(yes) ACKLSN(<0:0:36:38367>) HASTATUS(Normal) SYNCTIME(2026-07-07T10:37:37.430102Z) ALTDATE(2026-07-07) ALTTIME(10.37.37)
 INSTANCE(node-3) ROLE(Replica) REPLADDR(mq-node-3) CONNACTV(yes) INSYNC(yes) BACKLOG(0) CONNINST(yes) ACKLSN(<0:0:36:38367>) HASTATUS(Normal) SYNCTIME(2026-07-07T10:37:36.077777Z) ALTDATE(2026-07-07) ALTTIME(10.37.37)
```

> **Notes:**
> - `QUORUM(3/3)` — all three nodes are participating.
> - `node-2` is active in this example; `node-1` and `node-3` are replicas.
> - Reference: [dspmq command — IBM Docs](https://www.ibm.com/docs/en/ibm-mq/9.4.x?topic=reference-dspmq-display-queue-managers)
<!-- Displays operational information for an instance in a Native HA configuration. Used on its own, displays ROLE, INSTANCE, INSYNC, QUORUM, GRPLSN, GRPNAME, and GRPROLE fields. (see Table 2). Combine with the -x parameter to view additional information on all the instances in the Native HA configuration (see Table 4). Combine with the -g parameter to view information about Native HA groups (see Table 5) -->

**Confirm all MQ listener ports are reachable:**

```bash
nc -vz 127.0.0.1 1414
nc -vz 127.0.0.1 1415
nc -vz 127.0.0.1 1416
```

**Probe which node is active via the REST API** (useful when scripting a Layer 4 load-balancer health check):

```bash
ADMIN_USER=admin
ADMIN_PASS=passw0rd
QM_NAME=QM1

for PORT in 9443 9444 9445; do
  echo "=== port $PORT ==="
  curl -sk -u ${ADMIN_USER}:${ADMIN_PASS} \
    https://localhost:$PORT/ibmmq/rest/v1/admin/qmgr/$QM_NAME \
    | python3 -m json.tool 2>/dev/null || echo "(no response)"
done
```

Response from the **active** node:

```json
{
  "qmgr": [
    {
      "name": "QM1",
      "state": "running"
    }
  ]
}
```

Response from a **replica** node:

```json
{
  "error": [{
    "msgId": "MQWB0004E",
    "completionCode": 2,
    "reasonCode": 2543,
    "type": "pcf",
    "message": "MQWB0004E: An internal error occurred while communicating with the queue manager. The root MQ reason code was 2543 : MQRC_STANDBY_Q_MGR."
  }]
}
```

## Test Applications

All test applications support two connection modes — **connection name list** (default) and **CCDT** — and enable automatic client reconnect so that a failover is transparent to the application layer.

### CCDT (Client Channel Definition Table)

A **Client Channel Definition Table (CCDT)** is a JSON file that describes the MQ channel and connection endpoints the client should use. It is an alternative to specifying `connectionList` and `channel` separately — the client reads both from the CCDT file instead.

| | Connection List | CCDT |
|---|---|---|
| Config location | App config / env vars | JSON file on disk (or URL) |
| Channel | Set separately | Embedded in the JSON |
| Multi-host | Comma-separated list | Array of `connection` objects |
| When to use | Simple setups, local dev | Production, shared config, uniform cluster |

Two CCDT files are provided in the [`ccdt/`](ccdt/) directory:

**[`ccdt/ccdt.nativeha.json`](ccdt/ccdt.nativeha.json)** — for Native HA (3 nodes, queue manager `QM1`):

```json
{
  "channel": [
    {
      "name": "DEV.APP.SVRCONN",
      "clientConnection": {
        "connection": [
          {"host": "127.0.0.1", "port": 1414},
          {"host": "127.0.0.1", "port": 1415},
          {"host": "127.0.0.1", "port": 1416}
        ],
        "queueManager": "QM1"
      },
      "type": "clientConnection"
    }
  ]
}
```

<!-- **[`ccdt/ccdt.cluster.json`](ccdt/ccdt.cluster.json)** — for Native HA + Uniform Cluster (6 nodes, queue manager `UNIQA`):

```json
{
  "channel": [
    {
      "name": "DEV.APP.SVRCONN",
      "clientConnection": {
        "connection": [
          {"host": "127.0.0.1", "port": 1414},
          {"host": "127.0.0.1", "port": 1415},
          {"host": "127.0.0.1", "port": 1416},
          {"host": "127.0.0.1", "port": 1417},
          {"host": "127.0.0.1", "port": 1418},
          {"host": "127.0.0.1", "port": 1419}
        ],
        "queueManager": "UNIQA"
      },
      "type": "clientConnection"
    }
  ]
}
``` -->

> **Note:** The `queueManager` field in the CCDT matches the queue manager the client connects to. Use `QM1` for Native HA and `UNIQA` (or `*UNIQA`) for a Uniform Cluster — the asterisk prefix tells the client to accept any queue manager whose name starts with `UNIQA`.

---

### All-in-one Java Application

[`all-in-one-app`](all-in-one-app) is a Quarkus/JMS backend with a bundled Vue.js frontend.

The app supports two connection modes — **connection list** (default) and **CCDT**. The mode is selected at startup based on whether `ibm.mq.ccdt-url` is set. If it is present and non-blank, CCDT is used; otherwise the connection list and channel are used.

#### Option A — Connection list (default)

[`application.properties`](all-in-one-app/src/main/resources/application.properties):

```properties
# ibm.mq.ccdt-url is commented out — connection list is active
ibm.mq.connection-list=localhost(1414),localhost(1415),localhost(1416)
ibm.mq.channel=DEV.APP.SVRCONN
ibm.mq.queue-manager=QM1
```

#### Option B — CCDT

uncomment `ibm.mq.ccdt-url` and comment out `connection-list` and `channel`:

```properties
ibm.mq.ccdt-url=file:ccdt/ccdt.nativeha.json
#ibm.mq.connection-list=localhost(1414),localhost(1415),localhost(1416)
#ibm.mq.channel=DEV.APP.SVRCONN
ibm.mq.queue-manager=QM1
```

> The CCDT file embeds the channel name and all connection endpoints, so `connection-list` and `channel` are ignored when `ccdt-url` is set.

**Connection factory** — [`MQConnectionFactoryProducer.java`](all-in-one-app/src/main/java/com/example/config/MQConnectionFactoryProducer.java):

```java
@ApplicationScoped
public class MQConnectionFactoryProducer {
    private static final Logger LOG = Logger.getLogger(MQConnectionFactoryProducer.class);

    @Inject
    MQConfig config;

    @Produces
    @ApplicationScoped
    public ConnectionFactory connectionFactory() throws JMSException {
        MQConnectionFactory factory = new MQConnectionFactory();

        // ccdtUrl() returns Optional<String> — present and non-blank means use CCDT
        String ccdtUrl = config.ccdtUrl().filter(s -> !s.isBlank()).orElse(null);
        if (ccdtUrl != null) {
            factory.setStringProperty(WMQConstants.WMQ_CCDTURL, ccdtUrl);
            LOG.info("Use CCDT: " + ccdtUrl);
        } else {
            factory.setConnectionNameList(config.connectionList());
            factory.setChannel(config.channel());
            LOG.info("Use Connection List: " + config.connectionList());
        }

        factory.setQueueManager(config.queueManager());
        factory.setIntProperty(WMQConstants.WMQ_CONNECTION_MODE, WMQConstants.WMQ_CM_CLIENT);
        factory.setStringProperty(WMQConstants.USERID, config.username());
        factory.setStringProperty(WMQConstants.PASSWORD, config.password());
        factory.setStringProperty(WMQConstants.WMQ_APPLICATIONNAME, config.applicationName());
        // WMQ_CLIENT_RECONNECT retries indefinitely across all hosts in the
        // connection name list (or CCDT) until the active node is found.
        // WMQ_CLIENT_RECONNECT_TIMEOUT caps each individual reconnect attempt
        // (ibm.mq.client-reconnect-timeout, default 5 s).
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_OPTIONS, WMQConstants.WMQ_CLIENT_RECONNECT);
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_TIMEOUT, config.clientReconnectTimeout());
        return factory;
    }
}
```

#### Build and run

```bash
# Build
cd all-in-one-app
./mvnw clean package

# Run from JAR
java -jar target/quarkus-app/quarkus-run.jar
```

Open [http://localhost:8080](http://localhost:8080) in a browser. The top menu bar shows which MQ node the application is currently connected to.

![Application connected to mq-node-2](images/mq-app.png)

> To run as a container instead, see [Run apps as containers (Podman)](#run-apps-as-containers-podman).

#### OpenAPI and Swagger UI

The REST API is fully documented with OpenAPI 3. Once the application is running:
You can access swagger-ui at /q/swagger-ui

| URL | Description |
|-----|-------------|
| [http://localhost:8080/q/swagger-ui](http://localhost:8080/q/swagger-ui) | Interactive Swagger UI |
| [http://localhost:8080/q/openapi](http://localhost:8080/q/openapi) | OpenAPI 3 spec (YAML) |
| [http://localhost:8080/q/openapi?format=json](http://localhost:8080/q/openapi?format=json) | OpenAPI 3 spec (JSON) |

The spec covers three tag groups:

| Tag | Endpoints |
|-----|-----------|
| **MQ Info** | `GET /api/info` — queue manager name, host, port, connection status |
| **MQ Consumer** | `POST /api/consumer/start`, `POST /api/consumer/stop`, `GET /api/consumer/status`, `GET /api/consumer/count` |
| **MQ Messages** | `POST /api/messages` (send), `GET /api/messages/count` |

Swagger-UI

![swagger UI](images/swagger-ui.png)

#### Implementation Notes

**Frontend resilience during failover — `sendOneWithRetry()`**

The bulk-send feature ("Send N messages") in the Vue.js frontend is implemented as a
`for` loop that calls `POST /api/messages` once per message with an optional delay between
calls. During a Native HA failover the HTTP request can fail while the Java process
reconnects to the new active node. Without extra handling, the first failure would throw
out of the loop and **silently drop all remaining messages**.

[`PutPanel.vue`](all-in-one-app/src/main/frontend/src/components/PutPanel.vue) wraps
each individual send in `sendOneWithRetry()`:

```js
async function sendOneWithRetry() {
  while (true) {
    if (cancelSignal) return false        // user pressed Cancel
    try {
      await sendOne()                     // POST /api/messages
      return true                         // success — advance the loop
    } catch (err) {
      notification.value = {
        type: 'warning',
        message: `Send failed (${err.message}) — retrying in ${RETRY_DELAY_MS / 1000}s…`,
      }
      await sleep(RETRY_DELAY_MS)         // wait 2 s, then retry the same message
      notification.value = null
    }
  }
}
```

The outer `send()` loop calls `sendOneWithRetry()` for every message instead of calling
`sendOne()` directly:

```
for each message in batch:
    sendOneWithRetry()  ← retries this one message until success or Cancel
    advance progress bar
    sleep(delayMs)
```

**What happens during a failover:**

1. The active MQ node is stopped. The in-flight `POST /api/messages` fails (network error
   or `503` while Quarkus reconnects).
2. `sendOneWithRetry()` catches the error, shows a yellow "retrying in 2s…" banner, and
   sleeps 2 seconds.
3. Meanwhile the IBM MQ client library (`WMQ_CLIENT_RECONNECT`) reconnects Quarkus to the
   new active node — typically within a few seconds.
4. On the next retry the `POST` succeeds. The warning banner clears automatically.
5. The loop continues from the **same message position** — no message is skipped or
   double-sent.
6. The user can press **Cancel** at any time to stop retrying and end the batch cleanly.

**Why the server-side counter never resets**

The `[#N]` sequence number is generated by an `AtomicLong counter` in
[`MessageResource.java`](all-in-one-app/src/main/java/com/example/resource/MessageResource.java)
on the `@ApplicationScoped` CDI bean. The bean lives for the lifetime of the process, so
the counter is only reset if the JVM restarts. The MQ reconnect is fully transparent at
the Java layer — only the TCP connection is re-established, the bean and its counter are
untouched.

---

### Golang REST API

[`api-app-go`](api-app-go) is a standalone Go REST API backend. Pair it with the [`ui-app`](ui-app) Vue.js frontend.

**Configuration** — [`api-app-go/config/config.go`](api-app-go/config/config.go):

```go
func Load() *Config {
    return &Config{
        CcdtUrl:          getenv("IBM_MQ_CCDT_URL", ""),
        ConnectionList:   getenv("IBM_MQ_CONNECTION_LIST", "localhost(1414),localhost(1415),localhost(1416)"),
        Host:             getenv("IBM_MQ_HOST", "localhost"),
        Port:             getenvInt("IBM_MQ_PORT", 1414),
        AppName:          getenv("MQAPPLNAME", "API-APP-GO"),
        Channel:          getenv("IBM_MQ_CHANNEL", "DEV.APP.SVRCONN"),
        QueueManager:     getenv("IBM_MQ_QUEUE_MANAGER", "QM1"),
        Username:         getenv("IBM_MQ_USERNAME", "app"),
        Password:         getenv("IBM_MQ_PASSWORD", "passw0rd"),
        Queue:             getenv("IBM_MQ_QUEUE", "DEV.DEMO.QL.IN"),
        HeartbeatInterval: getenvInt("IBM_MQ_HEARTBEAT_INTERVAL", 5),
        ServerPort:        getenv("SERVER_PORT", "8081"),
    }
}
```

When `IBM_MQ_CCDT_URL` is set, it takes precedence over `IBM_MQ_CONNECTION_LIST` and `IBM_MQ_CHANNEL` — both channel name and connection endpoints are read from the CCDT file.

**Automatic reconnect** — [`api-app-go/mq/connect.go`](api-app-go/mq/connect.go):

```go
cno.Options = ibmmq.MQCNO_CLIENT_BINDING |
    // MQCNO_RECONNECT: allows reconnect within a Native HA group (failover)
    // AND across queue managers in a Uniform Cluster (balancing).
    // MQCNO_RECONNECT_Q_MGR would block cross-QM balancing entirely.
    ibmmq.MQCNO_RECONNECT
```

`MQCNO_RECONNECT` is what makes failover transparent: when the connection drops, the
MQ client library reconnects through the connection name list (or CCDT) automatically,
blocking in-flight MQI calls rather than failing them.

`cd.HeartbeatInterval` is set to `HeartbeatInterval` (env `IBM_MQ_HEARTBEAT_INTERVAL`,
default 5 s). This is the **channel heartbeat interval** — it governs how quickly a dead
connection is *detected*, not a reconnect timeout. It is distinct from the Java app's
`WMQ_CLIENT_RECONNECT_TIMEOUT`, which is a genuine cap on each reconnect attempt; the two
apps are not equivalent on this point. The Go consumer's own retry cadence lives in
`reconnectLoop()` (a hard-coded 1 s between attempts, see below), not in this setting.

**Prerequisites — IBM MQ C client libraries**

The `mq-golang` library uses CGO and requires the IBM MQ C client headers and shared libraries at build time.

- **macOS:**

  ```bash
  brew tap ibm-messaging/ibmmq
  brew install ibm-messaging/ibmmq/mqdevtoolkit
  # Installs to /opt/mqm — the default CGO search path for mq-golang
  ```

- **Linux:**

  ```bash
  MQ_URL=https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist/9.4.5.1-IBM-MQC-Redist-LinuxX64.tar.gz
  mkdir -p /opt/mqm && curl -fsSL "$MQ_URL" | tar -xz -C /opt/mqm
  ```

  If installed to a non-default path, set the CGO flags before building:

  ```bash
  export CGO_CFLAGS="-I/your/path/inc -D_REENTRANT"
  export CGO_LDFLAGS="-L/your/path/lib64 -lmqm_r -Wl,-rpath,/your/path/lib64"
  ```

**Build and run the API:**

```bash
cd api-app-go
go build -o api-app-go .
./api-app-go
```

The API listens on **http://localhost:8081** by default. Override any setting via environment variable before running.

**Connection list:**

```bash
IBM_MQ_CONNECTION_LIST="localhost(1414),localhost(1415),localhost(1416)" \
IBM_MQ_USERNAME="app" \
IBM_MQ_PASSWORD="passw0rd" \
IBM_MQ_QUEUE="DEV.DEMO.QL.IN" \
./api-app-go
```

**CCDT** — set `IBM_MQ_CCDT_URL` instead; `IBM_MQ_CONNECTION_LIST` and `IBM_MQ_CHANNEL` are ignored when CCDT is used:

```bash
IBM_MQ_CCDT_URL="file:///$(pwd)/../ccdt/ccdt.nativeha.json" \
IBM_MQ_QUEUE_MANAGER="QM1" \
IBM_MQ_USERNAME="app" \
IBM_MQ_PASSWORD="passw0rd" \
IBM_MQ_QUEUE="DEV.DEMO.QL.IN" \
./api-app-go
```

**Producer behaviour during failover**

The Go producer ([`api-app-go/mq/producer.go`](api-app-go/mq/producer.go)) opens a
**fresh MQ connection per `PUT` call** — there is no long-lived connection to drop. On
every call it runs: connect → open queue → put → close. If any step fails, the counter
is **atomically rolled back** before the error is returned:

```go
func (p *Producer) Put(text string) (string, error) {
    n := p.counter.Add(1)              // reserve a sequence number
    body := fmt.Sprintf("[#%d] %s", n, text)

    qmgr, _, err := connect(p.cfg)
    if err != nil {
        p.counter.Add(-1)              // rollback — no gap in [#N] sequence
        return "", fmt.Errorf("MQ connect: %w", err)
    }
    defer qmgr.Disc()
    ...
    if err := qObj.Put(md, pmo, []byte(body)); err != nil {
        p.counter.Add(-1)              // rollback
        return "", fmt.Errorf("MQ put: %w", err)
    }
    return body, nil                   // only reaches here on confirmed delivery
}
```

The Java `all-in-one-app` producer reaches the same guarantees a different way. Its
`MessageResource.putMessage()` uses a SmallRye Reactive Messaging emitter (`@Channel("mq-put")`,
`smallrye-jms` connector) over a **long-lived** connection, and **awaits the emitter's
acknowledgement** before responding — so the two backends now behave equivalently:

| | Java `all-in-one-app` | Go `api-app-go` |
|--|--|--|
| Connection model | Long-lived, reused across requests | Fresh connection per `PUT` |
| Counter on failure | Incremented, then **rolled back** (`decrementAndGet`) | Incremented, then **rolled back** (`counter.Add(-1)`) |
| Counter after failover | Always equals confirmed deliveries (no gap) | Always equals confirmed deliveries (no gap) |
| HTTP response on success / MQ error | `202 Accepted` / `500 Internal Server Error` | `202 Accepted` / `500 Internal Server Error` |
| Failover handling | Client `WMQ_CLIENT_RECONNECT` on the shared connection | `MQCNO_RECONNECT` scoped to that single `PUT` |

Because `PUT /api/messages` returns `500` when the MQ connection fails, the
`sendOneWithRetry()` loop in the Vue frontend catches it, waits 2 seconds, and retries
the **same message**. The retry loop is identical regardless of which backend is used.

> To run as a container instead, see [Run apps as containers (Podman)](#run-apps-as-containers-podman).

### Golang JMS REST API

[`api-app-go-jms20`](api-app-go-jms20) is a Go REST API that uses the higher-level
[mq-golang-jms20](https://github.com/ibm-messaging/mq-golang-jms20) JMS-style library
instead of the raw `mq-golang/v5` C-binding. It is functionally identical to `api-app-go` —
same endpoints, same message format, same WebSocket broadcast — with the MQ layer replaced by
`JMSContext`, `JMSProducer`, and `JMSConsumer`.

| | `api-app-go` | `api-app-go-jms20` |
|--|--|--|
| Library | `mq-golang/v5/ibmmq` | `mq-golang-jms20` |
| Send | `qObj.Put(md, pmo, bytes)` | `ctx.CreateProducer().SendString(dest, body)` |
| Receive | `qObj.Get(md, gmo, buf)` + MQRC 2033 check | `consumer.Receive(500)` — `nil` = no message |

**Connection factory** — [`api-app-go-jms20/mq/connect.go`](api-app-go-jms20/mq/connect.go):

```go
cf := mqjms.ConnectionFactoryImpl{
    QMName:      cfg.QueueManager,
    Hostname:    host,       // first entry of connection list
    PortNumber:  port,
    ChannelName: cfg.Channel,
    UserName:    cfg.Username,
    Password:    cfg.Password,
}
```

**Multi-host HA / CCDT** — `ConnectionFactoryImpl` only exposes a single `Hostname`+`PortNumber`,
so an `MQOptions` callback is used to inject the full connection details and enable
`MQCNO_RECONNECT` at connect time:

```go
func connectOption(cfg *config.Config) jms20subset.MQOptions {
    return func(cno *ibmmq.MQCNO) {
        cno.Options |= ibmmq.MQCNO_RECONNECT
        cno.ApplName = cfg.AppName

        if cfg.CcdtUrl != "" {
            // CCDT takes precedence — channel and connection list are read
            // from the CCDT file; no ConnectionName override is needed.
            cno.CCDTUrl = resolveCcdtUrl(cfg.CcdtUrl)
        } else {
            // Override ConnectionName with the full multi-host list.
            cno.ClientConn.ConnectionName = connectionName(cfg)
            cno.ClientConn.HeartbeatInterval = int32(cfg.HeartbeatInterval)
        }
    }
}
```

`MQCNO_RECONNECT` (not `MQCNO_RECONNECT_Q_MGR`) allows reconnect within a Native HA group **and** across queue managers in a Uniform Cluster. `_Q_MGR` would block cross-QM balancing entirely.

**Consumer poll loop** — [`api-app-go-jms20/mq/consumer.go`](api-app-go-jms20/mq/consumer.go):

```go
// 500ms receive timeout — mirrors jmsConsumer.receive(500) in MQConsumer.java.
// nil message with nil error means timeout expired (no message available).
msg, jmsErr := cons.Receive(500)
if jmsErr != nil { /* reconnectLoop() */ }
if msg == nil    { continue }           // timeout, no message
text := *msg.(jms20subset.TextMessage).GetText()
```

**Prerequisites — IBM MQ C client libraries**

`mq-golang-jms20` wraps `mq-golang/v5` and therefore also requires CGO and the IBM MQ
C client. Use the same setup as for `api-app-go` above.

**Build and run the API:**

```bash
cd api-app-go-jms20
go build -o api-app-go-jms20 .
./api-app-go-jms20
```

The API listens on **http://localhost:8081** by default (same port as `api-app-go`).
Override settings via environment variables before running:

```bash
IBM_MQ_CONNECTION_LIST="localhost(1414),localhost(1415),localhost(1416)" \
IBM_MQ_USERNAME="app" \
IBM_MQ_PASSWORD="passw0rd" \
IBM_MQ_QUEUE="DEV.DEMO.QL.IN" \
./api-app-go-jms20
```

> To run as a container instead, see [Run apps as containers (Podman)](#run-apps-as-containers-podman).

### UI app (`ui-app`)

The Vue.js frontend is a separate Vite app. It proxies `/api` and `/ws` to `http://localhost:8081` automatically in dev mode, so no extra configuration is needed.

```bash
cd ui-app
npm install        # first time only
npm run dev
```

Open [http://localhost:8080](http://localhost:8080) in a browser.

> To point the UI at a non-default API host, copy [`.env.example`](ui-app/.env.example) and set `VITE_API_BASE_URL`:
>
> ```bash
> cp ui-app/.env.example ui-app/.env
> # edit .env and set VITE_API_BASE_URL=http://<api-host>:8081
> npm run dev
> ```

> To run as a container instead, see [Run apps as containers (Podman)](#run-apps-as-containers-podman).

### Run apps as containers (Podman)

All three apps — **all-in-one**, **Go API** (`api-app-go` and `api-app-go-jms20`), and the **Vue.js UI** — have published container images. Run them with podman against the local Native HA group.

> **Architecture note:** Both Go API images are **linux/amd64** (IBM ships the MQ C client for LinuxX64 only)
> and segfault under QEMU — run on a native amd64 host, not Apple Silicon. Inside a container `localhost`
> is the *container*, not your host — use `host.containers.internal` to reach MQ ports published on the host.

#### All-in-one app

You can skip the local build and test the `all-in-one-app` straight from the published image
[`quay.io/voravitl/simple-mq-app`](https://quay.io/repository/voravitl/simple-mq-app). Mount the
Native HA container CCDT [`ccdt/ccdt.nativeha.container.json`](ccdt/ccdt.nativeha.container.json) —
it targets `QM1` across the three nodes using `host.containers.internal`, so the container can reach
the MQ ports published on the host:

```bash
podman run --detach --name="all-in-one" -p 8080:8080 \
  --network mq-ha-net \
  -v ./config/application.properties:/config/application.properties \
  -v ./ccdt/ccdt.nativeha.container.json:/config/ccdt.json:ro \
  -e QUARKUS_CONFIG_LOCATIONS="file:///config/application.properties" \
  quay.io/voravitl/simple-mq-app:latest
```

Open [http://localhost:8080](http://localhost:8080) — the top menu bar shows which node the app is
connected to, exactly as with the locally built version.

#### Go API

To switch between the two Go API variants, use the corresponding image:

| Variant | Image |
|---------|-------|
| `api-app-go` (raw MQI) | `quay.io/voravitl/simple-mq-api-go:latest` |
| `api-app-go-jms20` (JMS20) | `quay.io/voravitl/simple-mq-api-go-jms20:latest` |

MQ connection settings are read from environment variables. The example below uses `api-app-go`;
replacing the image name runs `api-app-go-jms20` with identical env vars:

```bash
podman run -d --name api-app-go -p 8081:8081 \
  -e IBM_MQ_CONNECTION_LIST="host.containers.internal(1414),host.containers.internal(1415),host.containers.internal(1416)" \
  -e IBM_MQ_CHANNEL=DEV.APP.SVRCONN \
  -e IBM_MQ_QUEUE_MANAGER=QM1 \
  -e IBM_MQ_QUEUE=DEV.DEMO.QL.IN \
  -e IBM_MQ_USERNAME=app \
  -e IBM_MQ_PASSWORD=passw0rd \
  quay.io/voravitl/simple-mq-api-go:latest

curl http://localhost:8081/api/status   # verify
```

| Variable | Default | Notes |
|----------|---------|-------|
| `IBM_MQ_CONNECTION_LIST` | `localhost(1414),localhost(1415),localhost(1416)` | Use `host.containers.internal(...)` from a container |
| `IBM_MQ_CHANNEL` | `DEV.APP.SVRCONN` | |
| `IBM_MQ_QUEUE_MANAGER` | `QM1` | |
| `IBM_MQ_QUEUE` | `DEV.DEMO.QL.IN` | |
| `IBM_MQ_USERNAME` / `IBM_MQ_PASSWORD` | `app` / `passw0rd` | |
| `IBM_MQ_CCDT_URL` | *(unset)* | if set, overrides connection list + channel |
| `SERVER_PORT` | `8081` | |

#### UI app

nginx serves the SPA and reverse-proxies `/api` and `/ws` to the backend. The backend URL is a
**runtime** env var (`API_UPSTREAM`), so the image is built once (with an empty `VITE_API_BASE_URL`,
i.e. same-origin) and pointed at any API at `podman run`:

```bash
cd ui-app
: > .env                  # empty VITE_API_BASE_URL — browser calls same-origin /api + /ws
./build_container.sh      # npm build on host, then builds the amd64 image

podman run -d --name ui-app -p 8080:8080 \
  -e API_UPSTREAM=http://host.containers.internal:8081 \
  quay.io/voravitl/simple-mq-ui:latest
```

Open [http://localhost:8080](http://localhost:8080). The browser talks only to
the UI on 8080; nginx proxies `/api` and `/ws/messages` to `API_UPSTREAM`
(defaults to `http://localhost:8081` if unset). Because everything is
same-origin, there's no CORS to configure.

### Batch test with helper scripts

Two sets of helper scripts at the repository root drive multi-instance put/consume tests end to end — one set runs the `all-in-one-app` as containers, the other starts native `api-app-go` processes.

| Script | Purpose |
|--------|---------|
| [`start-all-in-one-apps-container.sh`](start-all-in-one-apps-container.sh) | Starts several `all-in-one-app` containers (host ports `9190`+). For each instance sends 10 messages and starts the consumer. |
| [`clear-all-in-one-apps-container.sh`](clear-all-in-one-apps-container.sh) | Stops and force-removes every running `all-in-one-*` container. |
| [`start-go-lang-apps.sh`](start-go-lang-apps.sh) | Starts several `api-app-go` native processes (ports `8100`+). For each instance sends 10 messages and starts the consumer. |
| [`clear-go-lang-apps.sh`](clear-go-lang-apps.sh) | Kills all `api-app-go` processes and removes log files from `logs/`. |

#### Container batch test (all-in-one)

```bash
./start-all-in-one-apps-container.sh   # start the instances + run the put/consume test
./clear-all-in-one-apps-container.sh   # tear everything down
```

> **Note:** `start-all-in-one-apps-container.sh` ships preconfigured for the
> [Uniform Cluster](native-ha-with-uniform-cluster.md) — `IBM_MQ_QUEUE_MANAGER=*UNIQA` with
> [`ccdt/ccdt.cluster.container.json`](ccdt/ccdt.cluster.container.json). For this plain Native HA
> setup, edit it to use `IBM_MQ_QUEUE_MANAGER=QM1` and mount
> [`ccdt/ccdt.nativeha.container.json`](ccdt/ccdt.nativeha.container.json).

#### Native process batch test (Go)

```bash
./start-go-lang-apps.sh    # start api-app-go instances + run the put/consume test
./clear-go-lang-apps.sh    # kill processes and remove logs
```

> **Note:** `start-go-lang-apps.sh` ships preconfigured for the Uniform Cluster
> (`IBM_MQ_QUEUE_MANAGER=*UNIQA`, `IBM_MQ_CCDT_URL=file:./ccdt/ccdt.cluster.json`).
> For plain Native HA, edit it to use `IBM_MQ_QUEUE_MANAGER=QM1` and
> `IBM_MQ_CCDT_URL=file:./ccdt/ccdt.nativeha.json`.

**Check APP status**
- Run runmqsc at active node
```bash
podman exec mq-node-1 bash -c "echo 'DISPLAY APSTATUS(*)' | runmqsc QM1"
```
Output
```bash
1 : DISPLAY APSTATUS(*)
AMQ8932I: Display application status details.
   APPLNAME(all-in-one)                    CLUSTER( )
   COUNT(1)                                MOVCOUNT(0)
   BALANCED(NOTAPPLIC)
```

**Application Log**
- Use Connection List
```log
2026-07-12 10:19:08,192 INFO  [com.example.config.MQConnectionFactoryProducer] (Quarkus Main Thread) Use Connection List: localhost(1414),localhost(1415),localhost(1416)
```
- Connect to QM1
```log
2026-07-12 10:19:55,047 DEBUG [com.example.resource.InfoResource] (executor-thread-1) Requesting MQ info — configured queueManager: QM1
2026-07-12 10:19:55,093 DEBUG [com.example.resource.InfoResource] (executor-thread-1) MQ connection established successfully
2026-07-12 10:19:55,094 DEBUG [com.example.resource.InfoResource] (executor-thread-1) MQ Server Host: localhost/127.0.0.1 (1414), resolvedQueueManager: QM1
2026-07-12 10:19:55,112 DEBUG [com.example.resource.InfoResource] (executor-thread-1) InfoResponse: connected=true, queueManager=QM1, host=localhost/127.0.0.1, port=1414
```

## Reconnect behaviour: producer vs consumer

Every app uses the same **two-layer** model. Layer 1 is the IBM MQ *client library*
auto-reconnect, which makes HA/cluster failover transparent. Layer 2 is an *application*
reconnect loop that only runs when Layer 1 gives up and surfaces an error to the app.

| Layer | `all-in-one-app` (Java/JMS) | `api-app-go` (raw `ibmmq`) |
|--|--|--|
| **1 — client auto-reconnect** | `WMQ_CLIENT_RECONNECT` on the shared `ConnectionFactory`, capped per attempt by `WMQ_CLIENT_RECONNECT_TIMEOUT` (`ibm.mq.client-reconnect-timeout`, default 5 s) | `MQCNO_RECONNECT` on every connection; `HeartbeatInterval` (`IBM_MQ_HEARTBEAT_INTERVAL`, default 5 s) only sets how fast a dead link is *detected* |
| **2 — app reconnect loop** | `MQConsumer.reconnectLoop()` — rebuild JMS resources every **1 s** | `Consumer.reconnectLoop()` — rebuild qmgr+queue handles every **1 s** |

### Consumer

Long-lived connection with a poll loop:

1. Poll with a **500 ms** receive timeout (`consumer.receive(500)` / `qObj.Get` with `WaitInterval=500`).
2. Timeout with no message → loop and re-check the stop flag.
3. Any other error, when **not** shutting down → log "disconnected — reconnecting" and enter the app-level reconnect loop, which tears down and reopens resources every **1 s** until it succeeds or stop is requested.

Because Layer 1 absorbs most failovers transparently, the app-level `reconnectLoop` is the
**fallback** for when the client library exhausts its reconnect attempt and throws. In Go
the read goroutine exclusively owns the MQ handles (Stop only sets a flag + `wg.Wait()`),
so shutdown and reconnect never issue MQI verbs concurrently; the Java consumer achieves
the same with a `closing` flag, `volatile` handles, and `synchronized` start/stop.

### Producer

- **Java** — long-lived connection via the SmallRye JMS emitter, reused across requests. A
  failover mid-send is absorbed by Layer 1 (`WMQ_CLIENT_RECONNECT`); only if the client
  exceeds its reconnect timeout does `emitter.send()` fail → the request returns `500` and
  rolls back the counter. There is **no** producer-specific app-level retry loop.
- **Go** — **stateless**: a fresh connection is opened per `PUT` (connect → open → put →
  disc). There is no persistent connection to "reconnect" — failover *between* requests is
  a non-event, and failover *during* a request is covered by `MQCNO_RECONNECT` for that one
  call. On failure the counter is rolled back and the request returns `500`.

In both cases the client (`ui-app` / bundled frontend) retries the failed `PUT` after 2 s,
so a failover during a bulk send loses no messages.

---

## Failover Test

This walkthrough demonstrates zero-message-loss failover.

**1. Identify the active node:**

```bash
podman exec mq-node-1 dspmq -o nativeha -x
```

**2. Put 500 messages and start the consumer:**

![Put 500 messages](images/mq-app-put-500.png)

**3. Stop the active node** (replace `mq-node-2` with whichever node is active):

```bash
podman stop mq-node-2
```

**4. Observe automatic reconnection** — the application reconnects to the new active node within seconds:

![Application reconnected to new active node](images/mq-app-automatic-connect-new-active.png)

**5. Verify zero message loss:**

![No messages lost after failover](images/mq-app-message-not-lost.png)

**6. Bring the stopped node back online:**

```bash
podman start mq-node-2
```

The recovered node rejoins the group as a replica and begins catching up via log replication.
