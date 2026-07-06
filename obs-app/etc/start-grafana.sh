#!/bin/bash
# Start Grafana pre-configured to use Prometheus as its data source.
# Run from the repository root:
#   bash obs-app/etc/start-grafana.sh

set -e

echo "Starting Grafana..."
podman run --name obs-grafana \
  -p 3000:3000 \
  -e TZ=Asia/Bangkok \
  -e GF_SECURITY_ADMIN_USER=admin \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  -e GF_AUTH_ANONYMOUS_ENABLED=true \
  -e GF_AUTH_ANONYMOUS_ORG_ROLE=Viewer \
  -e "GF_DATASOURCES_DEFAULT_TYPE=prometheus" \
  -e "GF_DATASOURCES_DEFAULT_URL=http://host.containers.internal:9090" \
  -d grafana/grafana:latest

echo ""
echo "Grafana started:"
echo "  UI       → http://localhost:3000"
echo "  Login    → admin / admin"
echo ""
echo "To import the Quarkus dashboard:"
echo "  1. Open http://localhost:3000"
echo "  2. Dashboards → Import"
echo "  3. Upload etc/14370_rev6.json"
echo "  4. Set data source to Prometheus → Import"
