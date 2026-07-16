# Observability
## Architecture

```
  ┌────────────────────────┐
  │  Grafana               │
  │  (container) :3000     │
  └────────────────────────┘
               │
               │  (via host.containers.internal — exits mq-ha-net to host)
               ▼
  ┌─────────────────────────────── mq-ha-net (podman network) ──────────────────────────────┐
  │                                                                                         │
  │   ┌─────────────────┐   scrape :9157      ┌──────────────────────────────────────────┐  │
  │   │                 │ ──────────────────► │  IBM MQ node-1  :9157 /metrics           │  │
  │   │   Prometheus    │ ──────────────────► │  IBM MQ node-2  :9157 /metrics           │  │
  │   │   :9090         │ ──────────────────► │  IBM MQ node-3  :9157 /metrics           │  │
  │   │                 │                     └──────────────────────────────────────────┘  │
  │   └────────┬────────┘                                                                   │
  │            │ scrape host.containers.internal:8080 /q/metrics                            │
  └────────────┼────────────────────────────────────────────────────────────────────────────┘
               │
               │  (via host.containers.internal — exits mq-ha-net to host)
               ▼
  ┌────────────────────────┐    OTLP gRPC       ┌────────────────────────┐
  │  Quarkus all-in-one    │ ────:4317────────► │  Jaeger                │
  │  (host) :8080          │                    │  (container) :16686 UI │
  └────────────────────────┘                    └────────────────────────┘

 
```

> **Note:** Prometheus joins `mq-ha-net` to reach MQ containers by their internal addresses.
> Grafana and Jaeger run without a named network and communicate via exposed host ports.

## IBM MQ Metrics

Metrics can be enabled by environment variable *MQ_ENABLE_METRICS=true* for container or use following configuration in MQSC cofig

```bash
ALTER QMGR STATMQI(ON) STATQ(ON) STATCHL(LOW) ACCTMQI(ON) ACCTQ(ON)
```

Sample metrics

|                     Prometheus Metric                     |                Maps to MQ metric               |
|:---------------------------------------------------------:|:----------------------------------------------:|
| ibmmq_qmgr_cpu_load_fifteen_minute_average_percentage     | CPU 15-min avg %                               |
| ibmmq_qmgr_ram_total_estimate_for_queue_manager_megabytes | RAM used by QM                                 |
| ibmmq_qmgr_log_write_latency_microseconds                 | Log write latency                              |
| ibmmq_qmgr_log_occupancy_percentage                       | Log disk % used                                |
| ibmmq_queue_depth                                         | Current queue depth                            |
| ibmmq_queue_time_longest_unit_of_work_microseconds        | Oldest msg age                                 |
| ibmmq_queue_mqput_mqput1_count                            | PUT rate                                       |
| ibmmq_queue_destructive_mqget_count                       | GET rate                                       |
| ibmmq_channel_status                                      | Channel state (MQCHS_* values)                 |
| ibmmq_channel_status_squash                               | Simplified: 0=stopped, 1=transition, 2=running |
| ibmmq_channel_bytes_sent                                  | Channel throughput                             |

<!-- Native HA specific metrics

|               Prometheus Metric               |             Maps to MQ metric             |
|:---------------------------------------------:|:-----------------------------------------:|
| ibmmq_nha_role                                | NativeHA active/replica status            |
| ibmmq_nha_quorum                              | NativeHA quorum count                     |
| ibmmq_nha_backlog_bytes                       | How far behind a replica is               |
| ibmmq_nha_synchronous_log_sent_bytes_in_doubt | Log bytes not yet acknowledged by replica |
| ibmmq_nha_synchronous_log_sent_bytes          | Log bytes replicated synchronously        | -->

Remark: For Native HA, metrics will available on active node only



## Setup

### Prometheus
- Start Prometheus container with [start-prometheus.sh](obs-app/etc/start-prometheus.sh)
```bash
cd obs-app/etc
./start-prometheus.sh
```
Output
```bash
Prometheus started:
  UI     → http://localhost:9090
  Targets → http://localhost:9090/targets
```
- Prometheus is already configured with [scraping metrics](obs-app/etc/prometheus.yaml) from MQ and Java Apps

![](images/prometheus-target.png)

- Test PromQL 
  - PUT tps in last 5 minutes by Queue Manager
  ```
  sum(rate(ibmmq_qmgr_mqput_mqput1_total[5m])) by (qmgr)
  ```

  ![](images/promql-put-tps.png)
  - GET tps in last 5 minutes by Queue Manager  
  ```
  sum(rate(ibmmq_qmgr_destructive_get_total[5m])) by (qmgr)
  ```

  ![](images/promql-get-tps.png)

### Grafana
- Start Prometheus container with [start-prometheus.sh](obs-app/etc/start-grafana.sh)
```bash
./start-grafana.sh
```
Output
```bash
Grafana started:
  UI       → http://localhost:3000
  Login    → admin / admin

To import the Quarkus dashboard:
  1. Open http://localhost:3000
  2. Dashboards → Import
  3. Set data source to Prometheus → Import
```
- Add Prometheus datasource
  - Open [Grafana Dashboard](http://localhost:3000/)
  - Grafana already configured with default data source to http://host.containers.internal:9090
  - Click Sign-In and sign-in with user admin password admin
## OpenTelemetry

### all-in-one-app with OTEL enabled at runtime

#### Maven dependencies

Two extensions are required in [`all-in-one-app/pom.xml`](all-in-one-app/pom.xml):

```xml
<!-- OpenTelemetry tracing (OTLP exporter included) -->
<dependency>
  <groupId>io.quarkus</groupId>
  <artifactId>quarkus-opentelemetry</artifactId>
</dependency>

<!-- Micrometer → OTLP bridge (optional: keeps Prometheus metrics separate) -->
<dependency>
  <groupId>io.quarkus</groupId>
  <artifactId>quarkus-micrometer-opentelemetry</artifactId>
</dependency>
```

#### application.properties

Configuration in [`all-in-one-app/src/main/resources/application.properties`](all-in-one-app/src/main/resources/application.properties):

```properties
# ── OpenTelemetry ────────────────────────────────────────────────────────────
# Enable the OTEL extension
quarkus.otel.enabled=true

# OTEL metrics export disabled — Prometheus/Micrometer is used for metrics instead
quarkus.otel.metrics.enabled=false

# SDK active at runtime (set to true to disable tracing without rebuilding)
quarkus.otel.sdk.disabled=false

# Service name displayed in Jaeger
quarkus.application.name=all-in-one

# OTLP gRPC collector endpoint (Jaeger accepts OTLP directly on :4317)
quarkus.otel.exporter.otlp.endpoint=http://localhost:4317

# Optional: add an auth header if the collector requires it
#quarkus.otel.exporter.otlp.headers=authorization=Bearer my_secret

# Optional: enrich log lines with trace context fields
#quarkus.log.console.format=%d{HH:mm:ss} %-5p traceId=%X{traceId}, parentId=%X{parentId}, spanId=%X{spanId}, sampled=%X{sampled} [%c{2.}] (%t) %s%e%n
```

> **Runtime toggle:** To disable tracing on a running instance without a rebuild, pass
> `-Dquarkus.otel.sdk.disabled=true` on the JVM command line or set the environment variable
> `QUARKUS_OTEL_SDK_DISABLED=true`. Set it back to `false` to re-enable.

### Jaeger

Start Jaeger using [`obs-app/etc/start-jaeger.sh`](obs-app/etc/start-jaeger.sh):

```bash
cd obs-app/etc
./start-jaeger.sh
```

Output:
```
Observability stack started:
  Jaeger UI      → http://localhost:16686
```

Jaeger listens on:

| Port  | Protocol  | Purpose                        |
|-------|-----------|--------------------------------|
| 16686 | HTTP      | Jaeger UI                      |
| 4317  | gRPC      | OTLP trace ingestion           |
| 14268 | HTTP      | Jaeger native span ingestion   |
| 14250 | gRPC      | Jaeger native model ingestion  |

Once the `all-in-one-app` is running and receiving traffic, open the [Jaeger UI](http://localhost:16686), select service **all-in-one**, and click **Find Traces** to view distributed traces.

![](images/jaeger-traces.png)

