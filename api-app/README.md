# IBM MQ API — Quarkus REST Microservice

REST API and WebSocket backend for the IBM MQ Demo. Connects to IBM MQ via JMS and exposes HTTP endpoints for sending and consuming messages.

## Prerequisites

- Java 17 or later (Java 25 recommended — matches the Maven compiler target)
- Maven 3.9+
- A running IBM MQ instance (see root [`README.md`](../README.md) for container setup)

## Configuration

Edit [`src/main/resources/application.properties`](src/main/resources/application.properties) to match your IBM MQ connection:

```properties
ibm.mq.host=localhost
ibm.mq.port=1414
ibm.mq.channel=DEV.APP.SVRCONN
ibm.mq.queue-manager=QM1
ibm.mq.username=app
ibm.mq.password=passw0rd
ibm.mq.queue=DEV.QUEUE.1
```

## Running in Development Mode

```bash
cd api-app
mvn quarkus:dev
```

The API will start on **http://localhost:8081**.

## Building a Production JAR

```bash
cd api-app
mvn package
java -jar target/ibm-mq-api-1.0.0-SNAPSHOT-runner.jar
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/messages` | Send a message to IBM MQ. Body: `{"text":"..."}` |
| `POST` | `/api/consumer/start` | Start the JMS consumer |
| `POST` | `/api/consumer/stop` | Stop the JMS consumer |
| `GET`  | `/api/consumer/status` | Get consumer status (`running` or `stopped`) |
| `WS`   | `/ws/messages` | WebSocket — broadcasts messages as they are consumed |
| `GET`  | `/metrics` | Prometheus metrics (Micrometer) |

## CORS

CORS is enabled for all origins (`*`) so the UI on port 8080 can call this API.
To restrict origins, update `quarkus.http.cors.origins` in `application.properties`.
