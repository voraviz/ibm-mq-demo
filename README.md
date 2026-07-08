# IBM MQ Demo

## Project Structure

```
ibm_mq/
├── all-in-one-app/   # Original monolithic Quarkus + Vue app (reference only)
├── api-app/          # Quarkus REST API microservice (port 8081)
├── ui-app/           # Vue 3 + Vite frontend (port 8080)
├── obs-app/          # Observability variant — Prometheus metrics + OTEL tracing
│   ├── api/          #   Quarkus API (port 8082) with quarkus-opentelemetry added
│   ├── ui/           #   Vue 3 + Vite frontend (port 8083)
│   └── etc/          #   OTEL Collector config + startup script
├── etc/              # Reference infrastructure scripts and dashboards
├── design.md         # IBM Carbon Design System specification
└── README.md
```

### Quick Start

1. Start IBM MQ (see below)
2. Start the API: `cd api-app && mvn quarkus:dev`
3. Start the UI: `cd ui-app && npm install && npm run dev`
4. Open **http://localhost:8080**

See [`api-app/README.md`](api-app/README.md), [`ui-app/README.md`](ui-app/README.md), and [`obs-app/api/README.md`](obs-app/api/README.md) for full details.

---

## IBM MQ Container

- Create MQ container
```bash
printf "passw0rd" | podman secret create mqAdminPassword -
printf "passw0rd" | podman secret create mqAppPassword -
podman volume create qm1data
podman run --secret mqAdminPassword --secret mqAppPassword \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --env MQ_ENABLE_METRICS=true \
  --volume qm1data:/mnt/mqm \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --publish 9157:9157 \
  --name QM1 \
  --detach \
  icr.io/ibm-messaging/mq
```
- Login to console with [https://localhost:9443](https://localhost:9443) with user *admin* and password *passw0rd*


- Create volume
```bash
podman volume create qm1data
```
- Run
```bash
podman run --env LICENSE=accept --env MQ_QMGR_NAME=QM1 --volume qm1data:/mnt/mqm --publish 1414:1414 --publish 9443:9443 --detach --env MQ_APP_USER=app --env MQ_APP_PASSWORD=passw0rd --env MQ_ADMIN_USER=admin --env MQ_ADMIN_PASSWORD=passw0rd --name QM1 icr.io/ibm-messaging/mq:latest
```
- Shell into pod
```bash
podman exec -ti QM1 bash
```
- MQ CLI
```bash
dspmqver
dspmq
```
Reference:
- [IBM MQ Developer Essentials](https://developer.ibm.com/learningpaths/ibm-mq-badge/)


```bash
# Terminal 1 — start the API
cd api-app && mvn quarkus:dev

# Terminal 2 — start the UI
cd ui-app && npm install && npm run dev

# Open http://localhost:8080
```



---

## Observability Stack (obs-app)

`obs-app/` is a fully instrumented variant of the demo with:
- **Prometheus metrics** — JVM, HTTP, and custom `mq.messages.put` counter at `/q/metrics`
- **Distributed tracing** — OTEL auto-instrumentation exports traces to Jaeger via OTEL Collector

### Start

**1. Start IBM MQ with metrics enabled**
```bash
printf "passw0rd" | podman secret create mqAdminPassword - 2>/dev/null || true
printf "passw0rd" | podman secret create mqAppPassword - 2>/dev/null || true
podman run --secret mqAdminPassword --secret mqAppPassword \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --env MQ_ENABLE_METRICS=true \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --publish 9157:9157 \
  --name QM1 --detach \
  icr.io/ibm-messaging/mq
```

**2. Start OTEL Collector + Jaeger** (run from repo root)
```bash
bash obs-app/etc/start-obs-stack.sh
```

**3. Start the API**
```bash
cd obs-app/api && mvn quarkus:dev
```

**4. Start the UI**
```bash
cd obs-app/ui && npm install && npm run dev
```

**5. Open** http://localhost:8083

### Endpoints

| URL | Description |
|---|---|
| http://localhost:8083 | UI |
| http://localhost:8082/q/metrics | Prometheus metrics |
| http://localhost:16686 | Jaeger trace UI |

### Grafana Dashboard

Import `etc/14370_rev6.json` into Grafana and point the Prometheus data source at `http://localhost:8082`.
http://host.containers.internal:9090




 <!-- ```bash
  skopeo list-tags docker://icr.io/ibm-messaging/mq
  ``` -->
<!-- - Native HA Active node will re
```bash
ADMIN_USER=admin
ADMIN_PASS=passw0rd
QM_NAME=QM1
CONSOLE_PORT=9444
curl -sk -u ${ADMIN_USER}:${ADMIN_PASS} \
  https://localhost:$CONSOLE_PORT/ibmmq/rest/v1/admin/qmgr/$QM_NAME | jq

```
Output
```bash
{
  "qmgr": [
    {
      "name": "QM1",
      "state": "running"
    }
  ]
}
```

```bash
curl http://localhost:9157/metrics
```

```bash
for i in mq-node4-data mq-node5-data mq-node6-data
do
 podman volume exists $i
 if [ $? -eq 0 ];
 then
  podman volume rm $i
 fi
 podman volume create $i 
``` -->