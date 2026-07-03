package com.example.resource;

import com.example.config.MQConfig;
import com.ibm.msg.client.jakarta.jms.JmsConnection;
import com.ibm.msg.client.jakarta.wmq.WMQConstants;
import jakarta.inject.Inject;
import jakarta.jms.Connection;
import jakarta.jms.ConnectionFactory;
import jakarta.ws.rs.GET;
import jakarta.ws.rs.Path;
import jakarta.ws.rs.Produces;
import jakarta.ws.rs.core.MediaType;
import jakarta.ws.rs.core.Response;

@Path("/api/info")
@Produces(MediaType.APPLICATION_JSON)
public class InfoResource {

    @Inject
    MQConfig mqConfig;

    @Inject
    ConnectionFactory connectionFactory;

    @GET
    public Response info() {
        boolean connected = false;
        String queueManager = mqConfig.queueManager();
        String host = mqConfig.host();
        int port = mqConfig.port();

        try (Connection connection = connectionFactory.createConnection()) {
            connected = true;
            if (connection instanceof JmsConnection jmsConnection) {
                String resolvedQueueManager = jmsConnection.getStringProperty(WMQConstants.WMQ_RESOLVED_QUEUE_MANAGER);
                String resolvedHost = jmsConnection.getStringProperty(WMQConstants.WMQ_HOST_NAME);
                int resolvedPort = jmsConnection.getIntProperty(WMQConstants.WMQ_PORT);
                if (resolvedQueueManager != null && !resolvedQueueManager.isBlank()) {
                    queueManager = resolvedQueueManager;
                }
                if (resolvedHost != null && !resolvedHost.isBlank()) {
                    host = resolvedHost;
                }
                port = resolvedPort;
            }
        } catch (Exception ignored) {
        }

        return Response.ok(new InfoResponse(queueManager, host, port, connected)).build();
    }

    public record InfoResponse(String queueManager, String host, int port, boolean connected) {}
}
