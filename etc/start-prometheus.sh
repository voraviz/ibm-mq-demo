#!/bin/bash
# Start Prometheus scraping the all-in-one app and IBM MQ.
# Run from the repository root:
#   bash etc/start-prometheus.sh

set -e

echo "Starting Prometheus..."
podman run --name prometheus \
  -v "./prometheus.cluster.yaml:/etc/prometheus/prometheus.yaml:Z" \
  --network mq-ha-net \
  -p 9090:9090 \
  -e TZ=Asia/Bangkok \
  -d prom/prometheus:latest \
  "--config.file=/etc/prometheus/prometheus.yaml"

echo ""
echo "Prometheus started:"
echo "  UI     → http://localhost:9090"
echo "  Targets → http://localhost:9090/targets"
