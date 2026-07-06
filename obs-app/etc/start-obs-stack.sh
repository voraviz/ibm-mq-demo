#!/bin/bash
# Start the observability stack for obs-app.
# Run this script from the repository root:
#   bash obs-app/etc/start-obs-stack.sh

set -e

echo "Starting Jaeger..."
# Port 4319 exposes Jaeger's internal OTLP gRPC port (4317) on the host
# so the OTEL Collector can forward traces to it via host.containers.internal:4319
podman run --name obs-jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 4317:4317 \
  -e TZ=Asia/Bangkok \
  -d jaegertracing/all-in-one:latest

#echo "Starting OTEL Collector..."
# podman run --name obs-otel-collector \
#   -v "$(pwd)/obs-app/etc/otel-collector-config.yaml:/etc/otelcol/config.yaml:Z" \
#   -p 13133:13133 \
#   -p 4317:4317 \
#   -e TZ=Asia/Bangkok \
#   -d otel/opentelemetry-collector:latest \
#   "--config=/etc/otelcol/config.yaml"

# echo ""
echo "Observability stack started:"
echo "  Jaeger UI      → http://localhost:16686"
# echo "  OTEL Collector → localhost:4317 (OTLP gRPC)"
