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
  │  Quarkus obs-app       │ ────:4317────────► │  Jaeger                │
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
- Start Grafana container with [start-grafana.sh](obs-app/etc/start-grafana.sh)
```bash
cd obs-app/etc
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
  3. Upload obs-app/etc/14370_rev6.json
  4. Set data source to Prometheus → Import
```
- Add Prometheus datasource
  - Open [Grafana Dashboard](http://localhost:3000/)
  - Grafana already configured with default data source to http://host.containers.internal:9090
  - Click Sign-In and sign-in with user admin password admin
### Jaeger



<!-- - Enable metrics and OTEL in ui-app and api-app
- Keep original ui-app and api-app. Do not modify both apps
- Use ui-app and api-app as reference
- api-app already has quarkus-micrometer-registry-prometheus
- Local MQ container start with metrics enabled 
```bash
podman run --secret mqAdminPassword --secret mqAppPassword \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --env MQ_ENABLE_METRICS=true \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --publish 9157:9157 \
  --name QM1 \
  --detach \
  icr.io/ibm-messaging/mq
```
- Script to start jaeger and otel collector containers is [etc/start-jaeger-otel.sh](etc/start-jaeger-otel.sh)
- Example of Quarkus Grafana dashboard [etc/14370_rev6.json](etc/14370_rev6.json) -->

