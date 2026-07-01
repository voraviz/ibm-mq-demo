# IBM MQ Demo Application
- Web App develope using Quarkus framework
- Example of IBM MQ container is 
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
  --detach \
```
- Provide UI to put and get message to/from IBM MQ e.g. left panel put and right panel get
- can start/stop get separately from put 
- enable micrometer metrics
- Contain config to connect to IBM MQ with hostname, port, username, password and queue name
- design UI need to be algin with design.md 
##  UI and API microservices
- Separate UI and API into 2 applications 
- Keep existing Quarkus in all-in-one directory
- API is RESTful using Quarkus. You can still use code in all-in-one-app as reference to remove vue
- Create the same UI app from vue in existing quarkus code in all-in-one directory
- Default port for API is 8081 and UI is 8080 
