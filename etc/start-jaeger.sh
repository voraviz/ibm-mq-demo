#!/bin/bash
# Start the observability stack (Jaeger for OTLP trace ingestion).
# Run this script from the repository root:
#   bash etc/start-jaeger.sh

set -e

echo "Starting Jaeger..."

podman run --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 4317:4317 \
  -e TZ=Asia/Bangkok \
  -d jaegertracing/all-in-one:latest
# echo ""
echo "Observability stack started:"
echo "  Jaeger UI      → http://localhost:16686"
# echo "  OTEL Collector → localhost:4317 (OTLP gRPC)"
