# IBM MQ Demo

This repository demonstrates IBM MQ capabilities across three areas:

| Capability | Guide | Status |
|---|---|---|
| **Native HA** | [native-ha.md](native-ha.md) | ✅ Available |
| **Native HA + Uniform Cluster** | [native-ha-with-uniform-cluster.md](native-ha-with-uniform-cluster.md) | ✅ Available |
| **Observability** | [metrics-and-otel.md](Metrics-and-OTEL.md) | ✅ Available|
| **Security (SBOM & scanning)** | [SECURITY.md](SECURITY.md) | ✅ Available |
<!-- | **Observability** | [metrics-and-otel.md](Metrics-and-OTEL.md) | 🚧 Work in progress (You can try)| -->

---

## Repository Structure

```
ibm-mq-demo/
├── mq-native-ha/         # MQ Native HA container configs (ini + MQSC)
├── all-in-one-app/       # Quarkus/JMS + Vue.js all-in-one app (Java)
├── api-app-go-jms20/     # Go REST API backend using mq-golang-jms20
├── ui-app/               # Vue 3 + Vite frontend (pairs with api-app-go)
├── config/               # Shared app config (application[.otel].properties)
├── etc/                  # Prometheus/Grafana/Jaeger configs + startup scripts
├── native-ha.md                        # Native HA setup and failover walkthrough
├── native-ha-with-uniform-cluster.md   # Native HA + Uniform Cluster walkthrough
├── ccdt/                 # CCDT JSON files (nativeha + cluster)
├── Metrics-and-OTEL.md   # Observability guide
├── SECURITY.md           # SBOM & vulnerability scanning
├── scan-sbom.sh          # syft SBOM + grype/trivy scan (any app)
├── create-cluster.sh     # create 2 Native HA group with Uniform Cluster 
├── create-nativeha.sh    # create 1 Native HA group with MQ Exporter
└── README.md
```

---

## Native HA

Three-node IBM MQ cluster with automatic failover using the Raft consensus protocol. Includes setup steps, application reconnect configuration, and a failover walkthrough.

→ See **[native-ha.md](native-ha.md)**

---

## Native HA + Uniform Cluster

Two Native HA groups (`QM1` and `QM2`, three nodes each) joined into a single **Uniform Cluster** (`UNIQA`). The cluster distributes client connections evenly across both queue managers and automatically rebalances when a node or an entire HA group fails. Applications connect using a CCDT file covering all six nodes with queue manager set to `*UNIQA`.

→ See **[native-ha-with-uniform-cluster.md](native-ha-with-uniform-cluster.md)**

---

## Observability

<!-- > 🚧 **Work in progress** -->

Planned coverage: Prometheus metrics, OpenTelemetry distributed tracing, Grafana dashboards, and Jaeger trace UI.

→ See **[Metrics-and-OTEL.md](Metrics-and-OTEL.md)** for current notes.

---

## Security

SBOM generation and vulnerability scanning (syft + grype + trivy) via
[`scan-sbom.sh`](scan-sbom.sh), and why both scanners are run.

→ See **[SECURITY.md](SECURITY.md)**
