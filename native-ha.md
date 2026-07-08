# IBM MQ Native HA
- [IBM MQ Native HA](#ibm-mq-native-ha)
  - [MQ Configuration](#mq-configuration)
  - [All-in-one Java Application](#all-in-one-java-application)
  - [Test Native HA Cluster](#test-native-ha-cluster)

## MQ Configuration 
Create MQ Native HA with 3 containers by podman ( or docker)

```bash

               +----------------------------------------+
               |           Client Applications          |
               |    onnects via Connection Name List    |     
               |                                        |
               +----------------------------------------+
                                   │
       ┌───────────────────────────┼───────────────────────────┐
       │ (Active Traffic)          │                           │ 
       ▼                           ▼                           ▼
+────────────────────────+  +────────────────────────+  +────────────────────────+
│ Container: mq-node-1   │  │ Container: mq-node-2   │  │ Container: mq-node-3   │
│ Port:      1414        │  │ Port:      1415        │  │ Port:      1416        │
│                        │  │                        │  │                        │
│  ┌──────────────────┐  │  │  ┌──────────────────┐  │  │  ┌──────────────────┐  │
│  │   Queue Manager  │  │  │  │   Queue Manager  │  │  │  │   Queue Manager  │  │
│  │     (ACTIVE)     │  │  │  │    (REPLICA)     │  │  │  │    (REPLICA)     │  │
│  └─────────┬────────┘  │  │  └─────────┬────────┘  │  │  └─────────┬────────┘  │
+────────────┼───────────+  +────────────┼───────────+  +────────────┼───────────+
             │                           ▲                           ▲
             │      Log Replication      │                           │
             └───────────────────────────┴───────────────────────────┘
                           (Raft Consensus Protocol)                
```                                                             
  

- Create podman network, Dedicated network so containers resolve each other by name
```bash
podman network create mq-ha-net
```

- Create podman volume. Alternatively for docker, you can mapped local file system to /var/mqm
```bash
# Persistent volumes — one per node
for i in mq-node1-data mq-node2-data mq-node3-data
do
 podman volume exists $i
 if [ $? -eq 0 ];
 then
  podman volume rm $i
 fi
 podman volume create $i 
done
```
- Create secret for store admin and app password
- MQ Configuration
  
  | Component        | Configuration file                                                 |
  |------------------|--------------------------------------------------------------------|
  | MQSC             | [/etc/mqm/config.mqsc](mq-native-ha/config/config.auth.mqsc)       |
  | Native HA Node 1 | [/etc/mqm/native-ha.ini](mq-native-ha/config/qm-node1.ini) |
  | Native HA Node 2 | [/etc/mqm/native-ha.ini](mq-native-ha/config/qm-node2.ini) |
  | Native HA Node 3 | [/etc/mqm/native-ha.ini](mq-native-ha/config/qm-node3.ini) |

  Native HA configuration for node1

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

- Create secret for  administrator and application user

```bash
printf "passw0rd" | podman secret create mqAdminPassword -
printf "passw0rd" | podman secret create mqAppPassword -
```

- Create MQ containers
```bash
# On Apple Silicon need explicitly specified amd64 platform: --platform linux/amd64
# NODE 1
TAG=10.0.0.0-r1-amd64 
CONFIG=config.auth.mqsc
podman run -d --secret mqAdminPassword --secret mqAppPassword \
  --name mq-node-1 \
  --platform linux/amd64 \
  --network mq-ha-net \
  --hostname mq-node-1 \
  -p 1414:1414 \
  -p 9443:9443 \
  -p 9157:9157 \
  -v mq-node1-data:/var/mqm \
  -v ./mq-native-ha/config/qm-node1.ini:/etc/mqm/native-ha.ini:ro \
  -v ./mq-native-ha/config/$CONFIG:/etc/mqm/config.mqsc:ro \
  -e LICENSE=accept \
  -e MQ_QMGR_NAME=QM1 \
  -e MQ_NATIVE_HA=true \
  -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
  -e MQ_ENABLE_METRICS=true \
  icr.io/ibm-messaging/mq:$TAG

# NODE 2
podman run -d --secret mqAdminPassword --secret mqAppPassword  \
  --name mq-node-2 \
  --platform linux/amd64 \
  --network mq-ha-net \
  --hostname mq-node-2 \
  -p 1415:1414 \
  -p 9444:9443 \
  -p 9158:9157 \
  -v mq-node2-data:/var/mqm \
  -v ./mq-native-ha/config/qm-node2.ini:/etc/mqm/native-ha.ini:ro \
  -v ./mq-native-ha/config/$CONFIG:/etc/mqm/config.mqsc:ro \
  -e LICENSE=accept \
  -e MQ_QMGR_NAME=QM1 \
  -e MQ_NATIVE_HA=true \
  -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
  -e MQ_ENABLE_METRICS=true \
  icr.io/ibm-messaging/mq:$TAG

# NODE 3
podman run -d --secret mqAdminPassword --secret mqAppPassword \
  --name mq-node-3 \
  --platform linux/amd64 \
  --network mq-ha-net \
  --hostname mq-node-3 \
  -p 1416:1414 \
  -p 9445:9443 \
  -p 9159:9157 \
  -v mq-node3-data:/var/mqm \
  -v ./mq-native-ha/config/qm-node3.ini:/etc/mqm/native-ha.ini:ro \
  -v ./mq-native-ha/config/$CONFIG:/etc/mqm/config.mqsc:ro \
  -e LICENSE=accept \
  -e MQ_QMGR_NAME=QM1 \
  -e MQ_NATIVE_HA=true \
  -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
  -e MQ_ENABLE_METRICS=true \
  icr.io/ibm-messaging/mq:$TAG 
  ```

- verify containers

```bash
podman ps --format "table {{.Names}}\t{{.Ports}}\t{{.Status}}"
```

Expected Output

```bash
NAMES       PORTS                                                                             STATUS
mq-node-1   0.0.0.0:1414->1414/tcp, 0.0.0.0:9157->9157/tcp, 0.0.0.0:9443->9443/tcp, 9415/tcp  Up 14 seconds
mq-node-2   0.0.0.0:1415->1414/tcp, 0.0.0.0:9158->9157/tcp, 0.0.0.0:9444->9443/tcp, 9415/tcp  Up 12 seconds
mq-node-3   0.0.0.0:1416->1414/tcp, 0.0.0.0:9159->9157/tcp, 0.0.0.0:9445->9443/tcp, 9415/tcp  Up 10 seconds
```

- verify HA status. Wait ~30 seconds for election to complete
  Reference: [IBM MQ 9.4.x dspmq](https://www.ibm.com/docs/en/ibm-mq/9.4.x?topic=reference-dspmq-display-queue-managers)

```bash
podman exec mq-node-1 dspmq -o nativeha -x
```
Output Example. node-2 is Active node

```bash
QMNAME(QM1)       ROLE(Replica) INSTANCE(node-1) INSYNC(yes) QUORUM(3/3) GRPLSN(<0:0:36:38367>) GRPNAME(HA-GROUP-1) GRPROLE(Live)
 INSTANCE(node-1) ROLE(Replica) REPLADDR(mq-node-1) CONNACTV(yes) INSYNC(yes) BACKLOG(0) CONNINST(yes) ACKLSN(<0:0:36:38367>) HASTATUS(Normal) SYNCTIME(2026-07-07T10:37:36.077777Z) ALTDATE(2026-07-07) ALTTIME(10.37.37)
 INSTANCE(node-2) ROLE(Active) REPLADDR(mq-node-2) CONNACTV(yes) INSYNC(yes) BACKLOG(0) CONNINST(yes) ACKLSN(<0:0:36:38367>) HASTATUS(Normal) SYNCTIME(2026-07-07T10:37:37.430102Z) ALTDATE(2026-07-07) ALTTIME(10.37.37)
 INSTANCE(node-3) ROLE(Replica) REPLADDR(mq-node-3) CONNACTV(yes) INSYNC(yes) BACKLOG(0) CONNINST(yes) ACKLSN(<0:0:36:38367>) HASTATUS(Normal) SYNCTIME(2026-07-07T10:37:36.077777Z) ALTDATE(2026-07-07) ALTTIME(10.37.37)
```

- MQ ports of all nodes are up
    
```bash
nc -vz 127.0.0.1 1414;nc -vz 127.0.0.1 1415;nc -vz 127.0.0.1 1416
```
    
Output
    
```bash
Connection to 127.0.0.1 port 1414 [tcp/ibm-mqseries] succeeded!
Connection to 127.0.0.1 port 1415 [tcp/dbstar] succeeded!
Connection to 127.0.0.1 port 1416 [tcp/novell-lu6.2] succeeded!
```

## All-in-one Java Application

Java application as backend to connect to MQ wih bundled Vue.js application as frontend UI
- App configuration with 3 nodes
```properties
# ── IBM MQ connection parameters ────────────────────────────────────────────
ibm.mq.connection-list=localhost(1414),localhost(1415),localhost(1416)
```
- Enable automatic reconnect and config reconnect timeout
```java
@ApplicationScoped
public class MQConnectionFactoryProducer {

    @Inject
    MQConfig config;

    @Produces
    @ApplicationScoped
    public ConnectionFactory connectionFactory() throws JMSException {
        MQConnectionFactory factory = new MQConnectionFactory();
        factory.setConnectionNameList(config.connectionList());
        factory.setChannel(config.channel());
        factory.setQueueManager(config.queueManager());
        factory.setTransportType(WMQConstants.WMQ_CM_CLIENT);
        factory.setStringProperty(WMQConstants.USERID, config.username());
        factory.setStringProperty(WMQConstants.PASSWORD, config.password());
        // Automatically reconnect to the queue manager when it becomes available again.
        // WMQ_CLIENT_RECONNECT retries indefinitely until the QM is reachable.
        // WMQ_CLIENT_RECONNECT_TIMEOUT (seconds) caps how long a single reconnect
        // attempt waits before the client gives up and throws to the application.
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_OPTIONS, WMQConstants.WMQ_CLIENT_RECONNECT);
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_TIMEOUT, 30); // 30 sec
        return factory;
    }
}
```
- Build Java application with maven

```bash
mvn clean package
```

- Run application

```bash
java -jar target/quarkus-app/quarkus-run.jar
```

- You can also run all-in-one-app from container
```bash
podman run -p 8080:8080 \
-e IBM.MQ.CONNECTION-LIST="host.containers.internal(1414),host.containers.internal(1415),host.containers.internal(1416)" \
quay.io/voravitl/simple-mq-app:latest
```
Output

```log
 --/ __ \/ / / / _ | / _ \/ //_/ / / / __/
 -/ /_/ / /_/ / __ |/ , _/ ,< / /_/ /\ \
--\___\_\____/_/ |_/_/|_/_/|_|\____/___/
2026-07-07 18:10:17,975 WARN  [io.smallrye.reactive.messaging.jms] (main) Please add one of the additional mapping modules (-jsonb or -jackson) to be able to (de)serialize JSON messages.
2026-07-07 18:10:18,515 INFO  [io.quarkus] (main) ibm-mq-demo 1.0.0-SNAPSHOT on JVM (powered by Quarkus 3.36.1) started in 0.939s. Listening on: http://0.0.0.0:8080
2026-07-07 18:10:18,516 INFO  [io.quarkus] (main) Profile prod activated.
2026-07-07 18:10:18,516 INFO  [io.quarkus] (main) Installed features: [cdi, messaging, micrometer, rest, rest-jackson, smallrye-context-propagation, vertx, websockets-next]
```

- Use web browser open to http://localhost:8080
  Remark: Top menu bar show that application connect to mq-node-2 that run on port 1415

![](images/mq-app.png)

## Test Native HA Cluster
- Find what node is active with *dspmq* command
    
```bash
podman exec mq-node-1 dspmq -o nativeha -x
```

- Put 500 messages and start Consumer
  
![](images/mq-app-put-500.png)

- Stop active node
  e.g. MQ is active on mq-node-2

```bash
podman stop mq-node-2
```

- Application will automatically connect to new active node

![](images/mq-app-automatic-connect-new-active.png)

- Verify that there is no messages lost
  
![](images/mq-app-message-not-lost.png)

- Bring back the node that you previously stop i.e. mq-node-2

```bash
podman start mq-node-2
```

 



