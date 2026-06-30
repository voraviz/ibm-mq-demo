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
  --name QM1 \
  --detach \
  icr.io/ibm-messaging/mq
```
- Login to console with [https://localhost:9443](https://localhost:9443) with user *admin* and password *passw0rd*


- Create volume
```bash
podman volume create qm1data
```
- Run
```bash
podman run --env LICENSE=accept --env MQ_QMGR_NAME=QM1 --volume qm1data:/mnt/mqm --publish 1414:1414 --publish 9443:9443 --detach --env MQ_APP_USER=app --env MQ_APP_PASSWORD=passw0rd --env MQ_ADMIN_USER=admin --env MQ_ADMIN_PASSWORD=passw0rd --name QM1 icr.io/ibm-messaging/mq:latest
```
- Shell into pod
```bash
podman exec -ti QM1 bash
```
- MQ CLI
```bash
dspmqver
dspmq
```
Reference:
- [IBM MQ Developer Essentials](https://developer.ibm.com/learningpaths/ibm-mq-badge/)
