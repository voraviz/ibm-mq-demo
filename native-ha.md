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
    - [Golang REST API](#golang-rest-api)
    - [Golang JMS REST API](#golang-jms-rest-api)
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

**Option A — Connection list (default)** — [`application.properties`](all-in-one-app/src/main/resources/application.properties):

```properties
# ibm.mq.ccdt-url is commented out — connection list is active
ibm.mq.connection-list=localhost(1414),localhost(1415),localhost(1416)
ibm.mq.channel=DEV.APP.SVRCONN
ibm.mq.queue-manager=QM1
```

**Option B — CCDT** — uncomment `ibm.mq.ccdt-url` and comment out `connection-list` and `channel`:

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
        // WMQ_CLIENT_RECONNECT_TIMEOUT caps each individual reconnect attempt.
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_OPTIONS, WMQConstants.WMQ_CLIENT_RECONNECT);
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_TIMEOUT, 30);
        return factory;
    }
}
```

**Build and run:**

```bash
# Build
cd all-in-one-app
./mvnw clean package

# Run from JAR
java -jar target/quarkus-app/quarkus-run.jar

# Or run from container (host.containers.internal resolves to the host machine)
podman run -p 8080:8080 \
  -e IBM.MQ.CONNECTION-LIST="host.containers.internal(1414),host.containers.internal(1415),host.containers.internal(1416)" \
  quay.io/voravitl/simple-mq-app:latest
```

Open [http://localhost:8080](http://localhost:8080) in a browser. The top menu bar shows which MQ node the application is currently connected to.

![Application connected to mq-node-2](images/mq-app.png)

**OpenAPI & Swagger UI**

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
        Queue:            getenv("IBM_MQ_QUEUE", "DEV.DEMO.QL.IN"),
        ReconnectTimeout: getenvInt("IBM_MQ_RECONNECT_TIMEOUT", 30),
        ServerPort:       getenv("SERVER_PORT", "8081"),
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

`cd.HeartbeatInterval` is set to `ReconnectTimeout` (default 30 s), matching the Java `WMQ_CLIENT_RECONNECT_TIMEOUT` behaviour — a stalled reconnect attempt is abandoned after this period.

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

This gives two properties that differ from the Java SmallRye emitter:

| | Java `all-in-one-app` | Go `api-app-go` |
|--|--|--|
| Counter before delivery | Increments **before** `emitter.send()` | Increments, then **rolled back** on any failure |
| Counter after failover | May be ahead of actual delivery (gap) | Always equals confirmed deliveries (no gap) |
| HTTP response on MQ error | `202 Accepted` (fire-and-forget) | `500 Internal Server Error` (explicit error) |

Because `PUT /api/messages` returns `500` when the MQ connection fails, the
`sendOneWithRetry()` loop in the Vue frontend catches it, waits 2 seconds, and retries
the **same message** — just as it does for the Java backend. The retry loop is identical
regardless of which backend is used.

**Run the UI (`ui-app`):**

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
            cno.ClientConn.HeartbeatInterval = int32(cfg.ReconnectTimeout)
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
