# obs-app — API

Quarkus REST API with full observability: Prometheus metrics + OpenTelemetry distributed tracing.

This is based on `api-app/` with one additional extension: `quarkus-opentelemetry`.

## Ports

| Service | Port |
|---|---|
| API (HTTP) | **8082** |
| Prometheus metrics | **8082** (`/q/metrics`) |

## Start

```bash
cd obs-app/api
mvn quarkus:dev
```

## Observability

### Prometheus metrics
Exposed at `http://localhost:8082/q/metrics`. Includes:
- JVM metrics (memory, GC, threads)
- HTTP request metrics
- Custom `mq.messages.put` counter

### Distributed tracing (OTEL → Jaeger)
Traces are exported via OTLP gRPC to `localhost:4317` (OTEL Collector).
Start the full stack first:

```bash
cd obs-app/etc
./start-obs-stack.sh
```

View traces at **http://localhost:16686** (Jaeger UI).

## Prerequisites

- IBM MQ running on `localhost:1414` (start with `MQ_ENABLE_METRICS=true`)
- OTEL Collector + Jaeger running (see `obs-app/etc/start-obs-stack.sh`)
