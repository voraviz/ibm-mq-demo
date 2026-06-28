# IBM MQ Demo
- Start MQ container
```bash
printf "passw0rd" | podman secret create mqAdminPassword -
printf "passw0rd" | podman secret create mqAppPassword -
podman run --secret mqAdminPassword --secret mqAppPassword \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --env MQ_ENABLE_METRICS=true \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --publish 9157:9157 \
  --detach icr.io/ibm-messaging/mq
```
- Login to console with [https://localhost:9443](https://localhost:9443) with user *admin* and password *passw0rd*


