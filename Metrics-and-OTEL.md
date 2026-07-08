# Grafana and Jaeger
- Enable metrics and OTEL in ui-app and api-app
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
- Example of Quarkus Grafana dashboard [etc/14370_rev6.json](etc/14370_rev6.json)

