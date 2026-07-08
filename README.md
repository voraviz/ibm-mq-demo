# IBM MQ Demo

This repository demonstrates IBM MQ capabilities across three areas:

| Capability | Guide | Status |
|---|---|---|
| **Native HA** | [native-ha.md](native-ha.md) | ✅ Available |
| **Uniform Cluster** | uniform-cluster.md | 🚧 Work in progress |
| **Observability** | metrics-and-otel.md | 🚧 Work in progress |

---

## Repository Structure

```
ibm-mq-demo/
├── mq-native-ha/         # MQ Native HA container configs (ini + MQSC)
├── all-in-one-app/       # Quarkus/JMS + Vue.js all-in-one app (Java)
├── api-app-go/           # Standalone Go REST API backend
├── ui-app/               # Vue 3 + Vite frontend (pairs with api-app-go)
├── api-app/              # Quarkus REST API microservice (Java)
├── obs-app/              # Observability variant — Prometheus + OTEL tracing
│   ├── api/              #   Quarkus API with quarkus-opentelemetry
│   ├── ui/               #   Vue 3 + Vite frontend
│   └── etc/              #   OTEL Collector config + startup script
├── native-ha.md          # Native HA setup and failover walkthrough
├── Metrics-and-OTEL.md   # Observability guide (WIP)
└── README.md
```

---

## Native HA

Three-node IBM MQ cluster with automatic failover using the Raft consensus protocol. Includes setup steps, application reconnect configuration, and a failover walkthrough.

→ See **[native-ha.md](native-ha.md)**

---

## Uniform Cluster

> 🚧 **Work in progress**

---

## Observability

> 🚧 **Work in progress**

Planned coverage: Prometheus metrics, OpenTelemetry distributed tracing, Grafana dashboards, and Jaeger trace UI.

→ See **[Metrics-and-OTEL.md](metrics-and-otel.md)** for current notes.
