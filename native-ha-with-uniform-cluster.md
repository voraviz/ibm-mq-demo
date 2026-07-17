# IBM MQ Native HA with Uniform Cluster

IBM MQ **Uniform Cluster** extends Native HA by grouping two or more HA pairs into a single logical cluster. Each HA group runs as an independent queue manager, and the cluster layer distributes client connections and workload evenly across all active nodes. A client connecting to any node in the cluster is automatically rebalanced if the cluster becomes uneven.

## Table of Contents

- [IBM MQ Native HA with Uniform Cluster](#ibm-mq-native-ha-with-uniform-cluster)
  - [Table of Contents](#table-of-contents)
  - [Architecture](#architecture)
  - [MQ Configuration](#mq-configuration)
    - [Prerequisites](#prerequisites)
    - [Create Network and Volumes](#create-network-and-volumes)
    - [Create Secrets](#create-secrets)
    - [Node Configuration](#node-configuration)
    - [Start Containers](#start-containers)
    - [Verify the Cluster](#verify-the-cluster)
    - [CCDT (Client Channel Definition Table)](#ccdt-client-channel-definition-table)
  - [Test Applications](#test-applications)
    - [All-in-one Java Application](#all-in-one-java-application)
    - [Golang REST API](#golang-rest-api)
    - [Golang JMS REST API](#golang-jms-rest-api)
  - [Failover Test](#failover-test)
    - [Scenario 1 — Single node failover within HA-GROUP-1](#scenario-1--single-node-failover-within-ha-group-1)
    - [Scenario 2 — Full HA-GROUP-1 loss (all 3 nodes stopped)](#scenario-2--full-ha-group-1-loss-all-3-nodes-stopped)
  - [Application Load Balancing](#application-load-balancing)

---

## Architecture

```
                    +------------------------------------------+
                    |          Client Applications             |
                    |    Connects via CCDT (all 6 nodes)       |
                    |  127.0.0.1(1414–1419), QM: *UNIQA        |
                    +------------------------------------------+
                                        │
          ┌─────────────────────────────┴─────────────────────────────┐
          │  Uniform Cluster UNIQA                                    │
          │  (workload balanced across both HA groups)                │
          │                                                           │
          │  ┌──────────── HA-GROUP-1 (QM1) ─────────────┐            │
          │  │                                           │            │
          │  │  mq-node-1     mq-node-2     mq-node-3    │            │
          │  │  port: 1414    port: 1415    port: 1416   │            │
          │  │  (ACTIVE)      (Replica)     (Replica)    │            │
          │  │                                           │            │
          │  │  Raft replication ──────────────────────► │            │
          │  └───────────────────────────────────────────┘            │
          │                                                           │
          │  ┌──────────── HA-GROUP-2 (QM2) ─────────────┐            │
          │  │                                           │            │
          │  │  mq-node-4     mq-node-5     mq-node-6    │            │
          │  │  port: 1417    port: 1418    port: 1419   │            │
          │  │  (ACTIVE)      (Replica)     (Replica)    │            │
          │  │                                           │            │
          │  │  Raft replication ──────────────────────► │            │
          │  └───────────────────────────────────────────┘            │
          └───────────────────────────────────────────────────────────┘
```

> **Key difference from Native HA:** In a plain Native HA setup the client must reconnect to the *same* queue manager after failover. In a Uniform Cluster the client connects to the cluster name (`*UNIQA`) and the cluster redistributes connections across both HA groups dynamically.

---

## MQ Configuration

### Prerequisites

- [Podman](https://podman.io/) (or Docker) installed.
- On Apple Silicon (M-series), all `podman run` commands include `--platform linux/amd64` because IBM does not publish an arm64 MQ server container image — it is available only for `linux/amd64`, `linux/s390x`, and `linux/ppc64le`.

### Create Network and Volumes

Create a dedicated container network so all six nodes can resolve each other by hostname:

```bash
podman network create mq-ha-net
```

Create one persistent volume per node. The script removes existing volumes before re-creating them so you can re-run it on a clean slate:

```bash
for i in mq-node-1-data mq-node-2-data mq-node-3-data mq-node-4-data mq-node-5-data mq-node-6-data; do
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

### Node Configuration

Each node requires a `native-ha.ini` file. The configuration is identical to a plain Native HA setup with one addition: an `AutoCluster` section that registers the node with the uniform cluster.

**Node 1** — [`mq-native-ha/config/qm-node1-cluster.ini`](mq-native-ha/config/qm-node1-cluster.ini)

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

AutoCluster:
  ClusterName=UNIQA
  Type=Uniform
  Repository1Name=QM1
  Repository1Conname=mq-node-1(1414),mq-node-2(1414),mq-node-3(1414)
  Repository2Name=QM2
  Repository2Conname=mq-node-4(1414),mq-node-5(1414),mq-node-6(1414)

Variables:
  CONNAME=mq-node-1(1414)
```

Nodes 2 and 3 have the same `AutoCluster` block — only `NativeHALocalInstance.Name` and `Variables.CONNAME` differ. Nodes 4–6 belong to `HA-GROUP-2` and use `QM2` as the queue manager name.

| Component        | File                                                                                               |
|------------------|----------------------------------------------------------------------------------------------------|
| MQSC             | [`mq-native-ha/config/config.cluster.mqsc`](mq-native-ha/config/config.cluster.mqsc)              |
| Node 1 (QM1)     | [`mq-native-ha/config/qm-node1-cluster.ini`](mq-native-ha/config/qm-node1-cluster.ini)            |
| Node 2 (QM1)     | [`mq-native-ha/config/qm-node2-cluster.ini`](mq-native-ha/config/qm-node2-cluster.ini)            |
| Node 3 (QM1)     | [`mq-native-ha/config/qm-node3-cluster.ini`](mq-native-ha/config/qm-node3-cluster.ini)            |
| Node 4 (QM2)     | [`mq-native-ha/config/qm-node4-cluster.ini`](mq-native-ha/config/qm-node4-cluster.ini)            |
| Node 5 (QM2)     | [`mq-native-ha/config/qm-node5-cluster.ini`](mq-native-ha/config/qm-node5-cluster.ini)            |
| Node 6 (QM2)     | [`mq-native-ha/config/qm-node6-cluster.ini`](mq-native-ha/config/qm-node6-cluster.ini)            |

### Start Containers

The loop increments ports automatically: nodes 1–3 use `1414–1416` (QM1), nodes 4–6 use `1417–1419` (QM2).

```bash
TAG=10.0.0.0-r1-amd64
MQ_PORT=1414
MQ_PROMETHEUS_PORT=9517
MQ_CONSOLE_PORT=9443
QUEUE_MANAGER=QM1

for i in 1 2 3 4 5 6; do
  if [ $i -gt 3 ]; then
    QUEUE_MANAGER=QM2
  fi
  echo "Creating Node $i ..."
  podman run -d \
    --secret mqAdminPassword --secret mqAppPassword \
    --name mq-node-$i \
    --platform linux/amd64 \
    --network mq-ha-net \
    --hostname mq-node-$i \
    -p $MQ_PORT:1414 \
    -p $MQ_CONSOLE_PORT:9443 \
    -p $MQ_PROMETHEUS_PORT:9517 \
    -v mq-node-$i-data:/var/mqm \
    -v ./mq-native-ha/config/qm-node$i-cluster.ini:/etc/mqm/native-ha.ini:ro \
    -v ./mq-native-ha/config/config.cluster.mqsc:/etc/mqm/config.mqsc:ro \
    -e LICENSE=accept \
    -e MQ_QMGR_NAME=$QUEUE_MANAGER \
    -e MQ_NATIVE_HA=true \
    -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
    -e MQ_ENABLE_METRICS=true \
    icr.io/ibm-messaging/mq:$TAG
  MQ_PORT=$((MQ_PORT + 1))
  MQ_PROMETHEUS_PORT=$((MQ_PROMETHEUS_PORT + 1))
  MQ_CONSOLE_PORT=$((MQ_CONSOLE_PORT + 1))
done
```

Port mapping summary:

| Node | Queue Manager | MQ Port | Web Console | Prometheus |
|------|--------------|---------|-------------|------------|
| mq-node-1 | QM1 | 1414 | 9443 | 9517 |
| mq-node-2 | QM1 | 1415 | 9444 | 9518 |
| mq-node-3 | QM1 | 1416 | 9445 | 9519 |
| mq-node-4 | QM2 | 1417 | 9446 | 9520 |
| mq-node-5 | QM2 | 1418 | 9447 | 9521 |
| mq-node-6 | QM2 | 1419 | 9448 | 9522 |

### Verify the Cluster

**Check all containers are running:**

```bash
podman ps --format "table {{.Names}}\t{{.Ports}}\t{{.Status}}"
```

Expected output:

```
NAMES       PORTS                                                                                          STATUS
mq-node-1   0.0.0.0:1414->1414/tcp, 0.0.0.0:9443->9443/tcp, 0.0.0.0:9517->9517/tcp, 9157/tcp, 9415/tcp  Up 45 seconds
mq-node-2   0.0.0.0:1415->1414/tcp, 0.0.0.0:9444->9443/tcp, 0.0.0.0:9518->9517/tcp, 9157/tcp, 9415/tcp  Up 43 seconds
mq-node-3   0.0.0.0:1416->1414/tcp, 0.0.0.0:9445->9443/tcp, 0.0.0.0:9519->9517/tcp, 9157/tcp, 9415/tcp  Up 40 seconds
mq-node-4   0.0.0.0:1417->1414/tcp, 0.0.0.0:9446->9443/tcp, 0.0.0.0:9520->9517/tcp, 9157/tcp, 9415/tcp  Up 38 seconds
mq-node-5   0.0.0.0:1418->1414/tcp, 0.0.0.0:9447->9443/tcp, 0.0.0.0:9521->9517/tcp, 9157/tcp, 9415/tcp  Up 36 seconds
mq-node-6   0.0.0.0:1419->1414/tcp, 0.0.0.0:9448->9443/tcp, 0.0.0.0:9522->9517/tcp, 9157/tcp, 9415/tcp  Up 33 seconds
```

**Check Native HA status** — wait ~30 seconds for leader elections to complete:

```bash
podman exec mq-node-1 dspmq -o nativeha -x
podman exec mq-node-4 dspmq -o nativeha -x
```

QM1 (HA-GROUP-1):

```
QMNAME(QM1)       ROLE(Active)  INSTANCE(node-1) INSYNC(yes) QUORUM(3/3) GRPLSN(<0:0:17:6826>) GRPNAME(HA-GROUP-1) GRPROLE(Live)
 INSTANCE(node-1) ROLE(Active)  REPLADDR(mq-node-1) CONNACTV(yes) INSYNC(yes) BACKLOG(0) CONNINST(yes) ACKLSN(<0:0:17:6826>) HASTATUS(Normal)
 INSTANCE(node-2) ROLE(Replica) REPLADDR(mq-node-2) CONNACTV(yes) INSYNC(yes) BACKLOG(0) CONNINST(yes) ACKLSN(<0:0:17:6826>) HASTATUS(Normal)
 INSTANCE(node-3) ROLE(Replica) REPLADDR(mq-node-3) CONNACTV(yes) INSYNC(yes) BACKLOG(0) CONNINST(yes) ACKLSN(<0:0:17:6826>) HASTATUS(Normal)
```

QM2 (HA-GROUP-2):

```
QMNAME(QM2)       ROLE(Active)  INSTANCE(node-4) INSYNC(yes) QUORUM(3/3) GRPLSN(<0:0:12:52222>) GRPNAME(HA-GROUP-2) GRPROLE(Live)
 INSTANCE(node-4) ROLE(Active)  REPLADDR(mq-node-4) CONNACTV(yes) INSYNC(yes) BACKLOG(0) CONNINST(yes) ACKLSN(<0:0:12:52222>) HASTATUS(Normal)
 INSTANCE(node-5) ROLE(Replica) REPLADDR(mq-node-5) CONNACTV(yes) INSYNC(yes) BACKLOG(5) CONNINST(yes) ACKLSN(<0:0:12:52222>) HASTATUS(Normal)
 INSTANCE(node-6) ROLE(Replica) REPLADDR(mq-node-6) CONNACTV(yes) INSYNC(yes) BACKLOG(5) CONNINST(yes) ACKLSN(<0:0:12:52222>) HASTATUS(Normal)
```

**Check cluster membership** — both queue managers should see each other:

```bash
podman exec mq-node-1 bash -c "echo 'DISPLAY CLUSQMGR(*)' | runmqsc QM1"
podman exec mq-node-4 bash -c "echo 'DISPLAY CLUSQMGR(*)' | runmqsc QM2"
```

Expected output (same from both):

```
AMQ8441I: Display Cluster Queue Manager details.
   CLUSQMGR(QM1)   CHANNEL(UNIQA_QM1)   CLUSTER(UNIQA)
AMQ8441I: Display Cluster Queue Manager details.
   CLUSQMGR(QM2)   CHANNEL(UNIQA_QM2)   CLUSTER(UNIQA)
```

### CCDT (Client Channel Definition Table)

For a Uniform Cluster, **CCDT is the recommended connection method**. It lists all six node endpoints in a single file so the client can reach any node in the cluster regardless of which HA group is active.

**[`ccdt/ccdt.cluster.json`](ccdt/ccdt.cluster.json)** — covers all 6 nodes across both HA groups, queue manager `UNIQA`:

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
        "queueManager": "UNIQA"
      },
      "type": "clientConnection"
    },
    {
      "name": "DEV.APP.SVRCONN",
      "clientConnection": {
        "connection": [
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
```

> **`*UNIQA` vs `UNIQA`:**
> - **Java** (`all-in-one-app`): set `ibm.mq.queue-manager=*UNIQA`. The asterisk prefix is handled natively by the IBM MQ JMS client.
> - **Go** (`api-app-go`, `api-app-go-jms20`): set `IBM_MQ_QUEUE_MANAGER=*UNIQA`. The Go MQI C client also accepts the `*` prefix when a CCDT is used.
> - The CCDT file itself uses `queueManager: "UNIQA"` (no asterisk) as the cluster name for channel resolution.
>
> **Go CCDT path:** The Go MQI C library requires an **absolute** `file:///path` URL. A relative `file:../ccdt/...` silently fails and produces `MQRC_Q_MGR_NAME_ERROR (2058)`. Both Go apps call `resolveCcdtUrl()` which automatically converts any relative path to absolute.

---

## Test Applications

All applications connect to the Uniform Cluster using the **CCDT** file ([`ccdt/ccdt.cluster.json`](ccdt/ccdt.cluster.json)) and the queue manager name `*UNIQA` so the client accepts connections from either `QM1` or `QM2`.

<!-- | Application | Language | `BALANCED` | Notes |
|---|---|---|---|
| `all-in-one-app` | Java JMS | `YES` / `NO` | Full cluster rebalancing supported |
| `api-app-go` | Go MQI C | `NOTAPPLIC` | HA failover works; rebalancing not supported by MQI C |
| `api-app-go-jms20` | Go MQI C | `NOTAPPLIC` | Same as `api-app-go` — JMS20 wraps MQI C, not Java JMS | -->

### All-in-one Java Application

[`all-in-one-app`](all-in-one-app) is a Quarkus/JMS backend with a bundled Vue.js frontend.

**Configure CCDT** — [`application.properties`](all-in-one-app/src/main/resources/application.properties):

```properties
ibm.mq.ccdt-url=file:../ccdt/ccdt.cluster.json
#ibm.mq.connection-list=localhost(1414),localhost(1415),localhost(1416)
#ibm.mq.channel=DEV.APP.SVRCONN
ibm.mq.queue-manager=*UNIQA
```

> The path prefix `file:../ccdt/` is relative to the working directory when running from the project root via `mvn quarkus:dev`. Adjust the path when running from a different location or as a container.

**Build and run:**

```bash
cd all-in-one-app

# Run in dev mode (from all-in-one-app directory)
mvn quarkus:dev

# Or build and run from JAR (ccdt path relative to working directory)
mvn clean package
java -jar target/quarkus-app/quarkus-run.jar
```

Open [http://localhost:8080](http://localhost:8080) in a browser.

**Application log:**

- Use CCDT:
```log
2026-07-12 15:13:26,114 INFO  [com.example.config.MQConnectionFactoryProducer] (Quarkus Main Thread) Use CCDT: file:../ccdt/ccdt.cluster.json
```

- Connect to cluster (resolves to one of the active nodes):
```log
2026-07-12 15:13:55,047 DEBUG [com.example.resource.InfoResource] (executor-thread-1) Requesting MQ info — configured queueManager: *UNIQA
2026-07-12 15:13:55,093 DEBUG [com.example.resource.InfoResource] (executor-thread-1) MQ connection established successfully
2026-07-12 15:13:55,094 DEBUG [com.example.resource.InfoResource] (executor-thread-1) MQ Server Host: 127.0.0.1 (1414), resolvedQueueManager: QM1
2026-07-12 15:13:55,112 DEBUG [com.example.resource.InfoResource] (executor-thread-1) InfoResponse: connected=true, queueManager=QM1, host=127.0.0.1, port=1414
```

**Check application status on the cluster:**

Run on the active node of each HA group to see which group the client is connected to:

```bash
podman exec mq-node-1 bash -c "echo 'DISPLAY APSTATUS(*)' | runmqsc QM1"
podman exec mq-node-4 bash -c "echo 'DISPLAY APSTATUS(*)' | runmqsc QM2"
```

When the application is connected to `QM1`:

```
AMQ8932I: Display application status details.
   APPLNAME(all-in-one)                    CLUSTER(UNIQA)
   COUNT(1)                                MOVCOUNT(1)
   BALANCED(NO)                            TYPE(APPL)
```

> - `CLUSTER(UNIQA)` — confirms the connection is cluster-aware (compare to `CLUSTER( )` in a plain Native HA setup).
> - `MOVCOUNT(1)` — the cluster has moved this application connection once to balance load.
> - `BALANCED(NO)` — the cluster is still rebalancing; it becomes `YES` once load is distributed evenly.

---

### Golang REST API

[`api-app-go`](api-app-go) is a standalone Go REST API backend. Pair it with the [`ui-app`](ui-app) Vue.js frontend.

**Configure and run with CCDT:**

```bash
cd api-app-go
export IBM_MQ_CCDT_URL=file:../ccdt/ccdt.cluster.json
export IBM_MQ_QUEUE_MANAGER='*UNIQA'
./api-app-go
```

> `IBM_MQ_CCDT_URL` must be exported before running — it is not read from a config file. The Go app automatically resolves the relative `file:` path to an absolute `file:///...` form.

**Application log:**

- Use CCDT:
```log
2026/07/13 13:14:03 MQ connect: qmgr=*UNIQA ccdt=file:///Users/.../ccdt.cluster.json user=app
```

**Check application status:**

```bash
podman exec mq-node-1 bash -c "echo 'DISPLAY APSTATUS(*)' | runmqsc QM1"
podman exec mq-node-4 bash -c "echo 'DISPLAY APSTATUS(*)' | runmqsc QM2"
```

```
AMQ8932I: Display application status details.
   APPLNAME(API-APP-GO)                    CLUSTER(UNIQA)
   COUNT(1)                                MOVCOUNT(0)
   BALANCED(NO)                     TYPE(APPL)
```

<!-- > `BALANCED(NOTAPPLIC)` is expected for Go apps — the MQI C client does not participate in IBM MQ's application rebalancing protocol. `CLUSTER(UNIQA)` confirms the connection is cluster-aware. HA failover via `MQCNO_RECONNECT` still works across both groups. -->

---

### Golang JMS REST API

[`api-app-go-jms20`](api-app-go-jms20) is a Go REST API using the JMS20-style `mq-golang-jms20` library. Functionally identical to `api-app-go` at the MQ level — both use MQI C bindings underneath.

**Configure and run with CCDT:**

```bash
cd api-app-go-jms20
export IBM_MQ_CCDT_URL=file:../ccdt/ccdt.cluster.json
export IBM_MQ_QUEUE_MANAGER='*UNIQA'
./api-app-go-jms20
```

**Application log:**

- Use CCDT:
```log
2026/07/13 13:14:25 MQ connect: qmgr=*UNIQA ccdt=file:///Users/.../ccdt.cluster.json user=app
```

**Check application status:**

```bash
podman exec mq-node-1 bash -c "echo 'DISPLAY APSTATUS(*)' | runmqsc QM1"
podman exec mq-node-4 bash -c "echo 'DISPLAY APSTATUS(*)' | runmqsc QM2"
```

```
AMQ8932I: Display application status details.
   APPLNAME(API-APP-GO-JMS20)              CLUSTER(UNIQA)
   COUNT(1)                                MOVCOUNT(0)
   BALANCED(NO)                     TYPE(APPL)
```

<!-- > Same `BALANCED(NOTAPPLIC)` behaviour as `api-app-go` — `mq-golang-jms20` wraps MQI C, not the Java JMS provider, so application rebalancing is not available. -->

---

## Failover Test

This walkthrough demonstrates two failure scenarios unique to a Uniform Cluster:

1. **HA group failover** — an active node within a group fails; one replica is promoted.
2. **Full HA group loss** — an entire HA group is stopped; clients rebalance to the surviving group.

### Scenario 1 — Single node failover within HA-GROUP-1

**1. Identify the active node in HA-GROUP-1:**

```bash
podman exec mq-node-1 dspmq -o nativeha -x
```

**2. Stop the active node** (replace `mq-node-1` with whichever node shows `ROLE(Active)`):

```bash
podman stop mq-node-1
```

**3. Observe automatic reconnection** — the MQ client reconnects to the new active node in HA-GROUP-1 within seconds. The application stays connected to QM1.

**4. Bring the stopped node back online:**

```bash
podman start mq-node-1
```

The recovered node rejoins HA-GROUP-1 as a replica and begins log replication catch-up.

### Scenario 2 — Full HA-GROUP-1 loss (all 3 nodes stopped)

**1. Stop all nodes in HA-GROUP-1:**

```bash
podman stop mq-node-1 mq-node-2 mq-node-3
```

**2. Observe cluster rebalancing** — the uniform cluster detects that `QM1` is unavailable and moves all client connections to `QM2`. The `APSTATUS` output on `QM2` will now show the `all-in-one` application:

```bash
podman exec mq-node-4 bash -c "echo 'DISPLAY APSTATUS(*)' | runmqsc QM2"
```

```
AMQ8932I: Display application status details.
   APPLNAME(all-in-one)                    CLUSTER(UNIQA)
   COUNT(1)                                MOVCOUNT(1)
   BALANCED(YES)
```

**3. Restore HA-GROUP-1:**

```bash
podman start mq-node-1 mq-node-2 mq-node-3
```

Once `QM1` is back, the cluster rebalances connections across both groups again. `BALANCED` returns to `NO` briefly during rebalancing, then settles to `YES`.

## Application Load Balancing

This walkthrough demonstrates how a Uniform Cluster distributes application workload across queue managers.

**1. Start Script for check balancing status**

Open a terminal and run from the repository root:

```bash
./check-app-balancing-status.sh <appltag>
```

> The `<appltag>` value depends on which test app you use:
> - Java all-in-one-app → `jack`
> - Golang app → `keshi`

**2. Start Test Apps**

Open another terminal and run from the repository root.

*Golang app* — starts 10 `api-app-go` instances on ports 8100–8109:
```bash
./start-go-lang-apps.sh
```

*Java all-in-one-app* — starts 10 Quarkus instances on ports 9090–9099:
```bash
./start-all-in-one-apps.sh
```

Both scripts:
- Start 10 instances in background mode
- Send 10 messages to each instance
- Start the message listener on each instance
- Write logs to the `logs/` directory

**3. Check App Status**

`BALANCED(NO)` is expected immediately after startup while the cluster is still distributing connections.

![](images/apps-still-not-balanced.png)

It transitions to `YES` once all 10 application instances have settled evenly across both queue managers.

![](images/apps-already-balanced.png)


**4. Cleanup**

*Golang app* — run [clear-go-lang-apps.sh](clear-go-lang-apps.sh) to stop all `api-app-go` processes and remove log files:

```bash
./clear-go-lang-apps.sh
```

*Java all-in-one-app* — run [clear-all-in-one-apps.sh](clear-all-in-one-apps.sh) to stop all Quarkus processes and remove log files:

```bash
./clear-all-in-one-apps.sh
```

**5. [Optional] Bash Scripts**
- MQSC command check Application Status
  
  ```bash
  DISPLAY APSTATUS('<appltag>') TYPE(APPL)
  ```

  Run with podman exec (replace `jack` / `keshi` with your application tag)

  ```bash
  APPLTAG=jack
  podman exec "$QM1_ACTIVE_NODE" \
      bash -c "echo \"DISPLAY APSTATUS('$APPLTAG') TYPE(APPL)\" | runmqsc QM1"
  ```

- MQSC command to check connections

  ```bash
  DISPLAY conn(*) where(appltag eq '<appltag>') conntag
  ```

  Run with bash shell to count number of connections

  ```bash
  APPLTAG=jack   # or keshi for the Golang app
  QM1_ACTIVE_NODE=$(podman exec mq-node-1 dspmq -o nativeha -x \
      | grep "ROLE(Active)" | grep -v QMNAME \
      | awk '{print $3}' \
      | sed -r 's/^[^(]*\(([^)]+)\).*/\1/')
  QM1_CONN=$(podman exec "$QM1_ACTIVE_NODE" \
      bash -c "echo \"dis conn(*) where(appltag eq '$APPLTAG') conntag\" | runmqsc QM1" \
      | grep -c "APPLTAG")
  ```